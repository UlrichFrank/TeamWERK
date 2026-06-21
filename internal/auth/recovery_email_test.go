package auth_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

// --- Helpers --------------------------------------------------------------

func setRecoveryEmail(t *testing.T, db *sql.DB, userID int, email string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE users SET recovery_email=? WHERE id=?`, email, userID); err != nil {
		t.Fatalf("setRecoveryEmail: %v", err)
	}
}

func linkParent(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, memberID); err != nil {
		t.Fatalf("linkParent: %v", err)
	}
}

func createUserWithEmail(t *testing.T, db *sql.DB, email string) int {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.MinCost)
	res, err := db.Exec(`INSERT INTO users (email, password, role, can_login) VALUES (?,?, 'standard', 1)`, email, string(hash))
	if err != nil {
		t.Fatalf("createUserWithEmail: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertRecoveryToken legt eine email_change_tokens-Zeile (field='recovery_email')
// mit der gegebenen Stufe an und gibt den Klartext-Token zurück.
func insertRecoveryToken(t *testing.T, db *sql.DB, userID int, newEmail, stage string, expiry time.Time) string {
	t.Helper()
	plain, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO email_change_tokens (user_id, token, new_email, expires_at, field, stage) VALUES (?,?,?,?, 'recovery_email', ?)`,
		userID, hash, newEmail, expiry, stage); err != nil {
		t.Fatalf("insertRecoveryToken: %v", err)
	}
	return plain
}

// getNoRedirect führt ein GET aus, ohne Redirects zu folgen (für 302-Prüfung).
func getNoRedirect(t *testing.T, url string) *http.Response {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Get(url)
	if err != nil {
		t.Fatalf("getNoRedirect: %v", err)
	}
	return res
}

func resetTokenCount(t *testing.T, db *sql.DB, userID int) int {
	t.Helper()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id=?`, userID).Scan(&n)
	return n
}

// --- Forgot-Password ------------------------------------------------------

func TestForgotPassword_KindPerLoginName_MailAnRecoveryEmail(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt") // can_login=1, email NULL
	setRecoveryEmail(t, db, childID, "mama@test.local")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": "Lena.Schmidt"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := resetTokenCount(t, db, childID); got != 1 {
		t.Errorf("expected 1 reset token for child (matched via login_name), got %d", got)
	}
}

func TestForgotPassword_RecoveryEmailIstKeinLookupKey(t *testing.T) {
	db := testutil.NewDB(t)
	parentID := createUserWithEmail(t, db, "mama@test.local")
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "mama@test.local") // gleiche Adresse als recovery
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": "mama@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := resetTokenCount(t, db, parentID); got != 1 {
		t.Errorf("expected token for PARENT (email lookup), got %d", got)
	}
	if got := resetTokenCount(t, db, childID); got != 0 {
		t.Errorf("recovery_email darf KEIN Lookup-Key sein: Kind-Token erwartet 0, got %d", got)
	}
}

func TestForgotPassword_ErwachsenerUnveraendert(t *testing.T) {
	db := testutil.NewDB(t)
	adultID := createUserWithEmail(t, db, "erw@test.local")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": "erw@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := resetTokenCount(t, db, adultID); got != 1 {
		t.Errorf("expected 1 reset token for adult, got %d", got)
	}
}

func TestForgotPassword_UnbekannterIdentifier_204OhneToken(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/auth/forgot-password", "", map[string]string{"email": "nobody@nowhere.test"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM password_reset_tokens`).Scan(&n)
	if n != 0 {
		t.Errorf("expected 0 tokens for unknown identifier, got %d", n)
	}
}

// --- Eltern-Änderungs-Workflow -------------------------------------------

