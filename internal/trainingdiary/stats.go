package trainingdiary

import (
	"math"
	"net/http"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// memberSummary ist die Aggregat-Zeile eines Mitglieds in der
// Mannschaftsübersicht.
type memberSummary struct {
	MemberID   int     `json:"member_id"`
	MemberName string  `json:"member_name"`
	Entries    int     `json:"entries"`
	Minutes    int     `json:"minutes"`
	AvgRPE     float64 `json:"avg_rpe"`
}

// teamStatsResponse ist die Antwort von
// GET /api/teams/{id}/training-diary-stats.
type teamStatsResponse struct {
	SeasonID  int             `json:"season_id"`
	StartDate string          `json:"start_date"`
	EndDate   string          `json:"end_date"`
	Items     []memberSummary `json:"items"`
}

// GetTeamStats — GET /api/teams/{id}/training-diary-stats?season=<id>
//
// Liefert je Kadermitglied (Stamm- und erweiterter Kader) Einheiten, Minuten
// und Durchschnitts-RPE. Mitglieder ohne Einträge sind mit Nullwerten
// enthalten — die Übersicht soll die Mannschaft vollständig zeigen, nicht nur
// die Fleißigen.
func (h *Handler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	teamID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ok, err := h.canSeeTeamDiary(r.Context(), claims, teamID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	seasonID, startDate, endDate, err := h.resolveSeasonWindow(r.Context(), r.URL.Query().Get("season"))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if seasonID == 0 {
		// Keine aktive Saison und kein Parameter: leere Liste statt Fehler,
		// damit die Seite ohne Saisonpflege nicht kaputtgeht.
		writeJSON(w, http.StatusOK, teamStatsResponse{Items: []memberSummary{}})
		return
	}

	// Zuordnung über season_id der Einträge, NICHT über trained_on im
	// Saisonfenster: Einträge aus der Sommerpause tragen die (noch aktive)
	// Saison, liegen aber datumsmäßig hinter deren end_date. Ein
	// Datumsfilter würde genau sie unterschlagen.
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id,
		       m.first_name || ' ' || m.last_name,
		       COUNT(e.id),
		       COALESCE(SUM(e.duration_min), 0),
		       COALESCE(AVG(e.rpe), 0)
		  FROM members m
		  JOIN (
		        SELECT km.member_id AS member_id
		          FROM kader_members km
		          JOIN kader k ON k.id = km.kader_id
		         WHERE k.team_id = ? AND k.season_id = ?
		        UNION
		        SELECT kem.member_id AS member_id
		          FROM kader_extended_members kem
		          JOIN kader k ON k.id = kem.kader_id
		         WHERE k.team_id = ? AND k.season_id = ?
		       ) roster ON roster.member_id = m.id
		  LEFT JOIN training_diary_entries e
		         ON e.member_id = m.id AND e.season_id = ?
		 GROUP BY m.id, m.first_name, m.last_name
		 ORDER BY m.first_name, m.last_name`,
		teamID, seasonID, teamID, seasonID, seasonID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []memberSummary{}
	for rows.Next() {
		var s memberSummary
		var avg float64
		if err := rows.Scan(&s.MemberID, &s.MemberName, &s.Entries, &s.Minutes, &avg); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		// Eine Nachkommastelle; ohne Einträge bleibt 0 statt NaN.
		s.AvgRPE = math.Round(avg*10) / 10
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, teamStatsResponse{
		SeasonID:  seasonID,
		StartDate: startDate,
		EndDate:   endDate,
		Items:     items,
	})
}
