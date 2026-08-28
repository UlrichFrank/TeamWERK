package games

// Ablösung im Regen (Change dienst-abloesung): trägt ein Dienst das Kennzeichen
// `end_at_next_duty`, wird das aus End-Anker und End-Versatz aufgelöste Ende zum DECKEL —
// der Slot endet, sobald der nächste gleichartige Dienst desselben Spieltags beginnt.
//
//	Ende = MIN( Start des nächsten gleichartigen Dienstes , Deckel )
//
// Gekappt wird in applyChainCaps, NACH allen Inserts des Tages, über die real
// entstandenen duty_slots — nicht über vorhergesagte. Die Tests hier prüfen genau die
// Konsequenzen daraus: dass die Kette nur reale Slots sieht (manuell angelegte und
// ausgenommene inklusive), dass sie nur nach vorn kappt und dass sie nie einen Dienst
// entfallen lässt.
//
// Geteilte Helfer: regen_bulk_context_test.go (insertDutyTypeI/insertTemplateI/
// insertGameI/regenDate), dienst_dauer_regen_test.go (slotHours/setItemHours),
// dienst_dauer_dynamisch_test.go (setItemDuration/seedAgeClass).

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// setItemEndAtNext schaltet das Ablösungs-Kennzeichen an der (einzigen) Vorlagen-Zeile.
func setItemEndAtNext(t *testing.T, db *sql.DB, templateID int, on bool) {
	t.Helper()
	if _, err := db.Exec(`UPDATE game_template_items SET end_at_next_duty=? WHERE template_id=?`,
		on, templateID); err != nil {
		t.Fatalf("setItemEndAtNext: %v", err)
	}
}

// abloesungTemplate baut die Standard-Kuchenvorlage dieses Files: Start Anpfiff−30,
// Ende Spielende+30, Ablösung aktiv.
func abloesungTemplate(t *testing.T, db *sql.DB, dutyID int) int {
	t.Helper()
	templateID := insertTemplateI(t, db, dutyID, -30, 1)
	setItemHours(t, db, templateID, 2.0)
	setItemDuration(t, db, templateID, "dynamisch", "end", 30)
	setItemEndAtNext(t, db, templateID, true)
	return templateID
}

// TestRegen_DienstEndetBeiAbloesung: der Kernfall. Zwei Heimspiele hintereinander, beide
// mit Kuchendienst — der erste endet, wenn der zweite beginnt, statt bis zum Ende SEINES
// Spiels weiterzulaufen.
//
//	Spiel 1  10:00–11:15   Slot 09:30, Deckel 11:45  →  gekappt auf 11:15 (Start Slot 2)
//	Spiel 2  11:45–13:00   Slot 11:15, Deckel 13:30  →  kein Nachfolger, Deckel gilt
func TestRegen_DienstEndetBeiAbloesung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID) // A-Jugend: 2×30 + 15 = 75 min
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:45", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// 09:30 → 11:15 = 105 min
	if got, want := slotHours(t, db, game1), 105.0/60.0; got != want {
		t.Errorf("Slot 1 soll bei der Ablösung (11:15) enden: erwartet %v h, got %v", want, got)
	}
	// 11:15 → 13:30 = 135 min (Deckel)
	if got, want := slotHours(t, db, game2), 135.0/60.0; got != want {
		t.Errorf("Slot 2 hat keinen Nachfolger und behält seinen Deckel: erwartet %v h, got %v", want, got)
	}
}

// TestRegen_LetzterDienstDerKetteBehaeltDenDeckel: ohne Nachfolger ändert das Kennzeichen
// nichts — die Dauer ist identisch zu der ohne gesetztes Kennzeichen.
func TestRegen_LetzterDienstDerKetteBehaeltDenDeckel(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	withFlag := slotHours(t, db, gameID)

	// Gegenprobe: dieselbe Konstellation ohne Kennzeichen.
	setItemEndAtNext(t, db, templateID, false)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	withoutFlag := slotHours(t, db, gameID)

	if withFlag != withoutFlag {
		t.Errorf("ohne Nachfolger darf das Kennzeichen nichts ändern: mit=%v, ohne=%v", withFlag, withoutFlag)
	}
	if want := 135.0 / 60.0; withFlag != want {
		t.Errorf("erwartet den Deckel (09:30–11:45 = %v h), got %v", want, withFlag)
	}
}

