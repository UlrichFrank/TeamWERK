package h4aimport

import "testing"

func TestParseStaffel(t *testing.T) {
	tests := []struct {
		staffel  string
		gender   string
		ageClass string
		ok       bool
	}{
		{"mC-OL-3-BW", "m", "C-Jugend", true},
		{"mB-OL-3-BW", "m", "B-Jugend", true},
		{"mA-BOL-SRM", "m", "A-Jugend", true},
		{"gD-BOL-SRM", "mixed", "D-Jugend", true},
		{"wB-BZL-SRM", "f", "B-Jugend", true}, // weiblich → teams.gender "f"
		{"  mC-OL-3-BW  ", "m", "C-Jugend", true},
		// Muster nicht erfüllt → kein Vorschlag statt Rateversuch.
		{"Pokalrunde", "", "", false},
		{"xC-OL-3-BW", "", "", false},
		{"mZ-OL-3-BW", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		g, ac, ok := ParseStaffel(tt.staffel)
		if ok != tt.ok || g != tt.gender || ac != tt.ageClass {
			t.Errorf("ParseStaffel(%q) = (%q,%q,%v), erwartet (%q,%q,%v)",
				tt.staffel, g, ac, ok, tt.gender, tt.ageClass, tt.ok)
		}
	}
}

func TestLeagueRank(t *testing.T) {
	// Die Ligenpyramide der Jugend, von oben nach unten. Höhere Liga = kleinerer
	// Rang; die Reihenfolge trägt die Regel „Team Stuttgart 2 spielt in der
	// niedrigeren Staffel".
	pyramide := []struct{ kuerzel, staffel string }{
		{"JBLH (Bundesliga)", "mA-JBLH-1"},
		{"RL (Regionalliga)", "mA-RL-BW"},
		{"OL (Oberliga)", "mC-OL-3-BW"},
		{"BOL (Bezirksoberliga)", "mA-BOL-SRM"},
		{"BZL (Bezirksliga)", "wB-BZL-SRM"},
		{"BZK (Bezirksklasse)", "mC-BZK-SRM"},
		{"KK (Kreisklasse)", "mC-KK-SRM"},
	}
	prev := -1
	for _, stufe := range pyramide {
		r, ok := LeagueRank(stufe.staffel)
		if !ok {
			t.Fatalf("%s nicht erkannt (%q)", stufe.kuerzel, stufe.staffel)
		}
		if r <= prev {
			t.Errorf("%s hat Rang %d, muss größer als die Stufe darüber (%d) sein", stufe.kuerzel, r, prev)
		}
		prev = r
	}

	// Unbekanntes Kürzel meldet sich als solches, statt einen Rang zu erfinden:
	//   VL/KL — Verbands- und Kreisliga sind für diesen Verein nicht relevant.
	//   BL    — als Kürzel wäre Bundesliga wie Bezirksliga lesbar; eine Fehldeutung
	//           würde die Rangfolge umkehren.
	//   LL    — Landesliga existiert in der Jugendpyramide nicht.
	for _, s := range []string{"mB-VL-BW", "mC-KL-SRM", "mA-BL-1", "mA-LL-BW", "mC-XYZ-SRM", "Pokalrunde", ""} {
		if _, ok := LeagueRank(s); ok {
			t.Errorf("LeagueRank(%q) sollte unbekannt sein", s)
		}
	}
}

func TestTeamNumberFromAliasUndName(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"Team Stuttgart", 1},
		{"Team Stuttgart 2", 2},
		{"Team Stuttgart 3", 3},
		{"B-Jugend männlich", 1},
		{"B-Jugend männlich 2", 2},
		{"A-Jugend männlich 1", 1},
		{"", 1},
	}
	for _, tt := range tests {
		if got := TeamNumberFromAlias(tt.in); got != tt.want {
			t.Errorf("TeamNumberFromAlias(%q) = %d, erwartet %d", tt.in, got, tt.want)
		}
		if got := TeamNumberFromName(tt.in); got != tt.want {
			t.Errorf("TeamNumberFromName(%q) = %d, erwartet %d", tt.in, got, tt.want)
		}
	}
}
