package videos

import (
	"net/http"
	"sort"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// EligibleGames liefert die Spiele, für die der aufrufende Nutzer aktuell ein
// Video hochladen darf: Spiele der Teams mit Rollen-Berechtigung
// (CanUploadToTeam — Trainer des Teams, sportliche Leitung, Vorstand, Admin)
// vereinigt mit Spielen, für die eine Video-Upload-Dienst-Zuweisung existiert
// (video-download-duty-upload). GET /api/videos/upload-eligible-games
// (Authenticated-Tier).
//
// Dies ist ein reiner Vorauswahl-Endpoint für Client-UIs (Desktop-Tool). Die
// eigentliche Autorisierung bleibt serverseitig in CreateUpload — diese Liste
// darf nicht großzügiger sein als jene Prüfung, sonst zeigt der Client ein
// Spiel an, für das der Upload dann mit 403 scheitert.
func (h *Handler) EligibleGames(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ids := map[int]bool{}

	switch {
	case claims.Role == "admin" || claims.HasAnyFunction("vorstand", "sportliche_leitung"):
		// Teamübergreifend berechtigt (siehe CanUploadToTeam) → alle Spiele.
		rows, err := h.db.Query(`SELECT id FROM games`)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := collectGameIDs(rows, ids); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case claims.HasFunction("trainer"):
		rows, err := h.db.Query(`
			SELECT DISTINCT gt.game_id
			FROM game_teams gt
			JOIN trainer_memberships tm ON tm.team_id = gt.team_id
			JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ?`, claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := collectGameIDs(rows, ids); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Dienst-Pfad: unabhängig vom Rollen-Pfad, immer zusätzlich geprüft (ein
	// Spieler ohne jede Vereinsfunktion landet sonst nie in obigen Zweigen).
	dutyRows, err := h.db.Query(`
		SELECT DISTINCT ds.game_id
		FROM duty_assignments da
		JOIN duty_slots ds ON ds.id = da.duty_slot_id AND ds.game_id IS NOT NULL
		JOIN duty_types dt ON dt.id = ds.duty_type_id
		WHERE da.user_id = ? AND dt.grants_video_upload = 1`, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := collectGameIDs(dutyRows, ids); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	result := make([]int, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Ints(result)
	writeJSON(w, map[string]any{"game_ids": result})
}

// collectGameIDs scannt eine Query mit einer einzelnen int-Spalte in die
// gegebene Menge und schließt die rows.
func collectGameIDs(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}, into map[int]bool) error {
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		into[id] = true
	}
	return rows.Err()
}
