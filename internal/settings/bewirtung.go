package settings

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

// keyBewirtungVerhaeltnis ist der system_settings-Key für das vereinsweite
// Spiele-zu-Kuchen-Verhältnis (Bewirtungs-/Kuchendienst-Rotation). Die
// Default-Row ('1') legt Migration 045 idempotent an.
const keyBewirtungVerhaeltnis = "bewirtung_verhaeltnis"

// keyBewirtungMaxPerTeam ist der system_settings-Key für die vereinsweite
// Obergrenze "Max. Kuchen pro Mannschaft". Sie lag früher als
// game_template_items.rotation_max_per_team am einzelnen Vorlagen-Item; seit
// bewirtung-cap-global ist sie eine Vereinsregel neben dem Verhältnis, und das
// Item trägt nur noch den Schalter rotation_enabled. Die Row legt Migration 046
// idempotent an (Startwert aus dem größten Alt-Cap, sonst '1').
const keyBewirtungMaxPerTeam = "bewirtung_max_per_team"

// ErrInvalidVerhaeltnis signalisiert einen nicht-positiven Wert beim
// Schreiben — der Aufrufer (Handler oder Regen-Engine) übersetzt das in ein
// HTTP 400 bzw. lehnt den Aufruf ab, ohne zu persistieren.
var ErrInvalidVerhaeltnis = errors.New("bewirtung_verhaeltnis muss > 0 sein")

// ErrInvalidMaxPerTeam ist das Pendant für den Cap: < 1 ergibt fachlich keinen
// Slot und würde die Rotation still leerlaufen lassen.
var ErrInvalidMaxPerTeam = errors.New("bewirtung_max_per_team muss > 0 sein")

// RowQuerier ist die kleinste gemeinsame Lese-Schnittstelle von *sql.DB und
// *sql.Tx. Die Regen-Engine liest das Verhältnis innerhalb ihrer laufenden
// Transaktion (design.md Decision 6: Konsistenz mit der Transaktion statt
// eines zweiten Verbindungs-Reads), der HTTP-Handler direkt über *sql.DB.
type RowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// GetBewirtungVerhaeltnis liest das Spiele-zu-Kuchen-Verhältnis direkt aus
// system_settings. Bewusst OHNE Store/Cache (design.md, Decision 6 aus
// kuchendienst-rotation): der Wert wird nur einmal pro Regen-Lauf gelesen,
// kein Hot-Path — ein einfacher SELECT reicht, ohne In-Memory-Konsistenz-
// Sorgen gegenüber einer laufenden Transaktion.
//
// Fehlt die Row (z. B. in einer DB ohne Migration 045), liefert die Funktion
// den fachlichen Default 1 statt eines Fehlers — bestehendes Verhalten
// (ein Slot pro Team) bleibt dann unverändert.
func GetBewirtungVerhaeltnis(ctx context.Context, db RowQuerier) (float64, error) {
	var raw string
	err := db.QueryRowContext(ctx,
		`SELECT value FROM system_settings WHERE key = ?`, keyBewirtungVerhaeltnis,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// SetBewirtungVerhaeltnis validiert (> 0) und persistiert das Verhältnis als
// String. updatedBy ist die User-ID des schreibenden Vorstands (0 → NULL).
// Bei ungültigem Wert wird NICHTS geschrieben (ErrInvalidVerhaeltnis).
func SetBewirtungVerhaeltnis(ctx context.Context, db *sql.DB, value float64, updatedBy int) error {
	if value <= 0 {
		return ErrInvalidVerhaeltnis
	}
	var updatedByArg any
	if updatedBy > 0 {
		updatedByArg = updatedBy
	}
	_, err := db.ExecContext(ctx,
		`UPDATE system_settings
		 SET value = ?, updated_at = CURRENT_TIMESTAMP, updated_by = ?
		 WHERE key = ?`,
		strconv.FormatFloat(value, 'f', -1, 64), updatedByArg, keyBewirtungVerhaeltnis)
	return err
}

// GetBewirtungMaxPerTeam liest die vereinsweite Obergrenze pro Mannschaft.
// Gleiche Bauweise wie GetBewirtungVerhaeltnis (kein Store/Cache, einmal pro
// Regen-Lauf gelesen, RowQuerier damit die Regen-Engine innerhalb ihrer eigenen
// Transaktion lesen kann).
//
// Fehlt die Row (DB ohne Migration 046), liefert die Funktion den fachlichen
// Default 1 statt eines Fehlers.
func GetBewirtungMaxPerTeam(ctx context.Context, db RowQuerier) (int, error) {
	var raw string
	err := db.QueryRowContext(ctx,
		`SELECT value FROM system_settings WHERE key = ?`, keyBewirtungMaxPerTeam,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// SetBewirtungMaxPerTeam validiert (> 0) und persistiert den Cap.
// updatedBy ist die User-ID des schreibenden Vorstands (0 → NULL). Bei
// ungültigem Wert wird NICHTS geschrieben (ErrInvalidMaxPerTeam).
func SetBewirtungMaxPerTeam(ctx context.Context, db *sql.DB, value, updatedBy int) error {
	if value <= 0 {
		return ErrInvalidMaxPerTeam
	}
	var updatedByArg any
	if updatedBy > 0 {
		updatedByArg = updatedBy
	}
	_, err := db.ExecContext(ctx,
		`UPDATE system_settings
		 SET value = ?, updated_at = CURRENT_TIMESTAMP, updated_by = ?
		 WHERE key = ?`,
		strconv.Itoa(value), updatedByArg, keyBewirtungMaxPerTeam)
	return err
}
