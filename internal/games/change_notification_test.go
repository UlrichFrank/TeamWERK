package games_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── Änderungs-Benachrichtigung bei PUT /api/games/{id} ───────────────────────
//
// Invarianten (openspec/changes/aenderungs-benachrichtigung):
//   - keine Änderungsmeldung ohne Terminname + Zeitpunkt (Datum UND Uhrzeit) + Aktor,
//   - der alte Zeitpunkt steht genau dann im Text, wenn der Termin verschoben wurde,
//   - der Titel folgt dem event_type (generisch ≠ Spiel),
//   - der Aktor-Name fällt nie auf die E-Mail zurück,
//   - kein Erfolg, keine Meldung (403/404).
//
// Die Fixture stammt aus cancellation_test.go: Heimspiel gegen „HSG Ostfildern"
// am 14.09.2026 um 18:00, gelöscht/geändert von „Tim Meier".

// updateGameBody baut den Standard-Request-Body für UpdateGame. Datum und
// Uhrzeit sind die Stellschrauben der Tests.
func (f *cancelFixture) updateGameBody(date, clock string) map[string]any {
	return map[string]any{
		"date":       date,
		"time":       clock,
		"opponent":   "HSG Ostfildern",
		"team_ids":   []int{f.team},
		"event_type": "heim",
	}
}

func (f *cancelFixture) put(t *testing.T, token string, body any) *http.Response {
	t.Helper()
	return testutil.Do(t, f.srv, http.MethodPut, fmt.Sprintf("/api/games/%d", f.game), token, body)
}