// TestRegen_NachfolgerNachDemDeckelKapptNicht: die Ablösung zieht ein Ende nur nach vorn.
// Liegt der nächste gleichartige Dienst hinter dem gepflegten Ende, bleibt es beim Deckel —
// die Dauer wird NICHT verlängert.
func TestRegen_NachfolgerNachDemDeckelKapptNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	// Spiel 1 10:00 (Deckel 11:45), Spiel 2 erst 16:00 → dessen Slot startet 15:30.
	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "16:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got, want := slotHours(t, db, game1), 135.0/60.0; got != want {
		t.Errorf("Nachfolger liegt hinter dem Deckel — erwartet unveränderte %v h, got %v", want, got)
	}
}

// TestRegen_NachfolgerVorDemEigenenStartWirdIgnoriert: ein Slot desselben Diensttyps, der
// VOR dem eigenen Start beginnt, ist keine Ablösung. Buchstäbliches min() ergäbe hier eine
// negative Spanne und löschte den Dienst — ein vertippter Versatz darf den Kuchendienst
// aber nicht entfernen, er fällt auf seinen Deckel zurück (design.md §4).
func TestRegen_NachfolgerVorDemEigenenStartWirdIgnoriert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	// Spiel 1 um 12:00, Vorlage mit Start −30 → Slot 11:30, Deckel 13:45.
	tplNormal := abloesungTemplate(t, db, dutyID)
	// Spiel 2 um 12:30, Vorlage mit Start −120 → Slot 10:30, also VOR Slot 1.
	tplFrueh := insertTemplateI(t, db, dutyID, -120, 1)
	setItemHours(t, db, tplFrueh, 2.0)
	setItemDuration(t, db, tplFrueh, "dynamisch", "end", 30)
	setItemEndAtNext(t, db, tplFrueh, true)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", tplNormal)
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:30", tplFrueh)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Slot 1 (11:30): der einzige andere Slot startet 10:30, also davor → keine Ablösung.
	if got, want := slotHours(t, db, game1), 135.0/60.0; got != want {
		t.Errorf("rückwärts liegender Slot darf nicht ablösen — erwartet Deckel %v h, got %v", want, got)
	}
	// Und er existiert überhaupt noch.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duty_slots WHERE game_id=?`, game1).Scan(&n); err != nil {
		t.Fatalf("count slots: %v", err)
	}
	if n != 1 {
		t.Errorf("der Slot muss bestehen bleiben, fand %d", n)
	}
}

// TestRegen_AbloesungNurDurchDenselbenDiensttyp: „gleichartig" heißt derselbe Diensttyp.
// Ein Aufbau-Dienst zwischen zwei Kuchendiensten löst den Kuchendienst nicht ab.
func TestRegen_AbloesungNurDurchDenselbenDiensttyp(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	kuchenID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	aufbauID := insertDutyTypeI(t, db, "Aufbau", "", 0, "", 0)

	tplKuchen := abloesungTemplate(t, db, kuchenID)
	// Zweite Vorlage, anderer Diensttyp, Slot um 11:15 — genau dort, wo eine Ablösung
	// des Kuchendienstes läge.
	tplAufbau := insertTemplateI(t, db, aufbauID, -30, 1)
	setItemHours(t, db, tplAufbau, 1.0)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", tplKuchen)
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:45", tplAufbau)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got, want := slotHours(t, db, game1), 135.0/60.0; got != want {
		t.Errorf("ein anderer Diensttyp darf nicht ablösen — erwartet Deckel %v h, got %v", want, got)
	}
}

