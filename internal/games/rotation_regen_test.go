package games

// Tests für die tagesweite Bewirtungs-/Kuchendienst-Rotation (kuchendienst-rotation,
// design.md Decision 1–5, 7). Internes Test-Package wie regen_bulk_context_test.go,
// weil hier runAutoRegen/regenSingleDay direkt getrieben werden; die Helfer
// (insertDutyTypeI, insertTemplateI, insertGameI, regenDate, seedAgeClassRuleI, …)
// stammen aus regen_bulk_context_test.go / regen_bulk_restore_test.go.

import (
	"database/sql"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// insertRotationTemplateI legt eine Heim-Vorlage mit genau einem rotations-aktivierten
// Item an und setzt den vereinsweiten Cap.
//
// Der Cap ist seit bewirtung-cap-global KEINE Item-Eigenschaft mehr (system_settings,
// Key bewirtung_max_per_team) — er bleibt hier trotzdem Parameter, weil jeder Testfall
// ihn zusammen mit seiner Vorlage aufsetzt. Zwei Aufrufe mit UNTERSCHIEDLICHEN Caps im
// selben Test wären widersprüchlich; setMaxPerTeamI schlägt in dem Fall fehl, statt den
// vorherigen Wert still zu überschreiben.
func insertRotationTemplateI(t *testing.T, db *sql.DB, dutyTypeID, offsetMin, slotsCount, maxPerTeam int) int {
	t.Helper()
	templateID := insertTemplateI(t, db, dutyTypeID, offsetMin, slotsCount)
	if _, err := db.Exec(
		`UPDATE game_template_items SET rotation_enabled=1 WHERE template_id=?`,
		templateID); err != nil {
		t.Fatalf("set rotation_enabled: %v", err)
	}
	setMaxPerTeamI(t, db, maxPerTeam)
	return templateID
}

// setMaxPerTeamI überschreibt den vereinsweiten Cap „Max. Kuchen pro Mannschaft".
// Schlägt fehl, wenn die von Migration 046 angelegte Row fehlt — oder wenn bereits ein
// abweichender Wert gesetzt wurde (siehe insertRotationTemplateI).
func setMaxPerTeamI(t *testing.T, db *sql.DB, value int) {
	t.Helper()
	var current string
	if err := db.QueryRow(
		`SELECT value FROM system_settings WHERE key='bewirtung_max_per_team'`).Scan(&current); err != nil {
		t.Fatalf("read bewirtung_max_per_team: %v", err)
	}
	want := strconv.Itoa(value)
	if current != "1" && current != want {
		t.Fatalf("bewirtung_max_per_team ist bereits %q — zwei abweichende Caps in einem Test", current)
	}
	if _, err := db.Exec(
		`UPDATE system_settings SET value=? WHERE key='bewirtung_max_per_team'`, want); err != nil {
		t.Fatalf("set bewirtung_max_per_team: %v", err)
	}
}

// setVerhaeltnisI überschreibt das vereinsweite Spiele-zu-Kuchen-Verhältnis. Schlägt
// fehl, wenn die von Migration 045 angelegte Default-Row fehlt.
func setVerhaeltnisI(t *testing.T, db *sql.DB, value string) {
	t.Helper()
	res, err := db.Exec(
		`UPDATE system_settings SET value=? WHERE key='bewirtung_verhaeltnis'`, value)
	if err != nil {
		t.Fatalf("set bewirtung_verhaeltnis: %v", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("expected exactly one bewirtung_verhaeltnis row, updated %d", n)
	}
}

// teamOfRotationSlot liefert die team_id des (einzigen) Auto-Slots dieses Duty-Types
// am Spiel. found=false heißt: für dieses Spiel entstand gar kein Slot.
func teamOfRotationSlot(t *testing.T, db *sql.DB, gameID, dutyTypeID int) (team sql.NullInt64, found bool) {
	t.Helper()
	rows, err := db.Query(
		`SELECT team_id FROM duty_slots WHERE game_id=? AND duty_type_id=? AND is_custom=0`,
		gameID, dutyTypeID)
	if err != nil {
		t.Fatalf("teamOfRotationSlot: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		if err := rows.Scan(&team); err != nil {
			t.Fatalf("teamOfRotationSlot scan: %v", err)
		}
		n++
	}
	if n > 1 {
		t.Fatalf("expected at most one rotation slot for game %d, got %d", gameID, n)
	}
	return team, n == 1
}

// assertTeams vergleicht die Team-Zuordnung der Rotations-Slots einer chronologisch
// geordneten Spieleliste. want[i] == 0 bedeutet "Slot ohne Team" (unassigned),
// want[i] == -1 bedeutet "gar kein Slot".
func assertTeams(t *testing.T, db *sql.DB, dutyTypeID int, gameIDs []int, want []int) {
	t.Helper()
	for i, gid := range gameIDs {
		team, found := teamOfRotationSlot(t, db, gid, dutyTypeID)
		switch want[i] {
		case -1:
			if found {
				t.Errorf("game %d (position %d): expected no rotation slot, got one (team valid=%v)", gid, i+1, team.Valid)
			}
		case 0:
			if !found {
				t.Errorf("game %d (position %d): expected an unassigned rotation slot, got none", gid, i+1)
			} else if team.Valid {
				t.Errorf("game %d (position %d): expected team_id NULL, got %d", gid, i+1, team.Int64)
			}
		default:
			if !found {
				t.Errorf("game %d (position %d): expected a rotation slot for team %d, got none", gid, i+1, want[i])
			} else if !team.Valid || int(team.Int64) != want[i] {
				t.Errorf("game %d (position %d): expected team %d, got valid=%v value=%d", gid, i+1, want[i], team.Valid, team.Int64)
			}
		}
	}
}

// TestRotation_FuenfSpieleDreiTeamsCapZwei (spec: "Fünf Spiele, drei Teams, Cap zwei"):
// Warteschlange [A,B,C], Verhältnis 1, Cap 2 → Zuordnung A,A,B,B,C.
func TestRotation_FuenfSpieleDreiTeamsCapZwei(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamC := testutil.CreateTeam(t, db, "Team C")
	for _, tid := range []int{teamA, teamB, teamC} {
		seedAgeClassRuleI(t, db, tid)
	}
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	// Teams A,B,C,A,B → erste Auftreten ergeben die Warteschlange [A,B,C].
	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamC, "2026-06-13", "11:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "12:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "13:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{teamA, teamA, teamB, teamB, teamC})
	if len(summary.Unassigned) != 0 {
		t.Errorf("expected no unassigned entries, got %+v", summary.Unassigned)
	}
}

// TestRotation_TeamMitZweiSpielenNurEinmalInWarteschlange (spec: "Team mit mehreren
// Spielen erscheint einmal"): A um 9:00 und 11:00, B um 10:00 → Warteschlange [A,B].
// Mit Cap 2 bekommt das 10:00-Spiel (Team B spielt) den Kuchen von Team A.
func TestRotation_TeamMitZweiSpielenNurEinmalInWarteschlange(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tpl),
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Warteschlange [A,B]: A deckt die ersten zwei Slots (Cap 2), B den dritten.
	assertTeams(t, db, kuchen, games, []int{teamA, teamA, teamB})
}

// TestRotation_VerhaeltnisKleinerEins (spec: "Verhältnis kleiner eins reduziert den
// Bedarf"): 4 Heimspiele, Verhältnis 0.5 → nur die chronologisch ersten zwei Spiele
// bekommen einen Rotations-Slot.
func TestRotation_VerhaeltnisKleinerEins(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	setVerhaeltnisI(t, db, "0.5")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 1)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "12:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{teamA, teamB, -1, -1})
	// Kein Slot ist kein "Skipped" — das Item hat für diese Spiele schlicht nichts erzeugt.
	for _, s := range summary.Skipped {
		if s.DutyType == "Kuchen" {
			t.Errorf("expected no Skipped entry for rotation misses, got %+v", summary.Skipped)
		}
	}
}

