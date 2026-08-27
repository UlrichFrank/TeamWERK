package games

// Tests für die tagesweite Bewirtungs-/Kuchendienst-Rotation (kuchendienst-rotation +
// bewirtung-cap-global + bewirtung-kuchen-statt-slots). Internes Test-Package wie
// regen_bulk_context_test.go, weil hier runAutoRegen/regenSingleDay direkt getrieben
// werden; die Helfer (insertDutyTypeI, insertTemplateI, insertGameI, regenDate,
// seedAgeClassRuleI, …) stammen aus regen_bulk_context_test.go / regen_bulk_restore_test.go.
//
// Zugeteilt werden KUCHEN, nicht Slots: der Tagesbedarf wird auf möglichst wenige
// Mannschaften gebündelt, jede bekommt EINEN Slot an ihrem eigenen Termin, dessen
// slots_total ihre Kuchenzahl ist.

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
//
// slotsCount ist für rotations-aktive Items wirkungslos (bewirtung-kuchen-statt-slots) —
// die Personenzahl kommt aus der Zuteilung. Der Parameter bleibt, damit
// TestRotation_SlotsCountOhneWirkung genau das prüfen kann.
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

// rotSlot ist ein erwarteter Rotations-Slot: Mannschaft und Kuchenzahl (slots_total).
type rotSlot struct {
	team  int
	cakes int
}

// rotationSlotsAt liest die Rotations-Slots eines Spiels. Die Mannschaft steht seit
// dienst-slot-team-id-ausbauen NICHT mehr in duty_slots.team_id, sondern ergibt sich aus
// dem Anker-Spiel, an dem der Slot hängt (game_teams) — genau das ist die Zusage, die
// diese Tests prüfen sollen: „die Kuchen von Team A hängen an Team As Termin". Der Helfer
// löst deshalb über game_teams auf; die Erwartungen der Tests bleiben unverändert.
// Ein Spiel mit mehreren Teams liefert deren kleinste ID — solche Termine kommen in
// diesen Tests nicht vor (TestRotation_ZweiTeamsAmSelbenSpiel prüft den Fall gesondert).
func rotationSlotsAt(t *testing.T, db *sql.DB, gameID, dutyTypeID int) []rotSlot {
	t.Helper()
	rows, err := db.Query(
		`SELECT COALESCE((SELECT MIN(gt.team_id) FROM game_teams gt WHERE gt.game_id = ds.game_id), 0),
		        ds.slots_total
		 FROM duty_slots ds
		 WHERE ds.game_id=? AND ds.duty_type_id=? AND ds.is_custom=0
		 ORDER BY ds.id`, gameID, dutyTypeID)
	if err != nil {
		t.Fatalf("rotationSlotsAt: %v", err)
	}
	defer rows.Close()
	var out []rotSlot
	for rows.Next() {
		var team, total int
		if err := rows.Scan(&team, &total); err != nil {
			t.Fatalf("rotationSlotsAt scan: %v", err)
		}
		out = append(out, rotSlot{team: team, cakes: total})
	}
	return out
}

// assertRotation vergleicht die Rotations-Slots einer chronologisch geordneten
// Spieleliste. want[i] == nil bedeutet „an diesem Spiel entsteht kein Slot".
func assertRotation(t *testing.T, db *sql.DB, dutyTypeID int, gameIDs []int, want [][]rotSlot) {
	t.Helper()
	if len(gameIDs) != len(want) {
		t.Fatalf("assertRotation: %d Spiele, aber %d Erwartungen", len(gameIDs), len(want))
	}
	for i, gid := range gameIDs {
		got := rotationSlotsAt(t, db, gid, dutyTypeID)
		if len(got) != len(want[i]) {
			t.Errorf("game %d (Position %d): erwartet %d Slots %v, bekam %d Slots %v",
				gid, i+1, len(want[i]), want[i], len(got), got)
			continue
		}
		for j := range got {
			if got[j] != want[i][j] {
				t.Errorf("game %d (Position %d), Slot %d: erwartet %+v, bekam %+v",
					gid, i+1, j+1, want[i][j], got[j])
			}
		}
	}
}

