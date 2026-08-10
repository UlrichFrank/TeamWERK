package testutil

import (
	"database/sql"
	"strings"
	"testing"
)

// FindStringInDB durchsucht alle Tabellen und alle Spalten der Datenbank nach
// needle und liefert den ersten Fundort als (Tabelle, Spalte); ("", "") wenn
// nirgends gefunden.
//
// Werkzeug für Nichtpersistenz-Zusagen: „dieser Wert darf in keiner Zeile
// landen" ist nur dann eine harte Aussage, wenn wirklich alles durchsucht wird
// statt der Tabellen, an die man beim Schreiben des Tests gerade denkt. Jeder
// Aufrufer sollte zusätzlich eine Poison-Sanity fahren (bekannten Wert
// absichtlich setzen und wiederfinden) — ein stiller Scanner-Defekt meldet
// sonst dauerhaft „nicht gefunden".
func FindStringInDB(t *testing.T, db *sql.DB, needle string) (string, string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("FindStringInDB Tabellenliste: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	rows.Close()

	for _, tbl := range tables {
		r, err := db.Query(`SELECT * FROM "` + tbl + `"`)
		if err != nil {
			continue
		}
		cols, _ := r.Columns()
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				continue
			}
			for i, v := range vals {
				var s string
				switch tv := v.(type) {
				case string:
					s = tv
				case []byte:
					s = string(tv)
				default:
					continue
				}
				if strings.Contains(s, needle) {
					r.Close()
					return tbl, cols[i]
				}
			}
		}
		r.Close()
	}
	return "", ""
}
