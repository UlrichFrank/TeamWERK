package calendar_test

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// dutyFeedFixture legt einen Nutzer im Kader von "Team A" an, ein Heimspiel am
// 2026-09-19 in der "Sporthalle Vaihingen" und liefert DB und Server, damit
// die Tests Dienst-Slots mit genau den Eigenschaften anlegen, um die es geht.
type dutyFeedFixture struct {
	db        *sql.DB
	srv       *httptest.Server
	userID    int
	userToken string
	seasonID  int
	teamID    int
	gameID    int
	typeID    int
}

func newDutyFeedFixture(t *testing.T) dutyFeedFixture {
	t.Helper()
	db := testutil.NewDB(t)
	f := dutyFeedFixture{db: db}
	f.seasonID = testutil.CreateSeason(t, db, "2026/27")
	f.teamID = testutil.CreateTeam(t, db, "Team A")
	f.userID = testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, f.userID)
	kaderID := testutil.CreateKader(t, db, f.teamID, f.seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	var venueID int64
	if err := db.QueryRow(`INSERT INTO venues (name, street, postal_code, city, is_home_venue) VALUES (?, ?, ?, ?, 1) RETURNING id`,
		"Sporthalle Vaihingen", "Rosenstraße 5", "70563", "Stuttgart").Scan(&venueID); err != nil {
		t.Fatalf("insert venue: %v", err)
	}
	f.gameID = testutil.CreateGame(t, db, f.seasonID, f.teamID, "2026-09-19")
	if _, err := db.Exec(`UPDATE games SET venue_id = ? WHERE id = ?`, venueID, f.gameID); err != nil {
		t.Fatalf("set venue: %v", err)
	}
	f.typeID = testutil.CreateDutyType(t, db, "Bewirtung", 1.0)
	f.userToken = testutil.Token(t, f.userID, "standard", nil)
	f.srv = prodserver.New(t, db)
	return f
}

// slot legt einen dem Nutzer zugewiesenen Dienst an. gameID 0 = ohne
// Spielbezug (dann mit team_id, wie in den echten Schreibpfaden), eventTime ""
// = ohne Uhrzeit.
func (f dutyFeedFixture) slot(t *testing.T, gameID int, eventTime string, hours float64, roleDesc string) int {
	t.Helper()
	var gameArg, teamArg, timeArg, roleArg any
	if gameID > 0 {
		gameArg = gameID
	} else {
		teamArg = f.teamID
	}
	if eventTime != "" {
		timeArg = eventTime
	}
	if roleDesc != "" {
		roleArg = roleDesc
	}
	res, err := f.db.Exec(`INSERT INTO duty_slots
		(event_name, event_date, event_time, duty_type_id, role_desc, slots_total, team_id, season_id, game_id, hours_value)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
		"Heimspieltag", "2026-09-19", timeArg, f.typeID, roleArg, teamArg, f.seasonID, gameArg, hours)
	if err != nil {
		t.Fatalf("insert duty slot: %v", err)
	}
	id, _ := res.LastInsertId()
	if _, err := f.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, 'assigned')`,
		id, f.userID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	return int(id)
}

func (f dutyFeedFixture) feed(t *testing.T) string {
	t.Helper()
	return unfoldICS(feedBody(t, f.srv, f.userToken, allTogglesOn()))
}

// vevent liefert den VEVENT-Block mit der gegebenen UID (entfaltet).
func vevent(t *testing.T, body, uid string) string {
	t.Helper()
	for _, block := range strings.Split(body, "BEGIN:VEVENT\r\n")[1:] {
		block = block[:strings.Index(block, "END:VEVENT")]
		if strings.Contains(block, "UID:"+uid+"\r\n") {
			return block
		}
	}
	t.Fatalf("kein VEVENT mit UID %s im Feed:\n%s", uid, body)
	return ""
}

func propLine(block, name string) (string, bool) {
	for _, l := range strings.Split(block, "\r\n") {
		if strings.HasPrefix(l, name+":") || strings.HasPrefix(l, name+";") {
			return l, true
		}
	}
	return "", false
}

// ── Dienst-Events ────────────────────────────────────────────────────────────

func TestFeed_DienstAmSpielErbtSpielort(t *testing.T) {
	f := newDutyFeedFixture(t)
	dutyID := f.slot(t, f.gameID, "13:30", 3.0, "")
	body := f.feed(t)

	gameLoc, ok := propLine(vevent(t, body, fmt.Sprintf("game-%d@teamwerk", f.gameID)), "LOCATION")
	if !ok {
		t.Fatalf("Spiel ohne LOCATION:\n%s", body)
	}
	dutyLoc, ok := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", dutyID)), "LOCATION")
	if !ok {
		t.Fatalf("Dienst am Spiel ohne LOCATION:\n%s", body)
	}
	if dutyLoc != gameLoc {
		t.Errorf("Dienst-LOCATION %q ≠ Spiel-LOCATION %q", dutyLoc, gameLoc)
	}
	if !strings.Contains(dutyLoc, `Sporthalle Vaihingen\, Rosenstraße 5\, 70563 Stuttgart`) {
		t.Errorf("unerwartete LOCATION: %q", dutyLoc)
	}
}

func TestFeed_DienstOhneSpielbezugOhneOrt(t *testing.T) {
	f := newDutyFeedFixture(t)
	dutyID := f.slot(t, 0, "13:30", 1.0, "")
	body := f.feed(t)
	if l, ok := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", dutyID)), "LOCATION"); ok {
		t.Errorf("Dienst ohne Spielbezug trägt LOCATION: %q", l)
	}
}

