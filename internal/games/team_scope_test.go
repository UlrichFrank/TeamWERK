package games_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// ── Team-Scope für Event-Mutationen (Capability game-mutation-team-scope) ─────
//
// Regel: ein reiner Trainer (Vereinsfunktion `trainer` ohne sportliche_leitung/
// vorstand, keine Rolle admin) darf
//   - bei heim/auswärts nur eigene Mannschaften setzen,
//   - bei generisch beliebige Mannschaften setzen, solange EINE eigene dabei ist,
//   - ein Event nur mutieren/löschen, wenn eine eigene Mannschaft beteiligt ist.

// scopeFixture baut Saison + zwei Teams + einen Trainer von Team A. Team B hat
// einen eigenen Kader, aber keinen gemeinsamen Trainer — es ist die "fremde"
// Mannschaft.
type scopeFixture struct {
	db        *sql.DB
	season    int
	teamA     int
	teamB     int
	trainerID int
}

func newScopeFixture(t *testing.T) scopeFixture {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	testutil.CreateKader(t, db, teamB, season)
	trainerID := makeTrainer(t, db, teamA, season) // legt Kader A mit an
	return scopeFixture{db: db, season: season, teamA: teamA, teamB: teamB, trainerID: trainerID}
}

func (f scopeFixture) trainerToken(t *testing.T) string {
	t.Helper()
	return testutil.Token(t, f.trainerID, "standard", []string{"trainer"})
}

