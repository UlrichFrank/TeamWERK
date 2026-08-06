package games

import (
	"context"
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/h4aimport"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// insertTeam legt eine Mannschaft mit konkreter Altersklasse/Geschlecht an
// (testutil.CreateTeam setzt fest "Erwachsene"/"mixed", was für die
// Staffel-Ableitung nicht taugt).
func insertTeam(t *testing.T, db *sql.DB, name, ageClass, gender string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO teams (name, age_class, gender) VALUES (?,?,?)`, name, ageClass, gender)
	if err != nil {
		t.Fatalf("insertTeam(%q): %v", name, err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// besetzeKader gibt einer Mannschaft einen Kader mit einem Spieler in der Saison —
// das ist das Signal „spielt tatsächlich" gegen namensgleiche Karteileichen.
func besetzeKader(t *testing.T, db *sql.DB, teamID, seasonID int) {
	t.Helper()
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	memberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

// spiel baut eine Roh-Zeile, in der die eigene Mannschaft als Gast steht.
func spiel(staffel, alias string) h4aimport.RawGame {
	return h4aimport.RawGame{Staffel: staffel, Home: "Fremdverein", Guest: alias}
}

// Kernregel: „Team Stuttgart 2" ist die Mannschaft in der NIEDRIGEREN Staffel.
// Die Nummer im Vereinsnamen ordnet nicht — die Spielklasse tut es.
func TestSuggestStaffelTeams_ZweiteMannschaftIstNiedrigereStaffel(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	erste := insertTeam(t, db, "C-Jugend männlich", "C-Jugend", "m")
	zweite := insertTeam(t, db, "C-Jugend männlich 2", "C-Jugend", "m")
	besetzeKader(t, db, erste, seasonID)
	besetzeKader(t, db, zweite, seasonID)

	got, _ := h.suggestStaffelTeams(context.Background(), []h4aimport.RawGame{
		spiel("mC-OL-3-BW", "Team Stuttgart"),   // Oberliga = höher
		spiel("mC-BOL-SRM", "Team Stuttgart 2"), // Bezirksoberliga = niedriger
		spiel("mC-OL-3-BW", "Team Stuttgart"),   // Dublette ändert nichts
	})

	if c, ok := got[staffelKey{"mC-OL-3-BW", "Team Stuttgart"}]; !ok || c.id != erste {
		t.Errorf("höhere Staffel → %+v, erwartet Mannschaft %d", c, erste)
	}
	if c, ok := got[staffelKey{"mC-BOL-SRM", "Team Stuttgart 2"}]; !ok || c.id != zweite {
		t.Errorf("niedrigere Staffel → %+v, erwartet Mannschaft %d", c, zweite)
	}
}

// Die Zuordnung folgt der Liga, NICHT der Nummer im Vereinsnamen: steht "Team
// Stuttgart 2" ausnahmsweise in der höheren Staffel, bekommt es die erste Mannschaft.
func TestSuggestStaffelTeams_LigaSchlaegtVereinsnummer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	erste := insertTeam(t, db, "B-Jugend männlich", "B-Jugend", "m")
	zweite := insertTeam(t, db, "B-Jugend männlich 2", "B-Jugend", "m")
	besetzeKader(t, db, erste, seasonID)
	besetzeKader(t, db, zweite, seasonID)

	got, _ := h.suggestStaffelTeams(context.Background(), []h4aimport.RawGame{
		spiel("mB-OL-3-BW", "Team Stuttgart 2"), // Oberliga
		spiel("mB-BZL-SRM", "Team Stuttgart"),   // Bezirksliga
	})

	if c := got[staffelKey{"mB-OL-3-BW", "Team Stuttgart 2"}]; c.id != erste {
		t.Errorf("Oberliga → Mannschaft %d, erwartet %d (erste)", c.id, erste)
	}
	if c := got[staffelKey{"mB-BZL-SRM", "Team Stuttgart"}]; c.id != zweite {
		t.Errorf("Bezirksliga → Mannschaft %d, erwartet %d (zweite)", c.id, zweite)
	}
}

// Der A-Jugend-Fall aus dem Bestand: nur EINE Staffel, nur EINE spielende Mannschaft.
// Dann ist die Zuordnung eindeutig, auch wenn H4A "Team Stuttgart 2" meldet und die
// TeamWERK-Mannschaft schlicht "A-Jugend männlich" heißt.
func TestSuggestStaffelTeams_EinzelneStaffelIgnoriertVereinsnummer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	aktiv := insertTeam(t, db, "A-Jugend männlich", "A-Jugend", "m")
	besetzeKader(t, db, aktiv, seasonID)
	// Karteileichen mit passender Nummer, aber ohne Kader — dürfen nicht gewinnen.
	insertTeam(t, db, "A-Jugend männlich 2", "A-Jugend", "m")
	insertTeam(t, db, "A-Jugend männlich 2", "A-Jugend", "m")

	got, _ := h.suggestStaffelTeams(context.Background(), []h4aimport.RawGame{
		spiel("mA-BOL-SRM", "Team Stuttgart 2"),
	})

	c, ok := got[staffelKey{"mA-BOL-SRM", "Team Stuttgart 2"}]
	if !ok {
		t.Fatal("erwartet Vorschlag für einzige Staffel + einzige aktive Mannschaft")
	}
	if c.id != aktiv {
		t.Errorf("Vorschlag = %d (%s), erwartet %d", c.id, c.name, aktiv)
	}
}

// Eine frisch angelegte zweite Mannschaft hat noch keine Spieler im Kader — sie
// spielt trotzdem. Kaderbesetzung ist deshalb ausdrücklich KEINE Voraussetzung für
// einen Vorschlag, sondern nur Tiebreak zwischen Namensdubletten.
func TestSuggestStaffelTeams_UnbesetzterKaderVerhindertVorschlagNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	erste := insertTeam(t, db, "A-Jugend männlich", "A-Jugend", "m")
	zweite := insertTeam(t, db, "A-Jugend männlich 2", "A-Jugend", "m")
	besetzeKader(t, db, erste, seasonID) // nur die erste hat Spieler

	got, reasons := h.suggestStaffelTeams(context.Background(), []h4aimport.RawGame{
		spiel("mA-OL-3-BW", "Team Stuttgart"),   // Oberliga = höher
		spiel("mA-BOL-SRM", "Team Stuttgart 2"), // Bezirksoberliga = niedriger
	})

	if c := got[staffelKey{"mA-OL-3-BW", "Team Stuttgart"}]; c.id != erste {
		t.Errorf("höhere Staffel → %d, erwartet %d (%v)", c.id, erste, reasons)
	}
	if c := got[staffelKey{"mA-BOL-SRM", "Team Stuttgart 2"}]; c.id != zweite {
		t.Errorf("niedrigere Staffel → %d, erwartet %d trotz leerem Kader (%v)", c.id, zweite, reasons)
	}
}

// Fehlt für eine Spielklasse die passende Mannschaft, bleibt nur DIESE Staffel offen —
// die übrigen werden trotzdem zugeordnet. Alles-oder-nichts wäre unnötige Handarbeit.
func TestSuggestStaffelTeams_TeilzuordnungBeiFehlenderMannschaft(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	erste := insertTeam(t, db, "C-Jugend männlich", "C-Jugend", "m")
	besetzeKader(t, db, erste, seasonID)

	got, reasons := h.suggestStaffelTeams(context.Background(), []h4aimport.RawGame{
		spiel("mC-OL-3-BW", "Team Stuttgart"),   // höhere Liga → 1. Mannschaft
		spiel("mC-BOL-SRM", "Team Stuttgart 2"), // niedrigere Liga → 2. Mannschaft fehlt
	})

	if c := got[staffelKey{"mC-OL-3-BW", "Team Stuttgart"}]; c.id != erste {
		t.Errorf("höhere Staffel → %d, erwartet %d", c.id, erste)
	}
	offen := staffelKey{"mC-BOL-SRM", "Team Stuttgart 2"}
	if _, ok := got[offen]; ok {
		t.Error("erwartet keinen Vorschlag für die zweite Spielklasse")
	}
	if reasons[offen] == "" {
		t.Error("offene Staffel muss einen Grund tragen")
	}
}

// Kein Vorschlag ist besser als ein falscher: der Vorschlag wird beim apply als
// Mapping GELERNT und wirkt auf alle Folgeimporte.
func TestSuggestStaffelTeams_KeinVorschlagStattRaten(t *testing.T) {
	tests := []struct {
		name   string
		teams  [][3]string // name, ageClass, gender
		besetz []int       // Indizes in teams, die einen Kader bekommen
		games  []h4aimport.RawGame
	}{
		{
			name:   "unbekanntes Liga-Kürzel bei mehreren Staffeln",
			teams:  [][3]string{{"C-Jugend männlich", "C-Jugend", "m"}, {"C-Jugend männlich 2", "C-Jugend", "m"}},
			besetz: []int{0, 1},
			games: []h4aimport.RawGame{
				spiel("mC-OL-3-BW", "Team Stuttgart"),
				spiel("mC-XYZ-SRM", "Team Stuttgart 2"),
			},
		},
		{
			name:   "zwei Staffeln derselben Liga — keine Rangfolge",
			teams:  [][3]string{{"C-Jugend männlich", "C-Jugend", "m"}, {"C-Jugend männlich 2", "C-Jugend", "m"}},
			besetz: []int{0, 1},
			games: []h4aimport.RawGame{
				spiel("mC-OL-1-BW", "Team Stuttgart"),
				spiel("mC-OL-3-BW", "Team Stuttgart 2"),
			},
		},
		{
			name:   "Staffelcode ohne erkennbares Muster",
			teams:  [][3]string{{"C-Jugend männlich", "C-Jugend", "m"}},
			besetz: []int{0},
			games:  []h4aimport.RawGame{spiel("Pokalrunde", "Team Stuttgart")},
		},
		{
			name:   "keine Mannschaft der Altersklasse angelegt",
			teams:  nil,
			besetz: nil,
			games:  []h4aimport.RawGame{spiel("mC-OL-3-BW", "Team Stuttgart")},
		},
		{
			// Namensdubletten ohne jedes Unterscheidungsmerkmal (kein Kader, keine
			// Spiele) — hier ist nichts zu entscheiden.
			name: "ununterscheidbare Namensdubletten",
			teams: [][3]string{
				{"C-Jugend männlich", "C-Jugend", "m"},
				{"C-Jugend männlich", "C-Jugend", "m"},
			},
			besetz: nil,
			games:  []h4aimport.RawGame{spiel("mC-OL-3-BW", "Team Stuttgart")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			seasonID := testutil.CreateSeason(t, db, "2026/27")
			h := &Handler{db: db}

			ids := make([]int, len(tt.teams))
			for i, spec := range tt.teams {
				ids[i] = insertTeam(t, db, spec[0], spec[1], spec[2])
			}
			for _, i := range tt.besetz {
				besetzeKader(t, db, ids[i], seasonID)
			}

			got, reasons := h.suggestStaffelTeams(context.Background(), tt.games)
			if len(got) != 0 {
				t.Errorf("erwartet keinen Vorschlag, bekam %+v", got)
			}
			// Jede Ablehnung muss sich erklären — sonst steht der Importeur im
			// Modal vor „nicht zugeordnet" ohne zu wissen, was zu tun ist.
			for _, g := range tt.games {
				alias, _, _ := ownAlias(g)
				if reasons[staffelKey{g.Staffel, alias}] == "" {
					t.Errorf("kein Ablehnungsgrund für Staffel %q/%q", g.Staffel, alias)
				}
			}
		})
	}
}

// Das gelernte Mapping hat Vorrang vor dem Vorschlag — sonst würde eine bewusst
// abweichende Zuordnung bei jedem Folgeimport wieder überschrieben.
func TestBuildH4APlan_GelerntSchlaegtVorschlag(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	h := &Handler{db: db}

	abgeleitet := insertTeam(t, db, "C-Jugend männlich", "C-Jugend", "m")
	gelernt := insertTeam(t, db, "Förderkader gemischt", "Förderkader", "mixed")
	besetzeKader(t, db, abgeleitet, seasonID)

	raw := []h4aimport.RawGame{{
		Staffel: "mC-OL-3-BW", GameNo: "905996", Date: "2026-09-26", Time: "14:45",
		Home: "Fremdverein", Guest: "Team Stuttgart",
	}}

	plan, err := h.buildH4APlan(context.Background(), raw)
	if err != nil {
		t.Fatalf("buildH4APlan: %v", err)
	}
	if len(plan.New) != 1 {
		t.Fatalf("erwartet 1 neue Zeile, bekam %d", len(plan.New))
	}
	if plan.New[0].TeamSource != "vorschlag" || *plan.New[0].TeamID != abgeleitet {
		t.Errorf("ohne Lerneintrag: source=%q team=%v, erwartet vorschlag/%d",
			plan.New[0].TeamSource, plan.New[0].TeamID, abgeleitet)
	}

	if _, err := db.Exec(
		`INSERT INTO h4a_staffel_team_map (staffel, club_alias, team_id) VALUES (?,?,?)`,
		"mC-OL-3-BW", "Team Stuttgart", gelernt); err != nil {
		t.Fatalf("Lerneintrag: %v", err)
	}

	plan, err = h.buildH4APlan(context.Background(), raw)
	if err != nil {
		t.Fatalf("buildH4APlan (2): %v", err)
	}
	if plan.New[0].TeamSource != "gelernt" || *plan.New[0].TeamID != gelernt {
		t.Errorf("mit Lerneintrag: source=%q team=%v, erwartet gelernt/%d",
			plan.New[0].TeamSource, plan.New[0].TeamID, gelernt)
	}
}
