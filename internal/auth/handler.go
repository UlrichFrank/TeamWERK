package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db        *sql.DB
	jwtSecret string
	mailer    *mailer.Mailer
	baseURL   string
}

func NewHandler(db *sql.DB, jwtSecret string, m *mailer.Mailer, baseURL string) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret, mailer: m, baseURL: baseURL}
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
	var id int
	var hash, role string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, password, role FROM users WHERE LOWER(email) = LOWER(?)`, req.Email,
	).Scan(&id, &hash, &role)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	accessToken, err := IssueAccessToken(h.jwtSecret, id, req.Email, role, clubFunctions, isParent)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	plain, tokenHash, err := GenerateOpaqueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiry := RefreshTokenExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, tokenHash, expiry,
	); err != nil {
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
		Path:     "/api/auth",
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
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
		`SELECT u.id, u.email, u.role, rt.expires_at
		 FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id
		 WHERE rt.token_hash = ?`, tokenHash,
	).Scan(&id, &email, &role, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	h.db.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash)

	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	accessToken, _ := IssueAccessToken(h.jwtSecret, id, email, role, clubFunctions, isParent)
	plain, newHash, _ := GenerateOpaqueToken()
	newExpiry := RefreshTokenExpiry()
	h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, newHash, newExpiry,
	)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    plain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  newExpiry,
		Path:     "/api/auth",
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		h.db.ExecContext(r.Context(),
			`DELETE FROM refresh_tokens WHERE token_hash = ?`, HashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
		Path:     "/api/auth",
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestMembership(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Comment   string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" || req.Email == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var commentVal interface{}
	if req.Comment != "" {
		commentVal = req.Comment
	}
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO membership_requests (first_name, last_name, email, comment) VALUES (?,?,?,?)`,
		req.FirstName, req.LastName, req.Email, commentVal,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) ListMembershipRequests(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, first_name, last_name, email, COALESCE(comment,''), status, created_at FROM membership_requests WHERE status = 'pending' ORDER BY created_at`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type row struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Comment   string `json:"comment,omitempty"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	results := []row{}
	for rows.Next() {
		var r row
		rows.Scan(&r.ID, &r.FirstName, &r.LastName, &r.Email, &r.Comment, &r.Status, &r.CreatedAt)
		results = append(results, r)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) ApproveMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims := ClaimsFromCtx(r.Context())
	var firstName, lastName, email string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name, email FROM membership_requests WHERE id = ? AND status = 'pending'`, id,
	).Scan(&firstName, &lastName, &email)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := InvitationExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at) VALUES (?,?,?,?,?)`,
		email, nil, "standard", tokenHash, expiry,
	); err != nil {
		log.Printf("DB ERROR (ApproveMembership token for %s): %v", email, err)
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
		log.Printf("SMTP ERROR (ApproveMembership to %s): %v", email, err)
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
		log.Printf("SMTP ERROR (RejectMembership to %s): %v", email, err)
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
		req.Role = "standard"
	}
	if req.Role != "admin" && req.Role != "standard" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}
	caller := ClaimsFromCtx(r.Context())
	if req.Role == "admin" && (caller == nil || caller.Role != "admin") {
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
		log.Printf("DB ERROR (Invite for %s): %v", req.Email, err)
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
		log.Printf("SMTP ERROR (Invite to %s): %v", req.Email, err)
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
	tokenHash := HashToken(req.Token)
	var id int
	var email, role string
	var teamID sql.NullInt64
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, email, role, team_id, expires_at FROM invitation_tokens WHERE token = ? AND used_at IS NULL`,
		tokenHash,
	).Scan(&id, &email, &role, &teamID, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO users (email, first_name, last_name, password, role, team_id) VALUES (?,?,?,?,?,?)`,
		email, req.FirstName, req.LastName, string(hash), role, teamID,
	)
	if err != nil {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	}
	_ = res
	h.db.ExecContext(r.Context(), `UPDATE invitation_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var userID int
	var firstName, lastName string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, first_name, last_name FROM users WHERE email = ?`, req.Email,
	).Scan(&userID, &firstName, &lastName)
	w.WriteHeader(http.StatusNoContent) // always same response
	if err != nil {
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
	h.mailer.Send(req.Email, "Passwort zurücksetzen – TeamWERK",
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
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	h.db.ExecContext(r.Context(), `UPDATE users SET password=? WHERE id=?`, string(hash), userID)
	h.db.ExecContext(r.Context(), `UPDATE password_reset_tokens SET used_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	h.db.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE user_id=?`, userID)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	searchFilter := ""
	if search != "" {
		searchFilter = ` WHERE u.first_name LIKE ? OR u.last_name LIKE ? OR u.email LIKE ?`
	}

	countQuery := `SELECT COUNT(*) FROM users u` + searchFilter
	var total int
	if search != "" {
		err := h.db.QueryRowContext(r.Context(), countQuery, "%"+search+"%", "%"+search+"%", "%"+search+"%").Scan(&total)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		err := h.db.QueryRowContext(r.Context(), countQuery).Scan(&total)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.role, m.id
		FROM users u LEFT JOIN members m ON m.user_id = u.id` + searchFilter + ` ORDER BY u.last_name, u.first_name LIMIT ? OFFSET ?`
	var rows *sql.Rows
	var err error
	if search != "" {
		rows, err = h.db.QueryContext(r.Context(), query, "%"+search+"%", "%"+search+"%", "%"+search+"%", limit, offset)
	} else {
		rows, err = h.db.QueryContext(r.Context(), query, limit, offset)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type user struct {
		ID        int  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		MemberID  *int   `json:"member_id"`
	}
	result := []user{}
	for rows.Next() {
		var u user
		var memberID sql.NullInt64
		rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role, &memberID)
		if memberID.Valid {
			id := int(memberID.Int64)
			u.MemberID = &id
		}
		result = append(result, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": result, "total": total})
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
	if req.Role != "admin" && req.Role != "standard" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}
	if req.Role == "admin" && caller.Role != "admin" {
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
		`DELETE FROM invitation_tokens WHERE email = (SELECT email FROM users WHERE id = ?)`,
		`DELETE FROM password_reset_tokens WHERE user_id = ?`,
		`DELETE FROM family_links WHERE parent_user_id = ?`,
		`DELETE FROM duty_assignments WHERE user_id = ?`,
		`DELETE FROM duty_accounts WHERE user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(r.Context(), q, targetID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/invitations
func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, email, role, COALESCE(comment,''), expires_at
		 FROM invitation_tokens
		 WHERE used_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY expires_at`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type invitation struct {
		ID        int    `json:"id"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Comment   string `json:"comment,omitempty"`
		ExpiresAt string `json:"expires_at"`
	}
	result := []invitation{}
	for rows.Next() {
		var inv invitation
		rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.Comment, &inv.ExpiresAt)
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
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
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
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*)>0 FROM users WHERE email=?`, req.NewEmail).Scan(&exists)
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
		log.Printf("SMTP ERROR (EmailChange to %s): %v", req.NewEmail, err)
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
		`SELECT id, user_id, new_email, expires_at FROM email_change_tokens WHERE token=? AND used_at IS NULL`,
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

func (h *Handler) notifyTrainersOfRequest(r *http.Request, teamID int, name, email string) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT u.email FROM users u
		 JOIN team_trainers tt ON tt.user_id = u.id
		 JOIN members m ON m.user_id = u.id
		 JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'trainer'
		 WHERE tt.team_id = ?
		 UNION
		 SELECT u2.email FROM users u2 WHERE u2.role = 'admin'`,
		teamID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var trainerEmail string
		rows.Scan(&trainerEmail)
		h.mailer.Send(trainerEmail, "Neuer Beitrittsantrag – TeamWERK",
			fmt.Sprintf("Neuer Antrag von %s (%s).\nBitte in TeamWERK prüfen: %s/admin/membership-requests", name, email, h.baseURL))
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
