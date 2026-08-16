package trainings_test

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// ── Absage-Benachrichtigungen mit Inhalt (openspec absage-benachrichtigung, Abschnitte 4+5) ──

// sentTraining hält einen notify.Send-Aufruf fest. Im Gegensatz zu
// captureNotifyCategory (notify_category_test.go) brauchen die Tests hier auch
// Titel, Body und URL für die Textprüfungen.
type sentTraining struct {
	category string
	title    string
	body     string
	url      string
}

// captureTrainingNotifications überschreibt den notify.Send-Seam für die Dauer
// des Tests und liefert einen gepufferten Kanal aller Aufrufe.
func captureTrainingNotifications(t *testing.T) chan sentTraining {
	t.Helper()
	ch := make(chan sentTraining, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, _ []int, category, title, body, url string, _ ...notify.Option) {
		ch <- sentTraining{category: category, title: title, body: body, url: url}
	}
	t.Cleanup(func() { notify.Send = orig })
	return ch
}

// drainTraining liest alle bereits gesendeten Benachrichtigungen ab. Die
// Handler rufen notify.Send synchron vor dem Response auf — was nach der
// Antwort im Kanal liegt, ist also vollständig.
func drainTraining(ch chan sentTraining) []sentTraining {
	var out []sentTraining
	for {
		select {
		case n := <-ch:
			out = append(out, n)
		default:
			return out
		}
	}
}

// setUserName hinterlegt Vor- und Nachname, damit notify.ActorName einen Namen
// findet (CreateUser legt nur E-Mail + Rolle an).
func setUserName(t *testing.T, db *sql.DB, userID int, first, last string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE users SET first_name=?, last_name=? WHERE id=?`, first, last, userID); err != nil {
		t.Fatalf("setUserName: %v", err)
	}
}

// mustContainAll prüft, dass s alle needles enthält.
func mustContainAll(t *testing.T, s string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(s, n) {
			t.Errorf("Body %q enthält %q nicht", s, n)
		}
	}
}

// makeTrainer legt einen Nutzer mit Vereinsfunktion 'trainer' an und hängt ihn
// als Trainer an den Kader des Teams — hasTeamAccess verlangt beides.
func makeTrainer(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	testutil.AddClubFunction(t, db, mid, "trainer")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, mid)
	return uid
}

// ── DELETE /api/training-sessions/{id} (Task 4.1) ──

// TestDeleteSession_AbsageNenntTitelDatumAktorUndGrund prüft den neuen Text:
// Titel der Einheit, Datum als TT.MM.JJJJ, Name des Auslösers und der
// mitgeschickte Grund — statt des alten Platzhalters. Die url ist leer, weil
// die Einheit nicht mehr existiert.
func TestDeleteSession_AbsageNenntTitelDatumAktorUndGrund(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	if _, err := db.Exec(`UPDATE training_sessions SET title='Krafteinheit' WHERE id=?`, sessionID); err != nil {
		t.Fatalf("title setzen: %v", err)
	}
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Training abgesagt" {
		t.Errorf("title = %q, want %q", got[0].title, "Training abgesagt")
	}
	if got[0].url != "" {
		t.Errorf("url = %q, want leer (gelöschte Einheit hat kein Ziel)", got[0].url)
	}
	if got[0].category != "trainings" {
		t.Errorf("category = %q, want trainings", got[0].category)
	}
	mustContainAll(t, got[0].body, "Krafteinheit", "15.03.2026", "Tim Meier", "Halle gesperrt")
}

// TestDeleteSession_OhneTitel_FaelltAufTrainingZurueck — title ist DEFAULT ”,
// der Satz braucht trotzdem ein Substantiv.
func TestDeleteSession_OhneTitel_FaelltAufTrainingZurueck(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	if _, err := db.Exec(`UPDATE training_sessions SET title='' WHERE id=?`, sessionID); err != nil {
		t.Fatalf("title leeren: %v", err)
	}
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d", len(got))
	}
	mustContainAll(t, got[0].body, "Training", "15.03.2026")
}

// TestDeleteSession_VorstandSilent_KeineBenachrichtigungAberBroadcast prüft
// design.md §8: silent unterdrückt nur notify.Send, nie das SSE-Live-Update.
// Der Routen-Gate verlangt trainer/sportliche_leitung, ein reiner Vorstand
// erreicht die Route also gar nicht — deshalb trainer+vorstand.
func TestDeleteSession_VorstandSilent_KeineBenachrichtigungAberBroadcast(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")

	vorstandID := makeTrainer(t, db, teamID, seasonID)
	var memberID int
	if err := db.QueryRow(`SELECT id FROM members WHERE user_id=?`, vorstandID).Scan(&memberID); err != nil {
		t.Fatalf("member lesen: %v", err)
	}
	testutil.AddClubFunction(t, db, memberID, "vorstand")

	eh := hub.NewHub()
	h := trainings.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)
	// Vorstand steht über collectByFunctions in der Team-Audience.
	ch := eh.SubscribeUser(vorstandID)
	defer eh.UnsubscribeUser(vorstandID, ch)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, vorstandID, "standard", []string{"trainer", "vorstand"}),
		map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
	select {
	case ev := <-ch:
		if ev != "trainings" {
			t.Errorf("Broadcast %q, want trainings", ev)
		}
	default:
		t.Error("kein Broadcast trotz silent — Live-Updates dürfen nicht unterdrückt werden")
	}
}

// TestDeleteSession_TrainerSilent_BenachrichtigungGehtTrotzdemRaus prüft die
// Fail-safe-Regel aus design.md §4: ohne Capability wird silent ignoriert,
// nicht mit 403 quittiert.
func TestDeleteSession_TrainerSilent_BenachrichtigungGehtTrotzdemRaus(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	trainerID := makeTrainer(t, db, teamID, seasonID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, trainerID, "standard", []string{"trainer"}), map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete 1 Benachrichtigung (Trainer darf nicht stummschalten), bekam %d", len(got))
	}
	if got[0].title != "Training abgesagt" {
		t.Errorf("title = %q, want %q", got[0].title, "Training abgesagt")
	}
}

// TestDeleteSession_LeererBody_BleibtErfolgreich — alte PWA-Installationen
// senden keinen Body (design.md §5).
func TestDeleteSession_LeererBody_BleibtErfolgreich(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE id=?`, sessionID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("Einheit nicht gelöscht (%d Zeilen)", count)
	}
}

