package scheduler

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// sendAttendanceReminders ist der Scheduler-Hook für die täglichen
// Anwesenheits-Erinnerungs-Pushes an Trainer mit offenen Erfassungen.
// Wird einmal pro Cron-Tick aufgerufen; idempotent pro (Trainer × Tag).
//
// Quelle der Wahrheit: openspec/changes/anwesenheits-statistik/design.md D6.
func (s *Scheduler) sendAttendanceReminders() {
	s.sendAttendanceRemindersAt(time.Now().In(timez.Berlin()))
}

// sendAttendanceRemindersAt ist die testbare Variante mit explizitem
// „now"-Zeitpunkt (lokale Berliner Wallclock).
func (s *Scheduler) sendAttendanceRemindersAt(now time.Time) {
	// Tageszeit-Gate: erst nach 19:00 lokal senden, damit der Trainer nach
	// dem typischen Trainingstag erinnert wird. Idempotenz unten verhindert
	// Doppelversand bei späteren Cron-Ticks.
	if now.Hour() < 19 {
		return
	}

	// Aktive Saison ermitteln (Cut-off: existiert keine, ist Schluss).
	var seasonID int
	var startDate string
	row := s.db.QueryRow(`SELECT id, start_date FROM seasons WHERE is_active = 1`)
	if err := row.Scan(&seasonID, &startDate); err != nil {
		// Keine aktive Saison → nichts zu tun.
		return
	}

	openEvents, err := s.loadOpenAttendanceEvents(seasonID, startDate)
	if err != nil {
		logIfBusy(err, "sendAttendanceReminders.load")
		slog.Error("attendance reminders load failed", "error", err)
		return
	}
	if len(openEvents) == 0 {
		return
	}

	today := now.Format("2006-01-02")
	refID := hashDate(today)

	sent := 0
	for _, perTrainer := range groupByTrainer(openEvents) {
		userID := perTrainer.userID
		// Präferenz respektieren: Trainer mit deaktiviertem 'operativ' vorab
		// aussortieren (vor notification_log), damit ein späteres Wieder-
		// Aktivieren künftige Läufe wieder erfasst.
		if len(push.FilterByPushPref(s.db, []int{userID}, "operativ")) == 0 {
			continue
		}
		// Idempotenz: pro User+heute genau eine Push, INSERT OR IGNORE.
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
			userID, "attendance-reminder", refID)
		if err != nil {
			logIfBusy(err, "sendAttendanceReminders.claim")
			continue
		}
		if n, _ := res.RowsAffected(); n != 1 {
			continue // bereits heute gesendet
		}

		title := "Anwesenheiten fehlen"
		body := buildAttendanceReminderBody(perTrainer.events)
		url := fmt.Sprintf("/team/%d/anwesenheit", perTrainer.events[0].teamID)
		go push.SendToUsers(s.db, s.cfg, []int{userID}, title, body, url)
		sent++
	}

	if sent > 0 {
		slog.Info("attendance reminders sent", "count", sent, "date", today)
	}
}

// openAttendanceEvent ist eine offene Termin-Zeile aus Sicht eines Trainers.
type openAttendanceEvent struct {
	userID    int
	teamID    int
	teamName  string
	eventType string // "training" | "game"
	eventID   int
	eventDate string // YYYY-MM-DD
}

// loadOpenAttendanceEvents liefert alle (Trainer × offener Termin)-Tupel
// für die übergebene Saison. Die Aggregation pro Trainer findet in Go statt.
func (s *Scheduler) loadOpenAttendanceEvents(seasonID int, startDate string) ([]openAttendanceEvent, error) {
	rows, err := s.db.Query(`
		WITH trainers AS (
			SELECT DISTINCT u.id AS user_id, t.id AS team_id, t.name AS team_name
			FROM kader k
			JOIN teams t ON t.id = k.team_id
			JOIN kader_trainers kt ON kt.kader_id = k.id
			JOIN members m ON m.id = kt.member_id
			JOIN users u ON u.id = m.user_id
			WHERE k.season_id = ?
		)
		SELECT tr.user_id, tr.team_id, tr.team_name, 'training' AS ev_type, ts.id, ts.date
		FROM trainers tr
		JOIN training_sessions ts ON ts.team_id = tr.team_id
		                          AND ts.season_id = ?
		                          AND ts.status != 'cancelled'
		                          AND date(ts.date) >= date(?)
		                          AND date(ts.date) < date('now')
		                          AND NOT EXISTS (
		                            SELECT 1 FROM training_attendances ta WHERE ta.training_id = ts.id
		                          )
		UNION ALL
		SELECT tr.user_id, tr.team_id, tr.team_name, 'game', g.id, g.date
		FROM trainers tr
		JOIN game_teams gt ON gt.team_id = tr.team_id
		JOIN games g ON g.id = gt.game_id
		             AND g.season_id = ?
		             AND date(g.date) >= date(?)
		             AND date(g.date) < date('now')
		             AND NOT EXISTS (
		               SELECT 1 FROM game_attendances ga WHERE ga.game_id = g.id
		             )
		ORDER BY user_id, 6, 5`, // user_id, date, event_id
		seasonID, seasonID, startDate, seasonID, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []openAttendanceEvent
	for rows.Next() {
		var e openAttendanceEvent
		if err := rows.Scan(&e.userID, &e.teamID, &e.teamName, &e.eventType, &e.eventID, &e.eventDate); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// trainerEvents bündelt alle offenen Termine eines Trainers.
type trainerEvents struct {
	userID int
	events []openAttendanceEvent
}

func groupByTrainer(events []openAttendanceEvent) []trainerEvents {
	grouped := map[int]*trainerEvents{}
	for _, ev := range events {
		g, ok := grouped[ev.userID]
		if !ok {
			g = &trainerEvents{userID: ev.userID}
			grouped[ev.userID] = g
		}
		g.events = append(g.events, ev)
	}
	// Deterministische Reihenfolge für Tests + Logs.
	out := make([]trainerEvents, 0, len(grouped))
	for _, g := range grouped {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].userID < out[j].userID })
	return out
}

// buildAttendanceReminderBody formatiert den Push-Body:
// "N offene Erfassungen: <Team> <Wd DD.MM.> (Training|Spiel), …".
// Maximal 3 Termine explizit, der Rest als "… und K weitere".
func buildAttendanceReminderBody(events []openAttendanceEvent) string {
	n := len(events)
	limit := 3
	if n < limit {
		limit = n
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		ev := events[i]
		parts = append(parts, fmt.Sprintf("%s %s (%s)",
			ev.teamName,
			formatReminderDate(ev.eventDate),
			eventTypeLabel(ev.eventType)))
	}
	body := fmt.Sprintf("%d offene Erfassungen: %s", n, joinWithSep(parts, ", "))
	if n > limit {
		body += fmt.Sprintf(" … und %d weitere", n-limit)
	}
	return body
}

func eventTypeLabel(t string) string {
	if t == "game" {
		return "Spiel"
	}
	return "Training"
}

// formatReminderDate liefert "Wd DD.MM." in deutscher Notation (z.B. "Di 14.10.").
func formatReminderDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	wd := []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}[t.Weekday()]
	return fmt.Sprintf("%s %02d.%02d.", wd, t.Day(), int(t.Month()))
}

func joinWithSep(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
