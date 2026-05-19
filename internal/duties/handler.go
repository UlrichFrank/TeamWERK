package duties

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

// GET /api/admin/duty-types
func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes FROM duty_types ORDER BY name`)
	defer rows.Close()
	type dt struct {
		ID                   int      `json:"id"`
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute,omitempty"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
	}
	result := []dt{}
	for rows.Next() {
		var d dt
		var cs sql.NullFloat64
		rows.Scan(&d.ID, &d.Name, &d.HoursValue, &cs, &d.DefaultAnchor, &d.DefaultOffsetMinutes)
		if cs.Valid {
			d.CashSubstitute = &cs.Float64
		}
		result = append(result, d)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/duty-types
func (h *Handler) CreateType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DefaultAnchor == "" {
		req.DefaultAnchor = "start"
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO duty_types (name, hours_value, cash_substitute, default_anchor, default_offset_minutes) VALUES (?,?,?,?,?)`,
		req.Name, req.HoursValue, req.CashSubstitute, req.DefaultAnchor, req.DefaultOffsetMinutes)
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/admin/duty-types/:id
func (h *Handler) UpdateType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name                 string   `json:"name"`
		HoursValue           float64  `json:"hours_value"`
		CashSubstitute       *float64 `json:"cash_substitute"`
		DefaultAnchor        string   `json:"default_anchor"`
		DefaultOffsetMinutes int      `json:"default_offset_minutes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DefaultAnchor == "" {
		req.DefaultAnchor = "start"
	}
	h.db.ExecContext(r.Context(),
		`UPDATE duty_types SET name=?, hours_value=?, cash_substitute=?, default_anchor=?, default_offset_minutes=? WHERE id=?`,
		req.Name, req.HoursValue, req.CashSubstitute, req.DefaultAnchor, req.DefaultOffsetMinutes, id)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/duty-types/:id
func (h *Handler) DeleteType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.ExecContext(r.Context(), `DELETE FROM duty_types WHERE id=?`, id)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-slots
func (h *Handler) ListSlots(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT ds.id, ds.event_name, ds.event_date, ds.slots_total, ds.slots_filled,
		        dt.name, COALESCE(ds.role_desc,'')
		 FROM duty_slots ds JOIN duty_types dt ON dt.id = ds.duty_type_id
		 ORDER BY ds.event_date DESC`)
	defer rows.Close()
	type slot struct {
		ID          int    `json:"id"`
		EventName   string `json:"event_name"`
		EventDate   string `json:"event_date"`
		SlotsTotal  int    `json:"slots_total"`
		SlotsFilled int    `json:"slots_filled"`
		DutyType    string `json:"duty_type"`
		RoleDesc    string `json:"role_desc,omitempty"`
	}
	result := []slot{}
	for rows.Next() {
		var s slot
		rows.Scan(&s.ID, &s.EventName, &s.EventDate, &s.SlotsTotal, &s.SlotsFilled, &s.DutyType, &s.RoleDesc)
		result = append(result, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/duty-slots
func (h *Handler) CreateSlot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventName  string `json:"event_name"`
		EventDate  string `json:"event_date"`
		EventTime  string `json:"event_time"`
		DutyTypeID int    `json:"duty_type_id"`
		RoleDesc   string `json:"role_desc"`
		SlotsTotal int    `json:"slots_total"`
		TeamID     *int   `json:"team_id"`
		SeasonID   int    `json:"season_id"`
		GameID     *int   `json:"game_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var eventTime any = nil
	if req.EventTime != "" {
		eventTime = req.EventTime
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		req.EventName, req.EventDate, eventTime, req.DutyTypeID, req.RoleDesc, req.SlotsTotal, req.TeamID, req.SeasonID, req.GameID)
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/duty-slots/:id
func (h *Handler) UpdateSlot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		EventName  string `json:"event_name"`
		EventDate  string `json:"event_date"`
		EventTime  string `json:"event_time"`
		RoleDesc   string `json:"role_desc"`
		SlotsTotal int    `json:"slots_total"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var eventTime any = nil
	if req.EventTime != "" {
		eventTime = req.EventTime
	}
	h.db.ExecContext(r.Context(),
		`UPDATE duty_slots SET event_name=?, event_date=?, event_time=?, role_desc=?, slots_total=? WHERE id=?`,
		req.EventName, req.EventDate, eventTime, req.RoleDesc, req.SlotsTotal, id)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/duty-slots/:id
func (h *Handler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM duty_slots WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-board  — open slots only
func (h *Handler) Board(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT ds.id, ds.event_name, ds.event_date, ds.slots_total - ds.slots_filled as vacancies,
		        dt.name, COALESCE(ds.role_desc,'')
		 FROM duty_slots ds JOIN duty_types dt ON dt.id = ds.duty_type_id
		 WHERE ds.slots_filled < ds.slots_total AND ds.event_date >= date('now')
		 ORDER BY ds.event_date`)
	defer rows.Close()
	type slot struct {
		ID        int    `json:"id"`
		EventName string `json:"event_name"`
		EventDate string `json:"event_date"`
		Vacancies int    `json:"vacancies"`
		DutyType  string `json:"duty_type"`
		RoleDesc  string `json:"role_desc,omitempty"`
	}
	result := []slot{}
	for rows.Next() {
		var s slot
		rows.Scan(&s.ID, &s.EventName, &s.EventDate, &s.Vacancies, &s.DutyType, &s.RoleDesc)
		result = append(result, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/duty-board/:slotId/claim
func (h *Handler) Claim(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")
	claims := auth.ClaimsFromCtx(r.Context())
	var total, filled int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT slots_total, slots_filled FROM duty_slots WHERE id=?`, slotID).
		Scan(&total, &filled)
	if err != nil || filled >= total {
		http.Error(w, "slot full or not found", http.StatusConflict)
		return
	}
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?,?)`, slotID, claims.UserID)
	if err != nil {
		http.Error(w, "already claimed", http.StatusConflict)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE duty_slots SET slots_filled = slots_filled + 1 WHERE id=?`, slotID)
	h.updateAccount(r, claims.UserID, slotID, true)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-slots/:id/assignments
func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("id")
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT da.id, u.name, da.status, COALESCE(da.cash_amount,0)
		 FROM duty_assignments da JOIN users u ON u.id = da.user_id
		 WHERE da.duty_slot_id=? ORDER BY da.created_at`, slotID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type assignment struct {
		ID         int     `json:"id"`
		UserName   string  `json:"user_name"`
		Status     string  `json:"status"`
		CashAmount float64 `json:"cash_amount,omitempty"`
	}
	result := []assignment{}
	for rows.Next() {
		var a assignment
		rows.Scan(&a.ID, &a.UserName, &a.Status, &a.CashAmount)
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/duty-assignments/:id/fulfill
func (h *Handler) Fulfill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.ExecContext(r.Context(),
		`UPDATE duty_assignments SET status='fulfilled', fulfilled_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/duty-assignments/:id/cash-substitute
func (h *Handler) CashSubstitute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Amount float64 `json:"amount"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`UPDATE duty_assignments SET status='cash_substitute', cash_amount=?, fulfilled_at=CURRENT_TIMESTAMP WHERE id=?`,
		req.Amount, id)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/duty-accounts
func (h *Handler) Accounts(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var rows *sql.Rows
	if claims.Role == "admin" {
		rows, _ = h.db.QueryContext(r.Context(),
			`SELECT da.user_id, u.name, da.season_id, da.soll, da.ist
			 FROM duty_accounts da JOIN users u ON u.id = da.user_id
			 ORDER BY u.name`)
	} else {
		rows, _ = h.db.QueryContext(r.Context(),
			`SELECT da.user_id, u.name, da.season_id, da.soll, da.ist
			 FROM duty_accounts da JOIN users u ON u.id = da.user_id
			 WHERE da.user_id=?`, claims.UserID)
	}
	defer rows.Close()
	type account struct {
		UserID   int     `json:"user_id"`
		Name     string  `json:"name"`
		SeasonID int     `json:"season_id"`
		Soll     float64 `json:"soll"`
		Ist      float64 `json:"ist"`
		Balance  float64 `json:"balance"`
	}
	result := []account{}
	for rows.Next() {
		var a account
		rows.Scan(&a.UserID, &a.Name, &a.SeasonID, &a.Soll, &a.Ist)
		a.Balance = a.Soll - a.Ist
		result = append(result, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/admin/duty-accounts/export
func (h *Handler) ExportAccounts(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT u.name, da.soll, da.ist, da.soll - da.ist as balance,
		        COALESCE(SUM(CASE WHEN dassign.status='cash_substitute' THEN dassign.cash_amount ELSE 0 END), 0)
		 FROM duty_accounts da
		 JOIN users u ON u.id = da.user_id
		 LEFT JOIN duty_assignments dassign ON dassign.user_id = da.user_id
		 GROUP BY da.user_id, da.season_id
		 ORDER BY u.name`)
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="dienstkonten.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"Name", "Soll", "Ist", "Saldo", "Geldersatz"})
	for rows.Next() {
		var name string
		var soll, ist, balance, cash float64
		rows.Scan(&name, &soll, &ist, &balance, &cash)
		cw.Write([]string{name,
			fmtFloat(soll), fmtFloat(ist), fmtFloat(balance), fmtFloat(cash)})
	}
	cw.Flush()
}

