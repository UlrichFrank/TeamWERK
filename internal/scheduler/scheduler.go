package scheduler

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/health"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// logIfBusy emittiert ein strukturiertes slog.Warn, falls err einen SQLITE_BUSY-
// Return des SQLite-Treibers darstellt. Die App alarmiert selbst nicht — der
// Log-Record (event="sqlite_busy", source="scheduler") ist das Signal, das ein
// externer Log-Collector per Query alarmierbar macht. Wir zählen NICHT in
// teamwerk_sqlite_busy_total: Scheduler ist ein separater Prozess, der
// In-Memory-Counter im HTTP-Daemon würde davon nichts mitbekommen.
func logIfBusy(err error, op string) {
	if health.IsSQLiteBusy(err) {
		slog.Warn("sqlite_busy",
			"event", "sqlite_busy",
			"source", "scheduler",
			"op", op,
			"error", err.Error(),
		)
	}
}

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
	s.sendEventNoteReminders()
	s.cleanStaleVideoUploads()
	s.failStaleVideoUploads()
	s.cleanFailedVideoRaw()
	s.runVideoRetention()
	s.sendAttendanceReminders()
	s.sendMatchReportReviewReminders()
	s.recordHeartbeat()
}

// sendMatchReportReviewReminders erinnert die Freigeber (Vereinsfunktion
// 'medien' oder 'vorstand'), wenn ein Spielbericht länger als 5 Tage im
// State 'pending_review' liegt. Genau eine Reminder-Notification pro
// (user, report) — idempotent via notification_log.
//
// Query-Logik (siehe spielbericht-medien-gate design.md D-5):
//   - Alle Berichte state='pending_review' AND submitted_at < now - 5 days
//   - × alle aktuellen Freigeber (users mit member_club_functions IN medien/vorstand)
//   - INSERT OR IGNORE INTO notification_log für Idempotenz.
func (s *Scheduler) sendMatchReportReviewReminders() {
	rows, err := s.db.Query(`
		SELECT r.id, g.opponent
		FROM match_reports r
		JOIN games g ON g.id = r.game_id
		WHERE r.state = 'pending_review'
		  AND r.submitted_at IS NOT NULL
		  AND datetime(r.submitted_at) < datetime('now','-5 days')`)
	if err != nil {
		logIfBusy(err, "sendMatchReportReviewReminders.query")
		slog.Error("scheduler match-report reminder query failed", "error", err)
		return
	}
	type pendingReport struct {
		id       int
		opponent string
	}
	var reports []pendingReport
	for rows.Next() {
		var pr pendingReport
		if err := rows.Scan(&pr.id, &pr.opponent); err != nil {
			continue
		}
		reports = append(reports, pr)
	}
	rows.Close()
	if len(reports) == 0 {
		return
	}

	reviewers := s.matchReportReviewers()
	if len(reviewers) == 0 {
		return
	}

	sent := 0
	for _, pr := range reports {
		title := "Spielbericht wartet auf Freigabe"
		body := fmt.Sprintf("„%s\" liegt seit über 5 Tagen zur Prüfung.", pr.opponent)
		url := fmt.Sprintf("/berichte/%d", pr.id)
		for _, uid := range reviewers {
			res, err := s.db.Exec(
				`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
				uid, "match_report_review_reminder", pr.id)
			if err != nil {
				logIfBusy(err, "sendMatchReportReviewReminders.claim")
				continue
			}
			if n, _ := res.RowsAffected(); n == 1 {
				go push.SendToUsers(s.db, s.cfg, []int{uid}, title, body, url)
				sent++
			}
		}
	}
	if sent > 0 {
		slog.Info("scheduler match-report review reminders sent", "count", sent)
	}
}

// matchReportReviewers listet aktuelle Freigeber (User mit Vereinsfunktion
// 'medien' oder 'vorstand'). Rein lesende Query — keine Notification-Nebenwirkung.
func (s *Scheduler) matchReportReviewers() []int {
	rows, err := s.db.Query(`
		SELECT DISTINCT u.id
		FROM users u
		JOIN members m ON m.user_id = u.id
		JOIN member_club_functions mcf ON mcf.member_id = m.id
		WHERE mcf.function IN ('medien','vorstand')`)
	if err != nil {
		logIfBusy(err, "matchReportReviewers.query")
		slog.Error("scheduler match-report reviewers query failed", "error", err)
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// cleanStaleVideoUploads entfernt unfertige tus-Sessions (>24 h) im
// Video-uploads-Verzeichnis. Idempotent; ein fehlendes Verzeichnis ist kein
// Fehler (Video-Feature noch nie genutzt / Storage nicht angelegt).
//
// Die Logik ist hier inline (statt videos.CleanupStaleUploads aufzurufen): der
// Scheduler ist ein Foundation-Package und darf das Domain-Package videos nicht
// importieren (Architektur-Test). Reine Filesystem-Operation ohne Domänenwissen.
func (s *Scheduler) cleanStaleVideoUploads() {
	dir := filepath.Join(s.cfg.VideoStorageDir, "uploads")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		slog.Error("scheduler stale-upload cleanup failed", "error", err)
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		slog.Info("stale video uploads cleaned", "count", removed)
	}
}

// failStaleVideoUploads markiert Videos, die länger als 24 h im Status 'uploading'
// verharren, als 'failed'. Ohne diesen Job bleiben abgebrochene/verwaiste Uploads
// (z.B. nach Netzwerkabbruch oder wenn ein frischer Upload eine andere Zeile bespielt)
// dauerhaft als Geister-Eintrag „Wird hochgeladen" in der Liste stehen. Der 24-h-Cutoff
// deckt sich mit dem Stale-tus-Session-Cleanup; ein legitimer Upload (Hard-Limit 2,5 GB)
// dauert nie annähernd so lange. Inline statt Aufruf ins Domain-Package videos (Scheduler
// ist Foundation, darf videos nicht importieren — Architektur-Test).
func (s *Scheduler) failStaleVideoUploads() {
	res, err := s.db.Exec(
		`UPDATE videos
		 SET status = 'failed', failure_reason = 'Upload abgebrochen'
		 WHERE status = 'uploading'
		   AND created_at < datetime('now', '-24 hours')`)
	if err != nil {
		logIfBusy(err, "failStaleVideoUploads")
		slog.Error("scheduler stale uploading-video cleanup failed", "error", err)
		return
	}
	if n, err := res.RowsAffected(); err == nil && n > 0 {
		slog.Info("stale uploading videos marked failed", "count", n)
	}
}

// cleanFailedVideoRaw löscht die Roh-Uploads (raw/{id}.mp4) von Videos, die seit
// mehr als 7 Tagen den Status 'failed' haben. Bei Transcode-Fehlern bleibt die
// Rohdatei zunächst für Debug erhalten (siehe videos.Worker.fail); nach 7 Tagen
// wird sie hier aufgeräumt. Die DB-Zeile bleibt bestehen.
//
// Inline (statt Aufruf ins Domain-Package videos): der Scheduler ist
// Foundation und darf videos nicht importieren (Architektur-Test). Reine
// DB-Lese- + Filesystem-Operation, das raw/-Pfadschema ist trivial.
func (s *Scheduler) cleanFailedVideoRaw() {
	rows, err := s.db.Query(
		`SELECT id FROM videos
		 WHERE status = 'failed'
		   AND created_at < datetime('now', '-7 days')`)
	if err != nil {
		logIfBusy(err, "cleanFailedVideoRaw")
		slog.Error("scheduler failed-video raw cleanup query failed", "error", err)
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	removed := 0
	for _, id := range ids {
		raw := filepath.Join(s.cfg.VideoStorageDir, "raw", fmt.Sprintf("%d.mp4", id))
		if err := os.Remove(raw); err == nil {
			removed++
		} else if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("failed-video raw cleanup: remove failed", "video_id", id, "error", err)
		}
	}
	if removed > 0 {
		slog.Info("failed video raw files cleaned", "count", removed)
	}
}

// runVideoRetention ist der tägliche Saison-basierte Retention-Job (Ziel 03:00
// lokal, faktisch jede Minute aufgerufen — wie die übrigen Daily-Jobs nicht über
// eine Uhrzeit, sondern über Idempotenz gegen Doppel-Arbeit abgesichert):
//
//  1. T-7-Vorwarnung: genau 7 Tage vor der Löschung (Saisonende = heute - 83 d)
//     erhalten alle Trainer des Video-Teams einen Push. Idempotent über
//     notification_log (kind 'video_retention_warning', ref_id = video id).
//  2. Löschung: Videos, deren Saison vor mehr als 90 Tagen endete, werden
//     entfernt (DB-Zeile + raw/{id}.mp4 + processed/{id}/). Diese Operation ist
//     von Natur aus idempotent — gelöschte Zeilen tauchen nicht erneut auf.
//
// Videos mit season_end_date IS NULL werden NIE automatisch gelöscht. Inline
// statt Aufruf ins Domain-Package videos (Scheduler ist Foundation, darf videos
// nicht importieren — Architektur-Test); das Pfadschema raw/{id}.mp4 und
// processed/{id} wird trivial nachgebaut (vgl. internal/videos/paths.go).
func (s *Scheduler) runVideoRetention() {
	s.sendVideoRetentionWarnings()
	s.deleteRetainedVideos()
}

// sendVideoRetentionWarnings schickt die T-7-Vorwarnung an alle Trainer der
// betroffenen Teams. Stichtag: Saisonende == date('now','-83 days') ⇒ die
// 90-Tage-Löschung steht in genau 7 Tagen an.
func (s *Scheduler) sendVideoRetentionWarnings() {
	rows, err := s.db.Query(`
		SELECT v.id, v.team_id, v.season_id, v.title, se.end_date
		FROM videos v
		JOIN seasons se ON se.id = v.season_id
		WHERE se.end_date IS NOT NULL
		  AND date(se.end_date) = date('now','-83 days')`)
	if err != nil {
		logIfBusy(err, "sendVideoRetentionWarnings.query")
		slog.Error("scheduler video retention warning query failed", "error", err)
		return
	}
	type warnRow struct {
		id       int
		teamID   int
		seasonID int
		title    string
		endDate  string
	}
	var warns []warnRow
	for rows.Next() {
		var w warnRow
		if err := rows.Scan(&w.id, &w.teamID, &w.seasonID, &w.title, &w.endDate); err != nil {
			continue
		}
		warns = append(warns, w)
	}
	rows.Close()

	// Löschdatum = Saisonende + 90 Tage (= heute + 7 Tage). Für die Anzeige
	// reicht heute+7, da der Stichtag oben exakt T-7 erzwingt.
	deleteOn := time.Now().AddDate(0, 0, 7).Format("02.01.")

	sent := 0
	for _, w := range warns {
		trainers := s.teamTrainerUsers(w.teamID, w.seasonID)
		title := "Video wird gelöscht"
		body := fmt.Sprintf("Video „%s\" wird am %s gelöscht.", w.title, deleteOn)
		for _, uid := range trainers {
			// Idempotenz: Log-Zeile VOR dem Senden schreiben; nur bei
			// neu eingefügter Zeile (RowsAffected==1) wird gepusht. Mirror
			// des claimUnsent-/duty-reminder-Patterns.
			res, err := s.db.Exec(
				`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
				uid, "video_retention_warning", w.id)
			if err != nil {
				logIfBusy(err, "sendVideoRetentionWarnings.claim")
				continue
			}
			if n, _ := res.RowsAffected(); n == 1 {
				go push.SendToUsers(s.db, s.cfg, []int{uid}, title, body, fmt.Sprintf("/videos/%d", w.id))
				sent++
			}
		}
	}
	if sent > 0 {
		slog.Info("scheduler video retention warnings sent", "count", sent)
	}
}