// TestRotation_VerhaeltnisGroesserEinsWirdGedeckelt (spec: "Verhältnis größer eins wirkt
// nur bis zur Spieleanzahl"): 3 Heimspiele, Verhältnis 2 → höchstens 3 Slots, nicht 6.
func TestRotation_VerhaeltnisGroesserEinsWirdGedeckelt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamC := testutil.CreateTeam(t, db, "Team C")
	for _, tid := range []int{teamA, teamB, teamC} {
		seedAgeClassRuleI(t, db, tid)
	}
	h := newRegenTestHandler(t, db)
	setVerhaeltnisI(t, db, "2")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 1)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamC, "2026-06-13", "11:00", tpl),
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{teamA, teamB, teamC})
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND is_custom=0", kuchen); got != 3 {
		t.Errorf("expected exactly 3 rotation slots (one per game), got %d", got)
	}
}

// TestRotation_CapUeberlaufLaesstSlotOhneTeam (spec: "Cap-Überlauf lässt Slots
// unzugeordnet statt den Cap zu verletzen"): 5 Spiele, Cap 2, nur zwei Teams in der
// Warteschlange → A,A,B,B + ein Slot mit team_id=NULL, ausgewiesen in Unassigned.
func TestRotation_CapUeberlaufLaesstSlotOhneTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "12:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "13:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{teamA, teamA, teamB, teamB, 0})

	if len(summary.Unassigned) != 1 {
		t.Fatalf("expected exactly one unassigned entry, got %+v", summary.Unassigned)
	}
	u := summary.Unassigned[0]
	if u.GameID != games[4] || u.DutyType != "Kuchen" || u.Date != "2026-06-13" {
		t.Errorf("unexpected unassigned entry %+v (want game %d, Kuchen, 2026-06-13)", u, games[4])
	}

	for _, tid := range []int{teamA, teamB} {
		if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND team_id=?", kuchen, tid); got > 2 {
			t.Errorf("team %d got %d rotation slots, cap is 2", tid, got)
		}
	}
}