func TestRequestRecoveryEmailChange_MailAnAlteAdresse(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	memberID := testutil.CreateMember(t, db, childID)
	parentID := testutil.CreateUser(t, db, "standard")
	linkParent(t, db, parentID, memberID)
	srv := newAuthServer(t, db)

	tok := testutil.TokenWithIsParent(t, parentID, "standard", nil, true)
	res := testutil.Do(t, srv, http.MethodPost, "/api/profile/kind/"+strconv.Itoa(memberID)+"/recovery-email", tok,
		map[string]string{"new_email": "new@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stage, newEmail string
	var usedAt sql.NullTime
	if err := db.QueryRow(
		`SELECT COALESCE(stage,''), new_email, used_at FROM email_change_tokens WHERE user_id=? AND field='recovery_email'`,
		childID).Scan(&stage, &newEmail, &usedAt); err != nil {
		t.Fatalf("recovery token not found: %v", err)
	}
	if stage != "auth" {
		t.Errorf("stage = %q, want auth", stage)
	}
	if newEmail != "new@test.local" {
		t.Errorf("new_email = %q", newEmail)
	}
	// recovery_email noch unverändert
	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "old@test.local" {
		t.Errorf("recovery_email darf noch nicht geändert sein, got %q", rec)
	}
}

func TestRequestRecoveryEmailChange_FremdesKind_403(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	memberID := testutil.CreateMember(t, db, childID)
	stranger := testutil.CreateUser(t, db, "standard") // NICHT verknüpft
	srv := newAuthServer(t, db)

	tok := testutil.Token(t, stranger, "standard", nil)
	res := testutil.Do(t, srv, http.MethodPost, "/api/profile/kind/"+strconv.Itoa(memberID)+"/recovery-email", tok,
		map[string]string{"new_email": "new@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM email_change_tokens WHERE field='recovery_email'`).Scan(&n)
	if n != 0 {
		t.Errorf("expected no token, got %d", n)
	}
}

func TestConfirmRecovery_StufeAlt_LoestStufeNeuAus(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	plain := insertRecoveryToken(t, db, childID, "new@test.local", "auth", time.Now().Add(24*time.Hour))
	srv := newAuthServer(t, db)

	res := getNoRedirect(t, srv.URL+"/api/profile/recovery-email/confirm?token="+plain)
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", res.StatusCode)
	}

	var stage string
	var usedAt sql.NullTime
	db.QueryRow(`SELECT COALESCE(stage,''), used_at FROM email_change_tokens WHERE user_id=? AND field='recovery_email'`,
		childID).Scan(&stage, &usedAt)
	if stage != "verify" {
		t.Errorf("stage = %q, want verify", stage)
	}
	if usedAt.Valid {
		t.Errorf("token sollte noch nicht verbraucht sein (used_at)")
	}
	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "old@test.local" {
		t.Errorf("recovery_email noch unverändert erwartet, got %q", rec)
	}
}

func TestConfirmRecovery_StufeNeu_SchreibtRecoveryEmail(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	plain := insertRecoveryToken(t, db, childID, "new@test.local", "verify", time.Now().Add(24*time.Hour))
	srv := newAuthServer(t, db)

	res := getNoRedirect(t, srv.URL+"/api/profile/recovery-email/confirm?token="+plain)
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", res.StatusCode)
	}

	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "new@test.local" {
		t.Errorf("recovery_email = %q, want new@test.local", rec)
	}
	var usedAt sql.NullTime
	db.QueryRow(`SELECT used_at FROM email_change_tokens WHERE user_id=? AND field='recovery_email'`, childID).Scan(&usedAt)
	if !usedAt.Valid {
		t.Errorf("token sollte verbraucht sein (used_at gesetzt)")
	}
}

func TestConfirmRecovery_AbgelaufenerToken_RedirectInvalid(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	plain := insertRecoveryToken(t, db, childID, "new@test.local", "verify", time.Now().Add(-time.Hour)) // abgelaufen
	srv := newAuthServer(t, db)

	res := getNoRedirect(t, srv.URL+"/api/profile/recovery-email/confirm?token="+plain)
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", res.StatusCode)
	}
	if loc := res.Header.Get("Location"); loc == "" || !containsSub(loc, "error=invalid_token") {
		t.Errorf("Location = %q, want enthält error=invalid_token", loc)
	}
	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "old@test.local" {
		t.Errorf("recovery_email darf unverändert bleiben, got %q", rec)
	}
}

// --- Admin/Vorstand-Override ----------------------------------------------

func TestAdminSetRecoveryEmail_DirektOhneWorkflow(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	tok := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodPut, "/api/users/"+strconv.Itoa(childID)+"/recovery-email", tok,
		map[string]string{"recovery_email": "set@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "set@test.local" {
		t.Errorf("recovery_email = %q, want set@test.local", rec)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM email_change_tokens`).Scan(&n)
	if n != 0 {
		t.Errorf("Override darf keinen Token erzeugen, got %d", n)
	}
}

func TestSetRecoveryEmail_OhneFunktion_403(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	standardID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	tok := testutil.Token(t, standardID, "standard", nil)
	res := testutil.Do(t, srv, http.MethodPut, "/api/users/"+strconv.Itoa(childID)+"/recovery-email", tok,
		map[string]string{"recovery_email": "set@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// --- Self-Edit härten -----------------------------------------------------

func TestUpdateAccount_KindKannRecoveryEmailNichtSetzen(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "old@test.local")
	srv := newAuthServer(t, db)

	tok := testutil.Token(t, childID, "standard", nil)
	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/account", tok,
		map[string]string{"first_name": "Lena", "last_name": "Schmidt", "recovery_email": "hijack@test.local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var rec string
	db.QueryRow(`SELECT recovery_email FROM users WHERE id=?`, childID).Scan(&rec)
	if rec != "old@test.local" {
		t.Errorf("Kind darf recovery_email nicht setzen, got %q", rec)
	}
}

// --- Approval-Wiring ------------------------------------------------------

func TestApproveChild_PersistiertRecoveryEmail(t *testing.T) {
	db := testutil.NewDB(t)
	reqID := createChildMembershipRequest(t, db, "Lena", "Schmidt", "mama@test.local")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Post(t, srv, "/api/membership-requests/"+strconv.Itoa(reqID)+"/approve",
		testutil.Token(t, adminID, "admin", nil), nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var rec sql.NullString
	if err := db.QueryRow(`SELECT recovery_email FROM users WHERE login_name='Lena.Schmidt'`).Scan(&rec); err != nil {
		t.Fatalf("child user not found: %v", err)
	}
	if !rec.Valid || rec.String != "mama@test.local" {
		t.Errorf("recovery_email = %v, want mama@test.local", rec.String)
	}
}

// --- Read-Surface ---------------------------------------------------------

func TestGetChildProfile_ZeigtRecoveryEmail(t *testing.T) {
	db := testutil.NewDB(t)
	childID := createChildAccount(t, db, "Lena.Schmidt")
	setRecoveryEmail(t, db, childID, "mama@test.local")
	memberID := testutil.CreateMember(t, db, childID)
	parentID := testutil.CreateUser(t, db, "standard")
	linkParent(t, db, parentID, memberID)
	srv := newAuthServer(t, db)

	tok := testutil.TokenWithIsParent(t, parentID, "standard", nil, true)
	res := testutil.Get(t, srv, "/api/profile/kind/"+strconv.Itoa(memberID), tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		UserContact struct {
			RecoveryEmail string `json:"recovery_email"`
		} `json:"user_contact"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.UserContact.RecoveryEmail != "mama@test.local" {
		t.Errorf("user_contact.recovery_email = %q, want mama@test.local", body.UserContact.RecoveryEmail)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
