package beitragslauf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProtokollResult ist das Ergebnis eines Einzugs für ein Mitglied.
type ProtokollResult struct {
	MemberNumber string
	Name         string
	BetragCent   int
	Success      bool
}

// protokollPfad bildet den Dateinamen pro Saisonjahr: beitragslauf_2026-2027.txt
func protokollPfad(dir, saisonKurz string) string {
	safe := strings.NewReplacer("/", "-", " ", "", string(filepath.Separator), "-").Replace(saisonKurz)
	return filepath.Join(dir, "beitragslauf_"+safe+".txt")
}

// AppendProtokoll hängt einen Block an die Saison-Protokolldatei an
// (append-only, bestehende Blöcke bleiben unverändert).
func AppendProtokoll(dir, saisonKurz, user string, at time.Time, results []ProtokollResult) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := protokollPfad(dir, saisonKurz)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var ok, fail []ProtokollResult
	var sumOK int
	for _, r := range results {
		if r.Success {
			ok = append(ok, r)
			sumOK += r.BetragCent
		} else {
			fail = append(fail, r)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "=== Lauf bestätigt %s durch %s ===\n", at.UTC().Format(time.RFC3339), user)
	fmt.Fprintf(&b, "Erfolgreich (%d) — Summe %s €\n", len(ok), euroComma(sumOK))
	for _, r := range ok {
		fmt.Fprintf(&b, "  Mitgl.-Nr %-8s %-24s %10s €\n", r.MemberNumber, r.Name, euroComma(r.BetragCent))
	}
	if len(fail) > 0 {
		fmt.Fprintf(&b, "Nicht erfolgreich (%d)\n", len(fail))
		for _, r := range fail {
			fmt.Fprintf(&b, "  Mitgl.-Nr %-8s %-24s %10s €\n", r.MemberNumber, r.Name, euroComma(r.BetragCent))
		}
	}
	b.WriteString("\n")

	_, err = f.WriteString(b.String())
	return err
}

// ReadProtokoll liefert den Inhalt der Saison-Protokolldatei.
// Existiert keine Datei, wird ("", nil) zurückgegeben.
func ReadProtokoll(dir, saisonKurz string) ([]byte, error) {
	data, err := os.ReadFile(protokollPfad(dir, saisonKurz))
	if os.IsNotExist(err) {
		return []byte{}, nil
	}
	return data, err
}

// euroComma formatiert Cent mit Komma-Dezimaltrenner: 9600 → "96,00".
func euroComma(cent int) string {
	if cent < 0 {
		cent = -cent
	}
	return fmt.Sprintf("%d,%02d", cent/100, cent%100)
}