// assertKeinAuffangSlot sichert die Kernaussage von Decision 4: der Restbedarf verfällt,
// er wird NICHT zu einem zusätzlichen Auffang-Slot. Früher war „ohne Team" das Merkmal
// eines solchen Slots; seit spielgebundene Slots grundsätzlich kein Team tragen, ist es
// die Anzahl — ein Auffang-Slot wäre ein Slot mehr, als Mannschaften herangezogen wurden.
func assertKeinAuffangSlot(t *testing.T, db *sql.DB, dutyTypeID, erwarteteSlots int) {
	t.Helper()
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND is_custom=0", dutyTypeID); got != erwarteteSlots {
		t.Errorf("erwartet genau %d Rotations-Slots (kein Auffang-Slot für den Restbedarf), bekam %d",
			erwarteteSlots, got)
	}
}

// TestRotation_FuenfSpieleVierTeamsCapZwei (spec: „Fünf Spiele, vier Teams, Cap zwei"):
// Bedarf 5 Kuchen, Warteschlange [A,B,C,D], Cap 2 → A 2, B 2, C 1, D nichts. Drei Slots,
// nicht fünf — und jeder am eigenen Termin seiner Mannschaft.
func TestRotation_FuenfSpieleVierTeamsCapZwei(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	teamC := testutil.CreateTeam(t, db, "Team C")
	teamD := testutil.CreateTeam(t, db, "Team D")
	for _, tid := range []int{teamA, teamB, teamC, teamD} {
		seedAgeClassRuleI(t, db, tid)
	}
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamC, "2026-06-13", "11:00", tpl),
		insertGameI(t, db, seasonID, teamD, "2026-06-13", "12:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "13:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertRotation(t, db, kuchen, games, [][]rotSlot{
		{{teamA, 2}}, {{teamB, 2}}, {{teamC, 1}}, nil, nil,
	})
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND is_custom=0", kuchen); got != 3 {
		t.Errorf("erwartet genau 3 Rotations-Slots (nicht einen pro Spiel), bekam %d", got)
	}
	if len(summary.Unassigned) != 0 {
		t.Errorf("erwartet keine unassigned-Einträge, bekam %+v", summary.Unassigned)
	}
}

// TestRotation_Spieltag27September ist der Regressionstest aus dem gemeldeten Fehlerbild:
// fünf Heimspiele (gD, wB, mA2, mA1, mA1 — die letzten beiden dieselbe Mannschaft),
// Verhältnis 1, Cap 2. Erwartet: gD 2 Kuchen, wB 2, mA2 1 — mA1 nichts.
func TestRotation_Spieltag27September(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	gD := testutil.CreateTeam(t, db, "gD")
	wB := testutil.CreateTeam(t, db, "wB")
	mA2 := testutil.CreateTeam(t, db, "mA2")
	mA1 := testutil.CreateTeam(t, db, "mA1")
	for _, tid := range []int{gD, wB, mA2, mA1} {
		seedAgeClassRuleI(t, db, tid)
	}
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 2, 2)

	games := []int{
		insertGameI(t, db, seasonID, gD, "2026-09-27", "10:00", tpl),
		insertGameI(t, db, seasonID, wB, "2026-09-27", "11:30", tpl),
		insertGameI(t, db, seasonID, mA2, "2026-09-27", "13:15", tpl),
		insertGameI(t, db, seasonID, mA1, "2026-09-27", "15:15", tpl),
		insertGameI(t, db, seasonID, mA1, "2026-09-27", "15:15", tpl),
	}

	regenDate(t, h, db, "2026-09-27", seasonID, nil)

	assertRotation(t, db, kuchen, games, [][]rotSlot{
		{{gD, 2}}, {{wB, 2}}, {{mA2, 1}}, nil, nil,
	})
}

// TestRotation_TeamMitZweiSpielenNurEinmalInWarteschlange (spec: „Mannschaft mit zwei
// Heimspielen bekommt einen Slot am früheren"): A um 9:00 und 11:00, B um 10:00 →
// Warteschlange [A,B], Bedarf 3, Cap 2 → A 2 Kuchen um 9:00, B 1 Kuchen um 10:00.
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

	assertRotation(t, db, kuchen, games, [][]rotSlot{
		{{teamA, 2}}, {{teamB, 1}}, nil,
	})
}

