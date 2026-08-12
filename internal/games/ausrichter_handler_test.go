package games_test

// Tests für die Preview/Apply-Routen des Tages-Ausrichters
// (heimspieltag-ausrichter, design.md Decision 7). Nutzt die Helfer aus
// handler_test.go (insertDutyType, insertHeimTemplate, seedAgeClassRule,
// countRows) und bulkregen_handler_test.go (bulkFutureDate, bulkVorstandToken,
// dbFingerprint) — die Vorschau-Garantie ist hier dieselbe wie beim Massenlauf,
// also wird sie auch mit demselben Werkzeug bewiesen.

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// --- Helfer ---------------------------------------------------------------------------

type gameDayHostBalanceDTO struct {
	Created           int `json:"created"`
	Deleted           int `json:"deleted"`
	AssignmentsKept   int `json:"assignments_kept"`
	AssignmentsLost   int `json:"assignments_lost"`
	SlotsBefore       int `json:"slots_before"`
	SlotsAfter        int `json:"slots_after"`
	AssignmentsBefore int `json:"assignments_before"`
	AssignmentsAfter  int `json:"assignments_after"`
}

type gameDayHostDTO struct {
	Date           string                `json:"date"`
	AusrichterID   int                   `json:"ausrichter_id"`
	AusrichterName string                `json:"ausrichter_name"`
	IsExplicit     bool                  `json:"is_explicit"`
	Balance        gameDayHostBalanceDTO `json:"balance"`
	Applied        bool                  `json:"applied"`
}

func decodeGameDayHost(t *testing.T, res *http.Response) gameDayHostDTO {
	t.Helper()
	var got gameDayHostDTO
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode game-day host response: %v", err)
	}
	return got
}

