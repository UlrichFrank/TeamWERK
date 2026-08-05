package h4aimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "gametable_owngames.html"))
	if err != nil {
		t.Fatalf("Fixture nicht lesbar: %v", err)
	}
	return string(b)
}

func TestParseGames_Fixture(t *testing.T) {
	games, err := ParseGames(loadFixture(t))
	if err != nil {
		t.Fatalf("ParseGames: unerwarteter Fehler: %v", err)
	}
	if len(games) != 5 {
		t.Fatalf("erwartet 5 Spiele, bekommen %d", len(games))
	}

	byNo := map[string]RawGame{}
	for _, g := range games {
		byNo[g.GameNo] = g
	}

	// Heimspiel: eigenes Team im Heim-Feld.
	g := byNo["905996"]
	if !g.IsHome {
		t.Errorf("905996: IsHome=false, erwartet true")
	}
	if !strings.Contains(g.Home, "Team Stuttgart") || strings.Contains(g.Home, "2") {
		t.Errorf("905996: Home=%q, erwartet 'Team Stuttgart' ohne '2'", g.Home)
	}
	if g.Guest != "Bregenz Handb." {
		t.Errorf("905996: Guest=%q, erwartet 'Bregenz Handb.'", g.Guest)
	}
	if g.HallNumber != "3059" {
		t.Errorf("905996: HallNumber=%q, erwartet '3059'", g.HallNumber)
	}
	if g.Date != "2026-09-26" {
		t.Errorf("905996: Date=%q, erwartet '2026-09-26'", g.Date)
	}
	if g.Time != "14:45" {
		t.Errorf("905996: Time=%q, erwartet '14:45'", g.Time)
	}

	// Auswärtsspiel: eigenes Team im Gast-Feld.
	g = byNo["211004"]
	if g.IsHome {
		t.Errorf("211004: IsHome=true, erwartet false")
	}
	if !strings.Contains(g.Guest, "Team Stuttgart 2") {
		t.Errorf("211004: Guest=%q, erwartet 'Team Stuttgart 2'", g.Guest)
	}
	if g.HallNumber != "3029" {
		t.Errorf("211004: HallNumber=%q, erwartet '3029'", g.HallNumber)
	}
	if g.Date != "2026-09-19" {
		t.Errorf("211004: Date=%q, erwartet '2026-09-19'", g.Date)
	}
	if g.InternalID != "9715601" {
		t.Errorf("211004: InternalID=%q, erwartet '9715601'", g.InternalID)
	}
	if g.Staffel != "mA-BOL-SRM" {
		t.Errorf("211004: Staffel=%q, erwartet 'mA-BOL-SRM'", g.Staffel)
	}
}

func TestParseGames_NoGamesIsError(t *testing.T) {
	if _, err := ParseGames("<div>kein spiel</div>"); err == nil {
		t.Fatal("erwartet Fehler bei fehlender Spieltabelle, bekommen nil")
	}
}
