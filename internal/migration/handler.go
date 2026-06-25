// Package migration stellt die TEMPORÄRE, einmalige Bestandsmigration vom
// serverseitigen `v1:`-At-Rest-Modell (Schlüssel = FIELD_ENCRYPTION_KEY) auf das
// clientseitige Zero-Knowledge-Envelope-Modell (Modell B) bereit.
//
// Ablauf: Der Browser eines Tresor-Inhabers lädt den entschlüsselten Altbestand über die
// noch vorhandene Server-Brücke (`GET …/data`, nur solange FIELD_ENCRYPTION_KEY gesetzt
// ist), re-verschlüsselt ihn clientseitig an den Gruppen-Public-Key und lädt die Envelopes
// hoch (`POST …/upload`). Der Upload schreibt den Envelope UND nullt die Legacy-`v1:`-Spalte
// in einer Transaktion — die Migration ist damit idempotent und der Endpoint deaktiviert sich
// faktisch selbst, sobald kein Altbestand mehr existiert (`GET …/status` → complete).
//
// Dieses Package wird nach Vollmigration in Branch B (`feat/zk-remove-bridge`) wieder entfernt.
package migration

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// Handler bedient die Migrations-Routen. Er braucht direkten DB-Zugriff, das Upload-
// Verzeichnis (für SEPA-Mandat-Blobs) und den Event-Hub für Live-Updates.
type Handler struct {
	db        *sql.DB
	hub       *hub.EventHub
	uploadDir string
}

func NewHandler(db *sql.DB, h *hub.EventHub, uploadDir string) *Handler {
	return &Handler{db: db, hub: h, uploadDir: uploadDir}
}

// ---- GET /api/admin/migrate-legacy/status ----

type statusResp struct {
	BridgeAvailable bool `json:"bridge_available"`
	PendingMembers  int  `json:"pending_members"`
	PendingClub     bool `json:"pending_club"`
	PendingMandates int  `json:"pending_mandates"`
	PendingDrafts   int  `json:"pending_drafts"`
	Complete        bool `json:"complete"`
}

