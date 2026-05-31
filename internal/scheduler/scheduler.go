package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type Scheduler struct {
	db     *sql.DB
	mailer Mailer
}

func New(db *sql.DB, m Mailer) *Scheduler { return &Scheduler{db: db, mailer: m} }

func (s *Scheduler) Run() {
	s.cleanExpiredTokens()
	s.sendDutyReminders()
}

func (s *Scheduler) cleanExpiredTokens() {
	res, err := s.db.Exec(
		`DELETE FROM invitation_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM password_reset_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM refresh_tokens WHERE expires_at < CURRENT_TIMESTAMP;`)
	if err != nil {
		log.Printf("scheduler: cleanup error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		log.Printf("scheduler: cleaned %d expired tokens", n)
	}
}

type openSlot struct {
	id         int
	eventName  string
	eventDate  string
	eventTime  string
	dutyType   string
	roleDesc   string
	slotsOpen  int
	teamID     sql.NullInt64
	targetRole string
}

type reminderUser struct {
	id    int
	email string
	name  string
}

func (s *Scheduler) sendDutyReminders() {
	targetDate := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT ds.id, ds.event_name, ds.event_date, COALESCE(ds.event_time,''),
		       dt.name, COALESCE(ds.role_desc,''),
		       ds.slots_total - ds.slots_filled, ds.team_id, dt.target_role
		FROM duty_slots ds
		JOIN duty_types dt ON dt.id = ds.duty_type_id
		WHERE ds.event_date = ?
		  AND ds.slots_filled < ds.slots_total`, targetDate)
	if err != nil {
		log.Printf("scheduler: duty reminders: query slots: %v", err)
		return
	}
	defer rows.Close()

	var slots []openSlot
	for rows.Next() {
		var sl openSlot
		rows.Scan(&sl.id, &sl.eventName, &sl.eventDate, &sl.eventTime,
			&sl.dutyType, &sl.roleDesc, &sl.slotsOpen, &sl.teamID, &sl.targetRole)
		slots = append(slots, sl)
	}
	if len(slots) == 0 {
		return
	}

	// Build user → relevant slots map
	userSlots := map[int][]openSlot{}
	userInfo := map[int]reminderUser{}

	for _, sl := range slots {
		users, err := s.eligibleUsers(sl)
		if err != nil {
			log.Printf("scheduler: duty reminders: eligible users for slot %d: %v", sl.id, err)
			continue
		}
		for _, u := range users {
			userSlots[u.id] = append(userSlots[u.id], sl)
			userInfo[u.id] = u
		}
	}

	sent := 0
	for uid, uSlots := range userSlots {
		u := userInfo[uid]

		// Skip if already reminded for this event_date
		var exists int
		s.db.QueryRow(`SELECT 1 FROM duty_reminder_log WHERE user_id=? AND event_date=?`, uid, targetDate).Scan(&exists)
		if exists == 1 {
			continue
		}

		body := buildReminderMail(u.name, targetDate, uSlots)
		subject := fmt.Sprintf("Offene Dienste am %s", formatDate(targetDate))
		if err := s.mailer.Send(u.email, subject, body); err != nil {
			log.Printf("scheduler: duty reminders: send to %s: %v", u.email, err)
			continue
		}
		s.db.Exec(`INSERT OR IGNORE INTO duty_reminder_log (user_id, event_date) VALUES (?,?)`, uid, targetDate)
		sent++
	}

	if sent > 0 {
		log.Printf("scheduler: duty reminders: sent %d reminder mails for %s", sent, targetDate)
	}
}

func (s *Scheduler) eligibleUsers(sl openSlot) ([]reminderUser, error) {
	notAssigned := `NOT EXISTS (
		SELECT 1 FROM duty_assignments da
		WHERE da.duty_slot_id = ? AND da.user_id = u.id
	)`

	var (
		query string
		args  []any
	)

	switch sl.targetRole {
	case "spieler":
		if sl.teamID.Valid {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN members m ON m.user_id = u.id
				JOIN team_memberships tm ON tm.member_id = m.id
				JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				WHERE u.role = 'spieler'
				  AND tm.team_id = ?
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				WHERE u.role = 'spieler'
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.id}
		}

	case "elternteil":
		if sl.teamID.Valid {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN family_links fl ON fl.parent_user_id = u.id
				JOIN members m ON m.id = fl.member_id
				JOIN team_memberships tm ON tm.member_id = m.id
				JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				WHERE u.role = 'elternteil'
				  AND tm.team_id = ?
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				WHERE u.role = 'elternteil'
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.id}
		}

	case "trainer":
		if sl.teamID.Valid {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN team_trainers tt ON tt.user_id = u.id
				WHERE u.role = 'trainer'
				  AND tt.team_id = ?
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				WHERE u.role = 'trainer'
				  AND u.duty_reminder_days IS NOT NULL
				  AND ` + notAssigned
			args = []any{sl.id}
		}

	default:
		query = `
			SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
			FROM users u
			WHERE u.role = ?
			  AND u.duty_reminder_days IS NOT NULL
			  AND ` + notAssigned
		args = []any{sl.targetRole, sl.id}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []reminderUser
	for rows.Next() {
		var u reminderUser
		rows.Scan(&u.id, &u.email, &u.name)
		users = append(users, u)
	}
	return users, nil
}

func buildReminderMail(name, date string, slots []openSlot) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Hallo %s,\n\n", name)
	fmt.Fprintf(&sb, "am %s sind noch folgende Dienste offen, für die du dich eintragen kannst:\n\n", formatDate(date))

	for _, sl := range slots {
		timeStr := ""
		if sl.eventTime != "" {
			timeStr = " um " + sl.eventTime + " Uhr"
		}
		fmt.Fprintf(&sb, "  • %s%s\n", sl.eventName, timeStr)
		fmt.Fprintf(&sb, "    Diensttyp: %s", sl.dutyType)
		if sl.roleDesc != "" {
			fmt.Fprintf(&sb, " – %s", sl.roleDesc)
		}
		fmt.Fprintf(&sb, "\n    Noch offene Plätze: %d\n\n", sl.slotsOpen)
	}

	sb.WriteString("Jetzt eintragen: https://intern.team-stuttgart.org/duty-board\n\n")
	sb.WriteString("Viele Grüße\nDein TeamWERK\n")
	return sb.String()
}

func formatDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02.01.2006")
}