// TestDeleteSession_UnbekannteID_404OhneBenachrichtigung
func TestDeleteSession_UnbekannteID_404OhneBenachrichtigung(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/999999",
		testutil.Token(t, adminID, "admin", nil), map[string]any{"reason": "egal"})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", res.StatusCode)
	}
	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
}

// ── DELETE /api/training-series/{id} (Task 4.2) ──

// TestDeleteSeries_AbsageNenntNamenZeitraumAktorUndGrund prüft den neuen Titel
// „Trainingsserie beendet" und den Body mit Serienname, Zeitraum (TT.MM.JJJJ),
// Auslöser und Grund; die url ist leer.
func TestDeleteSeries_AbsageNenntNamenZeitraumAktorUndGrund(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminID)
	if _, err := db.Exec(`UPDATE training_series SET name='Montagstraining' WHERE id=?`, seriesID); err != nil {
		t.Fatalf("name setzen: %v", err)
	}

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=all",
		testutil.Token(t, adminID, "admin", nil), map[string]any{"reason": "Halle gesperrt"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Trainingsserie beendet" {
		t.Errorf("title = %q, want %q", got[0].title, "Trainingsserie beendet")
	}
	if got[0].url != "" {
		t.Errorf("url = %q, want leer (gelöschte Serie hat kein Ziel)", got[0].url)
	}
	// scope=all ⇒ Zeitraum der gesamten Serie (CreateTrainingSeries-Fixture).
	mustContainAll(t, got[0].body, "Montagstraining", "01.10.2025", "30.06.2026", "Tim Meier", "Halle gesperrt")
}

// TestDeleteSeries_ScopeThisAndFollowing_NenntDenGeloeschtenBereich — die
// Zeitangabe folgt dem tatsächlich gelöschten Bereich, nicht valid_from.
func TestDeleteSeries_ScopeThisAndFollowing_NenntDenGeloeschtenBereich(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminID := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=this_and_following&from=2026-01-12",
		testutil.Token(t, adminID, "admin", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d", len(got))
	}
	mustContainAll(t, got[0].body, "12.01.2026")
	if strings.Contains(got[0].body, "01.10.2025") {
		t.Errorf("Body %q nennt valid_from statt des gelöschten Bereichs", got[0].body)
	}
}

// TestDeleteSeries_VorstandSilent_KeineBenachrichtigungAberBroadcast
func TestDeleteSeries_VorstandSilent_KeineBenachrichtigungAberBroadcast(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	vorstandID := makeTrainer(t, db, teamID, seasonID)
	var memberID int
	if err := db.QueryRow(`SELECT id FROM members WHERE user_id=?`, vorstandID).Scan(&memberID); err != nil {
		t.Fatalf("member lesen: %v", err)
	}
	testutil.AddClubFunction(t, db, memberID, "vorstand")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, vorstandID)

	eh := hub.NewHub()
	h := trainings.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)
	ch := eh.SubscribeUser(vorstandID)
	defer eh.UnsubscribeUser(vorstandID, ch)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=all",
		testutil.Token(t, vorstandID, "standard", []string{"trainer", "vorstand"}),
		map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
	select {
	case ev := <-ch:
		if ev != "trainings" {
			t.Errorf("Broadcast %q, want trainings", ev)
		}
	default:
		t.Error("kein Broadcast trotz silent — Live-Updates dürfen nicht unterdrückt werden")
	}
}

// TestDeleteSeries_TrainerSilent_BenachrichtigungGehtTrotzdemRaus
func TestDeleteSeries_TrainerSilent_BenachrichtigungGehtTrotzdemRaus(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerID := makeTrainer(t, db, teamID, seasonID)
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, trainerID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=all",
		testutil.Token(t, trainerID, "standard", []string{"trainer"}), map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete 1 Benachrichtigung (Trainer darf nicht stummschalten), bekam %d", len(got))
	}
	if got[0].title != "Trainingsserie beendet" {
		t.Errorf("title = %q, want %q", got[0].title, "Trainingsserie beendet")
	}
}

