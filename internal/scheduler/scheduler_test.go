package scheduler

import (
	"bytes"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// TestScheduler_SQLiteBusyEmitsLog verifiziert, dass der Scheduler-Pfad bei
// einem SQLITE_BUSY-Error ein strukturiertes slog.Warn-Record emittiert. Das
// ist die Cross-Prozess-Variante zum HTTP-Counter (siehe design.md).
func TestScheduler_SQLiteBusyEmitsLog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	logIfBusy(errors.New("database is locked"), "test-op")

	out := buf.String()
	for _, want := range []string{
		`"event":"sqlite_busy"`,
		`"source":"scheduler"`,
		`"op":"test-op"`,
		`"level":"WARN"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in log:\n%s", want, out)
		}
	}
}

// Sicherstellen, dass nicht-BUSY-Fehler KEIN Log emittieren.
func TestScheduler_NonBusyError_NoLog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	logIfBusy(errors.New("some other error"), "test-op")

	if strings.Contains(buf.String(), "sqlite_busy") {
		t.Fatalf("non-BUSY error must not emit sqlite_busy log:\n%s", buf.String())
	}
}

// Spieler-Auflösung: User mit Vereinsfunktion 'spieler' im aktiven Kader des Teams
// MUSS in der Empfängerliste auftauchen. Users.role spielt keine Rolle (war historisch
// 'spieler', heute 'standard').
func TestEligibleUsers_SpielerViaClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	// Vereinsfunktion 'spieler'.
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, memberID); err != nil {
		t.Fatalf("insert club function: %v", err)
	}
	// In aktiven Saison-Kader aufnehmen.
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}

	// Slot mit target_role='spieler' für das Team.
	dutyTypeID := createDutyTypeWithTarget(t, db, "Hallendienst", "spieler")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "spieler",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, userID) {
		t.Errorf("expected user %d (with spieler function in active kader) in recipients, got %+v", userID, users)
	}
}

// Eltern-Auflösung: User mit family_link zu einem Member mit Vereinsfunktion 'spieler'
// im aktiven Kader MUSS in der Empfängerliste auftauchen.
func TestEligibleUsers_ElternteilViaFamilyLinks(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)

	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, childMemberID); err != nil {
		t.Fatalf("insert family_link: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, childMemberID); err != nil {
		t.Fatalf("insert club function: %v", err)
	}
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}

	dutyTypeID := createDutyTypeWithTarget(t, db, "Kuchenbacken", "elternteil")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "elternteil",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if !containsUserID(users, parentUserID) {
		t.Errorf("expected parent user %d in recipients, got %+v", parentUserID, users)
	}
}

// Negativfall: User mit role='standard' und ohne member_club_functions wird NICHT
// als Spieler-Empfänger gefunden (alte Fehlbehauptung „role='spieler'" gilt nicht mehr).
func TestEligibleUsers_SpielerSkipsUserWithoutClubFunction(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	// User+Member ohne Vereinsfunktion, aber im Kader.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}

	dutyTypeID := createDutyTypeWithTarget(t, db, "Hallendienst", "spieler")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")

	s := New(db, testutil.TestConfig(), nil)
	users, err := s.eligibleUsers(openSlot{
		id:         slotID,
		targetRole: "spieler",
		teamID:     sql.NullInt64{Int64: int64(teamID), Valid: true},
	})
	if err != nil {
		t.Fatalf("eligibleUsers: %v", err)
	}
	if containsUserID(users, userID) {
		t.Errorf("user without 'spieler' club function should NOT be a recipient, but was: %+v", users)
	}
}

// ── Reminder slot tests (timezone-correct-event-reminders) ───────────────────
//
// The scheduler uses the real time.Now(); tests place events at a wall-clock
// offset from now (in Berlin) so they land in the desired slot window, and then
// assert on notification_log — that row is the idempotency contract and is
// written before the (no-op, no subscriptions) push, so it deterministically
// proves which slot fired.

// setupTeamPlayer creates an active season, a team, and a player user that is in
// that team's kader (so teamMembersAndParents resolves the user). Returns
// teamID, userID.
func setupTeamPlayer(t *testing.T, db *sql.DB) (teamID, userID int) {
	t.Helper()
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	userID = testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}
	return teamID, userID
}

// createGameAt inserts a game (+team link) whose start is `until` from now, in
// Berlin wall-clock, and returns the game ID.
func createGameAt(t *testing.T, db *sql.DB, teamID int, until time.Duration) int {
	t.Helper()
	at := time.Now().In(timez.Berlin()).Add(until)
	var seasonID int
	if err := db.QueryRow(`SELECT id FROM seasons WHERE is_active=1`).Scan(&seasonID); err != nil {
		t.Fatalf("active season: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?, 'Gegner', ?, ?, 'heim', 1)`,
		seasonID, at.Format("2006-01-02"), at.Format("15:04"))
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	id, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, id, teamID); err != nil {
		t.Fatalf("insert game_teams: %v", err)
	}
	return int(id)
}

