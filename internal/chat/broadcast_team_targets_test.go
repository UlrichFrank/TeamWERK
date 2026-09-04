package chat_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// broadcastWorld: zwei Teams in der aktiven Saison. Der Absender trainiert nur
// Team A. Team B existiert, damit jede Assertion über "sein Wirkungsbereich"
// eine Gegenprobe hat.
type broadcastWorld struct {
	db  *sql.DB
	srv *httptest.Server

	trainer      int // Kader-Trainer von Team A, sonst nichts
	teamA, teamB int
	spielerA     int // regulärer Kader A
	erweitertA   int // erweiterter Kader A
	elternA      int // Elternteil eines Spielers in A
	spielerB     int // regulärer Kader B
	fremdTrainer int // Kader-Trainer von Team B
}

// broadcastTargetSrv mountet beide Mitteilungs-Routen und verschluckt Pushes.
func broadcastTargetSrv(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	h.SetPushFn(func(*sql.DB, *appconfig.Config, int, string, string, string, int) {})
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/broadcasts", h.SendBroadcast)
		r.Get("/api/chat/broadcast-targets", h.ListBroadcastTargets)
	})
}

func setupBroadcastWorld(t *testing.T) *broadcastWorld {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	w := &broadcastWorld{db: db}
	w.teamA = testutil.CreateTeam(t, db, "mB1")
	w.teamB = testutil.CreateTeam(t, db, "mC2")
	kaderA := testutil.CreateKader(t, db, w.teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, w.teamB, seasonID)

	w.trainer = testutil.CreateUser(t, db, "standard")
	mTrainer := testutil.CreateMember(t, db, w.trainer)
	addTrainer(t, db, kaderA, mTrainer)

	w.fremdTrainer = testutil.CreateUser(t, db, "standard")
	mFremd := testutil.CreateMember(t, db, w.fremdTrainer)
	addTrainer(t, db, kaderB, mFremd)

	w.spielerA = testutil.CreateUser(t, db, "standard")
	mSpielerA := testutil.CreateMember(t, db, w.spielerA)
	addKaderMember(t, db, kaderA, mSpielerA)

	// Der erweiterte Kader zählt zu team_spieler — dieselbe Query wie die
	// Chat-Standardgruppe, und für eine Trainingsabsage genau richtig.
	w.erweitertA = testutil.CreateUser(t, db, "standard")
	mErweitert := testutil.CreateMember(t, db, w.erweitertA)
	if _, err := db.Exec(
		`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderA, mErweitert); err != nil {
		t.Fatalf("erweiterter Kader: %v", err)
	}

	w.elternA = testutil.CreateUser(t, db, "standard")
	kind := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderA, kind)
	linkParent(t, db, w.elternA, kind)

	w.spielerB = testutil.CreateUser(t, db, "standard")
	mSpielerB := testutil.CreateMember(t, db, w.spielerB)
	addKaderMember(t, db, kaderB, mSpielerB)

	w.srv = broadcastTargetSrv(t, db)
	return w
}

func addTrainer(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("addTrainer: %v", err)
	}
}

func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("addKaderMember: %v", err)
	}
}

func (w *broadcastWorld) token(t *testing.T, userID int, functions []string) string {
	t.Helper()
	return testutil.Token(t, userID, "standard", functions)
}

// send schickt eine Mitteilung und liefert Status plus (bei 201) die Empfänger
// aus broadcast_reads.
func (w *broadcastWorld) send(t *testing.T, token string, targets []any) (int, []int, sendResult) {
	t.Helper()
	res := testutil.Post(t, w.srv, "/api/chat/broadcasts", token,
		map[string]any{"body": "Ansage", "targets": targets})
	if res.StatusCode != http.StatusCreated {
		res.Body.Close()
		return res.StatusCode, nil, sendResult{}
	}
	body := decodeJSON[sendResult](t, res)
	rows, err := w.db.Query(
		`SELECT user_id FROM broadcast_reads WHERE broadcast_id = ? ORDER BY user_id`, body.ID)
	if err != nil {
		t.Fatalf("read broadcast_reads: %v", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	return res.StatusCode, ids, body
}

func targetsFor(t *testing.T, srv *httptest.Server, token string) []chat.AllowedTarget {
	t.Helper()
	res := testutil.Get(t, srv, "/api/chat/broadcast-targets", token)
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("GET broadcast-targets: status %d, want 200", res.StatusCode)
	}
	return decodeJSON[[]chat.AllowedTarget](t, res)
}

func hasTarget(targets []chat.AllowedTarget, kind string, teamID *int) bool {
	for _, a := range targets {
		if a.Kind != kind {
			continue
		}
		if (a.TeamID == nil) != (teamID == nil) {
			continue
		}
		if a.TeamID != nil && *a.TeamID != *teamID {
			continue
		}
		return true
	}
	return false
}

// TC: Der Trainer sieht genau die Gruppen seiner eigenen Kader — plus
// "Alle Trainer" (das Kollegium, nicht ein Publikum). Kein vereinsweites Ziel,
// kein fremdes Team.
func TestBroadcastTargets_TrainerNurEigeneKader(t *testing.T) {
	w := setupBroadcastWorld(t)
	got := targetsFor(t, w.srv, w.token(t, w.trainer, []string{"trainer"}))

	for _, kind := range []string{"team_spieler", "team_eltern", "team_trainer"} {
		if !hasTarget(got, kind, &w.teamA) {
			t.Errorf("%s für das eigene Team fehlt, got %+v", kind, got)
		}
		if hasTarget(got, kind, &w.teamB) {
			t.Errorf("%s für ein fremdes Team wird angeboten, got %+v", kind, got)
		}
	}
	if !hasTarget(got, "alle_trainer", nil) {
		t.Errorf("alle_trainer fehlt, got %+v", got)
	}
	for _, kind := range []string{"users", "members", "spieler", "eltern"} {
		if hasTarget(got, kind, nil) {
			t.Errorf("vereinsweites Ziel %q wird einem Trainer angeboten, got %+v", kind, got)
		}
	}
}

// TC: Vorstand sieht die vereinsweiten Ziele UND jedes Team der aktiven Saison,
// ohne selbst irgendwo Trainer zu sein.
func TestBroadcastTargets_Vorstand(t *testing.T) {
	w := setupBroadcastWorld(t)
	vorstand := testutil.CreateUser(t, w.db, "standard")
	got := targetsFor(t, w.srv, w.token(t, vorstand, []string{"vorstand"}))

	for _, kind := range []string{"users", "members", "spieler", "eltern"} {
		if !hasTarget(got, kind, nil) {
			t.Errorf("vereinsweites Ziel %q fehlt, got %+v", kind, got)
		}
	}
	for _, team := range []int{w.teamA, w.teamB} {
		if !hasTarget(got, "team_spieler", &team) {
			t.Errorf("team_spieler für Team %d fehlt, got %+v", team, got)
		}
	}
}

// TC: Ohne Senderecht gibt es keine Liste — nicht etwa eine leere. Der Composer
// unterscheidet daran "darf nicht" von "hat gerade keine Gruppe".
func TestBroadcastTargets_OhneSenderecht(t *testing.T) {
	w := setupBroadcastWorld(t)
	res := testutil.Get(t, w.srv, "/api/chat/broadcast-targets", w.token(t, w.spielerA, []string{"spieler"}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

// TC: Die Zählung je Ziel ist die Gruppengröße inklusive Absender — sie
// beschreibt die Gruppe, nicht den Fan-out.
func TestBroadcastTargets_CountIstGruppengroesse(t *testing.T) {
	w := setupBroadcastWorld(t)
	got := targetsFor(t, w.srv, w.token(t, w.trainer, []string{"trainer"}))
	for _, a := range got {
		if a.Kind == "team_trainer" && a.TeamID != nil && *a.TeamID == w.teamA {
			if a.Count != 1 {
				t.Errorf("team_trainer count = %d, want 1 (der Absender selbst)", a.Count)
			}
		}
		if a.Kind == "team_spieler" && a.TeamID != nil && *a.TeamID == w.teamA {
			// regulärer Kader + erweiterter Kader + das Kind des Elternteils,
			// das keinen eigenen Account hat → 2 erreichbare User.
			if a.Count != 2 {
				t.Errorf("team_spieler count = %d, want 2", a.Count)
			}
		}
	}
}

// TC: Der Happy-Path. Spieler (regulär + erweitert) und Eltern des eigenen Teams
// bekommen je eine Zeile, Team B bleibt außen vor.
func TestSendBroadcast_TrainerAnEigenesTeam(t *testing.T) {
	w := setupBroadcastWorld(t)
	status, ids, body := w.send(t, w.token(t, w.trainer, []string{"trainer"}), []any{
		map[string]any{"kind": "team_spieler", "teamId": w.teamA},
		map[string]any{"kind": "team_eltern", "teamId": w.teamA},
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201", status)
	}

	want := []int{w.trainer, w.spielerA, w.erweitertA, w.elternA}
	slices.Sort(want)
	got := slices.Clone(ids)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("broadcast_reads = %v, want %v (Absenderzeile inklusive)", got, want)
	}
	if body.Recipients != 3 {
		t.Errorf("recipients = %d, want 3 (ohne den Absender)", body.Recipients)
	}
	for _, id := range ids {
		if id == w.spielerB {
			t.Error("ein Spieler des fremden Teams hat eine Zeile bekommen")
		}
	}
}

// TC: Der Trainer erreicht seine Kollegen über alle Kader hinweg.
func TestSendBroadcast_TrainerAnAlleTrainer(t *testing.T) {
	w := setupBroadcastWorld(t)
	status, ids, body := w.send(t, w.token(t, w.trainer, []string{"trainer"}), clubWide("alle_trainer"))
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201", status)
	}
	if !slices.Contains(ids, w.fremdTrainer) {
		t.Errorf("der Trainer von Team B fehlt in %v", ids)
	}
	if body.Recipients != 1 {
		t.Errorf("recipients = %d, want 1", body.Recipients)
	}
}

// TC: Ein fremdes Team ist kein Wirkungsbereich.
func TestSendBroadcast_TrainerFremdesTeam(t *testing.T) {
	w := setupBroadcastWorld(t)
	status, _, _ := w.send(t, w.token(t, w.trainer, []string{"trainer"}),
		teamTarget("team_spieler", w.teamB))
	if status != http.StatusForbidden {
		t.Errorf("status %d, want 403", status)
	}
	var stored int
	w.db.QueryRow(`SELECT COUNT(*) FROM broadcasts`).Scan(&stored)
	if stored != 0 {
		t.Errorf("%d Mitteilungen gespeichert, want 0", stored)
	}
}

// TC: Ein einziges unerlaubtes Ziel kippt den ganzen Request. Still nur das
// erlaubte zuzustellen sähe für den Absender aus wie eine vollständige
// Zustellung — genau der stumme Fehler, gegen den der recipients-Zähler
// existiert.
func TestSendBroadcast_EinUnerlaubtesZielKipptAlles(t *testing.T) {
	w := setupBroadcastWorld(t)
	status, _, _ := w.send(t, w.token(t, w.trainer, []string{"trainer"}), []any{
		map[string]any{"kind": "team_spieler", "teamId": w.teamA},
		map[string]any{"kind": "users"},
	})
	if status != http.StatusForbidden {
		t.Fatalf("status %d, want 403", status)
	}
	var reads int
	w.db.QueryRow(`SELECT COUNT(*) FROM broadcast_reads`).Scan(&reads)
	if reads != 0 {
		t.Errorf("%d broadcast_reads-Zeilen trotz 403 — Teilzustellung", reads)
	}
}

// TC: Der Trainer bleibt von den vereinsweiten Zielen ausgeschlossen.
func TestSendBroadcast_TrainerNichtVereinsweit(t *testing.T) {
	w := setupBroadcastWorld(t)
	for _, kind := range []string{"users", "members", "spieler", "eltern"} {
		status, _, _ := w.send(t, w.token(t, w.trainer, []string{"trainer"}), clubWide(kind))
		if status != http.StatusForbidden {
			t.Errorf("Ziel %q: status %d, want 403", kind, status)
		}
	}
}

// TC: Senderecht ist nicht Leserecht. Der Chat zeigt einem Elternteil die
// Standardgruppen des Kindes-Teams (user_accessible_teams) — senden darf er
// dorthin trotzdem nicht, auch nicht als Trainer eines anderen Teams.
func TestSendBroadcast_TrainerErbtKeinRechtAlsElternteil(t *testing.T) {
	w := setupBroadcastWorld(t)
	kindInB := testutil.CreateMember(t, w.db, 0)
	var kaderB int
	if err := w.db.QueryRow(
		`SELECT id FROM kader WHERE team_id = ?`, w.teamB).Scan(&kaderB); err != nil {
		t.Fatalf("kader B: %v", err)
	}
	addKaderMember(t, w.db, kaderB, kindInB)
	linkParent(t, w.db, w.trainer, kindInB)

	status, _, _ := w.send(t, w.token(t, w.trainer, []string{"trainer"}),
		teamTarget("team_spieler", w.teamB))
	if status != http.StatusForbidden {
		t.Errorf("status %d, want 403 (Elternschaft begründet kein Senderecht)", status)
	}
}

// TC: Wer über mehrere Ziele getroffen wird, bekommt genau eine Zeile — daran
// hängt, dass er nur einen Push und einen Ungelesen-Zähler sieht.
func TestSendBroadcast_VereinigungDedupliziert(t *testing.T) {
	w := setupBroadcastWorld(t)
	// Zweites Kind desselben Elternteils im selben Team.
	var kaderA int
	if err := w.db.QueryRow(`SELECT id FROM kader WHERE team_id = ?`, w.teamA).Scan(&kaderA); err != nil {
		t.Fatalf("kader A: %v", err)
	}
	zweitesKind := testutil.CreateMember(t, w.db, 0)
	addKaderMember(t, w.db, kaderA, zweitesKind)
	linkParent(t, w.db, w.elternA, zweitesKind)

	status, ids, body := w.send(t, w.token(t, w.trainer, []string{"trainer"}), []any{
		map[string]any{"kind": "team_spieler", "teamId": w.teamA},
		map[string]any{"kind": "team_eltern", "teamId": w.teamA},
		map[string]any{"kind": "team_trainer", "teamId": w.teamA},
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201", status)
	}
	seen := map[int]int{}
	for _, id := range ids {
		seen[id]++
	}
	if seen[w.elternA] != 1 {
		t.Errorf("Elternteil zweier Kinder hat %d Zeilen, want 1", seen[w.elternA])
	}
	if body.Recipients != len(ids)-1 {
		t.Errorf("recipients = %d, want %d (Zeilen ohne Absender)", body.Recipients, len(ids)-1)
	}
}

// TC: Die gewählten Ziele landen als Zeilen an der Mitteilung.
func TestSendBroadcast_ZieleWerdenGespeichert(t *testing.T) {
	w := setupBroadcastWorld(t)
	status, _, body := w.send(t, w.token(t, w.trainer, []string{"trainer"}), []any{
		map[string]any{"kind": "team_spieler", "teamId": w.teamA},
		map[string]any{"kind": "alle_trainer"},
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201", status)
	}
	rows, err := w.db.Query(
		`SELECT kind, team_id FROM broadcast_targets WHERE broadcast_id = ? ORDER BY kind`, body.ID)
	if err != nil {
		t.Fatalf("read broadcast_targets: %v", err)
	}
	defer rows.Close()
	type target struct {
		kind   string
		teamID sql.NullInt64
	}
	var got []target
	for rows.Next() {
		var tg target
		if err := rows.Scan(&tg.kind, &tg.teamID); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, tg)
	}
	if len(got) != 2 {
		t.Fatalf("%d Zielzeilen, want 2: %+v", len(got), got)
	}
	if got[0].kind != "alle_trainer" || got[0].teamID.Valid {
		t.Errorf("erstes Ziel = %+v, want alle_trainer ohne team_id", got[0])
	}
	if got[1].kind != "team_spieler" || int(got[1].teamID.Int64) != w.teamA {
		t.Errorf("zweites Ziel = %+v, want team_spieler für Team A", got[1])
	}
}