// TestDeleteSeries_LeererBody_BleibtErfolgreich
func TestDeleteSeries_LeererBody_BleibtErfolgreich(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminID := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=all",
		testutil.Token(t, adminID, "admin", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_series WHERE id=?`, seriesID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("Serie nicht gelöscht (%d Zeilen)", count)
	}
}

// TestDeleteSeries_UnbekannteID_404OhneBenachrichtigung
func TestDeleteSeries_UnbekannteID_404OhneBenachrichtigung(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-series/999999?scope=all",
		testutil.Token(t, adminID, "admin", nil), map[string]any{"reason": "egal"})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", res.StatusCode)
	}
	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
}

// ── PUT /api/training-sessions/{id}: Absage über Statuswechsel (Task 5) ──

// updateSessionBody baut den Standard-Request-Body für UpdateSession.
func updateSessionBody(status, cancelReason string) map[string]any {
	return map[string]any{
		"title":         "Krafteinheit",
		"date":          "2026-03-15",
		"start_time":    "18:00",
		"end_time":      "19:30",
		"status":        status,
		"cancel_reason": cancelReason,
	}
}

// TestUpdateSession_ActiveZuCancelled_SendetAbsage prüft den Kern von
// Abschnitt 5: der Statuswechsel nach 'cancelled' sendet „Training abgesagt"
// samt erfasstem Grund — der Link bleibt aber bestehen, weil die Einheit
// weiterlebt und dort ihren Absagegrund anzeigt.
func TestUpdateSession_ActiveZuCancelled_SendetAbsage(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), updateSessionBody("cancelled", "Halle gesperrt"))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Training abgesagt" {
		t.Errorf("title = %q, want %q", got[0].title, "Training abgesagt")
	}
	want := "/termine?focus=training-" + strconv.Itoa(sessionID)
	if got[0].url != want {
		t.Errorf("url = %q, want %q", got[0].url, want)
	}
	mustContainAll(t, got[0].body, "Krafteinheit", "15.03.2026", "Tim Meier", "Halle gesperrt")
}

// TestUpdateSession_CancelledZuCancelled_SendetNurAenderung — ohne den
// Vorher-Vergleich würde jede Korrektur an einer bereits abgesagten Einheit
// erneut „Training abgesagt" ans ganze Team schicken.
func TestUpdateSession_CancelledZuCancelled_SendetNurAenderung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	if _, err := db.Exec(
		`UPDATE training_sessions SET status='cancelled', cancel_reason='Halle gesperrt' WHERE id=?`,
		sessionID); err != nil {
		t.Fatalf("status setzen: %v", err)
	}
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), updateSessionBody("cancelled", "Halle weiterhin gesperrt"))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Training geändert" {
		t.Errorf("title = %q, want %q (keine zweite Absage-Meldung)", got[0].title, "Training geändert")
	}
}

// TestUpdateSession_CancelledZuActive_SendetAenderung — eine zurückgenommene
// Absage ist keine Absage.
func TestUpdateSession_CancelledZuActive_SendetAenderung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	if _, err := db.Exec(`UPDATE training_sessions SET status='cancelled' WHERE id=?`, sessionID); err != nil {
		t.Fatalf("status setzen: %v", err)
	}
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), updateSessionBody("active", ""))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d", len(got))
	}
	if got[0].title != "Training geändert" {
		t.Errorf("title = %q, want %q", got[0].title, "Training geändert")
	}
}

// TestUpdateSession_UngueltigerStatus_400 — der Status-CHECK bleibt unberührt.
func TestUpdateSession_UngueltigerStatus_400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	adminID := testutil.CreateUser(t, db, "admin")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), updateSessionBody("abgesagt", ""))
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
}

// TestDeleteSession_ReinerVorstand_403 nagelt die Reichweitengrenze der
// Stummschaltung fest: der Trainings-Tier im Router ist
// RequireClubFunction("trainer", "sportliche_leitung") — ein Nutzer mit
// ausschließlich der Vereinsfunktion `vorstand` wird abgewiesen, bevor der
// Handler läuft. Die Capability `suppress_event_notification` nützt ihm hier
// also nichts; bei Spielen und Dienst-Slots (weiterer Tier) sehr wohl.
//
// Der `vorstand`-Zweig in hasTeamAccess ist für diese Routen damit toter Code.
// Bewusst nicht geheilt — den Tier zu öffnen wäre eine Rechteausweitung, die
// mit Benachrichtigungstexten nichts zu tun hat (design.md §4a).
func TestDeleteSession_ReinerVorstand_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	vorstandID := testutil.CreateUser(t, db, "standard")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodDelete, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}), map[string]any{"silent": true})
	res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want 403 — reiner Vorstand erreicht den Trainings-Tier nicht", res.StatusCode)
	}
	if got := drainTraining(sent); len(got) != 0 {
		t.Errorf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE id=?`, sessionID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("die Einheit wurde trotz 403 gelöscht")
	}
}