// Der Kernfall: eine Änderung, die den Termin nicht verschiebt, benennt Gegner,
// Datum, Uhrzeit und Aktor — und nichts über einen früheren Zeitpunkt.
func TestUpdateGame_MeldungNenntTerminZeitpunktUndAktor(t *testing.T) {
	f := newCancelFixture(t)

	res := f.put(t, f.vorstandToken(t), f.updateGameBody("2026-09-14", "18:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spielinfo geändert")
	for _, want := range []string{"HSG Ostfildern", "14.09.2026", "18:00", "Tim Meier"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
	if strings.Contains(n.body, "vorher") {
		t.Errorf("ohne Verschiebung darf keine Vorher-Angabe im Body stehen, got %q", n.body)
	}
	if n.category != "games" {
		t.Errorf("Kategorie games erwartet, got %q", n.category)
	}
	if want := fmt.Sprintf("/termine?focus=game-%d", f.game); n.url != want {
		t.Errorf("url = %q, want %q", n.url, want)
	}
}

// Verschiebung: ohne den alten Zeitpunkt suchte der Empfänger im Kalender nach
// einem Termin, den es unter der alten Zeit nicht mehr gibt.
func TestUpdateGame_VerschiebungNenntAltenUndNeuenZeitpunkt(t *testing.T) {
	f := newCancelFixture(t)

	res := f.put(t, f.vorstandToken(t), f.updateGameBody("2026-09-21", "19:30"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spielinfo geändert")
	for _, want := range []string{"21.09.2026", "19:30", "vorher 14.09.2026, 18:00 Uhr"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
}

// Nur die Uhrzeit wandert — auch das ist eine Verschiebung.
func TestUpdateGame_NurUhrzeitGeaendertNenntVorher(t *testing.T) {
	f := newCancelFixture(t)

	res := f.put(t, f.vorstandToken(t), f.updateGameBody("2026-09-14", "19:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spielinfo geändert")
	if !strings.Contains(n.body, "vorher 14.09.2026, 18:00 Uhr") {
		t.Errorf("Body muss den alten Zeitpunkt nennen, got %q", n.body)
	}
}

// Ein generischer Termin ist kein Spiel — „Spielinfo geändert" benennt dort das
// falsche Ereignis. Dieselbe Unterscheidung trifft die Absage bereits.
func TestUpdateGame_GenerischesEventMeldetTerminGeaendert(t *testing.T) {
	f := newCancelFixture(t)
	f.setEventType(t, "generisch", "Ferientraining mB")

	body := f.updateGameBody("2026-09-14", "18:00")
	body["event_type"] = "generisch"
	body["opponent"] = "Ferientraining mB"
	res := f.put(t, f.vorstandToken(t), body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	if got := f.rec.byTitle("Spielinfo geändert"); len(got) != 0 {
		t.Errorf("generisches Event darf nicht als Spiel gemeldet werden, got %v", f.rec.titles())
	}
	n := f.rec.one(t, "Termin geändert")
	for _, want := range []string{"Ferientraining mB", "14.09.2026", "18:00", "Tim Meier"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
}

// Ohne hinterlegten Namen bleibt der Satz generisch — die E-Mail-Adresse geht
// niemals an ein ganzes Team.
func TestUpdateGame_AktorOhneNamenVerraetKeineEmail(t *testing.T) {
	f := newCancelFixture(t)
	namenlos := testutil.CreateUser(t, f.db, "standard")
	token := testutil.Token(t, namenlos, "standard", []string{"vorstand"})

	res := f.put(t, token, f.updateGameBody("2026-09-14", "18:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Spielinfo geändert")
	if strings.Contains(n.body, "@") {
		t.Errorf("Body enthält eine E-Mail-Adresse, got %q", n.body)
	}
	if !strings.Contains(n.body, "Geändert von") {
		t.Errorf("Body muss den Aktor-Satz behalten, got %q", n.body)
	}
}

// ── Anlage-Benachrichtigung bei POST /api/games ──────────────────────────────

// createGameBody baut den Standard-Request-Body für CreateGame.
func (f *cancelFixture) createGameBody(eventType, opponent string) map[string]any {
	return map[string]any{
		"date":       "2026-10-05",
		"time":       "17:15",
		"opponent":   opponent,
		"team_ids":   []int{f.team},
		"event_type": eventType,
		"season_id":  f.season,
	}
}

// Vorher stand hier das rohe ISO-Datum aus dem Request ("am 2026-10-05"), ohne
// Uhrzeit und ohne Aktor.
func TestCreateGame_MeldungNenntZeitpunktUndAktor(t *testing.T) {
	f := newCancelFixture(t)

	res := testutil.Do(t, f.srv, http.MethodPost, "/api/games",
		f.vorstandToken(t), f.createGameBody("heim", "TSV Neuhausen"))
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Neues Spiel")
	for _, want := range []string{"TSV Neuhausen", "05.10.2026", "17:15", "Tim Meier"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
	if strings.Contains(n.body, "2026-10-05") {
		t.Errorf("Body darf kein rohes ISO-Datum enthalten, got %q", n.body)
	}
	if n.category != "games" {
		t.Errorf("Kategorie games erwartet, got %q", n.category)
	}
}

// Ein neuer generischer Termin ist kein neues Spiel.
func TestCreateGame_GenerischesEventMeldetNeuerTermin(t *testing.T) {
	f := newCancelFixture(t)

	res := testutil.Do(t, f.srv, http.MethodPost, "/api/games",
		f.vorstandToken(t), f.createGameBody("generisch", "Ferientraining mB"))
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, got %d", res.StatusCode)
	}

	if got := f.rec.byTitle("Neues Spiel"); len(got) != 0 {
		t.Errorf("generischer Termin darf nicht als Spiel gemeldet werden, got %v", f.rec.titles())
	}
	n := f.rec.one(t, "Neuer Termin")
	for _, want := range []string{"Ferientraining mB", "05.10.2026", "17:15", "Tim Meier"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("Body muss %q enthalten, got %q", want, n.body)
		}
	}
}

func TestCreateGame_AktorOhneNamenVerraetKeineEmail(t *testing.T) {
	f := newCancelFixture(t)
	namenlos := testutil.CreateUser(t, f.db, "standard")
	token := testutil.Token(t, namenlos, "standard", []string{"vorstand"})

	res := testutil.Do(t, f.srv, http.MethodPost, "/api/games", token,
		f.createGameBody("heim", "TSV Neuhausen"))
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, got %d", res.StatusCode)
	}

	n := f.rec.one(t, "Neues Spiel")
	if strings.Contains(n.body, "@") {
		t.Errorf("Body enthält eine E-Mail-Adresse, got %q", n.body)
	}
}

// ── Fehlerfälle ──────────────────────────────────────────────────────────────

// Ohne erfolgreiche Änderung darf keine Meldung entstehen.
func TestUpdateGame_FremdesTeamKeineBenachrichtigung(t *testing.T) {
	f := newCancelFixture(t)
	otherTeam := testutil.CreateTeam(t, f.db, "Team B")
	trainerU := makeTrainer(t, f.db, otherTeam, f.season)
	token := testutil.Token(t, trainerU, "standard", []string{"trainer"})

	res := f.put(t, token, f.updateGameBody("2026-09-21", "19:30"))
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, got %d", res.StatusCode)
	}
	if titles := f.rec.titles(); len(titles) != 0 {
		t.Errorf("bei 403 darf nichts versendet werden, got %v", titles)
	}
}

func TestUpdateGame_UnbekannteIDKeineBenachrichtigung(t *testing.T) {
	f := newCancelFixture(t)

	res := testutil.Do(t, f.srv, http.MethodPut, "/api/games/999999",
		f.vorstandToken(t), f.updateGameBody("2026-09-21", "19:30"))
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("erwartet 404, got %d", res.StatusCode)
	}
	if titles := f.rec.titles(); len(titles) != 0 {
		t.Errorf("bei 404 darf nichts versendet werden, got %v", titles)
	}
}
