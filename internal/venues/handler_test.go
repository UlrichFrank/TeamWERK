package venues_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

// TestVenues_ETag304 — GET /api/venues liefert ETag + private, no-cache;
// unveränderter Bestand revalidiert per 304, nach einer Mutation ändert sich
// der ETag. Fehlerfall (403 ohne Vereinsfunktion) bleibt unverändert.
func TestVenues_ETag304(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO venues (name, street, city, postal_code) VALUES ('Sporthalle Ost', 'Teststr. 1', 'Stuttgart', '70000')`,
	); err != nil {
		t.Fatalf("seed venue: %v", err)
	}
	h := venues.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/venues", h.List)
		})
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/venues", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", res.StatusCode)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", cc)
	}

	// Unverändert → 304.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/venues", nil)
	req.Header.Set("Authorization", tok)
	req.Header.Set("If-None-Match", etag)
	res304, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}

	// Mutation → neuer ETag, voller Body.
	if _, err := database.Exec(`UPDATE venues SET name='Sporthalle West'`); err != nil {
		t.Fatalf("update venue: %v", err)
	}
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/venues", nil)
	req2.Header.Set("Authorization", tok)
	req2.Header.Set("If-None-Match", etag)
	resAfter, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("GET nach Mutation: %v", err)
	}
	defer resAfter.Body.Close()
	if resAfter.StatusCode != http.StatusOK {
		t.Fatalf("nach Mutation: status %d, want 200", resAfter.StatusCode)
	}
	if newTag := resAfter.Header.Get("ETag"); newTag == etag {
		t.Errorf("ETag nach Mutation unverändert: %q", newTag)
	}

	// Fehlerfall unverändert: ohne berechtigte Vereinsfunktion → 403.
	spieler := testutil.Token(t, userID, "standard", []string{"spieler"})
	resForbidden := testutil.Get(t, srv, "/api/venues", spieler)
	defer resForbidden.Body.Close()
	if resForbidden.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", resForbidden.StatusCode)
	}
}

// mutationServer verdrahtet die schreibenden Venue-Routen hinter der gleichen
// Vereinsfunktions-Gruppe wie der Router (Authz/403 ist bereits über
// permissions/matrix_test.go abgedeckt → hier nur Fach-/Fehlerpfade).
func mutationServer(t *testing.T, h *venues.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Delete("/api/venues/{id}", h.Delete)
			r.Delete("/api/venues", h.DeleteAll)
			r.Post("/api/venues/import", h.Import)
		})
	})
}

func vorstandToken(t *testing.T, database *sql.DB) string {
	t.Helper()
	return testutil.Token(t, testutil.CreateUser(t, database, "standard"), "standard", []string{"vorstand"})
}

// TestVenues_Delete_Success_RemovesRow — DELETE /api/venues/{id} löscht die Zeile
// und liefert 204; die Zeile ist danach nicht mehr in der DB.
func TestVenues_Delete_Success_RemovesRow(t *testing.T) {
	database := testutil.NewDB(t)
	res, err := database.Exec(
		`INSERT INTO venues (name, street, city, postal_code) VALUES ('Halle A', 'Teststr. 1', 'Stuttgart', '70000')`)
	if err != nil {
		t.Fatalf("seed venue: %v", err)
	}
	id, _ := res.LastInsertId()

	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	del := testutil.Delete(t, srv, "/api/venues/"+strconv.FormatInt(id, 10), tok)
	defer del.Body.Close()
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("Delete: status %d, want 204", del.StatusCode)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM venues WHERE id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("Zeile nach Delete noch vorhanden: count=%d, want 0", count)
	}
}

// TestVenues_Delete_NotFound — DELETE einer unbekannten ID → 404.
func TestVenues_Delete_NotFound(t *testing.T) {
	database := testutil.NewDB(t)
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	res := testutil.Delete(t, srv, "/api/venues/9999", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("Delete unbekannte ID: status %d, want 404", res.StatusCode)
	}
}

// TestVenues_Delete_InvalidID — nicht-numerische ID → 400.
func TestVenues_Delete_InvalidID(t *testing.T) {
	database := testutil.NewDB(t)
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	res := testutil.Delete(t, srv, "/api/venues/abc", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Delete ungültige ID: status %d, want 400", res.StatusCode)
	}
}

// TestVenues_DeleteAll_KeepsHomeVenue — DELETE /api/venues löscht nur Venues mit
// is_home_venue=0; die Heimspielstätte bleibt erhalten.
func TestVenues_DeleteAll_KeepsHomeVenue(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO venues (name, street, city, postal_code, is_home_venue) VALUES
		 ('Heimhalle', 'Heimstr. 1', 'Stuttgart', '70000', 1),
		 ('Gasthalle', 'Gaststr. 2', 'Esslingen', '73728', 0)`); err != nil {
		t.Fatalf("seed venues: %v", err)
	}
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	res := testutil.Delete(t, srv, "/api/venues", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DeleteAll: status %d, want 204", res.StatusCode)
	}

	var count, homeFlag int
	if err := database.QueryRow(`SELECT COUNT(*) FROM venues`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("nach DeleteAll: count=%d, want 1", count)
	}
	if err := database.QueryRow(`SELECT is_home_venue FROM venues`).Scan(&homeFlag); err != nil {
		t.Fatalf("home flag: %v", err)
	}
	if homeFlag != 1 {
		t.Errorf("verbleibende Venue ist nicht die Heimspielstätte: is_home_venue=%d, want 1", homeFlag)
	}
}

