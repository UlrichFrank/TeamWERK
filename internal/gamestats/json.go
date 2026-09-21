package gamestats

import "encoding/json"

// decodeWarnings liest die gespeicherte Warnungsliste. Ein unlesbarer Wert
// ergibt eine leere Liste statt eines Fehlers: die Warnungen sind ein
// Zusatzhinweis, kein tragender Teil des Berichts.
func decodeWarnings(raw string) []string {
	return decodeStrings(raw)
}

// decodeStrings liest eine gespeicherte JSON-Liste von Zeichenketten.
func decodeStrings(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// encodeStrings schreibt eine Liste von Zeichenketten. Eine leere Liste wird
// als leerer String gespeichert, nicht als "[]" — das ist der Default der
// Spalte und macht "keine Namen" und "noch nicht getrennt" in der Abfrage
// ununterscheidbar, was beide Fälle richtig behandelt: keine Zeile in der
// Schiedsrichter-Rangliste.
func encodeStrings(v []string) string {
	if len(v) == 0 {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