// gameTeamIDs liest die aktuell verknüpften Team-IDs eines Events.
func gameTeamIDs(t *testing.T, db *sql.DB, gameID int) []int {
	t.Helper()
	rows, err := db.Query(`SELECT team_id FROM game_teams WHERE game_id=? ORDER BY team_id`, gameID)
	if err != nil {
		t.Fatalf("gameTeamIDs: %v", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("gameTeamIDs scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func countGames(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM games`).Scan(&n); err != nil {
		t.Fatalf("countGames: %v", err)
	}
	return n
}

// createGenericEvent legt direkt in der DB ein generisches Event für die
// übergebenen Teams an (umgeht die Route, damit der Ausgangszustand unabhängig
// von der zu testenden Autorisierung ist).
func createGenericEvent(t *testing.T, db *sql.DB, seasonID int, teamIDs ...int) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?,?,?,?,'generisch',0)`,
		seasonID, "Vereinsfest", "2026-06-20", "14:00")
	if err != nil {
		t.Fatalf("createGenericEvent: %v", err)
	}
	id, _ := res.LastInsertId()
	for _, tid := range teamIDs {
		if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, id, tid); err != nil {
			t.Fatalf("createGenericEvent game_teams: %v", err)
		}
	}
	return int(id)
}

// ── POST /api/games ──────────────────────────────────────────────────────────

// Happy-Path des Features: Trainer lädt fremde Mannschaften zu seinem
// generischen Event ein.
func TestCreateGame_TrainerGenericForeignTeamsAllowed(t *testing.T) {
	f := newScopeFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Post(t, srv, "/api/games", f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "14:00",
		"opponent":   "Vereinsfest",
		"event_type": "generisch",
		"team_ids":   []int{f.teamA, f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var gameID int
	if err := f.db.QueryRow(`SELECT id FROM games WHERE event_type='generisch'`).Scan(&gameID); err != nil {
		t.Fatalf("event not created: %v", err)
	}
	if got := gameTeamIDs(t, f.db, gameID); len(got) != 2 {
		t.Errorf("expected both teams linked, got %v", got)
	}
}

// Ohne eigene Mannschaft entstünde ein Event, das der Trainer selbst nicht
// sieht (ScopeGamesQuery) — deshalb 403.
func TestCreateGame_TrainerGenericWithoutOwnTeam(t *testing.T) {
	f := newScopeFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Post(t, srv, "/api/games", f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "14:00",
		"opponent":   "Fremdes Fest",
		"event_type": "generisch",
		"team_ids":   []int{f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if n := countGames(t, f.db); n != 0 {
		t.Errorf("no event must be created, got %d", n)
	}
}

// Bestandsverhalten: Heim-/Auswärtsspiele bleiben strikt auf eigene Teams.
func TestCreateGame_TrainerHomeGameForeignTeam(t *testing.T) {
	f := newScopeFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Post(t, srv, "/api/games", f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "18:00",
		"opponent":   "TSV Irgendwo",
		"event_type": "heim",
		"team_ids":   []int{f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if n := countGames(t, f.db); n != 0 {
		t.Errorf("no game must be created, got %d", n)
	}
}

// Auch bei generisch darf eine fremde Mannschaft nicht über ein Heimspiel
// hereinkommen: alle team_ids müssen eigene sein, wenn der Typ heim ist —
// selbst wenn eine eigene dabei ist.
func TestCreateGame_TrainerHomeGameMixedTeams(t *testing.T) {
	f := newScopeFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Post(t, srv, "/api/games", f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "18:00",
		"opponent":   "TSV Irgendwo",
		"event_type": "heim",
		"team_ids":   []int{f.teamA, f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// ── PUT /api/games/{id} ──────────────────────────────────────────────────────

// Die bislang offene Lücke: ein Trainer darf ein fremdes Event nicht ändern.
func TestUpdateGame_TrainerNotOnEvent(t *testing.T) {
	f := newScopeFixture(t)
	foreign := testutil.CreateGame(t, f.db, f.season, f.teamB, "2026-06-14")
	srv := prodserver.New(t, f.db)

	res := testutil.Put(t, srv, "/api/games/"+itoa(foreign), f.trainerToken(t), map[string]any{
		"date":     "2026-07-01",
		"time":     "20:00",
		"opponent": "Gekapert",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var date, opponent string
	f.db.QueryRow(`SELECT date, opponent FROM games WHERE id=?`, foreign).Scan(&date, &opponent)
	if date[:10] != "2026-06-14" || opponent != "Test Opponent" {
		t.Errorf("event must stay unchanged, got date=%s opponent=%s", date, opponent)
	}
}

// Happy-Path des Features im Bearbeiten-Dialog.
func TestUpdateGame_TrainerGenericAddsForeignTeam(t *testing.T) {
	f := newScopeFixture(t)
	eventID := createGenericEvent(t, f.db, f.season, f.teamA)
	srv := prodserver.New(t, f.db)

	res := testutil.Put(t, srv, "/api/games/"+itoa(eventID), f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "14:00",
		"opponent":   "Vereinsfest",
		"event_type": "generisch",
		"team_ids":   []int{f.teamA, f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if got := gameTeamIDs(t, f.db, eventID); len(got) != 2 {
		t.Errorf("expected foreign team added, got %v", got)
	}
}

// Der Trainer darf sich nicht selbst vom eigenen Event abschneiden.
func TestUpdateGame_TrainerRemovesOwnLastTeam(t *testing.T) {
	f := newScopeFixture(t)
	eventID := createGenericEvent(t, f.db, f.season, f.teamA, f.teamB)
	srv := prodserver.New(t, f.db)

	res := testutil.Put(t, srv, "/api/games/"+itoa(eventID), f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "14:00",
		"opponent":   "Vereinsfest",
		"event_type": "generisch",
		"team_ids":   []int{f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if got := gameTeamIDs(t, f.db, eventID); len(got) != 2 {
		t.Errorf("game_teams must stay unchanged, got %v", got)
	}
}

// Ein Trainer darf sein generisches Event nicht in ein Heimspiel einer fremden
// Mannschaft umwidmen — die Typ-Regel gilt für den ZIEL-Typ.
func TestUpdateGame_TrainerGenericToHomeWithForeignTeam(t *testing.T) {
	f := newScopeFixture(t)
	eventID := createGenericEvent(t, f.db, f.season, f.teamA)
	srv := prodserver.New(t, f.db)

	res := testutil.Put(t, srv, "/api/games/"+itoa(eventID), f.trainerToken(t), map[string]any{
		"date":       "2026-06-20",
		"time":       "18:00",
		"opponent":   "TSV Irgendwo",
		"event_type": "heim",
		"team_ids":   []int{f.teamA, f.teamB},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var et string
	f.db.QueryRow(`SELECT event_type FROM games WHERE id=?`, eventID).Scan(&et)
	if et != "generisch" {
		t.Errorf("event_type must stay 'generisch', got %q", et)
	}
}

// sportliche_leitung ist von der Team-Scope-Prüfung ausgenommen.
func TestUpdateGame_SportlicheLeitungUnrestricted(t *testing.T) {
	f := newScopeFixture(t)
	foreign := testutil.CreateGame(t, f.db, f.season, f.teamB, "2026-06-14")
	slUserID := testutil.CreateUser(t, f.db, "standard")
	srv := prodserver.New(t, f.db)

	token := testutil.Token(t, slUserID, "standard", []string{"sportliche_leitung"})
	res := testutil.Put(t, srv, "/api/games/"+itoa(foreign), token, map[string]any{
		"date":     "2026-07-01",
		"time":     "20:00",
		"opponent": "Verlegt",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

// Unbekannte ID muss 404 liefern, nicht 403 — sonst wären fremde Event-IDs
// über den Statuscode aufzählbar.
func TestUpdateGame_UnknownIDReturns404(t *testing.T) {
	f := newScopeFixture(t)
	srv := prodserver.New(t, f.db)

	res := testutil.Put(t, srv, "/api/games/999999", f.trainerToken(t), map[string]any{
		"date":     "2026-07-01",
		"time":     "20:00",
		"opponent": "Nichts",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// ── DELETE /api/games/{id} ───────────────────────────────────────────────────

func TestDeleteGame_TrainerNotOnEvent(t *testing.T) {
	f := newScopeFixture(t)
	foreign := testutil.CreateGame(t, f.db, f.season, f.teamB, "2026-06-14")
	srv := prodserver.New(t, f.db)

	res := testutil.Delete(t, srv, "/api/games/"+itoa(foreign), f.trainerToken(t))
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if n := countGames(t, f.db); n != 1 {
		t.Errorf("event must still exist, got %d games", n)
	}
}

func TestDeleteGame_TrainerOwnEvent(t *testing.T) {
	f := newScopeFixture(t)
	own := testutil.CreateGame(t, f.db, f.season, f.teamA, "2026-06-14")
	srv := prodserver.New(t, f.db)

	res := testutil.Delete(t, srv, "/api/games/"+itoa(own), f.trainerToken(t))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 200/204, got %d", res.StatusCode)
	}
	if n := countGames(t, f.db); n != 0 {
		t.Errorf("event must be deleted, got %d games", n)
	}
}
