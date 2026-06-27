package scheduler

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/notify"
)

// sendEventNoteReminders ist der Minuten-Tick für die debounced Termin-Hinweise.
// Fällige pending-Rows werden verarbeitet (Push nur für zukünftige Events) und
// in jedem Fall gelöscht.
func (s *Scheduler) sendEventNoteReminders() {
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		logIfBusy(err, "sendEventNoteReminders")
		slog.Error("scheduler event-note reminders failed", "error", err)
		return
	}
	if pushed > 0 {
		slog.Info("scheduler event-note reminders sent", "count", pushed)
	}
}

type pendingNote struct {
	refType string
	refID   int
	note    string
}

// processPendingEventNotes verarbeitet alle fälligen (notify_after <= now)
// pending-Rows. Für ein Event mit event_date >= heute (und noch vorhandenem
// Event) wird ein Push an teamMembersAndParents abgesetzt; in jedem Fall wird
// die Row gelöscht. Liefert die Anzahl tatsächlich versendeter Pushes.
func (s *Scheduler) processPendingEventNotes() (int, error) {
	rows, err := s.db.Query(`
		SELECT ref_type, ref_id, note_text
		FROM pending_event_notes_push
		WHERE notify_after <= datetime('now')`)
	if err != nil {
		return 0, err
	}
	var pending []pendingNote
	for rows.Next() {
		var p pendingNote
		if err := rows.Scan(&p.refType, &p.refID, &p.note); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, p)
	}
	rows.Close()

	today := time.Now().Format("2006-01-02")
	pushed := 0
	for _, p := range pending {
		eventDate, teamIDs, title, url := s.resolveEventNote(p)

		// Row immer löschen — idempotent, auch ohne Push.
		if _, err := s.db.Exec(
			`DELETE FROM pending_event_notes_push WHERE ref_type=? AND ref_id=?`,
			p.refType, p.refID); err != nil {
			logIfBusy(err, "processPendingEventNotes.delete")
			slog.Error("scheduler event-note delete failed", "ref_type", p.refType, "ref_id", p.refID, "error", err)
		}

		// Kein Push für gelöschtes (leeres Datum) oder vergangenes Event.
		if eventDate == "" || eventDate < today {
			continue
		}
		uids := s.teamMembersAndParentsMulti(teamIDs)
		if len(uids) == 0 {
			continue
		}
		category := "trainings"
		if p.refType == "game" {
			category = "games"
		}
		notify.Send(s.db, s.cfg, uids, category, title, p.note, url)
		pushed++
	}
	return pushed, nil
}

// resolveEventNote liest Datum, beteiligte Teams, Push-Titel und Deep-Link für
// eine pending-Row. Ein nicht (mehr) existierendes Event liefert ein leeres
// Datum, sodass der Aufrufer den Push überspringt.
func (s *Scheduler) resolveEventNote(p pendingNote) (eventDate string, teamIDs []int, title, url string) {
	switch p.refType {
	case "training":
		var date, sessTitle sql.NullString
		var teamID sql.NullInt64
		err := s.db.QueryRow(
			`SELECT date, team_id, COALESCE(NULLIF(title,''),'Training')
			 FROM training_sessions WHERE id=?`, p.refID).Scan(&date, &teamID, &sessTitle)
		if err != nil || !date.Valid {
			return "", nil, "", ""
		}
		if teamID.Valid {
			teamIDs = []int{int(teamID.Int64)}
		}
		return normalizeDate(date.String), teamIDs,
			"Hinweis zu " + sessTitle.String,
			fmt.Sprintf("/termine?focus=training-%d", p.refID)

	case "game":
		var date, opponent sql.NullString
		err := s.db.QueryRow(
			`SELECT date, COALESCE(opponent,'') FROM games WHERE id=?`, p.refID).Scan(&date, &opponent)
		if err != nil || !date.Valid {
			return "", nil, "", ""
		}
		return normalizeDate(date.String), s.gameTeamIDs(p.refID),
			"Hinweis zu " + opponent.String,
			fmt.Sprintf("/termine?focus=game-%d", p.refID)
	}
	return "", nil, "", ""
}

// normalizeDate schneidet einen evtl. als ISO-Timestamp gelieferten Datumswert
// auf YYYY-MM-DD zu (vgl. Gotcha SQLite DATE-Felder).
func normalizeDate(v string) string {
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}

// gameTeamIDs liefert die Team-IDs eines Games.
func (s *Scheduler) gameTeamIDs(gameID int) []int {
	rows, err := s.db.Query(`SELECT team_id FROM game_teams WHERE game_id=?`, gameID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

// teamMembersAndParentsMulti vereinigt die Empfänger mehrerer Teams (dedupliziert).
func (s *Scheduler) teamMembersAndParentsMulti(teamIDs []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, tid := range teamIDs {
		for _, uid := range s.teamMembersAndParents(tid) {
			if !seen[uid] {
				seen[uid] = true
				out = append(out, uid)
			}
		}
	}
	return out
}