// TestRotation_WarteschlangeStartetProSpieltagNeu (spec: "Warteschlange startet bei jedem
// Spieltag neu"): zwei aufeinanderfolgende Tage mit identischer Konstellation beginnen
// beide bei Position 1 — es gibt keinen tagesübergreifenden Rotationszustand.
func TestRotation_WarteschlangeStartetProSpieltagNeu(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	day1 := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
	}
	day2 := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-14", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-14", "10:00", tpl),
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	regenDate(t, h, db, "2026-06-14", seasonID, nil)

	// Cap 2 → an BEIDEN Tagen deckt Team A (Position 1) beide Slots ab.
	assertTeams(t, db, kuchen, day1, []int{teamA, teamA})
	assertTeams(t, db, kuchen, day2, []int{teamA, teamA})
}

// TestRotation_ZusageUeberlebtVerschobeneTeamZuordnung (spec: "Zusage überlebt eine
// team-verschiebende Spielplanänderung", design.md Decision 5): ein neu eingefügtes,
// früheres Heimspiel verschiebt die Team-Zurechnung eines bestehenden Rotations-Slots
// (A → B) bei gleichbleibender event_time. Die Zusage bleibt erhalten und wird nicht
// als "entfernt" gemeldet.
func TestRotation_ZusageUeberlebtVerschobeneTeamZuordnung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	game10 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)
	insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:30", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	team, found := teamOfRotationSlot(t, db, game10, kuchen)
	if !found || !team.Valid || int(team.Int64) != teamA {
		t.Fatalf("precondition: expected 10:00 slot assigned to team A (%d), got valid=%v value=%d", teamA, team.Valid, team.Int64)
	}
	oldSlotID := autoSlotID(t, db, game10)
	assignI(t, db, oldSlotID, userID, "assigned", nil, nil)

	// Neues, früheres Heimspiel von Team B → Warteschlange [B,A]; B deckt (Cap 2) die
	// ersten beiden Slots ab, der 10:00-Slot wechselt damit von A auf B.
	insertGameI(t, db, seasonID, teamB, "2026-06-13", "09:00", tpl)
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	team, found = teamOfRotationSlot(t, db, game10, kuchen)
	if !found || !team.Valid || int(team.Int64) != teamB {
		t.Fatalf("expected 10:00 slot to shift to team B (%d), got valid=%v value=%d", teamB, team.Valid, team.Int64)
	}
	newSlotID := autoSlotID(t, db, game10)
	if _, _, _, ok := assignmentByUser(t, db, newSlotID, userID); !ok {
		t.Errorf("expected assignment to survive the team shift on the rotation slot")
	}
	var filled int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, newSlotID).Scan(&filled); err != nil {
		t.Fatalf("read slots_filled: %v", err)
	}
	if filled != 1 {
		t.Errorf("expected slots_filled=1 after restore, got %d", filled)
	}
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			t.Errorf("expected no 'removed' notification for the restored rotation assignment, got NotifiedUsers=%v", summary.NotifiedUsers)
		}
	}
}

