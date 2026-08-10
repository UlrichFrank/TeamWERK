package games_test

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// ── Absage-Benachrichtigung bei DELETE /api/games/{id} ───────────────────────
//
// Invarianten (openspec/changes/absage-benachrichtigung):
//   - keine Absage-Meldung ohne Terminname + Datum TT.MM.JJJJ + Aktor,
//   - Team-Meldung ohne Linkziel (der Termin ist weg),
//   - Dienst-Meldung behält Wortlaut und /dienste-Link,
//   - `silent` nur mit Capability, sonst still ignoriert (kein 403),
//   - Grund wird nirgends persistiert.

// sentNotification ist eine aufgezeichnete notify.Send-Zustellung.
type sentNotification struct {
	userIDs  []int
	category string
	title    string
	body     string
	url      string
}

// notifyRecorder ersetzt den notify.Send-Seam und sammelt alle Zustellungen.
// Threadsicher, weil dispatchRegenNotifications pro Meldung eine Goroutine startet.
type notifyRecorder struct {
	mu   sync.Mutex
	sent []sentNotification
}

func recordNotifications(t *testing.T) *notifyRecorder {
	t.Helper()
	rec := &notifyRecorder{}
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, category, title, body, url string) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.sent = append(rec.sent, sentNotification{userIDs, category, title, body, url})
	}
	t.Cleanup(func() { notify.Send = orig })
	return rec
}

func (r *notifyRecorder) byTitle(title string) []sentNotification {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []sentNotification
	for _, n := range r.sent {
		if n.title == title {
			out = append(out, n)
		}
	}
	return out
}

// one erwartet genau eine Meldung mit dem Titel.
func (r *notifyRecorder) one(t *testing.T, title string) sentNotification {
	t.Helper()
	got := r.byTitle(title)
	if len(got) != 1 {
		t.Fatalf("erwartet genau eine Meldung %q, got %d (alle: %v)", title, len(got), r.titles())
	}
	return got[0]
}

func (r *notifyRecorder) titles() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, n := range r.sent {
		out = append(out, n.title)
	}
	return out
}

// ── Fixture ──────────────────────────────────────────────────────────────────

type cancelFixture struct {
	db       *sql.DB
	season   int
	team     int
	game     int
	vorstand int // löscht, heißt "Tim Meier"
	helper   int // Dienst-Zugeteilter
	srv      *httptest.Server
	hub      *hub.EventHub
	rec      *notifyRecorder
}

// newCancelFixture baut Saison, Team und ein Heimspiel gegen "HSG Ostfildern"
// am 14.09.2026 — die Werte aus den Spec-Szenarien.
func newCancelFixture(t *testing.T) *cancelFixture {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2026/27")
	team := testutil.CreateTeam(t, db, "Team A")
	game := createNamedGame(t, db, season, team, "2026-09-14", "HSG Ostfildern")

	vorstand := testutil.CreateUser(t, db, "standard")
	setUserName(t, db, vorstand, "Tim", "Meier")
	helper := testutil.CreateUser(t, db, "standard")

	rec := recordNotifications(t)
	srv, sharedHub := prodserver.NewWithHub(t, db)
	return &cancelFixture{
		db: db, season: season, team: team, game: game,
		vorstand: vorstand, helper: helper,
		srv: srv, hub: sharedHub, rec: rec,
	}
}

func (f *cancelFixture) vorstandToken(t *testing.T) string {
	t.Helper()
	return testutil.Token(t, f.vorstand, "standard", []string{"vorstand"})
}

// withDuty trägt f.helper auf einen Dienst des Spiels ein.
func (f *cancelFixture) withDuty(t *testing.T) {
	t.Helper()
	dt := insertDutyType(t, f.db, "Kasse", 1.0)
	slot := insertDutySlot(t, f.db, dt, f.season, f.team, f.game, "2026-09-14")
	insertDutyAssignment(t, f.db, slot, f.helper, "assigned")
}

