package games_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── Empfängermenge der Spiel-Meldungen ───────────────────────────────────────
//
// Invariante des Changes `terminmeldungen-erweiterter-kader`: Anlage, Änderung
// und Absage eines Spiels erreichen Stammkader UND erweiterten Kader, jeweils
// samt Eltern. Vor dem Change löste `games` seine Empfänger über
// `player_memberships` auf — eine View über `kader_members` —, der erweiterte
// Kader fiel dadurch komplett heraus und erfuhr von einem Spiel nur über den
// Kalender-Feed.

// audienceFixture erweitert die cancelFixture um einen echten Kader mit vier
// Beteiligten: Stammspieler + Elternteil, erweiterter Spieler + Elternteil.
type audienceFixture struct {
	*cancelFixture
	player       int // user_id, Stammkader
	playerParent int // user_id, Elternteil des Stammspielers
	ext          int // user_id, erweiterter Kader
	extParent    int // user_id, Elternteil des erweiterten Spielers
	trainer      int // user_id, Trainer des Kaders
	outsider     int // user_id, in keinem Kader dieser Mannschaft
	kader        int
}

func newAudienceFixture(t *testing.T) *audienceFixture {
	t.Helper()
	f := newCancelFixture(t)
	kader := testutil.CreateKader(t, f.db, f.team, f.season)

	player := testutil.CreateUser(t, f.db, "standard")
	playerMember := testutil.CreateMember(t, f.db, player)
	addKaderMemberRow(t, f.db, kader, playerMember)
	playerParent := testutil.CreateUser(t, f.db, "standard")
	addFamilyLinkRow(t, f.db, playerParent, playerMember)

	ext := testutil.CreateUser(t, f.db, "standard")
	extMember := testutil.CreateMember(t, f.db, ext)
	addExtendedMemberRow(t, f.db, kader, extMember)
	extParent := testutil.CreateUser(t, f.db, "standard")
	addFamilyLinkRow(t, f.db, extParent, extMember)

	trainer := testutil.CreateUser(t, f.db, "standard")
	trainerMember := testutil.CreateMember(t, f.db, trainer)
	testutil.AddClubFunction(t, f.db, trainerMember, "trainer")
	testutil.AddKaderTrainer(t, f.db, kader, trainerMember)
	setUserName(t, f.db, trainer, "Lea", "Wagner")

	outsider := testutil.CreateUser(t, f.db, "standard")
	testutil.CreateMember(t, f.db, outsider)

	return &audienceFixture{
		cancelFixture: f,
		player:        player, playerParent: playerParent,
		ext: ext, extParent: extParent, trainer: trainer,
		outsider: outsider, kader: kader,
	}
}

func addKaderMemberRow(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

func addExtendedMemberRow(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}
}

func addFamilyLinkRow(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, memberID); err != nil {
		t.Fatalf("family_links: %v", err)
	}
}

// assertAudience prüft eine aufgezeichnete Meldung gegen die vier Pflicht-
// Empfänger und den Nicht-Empfänger.
func (f *audienceFixture) assertAudience(t *testing.T, n sentNotification) {
	t.Helper()
	got := map[int]bool{}
	for _, id := range n.userIDs {
		got[id] = true
	}
	for _, want := range []struct {
		id    int
		rolle string
	}{
		{f.player, "Stammkader"},
		{f.playerParent, "Elternteil des Stammspielers"},
		{f.ext, "erweiterter Kader"},
		{f.extParent, "Elternteil des erweiterten Spielers"},
		{f.trainer, "Trainer des Kaders"},
	} {
		if !got[want.id] {
			t.Errorf("%s (user %d) fehlt in der Empfängermenge %v", want.rolle, want.id, n.userIDs)
		}
	}
	if got[f.outsider] {
		t.Errorf("Mitglied ohne Kader-Zugehörigkeit (user %d) darf keine Meldung bekommen", f.outsider)
	}
	if len(n.userIDs) != len(got) {
		t.Errorf("Empfängermenge enthält Dubletten: %v", n.userIDs)
	}
}

func TestCreateGame_ErwKaderUndElternWerdenBenachrichtigt(t *testing.T) {
	f := newAudienceFixture(t)

	res := testutil.Do(t, f.srv, http.MethodPost, "/api/games",
		f.vorstandToken(t), f.createGameBody("heim", "TSV Neuhausen"))
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, got %d", res.StatusCode)
	}

	f.assertAudience(t, f.rec.one(t, "Neues Spiel"))
}

func TestUpdateGame_ErwKaderWirdBenachrichtigt(t *testing.T) {
	f := newAudienceFixture(t)

	res := f.put(t, f.vorstandToken(t), f.updateGameBody("2026-09-14", "19:30"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spielinfo geändert")
	f.assertAudience(t, n)
	if n.category != "games" {
		t.Errorf("Kategorie games erwartet, got %q", n.category)
	}
}

func TestDeleteGame_ErwKaderWirdBenachrichtigt(t *testing.T) {
	f := newAudienceFixture(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spiel abgesagt")
	f.assertAudience(t, n)
	// Die Absage behält ihr leeres Linkziel — die größere Empfängermenge ändert
	// daran nichts.
	if n.url != "" {
		t.Errorf("Absage-Meldung muss ohne Linkziel bleiben, got %q", n.url)
	}
}

// `silent` unterdrückt die Meldung für ALLE — die neue Empfängermenge darf
// keinen Weg an der Stummschaltung vorbei öffnen.
func TestDeleteGame_SilentUnterdruecktAuchErwKader(t *testing.T) {
	f := newAudienceFixture(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	if got := f.rec.byTitle("Spiel abgesagt"); len(got) != 0 {
		t.Errorf("silent muss die Absage-Meldung vollständig unterdrücken, got %d Meldungen an %v",
			len(got), got[0].userIDs)
	}
}

// Der Auslöser bekommt seine eigene Meldung — die Bestätigung, dass sie raus ist,
// und der Beleg, welchen Wortlaut die Mannschaft gelesen hat. Ohne diesen Test
// wäre die Regel ein Nebeneffekt statt einer Zusage: ein „exclude actor", wie es
// die Mitfahrgelegenheiten kennen, wäre eine Zeile und niemandem aufgefallen.
func TestCreateGame_AusloeserBekommtDieEigeneMeldung(t *testing.T) {
	f := newAudienceFixture(t)
	token := testutil.Token(t, f.trainer, "standard", []string{"trainer"})

	res := testutil.Do(t, f.srv, http.MethodPost, "/api/games", token,
		f.createGameBody("heim", "TSV Neuhausen"))
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Neues Spiel")
	for _, id := range n.userIDs {
		if id == f.trainer {
			return
		}
	}
	t.Errorf("der anlegende Trainer (user %d) fehlt in der Empfängermenge %v", f.trainer, n.userIDs)
}
