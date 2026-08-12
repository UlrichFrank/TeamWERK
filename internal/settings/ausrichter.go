package settings

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Ausrichter ist der Verein, der einen Heim-Spieltag organisiert (Halle stellt,
// Bewirtung verantwortet). Die Liste ist vereinsweit; genau ein Eintrag trägt
// is_default (Partial-Unique-Index idx_ausrichter_default, Migration 048).
type Ausrichter struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Aktiv     bool   `json:"aktiv"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
}

// AusrichterInput ist die Eingabe für CreateAusrichter. Aktiv fehlt bewusst:
// ein frisch angelegter Ausrichter ist immer aktiv (Vorbild stammvereine).
type AusrichterInput struct {
	Name      string
	SortOrder int
	IsDefault bool
}

// AusrichterUpdate trägt Pointer-Semantik: ein nil-Feld bleibt unverändert.
// Gleiches Muster wie SetBewirtung (handler.go) — und aus demselben Grund
// werden in UpdateAusrichter ALLE Felder validiert, BEVOR der erste
// Schreibvorgang läuft, damit ein Fehler keine Teil-Persistenz hinterlässt.
type AusrichterUpdate struct {
	Name      *string
	Aktiv     *bool
	SortOrder *int
	IsDefault *bool
}

// AusrichterUsageReport benennt, was am Ausrichter hängt — die Vorab-Anzeige
// vor dem Löschen. Beide Listen verhalten sich beim Löschen unterschiedlich
// (siehe DeleteAusrichter), deshalb müssen sie getrennt sichtbar sein.
type AusrichterUsageReport struct {
	GameDays      []AusrichterGameDay      `json:"game_days"`
	TemplateItems []AusrichterTemplateItem `json:"template_items"`
}

// AusrichterGameDay ist ein Spieltag mit explizit gesetztem Ausrichter. Diese
// Tage überleben das Löschen und fallen auf den Default zurück.
type AusrichterGameDay struct {
	Date       string `json:"date"`
	SeasonID   int    `json:"season_id"`
	SeasonName string `json:"season_name"`
}

// AusrichterTemplateItem ist eine an den Ausrichter gebundene Vorlagen-Zeile.
// Diese Zeilen werden beim Löschen MITGELÖSCHT — deshalb tragen sie hier
// Vorlagen- und Dienst-Namen, damit der Vorstand vor dem Bestätigen sieht,
// welche Dienste verschwinden.
type AusrichterTemplateItem struct {
	ID           int    `json:"id"`
	TemplateID   int    `json:"template_id"`
	TemplateName string `json:"template_name"`
	DutyTypeName string `json:"duty_type_name"`
}

var (
	// ErrAusrichterNotFound: unbekannte ID → der Aufrufer übersetzt in 404
	// (CRUD) bzw. 400 (Bindung an eine Vorlagen-Zeile oder einen Spieltag).
	ErrAusrichterNotFound = errors.New("ausrichter nicht gefunden")

	// ErrDuplicateName: der Name ist bereits vergeben (UNIQUE(name)). Der
	// Aufrufer übersetzt in HTTP 409, ohne dass etwas geschrieben wurde.
	ErrDuplicateName = errors.New("ausrichter mit diesem Namen existiert bereits")

	// ErrEmptyName: leerer bzw. nur aus Whitespace bestehender Name → 400.
	ErrEmptyName = errors.New("name darf nicht leer sein")

	// ErrDefaultUndeletable: der Default-Eintrag ist nicht löschbar (HTTP 409
	// default_ausrichter_undeletable). Ohne ihn wäre die Auflösung aus
	// design.md Decision 2 nicht mehr total — jeder Spieltag ohne expliziten
	// Eintrag hätte plötzlich gar keinen Ausrichter. Erst einen anderen
	// Eintrag zum Default machen, dann löschen.
	ErrDefaultUndeletable = errors.New("der Default-Ausrichter ist nicht löschbar")

	// ErrDefaultRequired: der Default-Eintrag darf seine Markierung nicht
	// verlieren und nicht deaktiviert werden — beides nur, indem ein ANDERER
	// Eintrag zum Default gemacht wird. Zwei Fälle, eine Begründung:
	//   - is_default=0 auf dem aktuellen Default ließe null Defaults zurück;
	//     die Auflösung wäre nicht mehr total (dasselbe Loch wie beim Löschen).
	//   - aktiv=0 auf dem Default hielte die Auflösung zwar formal total, würde
	//     aber jeden Spieltag ohne expliziten Eintrag auf einen inaktiven
	//     Ausrichter auflösen — einen Wert, den man über die Preview/Apply-Route
	//     nicht einmal explizit setzen dürfte (dort ist inaktiv ein 400).
	//     Diesen Widerspruch schließen wir hier, statt ihn zu tolerieren.
	// Der Aufrufer übersetzt in HTTP 409.
	ErrDefaultRequired = errors.New("es muss immer genau einen aktiven Default-Ausrichter geben")

	// ErrNoDefaultAusrichter signalisiert, dass gar keine Default-Zeile
	// existiert. Das ist ein Datenfehler, kein Betriebszustand: Migration 048
	// legt die Zeile idempotent an, und alle Schreibpfade hier halten die
	// Invariante. Bewusst ANDERS als GetBewirtungVerhaeltnis, das bei fehlender
	// Row den fachlichen Default 1 liefert — dort gibt es einen sinnvollen
	// Ersatzwert ("ein Slot pro Team"), hier nicht: eine erfundene ID 0 würde
	// gegen kein Item matchen und das Ausrichter-Gate still alles ausfiltern.
	// Lieber laut scheitern als still den halben Dienstplan verschlucken.
	ErrNoDefaultAusrichter = errors.New("kein Default-Ausrichter gesetzt")
)

// Querier ist RowQuerier plus Mehrzeilen-Abfragen. Wie RowQuerier erfüllen ihn
// sowohl *sql.DB als auch *sql.Tx, damit Lesepfade wahlweise direkt oder
// innerhalb einer laufenden Transaktion laufen können.
type Querier interface {
	RowQuerier
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// dayKey normalisiert ein Datum auf die reine "2006-01-02"-Form.
//
// Grund ist der bekannte SQLite-DATE-Gotcha (docs/agent/06-gotchas.md): Spalten
// mit dem deklarierten Typ DATE kommen je nach Lesepfad als
// "2026-09-14T00:00:00Z" zurück. Ein solcher String als Vergleichsschlüssel
// matcht gegen ein gespeichertes "2026-09-14" nie — der Lauf tut dann still
// gar nichts. Dieselbe Verteidigung wie in dateWindow (internal/games/regen.go).
//
// Schreibpfade (HTTP-Handler) MÜSSEN das Datum ebenfalls in der reinen Form
// persistieren; die Normalisierung hier repariert nur die Lese-/Vergleichsseite.
func dayKey(date string) string {
	if len(date) > 10 {
		return date[:10]
	}
	return date
}

// ResolveAusrichterForDay liefert den für diesen Spieltag geltenden Ausrichter.
// Die Auflösung ist TOTAL (design.md Decision 2):
//
//	ausrichter(tag) = spieltag_ausrichter[(date, season_id)] ?? ausrichter.is_default
//
// Es gibt keinen Rückgabewert "kein Ausrichter". Nimmt — wie
// GetBewirtungVerhaeltnis — das schmale RowQuerier-Interface, damit die
// Regen-Engine innerhalb ihrer laufenden Transaktion liest. Bewusst ohne
// Store/Cache: ein SELECT je Regen-Tag, kein Hot-Path.
func ResolveAusrichterForDay(ctx context.Context, db RowQuerier, date string, seasonID int) (int, error) {
	id, _, err := ResolveAusrichterForDayDetailed(ctx, db, date, seasonID)
	return id, err
}

// ResolveAusrichterForDayDetailed ist dieselbe Auflösung, meldet zusätzlich
// aber, ob der Wert explizit für diesen Tag gesetzt war (isExplicit) oder vom
// Default geerbt wurde. Die HTTP-Schicht braucht das für is_explicit, der
// Kalender zeigt damit an, ob ein Tag geprüft wurde oder nur mitläuft.
//
// Eine einzige Query, weil sich die Total-Regel darin exakt abbilden lässt:
// die Default-Zeile ist die treibende Tabelle (genau eine, garantiert durch den
// Partial-Unique-Index), der Tageseintrag hängt als LEFT JOIN daran. Damit
// verhält sich ein Eintrag mit ausrichter_id IS NULL automatisch identisch zu
// einer fehlenden Zeile — beide lassen das COALESCE auf den Default fallen und
// beide liefern isExplicit=false, ohne einen eigenen Zweig zu brauchen. Fehlt
// die Default-Zeile, liefert der Join gar keine Row → ErrNoDefaultAusrichter.
func ResolveAusrichterForDayDetailed(ctx context.Context, db RowQuerier, date string, seasonID int) (int, bool, error) {
	var id, explicit int
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(sa.ausrichter_id, def.id),
		       CASE WHEN sa.ausrichter_id IS NOT NULL THEN 1 ELSE 0 END
		FROM (SELECT id FROM ausrichter WHERE is_default = 1) AS def
		LEFT JOIN spieltag_ausrichter AS sa
		       ON sa.date = ? AND sa.season_id = ?`,
		dayKey(date), seasonID,
	).Scan(&id, &explicit)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, ErrNoDefaultAusrichter
	}
	if err != nil {
		return 0, false, err
	}
	return id, explicit != 0, nil
}

