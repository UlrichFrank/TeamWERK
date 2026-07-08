package notifications_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestVapidKey_CacheControlImmutable — GET /api/push/vapid-public-key trägt
// einen Cache-Control-Header mit immutable + ETag; der Body enthält den
// konfigurierten VAPID-Public-Key. Revalidierung per If-None-Match → 304.
func TestVapidKey_CacheControlImmutable(t *testing.T) {
	database := testutil.NewDB(t)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VAPIDPublicKey: "test-vapid-public-key"}
	h := notifications.NewHandler(database, cfg)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/push/vapid-public-key", h.GetVAPIDPublicKey)
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/push/vapid-public-key", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable-Direktive", cc)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Errorf("kein ETag gesetzt")
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["publicKey"] != "test-vapid-public-key" {
		t.Errorf("publicKey = %q, want konfigurierter Key", body["publicKey"])
	}

	// Revalidierung: If-None-Match → 304, leerer Body.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/push/vapid-public-key", nil)
	req.Header.Set("Authorization", tok)
	req.Header.Set("If-None-Match", etag)
	res2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusNotModified {
		t.Errorf("revalidierter Abruf: status %d, want 304", res2.StatusCode)
	}
}

// prefsServer verdrahtet die notification-preferences-Routen für die Tests.
func prefsServer(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret}
	h := notifications.NewHandler(database, cfg)
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/profile/notification-preferences", h.GetNotificationPreferences)
		r.Put("/api/profile/notification-preferences", h.UpdateNotificationPreferences)
	})
}

// TestUpdatePreferences_ChatPersists — Regression Defekt 1: die Kategorie 'chat'
// darf gespeichert werden (früher 500 am DB-CHECK). PUT → 204, Zeile persistiert.
func TestUpdatePreferences_ChatPersists(t *testing.T) {
	database := testutil.NewDB(t)
	srv := prefsServer(t, database)
	uid := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, uid, "standard", nil)

	body := map[string]map[string]bool{"chat": {"push": false, "email": false}}
	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/notification-preferences", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	var push int
	err := database.QueryRow(
		`SELECT push_enabled FROM notification_preferences WHERE user_id = ? AND category = 'chat'`, uid,
	).Scan(&push)
	if err != nil {
		t.Fatalf("chat-Präferenz nicht persistiert: %v", err)
	}
	if push != 0 {
		t.Errorf("push_enabled = %d, want 0", push)
	}
}

// TestUpdatePreferences_UnknownCategoryRejected — unbekannte Kategorie ⇒ 400
// (nicht 500) UND kein Teil-Write (transaktional/vorab abgelehnt).
func TestUpdatePreferences_UnknownCategoryRejected(t *testing.T) {
	database := testutil.NewDB(t)
	srv := prefsServer(t, database)
	uid := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, uid, "standard", nil)

	body := map[string]map[string]bool{
		"games":     {"push": false},
		"bogus_cat": {"push": true},
	}
	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/notification-preferences", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}

	var n int
	database.QueryRow(`SELECT COUNT(*) FROM notification_preferences WHERE user_id = ?`, uid).Scan(&n)
	if n != 0 {
		t.Errorf("Teil-Write: %d Zeilen persistiert, want 0", n)
	}
}

// Fehlerfall: die Route liegt im Authenticated-Tier — ohne Bearer-Token 401.
func TestVapidKey_Unauthenticated(t *testing.T) {
	database := testutil.NewDB(t)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VAPIDPublicKey: "test-vapid-public-key"}
	h := notifications.NewHandler(database, cfg)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/push/vapid-public-key", h.GetVAPIDPublicKey)
	})
	res := testutil.Get(t, srv, "/api/push/vapid-public-key", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", res.StatusCode)
	}
}

// subServer verdrahtet die Abo-Routen.
func subServer(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret}
	h := notifications.NewHandler(database, cfg)
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/push/subscribe", h.Subscribe)
		r.Delete("/api/push/subscribe", h.Unsubscribe)
	})
}