func (f *cancelFixture) delete(t *testing.T, token string, body any) *http.Response {
	t.Helper()
	return testutil.Do(t, f.srv, http.MethodDelete, fmt.Sprintf("/api/games/%d", f.game), token, body)
}

// createNamedGame legt ein Heimspiel mit frei wählbarem Gegner an (testutil.CreateGame
// setzt "Test Opponent" fest — hier ist der Gegnername Teil der Assertion).
func createNamedGame(t *testing.T, db *sql.DB, seasonID, teamID int, date, opponent string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?,?,?,?,'heim',1)`,
		seasonID, opponent, date, "18:00")
	if err != nil {
		t.Fatalf("createNamedGame: %v", err)
	}
	id, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, id, teamID); err != nil {
		t.Fatalf("createNamedGame game_teams: %v", err)
	}
	return int(id)
}

func setUserName(t *testing.T, db *sql.DB, userID int, first, last string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE users SET first_name=?, last_name=? WHERE id=?`, first, last, userID); err != nil {
		t.Fatalf("setUserName: %v", err)
	}
}

// ── Team-Meldung ─────────────────────────────────────────────────────────────

// Der Kernfall: die Meldung benennt Gegner, Datum, Aktor und Grund — und führt
// nirgendwo mehr hin, weil der Termin gelöscht ist.
func TestDeleteGame_AbsageNenntGegnerDatumAktorUndGrund(t *testing.T) {
	f := newCancelFixture(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spiel abgesagt")
	for _, want := range []string{"HSG Ostfildern", "14.09.2026", "Tim Meier", "Halle gesperrt"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
	if n.category != "games" {
		t.Errorf("Kategorie games erwartet, got %q", n.category)
	}
	// Leeres Linkziel: /termine zeigt den Termin nach der Löschung nicht mehr.
	if n.url != "" {
		t.Errorf("url muss leer sein, got %q", n.url)
	}
}

// Ohne Grund darf keine leere Grund-Einleitung stehenbleiben ("… Tim Meier: .").
func TestDeleteGame_OhneGrundKeinLeererGrundSatz(t *testing.T) {
	f := newCancelFixture(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spiel abgesagt")
	if strings.Contains(n.body, ":") {
		t.Errorf("ohne Grund darf keine Grund-Einleitung im Body stehen, got %q", n.body)
	}
	if !strings.HasSuffix(n.body, "Abgesagt von Tim Meier.") {
		t.Errorf("Body muss nach dem Aktor-Satz enden, got %q", n.body)
	}
}

// Rückwärtskompatibilität: alte PWA-Installationen senden gar keinen Body.
func TestDeleteGame_OhneBodyBleibtErfolgreich(t *testing.T) {
	f := newCancelFixture(t)

	res := testutil.Delete(t, f.srv, fmt.Sprintf("/api/games/%d", f.game), f.vorstandToken(t))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200 ohne Body, got %d", res.StatusCode)
	}
	if n := countRows(t, f.db, "games", "id=?", f.game); n != 0 {
		t.Errorf("Spiel muss gelöscht sein, got %d", n)
	}
	got := f.rec.one(t, "Spiel abgesagt")
	if !strings.Contains(got.body, "HSG Ostfildern") {
		t.Errorf("Meldung muss trotzdem versendet werden, got %q", got.body)
	}
}

// Kaputtes JSON darf die Löschung ebenso wenig abbrechen wie ein fehlender Body.
func TestDeleteGame_KaputterBodyBleibtErfolgreich(t *testing.T) {
	f := newCancelFixture(t)

	req, err := http.NewRequest(http.MethodDelete,
		f.srv.URL+fmt.Sprintf("/api/games/%d", f.game), strings.NewReader("{nicht json"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Authorization", f.vorstandToken(t))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200 bei kaputtem Body, got %d", res.StatusCode)
	}
	if len(f.rec.byTitle("Spiel abgesagt")) != 1 {
		t.Errorf("Meldung muss versendet werden, got %v", f.rec.titles())
	}
}

// Ein überlanger Grund darf die Löschung nicht abbrechen, sondern wird still
// auf 200 Runen gekürzt.
func TestDeleteGame_UeberlangerGrundWirdGekuerzt(t *testing.T) {
	f := newCancelFixture(t)
	reason := strings.Repeat("ä", 500)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": reason})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spiel abgesagt")
	if !strings.Contains(n.body, strings.Repeat("ä", 200)) {
		t.Errorf("Body muss die ersten 200 Zeichen des Grundes enthalten, got %q", n.body)
	}
	if strings.Contains(n.body, strings.Repeat("ä", 201)) {
		t.Errorf("Body enthält mehr als 200 Zeichen des Grundes, got %q", n.body)
	}
}

// ── Dienst-Meldung ───────────────────────────────────────────────────────────

// Der Satz "Dein Dienst zum X am Y wurde gelöscht." ist in specs/push-duties als
// Wortlaut festgeschrieben; Aktor und Grund kommen nur hinten dran, der Link
// bleibt /dienste (die Dienstbörse existiert nach der Löschung weiter).
func TestDeleteGame_DienstMeldungBehaeltWortlautUndLink(t *testing.T) {
	f := newCancelFixture(t)
	f.withDuty(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Dienst entfällt")
	const satz = "Dein Dienst zum HSG Ostfildern am 14.09.2026 wurde gelöscht."
	if !strings.Contains(n.body, satz) {
		t.Errorf("Body muss %q enthalten, got %q", satz, n.body)
	}
	for _, want := range []string{"Tim Meier", "Halle gesperrt"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
	if n.url != "/dienste" {
		t.Errorf("Link muss /dienste bleiben, got %q", n.url)
	}
	if n.category != "duties" {
		t.Errorf("Kategorie duties erwartet, got %q", n.category)
	}
}

// ── silent ───────────────────────────────────────────────────────────────────

// Vorstand hat die Capability: keine der beiden Meldungen geht raus — das
// SSE-Live-Update aber sehr wohl, sonst zeigten offene Sessions den gelöschten
// Termin weiter an.
func TestDeleteGame_VorstandSilentUnterdruecktBeideMeldungen(t *testing.T) {
	f := newCancelFixture(t)
	f.withDuty(t)

	// Der Dienst-Zugeteilte ist Teil des Broadcast-Publikums (extraUserIDs).
	ch := f.hub.SubscribeUser(f.helper)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"silent": true, "reason": "Import-Dublette"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	if n := countRows(t, f.db, "games", "id=?", f.game); n != 0 {
		t.Errorf("Spiel muss trotz silent gelöscht sein, got %d", n)
	}

	for _, title := range []string{"Spiel abgesagt", "Dienst entfällt"} {
		if got := f.rec.byTitle(title); len(got) != 0 {
			t.Errorf("silent: %q darf nicht versendet werden, got %v", title, got)
		}
	}
	if ev, ok := recvWithin(ch, time.Second); !ok || ev != "games" {
		t.Errorf("Live-Update ist nicht unterdrückbar, got %q ok=%v", ev, ok)
	}
}

// Ein reiner Trainer darf löschen, aber nicht stummschalten. Das Flag wird
// ignoriert statt mit 403 quittiert — die Löschung selbst ist erlaubt.
func TestDeleteGame_TrainerSilentWirdIgnoriert(t *testing.T) {
	f := newCancelFixture(t)
	f.withDuty(t)
	trainerU := makeTrainer(t, f.db, f.team, f.season)
	setUserName(t, f.db, trainerU, "Lea", "Kunz")
	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})

	res := f.delete(t, token, map[string]any{"silent": true, "reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	team := f.rec.one(t, "Spiel abgesagt")
	if !strings.Contains(team.body, "Lea Kunz") {
		t.Errorf("Team-Meldung muss trotz silent rausgehen, got %q", team.body)
	}
	if len(f.rec.byTitle("Dienst entfällt")) != 1 {
		t.Errorf("Dienst-Meldung muss trotz silent rausgehen, got %v", f.rec.titles())
	}
}

// ── Fehlerfälle: keine Meldung ohne Löschung ─────────────────────────────────

// Ein reiner Trainer einer fremden Mannschaft bekommt 403 — und es darf keine
// Absage-Meldung entstehen (weder Spiel noch Dienst existieren danach anders).
func TestDeleteGame_FremdesTeamKeineBenachrichtigung(t *testing.T) {
	f := newCancelFixture(t)
	otherTeam := testutil.CreateTeam(t, f.db, "Team B")
	trainerU := makeTrainer(t, f.db, otherTeam, f.season)
	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})

	res := f.delete(t, token, map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, got %d", res.StatusCode)
	}
	if n := countRows(t, f.db, "games", "id=?", f.game); n != 1 {
		t.Errorf("Spiel muss erhalten bleiben, got %d", n)
	}
	if titles := f.rec.titles(); len(titles) != 0 {
		t.Errorf("bei 403 darf nichts versendet werden, got %v", titles)
	}
}

func TestDeleteGame_UnbekannteIDKeineBenachrichtigung(t *testing.T) {
	f := newCancelFixture(t)

	res := testutil.Do(t, f.srv, http.MethodDelete, "/api/games/999999",
		f.vorstandToken(t), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("erwartet 404, got %d", res.StatusCode)
	}
	if titles := f.rec.titles(); len(titles) != 0 {
		t.Errorf("bei 404 darf nichts versendet werden, got %v", titles)
	}
}

// ── Nichtpersistenz des Grundes ──────────────────────────────────────────────

// syncBuf ist ein threadsicherer Log-Puffer — notify-Goroutinen können nach dem
// Response noch schreiben, und der Race-Detector liest mit.
type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// Der Löschgrund hat bewusst keine Heimat (design.md §2): er existiert nur in
// der zugestellten Nachricht. Diese Zusage wird hier mechanisch geprüft — voller
// Scan aller Tabellen und Spalten plus Log-Puffer, inklusive Poison-Sanity für
// den Scanner selbst.
func TestDeleteGame_GrundWirdNichtPersistiert(t *testing.T) {
	const marker = "MARKER-8f3a-Loeschgrund-Nichtpersistenz"
	f := newCancelFixture(t)
	f.withDuty(t)

	var logBuf syncBuf
	oldOut, oldFlags := log.Writer(), log.Flags()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(oldOut); log.SetFlags(oldFlags) })

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": marker})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	body := make([]byte, 1<<16)
	n, _ := res.Body.Read(body)

	// Der Grund muss in der Meldung stehen — sonst prüfte der Rest nichts.
	if got := f.rec.one(t, "Spiel abgesagt"); !strings.Contains(got.body, marker) {
		t.Fatalf("Grund fehlt in der Meldung, Assertion wäre wertlos: %q", got.body)
	}
	if strings.Contains(string(body[:n]), marker) {
		t.Error("Grund wird in der Response zurückgespiegelt")
	}
	if strings.Contains(logBuf.String(), marker) {
		t.Error("Grund landet im Log")
	}
	if table, col := testutil.FindStringInDB(t, f.db, marker); table != "" {
		t.Errorf("Grund in der DB persistiert: %s.%s", table, col)
	}

	// Poison-Sanity: ein kaputter Scanner meldet immer „nicht gefunden".
	createNamedGame(t, f.db, f.season, f.team, "2026-10-01", marker)
	if table, col := testutil.FindStringInDB(t, f.db, marker); table != "games" || col != "opponent" {
		t.Fatalf("DB-Scanner findet den absichtlich gesetzten Marker nicht (got %s.%s) — Assertion oben wäre wertlos", table, col)
	}
}
