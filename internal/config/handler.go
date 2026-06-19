package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/sepa"
)

var (
	glaeubigerIDRegex = regexp.MustCompile(`^DE\d{2}[A-Z0-9]{3}\d{11}$`)
	bicRegex          = regexp.MustCompile(`^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
)

type AgeClassRule struct {
	AgeClass            string `json:"age_class"`
	HalfDurationMinutes int    `json:"half_duration_minutes"`
	BreakMinutes        int    `json:"break_minutes"`
}

func GetAgeClassRules(db *sql.DB) ([]AgeClassRule, error) {
	rows, err := db.Query(
		`SELECT age_class, half_duration_minutes, break_minutes FROM age_class_game_rules ORDER BY age_class`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AgeClassRule{}
	for rows.Next() {
		var r AgeClassRule
		rows.Scan(&r.AgeClass, &r.HalfDurationMinutes, &r.BreakMinutes)
		result = append(result, r)
	}
	return result, nil
}

var validAgeClasses = map[string]bool{"A-Jugend": true, "B-Jugend": true, "C-Jugend": true, "D-Jugend": true}

func UpdateAgeClassRule(db *sql.DB, ageClass string, half, brk int) error {
	if !validAgeClasses[ageClass] {
		return errors.New("ungültige Altersklasse")
	}
	if half <= 0 || brk <= 0 {
		return errors.New("Werte müssen größer als 0 sein")
	}
	res, err := db.Exec(
		`UPDATE age_class_game_rules SET half_duration_minutes=?, break_minutes=? WHERE age_class=?`,
		half, brk, ageClass)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("Altersklasse nicht gefunden")
	}
	return nil
}

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

// GET /api/admin/club
func (h *Handler) GetClub(w http.ResponseWriter, r *http.Request) {
	var id int
	var name string
	var logoURL, address, glaeubigerID, iban, bic, kontoinhaber sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT id, name, logo_url, address, glaeubiger_id, iban, bic, kontoinhaber FROM clubs LIMIT 1`).
		Scan(&id, &name, &logoURL, &address, &glaeubigerID, &iban, &bic, &kontoinhaber)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id": id, "name": name, "logo_url": logoURL.String, "address": address.String,
		"glaeubiger_id": glaeubigerID.String, "iban": iban.String,
		"bic": bic.String, "kontoinhaber": kontoinhaber.String,
	})
}

// PUT /api/admin/club
func (h *Handler) UpdateClub(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		LogoURL      string `json:"logo_url"`
		Address      string `json:"address"`
		GlaeubigerID string `json:"glaeubiger_id"`
		IBAN         string `json:"iban"`
		BIC          string `json:"bic"`
		Kontoinhaber string `json:"kontoinhaber"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Normalisieren + validieren (leere Werte sind erlaubt = noch nicht gepflegt).
	req.IBAN = sepa.NormalizeIBAN(req.IBAN)
	req.BIC = normalizeUpper(req.BIC)
	req.GlaeubigerID = normalizeUpper(req.GlaeubigerID)
	if req.GlaeubigerID != "" && !glaeubigerIDRegex.MatchString(req.GlaeubigerID) {
		http.Error(w, "ungültige Gläubiger-ID", http.StatusBadRequest)
		return
	}
	if req.IBAN != "" && !sepa.IsValidIBAN(req.IBAN) {
		http.Error(w, "ungültige IBAN", http.StatusBadRequest)
		return
	}
	if req.BIC != "" && !bicRegex.MatchString(req.BIC) {
		http.Error(w, "ungültige BIC", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(),
		`UPDATE clubs SET name=?, logo_url=?, address=?, glaeubiger_id=?, iban=?, bic=?, kontoinhaber=?, updated_at=?
		 WHERE id=(SELECT id FROM clubs LIMIT 1)`,
		req.Name, req.LogoURL, req.Address,
		nullIfEmpty(req.GlaeubigerID), nullIfEmpty(req.IBAN), nullIfEmpty(req.BIC), nullIfEmpty(req.Kontoinhaber),
		time.Now())
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}

func normalizeUpper(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GET /api/admin/seasons
func (h *Handler) ListSeasons(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(), `SELECT id, name, start_date, end_date, is_active FROM seasons ORDER BY start_date DESC`)
	defer rows.Close()
	type season struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		IsActive  bool   `json:"is_active"`
	}
	result := []season{}
	for rows.Next() {
		var s season
		var active int
		rows.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &active)
		s.IsActive = active == 1
		result = append(result, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/seasons
func (h *Handler) CreateSeason(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(), `INSERT INTO seasons (name, start_date, end_date) VALUES (?,?,?)`,
		req.Name, req.StartDate, req.EndDate)
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/admin/seasons/:id/activate
func (h *Handler) ActivateSeason(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.ExecContext(r.Context(), `UPDATE seasons SET is_active=0`)
	h.db.ExecContext(r.Context(), `UPDATE seasons SET is_active=1 WHERE id=?`, id)
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/seasons/:id
func (h *Handler) UpdateSeason(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.StartDate == "" || req.EndDate == "" {
		http.Error(w, "name, start_date und end_date sind erforderlich", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE seasons SET name=?, start_date=?, end_date=? WHERE id=?`,
		req.Name, req.StartDate, req.EndDate, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	type season struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(season{ID: id, Name: req.Name, StartDate: req.StartDate, EndDate: req.EndDate})
}

