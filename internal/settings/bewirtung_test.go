package settings_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// bewirtungServerSetup baut einen Test-Server mit den zwei
// bewirtung-Routen, gegated exakt wie in internal/app/router.go: GET im
// Authenticated-Tier (jeder eingeloggte Nutzer), PUT im Vorstand-Tier
// (auth.RequireClubFunction("vorstand") — admin fällt dort automatisch durch).
func bewirtungServerSetup(t *testing.T) (srv *testHTTPServer, db *sql.DB, evHub *hub.EventHub, createUser func(role string) int) {
	t.Helper()
	database := testutil.NewDB(t)
	store := settings.NewStoreForTest(database, 0)
	evHub = hub.NewHub()
	handler := settings.NewHandler(database, store, evHub)

	srv = newTestHTTPServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(testutil.TestJWTSecret))
			r.Get("/api/settings/bewirtung", handler.GetBewirtung)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireClubFunction("vorstand"))
				r.Put("/api/settings/bewirtung", handler.SetBewirtung)
			})
		})
	})
	createUser = func(role string) int { return testutil.CreateUser(t, database, role) }
	return srv, database, evHub, createUser
}

func TestBewirtung_Get_ReturnsDefault(t *testing.T) {
	srv, _, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv.raw, "/api/settings/bewirtung", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["verhaeltnis"] != float64(1) {
		t.Errorf("erwartet verhaeltnis=1 (Migration-Default), bekam %v", body["verhaeltnis"])
	}
}

func TestBewirtung_Get_Unauthenticated_Returns401(t *testing.T) {
	srv, _, _, _ := bewirtungServerSetup(t)
	defer srv.Close()

	res := testutil.Get(t, srv.raw, "/api/settings/bewirtung", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", res.StatusCode)
	}
}

func TestBewirtung_Put_AsVorstand_PersistsAndBroadcasts(t *testing.T) {
	srv, db, evHub, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	sub := evHub.Subscribe()
	defer evHub.Unsubscribe(sub)

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": 0.5})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["verhaeltnis"] != 0.5 {
		t.Errorf("erwartet verhaeltnis=0.5 in Response, bekam %v", body["verhaeltnis"])
	}

	var value string
	if err := db.QueryRowContext(context.Background(),
		`SELECT value FROM system_settings WHERE key='bewirtung_verhaeltnis'`).Scan(&value); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if value != "0.5" {
		t.Errorf("DB value: erwartet '0.5', bekam %q", value)
	}

	select {
	case ev := <-sub:
		if ev != "settings-changed" {
			t.Errorf("erwartet 'settings-changed', bekam %q", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("erwartet Broadcast 'settings-changed', kam nicht an")
	}
}

func TestBewirtung_Put_AsAdmin_Succeeds(t *testing.T) {
	srv, _, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	adminID := createUser("admin")
	token := testutil.Token(t, adminID, "admin", nil)

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": 1.5})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200 (admin bypässt RequireClubFunction), bekam %d", res.StatusCode)
	}
}

func TestBewirtung_Put_AsNonVorstand_Returns403_Unchanged(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"kassierer"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": 2.0})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}

	var value string
	if err := db.QueryRowContext(context.Background(),
		`SELECT value FROM system_settings WHERE key='bewirtung_verhaeltnis'`).Scan(&value); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if value != "1" {
		t.Errorf("Wert sollte unverändert Default '1' sein, bekam %q", value)
	}
}

func TestBewirtung_Put_NegativeValue_Returns400_Unchanged(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": -1})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("erwartet 400, bekam %d", res.StatusCode)
	}

	var value string
	if err := db.QueryRowContext(context.Background(),
		`SELECT value FROM system_settings WHERE key='bewirtung_verhaeltnis'`).Scan(&value); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if value != "1" {
		t.Errorf("Wert sollte unverändert Default '1' sein, bekam %q", value)
	}
}

func TestBewirtung_Put_NonNumericValue_Returns400_Unchanged(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": "abc"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("erwartet 400, bekam %d", res.StatusCode)
	}

	var value string
	if err := db.QueryRowContext(context.Background(),
		`SELECT value FROM system_settings WHERE key='bewirtung_verhaeltnis'`).Scan(&value); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if value != "1" {
		t.Errorf("Wert sollte unverändert Default '1' sein, bekam %q", value)
	}
}