// defaultAusrichterID liefert die von Migration 048 geseedete Default-Zeile —
// den Wert, auf den jeder Tag ohne expliziten Eintrag auflöst.
func defaultAusrichterID(t *testing.T, db *sql.DB) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM ausrichter WHERE is_default=1`).Scan(&id); err != nil {
		t.Fatalf("defaultAusrichterID: %v", err)
	}
	return id
}

// bindTemplateItemsToAusrichter bindet alle Items einer Vorlage an einen
// Ausrichter. Direktes UPDATE statt über die CRUD-Route: hier wird die
// Wirkung des Gates auf die Slots geprüft, nicht die Route-Validierung (die
// steht in ausrichter_template_test.go).
func bindTemplateItemsToAusrichter(t *testing.T, db *sql.DB, templateID, ausrichterID int) {
	t.Helper()
	res, err := db.Exec(`UPDATE game_template_items SET ausrichter_id=? WHERE template_id=?`,
		ausrichterID, templateID)
	if err != nil {
		t.Fatalf("bindTemplateItemsToAusrichter: %v", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		t.Fatalf("bindTemplateItemsToAusrichter: Vorlage %d hat keine Items", templateID)
	}
}

// hostFixture ist der gemeinsame Ausgangszustand: aktive Saison, ein Heimspiel
// in 10 Tagen mit Vorlage, dadurch genau ein automatisch erzeugter Dienst-Slot.
type hostFixture struct {
	db         *sql.DB
	seasonID   int
	teamID     int
	templateID int
	gameID     int
	date       string
	token      string
}

func newHostFixture(t *testing.T) hostFixture {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)
	date := bulkFutureDate(t, 10)

	res := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
		"date": date, "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim",
		"season_id": seasonID, "template_id": templateID,
	})
	res.Body.Close()

	var gameID int
	if err := db.QueryRow(`SELECT id FROM games WHERE season_id=?`, seasonID).Scan(&gameID); err != nil {
		t.Fatalf("load game id: %v", err)
	}
	if got := countRows(t, db, "duty_slots", "game_id=?", gameID); got != 1 {
		t.Fatalf("Fixture erwartet genau 1 Auto-Slot, hat %d", got)
	}
	return hostFixture{db: db, seasonID: seasonID, teamID: teamID, templateID: templateID,
		gameID: gameID, date: date, token: token}
}

// gebundenerTagMitZusage bringt die Fixture in den interessanten Zustand: das
// Vorlagen-Item ist an Ausrichter A gebunden, der Tag ist explizit auf A
// gesetzt (der Slot existiert also), und auf dem Slot liegt eine Zusage.
// Ein Wechsel weg von A gatet das Item aus und kostet die Zusage.
func gebundenerTagMitZusage(t *testing.T, f hostFixture, srv *httptest.Server) (ausrichterA, slotID, userID int) {
	t.Helper()
	ausrichterA = insertAusrichter(t, f.db, "TV Ausrichter A")
	bindTemplateItemsToAusrichter(t, f.db, f.templateID, ausrichterA)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": ausrichterA,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("apply(A) erwartet 200, got %d", res.StatusCode)
	}
	if err := f.db.QueryRow(`SELECT id FROM duty_slots WHERE game_id=?`, f.gameID).Scan(&slotID); err != nil {
		t.Fatalf("Slot nach apply(A) nicht vorhanden: %v", err)
	}
	userID = testutil.CreateUser(t, f.db, "standard")
	testutil.AssignDutySlot(t, f.db, slotID, userID)
	return ausrichterA, slotID, userID
}

// --- Vorschau schreibt nichts -----------------------------------------------------------

// TestPreviewGameDayHost_SchreibtNicht (spec: "Vorschau zeigt die Bilanz ohne zu
// schreiben"): die Vorschau läuft auf dem destruktivsten Fall — der Wechsel
// würde den einzigen Slot samt Zusage löschen — und darf danach keine einzige
// Zeile verändert haben, spieltag_ausrichter eingeschlossen.
func TestPreviewGameDayHost_SchreibtNicht(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	ausrichterA, _, _ := gebundenerTagMitZusage(t, f, srv)
	defaultID := defaultAusrichterID(t, f.db)

	before := dbFingerprint(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/preview", f.token, map[string]any{
		"date": f.date, "ausrichter_id": defaultID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeGameDayHost(t, res)
	if got.Balance.Deleted == 0 || got.Balance.AssignmentsLost != 1 || got.Balance.SlotsAfter != 0 {
		t.Errorf("Vorschau muss den Verlust ausweisen, bilanz=%+v", got.Balance)
	}

	if after := dbFingerprint(t, f.db); after != before {
		t.Errorf("preview darf die DB nicht verändern")
	}
	// Explizit statt nur über den Fingerprint: die drei Tabellen, um die es geht.
	if n := countRows(t, f.db, "duty_slots", "game_id=?", f.gameID); n != 1 {
		t.Errorf("duty_slots nach preview: erwartet 1, got %d", n)
	}
	if n := countRows(t, f.db, "duty_assignments", "1=1"); n != 1 {
		t.Errorf("duty_assignments nach preview: erwartet 1, got %d", n)
	}
	if n := countRows(t, f.db, "spieltag_ausrichter", "ausrichter_id=?", ausrichterA); n != 1 {
		t.Errorf("spieltag_ausrichter muss unverändert auf A stehen, got %d Zeilen", n)
	}

	// Poison-Sanity: der Vergleicher muss eine echte Änderung auch erkennen.
	f.db.Exec(`UPDATE games SET opponent='POISON' WHERE id=?`, f.gameID)
	if dbFingerprint(t, f.db) == before {
		t.Fatalf("dbFingerprint poison-sanity: Vergleicher erkennt echte Änderung nicht")
	}
}

// --- Apply ------------------------------------------------------------------------------

// TestApplyGameDayHost_SetztWertUndBroadcastet (spec: "Anwenden setzt den Wert
// und regeneriert den Tag"): der Wert steht danach in spieltag_ausrichter, der
// Regen ist gelaufen, und beide Live-Update-Kanäle wurden bedient.
func TestApplyGameDayHost_SetztWertUndBroadcastet(t *testing.T) {
	f := newHostFixture(t)
	srv, sharedHub := prodserver.NewWithHub(t, f.db)
	ausrichterA := insertAusrichter(t, f.db, "TV Ausrichter A")

	events := sharedHub.SubscribeUser(999101)
	defer sharedHub.UnsubscribeUser(999101, events)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": ausrichterA,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeGameDayHost(t, res)
	if !got.Applied || got.AusrichterID != ausrichterA || !got.IsExplicit {
		t.Errorf("Response erwartet applied/explizit auf A, got %+v", got)
	}

	var stored int
	if err := f.db.QueryRow(
		`SELECT ausrichter_id FROM spieltag_ausrichter WHERE date=? AND season_id=?`,
		f.date, f.seasonID).Scan(&stored); err != nil {
		t.Fatalf("spieltag_ausrichter nach apply: %v", err)
	}
	if stored != ausrichterA {
		t.Errorf("erwartet gespeicherten Ausrichter %d, got %d", ausrichterA, stored)
	}
	// Das Item ist ungebunden → der Regen erzeugt den Slot unverändert neu.
	if n := countRows(t, f.db, "duty_slots", "game_id=?", f.gameID); n != 1 {
		t.Errorf("Slot nach apply: erwartet 1, got %d", n)
	}

	seen := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(seen) < 2 {
		select {
		case ev := <-events:
			seen[ev] = true
		case <-deadline:
			t.Fatalf("erwartet Broadcasts 'duties' und 'games', gesehen: %v", seen)
		}
	}
	if !seen["duties"] || !seen["games"] {
		t.Errorf("erwartet Broadcasts 'duties' und 'games', gesehen: %v", seen)
	}
}

// TestApplyGameDayHost_AusgegatetesItemLoeschtSlotUndZusage (spec-Requirement
// "Ausrichter-Änderungen laufen über eine schreibfreie Vorschau" in seiner
// destruktiven Ausprägung): weg von A heißt, das gebundene Item erzeugt keine
// Slots mehr — der Slot verschwindet samt Zusage, und die Bilanz sagt das auch.
func TestApplyGameDayHost_AusgegatetesItemLoeschtSlotUndZusage(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	_, slotID, userID := gebundenerTagMitZusage(t, f, srv)
	defaultID := defaultAusrichterID(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": defaultID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeGameDayHost(t, res)
	if got.Balance.AssignmentsLost != 1 || got.Balance.AssignmentsKept != 0 {
		t.Errorf("Bilanz muss die verlorene Zusage ausweisen, got %+v", got.Balance)
	}
	if got.Balance.SlotsBefore != 1 || got.Balance.SlotsAfter != 0 {
		t.Errorf("Bilanz muss den Slot-Verlust ausweisen, got %+v", got.Balance)
	}
	if n := countRows(t, f.db, "duty_slots", "id=?", slotID); n != 0 {
		t.Errorf("ausgegateter Slot muss gelöscht sein")
	}
	if n := countRows(t, f.db, "duty_assignments", "user_id=?", userID); n != 0 {
		t.Errorf("Zusage des gelöschten Slots muss verschwunden sein")
	}
}

// TestGameDayHost_PreviewUndApplyLiefernDieselbeBilanz: derselbe Request, erst
// preview, dann apply — identische Zahlen. Das ist die eigentliche Zusage der
// Vorschau: sie ist kein Nachbau, sondern derselbe Lauf ohne Commit.
func TestGameDayHost_PreviewUndApplyLiefernDieselbeBilanz(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	gebundenerTagMitZusage(t, f, srv)
	defaultID := defaultAusrichterID(t, f.db)
	body := map[string]any{"date": f.date, "ausrichter_id": defaultID}

	previewRes := testutil.Post(t, srv, "/api/game-days/host/preview", f.token, body)
	if previewRes.StatusCode != http.StatusOK {
		previewRes.Body.Close()
		t.Fatalf("preview: expected 200, got %d", previewRes.StatusCode)
	}
	preview := decodeGameDayHost(t, previewRes)
	previewRes.Body.Close()

	applyRes := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, body)
	if applyRes.StatusCode != http.StatusOK {
		applyRes.Body.Close()
		t.Fatalf("apply: expected 200, got %d", applyRes.StatusCode)
	}
	applied := decodeGameDayHost(t, applyRes)
	applyRes.Body.Close()

	if preview.Balance != applied.Balance {
		t.Errorf("Vorschau und Anwenden müssen dieselbe Bilanz liefern:\npreview=%+v\napply  =%+v",
			preview.Balance, applied.Balance)
	}
	if preview.AusrichterID != applied.AusrichterID || preview.IsExplicit != applied.IsExplicit {
		t.Errorf("aufgelöster Zustand weicht ab: preview=%+v apply=%+v", preview, applied)
	}
}

// TestApplyGameDayHost_DatumWirdNormalisiert hält den SQLite-DATE-Gotcha auf der
// Schreibseite fest: ein ISO-Timestamp im Request muss als reines
// "2006-01-02" in der Spalte landen. Sonst vergliche
// ResolveAusrichterForDay (normalisiert nur den Parameter, nicht den
// Spaltenwert) nie erfolgreich — der Wechsel wäre gespeichert und trotzdem
// wirkungslos, ohne Fehlermeldung. Deshalb wird nicht nur die Spalte geprüft,
// sondern der Wert über die Lese-Route auch wiedergefunden.
func TestApplyGameDayHost_DatumWirdNormalisiert(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	ausrichterA := insertAusrichter(t, f.db, "TV Ausrichter A")

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date + "T00:00:00Z", "ausrichter_id": ausrichterA,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	// Nicht die Spalte in einen String scannen: der Treiber formatiert eine als
	// DATE deklarierte Spalte beim Lesen ohnehin zum ISO-Timestamp um, egal was
	// gespeichert ist — genau der Grund, warum dieser Bug so leicht zu übersehen
	// ist. Geprüft wird deshalb in SQL: Länge 10 und exakter Stringvergleich auf
	// die reine Form, also derselbe Vergleich, den ResolveAusrichterForDay führt.
	var storedLen, exactMatches int
	if err := f.db.QueryRow(
		`SELECT LENGTH(date), (SELECT COUNT(*) FROM spieltag_ausrichter WHERE date=?)
		 FROM spieltag_ausrichter WHERE season_id=?`, f.date, f.seasonID).
		Scan(&storedLen, &exactMatches); err != nil {
		t.Fatalf("spieltag_ausrichter: %v", err)
	}
	if storedLen != 10 || exactMatches != 1 {
		t.Errorf("erwartet gespeichertes Datum in reiner Form %q (LENGTH=10, exakter Treffer), got LENGTH=%d, Treffer=%d",
			f.date, storedLen, exactMatches)
	}

	getRes := testutil.Get(t, srv, "/api/game-days/"+f.date+"/host", f.token)
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("GET host: expected 200, got %d", getRes.StatusCode)
	}
	got := decodeGameDayHost(t, getRes)
	if got.AusrichterID != ausrichterA || !got.IsExplicit {
		t.Errorf("Auflösung findet den geschriebenen Tageswert nicht wieder: %+v", got)
	}
}

// --- Lese-Route -------------------------------------------------------------------------

// TestGetGameDayHost_GeerbtDannExplizit (spec: "Tag ohne Eintrag fällt auf den
// Default"): vor dem Setzen ist der Wert der Default und is_explicit=false,
// danach der gesetzte Wert und is_explicit=true.
func TestGetGameDayHost_GeerbtDannExplizit(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	defaultID := defaultAusrichterID(t, f.db)

	first := testutil.Get(t, srv, "/api/game-days/"+f.date+"/host", f.token)
	if first.StatusCode != http.StatusOK {
		first.Body.Close()
		t.Fatalf("expected 200, got %d", first.StatusCode)
	}
	geerbt := decodeGameDayHost(t, first)
	first.Body.Close()
	if geerbt.AusrichterID != defaultID || geerbt.IsExplicit {
		t.Errorf("ohne Eintrag erwartet Default %d mit is_explicit=false, got %+v", defaultID, geerbt)
	}
	if geerbt.AusrichterName == "" {
		t.Errorf("Antwort sollte den Namen des aufgelösten Ausrichters tragen")
	}

	ausrichterA := insertAusrichter(t, f.db, "TV Ausrichter A")
	applyRes := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": ausrichterA,
	})
	applyRes.Body.Close()

	second := testutil.Get(t, srv, "/api/game-days/"+f.date+"/host", f.token)
	defer second.Body.Close()
	explizit := decodeGameDayHost(t, second)
	if explizit.AusrichterID != ausrichterA || !explizit.IsExplicit {
		t.Errorf("nach dem Setzen erwartet %d mit is_explicit=true, got %+v", ausrichterA, explizit)
	}
}

// --- Fehlerfälle ------------------------------------------------------------------------

// TestApplyGameDayHost_UnbekannterAusrichter_400 (spec: "Unbekannter Ausrichter
// wird abgelehnt").
func TestApplyGameDayHost_UnbekannterAusrichter_400(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	before := dbFingerprint(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": 999999,
	})
	defer res.Body.Close()
	assertHostError(t, res, http.StatusBadRequest, "unknown_ausrichter")
	if dbFingerprint(t, f.db) != before {
		t.Errorf("ein abgelehnter Request darf nichts schreiben")
	}
}

// TestApplyGameDayHost_InaktiverAusrichter_400: ein deaktivierter Ausrichter ist
// kein gültiges Ziel — die Liste bietet ihn Standard-Nutzern gar nicht erst an.
func TestApplyGameDayHost_InaktiverAusrichter_400(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	inaktiv := insertAusrichter(t, f.db, "TV Ausgemustert")
	if _, err := f.db.Exec(`UPDATE ausrichter SET aktiv=0 WHERE id=?`, inaktiv); err != nil {
		t.Fatalf("deaktivieren: %v", err)
	}
	before := dbFingerprint(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": inaktiv,
	})
	defer res.Body.Close()
	assertHostError(t, res, http.StatusBadRequest, "inactive_ausrichter")
	if dbFingerprint(t, f.db) != before {
		t.Errorf("ein abgelehnter Request darf nichts schreiben")
	}
}

// TestApplyGameDayHost_KeineAktiveSaison_400: ohne aktive Saison gibt es keinen
// Schlüssel (date, season_id), auf den der Wert gehörte.
func TestApplyGameDayHost_KeineAktiveSaison_400(t *testing.T) {
	f := newHostFixture(t)
	if _, err := f.db.Exec(`UPDATE seasons SET is_active=0`); err != nil {
		t.Fatalf("Saison deaktivieren: %v", err)
	}
	srv := prodserver.New(t, f.db)
	before := dbFingerprint(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/apply", f.token, map[string]any{
		"date": f.date, "ausrichter_id": defaultAusrichterID(t, f.db),
	})
	defer res.Body.Close()
	assertHostError(t, res, http.StatusBadRequest, "no_active_season")
	if dbFingerprint(t, f.db) != before {
		t.Errorf("ein abgelehnter Request darf nichts schreiben")
	}
}

// TestPreviewGameDayHost_UngueltigesDatum_400: das Datum wird vor dem ersten
// Schreibvorgang geparst, nicht erst von SQLite interpretiert.
func TestPreviewGameDayHost_UngueltigesDatum_400(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Post(t, srv, "/api/game-days/host/preview", f.token, map[string]any{
		"date": "14.09.2026", "ausrichter_id": defaultAusrichterID(t, f.db),
	})
	defer res.Body.Close()
	assertHostError(t, res, http.StatusBadRequest, "invalid_date")
}

// TestGameDayHost_OhneManageGames_403 (spec: "Nutzer ohne manage_games wird
// abgewiesen"): beide Schreibrouten sind gesperrt, die Lese-Route bleibt offen.
func TestGameDayHost_OhneManageGames_403(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	uid := testutil.CreateUser(t, f.db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"spieler"})
	before := dbFingerprint(t, f.db)

	for _, path := range []string{"/api/game-days/host/preview", "/api/game-days/host/apply"} {
		res := testutil.Post(t, srv, path, token, map[string]any{
			"date": f.date, "ausrichter_id": defaultAusrichterID(t, f.db),
		})
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Errorf("%s: expected 403, got %d", path, res.StatusCode)
		}
	}
	if dbFingerprint(t, f.db) != before {
		t.Errorf("ein abgewiesener Request darf nichts schreiben")
	}

	readRes := testutil.Get(t, srv, "/api/game-days/"+f.date+"/host", token)
	defer readRes.Body.Close()
	if readRes.StatusCode != http.StatusOK {
		t.Errorf("Lesen muss für jeden Eingeloggten erlaubt bleiben, got %d", readRes.StatusCode)
	}
}

func assertHostError(t *testing.T, res *http.Response, wantStatus int, wantCode string) {
	t.Helper()
	if res.StatusCode != wantStatus {
		t.Fatalf("expected %d, got %d", wantStatus, res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Error != wantCode {
		t.Errorf("expected error %q, got %q", wantCode, body.Error)
	}
}