// PUT /api/admin/seasons/:id/duty-targets
func (h *Handler) SetSeasonTargets(w http.ResponseWriter, r *http.Request) {
	seasonID := r.PathValue("id")
	var req []struct {
		DutyTypeID  int     `json:"duty_type_id"`
		TargetHours float64 `json:"target_hours"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	for _, t := range req {
		h.db.ExecContext(r.Context(),
			`INSERT INTO duty_season_targets (season_id, duty_type_id, target_hours) VALUES (?,?,?)
			 ON CONFLICT(season_id, duty_type_id) DO UPDATE SET target_hours=excluded.target_hours`,
			seasonID, t.DutyTypeID, t.TargetHours)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateAccount(r *http.Request, userID int, slotID string, add bool) {
	var hours float64
	var seasonID int
	h.db.QueryRowContext(r.Context(),
		`SELECT dt.hours_value, ds.season_id FROM duty_slots ds
		 JOIN duty_types dt ON dt.id = ds.duty_type_id WHERE ds.id=?`, slotID).
		Scan(&hours, &seasonID)
	delta := hours
	if !add {
		delta = -hours
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO duty_accounts (user_id, season_id, soll, ist) VALUES (?,?,0,?)
		 ON CONFLICT(user_id, season_id) DO UPDATE SET ist = ist + excluded.ist`,
		userID, seasonID, delta)
}

func fmtFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
