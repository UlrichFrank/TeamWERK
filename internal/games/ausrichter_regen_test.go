package games

// Tests für das Ausrichter-Gate im Auto-Regen (heimspieltag-ausrichter, design.md
// Decision 4). Internes Test-Package wie rotation_regen_test.go, weil hier
// runAutoRegen/regenSingleDay direkt getrieben werden; die Helfer (insertDutyTypeI,
// insertTemplateI, insertGameI, insertRotationTemplateI, regenDate, assignI,
// assertTeams, countRowsI, …) stammen aus regen_bulk_context_test.go /
// regen_bulk_restore_test.go / rotation_regen_test.go.
//
// Das Gate sitzt an ZWEI Stellen, und die Reihenfolge ist der eigentliche Inhalt
// dieses Changes:
//   - Gate #1 in buildRotationPlan, im selben Durchlauf, der die rotations-aktiven
//     Items sammelt → wirkt VOR Warteschlange und Bedarfsrechnung.
//   - Gate #2 in regenGameItems, am Anfang der Item-Schleife → gilt für alle Items.
//
// Der Beweis, dass Gate #1 wirklich vor der Bedarfsrechnung greift und nicht erst
// beim Einfügen, ist die Abwesenheit eines team_id=NULL-Slots: käme das Gate zu
// spät, hätte die Warteschlange Positionen für verworfene Slots verbraucht und der
// Überlauf würde als "unassigned" sichtbar.

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// defaultAusrichterI liefert die von Migration 048 geseedete Default-Zeile. Auf sie
// löst jeder Spieltag ohne expliziten Eintrag auf (design.md Decision 2).
func defaultAusrichterI(t *testing.T, db *sql.DB) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM ausrichter WHERE is_default=1`).Scan(&id); err != nil {
		t.Fatalf("defaultAusrichterI: %v", err)
	}
	return id
}

// insertAusrichterI legt einen zusätzlichen (nicht-Default) Ausrichter an.
func insertAusrichterI(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO ausrichter (name) VALUES (?)`, name)
	if err != nil {
		t.Fatalf("insertAusrichterI: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// bindItemsToAusrichterI bindet alle Items einer Vorlage an einen Ausrichter.
// Bewusst per direktem UPDATE statt über die CRUD-Route: hier wird das Gate im
// Regen geprüft, nicht die Route-Validierung (die steht in ausrichter_template_test.go).
// Nur so lässt sich außerdem die Sicherung für Nicht-Heim-Termine testen — die Route
// ließe eine gebundene Zeile auf einer Auswärts-Vorlage gar nicht erst zu.
func bindItemsToAusrichterI(t *testing.T, db *sql.DB, templateID, ausrichterID int) {
	t.Helper()
	res, err := db.Exec(
		`UPDATE game_template_items SET ausrichter_id=? WHERE template_id=?`,
		ausrichterID, templateID)
	if err != nil {
		t.Fatalf("bindItemsToAusrichterI: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		t.Fatalf("bindItemsToAusrichterI: Vorlage %d hat keine Items", templateID)
	}
}

// setDayAusrichterI setzt den expliziten Tageswert. Das Datum wird in der reinen
// "2006-01-02"-Form geschrieben — dieselbe Form, die ResolveAusrichterForDay
// vergleicht (SQLite-DATE-Gotcha).
func setDayAusrichterI(t *testing.T, db *sql.DB, date string, seasonID, ausrichterID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO spieltag_ausrichter (date, season_id, ausrichter_id) VALUES (?,?,?)
		 ON CONFLICT(date, season_id) DO UPDATE SET ausrichter_id=excluded.ausrichter_id`,
		date, seasonID, ausrichterID); err != nil {
		t.Fatalf("setDayAusrichterI: %v", err)
	}
}

// insertEventI ist die event_type-freie Variante von insertGameI (das immer 'heim'
// anlegt) — für die Auswärts-/Generisch-Fälle, auf die das Gate nicht wirken darf.
func insertEventI(t *testing.T, db *sql.DB, seasonID, teamID int, date, timeStr string, templateID int, eventType string) int {
	t.Helper()
	var tplArg any
	if templateID > 0 {
		tplArg = templateID
	}
	isHome := 0
	if eventType == "heim" {
		isHome = 1
	}
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, template_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		seasonID, "Test", date, timeStr, isHome, eventType, tplArg)
	if err != nil {
		t.Fatalf("insertEventI: %v", err)
	}
	gameID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamID); err != nil {
		t.Fatalf("insertEventI game_teams: %v", err)
	}
	return int(gameID)
}

// TestAusrichterGate_AusgegatetesRotationsItemErzeugtKeinenBedarf (spec:
// "Ausgegatetes Rotations-Item erzeugt keinen Bedarf"): das rotations-aktive Item ist
// an Ausrichter A gebunden, der Tag löst auf B (den Default) auf → Bedarf 0.
//
// Neben "kein Slot" wird ausdrücklich "auch kein team_id=NULL-Slot" geprüft — die
// Spec nennt beides getrennt, weil der Überlauf-Slot der sichtbare Rest einer
// Bedarfsrechnung wäre, die das Gate nicht kennt.
//
// Zu ehrlicher Einordnung: dass Gate #1 vor der Bedarfsrechnung sitzt, entscheidet
// sich NICHT an diesem Test — ist auf einem Tag jedes rotations-aktive Item gegatet,
// räumt Gate #2 ohnehin alles ab. Der diskriminierende Fall ist der teilweise
// gegatete Tag darunter (dort verschiebt eine zu spät gefilterte Warteschlange die
// Team-Zuordnung nachweisbar); dieser Test hält die Spec-Invariante fest.
//
// Der zweite Durchlauf (Bindung entfernt) ist die Gegenprobe: dieselbe Konstellation
// erzeugt sehr wohl Slots — der Test ist also nicht vakuum-grün.
func TestAusrichterGate_AusgegatetesRotationsItemErzeugtKeinenBedarf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	setVerhaeltnisI(t, db, "1")

	fremder := insertAusrichterI(t, db, "TV Ötlingen")
	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)
	bindItemsToAusrichterI(t, db, tpl, fremder)

	// Kein expliziter Tageswert → der Tag löst auf den Default auf, nicht auf `fremder`.
	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{-1, -1})
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=?", kuchen); got != 0 {
		t.Errorf("expected no rotation slots at all, got %d", got)
	}
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=? AND team_id IS NULL", kuchen); got != 0 {
		t.Errorf("expected no team_id=NULL slot — das Gate greift dann erst nach der Bedarfsrechnung, got %d", got)
	}
	if len(summary.Unassigned) != 0 {
		t.Errorf("expected no unassigned entries (Bedarf 0), got %+v", summary.Unassigned)
	}

	// Gegenprobe: ohne Bindung erzeugt exakt dieselbe Konstellation Slots.
	if _, err := db.Exec(`UPDATE game_template_items SET ausrichter_id=NULL WHERE template_id=?`, tpl); err != nil {
		t.Fatalf("unbind: %v", err)
	}
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	assertTeams(t, db, kuchen, games, []int{teamA, teamA})
}

// TestAusrichterGate_TeilweiseGegateterTagZaehltNurPassendeSpiele (spec: "Teilweise
// gegatete Vorlagen zählen nur die passenden Spiele"): vier Heimspiele, Verhältnis 1,
// aber nur zwei tragen eine Vorlage, deren rotations-aktives Item das Gate passiert →
// Bedarf 2, nicht 4.
//
// Das ist der Test, der die REIHENFOLGE der beiden Gates festnagelt. Filterte man
// erst in regenGameItems, rechnete buildRotationPlan über alle vier Spiele: Bedarf 4,
// Warteschlange [A, B] mit Cap 2 → die Positionen 1+2 verbrauchten die gegateten
// Spiele, und der Slot des 11:00-Spiels landete bei B statt bei A. Es entstünden
// zufällig ebenfalls zwei Slots — nur an der falschen Mannschaft. Genau darauf zielt
// die Team-Assertion unten, nicht nur auf die Anzahl.
func TestAusrichterGate_TeilweiseGegateterTagZaehltNurPassendeSpiele(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	setVerhaeltnisI(t, db, "1")

	fremder := insertAusrichterI(t, db, "TV Ötlingen")
	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tplPass := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)  // ungebunden → gilt immer
	tplGated := insertRotationTemplateI(t, db, kuchen, 0, 1, 2) // an `fremder` gebunden
	bindItemsToAusrichterI(t, db, tplGated, fremder)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tplPass),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tplGated),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "11:00", tplPass),
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "12:00", tplGated),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Warteschlange nur aus den passierenden Spielen: [A (09:00), B (11:00)], Cap 2
	// → beide Slots an A. Ohne Gate #1 wären es vier Slots (A, A, B, B).
	assertTeams(t, db, kuchen, games, []int{teamA, -1, teamA, -1})
	if got := countRowsI(t, db, "duty_slots", "duty_type_id=?", kuchen); got != 2 {
		t.Errorf("expected Bedarf 2 (nur die passierenden Heimspiele zählen), got %d Slots", got)
	}
	if len(summary.Unassigned) != 0 {
		t.Errorf("expected no unassigned entries, got %+v", summary.Unassigned)
	}
}

// TestAusrichterGate_ItemOhneBindungUnveraendert (spec: "Bestehende Vorlagen verhalten
// sich unverändert"): Charakterisierung. Ein Item mit ausrichter_id IS NULL erzeugt
// exakt dieselben Slots wie vor dem Change — unabhängig davon, welcher Ausrichter für
// den Tag gilt. Das ist die Zusicherung, dass der Change rein additiv ist und ein
// Deploy ohne gepflegte Bindungen verhaltensneutral bleibt.
func TestAusrichterGate_ItemOhneBindungUnveraendert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamA)
	h := newRegenTestHandler(t, db)

	fremder := insertAusrichterI(t, db, "TV Ötlingen")
	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tpl := insertTemplateI(t, db, kasse, -60, 2) // ungebunden, nicht rotierend

	// Tag 1 ohne expliziten Eintrag (löst auf den Default auf),
	// Tag 2 explizit auf einen anderen Ausrichter gesetzt.
	game1 := insertGameI(t, db, seasonID, teamA, "2026-06-13", "18:00", tpl)
	game2 := insertGameI(t, db, seasonID, teamA, "2026-06-20", "18:00", tpl)
	setDayAusrichterI(t, db, "2026-06-20", seasonID, fremder)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	regenDate(t, h, db, "2026-06-20", seasonID, nil)

	for _, gameID := range []int{game1, game2} {
		var dutyTypeID, slotsTotal int
		var eventTime string
		var teamID sql.NullInt64
		if err := db.QueryRow(
			`SELECT duty_type_id, event_time, slots_total, team_id
			 FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).
			Scan(&dutyTypeID, &eventTime, &slotsTotal, &teamID); err != nil {
			t.Fatalf("game %d: expected exactly one auto slot: %v", gameID, err)
		}
		if dutyTypeID != kasse || eventTime != "17:00" || slotsTotal != 2 {
			t.Errorf("game %d: unexpected slot (duty_type=%d event_time=%q slots_total=%d), want (%d, \"17:00\", 2)",
				gameID, dutyTypeID, eventTime, slotsTotal, kasse)
		}
		if !teamID.Valid || int(teamID.Int64) != teamA {
			t.Errorf("game %d: expected team %d, got valid=%v value=%d", gameID, teamA, teamID.Valid, teamID.Int64)
		}
	}
}

// TestAusrichterGate_AuswaertsUndGenerischUnberuehrt (spec: "Auswärts- und generische
// Termine bleiben unberührt"): das Gate wirkt ausschließlich auf event_type='heim'.
// Die Bindung wird hier direkt in die DB geschrieben — über die Route wäre sie auf
// einer Nicht-Heim-Vorlage gar nicht anlegbar (400 ausrichter_requires_heim_template);
// genau deshalb ist der event_type-Zweig im Gate eine Sicherung und keine Fachlogik.
//
// Das gleichzeitig angelegte Heimspiel ist die Gegenprobe: es beweist im selben Lauf,
// dass das Gate scharf ist und nicht etwa gar nicht greift.
func TestAusrichterGate_AuswaertsUndGenerischUnberuehrt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamA)
	h := newRegenTestHandler(t, db)

	fremder := insertAusrichterI(t, db, "TV Ötlingen")
	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tpl := insertTemplateI(t, db, kasse, 0, 1)
	bindItemsToAusrichterI(t, db, tpl, fremder)

	// Der Tag löst auf den Default auf, `fremder` passt also nirgends.
	auswaerts := insertEventI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl, "auswärts")
	generisch := insertEventI(t, db, seasonID, teamA, "2026-06-13", "12:00", tpl, "generisch")
	heim := insertEventI(t, db, seasonID, teamA, "2026-06-13", "14:00", tpl, "heim")

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	for _, tc := range []struct {
		name   string
		gameID int
		want   int
	}{
		{"auswärts", auswaerts, 1},
		{"generisch", generisch, 1},
		{"heim (Gegenprobe: gegatet)", heim, 0},
	} {
		if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", tc.gameID); got != tc.want {
			t.Errorf("%s: expected %d auto slots, got %d", tc.name, tc.want, got)
		}
	}
}

