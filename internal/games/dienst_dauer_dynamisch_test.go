package games

// Dynamische Dienst-Dauer (Change dienst-dauer-dynamisch): im Modus "dynamisch"
// ergibt sich die Dauer aus der Differenz zweier aufgelöster Anker statt aus einer
// festen Stundenzahl — ein Zeitnehmer folgt damit der tatsächlichen Spieldauer, die
// je Altersklasse schwankt.
//
// Geteilte Helfer stammen aus regen_bulk_context_test.go; slotHours/setItemHours
// aus dienst_dauer_regen_test.go.

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// setItemDuration stellt Modus, End-Anker und End-Versatz der (einzigen) Zeile ein.
func setItemDuration(t *testing.T, db *sql.DB, templateID int, mode, endAnchor string, endOffset int) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE game_template_items SET duration_mode=?, end_anchor=?, end_offset_minutes=? WHERE template_id=?`,
		mode, endAnchor, endOffset, templateID); err != nil {
		t.Fatalf("setItemDuration: %v", err)
	}
}

// setItemAnchor stellt den START-Anker der (einzigen) Zeile ein. insertTemplateI legt
// ihn immer auf "start" an; für die Fälle, in denen Start und Ende an verschiedenen
// Ankern hängen müssen, wird er hier nachgezogen.
func setItemAnchor(t *testing.T, db *sql.DB, templateID int, anchor string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE game_template_items SET anchor=? WHERE template_id=?`,
		anchor, templateID); err != nil {
		t.Fatalf("setItemAnchor: %v", err)
	}
}

// seedAgeClass gibt einem Team eine Altersklasse mit eigener Halbzeit-/Pausenregel.
func seedAgeClass(t *testing.T, db *sql.DB, teamID int, ageClass string, half, brk int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, ageClass, teamID); err != nil {
		t.Fatalf("set age_class: %v", err)
	}
	// Bewusst kein INSERT OR IGNORE: age_class_game_rules hat einen CHECK auf genau
	// vier Klassen, und ein OR IGNORE würde einen Verstoß stumm schlucken. Der Regen
	// überspränge das Spiel dann kommentarlos und der Test schlüge mit "no rows"
	// irgendwo weiter hinten fehl.
	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?,?,?)`,
		ageClass, half, brk); err != nil {
		t.Fatalf("seed age_class_game_rules(%s): %v", ageClass, err)
	}
}

// TestRegen_AbsoluterModusUnveraendert (tasks 2.5): Regressionsschutz für das
// Herausziehen von resolveAnchorTime — der Default-Modus muss exakt bleiben.
func TestRegen_AbsoluterModusUnveraendert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, -30, 1)
	setItemHours(t, db, templateID, 1.5)
	// End-Felder sind gesetzt, dürfen im absoluten Modus aber nichts bewirken.
	setItemDuration(t, db, templateID, "absolut", "end", 999)

	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := slotHours(t, db, gameID); got != 1.5 {
		t.Errorf("absoluter Modus soll die gepflegte Zahl tragen (1.5), got %v", got)
	}
	var eventTime string
	if err := db.QueryRow(`SELECT event_time FROM duty_slots WHERE game_id=?`, gameID).Scan(&eventTime); err != nil {
		t.Fatalf("read event_time: %v", err)
	}
	if eventTime != "11:30" {
		t.Errorf("Startzeit soll unverändert Anpfiff−30min sein (11:30), got %q", eventTime)
	}
}

// TestRegen_DynamischeDauerFolgtSpieldauer (tasks 2.6): dieselbe Vorlage, zwei
// Altersklassen — die Slots bekommen unterschiedliche Dauern.
func TestRegen_DynamischeDauerFolgtSpieldauer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kurz := testutil.CreateTeam(t, db, "D-Jugend")
	lang := testutil.CreateTeam(t, db, "A-Jugend")
	seedAgeClass(t, db, kurz, "D-Jugend", 20, 5)  // 45 min Spielzeit
	seedAgeClass(t, db, lang, "A-Jugend", 30, 15) // 75 min Spielzeit
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Zeitnehmer", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, -15, 1)
	setItemHours(t, db, templateID, 1.0)
	// Start Anpfiff−15min, Ende Spielende+15min.
	setItemDuration(t, db, templateID, "dynamisch", "end", 15)

	gameKurz := insertGameI(t, db, seasonID, kurz, "2026-06-13", "10:00", templateID)
	gameLang := insertGameI(t, db, seasonID, lang, "2026-06-14", "10:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	regenDate(t, h, db, "2026-06-14", seasonID, nil)

	// D-Jugend: 15 + 45 + 15 = 75 min = 1.25 h
	if got := slotHours(t, db, gameKurz); got != 1.25 {
		t.Errorf("D-Jugend (45min Spiel): erwartet 1.25h, got %v", got)
	}
	// A-Jugend: 15 + 75 + 15 = 105 min
	want := 105.0 / 60.0
	if got := slotHours(t, db, gameLang); got != want {
		t.Errorf("A-Jugend (75min Spiel): erwartet %v, got %v", want, got)
	}
	if slotHours(t, db, gameKurz) == slotHours(t, db, gameLang) {
		t.Error("beide Altersklassen ergaben dieselbe Dauer — die Dauer folgt dem Spiel nicht")
	}
}

// TestRegen_DynamischeDauerNutztEndTime (tasks 2.7): eine gepflegte Endzeit am Termin
// hat Vorrang vor Anpfiff + errechneter Spieldauer.
func TestRegen_DynamischeDauerNutztEndTime(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID) // 30+15+30 = 75 min
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Abbau", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	setItemHours(t, db, templateID, 9.0)
	setItemDuration(t, db, templateID, "dynamisch", "end", 0)

	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	// Endzeit weicht bewusst von Anpfiff + 75min (= 11:15) ab.
	if _, err := db.Exec(`UPDATE games SET end_time=? WHERE id=?`, "12:00", gameID); err != nil {
		t.Fatalf("set end_time: %v", err)
	}
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// 10:00 → 12:00 = 2h. Ohne end_time wären es 1.25h.
	if got := slotHours(t, db, gameID); got != 2.0 {
		t.Errorf("gepflegte Endzeit soll Vorrang haben (2.0h), got %v", got)
	}
}

// TestRegen_DynamischeDauerAnkerStart (tasks 2.8): Halbzeit-Dienst — beide Anker am
// Anpfiff, die Dauer ist damit unabhängig von der Spieldauer.
func TestRegen_DynamischeDauerAnkerStart(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kurz := testutil.CreateTeam(t, db, "D-Jugend")
	lang := testutil.CreateTeam(t, db, "A-Jugend")
	seedAgeClass(t, db, kurz, "D-Jugend", 20, 5)
	seedAgeClass(t, db, lang, "A-Jugend", 30, 15)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Halbzeit-Verkauf", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, 25, 1)
	setItemHours(t, db, templateID, 9.0)
	// Start Anpfiff+25min, Ende Anpfiff+40min → immer 15 Minuten.
	setItemDuration(t, db, templateID, "dynamisch", "start", 40)

	g1 := insertGameI(t, db, seasonID, kurz, "2026-06-13", "10:00", templateID)
	g2 := insertGameI(t, db, seasonID, lang, "2026-06-14", "10:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	regenDate(t, h, db, "2026-06-14", seasonID, nil)

	want := 15.0 / 60.0
	if got := slotHours(t, db, g1); got != want {
		t.Errorf("D-Jugend: erwartet %v (15min), got %v", want, got)
	}
	if got := slotHours(t, db, g2); got != want {
		t.Errorf("A-Jugend: erwartet %v (15min), got %v — Anker start darf nicht von der Spieldauer abhängen", want, got)
	}
}

// TestRegen_UnaufloesbareDynamischeDauerErzeugtKeinenSlot (dienst-zeitmodus-strikt,
// Aufgabe 2.3): löst die dynamische Definition gegen den konkreten Termin keine positive
// Dauer auf, entsteht KEIN Slot — und insbesondere keiner mit der gepflegten absoluten
// Dauer. Genau der Rückfall, der bis hierher galt, ist entfallen: er ließ einen Slot mit
// einer plausiblen, aber falschen Zahl entstehen, ohne dass es je auffiel.
//
// Die Kombination hängt bewusst an VERSCHIEDENEN Ankern (Start am Spielende, Ende am
// Anpfiff+30) — nur dieser Fall überlebt die Schreib-Validierung und kann den Regen
// überhaupt erreichen. Bei gleichem Anker weisen die Routen die Definition ab
// (dynamicSpanImpossible), lange bevor ein Termin im Spiel ist.
func TestRegen_UnaufloesbareDynamischeDauerErzeugtKeinenSlot(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID) // 30+15+30 = 75 min Spieldauer
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	// Start am Spielende (+0), Ende bei Anpfiff+30 → 30 − 75 = −45 min.
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	setItemAnchor(t, db, templateID, "end")
	setItemHours(t, db, templateID, 1.5)
	setItemDuration(t, db, templateID, "dynamisch", "start", 30)

	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duty_slots WHERE game_id=?`, gameID).Scan(&count); err != nil {
		t.Fatalf("count slots: %v", err)
	}
	if count != 0 {
		var hours float64
		db.QueryRow(`SELECT hours_value FROM duty_slots WHERE game_id=?`, gameID).Scan(&hours)
		t.Fatalf("kein Slot erwartet, got %d (Dauer %v — Rückfall auf hours_value?)", count, hours)
	}
}

