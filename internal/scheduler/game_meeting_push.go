package scheduler

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// sendGameMeetingPushes ist der Minuten-Tick für die gebündelte Meldung einer
// geänderten Treffzeit (spiel-treffpunkt). Muster wie sendEventNoteReminders,
// aber mit Netto-Vergleich: pending_game_meeting_push hält den Stand VOR der
// ersten Änderung des Fensters, gemeldet wird nur, was sich danach wirklich
// unterscheidet.
func (s *Scheduler) sendGameMeetingPushes() {
	pushed, err := s.processPendingGameMeetings()
	if err != nil {
		logIfBusy(err, "sendGameMeetingPushes")
		slog.Error("scheduler game-meeting pushes failed", "error", err)
		return
	}
	if pushed > 0 {
		slog.Info("scheduler game-meeting pushes sent", "count", pushed)
	}
}

type pendingMeeting struct {
	gameID     int
	prevOffset sql.NullInt64
	prevPlace  string
}

// processPendingGameMeetings verarbeitet alle fälligen Zeilen und löscht sie in
// jedem Fall. Gelöschte Spiele räumt ON DELETE CASCADE ab. Liefert die Anzahl
// versendeter Meldungen.
func (s *Scheduler) processPendingGameMeetings() (int, error) {
	rows, err := s.db.Query(`
		SELECT game_id, prev_offset, prev_place
		FROM pending_game_meeting_push
		WHERE notify_after <= datetime('now')`)
	if err != nil {
		return 0, err
	}
	var pending []pendingMeeting
	for rows.Next() {
		var p pendingMeeting
		if err := rows.Scan(&p.gameID, &p.prevOffset, &p.prevPlace); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, p)
	}
	rows.Close()

	today := time.Now().In(timez.Berlin()).Format("2006-01-02")
	pushed := 0
	for _, p := range pending {
		if _, err := s.db.Exec(`DELETE FROM pending_game_meeting_push WHERE game_id=?`, p.gameID); err != nil {
			logIfBusy(err, "processPendingGameMeetings.delete")
			slog.Error("scheduler game-meeting delete failed", "game_id", p.gameID, "error", err)
		}

		var date, start, opponent, eventType, place string
		var teamNames sql.NullString
		var offset sql.NullInt64
		err := s.db.QueryRow(`
			SELECT g.date, g.time, g.opponent, g.event_type, g.meet_offset_minutes, g.meet_place,
			       (SELECT GROUP_CONCAT(`+appdb.TeamLongName("t")+`, ', ')
			        FROM game_teams gt JOIN teams t ON t.id = gt.team_id WHERE gt.game_id = g.id)
			FROM games g WHERE g.id = ?`, p.gameID).
			Scan(&date, &start, &opponent, &eventType, &offset, &place, &teamNames)
		if err != nil {
			if err != sql.ErrNoRows {
				slog.Error("scheduler game-meeting load failed", "game_id", p.gameID, "error", err)
			}
			continue
		}
		date = normalizeDate(date)
		if date < today {
			continue
		}
		if offset == p.prevOffset && place == p.prevPlace {
			continue // innerhalb des Fensters zurückgenommen
		}
		if !offset.Valid && !p.prevOffset.Valid {
			continue
		}

		uids := notify.TeamAudience(s.db, s.gameTeamIDs(p.gameID)...)
		if len(uids) == 0 {
			continue
		}
		head := fmt.Sprintf("%s: %s am %s", teamNames.String, opponent, germanDate(date))
		title, body := "Treffzeit", head+" — "+meetingPhrase(date, start, offset, place)
		if !offset.Valid {
			title, body = "Treffzeit entfällt", head+" — die Treffzeit entfällt"
		}
		notify.Send(s.db, s.cfg, uids, "games", title, body, fmt.Sprintf("/termine?focus=game-%d", p.gameID))
		pushed++
	}
	return pushed, nil
}

// meetingPhrase formatiert eine Treffzeit für Meldungstexte: „Treffen 13:30 Uhr,
// Parkplatz". Liegt das Treffen vor dem Spieltag, steht „(Vortag)" dabei. Leer
// ohne Treffzeit. Geteilt von Änderungsmeldung und Spielerinnerung.
func meetingPhrase(date, start string, offset sql.NullInt64, place string) string {
	meetDate, meetTime, ok := timez.MeetTime(date, start, offset)
	if !ok {
		return ""
	}
	out := "Treffen " + meetTime + " Uhr"
	if meetDate != normalizeDate(date) {
		out += " (Vortag)"
	}
	if place != "" {
		out += ", " + place
	}
	return out
}

// germanDate wandelt "2026-10-11" in "11.10.2026".
func germanDate(date string) string {
	t, err := time.Parse("2006-01-02", normalizeDate(date))
	if err != nil {
		return date
	}
	return t.Format("02.01.2006")
}