// TestAusrichterGate_EinTagesWertFuerAlleSpiele (spec: "Der Ausrichter wird einmal je
// Tag aufgelöst"): vier Heimspiele an einem Tag, zwei Vorlagen an zwei verschiedene
// Ausrichter gebunden. Genau die Spiele mit der auf den Tageswert passenden Vorlage
// bekommen Slots — und zwar konsistent über den ganzen Tag, ohne dass sich der
// verwendete Wert zwischen zwei Spielen unterscheiden könnte.
//
// Dass die Auflösung wirklich nur EIN Lesevorgang ist, garantiert der Code
// strukturell und nicht dieser Test: regenSingleDay ruft
// settings.ResolveAusrichterForDay genau einmal auf und hält das Ergebnis als
// `dayAusrichterID int` — beide Gates bekommen diesen int by value übergeben, keines
// von beiden erhält einen Querier. Ein zweiter Lesevorgang wäre damit nicht nur
// unterlassen, sondern typseitig ausgeschlossen. Ein zählender RowQuerier-Wrapper
// wäre hier nicht anschließbar (regenSingleDay liest über die eigene *sql.Tx) und
// verlangte eine Produktions-Naht, die nur der Test bräuchte.
func TestAusrichterGate_EinTagesWertFuerAlleSpiele(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamA)
	h := newRegenTestHandler(t, db)

	ausrichterX := insertAusrichterI(t, db, "TV Ötlingen")
	ausrichterY := insertAusrichterI(t, db, "SG Musterhausen")
	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tplX := insertTemplateI(t, db, kasse, 0, 1)
	tplY := insertTemplateI(t, db, kasse, 0, 1)
	bindItemsToAusrichterI(t, db, tplX, ausrichterX)
	bindItemsToAusrichterI(t, db, tplY, ausrichterY)

	setDayAusrichterI(t, db, "2026-06-13", seasonID, ausrichterX)

	games := map[int]int{} // gameID → erwartete Slot-Anzahl
	games[insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tplX)] = 1
	games[insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tplY)] = 0
	games[insertGameI(t, db, seasonID, teamA, "2026-06-13", "11:00", tplX)] = 1
	games[insertGameI(t, db, seasonID, teamA, "2026-06-13", "12:00", tplY)] = 0

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	for gameID, want := range games {
		if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", gameID); got != want {
			t.Errorf("game %d: expected %d auto slots for the day's host, got %d", gameID, want, got)
		}
	}

	// Der Tageswert gilt für alle Termine des Tages: ein Wechsel dreht das Bild
	// vollständig um, nicht nur für einzelne Spiele.
	setDayAusrichterI(t, db, "2026-06-13", seasonID, ausrichterY)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	for gameID, wasWanted := range games {
		want := 1 - wasWanted
		if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", gameID); got != want {
			t.Errorf("nach Ausrichter-Wechsel: game %d expected %d auto slots, got %d", gameID, want, got)
		}
	}
}

