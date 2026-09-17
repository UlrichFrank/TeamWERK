package scheduler

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// targetDateForOffset spiegelt die Datumsberechnung aus sendDutyRemindersForOffset
// (time.Now() + offset), damit Test-Fixtures exakt auf den Tag treffen, den der
// Scheduler beim Lauf tatsächlich abfragt.
func targetDateForOffset(offset int) string {
	return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
}

// recordingMailer zeichnet jeden Send-Aufruf auf, statt tatsächlich zu versenden —
// für Idempotenz-Assertions auf den Mail-Kanal, analog zu capturePush für Push.
type recordingMailer struct {
	mu    sync.Mutex
	calls []string // "to|subject"
}

func (m *recordingMailer) Send(to, subject, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, to+"|"+subject)
	return nil
}

func (m *recordingMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func countRefTypeLike(t *testing.T, db *sql.DB, userID int, pattern string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT ref_type FROM notification_log WHERE user_id = ? AND ref_type LIKE ?`, userID, pattern)
	if err != nil {
		t.Fatalf("query notification_log: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var rt string
		if err := rows.Scan(&rt); err != nil {
			t.Fatal(err)
		}
		out = append(out, rt)
	}
	return out
}

func assertHasBoardOverview(t *testing.T, db *sql.DB, userID int, want bool) {
	t.Helper()
	got := len(countRefTypeLike(t, db, userID, "duty_board_vorstand_%")) > 0
	if got != want {
		t.Errorf("user %d: Vorstands-Übersicht erhalten=%v, erwartet=%v", userID, got, want)
	}
}

// --- Mitglieder-Reminder: vier Offsets ---------------------------------------

// TestDutyReminders_FireOnlyAtDefinedOffsets deckt tasks.md 4.1: offene Slots an
// 7/3/2/1 Tagen lösen je einen Reminder aus, ein Slot an einem undefinierten
// Zwischentag (hier 5) keinen.
func TestDutyReminders_FireOnlyAtDefinedOffsets(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	playerID := addPlayerToTeam(t, db, seasonID, teamID)
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")

	for _, offset := range []int{7, 5, 3, 2, 1} {
		testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(offset))
	}

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	got := countRefTypeLike(t, db, playerID, "duty_reminder_%")
	want := map[string]bool{"duty_reminder_7d": true, "duty_reminder_3d": true, "duty_reminder_2d": true, "duty_reminder_1d": true}
	if len(got) != len(want) {
		t.Fatalf("erwartet %d Reminder-Zeilen (7/3/2/1), got %d: %v", len(want), len(got), got)
	}
	for _, rt := range got {
		if !want[rt] {
			t.Errorf("unerwarteter ref_type %q — Offset 5 ist kein definierter Reminder-Zeitpunkt", rt)
		}
	}
}

// TestDutyReminders_FullyBookedSlot_NoReminder: bestehende Garantie bleibt über
// alle vier Offsets erhalten (kein Reminder bei slots_filled = slots_total).
func TestDutyReminders_FullyBookedSlot_NoReminder(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	playerID := addPlayerToTeam(t, db, seasonID, teamID)
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	slotID := testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(7))
	if _, err := db.Exec(`UPDATE duty_slots SET slots_filled = slots_total WHERE id = ?`, slotID); err != nil {
		t.Fatalf("slot voll setzen: %v", err)
	}

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification_log WHERE user_id = ?`, playerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("voll belegter Slot darf keinen Reminder auslösen, got %d notification_log-Zeilen", count)
	}
}

// TestDutyReminders_Idempotent_SecondRunNoDuplicate deckt tasks.md 4.3: ein
// zweiter Scheduler-Lauf für denselben (user, event_date, offset) legt nichts nach.
func TestDutyReminders_Idempotent_SecondRunNoDuplicate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	playerID := addPlayerToTeam(t, db, seasonID, teamID)
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(7))

	s := New(db, testutil.TestConfig(), nil)
	s.sendDutyReminders()
	s.sendDutyReminders()

	got := countRefTypeLike(t, db, playerID, "duty_reminder_7d")
	if len(got) != 1 {
		t.Errorf("erwartet genau 1 Zeile nach zwei Scheduler-Läufen (Idempotenz), got %d", len(got))
	}
}

