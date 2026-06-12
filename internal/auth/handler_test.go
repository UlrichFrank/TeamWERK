package auth_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newAuthServer(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	cfg := testutil.TestConfig()
	m := mailer.New(appconfig.SMTPConfig{}, "http://localhost")
	h := auth.NewHandler(database, cfg, testutil.TestJWTSecret, m, "http://localhost")

	r := chi.NewRouter()
	// Public routes — no auth middleware
	r.Post("/api/auth/login", h.Login)
	r.Post("/api/auth/refresh", h.Refresh)
	r.Post("/api/auth/logout", h.Logout)
	r.Post("/api/auth/register", h.Register)
	r.Post("/api/auth/forgot-password", h.ForgotPassword)
	r.Post("/api/auth/reset-password", h.ResetPassword)
	// Protected routes — require auth + role
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(testutil.TestJWTSecret))
		r.Use(auth.RequireRole("admin", "standard"))
		r.Put("/api/admin/users/{id}/role", h.UpdateUserRole)
		r.Delete("/api/admin/users/{id}", h.DeleteUser)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func refreshTokenCount(t *testing.T, db *sql.DB, userID int) int {
	t.Helper()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM refresh_tokens WHERE user_id=?`, userID).Scan(&n)
	return n
}

// TC-A01: valider Login liefert access_token und setzt refresh_token-Cookie.
func TestLogin_ValidCredentials(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	// CreateUser sets password to "test" with bcrypt
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": emailSuffix(t, db, userID), "password": "test"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["access_token"] == "" {
		t.Error("access_token missing in response")
	}
	var hasCookie bool
	for _, c := range res.Cookies() {
		if c.Name == "refresh_token" && c.HttpOnly {
			hasCookie = true
		}
	}
	if !hasCookie {
		t.Error("expected HttpOnly refresh_token cookie")
	}
	if refreshTokenCount(t, db, userID) != 1 {
		t.Error("expected 1 refresh_token in DB")
	}
}

// TC-A02: falsches Passwort → 401.
func TestLogin_WrongPassword(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": emailSuffix(t, db, userID), "password": "wrong"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
	if refreshTokenCount(t, db, userID) != 0 {
		t.Error("no refresh_token should be created on failed login")
	}
}

// TC-A03: unbekannte E-Mail → 401.
func TestLogin_UnknownEmail(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "nobody@test.local", "password": "test"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

// TC-A04: Proxy-Account (can_login=0) kann sich nicht einloggen.
func TestLogin_ProxyAccountBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := db.Exec(
		`INSERT INTO users (email, password, role, can_login) VALUES (?, ?, ?, 0)`,
		"proxy@test.local", "", "standard"); err != nil {
		t.Fatalf("insert proxy: %v", err)
	}
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "proxy@test.local", "password": ""})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for proxy account, got %d", res.StatusCode)
	}
}

// TC-A05: Token-Refresh rotiert den Token.
func TestRefresh_ValidCookie(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	plain := testutil.CreateRefreshToken(t, db, userID)
	srv := newAuthServer(t, db)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: plain})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["access_token"] == "" {
		t.Error("new access_token missing")
	}
	// Old token must be deleted, new one inserted → still 1 token total.
	if refreshTokenCount(t, db, userID) != 1 {
		t.Errorf("expected exactly 1 refresh_token after rotation, got %d", refreshTokenCount(t, db, userID))
	}
}

// TC-A06: ungültiger Refresh-Cookie → 401.
func TestRefresh_InvalidCookie(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "notavalidtoken"})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
}

// TC-A07: Logout löscht Token aus DB und setzt MaxAge=-1.
func TestLogout_ClearsToken(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	plain := testutil.CreateRefreshToken(t, db, userID)
	srv := newAuthServer(t, db)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: plain})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", res.StatusCode)
	}
	if refreshTokenCount(t, db, userID) != 0 {
		t.Error("refresh_token should be deleted after logout")
	}
	for _, c := range res.Cookies() {
		if c.Name == "refresh_token" && c.MaxAge != -1 {
			t.Errorf("expected MaxAge=-1 for refresh_token cookie, got %d", c.MaxAge)
		}
	}
}

// TC-A08: Registrierung mit gültigem Einladungstoken.
func TestRegister_ValidToken(t *testing.T) {
	db := testutil.NewDB(t)
	plain := testutil.CreateInvitationToken(t, db, "new@test.local", "standard", time.Now().Add(48*time.Hour))
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token":      plain,
		"first_name": "Anna",
		"last_name":  "Muster",
		"password":   "sicher123",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var userCount int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE email='new@test.local'`).Scan(&userCount)
	if userCount != 1 {
		t.Error("user not created after registration")
	}
	var usedAt sql.NullString
	db.QueryRow(`SELECT used_at FROM invitation_tokens WHERE email='new@test.local'`).Scan(&usedAt)
	if !usedAt.Valid {
		t.Error("invitation_token.used_at should be set after registration")
	}
}

// TC-A09: Registrierung mit abgelaufenem Token → 400.
func TestRegister_ExpiredToken(t *testing.T) {
	db := testutil.NewDB(t)
	plain := testutil.CreateInvitationToken(t, db, "expired@test.local", "standard", time.Now().Add(-1*time.Hour))
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token":    plain,
		"password": "sicher123",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for expired token, got %d", res.StatusCode)
	}
}