func logCount(t *testing.T, db *sql.DB, refType string, refID int) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification_log WHERE ref_type=? AND ref_id=?`, refType, refID).Scan(&n); err != nil {
		t.Fatalf("count notification_log: %v", err)
	}
	return n
}

// 24h-Slot feuert für ein Spiel in ~20h, der 3h-Slot NICHT — und ein zweiter Lauf
// erzeugt kein Duplikat (Idempotenz über ref_type game_reminder_24h).
func TestGameReminder_24hSlotFiresOnceNot3h(t *testing.T) {
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	gameID := createGameAt(t, db, teamID, 20*time.Hour)

	s := New(db, testutil.TestConfig(), nil)
	s.sendGameReminders()
	s.sendGameReminders() // second run must not duplicate

	if got := logCount(t, db, "game_reminder_24h", gameID); got != 1 {
		t.Errorf("game_reminder_24h: want exactly 1 log row, got %d", got)
	}
	if got := logCount(t, db, "game_reminder_3h", gameID); got != 0 {
		t.Errorf("game_reminder_3h must NOT fire for a 20h-out game, got %d rows", got)
	}
}

// Spiel in ~2h: BEIDE Slots feuern (3h ist Teilmenge von 24h), je genau einmal.
func TestGameReminder_BothSlotsFireWithin3h(t *testing.T) {
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	gameID := createGameAt(t, db, teamID, 2*time.Hour)

	s := New(db, testutil.TestConfig(), nil)
	s.sendGameReminders()

	if got := logCount(t, db, "game_reminder_24h", gameID); got != 1 {
		t.Errorf("game_reminder_24h: want 1, got %d", got)
	}
	if got := logCount(t, db, "game_reminder_3h", gameID); got != 1 {
		t.Errorf("game_reminder_3h: want 1, got %d", got)
	}
}

// Vergangenes Spiel löst in keinem Slot einen Reminder aus.
func TestGameReminder_PastEventNoReminder(t *testing.T) {
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	gameID := createGameAt(t, db, teamID, -2*time.Hour)

	s := New(db, testutil.TestConfig(), nil)
	s.sendGameReminders()

	if got := logCount(t, db, "game_reminder_24h", gameID) + logCount(t, db, "game_reminder_3h", gameID); got != 0 {
		t.Errorf("past game must not fire any reminder, got %d log rows", got)
	}
}

// Training: 24h- und 3h-Slot feuern je einmal für eine aktive Einheit in ~2h;
// eine abgesagte (cancelled) Einheit löst keinen Reminder aus.
func TestTrainingReminder_SlotsAndCancelled(t *testing.T) {
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	var seasonID int
	db.QueryRow(`SELECT id FROM seasons WHERE is_active=1`).Scan(&seasonID)

	at := time.Now().In(timez.Berlin()).Add(2 * time.Hour)
	insertSession := func(status string) int {
		res, err := db.Exec(
			`INSERT INTO training_sessions (team_id, season_id, date, start_time, end_time, title, status)
			 VALUES (?, ?, ?, ?, '20:00', 'Einheit', ?)`,
			teamID, seasonID, at.Format("2006-01-02"), at.Format("15:04"), status)
		if err != nil {
			t.Fatalf("insert training_session: %v", err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}
	activeID := insertSession("active")
	cancelledID := insertSession("cancelled")

	s := New(db, testutil.TestConfig(), nil)
	s.sendTrainingReminders()

	if got := logCount(t, db, "training_reminder_24h", activeID); got != 1 {
		t.Errorf("active training_reminder_24h: want 1, got %d", got)
	}
	if got := logCount(t, db, "training_reminder_3h", activeID); got != 1 {
		t.Errorf("active training_reminder_3h: want 1, got %d", got)
	}
	if got := logCount(t, db, "training_reminder_24h", cancelledID) + logCount(t, db, "training_reminder_3h", cancelledID); got != 0 {
		t.Errorf("cancelled training must not fire, got %d log rows", got)
	}
}

// Fahrgemeinschaft: feuert genau einmal im ≤3h-Fenster für eine bestätigte
// Paarung; eine Fahrt in ~20h (außerhalb 3h) feuert NICHT.
func TestCarpoolingReminder_Exactly3hWindow(t *testing.T) {
	db := testutil.NewDB(t)
	teamID, riderUserID := setupTeamPlayer(t, db)
	driverUserID := testutil.CreateUser(t, db, "standard")

	// Helper: build a game + confirmed pairing (driver biete, rider suche).
	confirmedPairing := func(gameID int) {
		var bieteID, sucheID int64
		res, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ) VALUES (?, ?, 'biete')`, gameID, driverUserID)
		bieteID, _ = res.LastInsertId()
		res, _ = db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ) VALUES (?, ?, 'suche')`, gameID, riderUserID)
		sucheID, _ = res.LastInsertId()
		if _, err := db.Exec(
			`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`,
			bieteID, sucheID); err != nil {
			t.Fatalf("insert pairing: %v", err)
		}
	}

	soonGame := createGameAt(t, db, teamID, 2*time.Hour)
	confirmedPairing(soonGame)
	farGame := createGameAt(t, db, teamID, 20*time.Hour)
	confirmedPairing(farGame)

	s := New(db, testutil.TestConfig(), nil)
	s.sendCarpoolingReminders()
	s.sendCarpoolingReminders() // idempotent

	// Both participants of the confirmed pairing (driver + rider) are reminded,
	// so the within-3h game yields exactly 2 log rows — and the second run adds
	// none (idempotent per user+ref_type+game).
	if got := logCount(t, db, "carpooling_reminder", soonGame); got != 2 {
		t.Errorf("carpooling within 3h: want 2 log rows (driver+rider), got %d", got)
	}
	if got := logCount(t, db, "carpooling_reminder", farGame); got != 0 {
		t.Errorf("carpooling 20h out must NOT fire, got %d", got)
	}
}

// TestScheduler_DutyReminder_UsesConfigBaseURL verifiziert, dass der
// Duty-Reminder-Mailbody den Direktlink aus cfg.BaseURL baut und nicht mehr die
// früher hartkodierte internal.*-URL enthält.
func TestScheduler_DutyReminder_UsesConfigBaseURL(t *testing.T) {
	slots := []openSlot{{eventName: "Heimspiel", dutyType: "Kasse", slotsOpen: 2}}
	body := buildReminderMail("Alex", "2026-07-10", slots, "https://example.test")

	if !strings.Contains(body, "https://example.test/duty-board") {
		t.Errorf("body missing base-URL deep link, got:\n%s", body)
	}
	if strings.Contains(body, "internal.team-stuttgart.org") {
		t.Errorf("body must not contain the legacy internal.* URL, got:\n%s", body)
	}
}

func createDutyTypeWithTarget(t *testing.T, db *sql.DB, name, target string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO duty_types (name, hours_value, target_role) VALUES (?, 1.0, ?)`, name, target)
	if err != nil {
		t.Fatalf("create duty_type: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func containsUserID(users []reminderUser, id int) bool {
	for _, u := range users {
		if u.id == id {
			return true
		}
	}
	return false
}
