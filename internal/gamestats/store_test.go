package gamestats

import (
	"context"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Die Kerninvariante dieses Changes: ein Staffel-Spielplan mit 90 Begegnungen
// darf keine einzige Zeile in `games` erzeugen. Sonst leckten fremde Spiele in
// Spielplan, Dienst-Regeneration, iCal-Feed, RSVP und Anwesenheit.
func TestSaveSchedule_ErzeugtKeineGamesZeilen(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	before := countRows(t, db, "games")

	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(90)); err != nil {
		t.Fatalf("SaveSchedule: %v", err)
	}
	if after := countRows(t, db, "games"); after != before {
		t.Errorf("games ging von %d auf %d — der Staffel-Spielplan darf dort nichts anlegen", before, after)
	}
	if n := countRows(t, db, "bwhv_games"); n != 90 {
		t.Errorf("bwhv_games = %d, erwartet 90", n)
	}
}

func TestSaveSchedule_VerknuepftEigenesSpielUeberExternalId(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-09-20")
	if _, err := db.Exec(`UPDATE games SET external_id = ? WHERE id = ?`, "900000", gameID); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(3)); err != nil {
		t.Fatalf("SaveSchedule: %v", err)
	}
	games, err := s.StaffelGames(context.Background(), staffelID)
	if err != nil {
		t.Fatal(err)
	}
	var linked, unlinked int
	for _, g := range games {
		if g.GameID != nil {
			linked++
			if *g.GameID != gameID || g.GameNo != "900000" {
				t.Errorf("falsche Verknüpfung: Spiel %s -> games %d", g.GameNo, *g.GameID)
			}
		} else {
			unlinked++
		}
	}
	if linked != 1 || unlinked != 2 {
		t.Errorf("verknüpft = %d, unverknüpft = %d; erwartet 1 und 2", linked, unlinked)
	}
}

// Ein zweiter Lauf mit identischen Daten darf nichts als geändert melden —
// sonst broadcastet jeder Poll und jede offene Sitzung lädt grundlos nach.
func TestSaveSchedule_IdempotentBeiUnveraendertenDaten(t *testing.T) {
	_, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	if _, err := s.SaveSchedule(ctx, staffelID, sampleSchedule(5)); err != nil {
		t.Fatal(err)
	}
	changed, err := s.SaveSchedule(ctx, staffelID, sampleSchedule(5))
	if err != nil {
		t.Fatal(err)
	}
	if changed != 0 {
		t.Errorf("zweiter Lauf meldete %d Änderungen, erwartet 0", changed)
	}
}

func TestSaveSchedule_ErgebnisNachtraeglichErkanntAlsAenderung(t *testing.T) {
	_, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	sch := sampleSchedule(2)
	sch.Games[1].SGID = ""
	if _, err := s.SaveSchedule(ctx, staffelID, sch); err != nil {
		t.Fatal(err)
	}
	sch.Games[1].SGID = "999"
	sch.Games[1].HomeGoals, sch.Games[1].GuestGoals = intp(20), intp(19)
	changed, err := s.SaveSchedule(ctx, staffelID, sch)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Errorf("Änderungen = %d, erwartet 1", changed)
	}
}

func TestStaffelTable_WirdMitGespeichert(t *testing.T) {
	_, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	if _, err := s.SaveSchedule(ctx, staffelID, sampleSchedule(1)); err != nil {
		t.Fatal(err)
	}
	table, err := s.StaffelTable(ctx, staffelID)
	if err != nil {
		t.Fatal(err)
	}
	if len(table) != 1 || table[0].TeamName != "Verein A" {
		t.Errorf("Tabelle = %+v, erwartet eine Zeile für Verein A", table)
	}
}

// Übungsgruppen tragen keine Staffel — die Einschränkung steht im Handler,
// die Abfrage muss sie trotzdem respektieren.
func TestStaffelCodes_NurMannschaftsKader(t *testing.T) {
	db, s, seasonID, _ := newStaffel(t)
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	practiceID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	if _, err := db.Exec(`UPDATE kader SET staffel = 'mB-RL-BW' WHERE id = ?`, kaderID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE kader SET staffel = 'mC-OL-3-BW' WHERE id = ?`, practiceID); err != nil {
		t.Fatal(err)
	}

	codes, err := s.StaffelCodes(context.Background(), seasonID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := codes["mB-RL-BW"]; !ok {
		t.Errorf("Mannschafts-Kader fehlt: %v", codes)
	}
	if _, ok := codes["mC-OL-3-BW"]; ok {
		t.Errorf("Übungsgruppe darf keine Staffel beisteuern: %v", codes)
	}
}

// Der gemeldete Fehler: eine am Kader gepflegte Staffel war auf /staffeln
// unsichtbar, solange noch kein Abruf gelaufen war — die Liste las FROM
// bwhv_staffeln statt FROM kader. Die Zuordnung ist aber das, was der Vorstand
// pflegt; der Snapshot ist die Folge.
func TestListStaffelnWithTeam_ZeigtZuordnungOhneAbruf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`UPDATE kader SET staffel = 'mB-RL-BW' WHERE id = ?`, kaderID); err != nil {
		t.Fatal(err)
	}
	// Bewusst KEINE bwhv_staffeln-Zeile: es lief noch kein Abruf.

	list, err := NewStore(db).ListStaffelnWithTeam(context.Background(), seasonID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("Zuordnungen = %d, erwartet 1 — die Staffel steht am Kader", len(list))
	}
	if list[0].Code != "mB-RL-BW" || list[0].TeamName != "B-Jugend männlich" {
		t.Errorf("Eintrag = %+v, erwartet mB-RL-BW / B-Jugend männlich", list[0])
	}
	if list[0].ID != 0 || list[0].Polled {
		t.Errorf("ohne Abruf erwartet ID 0 und Polled=false, bekam ID %d / Polled %v",
			list[0].ID, list[0].Polled)
	}
}

func TestListStaffelnWithTeam_PolledNachAbruf(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`UPDATE kader SET staffel = 'mB-RL-BW' WHERE id = ?`, kaderID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(2)); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListStaffelnWithTeam(context.Background(), seasonID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("Zuordnungen = %d, erwartet 1", len(list))
	}
	if list[0].ID != staffelID || !list[0].Polled {
		t.Errorf("nach Abruf erwartet ID %d und Polled=true, bekam ID %d / Polled %v",
			staffelID, list[0].ID, list[0].Polled)
	}
}

// Ein Kader ohne Staffel taucht nicht auf, eine Übungsgruppe ebenfalls nicht.
func TestListStaffelnWithTeam_OhneZuordnungLeer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	testutil.CreateKader(t, db, teamID, seasonID)
	practiceID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	if _, err := db.Exec(`UPDATE kader SET staffel = 'mC-OL-3-BW' WHERE id = ?`, practiceID); err != nil {
		t.Fatal(err)
	}

	list, err := NewStore(db).ListStaffelnWithTeam(context.Background(), seasonID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("erwartet leer, bekam %+v", list)
	}
}