// TestRotation_SlotHaengtAmEigenenTermin (spec: „Slot hängt am eigenen Termin der
// Mannschaft"): game_id UND event_time werden gegen das Anker-Spiel gerechnet, nicht
// gegen das i-te Spiel des Tages. Offset −30 min macht die Herkunft der Zeit eindeutig.
func TestRotation_SlotHaengtAmEigenenTermin(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, -30, 1, 1)

	gameA := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)
	gameB := insertGameI(t, db, seasonID, teamB, "2026-06-13", "11:30", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	for _, tc := range []struct {
		gameID, teamID int
		wantTime       string
	}{
		{gameA, teamA, "09:30"},
		{gameB, teamB, "11:00"},
	} {
		var eventTime string
		if err := db.QueryRow(
			`SELECT event_time FROM duty_slots WHERE game_id=? AND duty_type_id=? AND is_custom=0`,
			tc.gameID, kuchen).Scan(&eventTime); err != nil {
			t.Fatalf("Slot von Team %d am Spiel %d: %v", tc.teamID, tc.gameID, err)
		}
		if eventTime != tc.wantTime {
			t.Errorf("Spiel %d: erwartet event_time %s (aus dem eigenen Anpfiff), bekam %s",
				tc.gameID, tc.wantTime, eventTime)
		}
		// Welche Mannschaft den Slot trägt, sagt seit dienst-slot-team-id-ausbauen
		// nicht mehr die Spalte, sondern der Termin, an dem er hängt.
		if got := rotationSlotsAt(t, db, tc.gameID, kuchen); len(got) != 1 || got[0].team != tc.teamID {
			t.Errorf("Spiel %d: erwartet genau einen Slot am Termin von Team %d, bekam %+v",
				tc.gameID, tc.teamID, got)
		}
	}
}

// TestRotation_SlotsCountOhneWirkung (spec: „slots_count der Vorlage bleibt ohne
// Wirkung"): das Item trägt slots_count=3, der Cap ist 2 und die Zuteilung ergibt zwei
// Kuchen → slots_total=2.
func TestRotation_SlotsCountOhneWirkung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 3, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Bedarf 2, Cap 2 → alles an A. slots_count=3 wäre 3 (oder 6) — beides falsch.
	assertRotation(t, db, kuchen, games, [][]rotSlot{{{teamA, 2}}, nil})
}

// TestRotation_VerhaeltnisKleinerEins (spec: „Verhältnis kleiner eins reduziert den
// Bedarf"): 4 Heimspiele, Verhältnis 0.5, Cap 2 → Bedarf 2, ein Slot für A mit 2 Kuchen.
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
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "12:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertRotation(t, db, kuchen, games, [][]rotSlot{{{teamA, 2}}, nil, nil, nil})
	// Kein Slot ist kein "Skipped" — das Item hat für diese Spiele schlicht nichts erzeugt.
	for _, s := range summary.Skipped {
		if s.DutyType == "Kuchen" {
			t.Errorf("erwartet keinen Skipped-Eintrag für Rotations-Misses, bekam %+v", summary.Skipped)
		}
	}
}

// TestRotation_VerhaeltnisGroesserEinsSchlaegtDurch (spec: „Verhältnis größer eins erhöht
// den Bedarf"): 3 Heimspiele, Verhältnis 2, Cap 2 → Bedarf 6, drei Slots à 2 Kuchen. Die
// frühere Deckelung auf die Spieleanzahl (kuchendienst-rotation) ist entfallen.
func TestRotation_VerhaeltnisGroesserEinsSchlaegtDurch(t *testing.T) {
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
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
		insertGameI(t, db, seasonID, teamC, "2026-06-13", "11:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertRotation(t, db, kuchen, games, [][]rotSlot{
		{{teamA, 2}}, {{teamB, 2}}, {{teamC, 2}},
	})
	if len(summary.Unassigned) != 0 {
		t.Errorf("Bedarf 6 = 3 Teams × Cap 2 ist voll gedeckt, bekam %+v", summary.Unassigned)
	}
}