// TestRotation_NichtRotationsItemBehaeltTeamImMatchKey (spec: "Nicht-Rotations-Item behält
// das bestehende Matching", Regressionstest): ohne rotation_enabled bleibt team_id
// Teil des Dreier-Keys — wechselt das Team des Spiels, ist die Zusage verloren und der
// Nutzer wird benachrichtigt. Zusätzlich entsteht weiterhin ein Slot pro Team des Spiels.
func TestRotation_NichtRotationsItemBehaeltTeamImMatchKey(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tpl := insertTemplateI(t, db, kasse, 0, 1) // rotation_enabled bleibt 0
	gameID := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)

	// Zweites Team am selben Spiel → weiterhin ein Slot pro Team.
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("add second team: %v", err)
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", gameID); got != 2 {
		t.Fatalf("expected one slot per team (2) for a non-rotation item, got %d", got)
	}

	var slotA int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND team_id=? AND is_custom=0`, gameID, teamA).Scan(&slotA); err != nil {
		t.Fatalf("find team A slot: %v", err)
	}
	assignI(t, db, slotA, userID, "assigned", nil, nil)

	// Team A verlässt das Spiel → der (duty_type_id, event_time, team_id)-Key trifft
	// nicht mehr, die Zusage geht verloren (unverändertes Bestandsverhalten).
	if _, err := db.Exec(`DELETE FROM game_teams WHERE game_id=? AND team_id=?`, gameID, teamA); err != nil {
		t.Fatalf("remove team A: %v", err)
	}
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var slotB int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND team_id=? AND is_custom=0`, gameID, teamB).Scan(&slotB); err != nil {
		t.Fatalf("find team B slot: %v", err)
	}
	if _, _, _, found := assignmentByUser(t, db, slotB, userID); found {
		t.Errorf("expected assignment NOT to move onto the other team's slot (team stays part of the match key)")
	}
	notified := false
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			notified = true
		}
	}
	if !notified {
		t.Errorf("expected user %d to be notified about the lost assignment, got NotifiedUsers=%v", userID, summary.NotifiedUsers)
	}
}

// TestRotation_EinSlotProSpielTrotzZweierTeams: ein rotations-aktiviertes Item erzeugt
// auch dann genau EINEN Slot pro Heimspiel, wenn am Spiel zwei Mannschaften hängen —
// beide treten unabhängig in die Warteschlange ein (design.md Risks).
func TestRotation_EinSlotProSpielTrotzZweierTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 1)

	game1 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl)
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, game1, teamB); err != nil {
		t.Fatalf("add second team: %v", err)
	}
	game2 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", game1); got != 1 {
		t.Errorf("expected exactly one rotation slot for the two-team game, got %d", got)
	}
	// Warteschlange [A,B] (beide über Spiel 1 eingetreten), Cap 1 → A, dann B.
	assertTeams(t, db, kuchen, []int{game1, game2}, []int{teamA, teamB})
}

// TestRotation_CapGiltVorlagenuebergreifend (spec: "Ein geänderter Cap wirkt sofort für
// alle Vorlagen", bewirtung-cap-global): zwei Heim-Vorlagen tragen rotations-aktive Items
// desselben Duty-Types. Vor dem Umzug in die Einstellungen hätte der Cap der ERSTEN
// Vorlage für beide gegolten ("erster gewinnt"); jetzt gibt es nur noch einen Wert.
func TestRotation_CapGiltVorlagenuebergreifend(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tplEins := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)
	tplZwei := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	// Vier Heimspiele, abwechselnd aus beiden Vorlagen, Warteschlange [A,B], Cap 2.
	game1 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tplEins)
	game2 := insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tplZwei)
	game3 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tplZwei)
	game4 := insertGameI(t, db, seasonID, teamB, "2026-06-13", "12:00", tplEins)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Ein einziger Cap 2 über beide Vorlagen hinweg → A, A, B, B.
	assertTeams(t, db, kuchen, []int{game1, game2, game3, game4}, []int{teamA, teamA, teamB, teamB})
}