// DELETE /api/admin/seasons/:id
func (h *Handler) DeleteSeason(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Prevent deleting the active season
	var isActive int
	if err := h.db.QueryRowContext(r.Context(), `SELECT is_active FROM seasons WHERE id=?`, id).Scan(&isActive); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if isActive == 1 {
		http.Error(w, "aktive Saison kann nicht gelöscht werden", http.StatusConflict)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM seasons WHERE id=?`, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/teams
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.QueryContext(r.Context(), `SELECT id, name, age_class, gender, is_active FROM teams ORDER BY name`)
	defer rows.Close()
	type team struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		AgeClass string `json:"age_class"`
		Gender   string `json:"gender"`
		IsActive bool   `json:"is_active"`
	}
	result := []team{}
	for rows.Next() {
		var t team
		var active int
		rows.Scan(&t.ID, &t.Name, &t.AgeClass, &t.Gender, &active)
		t.IsActive = active == 1
		result = append(result, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/admin/teams
func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string  `json:"name"`
		AgeClass *string `json:"age_class"`
		Gender   string  `json:"gender"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.AgeClass != nil && *req.AgeClass != "" {
		if !validAgeClasses[*req.AgeClass] {
			http.Error(w, "ungültige Altersklasse", http.StatusUnprocessableEntity)
			return
		}
	}
	var ageClassVal interface{}
	if req.AgeClass != nil && *req.AgeClass != "" {
		ageClassVal = *req.AgeClass
	}
	h.db.ExecContext(r.Context(), `INSERT INTO teams (name, age_class, gender) VALUES (?,?,?)`,
		req.Name, ageClassVal, req.Gender)
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/admin/teams/:id
func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name     string  `json:"name"`
		AgeClass *string `json:"age_class"`
		Gender   string  `json:"gender"`
		IsActive *bool   `json:"is_active"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.AgeClass != nil && *req.AgeClass != "" {
		if !validAgeClasses[*req.AgeClass] {
			http.Error(w, "ungültige Altersklasse", http.StatusUnprocessableEntity)
			return
		}
	}
	active := 1
	if req.IsActive != nil && !*req.IsActive {
		active = 0
	}
	var ageClassVal interface{}
	if req.AgeClass != nil && *req.AgeClass != "" {
		ageClassVal = *req.AgeClass
	}
	h.db.ExecContext(r.Context(), `UPDATE teams SET name=?, age_class=?, gender=?, is_active=? WHERE id=?`,
		req.Name, ageClassVal, req.Gender, active, id)
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/age-class-rules
func (h *Handler) GetAgeClassRulesHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := GetAgeClassRules(h.db)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// PUT /api/admin/age-class-rules/{ageClass}
func (h *Handler) UpdateAgeClassRuleHandler(w http.ResponseWriter, r *http.Request) {
	ageClass := r.PathValue("ageClass")
	var req struct {
		HalfDurationMinutes int `json:"half_duration_minutes"`
		BreakMinutes        int `json:"break_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := UpdateAgeClassRule(h.db, ageClass, req.HalfDurationMinutes, req.BreakMinutes); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	h.hub.Broadcast("settings")
	w.WriteHeader(http.StatusNoContent)
}