// TestRotation_RestbedarfVerfaellt (spec: „Restbedarf verfällt, wenn die Warteschlange
// erschöpft ist"): 5 Heimspiele, nur zwei Mannschaften, Cap 2 → Bedarf 5, zugeteilt 4,
// ein Kuchen verfällt. Kein Auffang-Slot ohne Team, aber ein Eintrag in der Zusammenfassung.
func TestRotation_RestbedarfVerfaellt(t *testing.T) {
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

	assertRotation(t, db, kuchen, games, [][]rotSlot{
		{{teamA, 2}}, {{teamB, 2}}, nil, nil, nil,
	})
	assertKeinAuffangSlot(t, db, kuchen, 2)

	if len(summary.Unassigned) != 1 {
		t.Fatalf("erwartet genau einen unassigned-Eintrag, bekam %+v", summary.Unassigned)
	}
	u := summary.Unassigned[0]
	if u.Count != 1 || u.DutyType != "Kuchen" || u.Date != "2026-06-13" {
		t.Errorf("unerwarteter unassigned-Eintrag %+v (erwartet Count 1, Kuchen, 2026-06-13)", u)
	}

	for _, tid := range []int{teamA, teamB} {
		var total sql.NullInt64
		if err := db.QueryRow(
			`SELECT SUM(slots_total) FROM duty_slots WHERE duty_type_id=? AND team_id=?`,
			kuchen, tid).Scan(&total); err != nil {
			t.Fatalf("Kuchen je Team lesen: %v", err)
		}
		if total.Int64 > 2 {
			t.Errorf("Team %d bekam %d Kuchen, Cap ist 2", tid, total.Int64)
		}
	}
}

// TestRotation_WarteschlangeStartetProSpieltagNeu (spec: „Warteschlange startet bei jedem
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

	// Bedarf 2, Cap 2 → an BEIDEN Tagen deckt Team A (Position 1) den ganzen Bedarf.
	assertRotation(t, db, kuchen, day1, [][]rotSlot{{{teamA, 2}}, nil})
	assertRotation(t, db, kuchen, day2, [][]rotSlot{{{teamA, 2}}, nil})
}

// TestRotation_ZusageUeberlebtGleichbleibendeZuteilung (spec: „Zusage überlebt einen Regen
// bei gleichbleibender Zuteilung"): der Dreier-Match (duty_type_id, event_time, team_id)
// trifft, die Zusage wird zurückgeschrieben und niemand wird benachrichtigt.
func TestRotation_ZusageUeberlebtGleichbleibendeZuteilung(t *testing.T) {
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

	gameA := insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl)
	insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	assignI(t, db, autoSlotID(t, db, gameA), userID, "assigned", nil, nil)

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameA)
	if _, _, _, ok := assignmentByUser(t, db, newSlotID, userID); !ok {
		t.Errorf("erwartet, dass die Zusage den Regen übersteht")
	}
	var filled int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, newSlotID).Scan(&filled); err != nil {
		t.Fatalf("slots_filled lesen: %v", err)
	}
	if filled != 1 {
		t.Errorf("erwartet slots_filled=1 nach dem Restore, bekam %d", filled)
	}
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			t.Errorf("erwartet keine 'entfernt'-Meldung für die wiederhergestellte Zusage, bekam %v", summary.NotifiedUsers)
		}
	}
}

