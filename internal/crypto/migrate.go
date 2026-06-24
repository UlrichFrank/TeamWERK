package crypto

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// PIIReport zählt die in einem encrypt-pii/decrypt-pii-Lauf transformierten Werte.
type PIIReport struct {
	MemberRows int // members-Zeilen mit geänderten iban/account_holder
	ClubRows   int // clubs-Zeilen mit geänderten SEPA-Feldern
	Drafts     int // member_change_drafts (field_name='bankdaten')
	Files      int // SEPA-Mandat-PDFs
}

// EncryptPII verschlüsselt alle Bestandswerte der vier Speicher in-place und ist
// idempotent (bereits verschlüsselte Werte werden übersprungen).
func EncryptPII(db *sql.DB, uploadDir string) (PIIReport, error) {
	return migratePII(db, uploadDir, true)
}

// DecryptPII ist das spiegelbildliche Gegenstück (Rollback/Schlüsselrotation),
// ebenfalls idempotent.
func DecryptPII(db *sql.DB, uploadDir string) (PIIReport, error) {
	return migratePII(db, uploadDir, false)
}

// transformValue ver-/entschlüsselt einen String-Wert idempotent.
// Liefert (neuerWert, changed, error).
func transformValue(v string, encrypt bool) (string, bool, error) {
	if encrypt {
		if IsEncryptedString(v) {
			return v, false, nil
		}
		out, err := Encrypt(v)
		if err != nil {
			return v, false, err
		}
		return out, true, nil
	}
	if !IsEncryptedString(v) {
		return v, false, nil
	}
	out, err := Decrypt(v)
	if err != nil {
		return v, false, err
	}
	return out, true, nil
}

func migratePII(db *sql.DB, uploadDir string, encrypt bool) (PIIReport, error) {
	var rep PIIReport
	if err := migrateMembers(db, encrypt, &rep); err != nil {
		return rep, err
	}
	if err := migrateClubs(db, encrypt, &rep); err != nil {
		return rep, err
	}
	if err := migrateDrafts(db, encrypt, &rep); err != nil {
		return rep, err
	}
	if err := migrateFiles(db, uploadDir, encrypt, &rep); err != nil {
		return rep, err
	}
	return rep, nil
}

func migrateMembers(db *sql.DB, encrypt bool, rep *PIIReport) error {
	rows, err := db.Query(`SELECT id, iban, account_holder FROM members
		WHERE iban IS NOT NULL OR account_holder IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("members select: %w", err)
	}
	type row struct {
		id     int
		iban   sql.NullString
		holder sql.NullString
	}
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.iban, &r.holder); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, r)
	}
	rows.Close()
	for _, r := range batch {
		newIBAN, c1, err := transformNullable(r.iban, encrypt)
		if err != nil {
			return fmt.Errorf("members id=%d iban: %w", r.id, err)
		}
		newHolder, c2, err := transformNullable(r.holder, encrypt)
		if err != nil {
			return fmt.Errorf("members id=%d account_holder: %w", r.id, err)
		}
		if !c1 && !c2 {
			continue
		}
		if _, err := db.Exec(`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
			newIBAN, newHolder, r.id); err != nil {
			return fmt.Errorf("members id=%d update: %w", r.id, err)
		}
		rep.MemberRows++
	}
	return nil
}

func migrateClubs(db *sql.DB, encrypt bool, rep *PIIReport) error {
	rows, err := db.Query(`SELECT id, glaeubiger_id, iban, bic, kontoinhaber FROM clubs`)
	if err != nil {
		return fmt.Errorf("clubs select: %w", err)
	}
	type row struct {
		id                            int
		glaeubiger, iban, bic, holder sql.NullString
	}
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.glaeubiger, &r.iban, &r.bic, &r.holder); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, r)
	}
	rows.Close()
	for _, r := range batch {
		ng, c1, err := transformNullable(r.glaeubiger, encrypt)
		if err != nil {
			return fmt.Errorf("clubs id=%d glaeubiger_id: %w", r.id, err)
		}
		ni, c2, err := transformNullable(r.iban, encrypt)
		if err != nil {
			return fmt.Errorf("clubs id=%d iban: %w", r.id, err)
		}
		nb, c3, err := transformNullable(r.bic, encrypt)
		if err != nil {
			return fmt.Errorf("clubs id=%d bic: %w", r.id, err)
		}
		nh, c4, err := transformNullable(r.holder, encrypt)
		if err != nil {
			return fmt.Errorf("clubs id=%d kontoinhaber: %w", r.id, err)
		}
		if !c1 && !c2 && !c3 && !c4 {
			continue
		}
		if _, err := db.Exec(`UPDATE clubs SET glaeubiger_id=?, iban=?, bic=?, kontoinhaber=? WHERE id=?`,
			ng, ni, nb, nh, r.id); err != nil {
			return fmt.Errorf("clubs id=%d update: %w", r.id, err)
		}
		rep.ClubRows++
	}
	return nil
}

func migrateDrafts(db *sql.DB, encrypt bool, rep *PIIReport) error {
	rows, err := db.Query(`SELECT id, new_value FROM member_change_drafts WHERE field_name='bankdaten'`)
	if err != nil {
		return fmt.Errorf("drafts select: %w", err)
	}
	type row struct {
		id  int
		val sql.NullString
	}
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.val); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, r)
	}
	rows.Close()
	for _, r := range batch {
		nv, changed, err := transformNullable(r.val, encrypt)
		if err != nil {
			return fmt.Errorf("draft id=%d: %w", r.id, err)
		}
		if !changed {
			continue
		}
		if _, err := db.Exec(`UPDATE member_change_drafts SET new_value=? WHERE id=?`, nv, r.id); err != nil {
			return fmt.Errorf("draft id=%d update: %w", r.id, err)
		}
		rep.Drafts++
	}
	return nil
}

func migrateFiles(db *sql.DB, uploadDir string, encrypt bool, rep *PIIReport) error {
	rows, err := db.Query(`SELECT sepa_mandat_path FROM members
		WHERE sepa_mandat_path IS NOT NULL AND sepa_mandat_path <> ''`)
	if err != nil {
		return fmt.Errorf("files select: %w", err)
	}
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		paths = append(paths, p)
	}
	rows.Close()
	for _, rel := range paths {
		full := filepath.Join(uploadDir, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			if os.IsNotExist(err) {
				continue // verwaister Pfad — überspringen
			}
			return fmt.Errorf("read %s: %w", rel, err)
		}
		alreadyEnc := IsEncryptedBytes(data)
		if encrypt && alreadyEnc {
			continue
		}
		if !encrypt && !alreadyEnc {
			continue
		}
		var out []byte
		if encrypt {
			out, err = EncryptBytes(data)
		} else {
			out, err = DecryptBytes(data)
		}
		if err != nil {
			return fmt.Errorf("transform %s: %w", rel, err)
		}
		if err := atomicWrite(full, out); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		rep.Files++
	}
	return nil
}

func transformNullable(ns sql.NullString, encrypt bool) (any, bool, error) {
	if !ns.Valid {
		return nil, false, nil
	}
	out, changed, err := transformValue(ns.String, encrypt)
	if err != nil {
		return nil, false, err
	}
	return out, changed, nil
}

// atomicWrite schreibt in eine Temp-Datei im selben Verzeichnis und ersetzt das
// Ziel per Rename (atomar auf demselben Dateisystem).
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".enc-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
