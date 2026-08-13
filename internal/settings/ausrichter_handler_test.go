package settings_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ausrichterServerSetup baut einen Test-Server mit allen fünf Ausrichter-
// Routen, gegated exakt wie in internal/app/router.go: beide GETs im
// Authenticated-Tier (jeder eingeloggte Nutzer), POST/PUT/DELETE im
// Vorstand-Tier (auth.RequireClubFunction("vorstand") — admin fällt dort
// automatisch durch, siehe bewirtungServerSetup).
func ausrichterServerSetup(t *testing.T) (srv *testHTTPServer, db *sql.DB, evHub *hub.EventHub, createUser func(role string) int) {
	t.Helper()
	database := testutil.NewDB(t)
	store := settings.NewStoreForTest(database, 0)
	evHub = hub.NewHub()
	handler := settings.NewHandler(database, store, evHub)

	srv = newTestHTTPServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(testutil.TestJWTSecret))
			r.Get("/api/ausrichter", handler.ListAusrichter)
			r.Get("/api/ausrichter/{id}/usage", handler.AusrichterUsage)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireClubFunction("vorstand"))
				r.Post("/api/ausrichter", handler.CreateAusrichter)
				r.Put("/api/ausrichter/{id}", handler.UpdateAusrichter)
				r.Delete("/api/ausrichter/{id}", handler.DeleteAusrichter)
			})
		})
	})
	createUser = func(role string) int { return testutil.CreateUser(t, database, role) }
	return srv, database, evHub, createUser
}

// --- GET /api/ausrichter --------------------------------------------------