// TestRegen_VarianteBestimmtDasKennzeichen: greift die Varianten-Logik, entsteht der Slot
// unter einem anderen Diensttyp — und dann gilt DESSEN Kennzeichen, nicht das der
// Vorlagen-Zeile. Dieselbe Regel wie für Modus, End-Anker und End-Versatz: ob sich eine
// Arbeit ablösen lässt, gehört zur Arbeit (design.md §5).
//
// Aufbau: drei Heimspiele 10:00/11:00/12:00, Item mit Start −30. Die Slots der Spiele 2
// und 3 (10:30 und 11:30) liegen zwischen zwei Spielen → same_day_behavior 'reduced'
// greift und macht sie zur Variante. Slot 2 hätte damit Slot 3 als Ablösung.
func TestRegen_VarianteBestimmtDasKennzeichen(t *testing.T) {
	for _, tc := range []struct {
		name          string
		variantEndsAt bool
		wantHours     float64
	}{
		// Variante ohne Kennzeichen → keine Kappung, der Deckel gilt:
		// Slot 10:30 → Spielende 12:15 + 30 = 12:45 → 135 min.
		{"Variante ohne Kennzeichen kappt nicht", false, 135.0 / 60.0},
		// Variante mit Kennzeichen → Kappung auf den Start von Slot 3 (11:30) → 60 min.
		{"Variante mit Kennzeichen kappt", true, 1.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			seasonID := testutil.CreateSeason(t, db, "2025/26")
			teamID := testutil.CreateTeam(t, db, "Team A")
			seedAgeClassRuleI(t, db, teamID)
			h := newRegenTestHandler(t, db)

			variantID := insertDutyTypeI(t, db, "Kuchen kurz", "", 0, "", 0)
			// Die Variante trägt ihre eigene vollständige Dauer-Definition.
			if _, err := db.Exec(`
				UPDATE duty_types
				   SET duration_mode='dynamisch', end_anchor='end', end_offset_minutes=30, end_at_next_duty=?
				 WHERE id=?`, tc.variantEndsAt, variantID); err != nil {
				t.Fatalf("seed variant: %v", err)
			}
			kuchenID := insertDutyTypeI(t, db, "Kuchen", "reduced", variantID, "", 0)

			// Die Vorlagen-Zeile trägt das Kennzeichen IMMER — es darf im Reduce-Fall
			// nicht gewinnen.
			templateID := abloesungTemplate(t, db, kuchenID)

			insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
			game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:00", templateID)
			insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)
			regenDate(t, h, db, "2026-06-13", seasonID, nil)

			// Sicherstellen, dass die Varianten-Logik überhaupt gegriffen hat — sonst
			// prüfte der Test nichts.
			var gotType int
			if err := db.QueryRow(
				`SELECT duty_type_id FROM duty_slots WHERE game_id=? AND is_custom=0`, game2).Scan(&gotType); err != nil {
				t.Fatalf("read slot type: %v", err)
			}
			if gotType != variantID {
				t.Fatalf("Testaufbau: Slot 2 sollte zur Variante reduziert sein, trägt aber Typ %d", gotType)
			}
			if got := slotHours(t, db, game2); got != tc.wantHours {
				t.Errorf("erwartet %v h, got %v", tc.wantHours, got)
			}
		})
	}
}

// TestRegen_ManuellerSlotLoestAbAberWirdNichtGekappt: ein von Hand angelegter Dienst
// desselben Typs ist ein realer Nachfolger und löst deshalb ab — gekappt wird er selbst
// nie, weil is_custom=1-Slots nie Kandidat sind. Beides fällt daraus ab, dass
// applyChainCaps über die echten duty_slots läuft (design.md §3).
func TestRegen_ManuellerSlotLoestAbAberWirdNichtGekappt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	// Von Hand angelegter Kuchendienst um 11:15, ohne Spielbezug.
	if _, err := db.Exec(`
		INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc,
		                        slots_total, season_id, is_custom, hours_value)
		VALUES ('Manuell', '2026-06-13', '11:15', ?, '', 1, ?, 1, 4.0)`,
		dutyID, seasonID); err != nil {
		t.Fatalf("seed custom slot: %v", err)
	}

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got, want := slotHours(t, db, game1), 105.0/60.0; got != want {
		t.Errorf("der manuelle Slot soll ablösen: erwartet %v h, got %v", want, got)
	}
	var customHours float64
	if err := db.QueryRow(
		`SELECT hours_value FROM duty_slots WHERE is_custom=1 AND event_date='2026-06-13'`).Scan(&customHours); err != nil {
		t.Fatalf("read custom slot: %v", err)
	}
	if customHours != 4.0 {
		t.Errorf("der manuelle Slot darf nicht gekappt werden: erwartet 4.0, got %v", customHours)
	}
}