// TestVenues_Import_MissingFileField — Multipart ohne "file"-Feld → 400.
func TestVenues_Import_MissingFileField(t *testing.T) {
	database := testutil.NewDB(t)
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("something", "value"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/venues/import", &buf)
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Import ohne file-Feld: status %d, want 400", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "missing file") {
		t.Errorf("Body = %q, want enthält \"missing file\"", string(body))
	}
}

// TestVenues_Import_HeaderRowNotFound — CSV ohne "Name"-Header → 400.
func TestVenues_Import_HeaderRowNotFound(t *testing.T) {
	database := testutil.NewDB(t)
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	csv := "Foo;Bar\nx;y\n"
	res := testutil.PostMultipart(t, srv, "/api/venues/import", tok, "file", "venues.csv", []byte(csv))
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Import ohne Header: status %d, want 400", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "header row not found") {
		t.Errorf("Body = %q, want enthält \"header row not found\"", string(body))
	}
}

// TestVenues_Import_InsertsAndUpserts — CSV mit 3 Preamble-Zeilen + Header +
// Datenzeilen: eine Zeile matcht (Update), eine neu (Insert), eine ohne Name
// (Skip). Prüft importResult-Zähler und die resultierenden DB-Zeilen.
func TestVenues_Import_InsertsAndUpserts(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO venues (name, street, city, postal_code, note) VALUES ('Halle A', 'Alte Str. 1', 'Stuttgart', '70000', '')`); err != nil {
		t.Fatalf("seed venue: %v", err)
	}
	h := venues.NewHandler(database, hub.NewHub())
	srv := mutationServer(t, h)
	tok := vorstandToken(t, database)

	// Spaltenlayout (handler.go:274-300): Name=0, Straße=2, PLZ=3, Ort=4, Notiz=5.
	// 3 Preamble-Zeilen, dann Header (Zelle 0 == "Name"), dann Datenzeilen.
	csv := strings.Join([]string{
		"Vereinsexport Spielstätten",
		"Erstellt am 2026-07-18",
		"Quelle: Verband",
		"Name,Kurz,Straße,PLZ,Ort,Notiz",
		"Halle A,HA,Neue Str. 5,70199,Stuttgart,Aktualisierte Notiz",
		"Halle B,HB,Bergstr. 2,73728,Esslingen,Neue Halle",
		",,,,,",
		"",
	}, "\n")

	res := testutil.PostMultipart(t, srv, "/api/venues/import", tok, "file", "venues.csv", []byte(csv))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Import: status %d, want 200 (body=%q)", res.StatusCode, string(body))
	}

	var result struct {
		Imported int `json:"imported"`
		Updated  int `json:"updated"`
		Skipped  int `json:"skipped"`
		Errors   []struct {
			Line   int    `json:"line"`
			Reason string `json:"reason"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported=%d, want 1", result.Imported)
	}
	if result.Updated != 1 {
		t.Errorf("Updated=%d, want 1", result.Updated)
	}
	if result.Skipped < 1 {
		t.Errorf("Skipped=%d, want >=1", result.Skipped)
	}
	var foundKeinName bool
	for _, e := range result.Errors {
		if strings.Contains(e.Reason, "kein Name") {
			foundKeinName = true
		}
	}
	if !foundKeinName {
		t.Errorf("Errors enthält kein \"kein Name\": %+v", result.Errors)
	}

	// Neue Venue eingefügt, country default 'DE'.
	var country string
	if err := database.QueryRow(
		`SELECT country FROM venues WHERE name = 'Halle B' AND city = 'Esslingen'`).Scan(&country); err != nil {
		t.Fatalf("neue Venue nicht gefunden: %v", err)
	}
	if country != "DE" {
		t.Errorf("neue Venue country=%q, want DE", country)
	}

	// Bestehende Venue aktualisiert (street/note; name/city bleiben Match-Key).
	var street, note string
	if err := database.QueryRow(
		`SELECT street, note FROM venues WHERE name = 'Halle A' AND city = 'Stuttgart'`).Scan(&street, &note); err != nil {
		t.Fatalf("bestehende Venue nicht gefunden: %v", err)
	}
	if street != "Neue Str. 5" {
		t.Errorf("street=%q, want \"Neue Str. 5\"", street)
	}
	if note != "Aktualisierte Notiz" {
		t.Errorf("note=%q, want \"Aktualisierte Notiz\"", note)
	}
}
