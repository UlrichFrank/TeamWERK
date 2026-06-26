package auth_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

// B-6: Register erzwingt die Passwort-Mindeststärke.
func TestRegister_PasswordPolicy(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	// Zu kurz (< 12) → 400, kein Account.
	short := testutil.CreateInvitationToken(t, db, "short@test.local", "standard", time.Now().Add(time.Hour))
	res := testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token": short, "first_name": "A", "last_name": "B", "password": "kurz",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("zu kurzes PW: erwartet 400, bekam %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE email='short@test.local'`).Scan(&n)
	if n != 0 {
		t.Errorf("kein Account bei zu kurzem PW erwartet, fand %d", n)
	}

	// Übergroß (> 72 Byte) → 400.
	big := testutil.CreateInvitationToken(t, db, "big@test.local", "standard", time.Now().Add(time.Hour))
	res = testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token": big, "first_name": "A", "last_name": "B", "password": strings.Repeat("x", 73),
	})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("zu langes PW: erwartet 400, bekam %d", res.StatusCode)
	}

	// Gültig (≥ 12, ≤ 72) → 201.
	ok := testutil.CreateInvitationToken(t, db, "ok@test.local", "standard", time.Now().Add(time.Hour))
	res = testutil.Post(t, srv, "/api/auth/register", "", map[string]any{
		"token": ok, "first_name": "A", "last_name": "B", "password": "gueltigesPW12",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("gültiges PW: erwartet 201, bekam %d", res.StatusCode)
	}
}

// B-6: ResetPassword lehnt zu kurzes Passwort ab und aktiviert dabei kein Kind-Konto.
func TestResetPassword_PasswordPolicy(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	// Kind-Konto-Charakteristik: can_login=0 (wird erst beim gültigen Setzen aktiviert).
	db.Exec(`UPDATE users SET can_login=0 WHERE id=?`, userID)
	plain := testutil.CreatePasswordResetToken(t, db, userID, time.Now().Add(time.Hour))
	srv := newAuthServer(t, db)

	// Zu kurz → 400, can_login bleibt 0.
	res := testutil.Post(t, srv, "/api/auth/reset-password", "",
		map[string]string{"token": plain, "password": "kurz"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("zu kurzes PW: erwartet 400, bekam %d", res.StatusCode)
	}
	var canLogin int
	db.QueryRow(`SELECT can_login FROM users WHERE id=?`, userID).Scan(&canLogin)
	if canLogin != 0 {
		t.Errorf("Kind-Konto darf bei abgelehntem Reset nicht aktiviert werden (can_login=%d)", canLogin)
	}

	// Gültig → 204, can_login=1.
	res = testutil.Post(t, srv, "/api/auth/reset-password", "",
		map[string]string{"token": plain, "password": "gueltigesPW12"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("gültiges PW: erwartet 204, bekam %d", res.StatusCode)
	}
	db.QueryRow(`SELECT can_login FROM users WHERE id=?`, userID).Scan(&canLogin)
	if canLogin != 1 {
		t.Errorf("nach gültigem Reset: erwartet can_login=1, bekam %d", canLogin)
	}
}

// B-6 (sanft): Login signalisiert einen Upgrade-Hinweis bei zu kurzem Bestandspasswort.
func TestLogin_WeakPasswordHint(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard") // Passwort "test" (4 < 12)
	email := emailSuffix(t, db, userID)
	srv := newAuthServer(t, db)

	decodeHint := func(res *http.Response) bool {
		var body struct {
			AccessToken               string `json:"access_token"`
			PasswordChangeRecommended bool   `json:"password_change_recommended"`
		}
		json.NewDecoder(res.Body).Decode(&body)
		res.Body.Close()
		return body.PasswordChangeRecommended
	}

	// Kurzes Bestandspasswort → Hinweis-Flag true (kein Block, Login gelingt).
	res := testutil.Post(t, srv, "/api/auth/login", "", map[string]string{"email": email, "password": "test"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Login mit Bestandspasswort: erwartet 200, bekam %d", res.StatusCode)
	}
	if !decodeHint(res) {
		t.Error("kurzes Bestandspasswort: erwartet password_change_recommended=true")
	}

	// Starkes Passwort setzen → Login ohne Hinweis-Flag.
	hash, _ := bcrypt.GenerateFromPassword([]byte("langGenugPW12"), bcrypt.MinCost)
	db.Exec(`UPDATE users SET password=? WHERE id=?`, string(hash), userID)
	res = testutil.Post(t, srv, "/api/auth/login", "", map[string]string{"email": email, "password": "langGenugPW12"})
	if decodeHint(res) {
		t.Error("starkes Passwort: erwartet keinen Upgrade-Hinweis")
	}
}

// B-6: ChangePassword lehnt zu kurzes neues Passwort ab.
func TestChangePassword_PasswordPolicy(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", nil)
	srv := newAuthServer(t, db)

	// Korrektes aktuelles PW, aber zu kurzes neues → 400.
	res := testutil.Post(t, srv, "/api/profile/password", token,
		map[string]string{"current_password": "test", "new_password": "kurz"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("zu kurzes neues PW: erwartet 400, bekam %d", res.StatusCode)
	}

	// Gültiges neues PW → 204.
	res = testutil.Post(t, srv, "/api/profile/password", token,
		map[string]string{"current_password": "test", "new_password": "gueltigesPW12"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("gültiges neues PW: erwartet 204, bekam %d", res.StatusCode)
	}
}