// ListAusrichter liefert die Liste, sortiert wie die UI sie zeigt.
// includeInactive=false blendet deaktivierte Einträge aus (Vorbild
// stammvereine.List: der Default-Blick ist der auf die nutzbaren Einträge).
func ListAusrichter(ctx context.Context, db Querier, includeInactive bool) ([]Ausrichter, error) {
	query := `SELECT id, name, aktiv, is_default, sort_order FROM ausrichter`
	if !includeInactive {
		query += ` WHERE aktiv = 1`
	}
	query += ` ORDER BY sort_order, name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Ausrichter{}
	for rows.Next() {
		a, err := scanAusrichter(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// GetAusrichter liest einen einzelnen Eintrag; unbekannte ID →
// ErrAusrichterNotFound.
func GetAusrichter(ctx context.Context, db RowQuerier, id int) (Ausrichter, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, name, aktiv, is_default, sort_order FROM ausrichter WHERE id = ?`, id)
	a, err := scanAusrichter(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Ausrichter{}, ErrAusrichterNotFound
	}
	return a, err
}

// scanRow ist die gemeinsame Form von *sql.Row und *sql.Rows.
type scanRow interface {
	Scan(dest ...any) error
}

func scanAusrichter(row scanRow) (Ausrichter, error) {
	var a Ausrichter
	var aktiv, isDefault int
	if err := row.Scan(&a.ID, &a.Name, &aktiv, &isDefault, &a.SortOrder); err != nil {
		return Ausrichter{}, err
	}
	a.Aktiv = aktiv != 0
	a.IsDefault = isDefault != 0
	return a, nil
}

