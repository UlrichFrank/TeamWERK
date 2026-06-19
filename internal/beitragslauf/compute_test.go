package beitragslauf

import (
	"testing"
	"time"
)

func TestAktivKategorie_MitOhneStammverein(t *testing.T) {
	if got := AktivKategorie(true); got != "aktiv_mit" {
		t.Errorf("AktivKategorie(true) = %q", got)
	}
	if got := AktivKategorie(false); got != "aktiv_ohne" {
		t.Errorf("AktivKategorie(false) = %q", got)
	}
}

func TestBeitragsGruppe_AlleStatus(t *testing.T) {
	cases := map[string]string{
		"aktiv":       "aktiv",
		"verletzt":    "aktiv",
		"pausiert":    "passiv",
		"passiv":      "passiv",
		"ausgetreten": "",
		"honorar":     "",
		"anwaerter":   "",
	}
	for status, want := range cases {
		if got := BeitragsGruppe(status); got != want {
			t.Errorf("BeitragsGruppe(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestMatchHomeClub(t *testing.T) {
	if m := MatchHomeClub("TV Cannstatt 1846"); !m.Matched || m.Warning != "" {
		t.Errorf("exakt: %+v", m)
	}
	if m := MatchHomeClub("tv cannstatt 1846"); !m.Matched || m.Warning != "" {
		t.Errorf("lowercase: %+v", m)
	}
	if m := MatchHomeClub("TV Cannstadt 1846"); !m.Matched || m.Warning == "" {
		t.Errorf("fuzzy: %+v", m)
	}
	if m := MatchHomeClub(""); m.Matched || m.Warning != "" {
		t.Errorf("leer: %+v", m)
	}
	if m := MatchHomeClub("FC Bayern"); m.Matched || m.Warning == "" {
		t.Errorf("unbekannt: %+v", m)
	}
}

func TestLookupBetragCent(t *testing.T) {
	d := func(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }
	saetze := map[string][]Satz{
		"aktiv_mit": {
			{Kategorie: "aktiv_mit", BetragCent: 10000, ValidFrom: d("2027-07-01")},
			{Kategorie: "aktiv_mit", BetragCent: 9600, ValidFrom: d("2026-07-01")},
		},
	}
	// Saison 2026 → 9600
	if got, _ := LookupBetragCent(saetze, "aktiv_mit", d("2026-07-01")); got != 9600 {
		t.Errorf("2026: %d, want 9600", got)
	}
	// Saison 2027 → 10000
	if got, _ := LookupBetragCent(saetze, "aktiv_mit", d("2027-07-01")); got != 10000 {
		t.Errorf("2027: %d, want 10000", got)
	}
	// vor erstem valid_from → Error
	if _, err := LookupBetragCent(saetze, "aktiv_mit", d("2025-07-01")); err == nil {
		t.Error("vor valid_from: erwarteter Error fehlt")
	}
}