// deleteRetainedVideos löscht Videos, deren Saison vor mehr als 90 Tagen endete,
// samt zugehöriger Dateien. season_end_date IS NULL ⇒ nie löschen.
func (s *Scheduler) deleteRetainedVideos() {
	rows, err := s.db.Query(`
		SELECT v.id
		FROM videos v
		JOIN seasons se ON se.id = v.season_id
		WHERE se.end_date IS NOT NULL
		  AND date(se.end_date) < date('now','-90 days')`)
	if err != nil {
		logIfBusy(err, "deleteRetainedVideos.query")
		slog.Error("scheduler video retention query failed", "error", err)
		return
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	rows.Close()

	deleted := 0
	for _, id := range ids {
		if _, err := s.db.Exec(`DELETE FROM videos WHERE id = ?`, id); err != nil {
			logIfBusy(err, "deleteRetainedVideos.delete")
			slog.Error("scheduler video retention delete failed", "video_id", id, "error", err)
			continue
		}
		// Dateien aufräumen (raw/{id}.mp4 + processed/{id}/); not-exist ignorieren.
		raw := filepath.Join(s.cfg.VideoStorageDir, "raw", fmt.Sprintf("%d.mp4", id))
		if err := os.Remove(raw); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("video retention: remove raw failed", "video_id", id, "error", err)
		}
		procDir := filepath.Join(s.cfg.VideoStorageDir, "processed", strconv.Itoa(id))
		if err := os.RemoveAll(procDir); err != nil {
			slog.Warn("video retention: remove processed failed", "video_id", id, "error", err)
		}
		deleted++
	}
	if deleted > 0 {
		slog.Info("scheduler video retention: videos deleted", "count", deleted)
	}
}

