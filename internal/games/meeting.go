package games

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// Grenzen der Treffzeit (spiel-treffpunkt). Der Abstand-Deckel fängt Tippfehler
// wie 01:30 statt 13:30 ab, ohne frühe Turnier-Abfahrten zu verbieten; dieselben
// Werte stehen als CHECK in Migration 073.
const (
	maxMeetOffsetMinutes = 720
	maxMeetPlaceRunes    = 100
)

// Meeting sind die abgeleiteten Treffzeit-Felder eines Spiels, wie sie jede
// Spiel-Antwort neben `time` trägt. Gespeichert ist nur der Abstand zum Anwurf;
// meet_time/meet_date entstehen bei jedem Lesen über timez.MeetTime.
type Meeting struct {
	MeetTime  *string `json:"meet_time"`
	MeetDate  *string `json:"meet_date"`
	MeetPlace string  `json:"meet_place"`
}

func newMeeting(date, start string, offset sql.NullInt64, place string) Meeting {
	d, hm, ok := timez.MeetTime(date, start, offset)
	if !ok {
		return Meeting{MeetPlace: ""}
	}
	return Meeting{MeetTime: &hm, MeetDate: &d, MeetPlace: place}
}

// PUT /api/games/{id}/meeting — setzt oder entfernt Treffzeit und Treffpunkt-Ort.
// Berechtigung wie beim Hinweistext (canEditGameInfo). Die Uhrzeit wird gegen
// den gespeicherten Anwurf in einen Abstand umgerechnet; eine Änderung landet
// gebündelt in pending_game_meeting_push (Scheduler meldet den Netto-Unterschied).
func (h *Handler) UpdateGameMeeting(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	gameID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return
	}
	var req struct {
		MeetTime  string `json:"meet_time"`
		MeetPlace string `json:"meet_place"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidBody, nil)
		return
	}

	var date, start, prevPlace string
	var prevOffset sql.NullInt64
	err = h.db.QueryRowContext(r.Context(),
		`SELECT date, time, meet_offset_minutes, meet_place FROM games WHERE id=?`, gameID).
		Scan(&date, &start, &prevOffset, &prevPlace)
	if err == sql.ErrNoRows {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, nil)
		return
	} else if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if !h.canEditGameInfo(r.Context(), claims, gameID) {
		httpx.WriteError(w, r, http.StatusForbidden, httpx.CodeForbidden, nil)
		return
	}

	place := strings.TrimSpace(req.MeetPlace)
	if utf8.RuneCountInString(place) > maxMeetPlaceRunes {
		httpx.WriteError(w, r, http.StatusBadRequest, "meet_place_too_long", nil)
		return
	}
	var offset sql.NullInt64
	if strings.TrimSpace(req.MeetTime) == "" {
		if place != "" {
			httpx.WriteError(w, r, http.StatusBadRequest, "meet_place_without_time", nil)
			return
		}
	} else {
		meetMin, ok := timez.ParseHHMM(strings.TrimSpace(req.MeetTime))
		if !ok {
			httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeValidation, nil)
			return
		}
		startMin, ok := timez.ParseHHMM(start)
		if !ok {
			// Kein auswertbarer Anwurf — eine Treffzeit davor wäre erfunden.
			httpx.WriteError(w, r, http.StatusBadRequest, "meet_after_start", nil)
			return
		}
		diff := startMin - meetMin
		if diff < 0 {
			httpx.WriteError(w, r, http.StatusBadRequest, "meet_after_start", nil)
			return
		}
		if diff > maxMeetOffsetMinutes {
			httpx.WriteError(w, r, http.StatusBadRequest, "meet_offset_out_of_range", nil)
			return
		}
		offset = sql.NullInt64{Int64: int64(diff), Valid: true}
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(r.Context(),
		`UPDATE games SET meet_offset_minutes = ?, meet_place = ? WHERE id = ?`,
		offset, place, gameID); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	// prev_* nur beim ersten Eintrag des Fensters: der Scheduler vergleicht den
	// Stand vor der ersten Korrektur mit dem zum Sendezeitpunkt.
	if _, err = tx.ExecContext(r.Context(), `
		INSERT INTO pending_game_meeting_push (game_id, prev_offset, prev_place, notify_after, updated_by)
		VALUES (?, ?, ?, datetime('now', '+5 minutes'), ?)
		ON CONFLICT(game_id) DO UPDATE SET
			notify_after = excluded.notify_after,
			updated_by   = excluded.updated_by`,
		gameID, prevOffset, prevPlace, claims.UserID); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if err = tx.Commit(); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}

	h.broadcastGame(r.Context(), gameID, "games")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newMeeting(date, start, offset, place))
}