// TestSubscribe_CreatesAndUpserts — POST legt ein Abo an (204); ein zweiter
// POST mit demselben Endpoint aktualisiert statt zu duplizieren.
func TestSubscribe_CreatesAndUpserts(t *testing.T) {
	database := testutil.NewDB(t)
	srv := subServer(t, database)
	uid := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, uid, "standard", nil)

	body := map[string]string{"endpoint": "https://push.test/abc", "p256dh": "k1", "auth": "a1"}
	res := testutil.Do(t, srv, http.MethodPost, "/api/push/subscribe", tok, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	// Zweiter POST, gleicher Endpoint, neue Keys → Upsert, kein Duplikat.
	body2 := map[string]string{"endpoint": "https://push.test/abc", "p256dh": "k2", "auth": "a2"}
	res2 := testutil.Do(t, srv, http.MethodPost, "/api/push/subscribe", tok, body2)
	res2.Body.Close()

	var n int
	database.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = ?`, "https://push.test/abc").Scan(&n)
	if n != 1 {
		t.Fatalf("Abo-Zeilen = %d, want 1 (Upsert)", n)
	}
	var p256 string
	database.QueryRow(`SELECT p256dh FROM push_subscriptions WHERE endpoint = ?`, "https://push.test/abc").Scan(&p256)
	if p256 != "k2" {
		t.Errorf("p256dh = %q, want k2 (aktualisiert)", p256)
	}
}

// TestSubscribe_MissingField — fehlendes Pflichtfeld ⇒ 400.
func TestSubscribe_MissingField(t *testing.T) {
	database := testutil.NewDB(t)
	srv := subServer(t, database)
	uid := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, uid, "standard", nil)

	body := map[string]string{"p256dh": "k1", "auth": "a1"} // kein endpoint
	res := testutil.Do(t, srv, http.MethodPost, "/api/push/subscribe", tok, body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
}

// TestUnsubscribe_CrossUserProtected — B darf das Abo von A nicht löschen.
func TestUnsubscribe_CrossUserProtected(t *testing.T) {
	database := testutil.NewDB(t)
	srv := subServer(t, database)
	a := testutil.CreateUser(t, database, "standard")
	b := testutil.CreateUser(t, database, "standard")
	database.Exec(`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES (?, 'https://push.test/a', 'k', 'x')`, a)
	tokB := testutil.Token(t, b, "standard", nil)

	body := map[string]string{"endpoint": "https://push.test/a"}
	res := testutil.Do(t, srv, http.MethodDelete, "/api/push/subscribe", tokB, body)
	res.Body.Close()

	var n int
	database.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = 'https://push.test/a'`).Scan(&n)
	if n != 1 {
		t.Fatalf("A's Abo wurde von B gelöscht (n=%d)", n)
	}
}

// TestGetPreferences_Defaults — ohne gespeicherte Zeilen liefert GET alle
// Kategorien mit push=true/email=false.
func TestGetPreferences_Defaults(t *testing.T) {
	database := testutil.NewDB(t)
	srv := prefsServer(t, database)
	uid := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, uid, "standard", nil)

	res := testutil.Get(t, srv, "/api/profile/notification-preferences", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	var prefs map[string]map[string]bool
	json.NewDecoder(res.Body).Decode(&prefs)
	if _, ok := prefs["chat"]; !ok {
		t.Errorf("chat fehlt in Preferences-Response")
	}
	if !prefs["games"]["push"] || prefs["games"]["email"] {
		t.Errorf("games = %v, want push=true/email=false", prefs["games"])
	}
}

// TestGetPreferences_Unauthenticated — ohne Token 401.
func TestGetPreferences_Unauthenticated(t *testing.T) {
	database := testutil.NewDB(t)
	srv := prefsServer(t, database)
	res := testutil.Get(t, srv, "/api/profile/notification-preferences", "")
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", res.StatusCode)
	}
}
