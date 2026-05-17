package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/teamstuttgart/vereinswerk/internal/mailer"
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
		`SELECT id, password, role FROM users WHERE email = ?`, req.Email,
	).Scan(&id, &hash, &role)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	accessToken, err := IssueAccessToken(h.jwtSecret, id, req.Email, role)
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

	accessToken, _ := IssueAccessToken(h.jwtSecret, id, email, role)
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
		Name   string `json:"name"`
		Email  string `json:"email"`
		TeamID *int   `json:"team_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Email == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO membership_requests (name, email, team_id) VALUES (?,?,?)`,
		req.Name, req.Email, req.TeamID,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if req.TeamID != nil {
		h.notifyTrainersOfRequest(r, *req.TeamID, req.Name, req.Email)
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) ListMembershipRequests(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	var rows *sql.Rows
	var err error
	if claims.Role == "admin" {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id, name, email, team_id, status, created_at FROM membership_requests WHERE status = 'pending' ORDER BY created_at`)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT mr.id, mr.name, mr.email, mr.team_id, mr.status, mr.created_at
			 FROM membership_requests mr
			 JOIN team_trainers tt ON tt.team_id = mr.team_id
			 WHERE tt.user_id = ? AND mr.status = 'pending'
			 ORDER BY mr.created_at`, claims.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type row struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		TeamID    *int   `json:"team_id"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	var results []row
	for rows.Next() {
		var r row
		var teamID sql.NullInt64
		rows.Scan(&r.ID, &r.Name, &r.Email, &teamID, &r.Status, &r.CreatedAt)
		if teamID.Valid {
			n := int(teamID.Int64)
			r.TeamID = &n
		}
		results = append(results, r)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) ApproveMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims := ClaimsFromCtx(r.Context())
	var name, email string
	var teamID sql.NullInt64
	err := h.db.QueryRowContext(r.Context(),
		`SELECT name, email, team_id FROM membership_requests WHERE id = ? AND status = 'pending'`, id,
	).Scan(&name, &email, &teamID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := InvitationExpiry()
	var teamIDVal *int64
	if teamID.Valid {
		teamIDVal = &teamID.Int64
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at) VALUES (?,?,?,?,?)`,
		email, teamIDVal, "elternteil", tokenHash, expiry,
	)
	h.db.ExecContext(r.Context(),
		`UPDATE membership_requests SET status='approved', handled_by=?, handled_at=CURRENT_TIMESTAMP WHERE id=?`,
		claims.UserID, id,
	)
	link := fmt.Sprintf("%s/register?token=%s", h.baseURL, plain)
	h.mailer.Send(email, "Deine Anmeldung bei VereinsWerk wurde bestätigt",
		fmt.Sprintf("Hallo %s,\n\nDeine Anfrage wurde genehmigt. Registriere dich hier:\n%s\n\nDer Link ist 48 Stunden gültig.", name, link))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RejectMembershipRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims := ClaimsFromCtx(r.Context())
	var name, email string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT name, email FROM membership_requests WHERE id = ? AND status = 'pending'`, id,
	).Scan(&name, &email)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE membership_requests SET status='rejected', handled_by=?, handled_at=CURRENT_TIMESTAMP WHERE id=?`,
		claims.UserID, id,
	)
	h.mailer.Send(email, "Deine Anmeldung bei VereinsWerk",
		fmt.Sprintf("Hallo %s,\n\nLeider konnte deine Anfrage nicht bestätigt werden. Wende dich an den Vereinsvorstand.", name))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		TeamID int    `json:"team_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "elternteil"
	}
	plain, tokenHash, _ := GenerateOpaqueToken()
	expiry := InvitationExpiry()
	h.db.ExecContext(r.Context(),
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at) VALUES (?,?,?,?,?)`,
		req.Email, req.TeamID, req.Role, tokenHash, expiry,
	)
	link := fmt.Sprintf("%s/register?token=%s", h.baseURL, plain)
	h.mailer.Send(req.Email, "Einladung zu VereinsWerk – Team Stuttgart",
		fmt.Sprintf("Du wurdest eingeladen! Registriere dich hier:\n%s\n\nDer Link ist 48 Stunden gültig.", link))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
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
		`INSERT INTO users (email, name, password, role, team_id) VALUES (?,?,?,?,?)`,
		email, req.Name, string(hash), role, teamID,
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
	var name string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name FROM users WHERE email = ?`, req.Email,
	).Scan(&userID, &name)
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
	h.mailer.Send(req.Email, "Passwort zurücksetzen – VereinsWerk",
		fmt.Sprintf("Hallo %s,\n\nPasswort zurücksetzen:\n%s\n\nDer Link ist 1 Stunde gültig.", name, link))
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
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, name, email, role FROM users ORDER BY name`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type user struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	var result []user
	for rows.Next() {
		var u user
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role)
		result = append(result, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) notifyTrainersOfRequest(r *http.Request, teamID int, name, email string) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT u.email FROM users u JOIN team_trainers tt ON tt.user_id = u.id WHERE tt.team_id = ? AND u.role IN ('trainer','admin')`,
		teamID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var trainerEmail string
		rows.Scan(&trainerEmail)
		h.mailer.Send(trainerEmail, "Neuer Beitrittsantrag – VereinsWerk",
			fmt.Sprintf("Neuer Antrag von %s (%s).\nBitte in VereinsWerk prüfen: %s/admin/membership-requests", name, email, h.baseURL))
	}
}