// TestDutyReminders_EmailOptIn_SentOnceAcrossReruns: Mail-Kanal ist genauso
// idempotent wie der Push-Kanal.
func TestDutyReminders_EmailOptIn_SentOnceAcrossReruns(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	playerID := addPlayerToTeam(t, db, seasonID, teamID)
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(1))
	testutil.CreateNotificationPreference(t, db, playerID, "duty_reminders", true, true)

	mailer := &recordingMailer{}
	s := New(db, testutil.TestConfig(), mailer)
	s.sendDutyReminders()
	s.sendDutyReminders()

	if mailer.count() != 1 {
		t.Errorf("erwartet genau 1 Mail nach zwei Läufen, got %d", mailer.count())
	}
}

// TestDutyReminders_DifferentOffsetSameEventDate_NotBlocked ist der
// Regressionsschutz aus tasks.md 4.4: ein bereits verschickter Reminder für
// einen anderen Offset darf denselben event_date nicht blockieren. Der alte,
// reine (user_id, event_date)-Schlüssel hätte das getan; hier wird der
// Vorzustand direkt gesetzt, weil ein einzelner Testlauf "heute" nicht über
// mehrere Kalendertage hinweg verschieben kann.
func TestDutyReminders_DifferentOffsetSameEventDate_NotBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	playerID := addPlayerToTeam(t, db, seasonID, teamID)
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	targetDate := targetDateForOffset(3)
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDate)
	testutil.CreateNotificationPreference(t, db, playerID, "duty_reminders", true, true)

	// Simuliert: derselbe User hat für dasselbe event_date bereits den
	// 7-Tage-Reminder bekommen (Mail- UND Push-Idempotenz-Zeile).
	if _, err := db.Exec(`INSERT INTO duty_reminder_log (user_id, event_date, days_before) VALUES (?, ?, 7)`, playerID, targetDate); err != nil {
		t.Fatalf("seed 7-Tage-Mail-Log: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notification_log (user_id, ref_type, ref_id) VALUES (?, 'duty_reminder_7d', ?)`, playerID, hashDate(targetDate)); err != nil {
		t.Fatalf("seed 7-Tage-Push-Log: %v", err)
	}

	mailer := &recordingMailer{}
	New(db, testutil.TestConfig(), mailer).sendDutyRemindersForOffset(3)

	if mailer.count() != 1 {
		t.Errorf("3-Tage-Mail wurde trotz vorhandenem 7-Tage-Log-Eintrag fälschlich unterdrückt (count=%d)", mailer.count())
	}
	if got := countRefTypeLike(t, db, playerID, "duty_reminder_3d"); len(got) != 1 {
		t.Errorf("3-Tage-Push wurde trotz vorhandenem 7-Tage-Log-Eintrag fälschlich unterdrückt (count=%d)", len(got))
	}
}

// --- Vorstands-Übersicht ------------------------------------------------------

// TestBoardOverview_FiresOnlyAt3And1 deckt tasks.md 4.5.
func TestBoardOverview_FiresOnlyAt3And1(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	boardUserID := testutil.CreateVorstandUser(t, db)

	for _, offset := range []int{7, 3, 2, 1} {
		testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(offset))
	}

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	got := countRefTypeLike(t, db, boardUserID, "duty_board_vorstand_%")
	want := map[string]bool{"duty_board_vorstand_3d": true, "duty_board_vorstand_1d": true}
	if len(got) != len(want) {
		t.Fatalf("erwartet genau 2 Übersichts-Zeilen (3/1 Tage), got %d: %v", len(got), got)
	}
	for _, rt := range got {
		if !want[rt] {
			t.Errorf("unerwarteter ref_type %q — Übersicht darf nur bei 3/1 Tagen feuern", rt)
		}
	}
}

