package auth

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/policy"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db        *sql.DB
	cfg       *appconfig.Config
	jwtSecret string
	mailer    *mailer.Mailer
	baseURL   string
	hub       *hub.EventHub

	// forgot-password Konto-Drosselung (in-process, best-effort): letzter
	// Mailversand je Konto-Name, geschützt durch fpMu. Resettet bei Neustart.
	fpMu   sync.Mutex
	fpLast map[string]time.Time
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, jwtSecret string, m *mailer.Mailer, baseURL string, h *hub.EventHub) *Handler {
	return &Handler{db: db, cfg: cfg, jwtSecret: jwtSecret, mailer: m, baseURL: baseURL, hub: h, fpLast: make(map[string]time.Time)}
}

// broadcastFinance sends a members/users live-update event only to the finance
// group (vorstand/vorstand_beisitzer/kassierer + admin) — the roles that may
// read member/user data — plus any explicitly affected users. Replaces the
// former global Broadcast("members"/"users"); the topic string and the
// Frontend useLiveUpdates contract stay unchanged, only the recipient set
// shrinks.
func (h *Handler) broadcastFinance(ctx context.Context, event string, extraUserIDs ...int) {
	if h.hub == nil {
		return
	}
	ids := hub.NewAudience(h.db).FinanceGroup(ctx, extraUserIDs...)
	h.hub.BroadcastToUsers(ids, event)
}

// forgotPasswordAllowed reports whether a reset mail may be sent for the given
// account name now, recording the send time when it returns true. A
// non-positive cooldown disables the throttle (e.g. in tests).
func (h *Handler) forgotPasswordAllowed(accountName string) bool {
	if h.cfg == nil || h.cfg.ForgotPasswordCooldownSec <= 0 {
		return true
	}
	cooldown := time.Duration(h.cfg.ForgotPasswordCooldownSec) * time.Second
	key := strings.ToLower(accountName)
	h.fpMu.Lock()
	defer h.fpMu.Unlock()
	if last, ok := h.fpLast[key]; ok && time.Since(last) < cooldown {
		return false
	}
	h.fpLast[key] = time.Now()
	return true
}

// bcryptCost ist der Cost-Faktor für alle Passwort-Hashes. Default ist
// bcrypt.DefaultCost (10); Tests setzen via TestMain auf bcrypt.MinCost (4),
// damit der Race-Detector nicht jede Hash-Operation um den Faktor ~40 ausbremst.
var bcryptCost = bcrypt.DefaultCost

// dummyHash ist ein lazy berechneter bcrypt-Hash für den Login-ErrNoRows-Pfad,
// damit die Antwortzeit bei unbekanntem Account nicht von einem echten Treffer
// unterscheidbar ist. Lazy, damit Tests den Cost-Faktor in TestMain absenken
// können, bevor der Hash zum ersten Mal gebraucht wird.
var (
	dummyHashOnce  sync.Once
	dummyHashCache []byte
)

func dummyHash() []byte {
	dummyHashOnce.Do(func() {
		dummyHashCache, _ = bcrypt.GenerateFromPassword([]byte("teamwerk-dummy-password-for-timing"), bcryptCost)
	})
	return dummyHashCache
}

// maxPasswordBytes ist die bcrypt-Grenze: längere Eingaben würden stillschweigend
// trunkiert, daher lehnen wir sie explizit ab.
const maxPasswordBytes = 72

const defaultPasswordMinLength = 12

// passwordMinLength liefert die wirksame Mindestlänge (Config oder Default).
func (h *Handler) passwordMinLength() int {
	if h.cfg != nil && h.cfg.PasswordMinLength > 0 {
		return h.cfg.PasswordMinLength
	}
	return defaultPasswordMinLength
}