// TestRotation_GesunkeneKuchenzahlVerliertJuengsteZusage (spec: „Gesunkene Kuchenzahl
// verliert die jüngste Zusage"): der Slot bleibt (gleiche Zeit, gleiches Team), sein
// slots_total sinkt von 2 auf 1 → die ältere Zusage wird zurückgeschrieben, die jüngere
// läuft als „entfernt" in die Benachrichtigung.
func TestRotation_GesunkeneKuchenzahlVerliertJuengsteZusage(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	alt := testutil.CreateUser(t, db, "standard")
	neu := testutil.CreateUser(t, db, "standard")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)

	gameA := insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl)
	insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	slotID := autoSlotID(t, db, gameA)
	assignI(t, db, slotID, alt, "assigned", nil, nil)
	assignI(t, db, slotID, neu, "assigned", nil, nil)

	// Verhältnis 0.5 → Bedarf 1 statt 2: derselbe Slot, aber nur noch ein Kuchen.
	setVerhaeltnisI(t, db, "0.5")
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameA)
	if _, _, _, ok := assignmentByUser(t, db, newSlotID, alt); !ok {
		t.Errorf("erwartet, dass die ältere Zusage erhalten bleibt")
	}
	if _, _, _, ok := assignmentByUser(t, db, newSlotID, neu); ok {
		t.Errorf("erwartet, dass die jüngere Zusage über der Kapazität verloren geht")
	}
	notified := false
	for _, uid := range summary.NotifiedUsers {
		if uid == neu {
			notified = true
		}
		if uid == alt {
			t.Errorf("erwartet keine Meldung für die erhaltene Zusage, bekam %v", summary.NotifiedUsers)
		}
	}
	if !notified {
		t.Errorf("erwartet eine 'entfernt'-Meldung für Nutzer %d, bekam %v", neu, summary.NotifiedUsers)
	}
}

// TestRotation_MannschaftFaelltRausVerliertZusage (spec: „Nicht mehr herangezogene
// Mannschaft verliert ihre Zusage"): sinkt der Bedarf so weit, dass eine Mannschaft
// keinen Kuchen mehr bekommt, verschwindet ihr Slot und die Zusage wird gemeldet.
func TestRotation_MannschaftFaelltRausVerliertZusage(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 1)

	insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl)
	gameB := insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	assignI(t, db, autoSlotID(t, db, gameB), userID, "assigned", nil, nil)

	// Verhältnis 0.5 → Bedarf 1: nur noch Team A (Position 1) bekommt einen Kuchen.
	setVerhaeltnisI(t, db, "0.5")
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := rotationSlotsAt(t, db, gameB, kuchen); len(got) != 0 {
		t.Errorf("erwartet keinen Slot mehr am Spiel von Team B, bekam %+v", got)
	}
	notified := false
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			notified = true
		}
	}
	if !notified {
		t.Errorf("erwartet eine 'entfernt'-Meldung für Nutzer %d, bekam %v", userID, summary.NotifiedUsers)
	}
}

