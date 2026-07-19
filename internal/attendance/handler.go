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
	TeamName         string         `json:"team_name"`
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
	memberWhere := ""
	if extended {
		memberJoin = `
		JOIN kader_extended_members kem ON kem.member_id = m.id
		JOIN kader k ON k.id = kem.kader_id AND k.team_id = ? AND k.season_id = ?`
		memberWhere = `
		WHERE NOT EXISTS (
			SELECT 1 FROM kader_members km2
			JOIN kader k2 ON k2.id = km2.kader_id
			WHERE km2.member_id = m.id AND k2.team_id = ? AND k2.season_id = ?
		)`
	}

	// Trainings-Zähler
	trainingSQL := `
		SELECT m.id, m.first_name || ' ' || m.last_name,
		       COALESCE(SUM(CASE WHEN ta.present = 1 AND msu.id IS NULL THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ta.present = 0 AND msu.id IS NULL THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN ta.present IS NULL
		                          AND tr.status = 'declined'
		                          AND tr.absence_id IS NOT NULL
		                          AND msu.id IS NULL THEN 1 ELSE 0 END), 0)
		FROM members m` + memberJoin + `
		LEFT JOIN training_sessions ts ON ts.team_id = k.team_id
		                              AND ts.season_id = k.season_id
		                              AND ts.status != 'cancelled'
		                              AND date(ts.date) BETWEEN date(?) AND date(?)
		LEFT JOIN training_attendances ta ON ta.training_id = ts.id AND ta.member_id = m.id
		LEFT JOIN training_responses tr ON tr.training_id = ts.id AND tr.member_id = m.id
		LEFT JOIN member_series_unavailabilities msu
		       ON msu.member_id = m.id
		      AND msu.training_series_id = ts.series_id
		      AND (msu.start_date IS NULL OR msu.start_date <= date(ts.date))
		      AND (msu.end_date   IS NULL OR msu.end_date   >= date(ts.date))` +
		memberWhere + `
		GROUP BY m.id, m.first_name, m.last_name
		ORDER BY m.first_name, m.last_name`

	var trainingArgs []any
	trainingArgs = append(trainingArgs, teamID, seasonID)
	trainingArgs = append(trainingArgs, startDate, endDate)
	if extended {
		trainingArgs = append(trainingArgs, teamID, seasonID)
	}

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
		                  AND g.event_type IN ('heim','auswärts')
		                  AND date(g.date) BETWEEN date(?) AND date(?)
		LEFT JOIN game_attendances ga ON ga.game_id = g.id AND ga.member_id = m.id
		LEFT JOIN game_responses gr ON gr.game_id = g.id AND gr.member_id = m.id` +
		memberWhere + `
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

	var teamName string
	if err := h.db.QueryRowContext(r.Context(), `SELECT name FROM teams WHERE id = ?`, teamID).Scan(&teamName); err == sql.ErrNoRows {
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
		TeamName:         teamName,
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

// memberStatsResponse ist die Antwortstruktur von
// GET /api/members/{id}/attendance-stats.
type memberStatsResponse struct {
	MemberID  int           `json:"member_id"`
	SeasonID  int           `json:"season_id"`
	StartDate string        `json:"start_date"`
	EndDate   string        `json:"end_date"`
	Counts    memberCounts  `json:"counts"`
	Events    []eventDetail `json:"events"`
}

type eventDetail struct {
	EventType string   `json:"event_type"` // "training" oder "game"
	EventID   int      `json:"event_id"`
	Date      string   `json:"date"`
	Title     string   `json:"title"`
	Category  Category `json:"category"`
	Reason    *string  `json:"reason"`
}

// canSeeMemberStats prüft die Authz für Mitglieds-Statistik:
// admin, sportliche_leitung, der Member selbst, Eltern via family_links,
// oder Trainer eines Teams, in dessen Kader das Mitglied steht.
func (h *Handler) canSeeMemberStats(ctx context.Context, claims *auth.Claims, memberID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		return true, nil
	}
	var ownUserID sql.NullInt64
	if err := h.db.QueryRowContext(ctx,
		`SELECT user_id FROM members WHERE id = ?`, memberID).Scan(&ownUserID); err != nil {
		return false, err
	}
	if ownUserID.Valid && int(ownUserID.Int64) == claims.UserID {
		return true, nil
	}
	var familyN int
	if err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
		claims.UserID, memberID).Scan(&familyN); err != nil {
		return false, err
	}
	if familyN > 0 {
		return true, nil
	}
	if !claims.HasFunction("trainer") {
		return false, nil
	}
	var trainerN int
	if err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members tm ON tm.id = trm.member_id AND tm.user_id = ?
		WHERE trm.team_id IN (
			SELECT k.team_id FROM kader k
			WHERE k.season_id = trm.season_id AND (
				EXISTS (SELECT 1 FROM kader_members km          WHERE km.kader_id = k.id  AND km.member_id  = ?)
				OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = ?)
			)
		)`,
		claims.UserID, memberID, memberID).Scan(&trainerN); err != nil {
		return false, err
	}
	return trainerN > 0, nil
}

// GetMemberStats — GET /api/members/{id}/attendance-stats?season=<id>
func (h *Handler) GetMemberStats(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var memberName string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name || ' ' || last_name FROM members WHERE id = ?`,
		memberID).Scan(&memberName); err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ok, err := h.canSeeMemberStats(r.Context(), claims, memberID)
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

	events, counts, err := h.loadMemberEvents(r.Context(), memberID, seasonID, startDate, endDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMemberStats: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	counts.MemberID = memberID
	counts.MemberName = memberName

	resp := memberStatsResponse{
		MemberID:  memberID,
		SeasonID:  seasonID,
		StartDate: startDate,
		EndDate:   endDate,
		Counts:    counts,
		Events:    events,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// loadMemberEvents liefert alle relevanten Trainings + Spiele eines Mitglieds
// in einer Saison, klassifiziert sie und aggregiert die Zähler.
func (h *Handler) loadMemberEvents(ctx context.Context, memberID, seasonID int, startDate, endDate string) ([]eventDetail, memberCounts, error) {
	events := []eventDetail{}
	counts := memberCounts{}

	// Trainings: alle Sessions der Teams, in deren Kader oder erweitertem
	// Kader das Mitglied in dieser Saison steht.
	trainingRows, err := h.db.QueryContext(ctx, `
		SELECT ts.id, ts.date, ts.title, ts.status,
		       ta.present, tr.status, tr.absence_id IS NOT NULL, tr.reason,
		       msu.id IS NOT NULL, msu.reason
		FROM training_sessions ts
		LEFT JOIN training_attendances ta ON ta.training_id = ts.id AND ta.member_id = ?
		LEFT JOIN training_responses  tr ON tr.training_id = ts.id AND tr.member_id = ?
		LEFT JOIN member_series_unavailabilities msu
		       ON msu.member_id = ?
		      AND msu.training_series_id = ts.series_id
		      AND (msu.start_date IS NULL OR msu.start_date <= date(ts.date))
		      AND (msu.end_date   IS NULL OR msu.end_date   >= date(ts.date))
		WHERE ts.season_id = ?
		  AND date(ts.date) BETWEEN date(?) AND date(?)
		  AND ts.team_id IN (
		    SELECT k.team_id FROM kader k
		    WHERE k.season_id = ? AND (
		      EXISTS (SELECT 1 FROM kader_members km          WHERE km.kader_id = k.id  AND km.member_id  = ?)
		      OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = ?)
		    )
		  )
		ORDER BY ts.date, ts.id`,
		memberID, memberID, memberID, seasonID, startDate, endDate, seasonID, memberID, memberID)
	if err != nil {
		return nil, counts, fmt.Errorf("training events: %w", err)
	}
	for trainingRows.Next() {
		var ev eventDetail
		var status string
		var present sql.NullInt64
		var respStatus, reason, unavailReason sql.NullString
		var hasAbsence, unavailable bool
		if err := trainingRows.Scan(&ev.EventID, &ev.Date, &ev.Title, &status,
			&present, &respStatus, &hasAbsence, &reason, &unavailable, &unavailReason); err != nil {
			trainingRows.Close()
			return nil, counts, err
		}
		ev.EventType = "training"
		if ev.Title == "" {
			ev.Title = "Training"
		}
		switch {
		case status == "cancelled":
			ev.Category = CategoryCanceled
		case unavailable:
			// Serien-Abmeldung dominiert (auch eine bereits erfasste Anwesenheit
			// oder entschuldigte Absage): zählt in keiner Säule.
			ev.Category = CategoryUnavailable
			if unavailReason.Valid && unavailReason.String != "" {
				s := unavailReason.String
				ev.Reason = &s
			}
		default:
			ev.Category = classifyRow(present, respStatus, hasAbsence)
			tallyCount(&counts, ev.EventType, ev.Category)
		}
		if ev.Reason == nil && reason.Valid && reason.String != "" {
			s := reason.String
			ev.Reason = &s
		}
		events = append(events, ev)
	}
	trainingRows.Close()

	// Spiele: nur Heim-/Auswärtsspiele (keine generischen Termine wie
	// Sommerfest), deren team_ids im selben Sinn das Mitglied enthalten.
	gameRows, err := h.db.QueryContext(ctx, `
		SELECT g.id, g.date,
		       COALESCE(NULLIF(g.opponent, ''), 'Spiel'),
		       ga.present, gr.status, gr.absence_id IS NOT NULL, gr.reason
		FROM games g
		LEFT JOIN game_attendances ga ON ga.game_id = g.id AND ga.member_id = ?
		LEFT JOIN game_responses   gr ON gr.game_id = g.id AND gr.member_id = ?
		WHERE g.season_id = ?
		  AND g.event_type IN ('heim','auswärts')
		  AND date(g.date) BETWEEN date(?) AND date(?)
		  AND EXISTS (
		    SELECT 1 FROM game_teams gt
		    JOIN kader k ON k.team_id = gt.team_id AND k.season_id = g.season_id
		    WHERE gt.game_id = g.id AND (
		      EXISTS (SELECT 1 FROM kader_members km          WHERE km.kader_id = k.id  AND km.member_id  = ?)
		      OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = ?)
		    )
		  )
		ORDER BY g.date, g.id`,
		memberID, memberID, seasonID, startDate, endDate, memberID, memberID)
	if err != nil {
		return nil, counts, fmt.Errorf("game events: %w", err)
	}
	defer gameRows.Close()
	for gameRows.Next() {
		var ev eventDetail
		var present sql.NullInt64
		var respStatus, reason sql.NullString
		var hasAbsence bool
		if err := gameRows.Scan(&ev.EventID, &ev.Date, &ev.Title,
			&present, &respStatus, &hasAbsence, &reason); err != nil {
			return nil, counts, err
		}
		ev.EventType = "game"
		ev.Category = classifyRow(present, respStatus, hasAbsence)
		tallyCount(&counts, ev.EventType, ev.Category)
		if reason.Valid && reason.String != "" {
			s := reason.String
			ev.Reason = &s
		}
		events = append(events, ev)
	}
	return events, counts, nil
}

// classifyRow ist die SQL-Adapter-Variante von Classify: nimmt die rohen
// NullInt64/NullString-Werte und ruft die reine Funktion auf.
func classifyRow(present sql.NullInt64, respStatus sql.NullString, hasAbsence bool) Category {
	var p *bool
	if present.Valid {
		b := present.Int64 == 1
		p = &b
	}
	declined := respStatus.Valid && respStatus.String == "declined"
	return Classify(p, declined, hasAbsence)
}

func tallyCount(c *memberCounts, evType string, cat Category) {
	switch evType {
	case "training":
		switch cat {
		case CategoryPresent:
			c.TrainingPresent++
		case CategoryMissed:
			c.TrainingMissed++
		case CategoryExcused:
			c.TrainingExcused++
		}
	case "game":
		switch cat {
		case CategoryPresent:
			c.GamePresent++
		case CategoryMissed:
			c.GameMissed++
		case CategoryExcused:
			c.GameExcused++
		}
	}
}

// openItem ist ein vergangener Termin ohne Anwesenheits-Erfassung.
type openItem struct {
	EventType string `json:"event_type"` // "training" oder "game"
	EventID   int    `json:"event_id"`
	Date      string `json:"date"`
	Title     string `json:"title"`
}

// GetTeamOpen — GET /api/teams/{id}/attendance-open
// Liefert vergangene, nicht cancelled Termine des Teams in der aktiven
// Saison, für die noch keine attendance-Zeile existiert.
func (h *Handler) GetTeamOpen(w http.ResponseWriter, r *http.Request) {
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

	seasonID, startDate, _, err := h.resolveSeason(r.Context(), "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if seasonID == 0 {
		// Keine aktive Saison: leere Liste statt Fehler — UI rendert dann
		// keinen Banner.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]openItem{})
		return
	}

	items := []openItem{}
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT ts.id, ts.date,
		       COALESCE(NULLIF(ts.title, ''), 'Training')
		FROM training_sessions ts
		WHERE ts.team_id = ?
		  AND ts.season_id = ?
		  AND ts.status != 'cancelled'
		  AND date(ts.date) >= date(?)
		  AND date(ts.date) < date('now')
		  AND NOT EXISTS (
		    SELECT 1 FROM training_attendances ta WHERE ta.training_id = ts.id
		  )
		ORDER BY ts.date, ts.id`,
		teamID, seasonID, startDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetTeamOpen trainings: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var it openItem
		it.EventType = "training"
		if err := rows.Scan(&it.EventID, &it.Date, &it.Title); err != nil {
			rows.Close()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}
	rows.Close()

	rows, err = h.db.QueryContext(r.Context(), `
		SELECT g.id, g.date,
		       COALESCE(NULLIF(g.opponent, ''), 'Spiel')
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id AND gt.team_id = ?
		WHERE g.season_id = ?
		  AND date(g.date) >= date(?)
		  AND date(g.date) < date('now')
		  AND NOT EXISTS (
		    SELECT 1 FROM game_attendances ga WHERE ga.game_id = g.id
		  )
		ORDER BY g.date, g.id`,
		teamID, seasonID, startDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetTeamOpen games: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var it openItem
		it.EventType = "game"
		if err := rows.Scan(&it.EventID, &it.Date, &it.Title); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