// validatePassword erzwingt die serverseitige Passwort-Mindeststärke. Ein leerer
// Fehler-Rückgabewert bedeutet „gültig". Gilt für Register/Reset/Change.
func (h *Handler) validatePassword(pw string) error {
	if len([]rune(pw)) < h.passwordMinLength() {
		return fmt.Errorf("password must be at least %d characters", h.passwordMinLength())
	}
	if len(pw) > maxPasswordBytes {
		return fmt.Errorf("password must not exceed %d bytes", maxPasswordBytes)
	}
	return nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Umschließende Whitespaces (Autofill/Copy-Paste, mobile Tastaturen) abschneiden.
	// Muss konsistent zu Register/ResetPassword sein, wo das Passwort ebenfalls
	// getrimmt gespeichert wird — sonst schlägt der bcrypt-Vergleich fehl.
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	var id, failedCount int
	var hash, role, ident string
	var lockedUntil sql.NullString
	// Login akzeptiert E-Mail ODER login_name (Vorname.Nachname für Kinder ohne
	// E-Mail). Beide Spalten werden case-insensitiv gegen denselben Eingabewert
	// geprüft. ident ist die Identität fürs JWT: E-Mail, sonst der login_name.
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, password, role, COALESCE(NULLIF(email, ''), login_name, ''), failed_login_count, locked_until
		 FROM users WHERE (LOWER(email) = LOWER(?) OR LOWER(login_name) = LOWER(?)) AND can_login = 1`,
		req.Email, req.Email,
	).Scan(&id, &hash, &role, &ident, &failedCount, &lockedUntil)
	if err == sql.ErrNoRows {
		bcrypt.CompareHashAndPassword(dummyHash(), []byte(req.Password)) //nolint:errcheck
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		slog.Error("login query failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Account-Lockout: ein gesperrtes Konto antwortet ohne bcrypt-Prüfung. Der
	// generische 429 ist von der IP-Drosselung (httprate) nicht unterscheidbar, so
	// dass die Sperre keine Konto-Existenz verrät.
	if accountLocked(lockedUntil) {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		h.registerFailedLogin(r.Context(), id, failedCount)
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Erfolgreicher Login hebt Zähler und Sperre auf.
	h.resetLoginFailures(r.Context(), id)
	h.db.ExecContext(r.Context(), `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	slog.Info("login loadJWTExtras done", "clubFunctions", clubFunctions, "isParent", isParent)
	accessToken, err := IssueAccessToken(h.jwtSecret, id, ident, role, clubFunctions, isParent)
	if err != nil {
		slog.Error("login issue access token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	plain, tokenHash, err := GenerateOpaqueToken()
	slog.Info("login generate opaque token done")
	if err != nil {
		slog.Error("login generate opaque token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiry := RefreshTokenExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, tokenHash, expiry,
	); err != nil {
		slog.Error("login insert refresh token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    plain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiry,
		Path:     "/",
	})
	clearLegacyRefreshCookie(w)
	// Sanfter Upgrade-Hinweis: Bestandspasswörter werden nicht zwangsweise
	// zurückgesetzt. Die Klartext-Länge ist nur hier (beim Login) bekannt — liegt
	// sie unter der aktuellen Mindestlänge, signalisieren wir dem Client eine
	// Empfehlung zur Passwortänderung (nicht blockierend).
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		AccessToken               string `json:"access_token"`
		PasswordChangeRecommended bool   `json:"password_change_recommended,omitempty"`
	}{
		AccessToken:               accessToken,
		PasswordChangeRecommended: len([]rune(req.Password)) < h.passwordMinLength(),
	})
}

const lockTimeFormat = time.RFC3339

// accountLocked reports whether locked_until lies in the future.
func accountLocked(lockedUntil sql.NullString) bool {
	if !lockedUntil.Valid || lockedUntil.String == "" {
		return false
	}
	t, err := time.Parse(lockTimeFormat, lockedUntil.String)
	if err != nil {
		return false
	}
	return time.Now().Before(t)
}

// registerFailedLogin increments the failure counter and, once the configured
// threshold is reached, locks the account for the configured window (resetting
// the counter so the next window starts fresh). A non-positive LoginMaxFailures
// disables the lockout (e.g. in tests).
func (h *Handler) registerFailedLogin(ctx context.Context, userID, currentFailures int) {
	if h.cfg == nil || h.cfg.LoginMaxFailures <= 0 {
		return
	}
	next := currentFailures + 1
	if next >= h.cfg.LoginMaxFailures {
		until := time.Now().Add(time.Duration(h.cfg.LoginLockMinutes) * time.Minute).Format(lockTimeFormat)
		h.db.ExecContext(ctx,
			`UPDATE users SET failed_login_count = 0, locked_until = ? WHERE id = ?`, until, userID)
		return
	}
	h.db.ExecContext(ctx,
		`UPDATE users SET failed_login_count = ? WHERE id = ?`, next, userID)
}

// resetLoginFailures clears the failure counter and any lock after a successful login.
func (h *Handler) resetLoginFailures(ctx context.Context, userID int) {
	h.db.ExecContext(ctx,
		`UPDATE users SET failed_login_count = 0, locked_until = NULL WHERE id = ?`, userID)
}

// clearLegacyRefreshCookie löscht das vor f967335 unter Path=/api/auth gesetzte
// refresh_token-Cookie. Sonst sendet der Browser bei Folge-Requests beide
// Cookies — Go's r.Cookie liest das pfadspezifischere (alte, ungültige) zuerst
// und gibt 401 zurück.
func clearLegacyRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		Path:   "/api/auth",
		MaxAge: -1,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Vor f967335 lag das Cookie unter Path=/api/auth. Browser sendet das
	// pfadspezifischere alte zuerst, Go's r.Cookie liest nur das erste, und
	// ein bereits rotierter Wert führt zu 401. Hier räumen, damit auch der
	// 401-Pfad das Legacy-Cookie aus dem Browser entfernt.
	clearLegacyRefreshCookie(w)

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tokenHash := HashToken(cookie.Value)
	var id int
	var email, role string
	var expiresAt time.Time
	err = h.db.QueryRowContext(r.Context(),
		`SELECT u.id, COALESCE(NULLIF(u.email, ''), u.login_name, ''), u.role, rt.expires_at
		 FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id
		 WHERE rt.token_hash = ?`, tokenHash,
	).Scan(&id, &email, &role, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	accessToken, err := IssueAccessToken(h.jwtSecret, id, email, role, clubFunctions, isParent)
	if err != nil {
		slog.Error("refresh issue access token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	plain, newHash, err := GenerateOpaqueToken()
	if err != nil {
		slog.Error("refresh generate opaque token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newExpiry := RefreshTokenExpiry()

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		slog.Error("refresh begin tx failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash); err != nil {
		slog.Error("refresh delete failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, newHash, newExpiry,
	); err != nil {
		slog.Error("refresh insert failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		slog.Error("refresh commit failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    plain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  newExpiry,
		Path:     "/",
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		h.db.ExecContext(r.Context(),
			`DELETE FROM refresh_tokens WHERE token_hash = ?`, HashToken(cookie.Value))
	}
	clearLegacyRefreshCookie(w)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
		Path:     "/",
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestMembership(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Email       string `json:"email"`
		Comment     string `json:"comment"`
		IsChild     bool   `json:"is_child"`
		ParentEmail string `json:"parent_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Kinderaccount: keine eigene E-Mail, dafür Vor-/Nachname (für den Member)
	// und eine gültige verwaltende Eltern-E-Mail (Korrespondenz). Die NOT-NULL-
	// Spalte email wird mit der Eltern-Adresse gespiegelt.
	isChild := 0
	contactEmail := req.Email
	var parentEmailVal any
	if req.IsChild {
		if req.LastName == "" || !looksLikeEmail(req.ParentEmail) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		isChild = 1
		contactEmail = req.ParentEmail
		parentEmailVal = req.ParentEmail
	} else if !looksLikeEmail(req.Email) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var commentVal interface{}
	if req.Comment != "" {
		commentVal = req.Comment
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO membership_requests (first_name, last_name, email, comment, is_child, parent_email) VALUES (?,?,?,?,?,?)`,
		req.FirstName, req.LastName, contactEmail, commentVal, isChild, parentEmailVal,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newID, _ := res.LastInsertId()
	w.WriteHeader(http.StatusCreated)
	go func() {
		rows, err := h.db.Query(`SELECT id FROM users WHERE role = 'admin'`)
		if err != nil {
			return
		}
		defer rows.Close()
		var adminIDs []int
		for rows.Next() {
			var id int
			rows.Scan(&id)
			adminIDs = append(adminIDs, id)
		}
		notify.Send(h.db, h.cfg, adminIDs, "membership",
			"Neue Beitrittsanfrage",
			req.FirstName+" "+req.LastName+" möchte Mitglied werden",
			fmt.Sprintf("/anfragen?id=%d", newID))
	}()
}

func (h *Handler) ListMembershipRequests(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, first_name, last_name, email, COALESCE(comment,''), status, created_at,
		        is_child, COALESCE(parent_email,'')
		 FROM membership_requests WHERE status = 'pending' ORDER BY created_at`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type row struct {
		ID          int    `json:"id"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Email       string `json:"email"`
		Comment     string `json:"comment,omitempty"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
		IsChild     bool   `json:"is_child"`
		ParentEmail string `json:"parent_email,omitempty"`
	}
	results := []row{}
	for rows.Next() {
		var r row
		var isChild int
		rows.Scan(&r.ID, &r.FirstName, &r.LastName, &r.Email, &r.Comment, &r.Status, &r.CreatedAt, &isChild, &r.ParentEmail)
		r.IsChild = isChild == 1
		results = append(results, r)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) ApproveMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims := ClaimsFromCtx(r.Context())
	var firstName, lastName, email, parentEmail string
	var isChild int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name, email, is_child, COALESCE(parent_email,'')
		 FROM membership_requests WHERE id = ? AND status = 'pending'`, id,
	).Scan(&firstName, &lastName, &email, &isChild, &parentEmail)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if isChild == 1 {
		h.approveChildRequest(w, r, id, firstName, lastName, parentEmail, claims.UserID)
		return
	}

	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := InvitationExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at, first_name, last_name) VALUES (?,?,?,?,?,?,?)`,
		email, nil, "standard", tokenHash, expiry, firstName, lastName,
	); err != nil {
		slog.Error("approve membership insert token failed", "email", email, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE membership_requests SET status='approved', handled_by=?, handled_at=CURRENT_TIMESTAMP WHERE id=?`,
		claims.UserID, id,
	)
	link := fmt.Sprintf("%s/register?token=%s", h.baseURL, plain)
	if err := h.mailer.Send(email, "Deine Anmeldung bei TeamWERK wurde bestätigt",
		fmt.Sprintf("Hallo %s,\n\nDeine Anfrage wurde genehmigt. Registriere dich hier:\n%s\n\nDer Link ist 48 Stunden gültig.", firstName, link)); err != nil {
		slog.Error("approve membership send mail failed", "email", email, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// approveChildRequest legt für einen Kinderantrag (ohne E-Mail) in einer
// Transaktion ausschließlich ein Kinder-Konto (login_name, can_login=0) an
// und versendet einen Passwort-Setz-Link an die Eltern-Adresse. Es wird
// KEIN Member und KEIN family_link angelegt (reine Korrespondenz; das
// Mitglied wird ggf. später separat über die Mitgliederverwaltung erfasst).
func (h *Handler) approveChildRequest(w http.ResponseWriter, r *http.Request, reqID, firstName, lastName, parentEmail string, handledBy int) {
	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	loginName, err := generateUniqueLoginName(ctx, tx, firstName, lastName)
	if err != nil {
		slog.Error("approve membership child login name failed", "firstName", firstName, "lastName", lastName, "error", err)
		http.Error(w, "konnte keinen eindeutigen Spielernamen erzeugen", http.StatusInternalServerError)
		return
	}

	// Eltern-Adresse als recovery_email persistieren — sie bleibt damit als
	// Wiederherstellungs-/Korrespondenz-Adresse erhalten (Passwort-Reset später).
	var recoveryEmail any
	if parentEmail != "" {
		recoveryEmail = parentEmail
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (email, login_name, first_name, last_name, password, role, can_login, recovery_email) VALUES (NULL, ?, ?, ?, '', 'standard', 0, ?)`,
		loginName, firstName, lastName, recoveryEmail,
	)
	if err != nil {
		slog.Error("approve membership child insert user failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	userID, _ := res.LastInsertId()

	// Passwort-Setz-Token (48 h) — Eltern setzen das Passwort, das aktiviert das
	// Konto (can_login=1, siehe ResetPassword).
	plain, tokenHash, err := GenerateOpaqueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?,?,?)`,
		userID, tokenHash, InvitationExpiry(),
	); err != nil {
		slog.Error("approve membership child insert token failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE membership_requests SET status='approved', handled_by=?, handled_at=CURRENT_TIMESTAMP WHERE id=?`,
		handledBy, reqID,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ab hier sind die Daten committed: Mailfehler dürfen den Vorgang nicht zurückrollen.
	h.broadcastFinance(r.Context(), "users")
	link := fmt.Sprintf("%s/reset-password?token=%s", h.baseURL, plain)
	body := fmt.Sprintf("Hallo,\n\nder Account für %s %s wurde angelegt.\n\nLogin-Name (zum Einloggen): %s\n\nBitte setze jetzt das Passwort:\n%s\n\nDer Link ist 48 Stunden gültig.",
		firstName, lastName, loginName, link)
	if err := h.mailer.Send(parentEmail, "Kinder-Account bei TeamWERK – Passwort setzen", body); err != nil {
		slog.Error("approve membership child send mail failed", "email", parentEmail, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RejectMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims := ClaimsFromCtx(r.Context())
	var firstName, email string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, email FROM membership_requests WHERE id = ? AND status = 'pending'`, id,
	).Scan(&firstName, &email)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE membership_requests SET status='rejected', handled_by=?, handled_at=CURRENT_TIMESTAMP WHERE id=?`,
		claims.UserID, id,
	)
	if err := h.mailer.Send(email, "Deine Anmeldung bei TeamWERK",
		fmt.Sprintf("Hallo %s,\n\nLeider konnte deine Anfrage nicht bestätigt werden. Wende dich an den Vereinsvorstand.", firstName)); err != nil {
		slog.Error("reject membership send mail failed", "email", email, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Role    string `json:"role"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = RoleStandard
	}
	if req.Role != RoleAdmin && req.Role != RoleStandard && req.Role != RolePressTeam {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}
	caller := ClaimsFromCtx(r.Context())
	if req.Role == RoleAdmin && (caller == nil || caller.Role != RoleAdmin) {
		http.Error(w, "only admins can invite admins", http.StatusForbidden)
		return
	}
	var commentVal interface{}
	if req.Comment != "" {
		commentVal = req.Comment
	}
	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := InvitationExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at, comment) VALUES (?,?,?,?,?,?)`,
		req.Email, nil, req.Role, tokenHash, expiry, commentVal,
	); err != nil {
		slog.Error("invite insert token failed", "email", req.Email, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	link := fmt.Sprintf("%s/register?token=%s", h.baseURL, plain)
	body := fmt.Sprintf(`Hallo,

du wurdest zur internen Verwaltungsplattform von Team Stuttgart (TeamWERK) eingeladen.

Klicke auf den folgenden Link, um dein Konto zu erstellen:

%s

Der Link ist 48 Stunden gültig. Falls du diese Einladung nicht erwartet hast, kannst du sie ignorieren.

Viele Grüße
Team Stuttgart`, link)
	if err := h.mailer.Send(req.Email, "Einladung zu TeamWERK – Team Stuttgart", body); err != nil {
		slog.Error("invite send mail failed", "email", req.Email, "error", err)
		http.Error(w, "mail delivery failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token     string `json:"token"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Password = strings.TrimSpace(req.Password) // konsistent zum Login-Trim
	tokenHash := HashToken(req.Token)
	var id int
	var email, role, tokenFirstName, tokenLastName string
	var teamID sql.NullInt64
	var memberID sql.NullInt64
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, email, role, team_id, expires_at, first_name, last_name, member_id FROM invitation_tokens WHERE token = ? AND used_at IS NULL`,
		tokenHash,
	).Scan(&id, &email, &role, &teamID, &expiresAt, &tokenFirstName, &tokenLastName, &memberID)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	firstName := req.FirstName
	if firstName == "" {
		firstName = tokenFirstName
	}
	lastName := req.LastName
	if lastName == "" {
		lastName = tokenLastName
	}
	if firstName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.validatePassword(req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		slog.Error("register bcrypt failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO users (email, first_name, last_name, password, role, team_id) VALUES (?,?,?,?,?,?)`,
		email, firstName, lastName, string(hash), role, teamID,
	)
	if err != nil {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	}
	newUserID, _ := res.LastInsertId()
	h.db.ExecContext(r.Context(), `UPDATE invitation_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	if memberID.Valid {
		h.db.ExecContext(r.Context(),
			`UPDATE members SET user_id = ? WHERE id = ? AND user_id IS NULL`,
			newUserID, memberID.Int64)
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetTokenInfo(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	tokenHash := HashToken(token)
	var firstName, lastName string
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name, expires_at FROM invitation_tokens WHERE token = ? AND used_at IS NULL`,
		tokenHash,
	).Scan(&firstName, &lastName, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"first_name": firstName, "last_name": lastName})
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"` // E-Mail ODER login_name (Kinder ohne eigene E-Mail)
	}
	json.NewDecoder(r.Body).Decode(&req)
	req.Email = strings.TrimSpace(req.Email)
	var userID int
	var firstName, lastName, dest string
	// Lookup über die "AccountName"-Qualität (E-Mail bei Erwachsenen, login_name
	// bei Kindern). recovery_email ist NIE Lookup-Key — nur Ziel-Adresse.
	// Versand an die "Wiederherstellungs"-Qualität: eigene E-Mail, sonst recovery_email.
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, first_name, last_name, COALESCE(NULLIF(email,''), recovery_email, '')
		 FROM users WHERE (LOWER(email) = LOWER(?) OR LOWER(login_name) = LOWER(?)) AND can_login = 1`,
		req.Email, req.Email,
	).Scan(&userID, &firstName, &lastName, &dest)
	w.WriteHeader(http.StatusNoContent) // always same response (keine Enumeration)
	if err != nil || dest == "" {
		return // kein Treffer oder kein Ziel: kein (nutzloser) Token
	}
	// Konto-Drosselung: innerhalb des Cooldowns keine weitere Mail/keinen Token —
	// verhindert Mail-Bombing eines bekannten Kontos auch über mehrere IPs. Die
	// Antwort bleibt der gleiche 204 (keine Enumeration, kein Timing-Signal).
	if !h.forgotPasswordAllowed(req.Email) {
		return
	}
	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := PasswordResetExpiry()
	h.db.ExecContext(r.Context(),
		`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?,?,?)`,
		userID, tokenHash, expiry,
	)
	link := fmt.Sprintf("%s/reset-password?token=%s", h.baseURL, plain)
	fullName := firstName
	if lastName != "" {
		fullName += " " + lastName
	}
	h.mailer.Send(dest, "Passwort zurücksetzen – TeamWERK", //nolint:errcheck // best-effort; token is stored regardless
		fmt.Sprintf("Hallo %s,\n\nPasswort zurücksetzen:\n%s\n\nDer Link ist 1 Stunde gültig.", fullName, link))
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Password = strings.TrimSpace(req.Password) // konsistent zum Login-Trim
	tokenHash := HashToken(req.Token)
	var id, userID int
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, expires_at FROM password_reset_tokens WHERE token = ? AND used_at IS NULL`,
		tokenHash,
	).Scan(&id, &userID, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	if err := h.validatePassword(req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	// can_login=1 aktiviert Kinder-Konten (can_login=0) beim ersten Passwort-Setzen;
	// für bereits aktive Konten (normaler Reset) ist es ein No-op.
	h.db.ExecContext(r.Context(), `UPDATE users SET password=?, can_login=1 WHERE id=?`, string(hash), userID)
	h.db.ExecContext(r.Context(), `UPDATE password_reset_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	h.db.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE user_id=?`, userID)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	unlinked := r.URL.Query().Get("unlinked") == "1"
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var conditions []string
	var filterArgs []any
	if search != "" {
		conditions = append(conditions, "(u.first_name LIKE ? OR u.last_name LIKE ? OR u.email LIKE ?)")
		s := "%" + search + "%"
		filterArgs = append(filterArgs, s, s, s)
	}
	if unlinked {
		conditions = append(conditions, "m.id IS NULL AND fl.parent_user_id IS NULL")
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	const joins = ` FROM users u LEFT JOIN members m ON m.user_id = u.id LEFT JOIN (SELECT DISTINCT parent_user_id FROM family_links) fl ON fl.parent_user_id = u.id`

	var total int
	countArgs := append(filterArgs, []any{}...)
	if err := h.db.QueryRowContext(r.Context(), `SELECT COUNT(*)`+joins+whereClause, countArgs...).Scan(&total); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	query := `SELECT u.id, u.first_name, u.last_name, COALESCE(u.email,''), u.role, m.id, u.last_login_at, u.can_login, (fl.parent_user_id IS NOT NULL)` + joins + whereClause + ` ORDER BY u.last_name, u.first_name LIMIT ? OFFSET ?`
	queryArgs := append(filterArgs, limit, offset)
	rows, err := h.db.QueryContext(r.Context(), query, queryArgs...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type user struct {
		ID            int     `json:"id"`
		FirstName     string  `json:"first_name"`
		LastName      string  `json:"last_name"`
		Email         string  `json:"email"`
		Role          string  `json:"role"`
		MemberID      *int    `json:"member_id"`
		LastLoginAt   *string `json:"last_login_at"`
		Proxy         bool    `json:"proxy"`
		HasFamilyLink bool    `json:"has_family_link"`
	}
	result := []user{}
	for rows.Next() {
		var u user
		var memberID sql.NullInt64
		var lastLoginAt sql.NullString
		var canLogin, hasFamilyLink int
		rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role, &memberID, &lastLoginAt, &canLogin, &hasFamilyLink)
		if memberID.Valid {
			id := int(memberID.Int64)
			u.MemberID = &id
		}
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.String
		}
		u.Proxy = canLogin == 0
		u.HasFamilyLink = hasFamilyLink == 1
		result = append(result, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": result, "total": total})
}

// GET /api/users/picker — returns users visible to the caller for folder permission assignment.
// Admin/Vorstand: all users. Others: users reachable via user_accessible_teams (active season).
func (h *Handler) UsersPicker(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())

	type pickerUser struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var rows *sql.Rows
	var err error

	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id, first_name || ' ' || last_name FROM users ORDER BY first_name, last_name`)
	} else {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT DISTINCT u.id, u.first_name || ' ' || u.last_name AS name
			  FROM user_accessible_teams uat
			  JOIN seasons s ON s.id = uat.season_id AND s.is_active = 1
			  JOIN kader k ON k.team_id = uat.team_id AND k.season_id = s.id
			  JOIN kader_trainers kt ON kt.kader_id = k.id
			  JOIN members mt ON mt.id = kt.member_id
			  JOIN users u ON u.id = mt.user_id
			 WHERE uat.user_id = ?
			UNION
			SELECT DISTINCT u.id, u.first_name || ' ' || u.last_name
			  FROM user_accessible_teams uat
			  JOIN seasons s ON s.id = uat.season_id AND s.is_active = 1
			  JOIN kader k ON k.team_id = uat.team_id AND k.season_id = s.id
			  JOIN kader_members km ON km.kader_id = k.id
			  JOIN members mp ON mp.id = km.member_id
			  JOIN users u ON u.id = mp.user_id
			 WHERE uat.user_id = ?
			UNION
			SELECT DISTINCT u.id, u.first_name || ' ' || u.last_name
			  FROM user_accessible_teams uat
			  JOIN seasons s ON s.id = uat.season_id AND s.is_active = 1
			  JOIN kader k ON k.team_id = uat.team_id AND k.season_id = s.id
			  JOIN kader_members km ON km.kader_id = k.id
			  JOIN family_links fl ON fl.member_id = km.member_id
			  JOIN users u ON u.id = fl.parent_user_id
			 WHERE uat.user_id = ?
			ORDER BY 2`,
			claims.UserID, claims.UserID, claims.UserID)
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []pickerUser{}
	for rows.Next() {
		var u pickerUser
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			continue
		}
		result = append(result, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/impersonate/{userId}
func (h *Handler) Impersonate(w http.ResponseWriter, r *http.Request) {
	caller := ClaimsFromCtx(r.Context())
	targetID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if caller.UserID == targetID {
		http.Error(w, "cannot impersonate yourself", http.StatusBadRequest)
		return
	}
	// Identität NULL-sicher auflösen: Kinder-Accounts haben email=NULL und
	// authentifizieren über login_name (konsistent mit Login/Refresh).
	var ident, role, firstName, lastName string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(NULLIF(email,''), login_name, ''), role, first_name, last_name FROM users WHERE id = ?`, targetID,
	).Scan(&ident, &role, &firstName, &lastName)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if role == "admin" {
		http.Error(w, "cannot impersonate admin", http.StatusBadRequest)
		return
	}
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), targetID)
	accessToken, err := IssueAccessToken(h.jwtSecret, targetID, ident, role, clubFunctions, isParent)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	name := firstName
	if lastName != "" {
		name += " " + lastName
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"user":         map[string]any{"id": targetID, "name": name},
	})
}

// PUT /api/users/{id} — activate proxy account (set can_login=1 + email)
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		CanLogin *int    `json:"can_login"`
		Email    *string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var canLoginDB int
	if err := h.db.QueryRowContext(r.Context(), `SELECT can_login FROM users WHERE id = ?`, targetID).Scan(&canLoginDB); err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if req.CanLogin != nil && *req.CanLogin == 1 && canLoginDB == 0 {
		if req.Email == nil || *req.Email == "" {
			http.Error(w, "email required to activate account", http.StatusBadRequest)
			return
		}
		var conflict bool
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*)>0 FROM users WHERE LOWER(email)=LOWER(?) AND can_login=1 AND id != ?`,
			*req.Email, targetID).Scan(&conflict)
		if conflict {
			http.Error(w, "email already taken", http.StatusConflict)
			return
		}
		var firstName, lastName string
		h.db.QueryRowContext(r.Context(), `SELECT first_name, last_name FROM users WHERE id = ?`, targetID).Scan(&firstName, &lastName)
		if _, err := h.db.ExecContext(r.Context(),
			`UPDATE users SET can_login=1, email=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			*req.Email, targetID,
		); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		go func() {
			plain, tokenHash, err := GenerateOpaqueToken()
			if err != nil {
				return
			}
			h.db.ExecContext(r.Context(), //nolint:errcheck
				`INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?,?,?)`,
				targetID, tokenHash, PasswordResetExpiry(),
			)
			fullName := firstName
			if lastName != "" {
				fullName += " " + lastName
			}
			link := fmt.Sprintf("%s/reset-password?token=%s", h.baseURL, plain)
			h.mailer.Send(*req.Email, "Dein TeamWERK-Konto wurde aktiviert", //nolint:errcheck
				fmt.Sprintf("Hallo %s,\n\ndein Konto wurde aktiviert. Bitte setze jetzt dein Passwort:\n%s\n\nDer Link ist 1 Stunde gültig.", fullName, link))
		}()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.FirstName == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO users (email, first_name, last_name, password, role) VALUES (?,?,?,?,'standard')`,
		req.Email, req.FirstName, req.LastName, string(hash),
	)
	if err != nil {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	}
	newID, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": newID}) //nolint:errcheck
}

// PUT /api/users/{id}/recovery-email
// Admin/Vorstand setzen die Wiederherstellungs-E-Mail eines Kontos direkt, ohne
// Bestätigungs-Workflow. Escape-Hatch, wenn die alte Adresse nicht mehr existiert
// und der doppelte Bestätigungs-Loop deshalb nicht abschließbar ist.
func (h *Handler) SetRecoveryEmail(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var req struct {
		RecoveryEmail string `json:"recovery_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !looksLikeEmail(req.RecoveryEmail) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET recovery_email=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		req.RecoveryEmail, targetID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// Recovery-E-Mail eines Ziel-Nutzers geändert → Finance-Gruppe + Betroffener.
	h.broadcastFinance(r.Context(), "members", targetID)
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/users/{id}/role
func (h *Handler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	caller := ClaimsFromCtx(r.Context())
	targetID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Role != RoleAdmin && req.Role != RoleStandard && req.Role != RolePressTeam {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}
	if req.Role == RoleAdmin && caller.Role != RoleAdmin {
		http.Error(w, "only admins can assign admin role", http.StatusForbidden)
		return
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) > 0 FROM users WHERE id = ?`, targetID).Scan(&exists); err != nil || !exists {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `UPDATE users SET role = ? WHERE id = ?`, req.Role, targetID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	targetID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if claims.UserID == targetID {
		http.Error(w, "cannot delete your own account", http.StatusBadRequest)
		return
	}

	var exists bool
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) > 0 FROM users WHERE id = ?`, targetID).Scan(&exists)
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, q := range []string{
		`DELETE FROM refresh_tokens WHERE user_id = ?`,
		`DELETE FROM invitation_tokens WHERE used_at IS NULL AND LOWER(email) = LOWER((SELECT email FROM users WHERE id = ?))`,
		`DELETE FROM invitation_tokens WHERE member_id IN (SELECT id FROM members WHERE user_id = ?)`,
		`DELETE FROM password_reset_tokens WHERE user_id = ?`,
		`DELETE FROM family_links WHERE parent_user_id = ?`,
		`DELETE FROM duty_assignments WHERE user_id = ?`,
		`DELETE FROM duty_accounts WHERE user_id = ?`,
		`DELETE FROM member_change_drafts WHERE created_by_user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(r.Context(), q, targetID); err != nil {
			slog.Error("DeleteUser stmt failed", "user_id", targetID, "query", q, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.broadcastFinance(r.Context(), "users")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/invitations
func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT it.id, it.email, it.role, COALESCE(it.comment,''), it.expires_at,
		        it.member_id, COALESCE(m.first_name || ' ' || m.last_name, '')
		 FROM invitation_tokens it
		 LEFT JOIN members m ON m.id = it.member_id
		 WHERE it.used_at IS NULL
		 ORDER BY it.expires_at`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type invitation struct {
		ID         int    `json:"id"`
		Email      string `json:"email"`
		Role       string `json:"role"`
		Comment    string `json:"comment,omitempty"`
		ExpiresAt  string `json:"expires_at"`
		MemberID   *int   `json:"member_id"`
		MemberName string `json:"member_name,omitempty"`
	}
	result := []invitation{}
	for rows.Next() {
		var inv invitation
		var memberID sql.NullInt64
		var memberName string
		rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.Comment, &inv.ExpiresAt, &memberID, &memberName)
		if memberID.Valid {
			id := int(memberID.Int64)
			inv.MemberID = &id
			inv.MemberName = strings.TrimSpace(memberName)
		}
		result = append(result, inv)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DELETE /api/admin/invitations/{id}
func (h *Handler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM invitation_tokens WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/membership-requests/{id}
func (h *Handler) DeleteMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM membership_requests WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/profile/account
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	var firstName, lastName string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name FROM users WHERE id=?`, claims.UserID,
	).Scan(&firstName, &lastName); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      claims.Email,
	})
}

// PUT /api/profile/account
func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET first_name=?, last_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		req.FirstName, req.LastName, claims.UserID,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/profile/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var hash string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT password FROM users WHERE id=?`, claims.UserID,
	).Scan(&hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.CurrentPassword)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.validatePassword(req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.db.ExecContext(r.Context(), `UPDATE users SET password=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, string(newHash), claims.UserID)
	h.db.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE user_id=?`, claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/profile/email
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	var req struct {
		NewEmail string `json:"new_email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewEmail == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var hash string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT password FROM users WHERE id=?`, claims.UserID,
	).Scan(&hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var exists bool
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*)>0 FROM users WHERE email=? AND can_login=1`, req.NewEmail).Scan(&exists)
	if exists {
		http.Error(w, "email already taken", http.StatusConflict)
		return
	}
	h.db.ExecContext(r.Context(), `DELETE FROM email_change_tokens WHERE user_id=?`, claims.UserID)
	plain, tokenHash, err := GenerateOpaqueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiry := time.Now().Add(24 * time.Hour)
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO email_change_tokens (user_id, token, new_email, expires_at) VALUES (?,?,?,?)`,
		claims.UserID, tokenHash, req.NewEmail, expiry,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	link := fmt.Sprintf("%s/api/profile/email/confirm?token=%s", h.baseURL, plain)
	body := fmt.Sprintf("Hallo,\n\nbitte bestätige deine neue E-Mail-Adresse für TeamWERK:\n\n%s\n\nDer Link ist 24 Stunden gültig.\n\nFalls du diese Änderung nicht beantragt hast, ignoriere diese Mail.", link)
	if err := h.mailer.Send(req.NewEmail, "E-Mail-Adresse bestätigen – TeamWERK", body); err != nil {
		slog.Error("email change send mail failed", "email", req.NewEmail, "error", err)
		h.db.ExecContext(r.Context(), `DELETE FROM email_change_tokens WHERE user_id=?`, claims.UserID)
		http.Error(w, "mail delivery failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/profile/email/confirm?token=xyz
func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	plain := r.URL.Query().Get("token")
	if plain == "" {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
		return
	}
	tokenHash := HashToken(plain)
	var id, userID int
	var newEmail string
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, new_email, expires_at FROM email_change_tokens WHERE token=? AND used_at IS NULL AND field='email'`,
		tokenHash,
	).Scan(&id, &userID, &newEmail, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
		return
	}
	h.db.ExecContext(r.Context(), `UPDATE users SET email=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newEmail, userID)
	h.db.ExecContext(r.Context(), `UPDATE email_change_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	h.db.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE user_id=?`, userID)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// POST /api/profile/kind/{memberId}/recovery-email
// Eltern stoßen die Änderung der Wiederherstellungs-E-Mail eines Kindkontos an.
// Doppelte Bestätigung: zuerst an die ALTE Adresse (Autorisierung), dann an die
// NEUE Adresse (Erreichbarkeit). Diese Route legt nur die Stufe 'auth' an.
func (h *Handler) RequestRecoveryEmailChange(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var req struct {
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !looksLikeEmail(req.NewEmail) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Caller muss verknüpftes Elternteil sein.
	var isParent int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		claims.UserID, memberID).Scan(&isParent)
	if isParent == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// Kindkonto + aktuelle (alte) Adresse ermitteln.
	var childUserID sql.NullInt64
	var oldEmail string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT m.user_id, COALESCE(u.recovery_email,'')
		   FROM members m LEFT JOIN users u ON u.id = m.user_id WHERE m.id=?`,
		memberID).Scan(&childUserID, &oldEmail); err != nil || !childUserID.Valid {
		http.Error(w, "kein Konto für dieses Kind", http.StatusConflict)
		return
	}
	if oldEmail == "" {
		// Keine alte Adresse → Bestätigungs-Loop nicht möglich. Vorstand-Override nötig.
		http.Error(w, "keine Wiederherstellungs-Adresse hinterlegt", http.StatusConflict)
		return
	}
	h.db.ExecContext(r.Context(),
		`DELETE FROM email_change_tokens WHERE user_id=? AND field='recovery_email'`, childUserID.Int64)
	plain, tokenHash, err := GenerateOpaqueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO email_change_tokens (user_id, token, new_email, expires_at, field, stage) VALUES (?,?,?,?, 'recovery_email', 'auth')`,
		childUserID.Int64, tokenHash, req.NewEmail, time.Now().Add(24*time.Hour),
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	link := fmt.Sprintf("%s/api/profile/recovery-email/confirm?token=%s", h.baseURL, plain)
	body := fmt.Sprintf("Hallo,\n\nfür ein Kinderkonto bei TeamWERK wurde beantragt, die Wiederherstellungs-E-Mail auf %s zu ändern.\n\nBitte bestätige die Änderung über diesen Link:\n%s\n\nDanach wird ein zweiter Bestätigungslink an die neue Adresse gesendet. Der Link ist 24 Stunden gültig.\n\nFalls du das nicht beantragt hast, ignoriere diese Mail — es ändert sich nichts.", req.NewEmail, link)
	if err := h.mailer.Send(oldEmail, "Wiederherstellungs-E-Mail ändern – TeamWERK", body); err != nil {
		slog.Error("recovery email change send mail failed", "email", oldEmail, "error", err)
		h.db.ExecContext(r.Context(),
			`DELETE FROM email_change_tokens WHERE user_id=? AND field='recovery_email'`, childUserID.Int64)
		http.Error(w, "mail delivery failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/profile/recovery-email/confirm?token=...
// Stufe 'auth' (alte Adresse bestätigt) → Token rotiert auf 'verify', Mail an
// neue Adresse. Stufe 'verify' (neue Adresse bestätigt) → recovery_email wird
// geschrieben.
func (h *Handler) ConfirmRecoveryEmailChange(w http.ResponseWriter, r *http.Request) {
	plain := r.URL.Query().Get("token")
	if plain == "" {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
		return
	}
	tokenHash := HashToken(plain)
	var id, userID int
	var newEmail, stage string
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, new_email, expires_at, COALESCE(stage,'') FROM email_change_tokens
		 WHERE token=? AND used_at IS NULL AND field='recovery_email'`,
		tokenHash,
	).Scan(&id, &userID, &newEmail, &expiresAt, &stage)
	if err != nil || time.Now().After(expiresAt) {
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
		return
	}
	switch stage {
	case "auth":
		// Alte Adresse bestätigt → Token rotieren, zweite Bestätigung an neue Adresse.
		plain2, hash2, gerr := GenerateOpaqueToken()
		if gerr != nil {
			http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
			return
		}
		h.db.ExecContext(r.Context(),
			`UPDATE email_change_tokens SET token=?, stage='verify', expires_at=?, created_at=CURRENT_TIMESTAMP WHERE id=?`,
			hash2, time.Now().Add(24*time.Hour), id)
		link := fmt.Sprintf("%s/api/profile/recovery-email/confirm?token=%s", h.baseURL, plain2)
		body := fmt.Sprintf("Hallo,\n\nbitte bestätige, dass diese Adresse künftig die Wiederherstellungs-E-Mail für das TeamWERK-Kinderkonto sein soll:\n\n%s\n\nDer Link ist 24 Stunden gültig.", link)
		h.mailer.Send(newEmail, "Neue Wiederherstellungs-E-Mail bestätigen – TeamWERK", body) //nolint:errcheck // best-effort
		http.Redirect(w, r, "/login?info=recovery_verify_sent", http.StatusFound)
	case "verify":
		// Neue Adresse bestätigt → schreiben.
		h.db.ExecContext(r.Context(), `UPDATE users SET recovery_email=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newEmail, userID)
		h.db.ExecContext(r.Context(), `UPDATE email_change_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
		// Selbstbedienung (Kinderkonto-Recovery) → Finance-Gruppe + Betroffener.
		h.broadcastFinance(r.Context(), "members", userID)
		http.Redirect(w, r, "/login?info=recovery_changed", http.StatusFound)
	default:
		http.Redirect(w, r, "/login?error=invalid_token", http.StatusFound)
	}
}

// loadJWTExtras queries club_functions and is_parent for inclusion in the JWT.
func (h *Handler) loadJWTExtras(ctx context.Context, userID int) ([]string, bool) {
	var functionsStr string
	h.db.QueryRowContext(ctx,
		`SELECT COALESCE(GROUP_CONCAT(mcf.function, ','), '')
		 FROM member_club_functions mcf
		 JOIN members m ON m.id = mcf.member_id
		 WHERE m.user_id = ?`, userID,
	).Scan(&functionsStr)

	var clubFunctions []string
	if functionsStr != "" {
		clubFunctions = strings.Split(functionsStr, ",")
	} else {
		clubFunctions = []string{}
	}

	var isParent bool
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) > 0 FROM family_links WHERE parent_user_id = ?`, userID,
	).Scan(&isParent)

	return clubFunctions, isParent
}

// POST /api/admin/invitations/import-csv
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		http.Error(w, "invalid csv", http.StatusBadRequest)
		return
	}
	emailIdx, email2Idx := -1, -1
	for i, h := range header {
		switch strings.TrimSpace(h) {
		case "Email":
			emailIdx = i
		case "Email 2":
			email2Idx = i
		}
	}
	if emailIdx == -1 {
		http.Error(w, "column 'Email' not found", http.StatusBadRequest)
		return
	}

	seen := map[string]bool{}
	var candidates []string
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		for _, idx := range []int{emailIdx, email2Idx} {
			if idx < 0 || idx >= len(record) {
				continue
			}
			e := strings.ToLower(strings.TrimSpace(record[idx]))
			if e == "" || seen[e] {
				continue
			}
			seen[e] = true
			candidates = append(candidates, e)
		}
	}

	created, skipped := 0, 0
	expiry := time.Now().Add(30 * 24 * time.Hour)
	for _, email := range candidates {
		var exists bool
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) > 0 FROM users WHERE LOWER(email) = ? AND can_login = 1`, email,
		).Scan(&exists)
		if exists {
			skipped++
			continue
		}
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) > 0 FROM invitation_tokens WHERE LOWER(email) = ? AND used_at IS NULL`, email,
		).Scan(&exists)
		if exists {
			skipped++
			continue
		}
		_, tokenHash, err := GenerateOpaqueToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if _, err := h.db.ExecContext(r.Context(),
			`INSERT INTO invitation_tokens (email, role, token, expires_at) VALUES (?,?,?,?)`,
			email, "standard", tokenHash, expiry,
		); err != nil {
			skipped++
			continue
		}
		created++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"created": created, "skipped": skipped})
}

// POST /api/admin/invitations/{id}/send
func (h *Handler) SendInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var email, plain string
	var tokenHash string
	var existingHash string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT email, token FROM invitation_tokens WHERE id = ? AND used_at IS NULL`, id,
	).Scan(&email, &existingHash)
	if err == sql.ErrNoRows {
		http.Error(w, "invitation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	plain, tokenHash, err = GenerateOpaqueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiry := InvitationExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE invitation_tokens SET token = ?, expires_at = ? WHERE id = ?`,
		tokenHash, expiry, id,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	link := fmt.Sprintf("%s/register?token=%s", h.baseURL, plain)
	body := fmt.Sprintf(`Hallo,

du wurdest zur internen Verwaltungsplattform von Team Stuttgart (TeamWERK) eingeladen.

Klicke auf den folgenden Link, um dein Konto zu erstellen:

%s

Der Link ist 48 Stunden gültig. Falls du diese Einladung nicht erwartet hast, kannst du sie ignorieren.

Viele Grüße
Team Stuttgart`, link)
	if err := h.mailer.Send(email, "Einladung zu TeamWERK – Team Stuttgart", body); err != nil {
		slog.Error("send invitation send mail failed", "email", email, "error", err)
		http.Error(w, "mail delivery failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/invitations/{id}/member
func (h *Handler) LinkInvitationMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		MemberID *int `json:"member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var exists bool
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) > 0 FROM invitation_tokens WHERE id = ? AND used_at IS NULL`, id,
	).Scan(&exists)
	if !exists {
		http.Error(w, "invitation not found", http.StatusNotFound)
		return
	}

	if req.MemberID != nil {
		var alreadyLinked bool
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) > 0 FROM members WHERE id = ? AND user_id IS NOT NULL`, *req.MemberID,
		).Scan(&alreadyLinked)
		if alreadyLinked {
			http.Error(w, "member already linked to a user", http.StatusConflict)
			return
		}
		if _, err := h.db.ExecContext(r.Context(),
			`UPDATE invitation_tokens SET member_id = ? WHERE id = ?`, *req.MemberID, id,
		); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := h.db.ExecContext(r.Context(),
			`UPDATE invitation_tokens SET member_id = NULL WHERE id = ?`, id,
		); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// principalFromClaims converts Claims to a policy.Principal.
func principalFromClaims(c *Claims) *policy.Principal {
	return &policy.Principal{
		UserID:        c.UserID,
		Role:          c.Role,
		ClubFunctions: c.ClubFunctions,
		IsParent:      c.IsParent,
	}
}

// GET /api/me — returns the authenticated user's identity, capabilities, and nav items.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	p := principalFromClaims(claims)

	resp := struct {
		User struct {
			ID    int    `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
		Capabilities []string         `json:"capabilities"`
		Nav          []policy.NavItem `json:"nav"`
	}{}
	resp.User.ID = claims.UserID
	resp.User.Email = claims.Email
	resp.User.Role = claims.Role
	resp.Capabilities = policy.Capabilities(p)
	resp.Nav = policy.NavFor(p)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