// TestRegen_UnaufloesbareDynamischeDauerStehtInDerZusammenfassung (dienst-zeitmodus-strikt,
// Aufgabe 2.3): derselbe Ausfall muss sichtbar sein. Ein still fehlender Dienst wäre
// nicht besser als die falsche Zahl davor — die Meldung ist die halbe Begründung des
// Changes. Sie steht in ihrer EIGENEN Liste, nicht in Skipped: Skipped meint die gewollte
// Auslassung der Varianten-Logik, invalid_span einen Definitionsfehler.
func TestRegen_UnaufloesbareDynamischeDauerStehtInDerZusammenfassung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	setItemAnchor(t, db, templateID, "end")
	setItemHours(t, db, templateID, 1.5)
	setItemDuration(t, db, templateID, "dynamisch", "start", 30)

	insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if len(summary.InvalidSpan) != 1 {
		t.Fatalf("genau ein invalid_span-Eintrag erwartet, got %d (%+v)", len(summary.InvalidSpan), summary.InvalidSpan)
	}
	got := summary.InvalidSpan[0]
	if got.Date != "2026-06-13" || got.DutyType != "Kuchen" {
		t.Errorf("Eintrag soll Datum und Diensttyp tragen, got %+v", got)
	}
	if len(summary.Skipped) != 0 {
		t.Errorf("der Ausfall gehört nicht in Skipped (gewollte Auslassung), got %+v", summary.Skipped)
	}
	if len(summary.Created) != 0 {
		t.Errorf("kein angelegter Slot erwartet, got %+v", summary.Created)
	}
}
