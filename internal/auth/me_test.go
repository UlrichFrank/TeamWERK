package auth_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestGetMe_VorstandCapabilities(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, "/api/me", token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		User struct {
			ID   int    `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
		Capabilities []string `json:"capabilities"`
		Nav          []struct {
			Route string `json:"route"`
		} `json:"nav"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !slices.Contains(body.Capabilities, "manage_members") {
		t.Error("vorstand should have manage_members capability")
	}

	hasMembers := false
	for _, n := range body.Nav {
		if n.Route == "/mitglieder" {
			hasMembers = true
		}
	}
	if !hasMembers {
		t.Error("vorstand should have /mitglieder in nav")
	}
}

func TestGetMe_SpielerNoManageMembers(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/me", token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		Capabilities []string `json:"capabilities"`
		Nav          []struct {
			Route string `json:"route"`
		} `json:"nav"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if slices.Contains(body.Capabilities, "manage_members") {
		t.Error("spieler should not have manage_members capability")
	}

	for _, n := range body.Nav {
		if n.Route == "/mitglieder" {
			t.Error("spieler should not have /mitglieder in nav")
		}
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Get(t, srv, "/api/me", "")
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

// Das Stummschalt-Recht steuert im Frontend die Sichtbarkeit des Häkchens
// „Ohne Benachrichtigung löschen". Es muss über /api/me ausgeliefert werden —
// und zwar nur an Vorstand und Admin, nicht an alle, die löschen dürfen.
func TestGetMe_SuppressEventNotificationCapability(t *testing.T) {
	cases := []struct {
		name          string
		role          string
		clubFunctions []string
		want          bool
	}{
		{name: "vorstand", role: "standard", clubFunctions: []string{"vorstand"}, want: true},
		{name: "admin", role: "admin", clubFunctions: nil, want: true},
		{name: "trainer", role: "standard", clubFunctions: []string{"trainer"}, want: false},
		{name: "sportliche_leitung", role: "standard", clubFunctions: []string{"sportliche_leitung"}, want: false},
		{name: "kassierer", role: "standard", clubFunctions: []string{"kassierer"}, want: false},
		{name: "spieler", role: "standard", clubFunctions: []string{"spieler"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			userID := testutil.CreateUser(t, db, tc.role)
			srv := newAuthServer(t, db)

			token := testutil.Token(t, userID, tc.role, tc.clubFunctions)
			res := testutil.Get(t, srv, "/api/me", token)
			defer res.Body.Close()

			var body struct {
				Capabilities []string `json:"capabilities"`
			}
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			got := slices.Contains(body.Capabilities, "suppress_event_notification")
			if got != tc.want {
				t.Errorf("suppress_event_notification = %v, want %v", got, tc.want)
			}
		})
	}
}
