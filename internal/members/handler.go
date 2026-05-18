package members

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type Member struct {
	ID           int     `json:"id"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	DateOfBirth  string  `json:"date_of_birth,omitempty"`
	MemberNumber string  `json:"member_number,omitempty"`
	PassNumber   string  `json:"pass_number,omitempty"`
	JerseyNumber *int    `json:"jersey_number,omitempty"`
	Position     string  `json:"position,omitempty"`
	Status       string  `json:"status"`
	UserID       *int    `json:"user_id,omitempty"`
}

// GET /api/members
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var rows *sql.Rows
	var err error
	if claims.Role == "admin" {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id, first_name, last_name, COALESCE(date_of_birth,''), COALESCE(member_number,''), COALESCE(pass_number,''),
			        jersey_number, COALESCE(position,''), status, user_id
			 FROM members WHERE status != 'ausgetreten' ORDER BY last_name, first_name`)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
			        m.jersey_number, COALESCE(m.position,''), m.status, m.user_id
			 FROM members m
			 JOIN team_memberships tm ON tm.member_id = m.id
			 JOIN team_trainers tt ON tt.team_id = tm.team_id
			 WHERE tt.user_id = ? AND m.status != 'ausgetreten'
			 ORDER BY m.last_name, m.first_name`, claims.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	result := []Member{}
	for rows.Next() {
		var m Member
		var jerseyNum sql.NullInt64
		var userID sql.NullInt64
		rows.Scan(&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth, &m.MemberNumber, &m.PassNumber,
			&jerseyNum, &m.Position, &m.Status, &userID)
		if jerseyNum.Valid {
			n := int(jerseyNum.Int64)
			m.JerseyNumber = &n
		}
		if userID.Valid {
			n := int(userID.Int64)
			m.UserID = &n
		}
		result = append(result, m)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/members
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		DateOfBirth  string `json:"date_of_birth"`
		MemberNumber string `json:"member_number"`
		PassNumber   string `json:"pass_number"`
		Position     string `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" || req.LastName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO members (first_name, last_name, date_of_birth, member_number, pass_number, position) VALUES (?,?,?,?,?,?)`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(req.MemberNumber),
		nullableString(req.PassNumber), nullableString(req.Position))
	if err != nil {
		http.Error(w, "duplicate pass number or internal error", http.StatusConflict)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// GET /api/members/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var m Member
	var jerseyNum sql.NullInt64
	var userID sql.NullInt64
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, first_name, last_name, COALESCE(date_of_birth,''), COALESCE(member_number,''), COALESCE(pass_number,''),
		        jersey_number, COALESCE(position,''), status, user_id
		 FROM members WHERE id=?`, id).
		Scan(&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth, &m.MemberNumber, &m.PassNumber,
			&jerseyNum, &m.Position, &m.Status, &userID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if jerseyNum.Valid {
		n := int(jerseyNum.Int64)
		m.JerseyNumber = &n
	}
	if userID.Valid {
		n := int(userID.Int64)
		m.UserID = &n
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

// PUT /api/members/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		DateOfBirth  string `json:"date_of_birth"`
		MemberNumber string `json:"member_number"`
		PassNumber   string `json:"pass_number"`
		JerseyNumber *int   `json:"jersey_number"`
		Position     string `json:"position"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	_, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET first_name=?, last_name=?, date_of_birth=?, member_number=?, pass_number=?, jersey_number=?, position=?, updated_at=? WHERE id=?`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(req.MemberNumber),
		nullableString(req.PassNumber), req.JerseyNumber, nullableString(req.Position), time.Now(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/members/:id/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(), `UPDATE members SET status=?, updated_at=? WHERE id=?`, req.Status, time.Now(), id)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/members/:id/team-assignment
func (h *Handler) AssignTeam(w http.ResponseWriter, r *http.Request) {
	memberID := r.PathValue("id")
	var req struct {
		TeamID    int  `json:"team_id"`
		SeasonID  int  `json:"season_id"`
		IsPrimary bool `json:"is_primary"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.IsPrimary {
		h.db.ExecContext(r.Context(),
			`UPDATE team_memberships SET is_primary=0 WHERE member_id=? AND season_id=?`, memberID, req.SeasonID)
	}
	h.db.ExecContext(r.Context(),
		`INSERT OR REPLACE INTO team_memberships (member_id, team_id, season_id, is_primary) VALUES (?,?,?,?)`,
		memberID, req.TeamID, req.SeasonID, boolToInt(req.IsPrimary))
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/members/:id/user
func (h *Handler) LinkUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		UserID *int `json:"user_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`UPDATE members SET user_id=?, updated_at=? WHERE id=?`,
		req.UserID, time.Now(), id)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/admin/family-links
func (h *Handler) CreateFamilyLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentUserID int `json:"parent_user_id"`
		MemberID     int `json:"member_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO family_links (parent_user_id, member_id) VALUES (?,?)`,
		req.ParentUserID, req.MemberID)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/members/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.pass_number,''), m.status,
		        COALESCE(t.name,'') as team
		 FROM members m
		 LEFT JOIN team_memberships tm ON tm.member_id = m.id AND tm.is_primary=1
		 LEFT JOIN teams t ON t.id = tm.team_id
		 WHERE m.status != 'ausgetreten'
		 ORDER BY m.last_name, m.first_name`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="mitglieder.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"Vorname", "Nachname", "Geburtsdatum", "Passnummer", "Status", "Mannschaft"})
	for rows.Next() {
		var firstName, lastName, dob, pass, status, team string
		rows.Scan(&firstName, &lastName, &dob, &pass, &status, &team)
		cw.Write([]string{firstName, lastName, dob, pass, status, team})
	}
	cw.Flush()
}

// GET /api/profile/me — returns the logged-in user's linked member profile(s)
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	result := []Member{}

	// Spieler / Trainer / Admin: own member profile via user_id
	var m Member
	var jerseyNum, userID sql.NullInt64
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, first_name, last_name, COALESCE(date_of_birth,''), COALESCE(member_number,''), COALESCE(pass_number,''),
		        jersey_number, COALESCE(position,''), status, user_id
		 FROM members WHERE user_id=?`, claims.UserID).
		Scan(&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth, &m.MemberNumber, &m.PassNumber,
			&jerseyNum, &m.Position, &m.Status, &userID)
	if err == nil {
		if jerseyNum.Valid {
			n := int(jerseyNum.Int64); m.JerseyNumber = &n
		}
		if userID.Valid {
			n := int(userID.Int64); m.UserID = &n
		}
		result = append(result, m)
	}

	// Elternteil: also include linked children via family_links
	if claims.Role == "elternteil" {
		rows, err := h.db.QueryContext(r.Context(),
			`SELECT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
			        m.jersey_number, COALESCE(m.position,''), m.status, m.user_id
			 FROM members m
			 JOIN family_links fl ON fl.member_id = m.id
			 WHERE fl.parent_user_id=?`, claims.UserID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var child Member
				var jn, uid sql.NullInt64
				rows.Scan(&child.ID, &child.FirstName, &child.LastName, &child.DateOfBirth, &child.MemberNumber, &child.PassNumber,
					&jn, &child.Position, &child.Status, &uid)
				if jn.Valid { n := int(jn.Int64); child.JerseyNumber = &n }
				if uid.Valid { n := int(uid.Int64); child.UserID = &n }
				result = append(result, child)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET/PUT /api/profile/vehicle
func (h *Handler) GetVehicle(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var seats int
	var notes string
	h.db.QueryRowContext(r.Context(),
		`SELECT seats, COALESCE(notes,'') FROM vehicle_info WHERE user_id=?`, claims.UserID).
		Scan(&seats, &notes)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"seats": seats, "notes": notes})
}

func (h *Handler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		Seats int    `json:"seats"`
		Notes string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`INSERT INTO vehicle_info (user_id, seats, notes, updated_at) VALUES (?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET seats=excluded.seats, notes=excluded.notes, updated_at=excluded.updated_at`,
		claims.UserID, req.Seats, req.Notes, time.Now())
	w.WriteHeader(http.StatusNoContent)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
