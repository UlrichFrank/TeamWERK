package duties_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// addKaderTrainerUser legt einen User mit Vereinsfunktion 'trainer' im Trainer-Kader des
// Teams an — bewusst OHNE Spieler-Kaderzeile: genau diese Konstellation blieb bisher
// unbenachrichtigt, weil die Empfängerbestimmung nur player_memberships kannte.
func addKaderTrainerUser(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, memberID); err != nil {
		t.Fatalf("insert club function trainer: %v", err)
	}
	testutil.AddKaderTrainer(t, db, testutil.CreateKader(t, db, teamID, seasonID), memberID)
	return userID
}

// addParentOfPlayer legt ein Elternteil an, dessen Kind als Spieler im Kader des Teams steht.
func addParentOfPlayer(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, childMemberID); err != nil {
		t.Fatalf("insert club function spieler: %v", err)
	}
	addPlayerMembership(t, db, childMemberID, teamID, seasonID)
	testutil.AddFamilyLink(t, db, parentUserID, childMemberID)
	return parentUserID
}

// postSlot legt einen Slot für ein Team an und gibt die Empfängermenge der
// „Neuer Dienst"-Meldung zurück (gelesen aus user_events, siehe notifiedUsers).
func postSlot(t *testing.T, db *sql.DB, body map[string]any) map[int]bool {
	t.Helper()
	adminID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	res := testutil.Post(t, srv, "/api/duty-slots", testutil.Token(t, adminID, "admin", nil), body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	return notifiedUsers(t, db)
}

// Ein Slot mit Zielgruppe „eltern" ist für Spieler und Trainer in der Dienstbörse
// unsichtbar — eine Push an sie wäre eine Meldung, die ins Leere klickt.
func TestCreateSlot_ZielgruppeEltern_BenachrichtigtNurEltern(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")

	player := addPlayer(t, db, teamA, seasonID)
	trainer := addKaderTrainerUser(t, db, teamA, seasonID)
	parent := addParentOfPlayer(t, db, teamA, seasonID)

	got := postSlot(t, db, map[string]any{
		"event_name":   "Kuchenverkauf",
		"event_date":   "2026-06-14",
		"duty_type_id": createDutyType(t, db, "Bewirtung", 2.0),
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    seasonID,
		"audiences":    []string{"eltern"},
	})

	if !got[parent] {
		t.Errorf("Elternteil (%d) fehlt in der Benachrichtigung: %v", parent, got)
	}
	if got[player] {
		t.Errorf("Spieler (%d) trifft die Zielgruppe 'eltern' nicht: %v", player, got)
	}
	if got[trainer] {
		t.Errorf("Trainer (%d) trifft die Zielgruppe 'eltern' nicht: %v", trainer, got)
	}
}

// Gegenprobe: Zielgruppe „trainer" erreicht den Trainer des Kaders — auch wenn er dort
// keine Spieler-Kaderzeile hat.
func TestCreateSlot_ZielgruppeTrainer_BenachrichtigtTrainerDesKaders(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")

	player := addPlayer(t, db, teamA, seasonID)
	trainer := addKaderTrainerUser(t, db, teamA, seasonID)

	got := postSlot(t, db, map[string]any{
		"event_name":   "Zeitnehmer",
		"event_date":   "2026-06-14",
		"duty_type_id": createDutyType(t, db, "Zeitnehmer", 2.0),
		"slots_total":  1,
		"team_id":      teamA,
		"season_id":    seasonID,
		"audiences":    []string{"trainer"},
	})

	if !got[trainer] {
		t.Errorf("Trainer des Kaders (%d) fehlt in der Benachrichtigung: %v", trainer, got)
	}
	if got[player] {
		t.Errorf("Spieler (%d) trifft die Zielgruppe 'trainer' nicht: %v", player, got)
	}
}

// Ohne eigene Zielgruppe am Slot gilt die Vorbelegung des Diensttyps — dieselbe
// COALESCE-Regel, die die Dienstbörse beim Lesen anwendet.
func TestCreateSlot_ZielgruppeAusDiensttyp_WirdAngewendet(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")

	player := addPlayer(t, db, teamA, seasonID)
	parent := addParentOfPlayer(t, db, teamA, seasonID)

	dtID := createDutyType(t, db, "Bewirtung", 2.0)
	if _, err := db.Exec(`UPDATE duty_types SET audiences='["eltern"]' WHERE id=?`, dtID); err != nil {
		t.Fatalf("set duty type audiences: %v", err)
	}

	got := postSlot(t, db, map[string]any{
		"event_name":   "Kuchenverkauf",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    seasonID,
	})

	if !got[parent] {
		t.Errorf("Elternteil (%d) fehlt in der Benachrichtigung: %v", parent, got)
	}
	if got[player] {
		t.Errorf("Spieler (%d) trifft die geerbte Zielgruppe 'eltern' nicht: %v", player, got)
	}
}

// Bestandszusage: ohne Zielgruppe bleibt es beim ganzen Kader inklusive Eltern.
func TestCreateSlot_OhneZielgruppe_BenachrichtigtDenGanzenKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")

	player := addPlayer(t, db, teamA, seasonID)
	trainer := addKaderTrainerUser(t, db, teamA, seasonID)
	parent := addParentOfPlayer(t, db, teamA, seasonID)
	fremd := addPlayer(t, db, teamB, seasonID)

	got := postSlot(t, db, map[string]any{
		"event_name":   "Hallenaufbau",
		"event_date":   "2026-06-14",
		"duty_type_id": createDutyType(t, db, "Aufbau", 2.0),
		"slots_total":  3,
		"team_id":      teamA,
		"season_id":    seasonID,
	})

	for name, uid := range map[string]int{"Spieler": player, "Trainer": trainer, "Elternteil": parent} {
		if !got[uid] {
			t.Errorf("%s (%d) fehlt in der Benachrichtigung: %v", name, uid, got)
		}
	}
	if got[fremd] {
		t.Errorf("Spieler eines fremden Teams (%d) darf nicht benachrichtigt werden: %v", fremd, got)
	}
}

// Eine Kaderzeile aus einer abgeschlossenen Saison begründet keine Betroffenheit —
// sonst benachrichtigt der Verein seine Ehemaligen bis in alle Ewigkeit.
func TestCreateSlot_AltePlayerMembership_WirdNichtBenachrichtigt(t *testing.T) {
	db := testutil.NewDB(t)
	teamA := testutil.CreateTeam(t, db, "A-Jugend")

	altSeason := testutil.CreateSeason(t, db, "2023/24")
	ehemaliger := addPlayer(t, db, teamA, altSeason)

	// CreateSeason deaktiviert die vorherige Saison — ab hier ist 2025/26 aktiv.
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	aktueller := addPlayer(t, db, teamA, seasonID)

	got := postSlot(t, db, map[string]any{
		"event_name":   "Hallenaufbau",
		"event_date":   "2026-06-14",
		"duty_type_id": createDutyType(t, db, "Aufbau", 2.0),
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    seasonID,
	})

	if !got[aktueller] {
		t.Errorf("Spieler der aktiven Saison (%d) fehlt in der Benachrichtigung: %v", aktueller, got)
	}
	if got[ehemaliger] {
		t.Errorf("Spieler nur aus der alten Saison (%d) darf nicht benachrichtigt werden: %v", ehemaliger, got)
	}
}