// TestRegen_AusgenommenerTerminLoestTrotzdemAb: „Ausnahme ≠ Kontext" — ein von einem
// Massenlauf ausgenommener Termin behält seine Slots, und die zählen für die Ablösung der
// neu erzeugten Dienste mit.
func TestRegen_AusgenommenerTerminLoestTrotzdemAb(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:45", templateID)

	// Erster Lauf erzeugt beide Slots.
	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	// Zweiter Lauf nimmt Spiel 2 aus — sein Slot (11:15) bleibt unangetastet stehen.
	regenDate(t, h, db, "2026-06-13", seasonID, map[int]bool{game2: true})

	if got, want := slotHours(t, db, game1), 105.0/60.0; got != want {
		t.Errorf("der Slot des ausgenommenen Termins soll ablösen: erwartet %v h, got %v", want, got)
	}
}

// TestRegen_KappungErzeugtNieEineNichtpositiveDauer: die tragende Invariante. Als
// Nachfolger zählt nur, was echt nach dem eigenen Start beginnt, und der Deckel lag beim
// Insert schon danach — das Minimum bleibt also immer positiv. Die Kappung ist deshalb ein
// reines UPDATE: kein Slot verschwindet, kein invalid_span entsteht.
func TestRegen_KappungErzeugtNieEineNichtpositiveDauer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := abloesungTemplate(t, db, dutyID)

	// Vier eng gestaffelte Heimspiele — jede Ablösung folgt dicht auf die vorherige.
	for _, ts := range []string{"10:00", "10:05", "10:10", "10:15"} {
		insertGameI(t, db, seasonID, teamID, "2026-06-13", ts, templateID)
	}
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if len(summary.InvalidSpan) != 0 {
		t.Errorf("die Kappung darf keinen invalid_span erzeugen, got %+v", summary.InvalidSpan)
	}
	rows, err := db.Query(`SELECT id, hours_value FROM duty_slots WHERE event_date='2026-06-13'`)
	if err != nil {
		t.Fatalf("query slots: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id int
		var hours float64
		if err := rows.Scan(&id, &hours); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if hours <= 0 {
			t.Errorf("Slot %d trägt nach der Kappung eine nicht-positive Dauer (%v)", id, hours)
		}
		n++
	}
	if n != 4 {
		t.Errorf("die Kappung darf keinen Slot entfallen lassen: erwartet 4, fand %d", n)
	}
}

// TestRegen_AbsoluterModusWirdNichtGekappt: im Modus `absolut` ist das Kennzeichen
// bedeutungslos — der Slot trägt exakt die gepflegte Stundenzahl, auch wenn ein
// gleichartiger Dienst früher beginnt.
func TestRegen_AbsoluterModusWirdNichtGekappt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, -30, 1)
	setItemHours(t, db, templateID, 3.0)
	setItemDuration(t, db, templateID, "absolut", "end", 30)
	setItemEndAtNext(t, db, templateID, true)

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:45", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := slotHours(t, db, game1); got != 3.0 {
		t.Errorf("absoluter Modus: erwartet die gepflegte Zahl 3.0, got %v", got)
	}
}

// TestRegen_OhneKennzeichenUnveraendert: Charakterisierung des Bestandsverhaltens.
// end_at_next_duty=0 (der Default) verhält sich exakt wie vor dem Change — der Slot läuft
// bis zum Ende seines eigenen Spiels weiter, auch wenn der nächste längst begonnen hat.
func TestRegen_OhneKennzeichenUnveraendert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, -30, 1)
	setItemHours(t, db, templateID, 2.0)
	setItemDuration(t, db, templateID, "dynamisch", "end", 30)
	// Kennzeichen bewusst NICHT gesetzt (DB-Default 0).

	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	insertGameI(t, db, seasonID, teamID, "2026-06-13", "11:45", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	// Deckel 09:30–11:45 = 135 min, trotz Nachfolger um 11:15.
	if got, want := slotHours(t, db, game1), 135.0/60.0; got != want {
		t.Errorf("ohne Kennzeichen soll nichts gekappt werden: erwartet %v h, got %v", want, got)
	}
}
