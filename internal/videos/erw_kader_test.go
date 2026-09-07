package videos

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── Erweiterter Kader sieht die Videos seiner Mannschaft ─────────────────────
//
// Change `videos-erweiterter-kader`: Videos waren der letzte Bereich, in dem der
// erweiterte Kader nicht gleichgestellt war — er fand das Video des Spiels, in
// dem er selbst gespielt hat, weder in der Liste noch über die Detail-Route.
// Sichtbarkeit (crud.go/access.go) und Ready-Meldung (worker.go) sind hier
// gemeinsam erweitert worden und müssen deckungsgleich bleiben.

func addExtendedMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}
}

func setMemberStatus(t *testing.T, db *sql.DB, memberID int, status string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE members SET status=? WHERE id=?`, status, memberID); err != nil {
		t.Fatalf("set status: %v", err)
	}
}

// erwFixture: Team A mit einem Video, ein Mitglied NUR im erweiterten Kader von
// Team A plus dessen Elternteil, und ein zweites Team B als Gegenprobe.
type erwFixture struct {
	db        *sql.DB
	srv       *httptest.Server
	teamA     int
	teamB     int
	kaderA    int
	videoA    int
	extUser   int
	extMember int
	extParent int
}

func newErwFixture(t *testing.T) *erwFixture {
	t.Helper()
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	uploader := testutil.CreateUser(t, db, "standard")
	videoA := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")
	_ = testutil.CreateVideo(t, db, teamB, season, uploader, "ready")

	extUser := testutil.CreateUser(t, db, "standard")
	extMember := testutil.CreateMember(t, db, extUser)
	addExtendedMember(t, db, kaderA, extMember)

	extParent := testutil.CreateUser(t, db, "standard")
	addFamilyLink(t, db, extParent, extMember)

	f := &erwFixture{
		db: db, teamA: teamA, teamB: teamB, kaderA: kaderA,
		videoA: videoA, extUser: extUser, extMember: extMember, extParent: extParent,
	}
	f.srv = srv
	return f
}

// listFor ruft die Videoliste für einen Nutzer ab.
func (f *erwFixture) listFor(t *testing.T, userID int) listResp {
	t.Helper()
	res := testutil.Get(t, f.srv, "/api/videos", testutil.Token(t, userID, "standard", nil))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	return decodeList(t, res)
}

func TestListVideos_ErwKaderSiehtTeamvideos(t *testing.T) {
	f := newErwFixture(t)

	lr := f.listFor(t, f.extUser)
	if lr.Total != 1 {
		t.Fatalf("total=%d, want 1 — der erweiterte Kader muss die Videos seiner Mannschaft sehen", lr.Total)
	}
	if lr.Items[0].TeamID != f.teamA {
		t.Errorf("team_id=%d, want %d", lr.Items[0].TeamID, f.teamA)
	}
}

func TestListVideos_ElternDesErwKadersSehenTeamvideos(t *testing.T) {
	f := newErwFixture(t)

	lr := f.listFor(t, f.extParent)
	if lr.Total != 1 {
		t.Fatalf("total=%d, want 1 — Eltern eines erweiterten Mitglieds sehen dessen Teamvideos", lr.Total)
	}
	if lr.Items[0].TeamID != f.teamA {
		t.Errorf("team_id=%d, want %d", lr.Items[0].TeamID, f.teamA)
	}
}

// Die Falle: Förderkinder tragen status='foerderkind'. Ein Statusfilter
// `= 'aktiv'` (wie bei den Stammkader-Zweigen) würde genau die Zielgruppe des
// erweiterten Kaders wieder ausschließen — unsichtbar, weil die Query weiter
// Zeilen liefert.
func TestListVideos_FoerderkindWirdNichtWegenStatusGefiltert(t *testing.T) {
	f := newErwFixture(t)
	setMemberStatus(t, f.db, f.extMember, "foerderkind")

	if lr := f.listFor(t, f.extUser); lr.Total != 1 {
		t.Errorf("Förderkind sieht %d Videos, want 1", lr.Total)
	}
	if lr := f.listFor(t, f.extParent); lr.Total != 1 {
		t.Errorf("Elternteil eines Förderkinds sieht %d Videos, want 1", lr.Total)
	}
}

func TestListVideos_AusgetretenesErwMitgliedSiehtNichts(t *testing.T) {
	f := newErwFixture(t)
	setMemberStatus(t, f.db, f.extMember, "ausgetreten")

	if lr := f.listFor(t, f.extUser); lr.Total != 0 {
		t.Errorf("ausgetretenes Mitglied sieht %d Videos, want 0", lr.Total)
	}
	if lr := f.listFor(t, f.extParent); lr.Total != 0 {
		t.Errorf("Elternteil eines ausgetretenen Mitglieds sieht %d Videos, want 0", lr.Total)
	}
}

func TestGetVideo_ErwKaderDarfDetailLesen(t *testing.T) {
	f := newErwFixture(t)

	res := testutil.Get(t, f.srv, "/api/videos/"+strconv.Itoa(f.videoA),
		testutil.Token(t, f.extUser, "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 — Liste und Detail müssen dieselbe Menge zulassen", res.StatusCode)
	}
}

func TestGetVideo_FremdesTeamBleibtVerboten(t *testing.T) {
	f := newErwFixture(t)
	// Ein erweitertes Mitglied von Team B hat auf das Video von Team A keinen
	// Zugriff — die Erweiterung ist team- und saisongebunden, kein Freibrief.
	var activeSeason int
	if err := f.db.QueryRow(`SELECT id FROM seasons WHERE is_active=1`).Scan(&activeSeason); err != nil {
		t.Fatalf("aktive Saison: %v", err)
	}
	kaderB := testutil.CreateKader(t, f.db, f.teamB, activeSeason)

	otherUser := testutil.CreateUser(t, f.db, "standard")
	otherMember := testutil.CreateMember(t, f.db, otherUser)
	addExtendedMember(t, f.db, kaderB, otherMember)

	res := testutil.Get(t, f.srv, "/api/videos/"+strconv.Itoa(f.videoA),
		testutil.Token(t, otherUser, "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 403/404 für ein fremdes Team", res.StatusCode)
	}
}

// Die Zusage aus der Spec: niemand wird über ein Video benachrichtigt, das er
// anschließend nicht öffnen kann — hier die andere Richtung, wer es öffnen darf,
// wird auch benachrichtigt.
func TestPushRecipients_ErwKaderUndEltern(t *testing.T) {
	f := newErwFixture(t)
	wk, _, _ := newTestWorker(t, f.db, nil)

	uids, err := wk.pushRecipients(f.videoA)
	if err != nil {
		t.Fatalf("pushRecipients: %v", err)
	}
	got := map[int]bool{}
	for _, id := range uids {
		got[id] = true
	}
	if !got[f.extUser] {
		t.Errorf("erweitertes Mitglied (user %d) fehlt in %v", f.extUser, uids)
	}
	if !got[f.extParent] {
		t.Errorf("Elternteil (user %d) fehlt in %v", f.extParent, uids)
	}
}
