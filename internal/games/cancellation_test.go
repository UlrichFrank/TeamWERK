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
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, category, title, body, url string, _ ...notify.Option) {
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

// newCancelFixtureBase baut Saison, Team und ein Heimspiel gegen
// "HSG Ostfildern" am 14.09.2026 — die Werte aus den Spec-Szenarien. Gemeinsame
// Basis für newCancelFixture (mit notify-Recorder) und newCancelFixtureRealNotify
// (mit dem echten notify.Send).
func newCancelFixtureBase(t *testing.T) *cancelFixture {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2026/27")
	team := testutil.CreateTeam(t, db, "Team A")
	game := createNamedGame(t, db, season, team, "2026-09-14", "HSG Ostfildern")

	vorstand := testutil.CreateUser(t, db, "standard")
	setUserName(t, db, vorstand, "Tim", "Meier")
	helper := testutil.CreateUser(t, db, "standard")

	srv, sharedHub := prodserver.NewWithHub(t, db)
	return &cancelFixture{
		db: db, season: season, team: team, game: game,
		vorstand: vorstand, helper: helper,
		srv: srv, hub: sharedHub,
	}
}

// newCancelFixture baut die Basis-Fixture und ersetzt notify.Send durch einen
// Recorder — für Tests, die nur prüfen, WAS versendet würde (Titel/Body/URL),
// ohne den echten Fan-out (Push/Email/Event-Log) auszulösen.
func newCancelFixture(t *testing.T) *cancelFixture {
	t.Helper()
	f := newCancelFixtureBase(t)
	f.rec = recordNotifications(t)
	return f
}

// newCancelFixtureRealNotify ist wie newCancelFixture, lässt notify.Send aber
// unangetastet. Nötig für TestDeleteGame_GrundStehtImEventLog: der Recorder
// ersetzt notify.Send vollständig und würde damit auch eventlog.Record nie
// laufen lassen. Push/Email sind dank leerer VAPID-/SMTP-Config in
// testutil.TestConfig() ein No-op, eventlog.Record läuft synchron im
// Request-Handler mit — kein Warten nötig.
func newCancelFixtureRealNotify(t *testing.T) *cancelFixture {
	t.Helper()
	return newCancelFixtureBase(t)
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

// ── Titel passend zum event_type ─────────────────────────────────────────────

// setEventType schaltet das Fixture-Spiel auf einen anderen event_type um und
// setzt dabei den Terminnamen (bei generisch trägt `opponent` den Event-Namen).
func (f *cancelFixture) setEventType(t *testing.T, eventType, name string) {
	t.Helper()
	if _, err := f.db.Exec(
		`UPDATE games SET event_type=?, is_home=?, opponent=? WHERE id=?`,
		eventType, eventType == "heim", name, f.game); err != nil {
		t.Fatalf("setEventType: %v", err)
	}
}

// Ein generisches Event ist kein Spiel — die Absage darf nicht "Spiel abgesagt"
// heißen. Der Body bleibt unverändert (Event-Name aus `opponent`).
func TestDeleteGame_GenerischesEventMeldetTerminAbgesagt(t *testing.T) {
	f := newCancelFixture(t)
	f.setEventType(t, "generisch", "Sommerfest")

	res := f.delete(t, f.vorstandToken(t), map[string]any{"reason": "Regen"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	if got := f.rec.byTitle("Spiel abgesagt"); len(got) != 0 {
		t.Errorf("generisches Event darf nicht als Spiel gemeldet werden, got %v", f.rec.titles())
	}
	n := f.rec.one(t, "Termin abgesagt")
	for _, want := range []string{"Sommerfest", "14.09.2026", "Tim Meier", "Regen"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
}

// Auswärtsspiele bleiben Spiele — die Titel-Wahl hängt am event_type, nicht am
// Heimrecht.
func TestDeleteGame_AuswaertsspielBleibtSpielAbgesagt(t *testing.T) {
	f := newCancelFixture(t)
	f.setEventType(t, "auswärts", "HSG Ostfildern")

	res := f.delete(t, f.vorstandToken(t), map[string]any{})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	if got := f.rec.byTitle("Termin abgesagt"); len(got) != 0 {
		t.Errorf("Auswärtsspiel muss als Spiel gemeldet werden, got %v", f.rec.titles())
	}
	f.rec.one(t, "Spiel abgesagt")
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

// ── Der Grund im Event-Log ───────────────────────────────────────────────────

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

// Seit dem event-log-Change hat der Löschgrund eine Heimat: er lebt als Teil
// des Meldungstexts in user_events.body (design.md Decision 10, ersetzt die
// frühere Festlegung "wird nirgends persistiert"). Diese Zusage wird hier
// mechanisch geprüft — voller Scan aller Tabellen und Spalten AUSSER
// user_events, plus Response und Log-Puffer, inklusive Poison-Sanity für den
// Scanner selbst. user_events ist die einzig erlaubte Fundstelle; jede andere
// Tabelle bleibt verboten (Games/Duty-Slots referenzieren den Grund nach wie
// vor nicht — es gibt kein games.status='cancelled').
//
// Der Recorder aus newCancelFixture würde notify.Send vollständig ersetzen und
// damit auch eventlog.Record nie ausführen — deshalb hier bewusst
// newCancelFixtureRealNotify statt newCancelFixture.
func TestDeleteGame_GrundStehtImEventLog(t *testing.T) {
	const marker = "MARKER-8f3a-Loeschgrund-EventLog"
	f := newCancelFixtureRealNotify(t)
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
	if strings.Contains(string(body[:n]), marker) {
		t.Error("Grund wird in der Response zurückgespiegelt")
	}
	if strings.Contains(logBuf.String(), marker) {
		t.Error("Grund landet im Server-Log")
	}

	// Erwartete Fundstelle: die Dienst-Absage in user_events (Kategorie
	// "duties", Link bleibt "/dienste" — die Dienstbörse existiert nach der
	// Löschung weiter). Die Team-Meldung ginge nur an tatsächliche
	// Kader-Mitglieder der aktiven Saison, die diese schlanke Fixture bewusst
	// nicht anlegt — der Dienst-Zugeteilte (f.helper) genügt als Nachweis, dass
	// der Grund im Log landet.
	var count int
	var url string
	if err := f.db.QueryRow(
		`SELECT COUNT(*), COALESCE(MAX(url), '') FROM user_events
		  WHERE category = 'duties' AND body LIKE ?`,
		"%"+marker+"%",
	).Scan(&count, &url); err != nil {
		t.Fatalf("user_events-Query: %v", err)
	}
	if count == 0 {
		t.Fatal("Grund muss als user_events-Zeile stehen — sonst prüfte der Rest nichts")
	}
	if url != "/dienste" {
		t.Errorf("url = %q, want /dienste", url)
	}

	// Verbotene Fundstellen: jede Tabelle außer user_events.
	if table, col := testutil.FindStringInDBExcept(t, f.db, marker, "user_events"); table != "" {
		t.Errorf("Grund zusätzlich außerhalb des Event-Logs persistiert: %s.%s", table, col)
	}

	// Poison-Sanity: ein kaputter Scanner meldet immer „nicht gefunden".
	createNamedGame(t, f.db, f.season, f.team, "2026-10-01", marker)
	if table, col := testutil.FindStringInDBExcept(t, f.db, marker, "user_events"); table != "games" || col != "opponent" {
		t.Fatalf("DB-Scanner findet den absichtlich gesetzten Marker nicht (got %s.%s) — Assertion oben wäre wertlos", table, col)
	}
}

// Retention macht den Grund flüchtig, nicht der Versand selbst: solange
// silent NICHT gesetzt ist, bleibt der Grund für die Dauer der Event-Log-
// Retention (3 Tage nach Ansicht) nachlesbar — geprüft oben. Bei silent
// entsteht (Spec „Stumme Löschung schreibt keinen Log") gar keine Zeile, siehe
// TestDeleteGame_SilentSchreibtKeineEventLogZeile weiter unten.

// ── silent unterdrückt auch den Event-Log ────────────────────────────────────

// Spec „Absagegründe leben im Log statt nur im Zustellkanal", Szenario „Stumme
// Löschung schreibt keinen Log": silent unterdrückt notify.Send vollständig
// (Handler-Code, nicht die Fassade) — und damit bleibt auch eventlog.Record
// ungerufen. Braucht wie oben den echten notify.Send statt des Recorders.
func TestDeleteGame_SilentSchreibtKeineEventLogZeile(t *testing.T) {
	f := newCancelFixtureRealNotify(t)
	f.withDuty(t)

	res := f.delete(t, f.vorstandToken(t), map[string]any{"silent": true, "reason": "Import-Dublette"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	var count int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count); err != nil {
		t.Fatalf("user_events-Query: %v", err)
	}
	if count != 0 {
		t.Errorf("silent muss auch den Event-Log unterdrücken, got %d Zeile(n)", count)
	}
}