// CreateAusrichter legt einen Eintrag an (immer aktiv). Ein leerer Name wird
// vor dem Schreiben abgelehnt (ErrEmptyName), ein doppelter Name schlägt am
// UNIQUE-Index fehl und wird als ErrDuplicateName gemeldet — in beiden Fällen
// wurde nichts geschrieben.
//
// Mit IsDefault=true läuft der Default-Wechsel in DERSELBEN Transaktion wie das
// Insert (siehe clearDefaultTx).
func CreateAusrichter(ctx context.Context, db *sql.DB, in AusrichterInput) (Ausrichter, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Ausrichter{}, ErrEmptyName
	}

	var created Ausrichter
	err := withTx(ctx, db, func(tx *sql.Tx) error {
		if in.IsDefault {
			if err := clearDefaultTx(ctx, tx, 0); err != nil {
				return err
			}
		}
		res, err := tx.ExecContext(ctx,
			`INSERT INTO ausrichter (name, aktiv, is_default, sort_order) VALUES (?, 1, ?, ?)`,
			name, boolToInt(in.IsDefault), in.SortOrder)
		if err != nil {
			return mapUniqueErr(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		created = Ausrichter{
			ID:        int(id),
			Name:      name,
			Aktiv:     true,
			IsDefault: in.IsDefault,
			SortOrder: in.SortOrder,
		}
		return nil
	})
	if err != nil {
		return Ausrichter{}, err
	}
	return created, nil
}

// UpdateAusrichter ändert die im Update gesetzten Felder (nil = unverändert).
//
// ALLE Prüfungen laufen VOR dem ersten Schreibvorgang — dasselbe Muster wie
// SetBewirtung: ein abgelehntes Feld darf kein anderes, für sich gültiges Feld
// desselben Requests halb durchschreiben lassen. Weil die Namensdublette erst
// die DB am UNIQUE-Index bemerkt, laufen alle Schreibvorgänge zusätzlich in
// einer Transaktion, die dann komplett zurückrollt.
func UpdateAusrichter(ctx context.Context, db *sql.DB, id int, upd AusrichterUpdate) (Ausrichter, error) {
	var updated Ausrichter
	err := withTx(ctx, db, func(tx *sql.Tx) error {
		current, err := getAusrichterTx(ctx, tx, id)
		if err != nil {
			return err
		}

		// --- Validierung, komplett vor dem ersten Schreibvorgang ---
		name := current.Name
		if upd.Name != nil {
			name = strings.TrimSpace(*upd.Name)
			if name == "" {
				return ErrEmptyName
			}
		}
		// Trägt dieser Eintrag nach dem Update die Default-Markierung? Ein
		// explizites is_default=false auf dem aktuellen Default ist die einzige
		// Möglichkeit, die Markierung zu verlieren — und genau die ist gesperrt.
		willBeDefault := current.IsDefault
		if upd.IsDefault != nil {
			if !*upd.IsDefault && current.IsDefault {
				return ErrDefaultRequired
			}
			willBeDefault = willBeDefault || *upd.IsDefault
		}
		if willBeDefault && upd.Aktiv != nil && !*upd.Aktiv {
			return ErrDefaultRequired
		}

		// --- Schreiben ---
		if upd.IsDefault != nil && *upd.IsDefault && !current.IsDefault {
			if err := clearDefaultTx(ctx, tx, id); err != nil {
				return err
			}
		}
		sets := []string{}
		args := []any{}
		if upd.Name != nil {
			sets = append(sets, "name = ?")
			args = append(args, name)
		}
		if upd.Aktiv != nil {
			sets = append(sets, "aktiv = ?")
			args = append(args, boolToInt(*upd.Aktiv))
		}
		if upd.SortOrder != nil {
			sets = append(sets, "sort_order = ?")
			args = append(args, *upd.SortOrder)
		}
		if upd.IsDefault != nil && *upd.IsDefault {
			sets = append(sets, "is_default = 1")
		}
		if len(sets) > 0 {
			args = append(args, id)
			if _, err := tx.ExecContext(ctx,
				`UPDATE ausrichter SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
				return mapUniqueErr(err)
			}
		}

		updated, err = getAusrichterTx(ctx, tx, id)
		return err
	})
	if err != nil {
		return Ausrichter{}, err
	}
	return updated, nil
}

// DeleteAusrichter löscht einen Ausrichter und räumt seine beiden Referenzen
// auf — bewusst ASYMMETRISCH (design.md Decision 6). Das ist der subtilste
// Punkt des ganzen Changes:
//
//	spieltag_ausrichter.ausrichter_id → SET NULL
//	    Der Spieltag verliert nur seine Abweichung und fällt auf den Default
//	    zurück. Weil die Auflösung ein NULL exakt wie eine fehlende Zeile
//	    behandelt (Decision 2), ist der Zustand danach ohne Aufräumschritt
//	    korrekt. Das Löschen VERKLEINERT hier den Sonderfall — genau die
//	    Erwartung an ein Löschen.
//
//	game_template_items mit dieser ausrichter_id → DELETE (nicht SET NULL!)
//	    Ein SET NULL hieße auf einer Vorlagen-Zeile "gilt ab jetzt IMMER": die
//	    Zeile erzeugte nach dem Löschen an MEHR Spieltagen Dienste als vorher,
//	    genau umgekehrt zur Absicht. Ein Dienst, der nur existiert, weil Verein
//	    X ausrichtet, hat ohne Verein X keine Bedeutung — also verschwindet er
//	    mit. Das FK ON DELETE RESTRICT auf der Spalte (Migration 048) existiert
//	    genau dafür: es verbaut den stillen SET-NULL-Pfad und erzwingt, dass
//	    diese Zeilen VOR dem Löschen des Ausrichters aktiv entfernt werden.
//	    Deshalb ist die Reihenfolge unten nicht beliebig.
//
// Der Default-Eintrag ist nicht löschbar (ErrDefaultUndeletable), sonst wäre die
// Auflösung nicht mehr total. Alles läuft in EINER Transaktion: schlägt ein
// Schritt fehl, bleibt weder ein entkoppelter Spieltag noch eine gelöschte
// Vorlagen-Zeile zurück.
//
// Vor dem Aufruf sollte AusrichterUsage die Betroffenen benannt und der
// Vorstand bestätigt haben — die Vorlagen-Zeilen sind unwiederbringlich weg.
func DeleteAusrichter(ctx context.Context, db *sql.DB, id int) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		current, err := getAusrichterTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.IsDefault {
			return ErrDefaultUndeletable
		}

		// Explizit, obwohl der FK ON DELETE SET NULL dasselbe täte: die Absicht
		// steht damit im Code statt nur im Schema, und der Schritt hängt nicht
		// daran, ob PRAGMA foreign_keys auf dieser Verbindung aktiv ist.
		if _, err := tx.ExecContext(ctx,
			`UPDATE spieltag_ausrichter SET ausrichter_id = NULL WHERE ausrichter_id = ?`, id); err != nil {
			return err
		}
		// MUSS vor dem DELETE des Ausrichters laufen — der FK ist RESTRICT.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM game_template_items WHERE ausrichter_id = ?`, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM ausrichter WHERE id = ?`, id)
		return err
	})
}

// AusrichterUsage benennt, was an diesem Ausrichter hängt: die Spieltage mit
// explizitem Eintrag (die das Löschen überleben und auf den Default fallen) und
// die gebundenen Vorlagen-Zeilen (die MITGELÖSCHT werden). Genau diese
// Asymmetrie muss der Vorstand vor dem Bestätigen sehen.
func AusrichterUsage(ctx context.Context, db Querier, id int) (AusrichterUsageReport, error) {
	report := AusrichterUsageReport{
		GameDays:      []AusrichterGameDay{},
		TemplateItems: []AusrichterTemplateItem{},
	}
	if _, err := GetAusrichter(ctx, db, id); err != nil {
		return report, err
	}

	dayRows, err := db.QueryContext(ctx, `
		SELECT sa.date, sa.season_id, s.name
		FROM spieltag_ausrichter sa
		JOIN seasons s ON s.id = sa.season_id
		WHERE sa.ausrichter_id = ?
		ORDER BY sa.date`, id)
	if err != nil {
		return report, err
	}
	defer dayRows.Close()
	for dayRows.Next() {
		var d AusrichterGameDay
		if err := dayRows.Scan(&d.Date, &d.SeasonID, &d.SeasonName); err != nil {
			return report, err
		}
		// DATE-Spalte: kann als "2026-09-14T00:00:00Z" zurückkommen.
		d.Date = dayKey(d.Date)
		report.GameDays = append(report.GameDays, d)
	}
	if err := dayRows.Err(); err != nil {
		return report, err
	}

	itemRows, err := db.QueryContext(ctx, `
		SELECT gti.id, gti.template_id, gt.name, dt.name
		FROM game_template_items gti
		JOIN game_templates gt ON gt.id = gti.template_id
		JOIN duty_types    dt ON dt.id = gti.duty_type_id
		WHERE gti.ausrichter_id = ?
		ORDER BY gt.name, gti.sort_order`, id)
	if err != nil {
		return report, err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var it AusrichterTemplateItem
		if err := itemRows.Scan(&it.ID, &it.TemplateID, &it.TemplateName, &it.DutyTypeName); err != nil {
			return report, err
		}
		report.TemplateItems = append(report.TemplateItems, it)
	}
	return report, itemRows.Err()
}

// clearDefaultTx nimmt allen anderen Einträgen die Default-Markierung.
//
// Bewusst über das Prädikat WHERE is_default = 1 statt über eine vorher
// gelesene ID: der Code verlässt sich damit NICHT auf seine eigene Buchführung
// darüber, wer gerade Default ist. Selbst wenn diese Annahme falsch wäre
// (veralteter Lesestand, mehrere Zeilen durch einen Direkteingriff in die DB),
// räumt der UPDATE trotzdem alle Markierungen ab, bevor die neue gesetzt wird.
// Die eigentliche Garantie bleibt der Partial-Unique-Index idx_ausrichter_default
// (Migration 048): käme ein konkurrierender Schreiber dazwischen, bricht der
// zweite Schreibvorgang hart ab, statt still zwei Defaults nebeneinander
// bestehen zu lassen. keepID=0 schont keinen Eintrag (Create-Pfad).
func clearDefaultTx(ctx context.Context, tx *sql.Tx, keepID int) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE ausrichter SET is_default = 0 WHERE is_default = 1 AND id <> ?`, keepID)
	return err
}

func getAusrichterTx(ctx context.Context, tx *sql.Tx, id int) (Ausrichter, error) {
	row := tx.QueryRowContext(ctx,
		`SELECT id, name, aktiv, is_default, sort_order FROM ausrichter WHERE id = ?`, id)
	a, err := scanAusrichter(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Ausrichter{}, ErrAusrichterNotFound
	}
	return a, err
}

// mapUniqueErr übersetzt die UNIQUE-Verletzung auf ausrichter.name in den
// typisierten ErrDuplicateName (→ HTTP 409), wie in stammvereine über den
// Fehlerstring erkannt. Die Einschränkung auf "ausrichter.name" ist wichtig:
// auf derselben Tabelle liegt mit idx_ausrichter_default ein zweiter
// UNIQUE-Index, dessen Verletzung ein Programmierfehler wäre und deshalb roh
// durchgereicht wird, statt sie als Namensdublette zu verkleiden.
func mapUniqueErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE") && strings.Contains(msg, "ausrichter.name") {
		return ErrDuplicateName
	}
	return err
}

// withTx führt fn in einer Transaktion aus und rollt bei jedem Fehler zurück —
// damit hinterlässt ein abgelehnter Schreibpfad garantiert keine Teil-Persistenz.
func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // No-Op nach erfolgreichem Commit
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
