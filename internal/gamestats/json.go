package gamestats

import "encoding/json"

// decodeWarnings liest die gespeicherte Warnungsliste. Ein unlesbarer Wert
// ergibt eine leere Liste statt eines Fehlers: die Warnungen sind ein
// Zusatzhinweis, kein tragender Teil des Berichts.
func decodeWarnings(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
