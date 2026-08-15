package duties_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// boardVenueGroup ist die minimale Sicht auf die /api/duty-board-Antwort, die
// diese Tests brauchen: der Spielort und genug Identität, um Gruppen und Slots
// mengenmäßig zu vergleichen.
type boardVenueGroup struct {
	GameID *int   `json:"game_id"`
	Venue  string `json:"venue"`
	Slots  []struct {
		ID int `json:"id"`
	} `json:"slots"`
}

func decodeBoard(t *testing.T, resp *http.Response) []boardVenueGroup {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var groups []boardVenueGroup
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	return groups
}

// createVenue legt eine Halle an und hängt sie an ein Spiel. CreateGame kennt
// bewusst keinen Venue-Parameter — die Fixture-Signatur bleibt unangetastet.
func createVenue(t *testing.T, db *sql.DB, gameID int, name string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO venues (name, street, city, postal_code) VALUES (?, 'Teststr. 1', 'Stuttgart', '70173')`, name)
	if err != nil {
		t.Fatalf("insert venue: %v", err)
	}
	id, _ := res.LastInsertId()
	if gameID > 0 {
		if _, err := db.Exec(`UPDATE games SET venue_id=? WHERE id=?`, id, gameID); err != nil {
			t.Fatalf("set games.venue_id: %v", err)
		}
	}
	return int(id)
}

func findGroupByGame(groups []boardVenueGroup, gameID int) *boardVenueGroup {
	for i := range groups {
		if groups[i].GameID != nil && *groups[i].GameID == gameID {
			return &groups[i]
		}
	}
	return nil
}

// TestDutyBoard_VenueAufGruppe: die Gruppe eines Spiels mit gesetztem venue_id
// trägt den Hallennamen; ein Spiel ohne venue_id und eine game-lose Gruppe
// tragen keinen. Der Ort ist die Datengrundlage des Textfilters auf /dienste.
func TestDutyBoard_VenueAufGruppe(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Board-Team")
	dutyTypeID := createDutyType(t, db, "Verkauf", 2)

	gameMitVenue := testutil.CreateGame(t, db, seasonID, teamID, "2099-01-10")
	createVenue(t, db, gameMitVenue, "Scharnhauser Park Halle")
	createDutySlot(t, db, dutyTypeID, seasonID, teamID, gameMitVenue, "2099-01-10")

	gameOhneVenue := testutil.CreateGame(t, db, seasonID, teamID, "2099-01-11")
	createDutySlot(t, db, dutyTypeID, seasonID, teamID, gameOhneVenue, "2099-01-11")

	// Game-loser Handslot (game_id NULL) — hat strukturell keinen Ort.
	createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2099-01-12")

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testServer(t, h)

	groups := decodeBoard(t, testutil.Get(t, srv, "/api/duty-board", token))

	mit := findGroupByGame(groups, gameMitVenue)
	if mit == nil {
		t.Fatalf("Gruppe für Spiel %d nicht gefunden", gameMitVenue)
	}
	if mit.Venue != "Scharnhauser Park Halle" {
		t.Errorf("erwartet venue=%q, bekam %q", "Scharnhauser Park Halle", mit.Venue)
	}

	ohne := findGroupByGame(groups, gameOhneVenue)
	if ohne == nil {
		t.Fatalf("Gruppe für Spiel %d nicht gefunden", gameOhneVenue)
	}
	if ohne.Venue != "" {
		t.Errorf("Spiel ohne venue_id: erwartet leeren Ort, bekam %q", ohne.Venue)
	}

	var gameLos *boardVenueGroup
	for i := range groups {
		if groups[i].GameID == nil {
			gameLos = &groups[i]
		}
	}
	if gameLos == nil {
		t.Fatal("game-lose Gruppe nicht gefunden")
	}
	if gameLos.Venue != "" {
		t.Errorf("game-lose Gruppe: erwartet leeren Ort, bekam %q", gameLos.Venue)
	}
}

// TestDutyBoard_VenueOhneOmitEmptyImJSON: `omitempty` hält die Payload für
// Gruppen ohne Ort unverändert groß — das Feld darf dort gar nicht auftauchen.
func TestDutyBoard_VenueOmitEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Board-Team")
	dutyTypeID := createDutyType(t, db, "Verkauf", 2)
	createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2099-01-12")

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testServer(t, h)

	resp := testutil.Get(t, srv, "/api/duty-board", token)
	defer resp.Body.Close()
	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("keine Gruppe in der Antwort")
	}
	for _, g := range raw {
		if _, ok := g["venue"]; ok {
			t.Errorf("Gruppe ohne Ort trägt trotzdem ein venue-Feld: %v", g)
		}
	}
}

// TestDutyBoard_VenueAendertKeineSichtbarkeit friert Invariante 1 des Changes
// termin-textfilter ein: der zusätzliche LEFT JOIN auf venues darf für keine
// Persona Zeilen dupliziert oder weggefiltert haben. Geprüft wird die exakte
// Menge der sichtbaren Slot-IDs je Rolle gegen die fachlich erwartete Menge.
func TestDutyBoard_VenueAendertKeineSichtbarkeit(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	dutyTypeID := createDutyType(t, db, "Verkauf", 2)

	// Team A: Spiel mit Halle. Team B: Spiel ohne Halle — beide Zustände des
	// neuen JOINs kommen in derselben Abfrage vor.
	gameA := testutil.CreateGame(t, db, seasonID, teamA, "2099-02-01")
	createVenue(t, db, gameA, "Halle A")
	slotA := createDutySlot(t, db, dutyTypeID, seasonID, teamA, gameA, "2099-02-01")

	gameB := testutil.CreateGame(t, db, seasonID, teamB, "2099-02-02")
	slotB := createDutySlot(t, db, dutyTypeID, seasonID, teamB, gameB, "2099-02-02")

	srv := testServer(t, h)

	visibleSlots := func(userID int, role string, functions []string) []int {
		token := testutil.Token(t, userID, role, functions)
		groups := decodeBoard(t, testutil.Get(t, srv, "/api/duty-board", token))
		ids := []int{}
		for _, g := range groups {
			for _, s := range g.Slots {
				ids = append(ids, s.ID)
			}
		}
		sort.Ints(ids)
		return ids
	}

	equal := func(t *testing.T, label string, got, want []int) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: erwartet Slots %v, bekam %v", label, want, got)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: erwartet Slots %v, bekam %v", label, want, got)
			}
		}
	}

	// Spieler in Team A: sieht genau den Slot von Team A — einmal, nicht doppelt.
	spielerUser := testutil.CreateUser(t, db, "standard")
	spielerMember := testutil.CreateMember(t, db, spielerUser)
	addPlayerMembership(t, db, spielerMember, teamA, seasonID)
	equal(t, "Spieler Team A", visibleSlots(spielerUser, "standard", []string{"spieler"}), []int{slotA})

	// Elternteil eines Kindes in Team B: sieht genau den Slot von Team B.
	elternUser := testutil.CreateUser(t, db, "standard")
	kindUser := testutil.CreateUser(t, db, "standard")
	kindMember := testutil.CreateMember(t, db, kindUser)
	addPlayerMembership(t, db, kindMember, teamB, seasonID)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		elternUser, kindMember); err != nil {
		t.Fatalf("insert family_links: %v", err)
	}
	equal(t, "Elternteil Team B", visibleSlots(elternUser, "standard", nil), []int{slotB})

	// Trainer von Team A (nur Trainer, kein Spieler dort).
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`,
		kaderA, trainerMember); err != nil {
		t.Fatalf("insert kader_trainers: %v", err)
	}
	equal(t, "Trainer Team A", visibleSlots(trainerUser, "standard", []string{"trainer"}), []int{slotA})

	// Vorstand sieht beide Teams — und jeden Slot genau einmal.
	vorstandUser := testutil.CreateUser(t, db, "standard")
	want := []int{slotA, slotB}
	sort.Ints(want)
	equal(t, "Vorstand", visibleSlots(vorstandUser, "standard", []string{"vorstand"}), want)
}