// TestRotation_ZweiTeamsAmSelbenSpielWerdenZusammengefasst (design.md Decision 4):
// Tragen zwei Mannschaften dasselbe Heimspiel, treten beide an derselben Ankerposition in
// die Warteschlange ein. Ihre Zuteilungen werden zu EINEM Slot mit summierten Kuchen
// verschmolzen — nicht zu je einem pro Mannschaft. Ohne team_id wären zwei Slots am selben
// Termin und derselben Uhrzeit im Restore ohnehin nicht mehr unterscheidbar; die
// Verschmelzung macht daraus eine bewusste Zusage statt eines stillen Datenverlusts.
//
// Der Preis steht im Change: die Pro-Mannschaft-Auskunft entfällt in genau diesem Fall.
// Er kommt im Bestand nicht vor (ein Heimspiel ist das Spiel EINER Mannschaft).
func TestRotation_ZweiTeamsAmSelbenSpielWerdenZusammengefasst(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	userA := testutil.CreateUser(t, db, "standard")

	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 1)

	game1 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl)
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, game1, teamB); err != nil {
		t.Fatalf("zweites Team ergänzen: %v", err)
	}
	game2 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Bedarf 2, Warteschlange [A,B] — beide über Spiel 1 eingetreten, Cap 1.
	// Ein Slot an Spiel 1 mit slots_total=2, keiner an Spiel 2.
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND is_custom=0", kuchen); got != 1 {
		t.Fatalf("erwartet genau einen zusammengefassten Slot, bekam %d", got)
	}
	var slotID, slotsTotal int
	if err := db.QueryRow(
		`SELECT id, slots_total FROM duty_slots WHERE game_id=? AND duty_type_id=? AND is_custom=0`,
		game1, kuchen).Scan(&slotID, &slotsTotal); err != nil {
		t.Fatalf("Slot an Spiel 1: %v", err)
	}
	if slotsTotal != 2 {
		t.Errorf("erwartet slots_total=2 (Kuchen von A und B summiert), bekam %d", slotsTotal)
	}
	if got := countRowsI(t, db, "duty_slots", "game_id=? AND duty_type_id=? AND is_custom=0", game2, kuchen); got != 0 {
		t.Errorf("an Spiel 2 soll kein Rotations-Slot entstehen, bekam %d", got)
	}

	// Zusage überlebt einen zweiten Lauf am unveränderten Slot.
	assignI(t, db, slotID, userA, "assigned", nil, nil)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var neuerSlot int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND duty_type_id=? AND is_custom=0`,
		game1, kuchen).Scan(&neuerSlot); err != nil {
		t.Fatalf("Slot nach zweitem Lauf: %v", err)
	}
	if _, _, _, ok := assignmentByUser(t, db, neuerSlot, userA); !ok {
		t.Errorf("Zusage von Nutzer %d hat den zweiten Regen-Lauf nicht überlebt", userA)
	}
}

// TestRegen_EinSlotProTerminNichtProTeam (dienst-slot-team-id-ausbauen): ein Vorlagen-Item
// erzeugt je Termin EINEN Slot, auch wenn mehrere Mannschaften daran hängen — die
// Team-Allowlist entscheidet, OB das Item gilt, nicht für wie viele Mannschaften.
//
// Ersetzt TestRotation_NichtRotationsItemBehaeltTeamImMatchKey: dessen Zusage („team_id
// bleibt Teil des Match-Keys, ein Mannschaftswechsel kostet die Zusage") ist mit dem
// Wegfall von team_id aus dem Schlüssel bewusst entfallen. Die neue Zusage ist die
// freundlichere: der Slot überlebt einen Kaderwechsel am Termin, samt Zusage.
func TestRegen_EinSlotProTerminNichtProTeam(t *testing.T) {
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

	// Zweites Team am selben Spiel → trotzdem nur EIN Slot.
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("add second team: %v", err)
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", gameID); got != 1 {
		t.Fatalf("erwartet genau einen Slot für den Termin (nicht einen pro Team), bekam %d", got)
	}

	var slotID int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&slotID); err != nil {
		t.Fatalf("find slot: %v", err)
	}
	if got := countRowsI(t, db, "duty_slots", "id=? AND team_id IS NULL", slotID); got != 1 {
		t.Errorf("spielgebundener Slot soll team_id=NULL tragen")
	}
	assignI(t, db, slotID, userID, "assigned", nil, nil)

	// Team A verlässt das Spiel. Der Slot gehört dem Termin, nicht der Mannschaft —
	// er bleibt bestehen und die Zusage mit ihm.
	if _, err := db.Exec(`DELETE FROM game_teams WHERE game_id=? AND team_id=?`, gameID, teamA); err != nil {
		t.Fatalf("remove team A: %v", err)
	}
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var neuerSlot int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&neuerSlot); err != nil {
		t.Fatalf("Slot nach Kaderwechsel: %v", err)
	}
	if _, _, _, found := assignmentByUser(t, db, neuerSlot, userID); !found {
		t.Errorf("Zusage soll den Mannschaftswechsel am Termin überleben")
	}
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			t.Errorf("erwartet keine 'entfernt'-Meldung für die erhaltene Zusage, bekam %v", summary.NotifiedUsers)
		}
	}
}

// TestRotation_CapGiltVorlagenuebergreifend (spec: „Ein geänderter Cap wirkt sofort für
// alle Vorlagen", bewirtung-cap-global): zwei Heim-Vorlagen tragen rotations-aktive Items
// desselben Duty-Types. Vor dem Umzug in die Einstellungen hätte der Cap der ERSTEN
// Vorlage für beide gegolten („erster gewinnt"); jetzt gibt es nur noch einen Wert.
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

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Ein einziger Cap 2 über beide Vorlagen hinweg → Bedarf 4, A 2 und B 2.
	assertRotation(t, db, kuchen, []int{game1, game2, game3, game4}, [][]rotSlot{
		{{teamA, 2}}, {{teamB, 2}}, nil, nil,
	})
	if len(summary.Unassigned) != 0 {
		t.Errorf("Bedarf 4 = 2 Teams × Cap 2 ist voll gedeckt, bekam %+v", summary.Unassigned)
	}
}
