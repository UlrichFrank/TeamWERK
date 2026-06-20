package auth_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

// createChildAccount legt ein aktiviertes Kinder-Konto an: keine E-Mail,
// login_name gesetzt, can_login=1, Passwort = "kidpass".
func createChildAccount(t *testing.T, db *sql.DB, loginName string) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("kidpass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO users (email, login_name, password, role, can_login) VALUES (NULL, ?, ?, 'standard', 1)`,
		loginName, string(hash))
	if err != nil {
		t.Fatalf("createChildAccount: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// TC-KID-LOGIN01: Kind loggt sich mit login_name (Vorname.Nachname) ein.
func TestLogin_ByLoginName(t *testing.T) {
	db := testutil.NewDB(t)
	userID := createChildAccount(t, db, "Lena.Schmidt")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "Lena.Schmidt", "password": "kidpass"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["access_token"] == "" {
		t.Error("access_token missing")
	}
	if refreshTokenCount(t, db, userID) != 1 {
		t.Error("expected 1 refresh_token")
	}
}

// TC-KID-LOGIN02: login_name wird case-insensitiv erkannt.
func TestLogin_ByLoginNameCaseInsensitive(t *testing.T) {
	db := testutil.NewDB(t)
	createChildAccount(t, db, "Lena.Schmidt")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "lena.schmidt", "password": "kidpass"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for lowercase login_name, got %d", res.StatusCode)
	}
}

// TC-KID-LOGIN03: inaktives Kind-Konto (can_login=0) → 401.
func TestLogin_ByLoginNameInactiveBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("kidpass"), bcrypt.MinCost)
	if _, err := db.Exec(
		`INSERT INTO users (email, login_name, password, role, can_login) VALUES (NULL, ?, ?, 'standard', 0)`,
		"Tim.Bauer", string(hash)); err != nil {
		t.Fatalf("insert: %v", err)
	}
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "Tim.Bauer", "password": "kidpass"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for inactive child account, got %d", res.StatusCode)
	}
}

// TC-KID-LOGIN04: korrekter login_name, falsches Passwort → 401.
func TestLogin_ByLoginNameWrongPassword(t *testing.T) {
	db := testutil.NewDB(t)
	userID := createChildAccount(t, db, "Lena.Schmidt")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/login",
		"", map[string]string{"email": "Lena.Schmidt", "password": "wrong"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", res.StatusCode)
	}
	if refreshTokenCount(t, db, userID) != 0 {
		t.Error("no refresh_token on failed login")
	}
}