// teamTrainerUsers liefert die User-IDs aller Trainer eines Teams in der
// gegebenen Saison (kader → kader_trainers → members → users). Inline-SQL, da
// der Scheduler das Domain-Package nicht importieren darf.
func (s *Scheduler) teamTrainerUsers(teamID, seasonID int) []int {
	rows, err := s.db.Query(`
		SELECT DISTINCT u.id
		FROM kader k
		JOIN kader_trainers kt ON kt.kader_id = k.id
		JOIN members m ON m.id = kt.member_id
		JOIN users u ON u.id = m.user_id
		WHERE k.team_id = ? AND k.season_id = ?`, teamID, seasonID)
	if err != nil {
		logIfBusy(err, "teamTrainerUsers")
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
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
		logIfBusy(err, "recordHeartbeat")
		slog.Error("scheduler heartbeat failed", "error", err)
	}
}

func (s *Scheduler) cleanExpiredTokens() {
	res, err := s.db.Exec(
		`DELETE FROM invitation_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM password_reset_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM refresh_tokens WHERE expires_at < CURRENT_TIMESTAMP;`)
	if err != nil {
		logIfBusy(err, "cleanExpiredTokens")
		slog.Error("scheduler cleanup failed", "error", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		slog.Info("expired tokens cleaned", "count", n)
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
		logIfBusy(err, "sendDutyReminders.query")
		slog.Error("scheduler duty reminders query slots failed", "error", err)
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
			slog.Error("scheduler duty reminders eligible users failed", "slot", sl.id, "error", err)
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
				body := buildReminderMail(u.name, targetDate, uSlots, s.cfg.BaseURL)
				subject := fmt.Sprintf("Offene Dienste am %s", formatDate(targetDate))
				if err := s.mailer.Send(u.email, subject, body); err != nil {
					slog.Error("scheduler duty reminders send mail failed", "email", u.email, "error", err)
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
		slog.Info("scheduler duty reminders emails sent", "count", emailSent, "date", targetDate)
	}
	if pushSent > 0 {
		slog.Info("scheduler duty reminders push sent", "count", pushSent, "date", targetDate)
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
	// Two reminder slots per game: 24h before (planning) and 3h before (event
	// day). The exact firing moment is the game's Berlin wall-clock instant, so
	// reminders are correct regardless of the server timezone (the VPS runs UTC).
	berlin := timez.Berlin()
	now := time.Now().In(berlin)
	// Candidate date range: any game whose start is within the next 24h has a
	// date of today or (just past midnight) tomorrow — widen to +25h to be safe.
	from := now.Format("2006-01-02")
	to := now.Add(25 * time.Hour).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT g.id, g.opponent, g.date, g.time, gt.team_id, t.name, g.event_type
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		JOIN teams t ON t.id = gt.team_id
		WHERE g.date BETWEEN ? AND ?`, from, to)
	if err != nil {
		logIfBusy(err, "sendGameReminders.query")
		slog.Error("scheduler game reminders failed", "error", err)
		return
	}
	defer rows.Close()

	type gameRow struct {
		id        int
		opponent  string
		date      string
		time      string
		teamID    int
		teamName  string
		eventType string
	}
	var games []gameRow
	for rows.Next() {
		var g gameRow
		rows.Scan(&g.id, &g.opponent, &g.date, &g.time, &g.teamID, &g.teamName, &g.eventType)
		games = append(games, g)
	}

	sent := 0
	for _, g := range games {
		eventAt := timez.ParseDT(g.date, g.time, berlin)
		until := eventAt.Sub(now)
		if until < 0 {
			continue // already started/past
		}
		title := "Spielerinnerung"
		if g.eventType == "generisch" {
			title = "Terminerinnerung"
		}
		recipients := s.teamMembersAndParents(g.teamID)
		url := fmt.Sprintf("/termine?focus=game-%d", g.id)

		if until <= 24*time.Hour {
			if uids := s.claimUnsent(recipients, "game_reminder_24h", g.id); len(uids) > 0 {
				body := g.teamName + ": " + g.opponent + " — morgen um " + g.time + " Uhr"
				notify.Send(s.db, s.cfg, uids, "games", title, body, url)
				sent += len(uids)
			}
		}
		if until <= 3*time.Hour {
			if uids := s.claimUnsent(recipients, "game_reminder_3h", g.id); len(uids) > 0 {
				body := g.teamName + ": " + g.opponent + " — heute um " + g.time + " Uhr"
				notify.Send(s.db, s.cfg, uids, "games", title, body, url)
				sent += len(uids)
			}
		}
	}
	if sent > 0 {
		slog.Info("scheduler game reminders sent", "count", sent)
	}
}

// claimUnsent atomically reserves a reminder slot per user and returns only the
// user IDs for which this call newly claimed it (RowsAffected == 1). The
// notification_log row is written BEFORE the push is sent, so a concurrent or
// repeated scheduler run cannot double-send: the second INSERT OR IGNORE yields
// RowsAffected == 0 and that user is dropped. Each (refType, refID) pair is one
// idempotent slot — e.g. "game_reminder_24h" and "game_reminder_3h" for the same
// game are independent slots that each fire exactly once.
func (s *Scheduler) claimUnsent(uids []int, refType string, refID int) []int {
	var claimed []int
	for _, uid := range uids {
		res, err := s.db.Exec(`INSERT OR IGNORE INTO notification_log (user_id, ref_type, ref_id) VALUES (?,?,?)`,
			uid, refType, refID)
		if err != nil {
			logIfBusy(err, "claimUnsent")
			continue
		}
		if n, _ := res.RowsAffected(); n == 1 {
			claimed = append(claimed, uid)
		}
	}
	return claimed
}

func (s *Scheduler) sendTrainingReminders() {
	// Same two-slot model as games (24h + 3h), Berlin wall-clock based.
	berlin := timez.Berlin()
	now := time.Now().In(berlin)
	from := now.Format("2006-01-02")
	to := now.Add(25 * time.Hour).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT ts.id, ts.team_id, COALESCE(NULLIF(ts.title,''),'Training'), ts.date, ts.start_time, t.name
		FROM training_sessions ts
		JOIN teams t ON t.id = ts.team_id
		WHERE ts.date BETWEEN ? AND ?
		  AND ts.status = 'active'`, from, to)
	if err != nil {
		logIfBusy(err, "sendTrainingReminders.query")
		slog.Error("scheduler training reminders failed", "error", err)
		return
	}
	defer rows.Close()

	type sessionRow struct {
		id        int
		teamID    int
		title     string
		date      string
		startTime string
		teamName  string
	}
	var sessions []sessionRow
	for rows.Next() {
		var s sessionRow
		rows.Scan(&s.id, &s.teamID, &s.title, &s.date, &s.startTime, &s.teamName)
		sessions = append(sessions, s)
	}

	sent := 0
	for _, sess := range sessions {
		eventAt := timez.ParseDT(sess.date, sess.startTime, berlin)
		until := eventAt.Sub(now)
		if until < 0 {
			continue
		}
		recipients := s.teamMembersAndParents(sess.teamID)
		url := fmt.Sprintf("/termine?focus=training-%d", sess.id)

		if until <= 24*time.Hour {
			if uids := s.claimUnsent(recipients, "training_reminder_24h", sess.id); len(uids) > 0 {
				body := sess.teamName + ": " + sess.title + " — morgen um " + sess.startTime + " Uhr"
				notify.Send(s.db, s.cfg, uids, "trainings", "Trainingserinnerung", body, url)
				sent += len(uids)
			}
		}
		if until <= 3*time.Hour {
			if uids := s.claimUnsent(recipients, "training_reminder_3h", sess.id); len(uids) > 0 {
				body := sess.teamName + ": " + sess.title + " — heute um " + sess.startTime + " Uhr"
				notify.Send(s.db, s.cfg, uids, "trainings", "Trainingserinnerung", body, url)
				sent += len(uids)
			}
		}
	}
	if sent > 0 {
		slog.Info("scheduler training reminders sent", "count", sent)
	}
}

func (s *Scheduler) sendCarpoolingReminders() {
	// Single slot: exactly 3h before departure (the game's Berlin wall-clock
	// start), for confirmed pairings only.
	berlin := timez.Berlin()
	now := time.Now().In(berlin)
	from := now.Format("2006-01-02")
	to := now.Add(4 * time.Hour).Format("2006-01-02")

	rows, err := s.db.Query(`
		SELECT DISTINCT mg.user_id, g.id, g.opponent, g.date, g.time
		FROM mitfahrgelegenheiten mg
		JOIN games g ON g.id = mg.game_id
		JOIN mitfahrt_paarungen p ON (p.biete_id = mg.id OR p.suche_id = mg.id) AND p.status = 'confirmed'
		WHERE g.date BETWEEN ? AND ?`, from, to)
	if err != nil {
		logIfBusy(err, "sendCarpoolingReminders.query")
		slog.Error("scheduler carpooling reminders failed", "error", err)
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
		until := timez.ParseDT(c.date, c.time, berlin).Sub(now)
		if until < 0 || until > 3*time.Hour {
			continue
		}
		uids := s.claimUnsent([]int{c.userID}, "carpooling_reminder", c.gameID)
		if len(uids) == 0 {
			continue
		}
		notify.Send(s.db, s.cfg, uids, "carpooling",
			"Fahrgemeinschaft heute", c.opponent+" — Abfahrt um "+c.time+" Uhr", "/mitfahrgelegenheiten")
		sent++
	}
	if sent > 0 {
		slog.Info("scheduler carpooling reminders sent", "count", sent)
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

func buildReminderMail(name, date string, slots []openSlot, baseURL string) string {
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

	fmt.Fprintf(&sb, "Jetzt eintragen: %s/duty-board\n\n", baseURL)
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