func TestAusrichterHandler_List_ReturnsDefaultEntry(t *testing.T) {
	srv, _, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv.raw, "/api/ausrichter", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body struct {
		Items []settings.Ausrichter `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Migration 048 seedet genau eine Default-Zeile — die muss in jeder frischen
	// Test-DB in der Liste auftauchen.
	if len(body.Items) != 1 || !body.Items[0].IsDefault {
		t.Fatalf("erwartet genau einen Default-Eintrag, bekam %+v", body.Items)
	}
}

func TestAusrichterHandler_List_Unauthenticated_Returns401(t *testing.T) {
	srv, _, _, _ := ausrichterServerSetup(t)
	defer srv.Close()

	res := testutil.Get(t, srv.raw, "/api/ausrichter", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", res.StatusCode)
	}
}

// TestAusrichterHandler_List_IncludeInactive_StandardUser_BleibtVerborgen:
// ?include_inactive=1 wirkt nur für Vorstand/Admin (Vorbild stammvereine) —
// ein Standard-Nutzer sieht deaktivierte Einträge auch mit dem Query-Param nicht.
func TestAusrichterHandler_List_IncludeInactive_StandardUser_BleibtVerborgen(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	created, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	aktiv := false
	if _, err := settings.UpdateAusrichter(context.Background(), db, created.ID,
		settings.AusrichterUpdate{Aktiv: &aktiv}); err != nil {
		t.Fatalf("deaktivieren: %v", err)
	}

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv.raw, "/api/ausrichter?include_inactive=1", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body struct {
		Items []settings.Ausrichter `json:"items"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	for _, item := range body.Items {
		if item.ID == created.ID {
			t.Fatalf("Standard-Nutzer darf deaktivierten Ausrichter %d nicht sehen, auch nicht mit include_inactive=1", created.ID)
		}
	}
}

// --- POST /api/ausrichter --------------------------------------------------

func TestAusrichterHandler_Create_AsVorstand_Returns201_AndBroadcasts(t *testing.T) {
	srv, db, evHub, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	sub := evHub.Subscribe()
	defer evHub.Unsubscribe(sub)

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Post(t, srv.raw, "/api/ausrichter", token, map[string]any{"name": "TV Ötlingen"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekam %d", res.StatusCode)
	}
	var created settings.Ausrichter
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Name != "TV Ötlingen" || !created.Aktiv || created.IsDefault {
		t.Errorf("unerwarteter Eintrag: %+v", created)
	}
	if countAusrichter(t, db) != 2 {
		t.Errorf("erwartet 2 Zeilen (Default + neuer Eintrag), bekam %d", countAusrichter(t, db))
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

func TestAusrichterHandler_Create_Namensdublette_Returns409_SchreibtNichts(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Post(t, srv.raw, "/api/ausrichter", token, map[string]any{"name": "TV Ötlingen"})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Vorbedingung: erwartet 201, bekam %d", res.StatusCode)
	}
	before := countAusrichter(t, db)

	res = testutil.Post(t, srv.raw, "/api/ausrichter", token, map[string]any{"name": "TV Ötlingen"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekam %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "duplicate_name" {
		t.Errorf("erwartet error=duplicate_name, bekam %q", errBody.Error)
	}
	if got := countAusrichter(t, db); got != before {
		t.Errorf("Dublette darf keine zweite Zeile schreiben: vorher %d, nachher %d", before, got)
	}
}

func TestAusrichterHandler_Create_AsNonVorstand_Returns403(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"kassierer"})

	before := countAusrichter(t, db)
	res := testutil.Post(t, srv.raw, "/api/ausrichter", token, map[string]any{"name": "TV Ötlingen"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}
	if got := countAusrichter(t, db); got != before {
		t.Errorf("403 darf nichts schreiben: vorher %d, nachher %d", before, got)
	}
}

func TestAusrichterHandler_Create_LeererName_Returns400(t *testing.T) {
	srv, _, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Post(t, srv.raw, "/api/ausrichter", token, map[string]any{"name": "   "})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("erwartet 400, bekam %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "empty_name" {
		t.Errorf("erwartet error=empty_name, bekam %q", errBody.Error)
	}
}

// --- PUT /api/ausrichter/{id} ----------------------------------------------

func TestAusrichterHandler_Update_DefaultWechsel_GenauEineZeile(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	created, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/ausrichter/"+itoa(created.ID), token,
		map[string]any{"is_default": true})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var updated settings.Ausrichter
	_ = json.NewDecoder(res.Body).Decode(&updated)
	if !updated.IsDefault {
		t.Error("erwartet is_default=true in der Response")
	}
	if got := countDefaults(t, db); got != 1 {
		t.Errorf("erwartet genau eine Default-Zeile, bekam %d", got)
	}
	if defaultAusrichterID(t, db) != created.ID {
		t.Errorf("erwartet neuen Default %d, bekam %d", created.ID, defaultAusrichterID(t, db))
	}
}

// TestAusrichterHandler_Update_DefaultAbwaehlen_Returns409_NichtsGeaendert:
// is_default:false auf dem Default ist gesperrt, weil sonst kein Default übrig bliebe.
func TestAusrichterHandler_Update_DefaultAbwaehlen_Returns409_NichtsGeaendert(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	defaultID := defaultAusrichterID(t, db)
	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/ausrichter/"+itoa(defaultID), token,
		map[string]any{"is_default": false})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekam %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "default_required" {
		t.Errorf("erwartet error=default_required, bekam %q", errBody.Error)
	}
	if got := countDefaults(t, db); got != 1 {
		t.Errorf("erwartet unverändert eine Default-Zeile, bekam %d", got)
	}
	if defaultAusrichterID(t, db) != defaultID {
		t.Error("der Default darf sich bei abgelehntem Request nicht ändern")
	}
}

func TestAusrichterHandler_Update_AsNonVorstand_Returns403(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	created, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"kassierer"})

	res := testutil.Put(t, srv.raw, "/api/ausrichter/"+itoa(created.ID), token,
		map[string]any{"name": "Umbenannt"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}
	got, err := settings.GetAusrichter(context.Background(), db, created.ID)
	if err != nil {
		t.Fatalf("GetAusrichter: %v", err)
	}
	if got.Name != "TV Ötlingen" {
		t.Errorf("Name darf bei 403 nicht geändert werden, bekam %q", got.Name)
	}
}

func TestAusrichterHandler_Update_Unbekannt_Returns404(t *testing.T) {
	srv, _, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Put(t, srv.raw, "/api/ausrichter/99999", token, map[string]any{"name": "X"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("erwartet 404, bekam %d", res.StatusCode)
	}
}

// --- DELETE /api/ausrichter/{id} -------------------------------------------

func TestAusrichterHandler_Delete_Returns200_KaskadeLaeuft(t *testing.T) {
	srv, db, evHub, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	sub := evHub.Subscribe()
	defer evHub.Unsubscribe(sub)

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	victim, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, victim.ID)
	templateID := createTemplate(t, db, "Heimspiel Standard")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kuchen", 1)
	itemID := createTemplateItem(t, db, templateID, dutyTypeID, victim.ID)

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Delete(t, srv.raw, "/api/ausrichter/"+itoa(victim.ID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}

	var ausrichterID sql.NullInt64
	if err := db.QueryRow(`SELECT ausrichter_id FROM spieltag_ausrichter WHERE date = ? AND season_id = ?`,
		"2026-09-14", seasonID).Scan(&ausrichterID); err != nil {
		t.Fatalf("spieltag_ausrichter lesen: %v", err)
	}
	if ausrichterID.Valid {
		t.Error("der Spieltag muss nach dem Löschen ausrichter_id=NULL tragen")
	}
	var itemCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE id = ?`, itemID).Scan(&itemCount); err != nil {
		t.Fatalf("item zählen: %v", err)
	}
	if itemCount != 0 {
		t.Error("die gebundene Vorlagen-Zeile muss mitgelöscht sein")
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

func TestAusrichterHandler_Delete_Default_Returns409_NichtsGeaendert(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	defaultID := defaultAusrichterID(t, db)
	before := countAusrichter(t, db)

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Delete(t, srv.raw, "/api/ausrichter/"+itoa(defaultID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekam %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "default_ausrichter_undeletable" {
		t.Errorf("erwartet error=default_ausrichter_undeletable, bekam %q", errBody.Error)
	}
	if got := countAusrichter(t, db); got != before {
		t.Errorf("abgelehntes Löschen darf keine Zeile entfernen: vorher %d, nachher %d", before, got)
	}
	if _, err := settings.GetAusrichter(context.Background(), db, defaultID); err != nil {
		t.Errorf("der Default muss weiterhin existieren: %v", err)
	}
}

func TestAusrichterHandler_Delete_AsNonVorstand_Returns403(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	created, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", []string{"kassierer"})

	res := testutil.Delete(t, srv.raw, "/api/ausrichter/"+itoa(created.ID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}
	if _, err := settings.GetAusrichter(context.Background(), db, created.ID); err != nil {
		t.Errorf("der Eintrag muss bei 403 weiter existieren: %v", err)
	}
}

// --- GET /api/ausrichter/{id}/usage -----------------------------------------

func TestAusrichterHandler_Usage_BenenntBeideReferenzen(t *testing.T) {
	srv, db, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	victim, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, victim.ID)
	templateID := createTemplate(t, db, "Heimspiel Standard")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kuchen", 1)
	createTemplateItem(t, db, templateID, dutyTypeID, victim.ID)

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv.raw, "/api/ausrichter/"+itoa(victim.ID)+"/usage", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var report settings.AusrichterUsageReport
	if err := json.NewDecoder(res.Body).Decode(&report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(report.GameDays) != 1 || report.GameDays[0].Date != "2026-09-14" {
		t.Errorf("erwartet einen Spieltag 2026-09-14, bekam %+v", report.GameDays)
	}
	if len(report.TemplateItems) != 1 || report.TemplateItems[0].DutyTypeName != "Kuchen" {
		t.Errorf("erwartet ein Vorlagen-Item 'Kuchen', bekam %+v", report.TemplateItems)
	}
}

func TestAusrichterHandler_Usage_Unbekannt_Returns404(t *testing.T) {
	srv, _, _, createUser := ausrichterServerSetup(t)
	defer srv.Close()

	userID := createUser("standard")
	token := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv.raw, "/api/ausrichter/99999/usage", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("erwartet 404, bekam %d", res.StatusCode)
	}
}

func TestAusrichterHandler_Usage_Unauthenticated_Returns401(t *testing.T) {
	srv, db, _, _ := ausrichterServerSetup(t)
	defer srv.Close()

	defaultID := defaultAusrichterID(t, db)
	res := testutil.Get(t, srv.raw, "/api/ausrichter/"+itoa(defaultID)+"/usage", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", res.StatusCode)
	}
}

// itoa ist strconv.Itoa unter kurzem Namen — hier nur für lesbare URL-Pfade
// in den Testfällen oben verwendet.
func itoa(id int) string {
	return strconv.Itoa(id)
}
