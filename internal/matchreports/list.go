package matchreports

import (
	"database/sql"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// ListItem ist eine kompakte Zeile für die Übersicht.
type ListItem struct {
	ID           int     `json:"id"`
	GameID       int     `json:"game_id"`
	State        string  `json:"state"`
	MatchDate    string  `json:"match_date"`
	Opponent     string  `json:"opponent"`
	PublishedURL *string `json:"published_url"`
}

// SlotItem ist ein noch offener „Spielbericht"-Slot, für den der User
// einen Draft anlegen kann.
type SlotItem struct {
	SlotID    int    `json:"slot_id"`
	GameID    int    `json:"game_id"`
	MatchDate string `json:"match_date"`
	Opponent  string `json:"opponent"`
}

// MyList liefert die Berichte des Requesters (Autor) + die noch offenen
// Spielbericht-Slots ohne Draft — die Einstiegsseite füllt daraus die
// Übersicht.
//
//	GET /api/match-reports/my
func (h *Handler) MyList(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isPressTeamOrAdmin(claims) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	reports, err := h.loadMyReports(claims.UserID)
	if err != nil {
		logErr("matchreports.MyList reports", err, "user", claims.UserID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	slots, err := h.loadMyOpenSlots(claims.UserID)
	if err != nil {
		logErr("matchreports.MyList slots", err, "user", claims.UserID)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reports":    reports,
		"open_slots": slots,
	})
}

func (h *Handler) loadMyReports(userID int) ([]ListItem, error) {
	rows, err := h.db.Query(
		`SELECT r.id, r.game_id, r.state, g.date, g.opponent, r.published_url
		 FROM match_reports r
		 JOIN games g ON g.id = r.game_id
		 WHERE r.author_user_id = ?
		 ORDER BY g.date DESC, r.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ListItem
	for rows.Next() {
		var it ListItem
		var url sql.NullString
		if err := rows.Scan(&it.ID, &it.GameID, &it.State, &it.MatchDate, &it.Opponent, &url); err != nil {
			return nil, err
		}
		if len(it.MatchDate) > 10 {
			it.MatchDate = it.MatchDate[:10]
		}
		if url.Valid {
			s := url.String
			it.PublishedURL = &s
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// loadMyOpenSlots liefert alle „Spielbericht"-Slots, die dem User zugewiesen
// sind und für die noch kein match_reports-Eintrag existiert. Nur solche
// Slots sind Kandidaten für „Neuer Bericht" — für andere gibt es entweder
// keinen Bericht-Anspruch oder er läuft bereits.
func (h *Handler) loadMyOpenSlots(userID int) ([]SlotItem, error) {
	rows, err := h.db.Query(
		`SELECT ds.id, ds.game_id, g.date, g.opponent
		 FROM duty_assignments da
		 JOIN duty_slots ds  ON ds.id  = da.duty_slot_id
		 JOIN duty_types dt  ON dt.id  = ds.duty_type_id
		 JOIN games g        ON g.id  = ds.game_id
		 LEFT JOIN match_reports mr ON mr.game_id = ds.game_id
		 WHERE da.user_id = ?
		   AND da.status IN ('assigned','fulfilled')
		   AND dt.name = 'Spielbericht'
		   AND mr.id IS NULL
		 ORDER BY g.date DESC, ds.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SlotItem
	for rows.Next() {
		var it SlotItem
		if err := rows.Scan(&it.SlotID, &it.GameID, &it.MatchDate, &it.Opponent); err != nil {
			return nil, err
		}
		if len(it.MatchDate) > 10 {
			it.MatchDate = it.MatchDate[:10]
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