func TestFeed_DienstAmSpielOhneVenueOhneOrt(t *testing.T) {
	f := newDutyFeedFixture(t)
	if _, err := f.db.Exec(`UPDATE games SET venue_id = NULL WHERE id = ?`, f.gameID); err != nil {
		t.Fatal(err)
	}
	dutyID := f.slot(t, f.gameID, "13:30", 1.0, "")
	body := f.feed(t)
	if l, ok := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", dutyID)), "LOCATION"); ok {
		t.Errorf("Dienst am Spiel ohne Venue trägt LOCATION: %q", l)
	}
}

func TestFeed_DienstdauerFolgtHoursValue(t *testing.T) {
	cases := []struct {
		hours   float64
		wantEnd string
	}{
		{3.0, "20260919T163000"},
		{1.5, "20260919T150000"},
		{0, "20260919T143000"},  // nicht-positiv → eine Stunde
		{-2, "20260919T143000"}, // kaputte Bestandszeile
	}
	for _, c := range cases {
		t.Run(fmt.Sprint(c.hours), func(t *testing.T) {
			f := newDutyFeedFixture(t)
			dutyID := f.slot(t, f.gameID, "13:30", c.hours, "")
			block := vevent(t, f.feed(t), fmt.Sprintf("duty-%d@teamwerk", dutyID))
			start, _ := propLine(block, "DTSTART")
			end, _ := propLine(block, "DTEND")
			if start != "DTSTART;TZID=Europe/Berlin:20260919T133000" {
				t.Errorf("DTSTART = %q", start)
			}
			if end != "DTEND;TZID=Europe/Berlin:"+c.wantEnd {
				t.Errorf("hours_value %v: DTEND = %q, erwartet %s", c.hours, end, c.wantEnd)
			}
			if end[strings.Index(end, ":")+1:] <= start[strings.Index(start, ":")+1:] {
				t.Errorf("DTEND <= DTSTART: %q / %q", start, end)
			}
		})
	}
}

func TestFeed_DienstRolleAlsDescription(t *testing.T) {
	f := newDutyFeedFixture(t)
	withRole := f.slot(t, f.gameID, "13:30", 1.0, "Kasse und Kuchenverkauf")
	withoutRole := f.slot(t, f.gameID, "15:00", 1.0, "")
	body := f.feed(t)

	if l, _ := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", withRole)), "DESCRIPTION"); l != "DESCRIPTION:Kasse und Kuchenverkauf" {
		t.Errorf("gepflegte Rolle fehlt, DESCRIPTION = %q", l)
	}
	if l, ok := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", withoutRole)), "DESCRIPTION"); ok {
		t.Errorf("Dienst ohne Rolle trägt DESCRIPTION: %q", l)
	}
}

func TestFeed_DienstOhneUhrzeitIstGanztags(t *testing.T) {
	f := newDutyFeedFixture(t)
	allDay := f.slot(t, f.gameID, "", 3.0, "")
	timed := f.slot(t, f.gameID, "13:30", 3.0, "")
	body := f.feed(t)

	block := vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", allDay))
	if l, _ := propLine(block, "DTSTART"); l != "DTSTART;VALUE=DATE:20260919" {
		t.Errorf("Ganztags-DTSTART = %q", l)
	}
	if l, _ := propLine(block, "DTEND"); l != "DTEND;VALUE=DATE:20260920" {
		t.Errorf("Ganztags-DTEND = %q", l)
	}
	if strings.Contains(block, "T000000") {
		t.Errorf("Dienst ohne Uhrzeit erzeugt Mitternachtstermin:\n%s", block)
	}

	if l, _ := propLine(vevent(t, body, fmt.Sprintf("duty-%d@teamwerk", timed)), "DTSTART"); !strings.HasPrefix(l, "DTSTART;TZID=Europe/Berlin:") {
		t.Errorf("Dienst mit Uhrzeit muss bei TZID bleiben, DTSTART = %q", l)
	}
}

// ── Kalender-Rahmen (RFC 5545) ───────────────────────────────────────────────

func TestFeed_VTimezoneVorErstemEvent(t *testing.T) {
	f := newDutyFeedFixture(t)
	f.slot(t, f.gameID, "13:30", 1.0, "")
	body := f.feed(t)

	if n := strings.Count(body, "BEGIN:VTIMEZONE\r\n"); n != 1 {
		t.Fatalf("erwartet genau ein VTIMEZONE, gefunden %d", n)
	}
	tz := body[strings.Index(body, "BEGIN:VTIMEZONE"):strings.Index(body, "END:VTIMEZONE")]
	if !strings.Contains(tz, "TZID:Europe/Berlin\r\n") {
		t.Errorf("VTIMEZONE ohne TZID:Europe/Berlin:\n%s", tz)
	}
	if strings.Index(body, "BEGIN:VTIMEZONE") > strings.Index(body, "BEGIN:VEVENT") {
		t.Error("VTIMEZONE steht nach dem ersten VEVENT")
	}
}

