package attendance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// Handler bündelt die HTTP-Endpoints für die Anwesenheits-Statistik.
type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

// NewHandler bindet DB und Event-Hub. Der Hub wird hier vorgehalten, damit
// spätere Mutationen Broadcasts auslösen können.
func NewHandler(db *sql.DB, h *hub.EventHub) *Handler {
	return &Handler{db: db, hub: h}
}

// memberCounts ist die Aggregat-Zeile eines Mitglieds (Drei-Säulen je
// Termin-Typ).
type memberCounts struct {
	MemberID        int    `json:"member_id"`
	MemberName      string `json:"member_name"`
	TrainingPresent int    `json:"training_present"`
	TrainingMissed  int    `json:"training_missed"`
	TrainingExcused int    `json:"training_excused"`
	GamePresent     int    `json:"game_present"`
	GameMissed      int    `json:"game_missed"`
	GameExcused     int    `json:"game_excused"`
}

// teamStatsResponse ist die Antwortstruktur von
// GET /api/teams/{id}/attendance-stats.
type teamStatsResponse struct {
	TeamID           int            `json:"team_id"`
	SeasonID         int            `json:"season_id"`
	StartDate        string         `json:"start_date"`
	EndDate          string         `json:"end_date"`
	RegularMembers   []memberCounts `json:"regular_members"`
	ExtendedMembers  []memberCounts `json:"extended_members"`
	RegularAverages  averages       `json:"regular_averages"`
	ExtendedAverages averages       `json:"extended_averages"`
}

type averages struct {
	TrainingPresent float64 `json:"training_present"`
	TrainingMissed  float64 `json:"training_missed"`
	TrainingExcused float64 `json:"training_excused"`
	GamePresent     float64 `json:"game_present"`
	GameMissed      float64 `json:"game_missed"`
	GameExcused     float64 `json:"game_excused"`
}