// Status meldet den Migrationsfortschritt. `complete` ist genau dann true, wenn in keinem der
// vier Legacy-Speicher (Member-Bank, Vereins-SEPA, Mandat-PDFs, Bankdaten-Drafts) noch
// Altbestand existiert. `bridge_available` spiegelt, ob der Server überhaupt noch
// entschlüsseln kann (FIELD_ENCRYPTION_KEY gesetzt).
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var resp statusResp
	resp.BridgeAvailable = crypto.HasKey()

	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE iban IS NOT NULL OR account_holder IS NOT NULL`).
		Scan(&resp.PendingMembers)

	var g, i, b, k sql.NullString
	h.db.QueryRowContext(ctx, `SELECT glaeubiger_id, iban, bic, kontoinhaber FROM clubs LIMIT 1`).
		Scan(&g, &i, &b, &k)
	resp.PendingClub = g.String != "" || i.String != "" || b.String != "" || k.String != ""

	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE sepa_mandat_path IS NOT NULL AND sepa_mandat_path <> ''
		 AND (sepa_mandat_dek_enc IS NULL OR sepa_mandat_dek_enc = '')`).
		Scan(&resp.PendingMandates)

	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM member_change_drafts WHERE field_name='bankdaten' AND new_value LIKE 'v1:%'`).
		Scan(&resp.PendingDrafts)

	resp.Complete = resp.PendingMembers == 0 && !resp.PendingClub &&
		resp.PendingMandates == 0 && resp.PendingDrafts == 0

	writeJSON(w, resp)
}

// ---- GET /api/admin/migrate-legacy/data ----

type memberPlain struct {
	MemberID      int    `json:"member_id"`
	IBAN          string `json:"iban"`
	AccountHolder string `json:"account_holder"`
}

type clubPlain struct {
	GlaeubigerID string `json:"glaeubiger_id"`
	IBAN         string `json:"iban"`
	BIC          string `json:"bic"`
	Kontoinhaber string `json:"kontoinhaber"`
}

type mandatePlain struct {
	MemberID  int    `json:"member_id"`
	PDFBase64 string `json:"pdf_base64"`
}

type dataResp struct {
	Members  []memberPlain  `json:"members"`
	Club     *clubPlain     `json:"club"`
	Mandates []mandatePlain `json:"mandates"`
}

// Data liefert den über die Brücke entschlüsselten Altbestand (Klartext über TLS) zur
// clientseitigen Re-Verschlüsselung. Nur verfügbar, solange die Brücke aktiv ist (404 sonst,
// „nur wenn Bridge"). Liefert ausschließlich noch nicht migrierte Datensätze.
func (h *Handler) Data(w http.ResponseWriter, r *http.Request) {
	if !crypto.HasKey() {
		http.Error(w, "Migrations-Brücke nicht verfügbar (FIELD_ENCRYPTION_KEY nicht gesetzt)", http.StatusNotFound)
		return
	}
	ctx := r.Context()
	out := dataResp{Members: []memberPlain{}, Mandates: []mandatePlain{}}

	// Mitglieds-Bankdaten.
	rows, err := h.db.QueryContext(ctx,
		`SELECT id, iban, account_holder FROM members WHERE iban IS NOT NULL OR account_holder IS NOT NULL`)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var id int
		var iban, holder sql.NullString
		if err := rows.Scan(&id, &iban, &holder); err != nil {
			rows.Close()
			http.Error(w, "DB-Fehler", http.StatusInternalServerError)
			return
		}
		pi, e1 := crypto.Decrypt(iban.String)
		ph, e2 := crypto.Decrypt(holder.String)
		if e1 != nil || e2 != nil {
			rows.Close()
			http.Error(w, "Entschlüsselung fehlgeschlagen (member)", http.StatusInternalServerError)
			return
		}
		out.Members = append(out.Members, memberPlain{MemberID: id, IBAN: pi, AccountHolder: ph})
	}
	rows.Close()

	// Vereins-SEPA-Stammdaten.
	var g, i, b, k sql.NullString
	h.db.QueryRowContext(ctx, `SELECT glaeubiger_id, iban, bic, kontoinhaber FROM clubs LIMIT 1`).
		Scan(&g, &i, &b, &k)
	if g.String != "" || i.String != "" || b.String != "" || k.String != "" {
		pg, e1 := crypto.Decrypt(g.String)
		pi, e2 := crypto.Decrypt(i.String)
		pb, e3 := crypto.Decrypt(b.String)
		pk, e4 := crypto.Decrypt(k.String)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			http.Error(w, "Entschlüsselung fehlgeschlagen (club)", http.StatusInternalServerError)
			return
		}
		out.Club = &clubPlain{GlaeubigerID: pg, IBAN: pi, BIC: pb, Kontoinhaber: pk}
	}

	// SEPA-Mandat-PDFs (noch ohne dek_enc → unmigriert).
	mrows, err := h.db.QueryContext(ctx,
		`SELECT id, sepa_mandat_path FROM members WHERE sepa_mandat_path IS NOT NULL AND sepa_mandat_path <> ''
		 AND (sepa_mandat_dek_enc IS NULL OR sepa_mandat_dek_enc = '')`)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	type mandRow struct {
		id   int
		path string
	}
	var mands []mandRow
	for mrows.Next() {
		var mr mandRow
		if err := mrows.Scan(&mr.id, &mr.path); err != nil {
			mrows.Close()
			http.Error(w, "DB-Fehler", http.StatusInternalServerError)
			return
		}
		mands = append(mands, mr)
	}
	mrows.Close()
	for _, mr := range mands {
		raw, err := os.ReadFile(filepath.Join(h.uploadDir, mr.path))
		if err != nil {
			if os.IsNotExist(err) {
				continue // verwaister Pfad — überspringen
			}
			http.Error(w, "Datei-Lesefehler", http.StatusInternalServerError)
			return
		}
		// Bereits clientseitig verschlüsselt (Client-Magic) heißt: kein Brücken-Decrypt möglich
		// — sollte mit dem retry-sicheren Upload (neue Datei erst bei Commit) nicht auftreten.
		if crypto.IsClientEncryptedBytes(raw) {
			http.Error(w, "Mandat-Datei bereits clientseitig verschlüsselt, aber dek_enc fehlt", http.StatusConflict)
			return
		}
		pdf, err := crypto.DecryptBytes(raw)
		if err != nil {
			http.Error(w, "Entschlüsselung fehlgeschlagen (mandat)", http.StatusInternalServerError)
			return
		}
		out.Mandates = append(out.Mandates, mandatePlain{MemberID: mr.id, PDFBase64: base64.StdEncoding.EncodeToString(pdf)})
	}

	writeJSON(w, out)
}

// ---- POST /api/admin/migrate-legacy/upload ----

type uploadReq struct {
	Members []struct {
		MemberID       int    `json:"member_id"`
		BankCiphertext string `json:"bank_ciphertext"`
		BankDekEnc     string `json:"bank_dek_enc"`
	} `json:"members"`
	Club *struct {
		SepaCiphertext string `json:"sepa_ciphertext"`
		SepaDekEnc     string `json:"sepa_dek_enc"`
	} `json:"club"`
	Mandates []struct {
		MemberID   int    `json:"member_id"`
		BlobBase64 string `json:"blob_base64"`
		DekEnc     string `json:"dek_enc"`
	} `json:"mandates"`
}

// Upload nimmt die clientseitig erzeugten Envelopes entgegen und schreibt sie. Pro Datensatz
// wird in EINER Transaktion der Envelope geschrieben UND die Legacy-`v1:`-Spalte genullt
// (idempotent + self-disabling). SEPA-Mandat-Blobs werden retry-sicher in eine NEUE Datei
// geschrieben (die alte Legacy-Datei bleibt bis zum Commit unangetastet). Nur während der
// Migration verfügbar (404 ohne Brücke).
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if !crypto.HasKey() {
		http.Error(w, "Migrations-Brücke nicht verfügbar (FIELD_ENCRYPTION_KEY nicht gesetzt)", http.StatusNotFound)
		return
	}
	var req uploadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	// Mandat-Blobs vorab dekodieren + validieren und als neue Dateien schreiben. Wir tracken
	// neue (bei Rollback zu entfernen) und alte Pfade (bei Commit zu entfernen).
	var newFiles, oldFiles []string
	cleanup := func(paths []string) {
		for _, p := range paths {
			os.Remove(filepath.Join(h.uploadDir, p))
		}
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Transaktionsfehler", http.StatusInternalServerError)
		return
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback() //nolint:errcheck
			cleanup(newFiles)
		}
	}()

	// Mitglieds-Bankdaten.
	for _, m := range req.Members {
		if m.BankCiphertext == "" || m.BankDekEnc == "" {
			http.Error(w, "Envelope unvollständig (member)", http.StatusBadRequest)
			return
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, ?, ?)
			 ON CONFLICT(member_id) DO UPDATE SET ciphertext=excluded.ciphertext, dek_enc_vorstand=excluded.dek_enc_vorstand`,
			m.MemberID, m.BankCiphertext, m.BankDekEnc); err != nil {
			http.Error(w, "Speicherfehler (member)", http.StatusInternalServerError)
			return
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE members SET iban=NULL, account_holder=NULL WHERE id=?`, m.MemberID); err != nil {
			http.Error(w, "Speicherfehler (member legacy)", http.StatusInternalServerError)
			return
		}
	}

	// Vereins-SEPA-Stammdaten.
	if req.Club != nil {
		if req.Club.SepaCiphertext == "" || req.Club.SepaDekEnc == "" {
			http.Error(w, "Envelope unvollständig (club)", http.StatusBadRequest)
			return
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE clubs SET sepa_ciphertext=?, sepa_dek_enc=?, glaeubiger_id=NULL, iban=NULL, bic=NULL, kontoinhaber=NULL, updated_at=?
			 WHERE id=(SELECT id FROM clubs LIMIT 1)`,
			req.Club.SepaCiphertext, req.Club.SepaDekEnc, time.Now()); err != nil {
			http.Error(w, "Speicherfehler (club)", http.StatusInternalServerError)
			return
		}
	}

	// SEPA-Mandat-PDFs.
	for _, md := range req.Mandates {
		if md.DekEnc == "" {
			http.Error(w, "Envelope unvollständig (mandat dek_enc)", http.StatusBadRequest)
			return
		}
		blob, err := base64.StdEncoding.DecodeString(md.BlobBase64)
		if err != nil {
			http.Error(w, "ungültiges base64 (mandat)", http.StatusBadRequest)
			return
		}
		if !crypto.IsClientEncryptedBytes(blob) {
			http.Error(w, "kein clientseitig verschlüsselter Blob (mandat)", http.StatusBadRequest)
			return
		}
		var oldPath sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT sepa_mandat_path FROM members WHERE id=?`, md.MemberID).Scan(&oldPath); err != nil {
			http.Error(w, "Mitglied nicht gefunden (mandat)", http.StatusBadRequest)
			return
		}
		newRel := filepath.Join("sepa-mandats", uuid.NewString()+".bin")
		dir := filepath.Join(h.uploadDir, "sepa-mandats")
		if err := os.MkdirAll(dir, 0755); err != nil {
			http.Error(w, "Datei-Schreibfehler (mandat)", http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(filepath.Join(h.uploadDir, newRel), blob, 0644); err != nil {
			http.Error(w, "Datei-Schreibfehler (mandat)", http.StatusInternalServerError)
			return
		}
		newFiles = append(newFiles, newRel)
		if oldPath.Valid && oldPath.String != "" && oldPath.String != newRel {
			oldFiles = append(oldFiles, oldPath.String)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE members SET sepa_mandat_path=?, sepa_mandat_dek_enc=? WHERE id=?`,
			newRel, md.DekEnc, md.MemberID); err != nil {
			http.Error(w, "Speicherfehler (mandat)", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Commit-Fehler", http.StatusInternalServerError)
		return
	}
	committed = true
	cleanup(oldFiles) // alte Legacy-Dateien erst nach erfolgreichem Commit entfernen

	h.hub.Broadcast("members")
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