// TestAusrichterGate_ZusageAufAusgegatetemItemLaeuftAlsRemoved (tasks.md 5.5): das
// Gate braucht KEINEN Sonderfall in snapshotCustomSlots/restoreAssignments. Ein
// ausgegatetes Item erzeugt keinen neuen Slot, der Match-Schlüssel der bestehenden
// Zusage trifft also auf nichts — sie läuft regulär über buildNotificationIntents.
//
// Das ist eine Annahme über den Zusammenbau dreier Stellen (Gate → kein outcome-
// Eintrag → default-Zweig in buildNotificationIntents) und deshalb hier festgenagelt
// statt nur geglaubt: die Meldung muss "removed" sein, NICHT "variant_changed" (das
// hieße fälschlich "dein Dienst wurde zu einer anderen Variante geändert"), und sie
// darf vor allem nicht stillschweigend ausbleiben.
func TestAusrichterGate_ZusageAufAusgegatetemItemLaeuftAlsRemoved(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamA)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	ausrichterA := insertAusrichterI(t, db, "TV Ötlingen")
	ausrichterB := insertAusrichterI(t, db, "SG Musterhausen")
	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tpl := insertTemplateI(t, db, kasse, 0, 1)
	bindItemsToAusrichterI(t, db, tpl, ausrichterA)

	setDayAusrichterI(t, db, "2026-06-13", seasonID, ausrichterA)
	gameID := insertGameI(t, db, seasonID, teamA, "2026-06-13", "18:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	slotID := autoSlotID(t, db, gameID)
	assignI(t, db, slotID, userID, "assigned", nil, nil)

	// Der Ausrichter des Tages wechselt → die gebundene Zeile fällt weg.
	setDayAusrichterI(t, db, "2026-06-13", seasonID, ausrichterB)
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := countRowsI(t, db, "duty_slots", "game_id=? AND is_custom=0", gameID); got != 0 {
		t.Errorf("expected the gated-out item to leave no slot behind, got %d", got)
	}
	if got := countRowsI(t, db, "duty_assignments", "user_id=?", userID); got != 0 {
		t.Errorf("expected the assignment to be gone with its slot, got %d", got)
	}

	var intents []NotificationIntent
	for _, n := range summary.Notifications {
		if n.UserID == userID {
			intents = append(intents, n)
		}
	}
	if len(intents) != 1 {
		t.Fatalf("expected exactly one notification intent for user %d, got %+v", userID, summary.Notifications)
	}
	if intents[0].Kind != "removed" {
		t.Errorf("expected Kind \"removed\" (nicht %q) — eine ausrichter-bedingt weggefallene Zusage ist entfernt, nicht variantengeändert", intents[0].Kind)
	}
	if intents[0].NewType != "" {
		t.Errorf("expected no NewType on a removed intent, got %q", intents[0].NewType)
	}
	found := false
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected user %d in NotifiedUsers, got %v", userID, summary.NotifiedUsers)
	}
}

