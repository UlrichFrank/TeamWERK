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

// ErrInvalidVerhaeltnis signalisiert einen nicht-positiven Wert beim
// Schreiben — der Aufrufer (Handler oder Regen-Engine) übersetzt das in ein
// HTTP 400 bzw. lehnt den Aufruf ab, ohne zu persistieren.
var ErrInvalidVerhaeltnis = errors.New("bewirtung_verhaeltnis muss > 0 sein")

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