// TestBoardOverview_OnlyVorstandFunctionReceives deckt tasks.md 4.6 (Empfängerkreis).
func TestBoardOverview_OnlyVorstandFunctionReceives(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(3))

	boardUserID := testutil.CreateVorstandUser(t, db)

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddClubFunction(t, db, trainerMemberID, "trainer")

	beisitzerUserID := testutil.CreateUser(t, db, "standard")
	beisitzerMemberID := testutil.CreateMember(t, db, beisitzerUserID)
	testutil.AddClubFunction(t, db, beisitzerMemberID, "vorstand_beisitzer")

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	assertHasBoardOverview(t, db, boardUserID, true)
	assertHasBoardOverview(t, db, trainerUserID, false)
	assertHasBoardOverview(t, db, beisitzerUserID, false)
}

// TestBoardOverview_VereinsweitUnabhaengigVonEigenemTeam deckt tasks.md 4.6
// (vereinsweite Aggregation): der Vorstand bekommt die Übersicht auch für ein
// Team, dem er selbst nicht angehört.
func TestBoardOverview_VereinsweitUnabhaengigVonEigenemTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamA, 0, targetDateForOffset(1))

	boardUserID := testutil.CreateVorstandUser(t, db) // kein Mitglied von Team A

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	assertHasBoardOverview(t, db, boardUserID, true)
}

// TestBoardOverview_And_PersonalReminder_AreIndependent deckt tasks.md 4.8: ein
// Vorstandsmitglied, das gleichzeitig eligible User eines offenen Slots ist,
// bekommt an einem Tag sowohl die persönliche Erinnerung als auch die Übersicht.
func TestBoardOverview_And_PersonalReminder_AreIndependent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(3))

	userID := dualRoleSpielerVorstand(t, db, seasonID, teamID)

	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	if got := countRefTypeLike(t, db, userID, "duty_reminder_3d"); len(got) != 1 {
		t.Errorf("persönliche Erinnerung fehlt (count=%d)", len(got))
	}
	assertHasBoardOverview(t, db, userID, true)
}

// TestDutyReminders_CategoryOptOut_SuppressesBothPersonalAndBoardPush deckt
// tasks.md 4.7: ein Vorstandsmitglied mit deaktivierter Kategorie duty_reminders
// bekommt weder die persönliche Erinnerung noch die Übersicht tatsächlich
// zugestellt (die interne Idempotenz-Zeile wird trotzdem geschrieben — das ist
// gewollt, siehe notify.Send-Kommentar; geprüft wird hier der reale Push-Versand).
func TestDutyReminders_CategoryOptOut_SuppressesBothPersonalAndBoardPush(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyTypeWithTarget(t, db, "Kasse", "spieler")
	testutil.CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, targetDateForOffset(3))

	userID := dualRoleSpielerVorstand(t, db, seasonID, teamID)
	testutil.CreateNotificationPreference(t, db, userID, "duty_reminders", false, false)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendDutyReminders()

	assertNoPush(t, pushes, userID, 300*time.Millisecond)
}

// dualRoleSpielerVorstand legt einen User an, der sowohl 'spieler' im Kader von
// teamID (aktive Saison) als auch 'vorstand' ist.
func dualRoleSpielerVorstand(t *testing.T, db *sql.DB, seasonID, teamID int) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	testutil.AddClubFunction(t, db, memberID, "spieler")
	testutil.AddClubFunction(t, db, memberID, "vorstand")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}
	return userID
}

// --- Mail-Framing --------------------------------------------------------------

func TestBuildBoardOverviewMail_ContainsEingreifenFramingAndOffsetLabel(t *testing.T) {
	slots := []openSlot{{eventName: "Heimspiel", dutyType: "Kasse", slotsOpen: 1}}
	body := buildBoardOverviewMail("Vorstand", "2026-07-10", slots, 1, "https://example.test")

	if !strings.Contains(body, "eingreifen") {
		t.Errorf("board overview mail sollte Eingreifen-Framing enthalten, got:\n%s", body)
	}
	if !strings.Contains(body, "https://example.test/duty-board") {
		t.Errorf("board overview mail missing base-URL deep link, got:\n%s", body)
	}
	if !strings.Contains(body, "morgen") {
		t.Errorf("offset=1 sollte 'morgen' im Text erzeugen, got:\n%s", body)
	}
}
