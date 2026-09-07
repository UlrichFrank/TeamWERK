package calendar_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// lineupFill beschreibt, wie die Aufstellung eines Spiels im Fixture gefüllt wird.
type lineupFill int

const (
	lineupEmpty     lineupFill = iota // keine Zeile in game_lineup — Aufstellung nie gepflegt
	lineupWithMe                      // Aufstellung existiert, das eigene Mitglied steht drin
	lineupWithoutMe                   // Aufstellung existiert, das eigene Mitglied fehlt
)

// lineupFeed baut einen Nutzer, der über membershipTable am Kader von "mB1" hängt,
// legt ein Spiel des gegebenen event_type an, füllt game_lineup gemäß fill und
// liefert den entfalteten Feed-Body zurück.
func lineupFeed(t *testing.T, membershipTable, eventType string, fill lineupFill, note string) string {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "mB1")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(
		`INSERT INTO `+membershipTable+` (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert into %s: %v", membershipTable, err)
	}

	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-08-15")
	if _, err := db.Exec(`UPDATE games SET event_type = ?, is_home = ?, note = ? WHERE id = ?`,
		eventType, eventType == "heim", note, gameID); err != nil {
		t.Fatalf("set event_type: %v", err)
	}
	fillLineup(t, db, gameID, memberID, fill)

	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)
	tok := postToken(t, srv, userToken, allTogglesOn())

	res := testutil.Get(t, srv, "/api/calendar/feed/"+tok["token"].(string), "")
	defer res.Body.Close()
	return unfoldICS(readBody(t, res.Body))
}

// fillLineup schreibt die Aufstellung. Für lineupWithoutMe braucht es ein
// FREMDES Mitglied in der Liste — eine leere Liste wäre der dritte Zustand und
// würde die Unterscheidung, um die es hier geht, gerade verfehlen.
func fillLineup(t *testing.T, db *sql.DB, gameID, memberID int, fill lineupFill) {
	t.Helper()
	switch fill {
	case lineupEmpty:
		return
	case lineupWithMe:
		if _, err := db.Exec(
			`INSERT INTO game_lineup (game_id, member_id) VALUES (?, ?)`, gameID, memberID); err != nil {
			t.Fatalf("insert lineup: %v", err)
		}
	case lineupWithoutMe:
		other := testutil.CreateMember(t, db, 0)
		if _, err := db.Exec(
			`INSERT INTO game_lineup (game_id, member_id) VALUES (?, ?)`, gameID, other); err != nil {
			t.Fatalf("insert lineup: %v", err)
		}
	}
}

// Steht der Spieler des erweiterten Kaders in der Aufstellung, sagt der Termin
// das im Titel und im Beschreibungstext.
func TestFeed_ErwKader_Aufgestellt(t *testing.T) {
	body := lineupFeed(t, "kader_extended_members", "heim", lineupWithMe, "")
	if !strings.Contains(body, "SUMMARY:Heim: Team (mB1 · erw. Kader · aufgestellt) – Test Opponent") {
		t.Errorf("Titel muss den Status 'aufgestellt' tragen, body:\n%s", body)
	}
	if !strings.Contains(body, "DESCRIPTION:Du bist für das Spiel aufgestellt.") {
		t.Errorf("Beschreibung muss den Aufstellungssatz tragen, body:\n%s", body)
	}
}

// Gegenprobe: eine gepflegte Aufstellung ohne das eigene Mitglied ist die
// einzige Lage, in der der Feed „nicht aufgestellt" sagen darf.
func TestFeed_ErwKader_NichtAufgestellt(t *testing.T) {
	body := lineupFeed(t, "kader_extended_members", "auswärts", lineupWithoutMe, "")
	if !strings.Contains(body, "· erw. Kader · nicht aufgestellt)") {
		t.Errorf("Titel muss den Status 'nicht aufgestellt' tragen, body:\n%s", body)
	}
	want := "DESCRIPTION:Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem " +
		"Trainer absprechen\\, ob eine Anwesenheit trotzdem erwünscht ist."
	if !strings.Contains(body, want) {
		t.Errorf("Beschreibung muss den vollständigen Absprache-Satz tragen, body:\n%s", body)
	}
}

// Die Invariante des Changes: aus einer nie gepflegten Aufstellung darf keine
// Absage werden. game_lineup ist leer — das heißt „unbekannt", nicht „nein".
func TestFeed_ErwKader_KeineAufstellungGespeichert(t *testing.T) {
	body := lineupFeed(t, "kader_extended_members", "heim", lineupEmpty, "")
	if !strings.Contains(body, "· erw. Kader · Aufstellung offen)") {
		t.Errorf("Titel muss 'Aufstellung offen' tragen, body:\n%s", body)
	}
	if !strings.Contains(body, "DESCRIPTION:Die Aufstellung für dieses Spiel steht noch nicht fest.") {
		t.Errorf("Beschreibung muss den Offen-Satz tragen, body:\n%s", body)
	}
	if strings.Contains(body, "NICHT aufgestellt") {
		t.Errorf("ohne gepflegte Aufstellung darf der Feed keine Absage melden, body:\n%s", body)
	}
}

// Wer im Stammkader steht, bekommt keinen Status — für ihn ist die Teilnahme
// der Regelfall, ein Vermerk an jedem Spiel wäre Rauschen.
func TestFeed_Stammkader_OhneStatus(t *testing.T) {
	body := lineupFeed(t, "kader_members", "heim", lineupWithoutMe, "")
	if !strings.Contains(body, "SUMMARY:Heim: Team (mB1) – Test Opponent") {
		t.Errorf("Titel des Stammkaders bleibt ohne Zusatz, body:\n%s", body)
	}
	if strings.Contains(body, "aufgestellt") || strings.Contains(body, "Aufstellung offen") {
		t.Errorf("Stammkader bekommt keinen Aufstellungshinweis, body:\n%s", body)
	}
}

// Hängt ein Nutzer regulär UND erweitert am selben Spiel, gewinnt die reguläre
// Zugehörigkeit — und damit entfällt auch der Status.
func TestFeed_DoppelteZugehoerigkeit_RegulaerSchlaegtErweitert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "mB1")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-08-15")
	fillLineup(t, db, gameID, memberID, lineupWithoutMe)

	userToken := testutil.Token(t, userID, "standard", nil)
	srv := prodserver.New(t, db)
	tok := postToken(t, srv, userToken, allTogglesOn())
	res := testutil.Get(t, srv, "/api/calendar/feed/"+tok["token"].(string), "")
	defer res.Body.Close()
	body := unfoldICS(readBody(t, res.Body))

	if n := strings.Count(body, "UID:game-"); n != 1 {
		t.Errorf("erwartet genau ein Spiel-VEVENT, bekommen %d, body:\n%s", n, body)
	}
	if strings.Contains(body, "erw. Kader") || strings.Contains(body, "aufgestellt") {
		t.Errorf("reguläre Zugehörigkeit schlägt die erweiterte — kein Zusatz, body:\n%s", body)
	}
}

// Generische Events haben keine Aufstellung; der Titel bleibt der Terminname.
func TestFeed_GenerischesEvent_OhneStatus(t *testing.T) {
	body := lineupFeed(t, "kader_extended_members", "generisch", lineupEmpty, "")
	if !strings.Contains(body, "UID:game-") {
		t.Fatalf("generisches Event fehlt im Feed, body:\n%s", body)
	}
	if strings.Contains(body, "aufgestellt") || strings.Contains(body, "Aufstellung offen") {
		t.Errorf("generisches Event trägt keinen Aufstellungshinweis, body:\n%s", body)
	}
}

// Die Notiz des Trainers bleibt erhalten und steht vor dem generierten Satz.
func TestFeed_NotizUndStatusStehenBeide(t *testing.T) {
	body := lineupFeed(t, "kader_extended_members", "heim", lineupWithMe, "Bitte Trikots mitbringen")
	want := "DESCRIPTION:Bitte Trikots mitbringen\\n\\nDu bist für das Spiel aufgestellt."
	if !strings.Contains(body, want) {
		t.Errorf("Notiz und Aufstellungssatz müssen beide im DESCRIPTION stehen (Notiz zuerst), body:\n%s", body)
	}
}
