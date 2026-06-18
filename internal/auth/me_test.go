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