// TestAusrichterGate_GebundenAnTagesAusrichterErzeugtSlots (spec: "Gebundene Zeile
// erzeugt nur bei passendem Ausrichter", positive Hälfte): dasselbe Item, das oben
// nichts erzeugt, erzeugt bei passendem Tageswert die vollen Slots — inklusive
// Rotations-Pfad, damit auch Gate #1 in seiner Durchlass-Richtung abgedeckt ist.
func TestAusrichterGate_GebundenAnTagesAusrichterErzeugtSlots(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	seedAgeClassRuleI(t, db, teamA)
	seedAgeClassRuleI(t, db, teamB)
	h := newRegenTestHandler(t, db)
	setVerhaeltnisI(t, db, "1")

	// Der Default ist hier bewusst der passende Wert: ein Tag ohne expliziten
	// Eintrag löst total auf den Default auf, ein an ihn gebundenes Item greift also.
	standard := defaultAusrichterI(t, db)
	kuchen := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	tpl := insertRotationTemplateI(t, db, kuchen, 0, 1, 2)
	bindItemsToAusrichterI(t, db, tpl, standard)

	games := []int{
		insertGameI(t, db, seasonID, teamA, "2026-06-13", "09:00", tpl),
		insertGameI(t, db, seasonID, teamB, "2026-06-13", "10:00", tpl),
	}

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	assertTeams(t, db, kuchen, games, []int{teamA, teamA})
	if len(summary.Unassigned) != 0 {
		t.Errorf("expected no unassigned entries, got %+v", summary.Unassigned)
	}
}