// TC-A10: Registrierung mit bereits benutztem Token → 400.
func TestRegister_UsedToken(t *testing.T) {
	db := testutil.NewDB(t)
	plain := testutil.CreateInvitationToken(t, db, "used@test.local", "standard", time.Now().Add(48*time.Hour))
	db.Exec(`UPDATE invitation_tokens SET used_at=CURRENT_TIMESTAMP WHERE email='used@test.local'`)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token":    plain,
		"password": "sicher123",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for used token, got %d", res.StatusCode)
	}
}

// TC-A11: ForgotPassword antwortet immer 204 — auch bei unbekannter E-Mail.
func TestForgotPassword_AlwaysNoContent(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	email := emailSuffix(t, db, userID)
	srv := newAuthServer(t, db)

	// Bekannte Mail: Token angelegt.
	resKnown := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": email})
	defer resKnown.Body.Close()
	if resKnown.StatusCode != http.StatusNoContent {
		t.Errorf("known email: expected 204, got %d", resKnown.StatusCode)
	}
	var tokenCount int
	db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id=?`, userID).Scan(&tokenCount)
	if tokenCount != 1 {
		t.Errorf("expected 1 reset token for known email, got %d", tokenCount)
	}

	// Unbekannte Mail: ebenfalls 204, kein Token.
	resUnknown := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": "nobody@test.local"})
	defer resUnknown.Body.Close()
	if resUnknown.StatusCode != http.StatusNoContent {
		t.Errorf("unknown email: expected 204, got %d", resUnknown.StatusCode)
	}
}

// TC-A12: ResetPassword mit gültigem Token — Passwort geändert, alle Refresh-Tokens gelöscht.
func TestResetPassword_Valid(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	// Aktive Session anlegen, die nach Reset gelöscht werden muss.
	testutil.CreateRefreshToken(t, db, userID)
	plain := testutil.CreatePasswordResetToken(t, db, userID, time.Now().Add(1*time.Hour))
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/reset-password", "", map[string]string{
		"token":    plain,
		"password": "neuesPasswort99",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if refreshTokenCount(t, db, userID) != 0 {
		t.Error("all refresh_tokens should be deleted after password reset")
	}
	var usedAt sql.NullString
	db.QueryRow(`SELECT used_at FROM password_reset_tokens WHERE user_id=?`, userID).Scan(&usedAt)
	if !usedAt.Valid {
		t.Error("password_reset_tokens.used_at should be set")
	}
}

// TC-A13: ResetPassword mit abgelaufenem Token → 400.
func TestResetPassword_ExpiredToken(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	plain := testutil.CreatePasswordResetToken(t, db, userID, time.Now().Add(-1*time.Minute))
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/reset-password", "", map[string]string{
		"token":    plain,
		"password": "neues",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}

// TC-A14: UpdateUserRole — nur Admin darf "admin" vergeben; ungültige Rolle → 400.
func TestUpdateUserRole_AdminOnly(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	targetID := testutil.CreateUser(t, db, "standard")
	otherID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	// Admin vergibt "admin" → 204.
	res := testutil.Do(t, srv, http.MethodPut,
		"/api/admin/users/"+itoa(targetID)+"/role",
		testutil.Token(t, adminID, "admin", nil),
		map[string]string{"role": "admin"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("admin→admin: expected 204, got %d", res.StatusCode)
	}

	// Nicht-Admin vergibt "admin" → 403.
	res2 := testutil.Do(t, srv, http.MethodPut,
		"/api/admin/users/"+itoa(targetID)+"/role",
		testutil.Token(t, otherID, "standard", nil),
		map[string]string{"role": "admin"})
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin→admin: expected 403, got %d", res2.StatusCode)
	}

	// Ungültige Rolle → 400.
	res3 := testutil.Do(t, srv, http.MethodPut,
		"/api/admin/users/"+itoa(targetID)+"/role",
		testutil.Token(t, adminID, "admin", nil),
		map[string]string{"role": "trainer"})
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid role: expected 400, got %d", res3.StatusCode)
	}
}

// TC-A15a: Admin darf sich nicht selbst löschen.
func TestDeleteUser_SelfForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/admin/users/"+itoa(adminID),
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for self-delete, got %d", res.StatusCode)
	}
}

// TC-A15b: Nutzer löschen cascadiert Tokens, Assignments und Familienlinks.
func TestDeleteUser_Cascade(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	targetID := testutil.CreateUser(t, db, "standard")
	testutil.CreateRefreshToken(t, db, targetID)
	childMemberID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, targetID, childMemberID)

	srv := newAuthServer(t, db)
	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/admin/users/"+itoa(targetID),
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if refreshTokenCount(t, db, targetID) != 0 {
		t.Error("refresh_tokens should be cascade-deleted")
	}
	var flCount int
	db.QueryRow(`SELECT COUNT(*) FROM family_links WHERE parent_user_id=?`, targetID).Scan(&flCount)
	if flCount != 0 {
		t.Error("family_links should be cascade-deleted")
	}
}

// emailSuffix reads the email of a user by ID from DB.
func emailSuffix(t *testing.T, db *sql.DB, userID int) string {
	t.Helper()
	var email string
	db.QueryRow(`SELECT email FROM users WHERE id=?`, userID).Scan(&email)
	return email
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

