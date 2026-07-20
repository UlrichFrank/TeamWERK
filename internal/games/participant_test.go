package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// am_i_participant: Spieler im Stammkader ohne Response bei Default=none
// SOLL true zurückliefern, my_rsvp bleibt null (Spec: game-rsvp →
// „Teilnehmer sehen RSVP-Buttons unabhängig von Response").
func TestListMyGames_AmIParticipant_RegularKader_DefaultNone(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/games/my", h.ListMyGames)
	})

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/games/my", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	var found map[string]any
	for _, it := range items {
		if int(it["id"].(float64)) == gameID {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("game %d not returned", gameID)
	}
	if p, _ := found["am_i_participant"].(bool); !p {
		t.Errorf("am_i_participant: got %v, want true", found["am_i_participant"])
	}
	if found["my_rsvp"] != nil {
		t.Errorf("my_rsvp: got %v, want nil (Default=none, keine Response)", found["my_rsvp"])
	}
}

// Erweiterter Kader ohne Response bei Default=none → am_i_participant=true.
func TestListMyGames_AmIParticipant_ExtendedKader(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addExtendedKaderMember(t, db, kaderID, mID)

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/games/my", h.ListMyGames)
	})

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/games/my", token)
	defer res.Body.Close()
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	for _, it := range items {
		if int(it["id"].(float64)) == gameID {
			if p, _ := it["am_i_participant"].(bool); !p {
				t.Errorf("am_i_participant: got %v, want true (extended kader)", it["am_i_participant"])
			}
			return
		}
	}
	t.Fatalf("game %d not returned", gameID)
}

// Trainer sieht am_i_participant=true.
func TestListMyGames_AmIParticipant_Trainer(t *testing.T) {
	db, gameID, teamID, seasonID, _ := setupCutoffGame(t)
	trainerUID := makeTrainer(t, db, teamID, seasonID)

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/games/my", h.ListMyGames)
	})

	token := testutil.Token(t, trainerUID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/games/my", token)
	defer res.Body.Close()
	var items []map[string]any
	json.NewDecoder(res.Body).Decode(&items)
	for _, it := range items {
		if int(it["id"].(float64)) == gameID {
			if p, _ := it["am_i_participant"].(bool); !p {
				t.Errorf("am_i_participant für Trainer: got %v, want true", it["am_i_participant"])
			}
			return
		}
	}
	t.Fatalf("game %d not returned to trainer", gameID)
}

// GetGame liefert am_i_participant analog zum List-Endpoint.
func TestGetGame_AmIParticipant_RegularKader(t *testing.T) {
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	uID := testutil.CreateUser(t, db, "standard")
	mID := testutil.CreateMember(t, db, uID)
	addKaderMember(t, db, kaderID, mID)

	h := games.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/games/{id}", h.GetGame)
	})

	token := testutil.Token(t, uID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d", gameID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	g, _ := body["game"].(map[string]any)
	if p, _ := g["am_i_participant"].(bool); !p {
		t.Errorf("am_i_participant: got %v, want true", g["am_i_participant"])
	}
}