// insertInactiveChildWithToken legt ein inaktives Kind-Konto (can_login=0) mit
// einem password_reset_token an und gibt (userID, plainToken) zurück.
func insertInactiveChildWithToken(t *testing.T, db *sql.DB, loginName string, expiry time.Time, used bool) (int, string) {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (email, login_name, password, role, can_login) VALUES (NULL, ?, '', 'standard', 0)`,
		loginName)
	if err != nil {
		t.Fatalf("insert child: %v", err)
	}
	userID, _ := res.LastInsertId()
	plain, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	usedAt := sql.NullTime{}
	if used {
		usedAt = sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true}
	}
	if _, err := db.Exec(
		`INSERT INTO password_reset_tokens (user_id, token, expires_at, used_at) VALUES (?,?,?,?)`,
		userID, hash, expiry, usedAt); err != nil {
		t.Fatalf("insert token: %v", err)
	}
	return int(userID), plain
}

// TC-KID-SETPW01: gültiger Token setzt Passwort und aktiviert das Konto.
func TestResetPassword_ActivatesChildAccount(t *testing.T) {
	db := testutil.NewDB(t)
	userID, plain := insertInactiveChildWithToken(t, db, "Lena.Schmidt", time.Now().Add(48*time.Hour), false)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/reset-password",
		"", map[string]string{"token": plain, "password": "neuespasswort"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var canLogin int
	db.QueryRow(`SELECT can_login FROM users WHERE id=?`, userID).Scan(&canLogin)
	if canLogin != 1 {
		t.Errorf("can_login = %d, want 1 (activated)", canLogin)
	}
}

// TC-KID-SETPW02: abgelaufener Token → 400, Konto bleibt inaktiv.
func TestResetPassword_ExpiredTokenKeepsInactive(t *testing.T) {
	db := testutil.NewDB(t)
	userID, plain := insertInactiveChildWithToken(t, db, "Tim.Bauer", time.Now().Add(-time.Hour), false)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/reset-password",
		"", map[string]string{"token": plain, "password": "neuespasswort"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for expired token, got %d", res.StatusCode)
	}
	var canLogin int
	db.QueryRow(`SELECT can_login FROM users WHERE id=?`, userID).Scan(&canLogin)
	if canLogin != 0 {
		t.Errorf("can_login = %d, want 0 (still inactive)", canLogin)
	}
}

// TC-KID-SETPW03: bereits verbrauchter Token → 400.
func TestResetPassword_UsedTokenRejected(t *testing.T) {
	db := testutil.NewDB(t)
	_, plain := insertInactiveChildWithToken(t, db, "Mia.Klein", time.Now().Add(48*time.Hour), true)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/reset-password",
		"", map[string]string{"token": plain, "password": "neuespasswort"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for used token, got %d", res.StatusCode)
	}
}

// createChildMembershipRequest legt einen pending Kinderantrag an.
func createChildMembershipRequest(t *testing.T, db *sql.DB, firstName, lastName, parentEmail string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO membership_requests (first_name, last_name, email, status, is_child, parent_email)
		 VALUES (?, ?, ?, 'pending', 1, ?)`,
		firstName, lastName, parentEmail, parentEmail)
	if err != nil {
		t.Fatalf("createChildMembershipRequest: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// TC-KID-APPROVE01: Kinder-Approve legt Konto + Member an, Status approved.
func TestApproveMembershipRequest_Child(t *testing.T) {
	db := testutil.NewDB(t)
	reqID := createChildMembershipRequest(t, db, "Lena", "Schmidt", "mama@test.local")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv,
		"/api/membership-requests/"+itoa(reqID)+"/approve",
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var userID, canLogin int
	var email sql.NullString
	if err := db.QueryRow(
		`SELECT id, can_login, email FROM users WHERE login_name = 'Lena.Schmidt'`,
	).Scan(&userID, &canLogin, &email); err != nil {
		t.Fatalf("child user not found: %v", err)
	}
	if canLogin != 0 {
		t.Errorf("can_login = %d, want 0 (not yet activated)", canLogin)
	}
	if email.Valid {
		t.Errorf("child user email should be NULL, got %q", email.String)
	}

	var memberUserID int
	if err := db.QueryRow(
		`SELECT user_id FROM members WHERE first_name='Lena' AND last_name='Schmidt'`,
	).Scan(&memberUserID); err != nil {
		t.Fatalf("member not found: %v", err)
	}
	if memberUserID != userID {
		t.Errorf("member.user_id = %d, want %d", memberUserID, userID)
	}

	var status string
	db.QueryRow(`SELECT status FROM membership_requests WHERE id=?`, reqID).Scan(&status)
	if status != "approved" {
		t.Errorf("status = %q, want approved", status)
	}

	var tokenCount int
	db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id=?`, userID).Scan(&tokenCount)
	if tokenCount != 1 {
		t.Errorf("expected 1 password_reset_token, got %d", tokenCount)
	}
}

// TC-KID-APPROVE02: Bei Namensgleichheit erhält der Spielername ein Suffix.
func TestApproveMembershipRequest_ChildCollisionSuffix(t *testing.T) {
	db := testutil.NewDB(t)
	createChildAccount(t, db, "Lena.Schmidt") // belegt den Namen bereits
	reqID := createChildMembershipRequest(t, db, "Lena", "Schmidt", "mama@test.local")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv,
		"/api/membership-requests/"+itoa(reqID)+"/approve",
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE login_name='Lena.Schmidt2'`).Scan(&n)
	if n != 1 {
		t.Errorf("expected a user with login_name Lena.Schmidt2, found %d", n)
	}
}

// TC-KID-APPROVE03: Kinder-Approve legt KEINEN family_link an (reine Korrespondenz).
func TestApproveMembershipRequest_ChildNoFamilyLink(t *testing.T) {
	db := testutil.NewDB(t)
	reqID := createChildMembershipRequest(t, db, "Tim", "Bauer", "papa@test.local")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv,
		"/api/membership-requests/"+itoa(reqID)+"/approve",
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var links int
	db.QueryRow(`SELECT COUNT(*) FROM family_links`).Scan(&links)
	if links != 0 {
		t.Errorf("expected no family_links, got %d", links)
	}
}

// TC-KID-REQ01: Kinderantrag wird mit is_child=1 und parent_email gespeichert.
func TestRequestMembership_Child(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/request-membership", "", map[string]any{
		"first_name":   "Lena",
		"last_name":    "Schmidt",
		"is_child":     true,
		"parent_email": "mama@test.local",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var isChild int
	var parentEmail, email string
	if err := db.QueryRow(
		`SELECT is_child, COALESCE(parent_email,''), email FROM membership_requests WHERE first_name='Lena' AND last_name='Schmidt'`,
	).Scan(&isChild, &parentEmail, &email); err != nil {
		t.Fatalf("query: %v", err)
	}
	if isChild != 1 {
		t.Errorf("is_child = %d, want 1", isChild)
	}
	if parentEmail != "mama@test.local" {
		t.Errorf("parent_email = %q, want mama@test.local", parentEmail)
	}
	if email != "mama@test.local" {
		t.Errorf("email column should mirror parent_email, got %q", email)
	}
}

// TC-KID-REQ02: Kinderantrag ohne gültige Eltern-E-Mail → 400.
func TestRequestMembership_ChildMissingParentEmail(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/request-membership", "", map[string]any{
		"first_name": "Lena",
		"last_name":  "Schmidt",
		"is_child":   true,
		// parent_email fehlt
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM membership_requests`).Scan(&n)
	if n != 0 {
		t.Errorf("no request should be stored, found %d", n)
	}
}

// TC-KID-REQ03: Standard-Antrag bleibt is_child=0.
func TestRequestMembership_StandardStaysNonChild(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/request-membership", "", map[string]any{
		"first_name": "Max",
		"last_name":  "Mustermann",
		"email":      "max@test.local",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var isChild int
	if err := db.QueryRow(
		`SELECT is_child FROM membership_requests WHERE email='max@test.local'`,
	).Scan(&isChild); err != nil {
		t.Fatalf("query: %v", err)
	}
	if isChild != 0 {
		t.Errorf("is_child = %d, want 0", isChild)
	}
}
