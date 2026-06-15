package auth

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db        *sql.DB
	cfg       *appconfig.Config
	jwtSecret string
	mailer    *mailer.Mailer
	baseURL   string
}

func NewHandler(db *sql.DB, cfg *appconfig.Config, jwtSecret string, m *mailer.Mailer, baseURL string) *Handler {
	return &Handler{db: db, cfg: cfg, jwtSecret: jwtSecret, mailer: m, baseURL: baseURL}
}

// dummyHash is a pre-computed bcrypt hash used in the login ErrNoRows branch to
// perform a constant-time dummy comparison, preventing timing-based email enumeration.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("teamwerk-dummy-password-for-timing"), bcrypt.DefaultCost)

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
		`SELECT id, password, role FROM users WHERE LOWER(email) = LOWER(?) AND can_login = 1`, req.Email,
	).Scan(&id, &hash, &role)
	if err == sql.ErrNoRows {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password)) //nolint:errcheck
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Login query error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	h.db.ExecContext(r.Context(), `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	log.Printf("Login: loadJWTExtras done - clubFunctions=%v, isParent=%v", clubFunctions, isParent)
	accessToken, err := IssueAccessToken(h.jwtSecret, id, req.Email, role, clubFunctions, isParent)
	if err != nil {
		log.Printf("Login: IssueAccessToken error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	plain, tokenHash, err := GenerateOpaqueToken()
	log.Printf("Login: GenerateOpaqueToken done")
	if err != nil {
		log.Printf("Login: GenerateOpaqueToken error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiry := RefreshTokenExpiry()
	if _, err := h.db.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, tokenHash, expiry,
	); err != nil {
		log.Printf("Login: INSERT refresh_tokens error: %v", err)
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
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), id)
	accessToken, err := IssueAccessToken(h.jwtSecret, id, email, role, clubFunctions, isParent)
	if err != nil {
		log.Printf("Refresh: IssueAccessToken error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	plain, newHash, err := GenerateOpaqueToken()
	if err != nil {
		log.Printf("Refresh: GenerateOpaqueToken error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newExpiry := RefreshTokenExpiry()

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("Refresh: BeginTx error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(r.Context(), `DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash); err != nil {
		log.Printf("Refresh: DELETE error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		id, newHash, newExpiry,
	); err != nil {
		log.Printf("Refresh: INSERT error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("Refresh: Commit error: %v", err)
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
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO membership_requests (first_name, last_name, email, comment) VALUES (?,?,?,?)`,
		req.FirstName, req.LastName, req.Email, commentVal,
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
			fmt.Sprintf("/admin/mitgliedschaft?id=%d", newID))
	}()
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
		`INSERT INTO invitation_tokens (email, team_id, role, token, expires_at, first_name, last_name) VALUES (?,?,?,?,?,?,?)`,
		email, nil, "standard", tokenHash, expiry, firstName, lastName,
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
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Register: bcrypt error: %v", err)
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
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var userID int
	var firstName, lastName string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, first_name, last_name FROM users WHERE email = ? AND can_login = 1`, req.Email,
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
	h.mailer.Send(req.Email, "Passwort zurücksetzen – TeamWERK", //nolint:errcheck // best-effort; token is stored regardless
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
	var email, role, firstName, lastName string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT email, role, first_name, last_name FROM users WHERE id = ?`, targetID,
	).Scan(&email, &role, &firstName, &lastName)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if role == "admin" {
		http.Error(w, "cannot impersonate admin", http.StatusBadRequest)
		return
	}
	clubFunctions, isParent := h.loadJWTExtras(r.Context(), targetID)
	accessToken, err := IssueAccessToken(h.jwtSecret, targetID, email, role, clubFunctions, isParent)
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
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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
		`DELETE FROM invitation_tokens WHERE used_at IS NULL AND LOWER(email) = LOWER((SELECT email FROM users WHERE id = ?))`,
		`DELETE FROM invitation_tokens WHERE member_id IN (SELECT id FROM members WHERE user_id = ?)`,
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
		ID         int     `json:"id"`
		Email      string  `json:"email"`
		Role       string  `json:"role"`
		Comment    string  `json:"comment,omitempty"`
		ExpiresAt  string  `json:"expires_at"`
		MemberID   *int    `json:"member_id"`
		MemberName string  `json:"member_name,omitempty"`
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
		log.Printf("SMTP ERROR (SendInvitation to %s): %v", email, err)
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
