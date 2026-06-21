package scheduler

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/push"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type Scheduler struct {
	db     *sql.DB
	cfg    *appconfig.Config
	mailer Mailer
}

func New(db *sql.DB, cfg *appconfig.Config, m Mailer) *Scheduler {
	return &Scheduler{db: db, cfg: cfg, mailer: m}
}

func (s *Scheduler) Run() {
	s.cleanExpiredTokens()
	s.sendDutyReminders()
	s.sendGameReminders()
	s.sendTrainingReminders()
	s.sendCarpoolingReminders()
	s.recordHeartbeat()
}

// recordHeartbeat schreibt den Zeitstempel des erfolgreichen Laufs in die
// Single-Row-Tabelle monitoring_heartbeat. Reine Datenquelle für den externen
// Dead-Man-Switch (scheduler_age_sec / teamwerk_scheduler_age_seconds) — die App
// alarmiert selbst nicht.
func (s *Scheduler) recordHeartbeat() {
	if _, err := s.db.Exec(
		`INSERT INTO monitoring_heartbeat (id, updated_at) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at`,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		log.Printf("scheduler: heartbeat error: %v", err)
	}
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

	emailSent, pushSent := 0, 0
	for uid, uSlots := range userSlots {
		u := userInfo[uid]

		// Email reminder (opt-in via notification_preferences)
		if push.HasEmailEnabled(s.db, uid, "duty_reminders") {
			var exists int
			s.db.QueryRow(`SELECT 1 FROM duty_reminder_log WHERE user_id=? AND event_date=?`, uid, targetDate).Scan(&exists)
			if exists == 0 {
				body := buildReminderMail(u.name, targetDate, uSlots)
				subject := fmt.Sprintf("Offene Dienste am %s", formatDate(targetDate))
				if err := s.mailer.Send(u.email, subject, body); err != nil {
					log.Printf("scheduler: duty reminders: send mail to %s: %v", u.email, err)
				} else {
					s.db.Exec(`INSERT OR IGNORE INTO duty_reminder_log (user_id, event_date) VALUES (?,?)`, uid, targetDate)
					emailSent++
				}
			}
		}

		// Push reminder (opt-in via notification_preferences, default: enabled)
		pushUsers := push.FilterByPushPref(s.db, []int{uid}, "duty_reminders")
		if len(pushUsers) == 0 {
			continue
		}
		// Idempotency via notification_log: INSERT first, then check RowsAffected.
		// This prevents double-send when two cron instances run concurrently.
		res, _ := s.db.Exec(`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
			uid, "duty_reminder", hashDate(targetDate))
		if n, _ := res.RowsAffected(); n == 1 {
			go push.SendToUsers(s.db, s.cfg, []int{uid},
				"Offene Dienste", "Am "+formatDate(targetDate)+" gibt es noch offene Dienste", "/dienste")
			pushSent++
		}
	}

	if emailSent > 0 {
		log.Printf("scheduler: duty reminders: sent %d email(s) for %s", emailSent, targetDate)
	}
	if pushSent > 0 {
		log.Printf("scheduler: duty reminders: sent %d push(s) for %s", pushSent, targetDate)
	}
}

// hashDate converts a YYYY-MM-DD string to an integer for use as ref_id.
func hashDate(date string) int {
	// Format: YYYYMMDD as integer
	clean := strings.ReplaceAll(date, "-", "")
	var n int
	fmt.Sscan(clean, &n)
	return n
}

func (s *Scheduler) sendGameReminders() {
	// Target: games starting in 20-28h (generous window to handle minute-level cron)
	now := time.Now()
	from := now.Add(20 * time.Hour).Format("2006-01-02")
	to := now.Add(28 * time.Hour).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT g.id, g.opponent, g.date, g.time, gt.team_id
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		WHERE g.date BETWEEN ? AND ?`, from, to)
	if err != nil {
		log.Printf("scheduler: game reminders: %v", err)
		return
	}
	defer rows.Close()

	type gameRow struct {
		id       int
		opponent string
		date     string
		time     string
		teamID   int
	}
	var games []gameRow
	for rows.Next() {
		var g gameRow
		rows.Scan(&g.id, &g.opponent, &g.date, &g.time, &g.teamID)
		games = append(games, g)
	}

	sent := 0
	for _, g := range games {
		uids := s.unsentUIDs(s.teamMembersAndParents(g.teamID), "game_reminder", g.id)
		if len(uids) == 0 {
			continue
		}
		notify.Send(s.db, s.cfg, uids, "games",
			"Spielerinnerung", g.opponent+" — morgen um "+g.time+" Uhr", fmt.Sprintf("/termine?focus=game-%d", g.id))
		for _, uid := range uids {
			s.db.Exec(`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
				uid, "game_reminder", g.id)
			sent++
		}
	}
	if sent > 0 {
		log.Printf("scheduler: game reminders: sent %d notification(s)", sent)
	}
}

// unsentUIDs filters out users that already have a notification_log row
// matching (refType, refID).
func (s *Scheduler) unsentUIDs(uids []int, refType string, refID int) []int {
	out := uids[:0:len(uids)]
	for _, uid := range uids {
		var exists int
		s.db.QueryRow(`SELECT 1 FROM notification_log WHERE user_id=? AND ref_type=? AND ref_id=?`,
			uid, refType, refID).Scan(&exists)
		if exists == 0 {
			out = append(out, uid)
		}
	}
	return out
}

func (s *Scheduler) sendTrainingReminders() {
	now := time.Now()
	from := now.Add(20 * time.Hour).Format("2006-01-02")
	to := now.Add(28 * time.Hour).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT ts.id, ts.team_id, COALESCE(ts.title,'Training'), ts.date, ts.start_time
		FROM training_sessions ts
		WHERE ts.date BETWEEN ? AND ?
		  AND ts.status = 'active'`, from, to)
	if err != nil {
		log.Printf("scheduler: training reminders: %v", err)
		return
	}
	defer rows.Close()

	type sessionRow struct {
		id        int
		teamID    int
		title     string
		date      string
		startTime string
	}
	var sessions []sessionRow
	for rows.Next() {
		var s sessionRow
		rows.Scan(&s.id, &s.teamID, &s.title, &s.date, &s.startTime)
		sessions = append(sessions, s)
	}

	sent := 0
	for _, sess := range sessions {
		uids := s.unsentUIDs(s.teamMembersAndParents(sess.teamID), "training_reminder", sess.id)
		if len(uids) == 0 {
			continue
		}
		notify.Send(s.db, s.cfg, uids, "trainings",
			"Trainingserinnerung", sess.title+" — morgen um "+sess.startTime+" Uhr", fmt.Sprintf("/termine?focus=training-%d", sess.id))
		for _, uid := range uids {
			s.db.Exec(`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
				uid, "training_reminder", sess.id)
			sent++
		}
	}
	if sent > 0 {
		log.Printf("scheduler: training reminders: sent %d notification(s)", sent)
	}
}

func (s *Scheduler) sendCarpoolingReminders() {
	now := time.Now()
	from := now.Add(2 * time.Hour).Format("2006-01-02 15:04")
	to := now.Add(4 * time.Hour).Format("2006-01-02 15:04")

	// Find games in ~3h that have confirmed pairings
	rows, err := s.db.Query(`
		SELECT DISTINCT mg.user_id, g.id, g.opponent, g.date, g.time
		FROM mitfahrgelegenheiten mg
		JOIN games g ON g.id = mg.game_id
		JOIN mitfahrt_paarungen p ON (p.biete_id = mg.id OR p.suche_id = mg.id) AND p.status = 'confirmed'
		WHERE (g.date || ' ' || g.time) BETWEEN ? AND ?`, from, to)
	if err != nil {
		log.Printf("scheduler: carpooling reminders: %v", err)
		return
	}
	defer rows.Close()

	type carpoolRow struct {
		userID   int
		gameID   int
		opponent string
		date     string
		time     string
	}
	var entries []carpoolRow
	for rows.Next() {
		var c carpoolRow
		rows.Scan(&c.userID, &c.gameID, &c.opponent, &c.date, &c.time)
		entries = append(entries, c)
	}

	sent := 0
	for _, c := range entries {
		uids := s.unsentUIDs([]int{c.userID}, "carpooling_reminder", c.gameID)
		if len(uids) == 0 {
			continue
		}
		notify.Send(s.db, s.cfg, uids, "carpooling",
			"Fahrgemeinschaft heute", c.opponent+" — Abfahrt um "+c.time+" Uhr", "/mitfahrgelegenheiten")
		s.db.Exec(`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
			c.userID, "carpooling_reminder", c.gameID)
		sent++
	}
	if sent > 0 {
		log.Printf("scheduler: carpooling reminders: sent %d notification(s)", sent)
	}
}

// teamMembersAndParents returns user IDs of kader members + parents for a team in the active season.
func (s *Scheduler) teamMembersAndParents(teamID int) []int {
	rows, err := s.db.Query(
		`SELECT DISTINCT u.id FROM users u
		 JOIN members m ON m.user_id = u.id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons se ON se.id = pm.season_id AND se.is_active = 1
		 WHERE pm.team_id = ?
		 UNION
		 SELECT DISTINCT fl.parent_user_id FROM family_links fl
		 JOIN members m ON m.id = fl.member_id
		 JOIN player_memberships pm ON pm.member_id = m.id
		 JOIN seasons se ON se.id = pm.season_id AND se.is_active = 1
		 WHERE pm.team_id = ?`, teamID, teamID)
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
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'
				JOIN player_memberships tm ON tm.member_id = m.id
				JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				WHERE tm.team_id = ?
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN members m ON m.user_id = u.id
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'
				WHERE ` + notAssigned
			args = []any{sl.id}
		}

	case "elternteil":
		if sl.teamID.Valid {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN family_links fl ON fl.parent_user_id = u.id
				JOIN members m ON m.id = fl.member_id
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'
				JOIN player_memberships tm ON tm.member_id = m.id
				JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
				WHERE tm.team_id = ?
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN family_links fl ON fl.parent_user_id = u.id
				JOIN members m ON m.id = fl.member_id
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'
				WHERE ` + notAssigned
			args = []any{sl.id}
		}

	case "trainer":
		if sl.teamID.Valid {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN members m ON m.user_id = u.id
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'trainer'
				JOIN kader_trainers kt ON kt.member_id = m.id
				JOIN kader k ON k.id = kt.kader_id
				WHERE k.team_id = ?
				  AND ` + notAssigned
			args = []any{sl.teamID.Int64, sl.id}
		} else {
			query = `
				SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
				FROM users u
				JOIN members m ON m.user_id = u.id
				JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'trainer'
				WHERE ` + notAssigned
			args = []any{sl.id}
		}

	default:
		// Andere target_role-Werte (vorstand, sportliche_leitung, vorstand_beisitzer, kassierer)
		// werden über die Vereinsfunktion aufgelöst.
		query = `
			SELECT DISTINCT u.id, u.email, u.first_name || ' ' || u.last_name
			FROM users u
			JOIN members m ON m.user_id = u.id
			JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = ?
			WHERE ` + notAssigned
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

	sb.WriteString("Jetzt eintragen: https://internal.team-stuttgart.org/duty-board\n\n")
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