var dtstampRe = regexp.MustCompile(`^DTSTAMP:\d{8}T\d{6}Z$`)

func TestFeed_JedesEventTraegtStabilenDtstamp(t *testing.T) {
	f := newDutyFeedFixture(t)
	dutyID := f.slot(t, f.gameID, "13:30", 1.0, "")
	testutil.CreateTrainingSessionForKader(t, f.db, kaderOf(t, f), f.seasonID, "2026-09-17", "Training")

	// created_at in der zonenlosen SQLite-Form ist UTC: der Wert muss 1:1 als
	// …Z erscheinen, nicht um den Berliner Offset verschoben. Zugleich belegt
	// er, dass DTSTAMP aus created_at kommt und nicht aus time.Now().
	if _, err := f.db.Exec(`UPDATE duty_slots SET created_at = '2026-01-02 03:04:05' WHERE id = ?`, dutyID); err != nil {
		t.Fatal(err)
	}
	if l, _ := propLine(vevent(t, f.feed(t), fmt.Sprintf("duty-%d@teamwerk", dutyID)), "DTSTAMP"); l != "DTSTAMP:20260102T030405Z" {
		t.Errorf("DTSTAMP = %q, erwartet created_at als UTC", l)
	}

	stamps := func() []string {
		body := f.feed(t)
		var out []string
		blocks := strings.Split(body, "BEGIN:VEVENT\r\n")[1:]
		if len(blocks) < 3 {
			t.Fatalf("erwartet Spiel, Training und Dienst, gefunden %d Events:\n%s", len(blocks), body)
		}
		for _, b := range blocks {
			l, ok := propLine(b, "DTSTAMP")
			if !ok || !dtstampRe.MatchString(l) {
				t.Errorf("VEVENT ohne gültiges DTSTAMP (%q):\n%s", l, b)
			}
			out = append(out, l)
		}
		return out
	}
	first, second := stamps(), stamps()
	if strings.Join(first, "|") != strings.Join(second, "|") {
		t.Errorf("DTSTAMP ändert sich zwischen zwei Abrufen:\n%v\n%v", first, second)
	}
}

func kaderOf(t *testing.T, f dutyFeedFixture) int {
	t.Helper()
	var id int
	if err := f.db.QueryRow(`SELECT id FROM kader WHERE team_id = ? AND season_id = ?`, f.teamID, f.seasonID).Scan(&id); err != nil {
		t.Fatalf("kader: %v", err)
	}
	return id
}

// TestRenderICal_FaltungZerteiltKeineRune rendert Titel mit langen Namen und
// Mehrbyte-Zeichen. Die Gegner-Namen variieren in der Länge, damit die
// 75-Oktett-Grenze mehrfach auf „ö", „·" und „–" fällt — mit festen
// Testdaten hinge es an Zufall, ob der Fall überhaupt eintritt.
func TestRenderICal_FaltungZerteiltKeineRune(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2026/27")
	teamID := testutil.CreateTeam(t, db, "mB1")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	other := testutil.CreateMember(t, db, 0)
	for pad := 0; pad < 8; pad++ {
		gameID := testutil.CreateGame(t, db, seasonID, teamID, fmt.Sprintf("2026-09-%02d", 10+pad))
		opponent := "SG Köngen-Wendlingen-Unterensingen" + strings.Repeat("ä", pad)
		db.Exec(`UPDATE games SET opponent = ?, note = ? WHERE id = ?`, opponent,
			"Treffpunkt Halle · Trikots mitbringen – bitte pünktlich "+strings.Repeat("ü", pad), gameID)
		db.Exec(`INSERT INTO game_lineup (game_id, member_id) VALUES (?, ?)`, gameID, other)
	}
	srv := prodserver.New(t, db)
	raw := feedBody(t, srv, testutil.Token(t, userID, "standard", nil), allTogglesOn())

	if !strings.Contains(unfoldICS(raw), "mB1 · erw. Kader · nicht aufgestellt") {
		t.Fatalf("Fixture trifft den langen Titel nicht:\n%s", raw)
	}
	folded := 0
	for _, line := range strings.Split(strings.TrimSuffix(raw, "\r\n"), "\r\n") {
		if !utf8.ValidString(line) {
			t.Errorf("Zeile ist kein gültiges UTF-8: %q", line)
		}
		if len(line) > 75 {
			t.Errorf("Zeile mit %d Oktetten > 75: %q", len(line), line)
		}
		if strings.HasPrefix(line, " ") {
			folded++
		}
	}
	if folded == 0 {
		t.Fatal("keine Zeile gefaltet — Test prüft nichts")
	}
}
