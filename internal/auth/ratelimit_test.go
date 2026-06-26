package auth_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// authServerWithConfig mounts the unauthenticated auth routes onto a chi router
// backed by an auth.Handler with the given config (no IP rate limiter — that is
// wired in app.BuildRouter and tested separately).
func authServerWithConfig(t *testing.T, db *sql.DB, cfg *appconfig.Config) *httptest.Server {
	t.Helper()
	m := mailer.New(appconfig.SMTPConfig{}, "http://localhost", true) // disabled
	h := auth.NewHandler(db, cfg, testutil.TestJWTSecret, m, "http://localhost", hub.NewHub())
	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)
	r.Post("/api/auth/forgot-password", h.ForgotPassword)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// B-2: IP-Rate-Limiting drosselt die Auth-Routen ab dem konfigurierten Limit.
func TestAuthRateLimit_IPThrottle(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "standard")
	srv := prodserver.NewWithAuthRateLimit(t, db, 3) // 3 Anfragen/Minute/IP

	body := map[string]string{"email": "nobody@test.local", "password": "x"}
	// Die ersten 3 Anfragen passieren das Limit (401 bei unbekannter Mail).
	for i := 0; i < 3; i++ {
		res := testutil.Post(t, srv, "/api/auth/login", "", body)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("Anfrage %d unerwartet gedrosselt (429)", i+1)
		}
	}
	// Die nächste Anfrage überschreitet das Limit → 429.
	res := testutil.Post(t, srv, "/api/auth/login", "", body)
	res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("nach Limitüberschreitung: erwartet 429, bekam %d", res.StatusCode)
	}
}

// B-2: deaktiviertes Limit (0) drosselt nicht — Bestandsverhalten bleibt erhalten.
func TestAuthRateLimit_DisabledByDefault(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateUser(t, db, "standard")
	srv := prodserver.New(t, db) // AuthRateLimitPerMin = 0

	body := map[string]string{"email": "nobody@test.local", "password": "x"}
	for i := 0; i < 8; i++ {
		res := testutil.Post(t, srv, "/api/auth/login", "", body)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("ohne Limit unerwartet gedrosselt bei Anfrage %d", i+1)
		}
	}
}

// B-2: Account-Lockout sperrt nach N Fehlversuchen und antwortet ohne bcrypt.
func TestLogin_AccountLockout(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	email := emailSuffix(t, db, userID)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, LoginMaxFailures: 3, LoginLockMinutes: 15}
	srv := authServerWithConfig(t, db, cfg)

	// 3 Fehlversuche → Konto wird gesperrt.
	for i := 0; i < 3; i++ {
		res := testutil.Post(t, srv, "/api/auth/login", "",
			map[string]string{"email": email, "password": "wrong"})
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Fehlversuch %d: erwartet 401, bekam %d", i+1, res.StatusCode)
		}
	}

	// locked_until ist gesetzt.
	var lockedUntil sql.NullString
	db.QueryRow(`SELECT locked_until FROM users WHERE id=?`, userID).Scan(&lockedUntil)
	if !lockedUntil.Valid || lockedUntil.String == "" {
		t.Fatal("locked_until sollte nach 3 Fehlversuchen gesetzt sein")
	}

	// Trotz KORREKTEM Passwort: gesperrtes Konto → 429 (kein bcrypt, kein Login).
	res := testutil.Post(t, srv, "/api/auth/login", "",
		map[string]string{"email": email, "password": "test"})
	res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("gesperrtes Konto mit korrektem PW: erwartet 429, bekam %d", res.StatusCode)
	}
}

// B-2: erfolgreicher Login setzt den Fehlversuchszähler zurück.
func TestLogin_SuccessResetsFailureCounter(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	email := emailSuffix(t, db, userID)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, LoginMaxFailures: 5, LoginLockMinutes: 15}
	srv := authServerWithConfig(t, db, cfg)

	// 2 Fehlversuche (unter der Schwelle).
	for i := 0; i < 2; i++ {
		res := testutil.Post(t, srv, "/api/auth/login", "",
			map[string]string{"email": email, "password": "wrong"})
		res.Body.Close()
	}
	var count int
	db.QueryRow(`SELECT failed_login_count FROM users WHERE id=?`, userID).Scan(&count)
	if count != 2 {
		t.Fatalf("erwartet failed_login_count=2, bekam %d", count)
	}

	// Erfolgreicher Login.
	res := testutil.Post(t, srv, "/api/auth/login", "",
		map[string]string{"email": email, "password": "test"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("korrekter Login: erwartet 200, bekam %d", res.StatusCode)
	}
	db.QueryRow(`SELECT failed_login_count FROM users WHERE id=?`, userID).Scan(&count)
	if count != 0 {
		t.Errorf("nach Erfolg: erwartet failed_login_count=0, bekam %d", count)
	}
}

// B-2: forgot-password-Drosselung verhindert eine zweite Reset-Mail/Token im Cooldown.
func TestForgotPassword_PerAccountThrottle(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	email := emailSuffix(t, db, userID)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, ForgotPasswordCooldownSec: 60}
	srv := authServerWithConfig(t, db, cfg)

	// Erste Anfrage: 204, Token erzeugt.
	res := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": email})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erste forgot-password: erwartet 204, bekam %d", res.StatusCode)
	}
	if n := resetTokenCount(t, db, userID); n != 1 {
		t.Fatalf("nach erster Anfrage: erwartet 1 Reset-Token, bekam %d", n)
	}

	// Zweite Anfrage im Cooldown: weiterhin 204 (keine Enumeration), aber kein neuer Token.
	res = testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": email})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("zweite forgot-password: erwartet 204, bekam %d", res.StatusCode)
	}
	if n := resetTokenCount(t, db, userID); n != 1 {
		t.Errorf("im Cooldown: erwartet weiterhin 1 Reset-Token, bekam %d", n)
	}
}