// canSeeTeamStats prüft die Authz für team-bezogene Endpoints: admin,
// sportliche_leitung (alle Teams) oder Trainer des Teams (kader_trainers in
// der aktiven Saison).
func (h *Handler) canSeeTeamStats(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		return true, nil
	}
	if !claims.HasFunction("trainer") {
		return false, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		WHERE trm.team_id = ?`,
		claims.UserID, teamID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// resolveSeason ermittelt die anzuwendende Saison + ihren Zeitraum (start,
// end-clamped-to-today). Liefert (0, "", "", nil) wenn keine aktive Saison
// existiert.
func (h *Handler) resolveSeason(ctx context.Context, seasonParam string) (id int, startDate, endDate string, err error) {
	row := h.db.QueryRowContext(ctx, `
		SELECT id, start_date,
		       CASE WHEN date(end_date) < date('now') THEN end_date ELSE date('now') END
		FROM seasons
		WHERE id = COALESCE(?, (SELECT id FROM seasons WHERE is_active = 1))`,
		nullableStr(seasonParam))
	if scanErr := row.Scan(&id, &startDate, &endDate); scanErr == sql.ErrNoRows {
		return 0, "", "", nil
	} else if scanErr != nil {
		return 0, "", "", scanErr
	}
	return id, startDate, endDate, nil
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// loadCounts berechnet die sechs Zähler je Mitglied. extended=true zählt
// die erweiterten Kader-Mitglieder, die NICHT im Stammkader sind.
func (h *Handler) loadCounts(ctx context.Context, teamID, seasonID int, startDate, endDate string, extended bool) ([]memberCounts, error) {
	memberJoin := `
		JOIN kader_members km ON km.member_id = m.id
		JOIN kader k ON k.id = km.kader_id AND k.team_id = ? AND k.season_id = ?`
	if extended {
		memberJoin = `
		JOIN kader_extended_members kem ON kem.member_id = m.id
		JOIN kader k ON k.id = kem.kader_id AND k.team_id = ? AND k.season_id = ?
		WHERE NOT EXISTS (
			SELECT 1 FROM kader_members km2
			JOIN kader k2 ON k2.id = km2.kader_id
			WHERE km2.member_id = m.id AND k2.team_id = ? AND k2.season_id = ?
		)`
	}

	// Trainings-Zähler
	trainingSQL := `
		SELECT m.id, m.first_name || ' ' || m.last_name,
		       COALESCE(SUM(CASE WHEN ta.present = 1 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ta.present = 0 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ta.present IS NULL
		                          AND tr.status = 'declined'
		                          AND tr.absence_id IS NOT NULL THEN 1 ELSE 0 END), 0)
		FROM members m` + memberJoin + `
		LEFT JOIN training_sessions ts ON ts.team_id = k.team_id
		                              AND ts.season_id = k.season_id
		                              AND ts.status != 'cancelled'
		                              AND date(ts.date) BETWEEN date(?) AND date(?)
		LEFT JOIN training_attendances ta ON ta.training_id = ts.id AND ta.member_id = m.id
		LEFT JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = m.id
		GROUP BY m.id, m.first_name, m.last_name
		ORDER BY m.first_name, m.last_name`

	var trainingArgs []any
	trainingArgs = append(trainingArgs, teamID, seasonID)
	if extended {
		trainingArgs = append(trainingArgs, teamID, seasonID)
	}
	trainingArgs = append(trainingArgs, startDate, endDate)

	rows, err := h.db.QueryContext(ctx, trainingSQL, trainingArgs...)
	if err != nil {
		return nil, fmt.Errorf("training counts: %w", err)
	}
	defer rows.Close()

	byID := map[int]*memberCounts{}
	order := []int{}
	for rows.Next() {
		var c memberCounts
		if err := rows.Scan(&c.MemberID, &c.MemberName,
			&c.TrainingPresent, &c.TrainingMissed, &c.TrainingExcused); err != nil {
			return nil, err
		}
		byID[c.MemberID] = &c
		order = append(order, c.MemberID)
	}
	rows.Close()

	// Spiele-Zähler (über game_teams gejoint)
	gameSQL := `
		SELECT m.id,
		       COALESCE(SUM(CASE WHEN ga.present = 1 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ga.present = 0 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ga.present IS NULL
		                          AND gr.status = 'declined'
		                          AND gr.absence_id IS NOT NULL THEN 1 ELSE 0 END), 0)
		FROM members m` + memberJoin + `
		LEFT JOIN game_teams gt ON gt.team_id = k.team_id
		LEFT JOIN games g ON g.id = gt.game_id
		                  AND g.season_id = k.season_id
		                  AND g.status != 'cancelled'
		                  AND date(g.date) BETWEEN date(?) AND date(?)
		LEFT JOIN game_attendances ga ON ga.game_id = g.id AND ga.member_id = m.id
		LEFT JOIN game_responses gr ON gr.game_id = g.id AND gr.member_id = m.id
		GROUP BY m.id`

	rows, err = h.db.QueryContext(ctx, gameSQL, trainingArgs...)
	if err != nil {
		return nil, fmt.Errorf("game counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var memberID, present, missed, excused int
		if err := rows.Scan(&memberID, &present, &missed, &excused); err != nil {
			return nil, err
		}
		if c, ok := byID[memberID]; ok {
			c.GamePresent = present
			c.GameMissed = missed
			c.GameExcused = excused
		}
	}

	result := make([]memberCounts, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, nil
}

func computeAverages(rows []memberCounts) averages {
	if len(rows) == 0 {
		return averages{}
	}
	var a averages
	for _, r := range rows {
		a.TrainingPresent += float64(r.TrainingPresent)
		a.TrainingMissed += float64(r.TrainingMissed)
		a.TrainingExcused += float64(r.TrainingExcused)
		a.GamePresent += float64(r.GamePresent)
		a.GameMissed += float64(r.GameMissed)
		a.GameExcused += float64(r.GameExcused)
	}
	n := float64(len(rows))
	a.TrainingPresent /= n
	a.TrainingMissed /= n
	a.TrainingExcused /= n
	a.GamePresent /= n
	a.GameMissed /= n
	a.GameExcused /= n
	return a
}

// GetTeamStats — GET /api/teams/{id}/attendance-stats?season=<id>
func (h *Handler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	teamID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var exists int
	if err := h.db.QueryRowContext(r.Context(), `SELECT 1 FROM teams WHERE id = ?`, teamID).Scan(&exists); err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ok, err := h.canSeeTeamStats(r.Context(), claims, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	seasonID, startDate, endDate, err := h.resolveSeason(r.Context(), r.URL.Query().Get("season"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if seasonID == 0 {
		http.Error(w, "no active season", http.StatusNotFound)
		return
	}

	regular, err := h.loadCounts(r.Context(), teamID, seasonID, startDate, endDate, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetTeamStats regular: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	extended, err := h.loadCounts(r.Context(), teamID, seasonID, startDate, endDate, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetTeamStats extended: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := teamStatsResponse{
		TeamID:           teamID,
		SeasonID:         seasonID,
		StartDate:        startDate,
		EndDate:          endDate,
		RegularMembers:   regular,
		ExtendedMembers:  extended,
		RegularAverages:  computeAverages(regular),
		ExtendedAverages: computeAverages(extended),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
