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
	if body["max_per_team"] != float64(1) {
		t.Errorf("erwartet max_per_team=1 (Migration-Default), bekam %v", body["max_per_team"])
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

// settingValue liest einen system_settings-Wert direkt aus der DB.
func settingValue(t *testing.T, db *sql.DB, key string) string {
	t.Helper()
	var value string
	if err := db.QueryRowContext(context.Background(),
		`SELECT value FROM system_settings WHERE key=?`, key).Scan(&value); err != nil {
		t.Fatalf("query %s: %v", key, err)
	}
	return value
}

// TestBewirtung_Put_MaxPerTeam_PersistsAndLeavesVerhaeltnis: der Cap lässt sich
// allein setzen — ein im Body fehlendes Feld bleibt unverändert (Pointer-Semantik).
func TestBewirtung_Put_MaxPerTeam_PersistsAndLeavesVerhaeltnis(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	// Verhältnis vorbelegen, damit "unverändert" eine echte Aussage ist.
	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"verhaeltnis": 0.5})
	res.Body.Close()

	res = testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"max_per_team": 3})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["max_per_team"] != float64(3) {
		t.Errorf("erwartet max_per_team=3 in Response, bekam %v", body["max_per_team"])
	}
	if body["verhaeltnis"] != 0.5 {
		t.Errorf("Response soll das unveränderte Verhältnis 0.5 spiegeln, bekam %v", body["verhaeltnis"])
	}

	if got := settingValue(t, db, "bewirtung_max_per_team"); got != "3" {
		t.Errorf("DB max_per_team: erwartet '3', bekam %q", got)
	}
	if got := settingValue(t, db, "bewirtung_verhaeltnis"); got != "0.5" {
		t.Errorf("DB verhaeltnis sollte unverändert '0.5' sein, bekam %q", got)
	}
}

// TestBewirtung_Put_MaxPerTeamNull_Returns400_NichtsPersistiert: ein ungültiger Cap
// darf auch das im selben Request mitgesendete, für sich gültige Verhältnis NICHT
// durchschreiben — sonst hinterlässt ein 400 einen halb geschriebenen Zustand.
func TestBewirtung_Put_MaxPerTeamNull_Returns400_NichtsPersistiert(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token,
		map[string]any{"verhaeltnis": 2.0, "max_per_team": 0})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekam %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "invalid_max_per_team" {
		t.Errorf("erwartet error=invalid_max_per_team, bekam %q", errBody.Error)
	}

	if got := settingValue(t, db, "bewirtung_max_per_team"); got != "1" {
		t.Errorf("max_per_team sollte unverändert '1' sein, bekam %q", got)
	}
	if got := settingValue(t, db, "bewirtung_verhaeltnis"); got != "1" {
		t.Errorf("verhaeltnis darf bei abgelehntem Cap nicht geschrieben werden, bekam %q", got)
	}
}

// TestBewirtung_Put_MaxPerTeam_AsNonVorstand_Returns403_Unchanged: das Gating gilt
// für den Cap genauso wie für das Verhältnis.
func TestBewirtung_Put_MaxPerTeam_AsNonVorstand_Returns403_Unchanged(t *testing.T) {
	srv, db, _, createUser := bewirtungServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"kassierer"})

	res := testutil.Put(t, srv.raw, "/api/settings/bewirtung", token, map[string]any{"max_per_team": 5})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}
	if got := settingValue(t, db, "bewirtung_max_per_team"); got != "1" {
		t.Errorf("Wert sollte unverändert Default '1' sein, bekam %q", got)
	}
}

// TestBewirtung_GetMaxPerTeam_FehlendeRow_LiefertDefault: fehlt die Row (DB ohne
// Migration 046), liefert der Getter den fachlichen Default 1 statt eines Fehlers —
// die Rotation läuft dann wie vor der Einstellung weiter.
func TestBewirtung_GetMaxPerTeam_FehlendeRow_LiefertDefault(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := db.Exec(`DELETE FROM system_settings WHERE key='bewirtung_max_per_team'`); err != nil {
		t.Fatalf("delete row: %v", err)
	}
	got, err := settings.GetBewirtungMaxPerTeam(context.Background(), db)
	if err != nil {
		t.Fatalf("GetBewirtungMaxPerTeam: %v", err)
	}
	if got != 1 {
		t.Errorf("erwartet Default 1, bekam %d", got)
	}
}

// TestBewirtung_SetMaxPerTeam_NichtPositiv_SchreibtNichts: der Store-Guard hält
// auch dann, wenn er ohne den Handler aufgerufen wird.
func TestBewirtung_SetMaxPerTeam_NichtPositiv_SchreibtNichts(t *testing.T) {
	db := testutil.NewDB(t)
	for _, v := range []int{0, -2} {
		if err := settings.SetBewirtungMaxPerTeam(context.Background(), db, v, 0); err != settings.ErrInvalidMaxPerTeam {
			t.Errorf("Wert %d: erwartet ErrInvalidMaxPerTeam, bekam %v", v, err)
		}
	}
	if got := settingValue(t, db, "bewirtung_max_per_team"); got != "1" {
		t.Errorf("Wert sollte unverändert '1' sein, bekam %q", got)
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
