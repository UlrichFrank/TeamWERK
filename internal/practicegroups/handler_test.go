package practicegroups_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/practicegroups"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// routes registriert den Übungsgruppen-Baum so, wie ihn BuildRouter im
// Vorstand-Tier mountet — inklusive Gate, damit der 403-Fall echt geprüft wird
// und nicht nur der Handler ohne Wache.
func routes(h *practicegroups.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Get("/api/practice-groups", h.List)
			r.Post("/api/practice-groups", h.Create)
			r.Get("/api/practice-groups/{id}", h.Get)
			r.Put("/api/practice-groups/{id}", h.Update)
			r.Delete("/api/practice-groups/{id}", h.Delete)
			r.Get("/api/practice-groups/{id}/member-suggestions", h.MemberSuggestions)
		})
	}
}

// waitForEvent liest ein Event vom Hub oder scheitert nach einer kurzen Frist.
func waitForEvent(t *testing.T, ch chan string) string {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("kein Broadcast empfangen")
		return ""
	}
}

// ── Anlage ────────────────────────────────────────────────────────────────────

// TestCreatePracticeGroup_Erfolg: die angelegte Zeile trägt kind='practice' und
// hat weder Altersklasse noch Geschlecht noch Jahrgang noch Team.
func TestCreatePracticeGroup_Erfolg(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	seasonID := testutil.CreateSeason(t, db, "2025/26")

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Torwarttraining"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", resp.StatusCode)
	}
	var body struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		SeasonID int    `json:"season_id"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Name != "Torwarttraining" || body.SeasonID != seasonID {
		t.Fatalf("unerwartete Antwort: %+v", body)
	}

	var kind string
	var ageClass, gender, birthYear, teamID any
	err := db.QueryRow(
		`SELECT kind, age_class, gender, dedicated_birth_year, team_id FROM kader WHERE id=?`,
		body.ID).Scan(&kind, &ageClass, &gender, &birthYear, &teamID)
	if err != nil {
		t.Fatalf("kader lesen: %v", err)
	}
	if kind != "practice" {
		t.Errorf("kind = %q, erwartet practice", kind)
	}
	for label, v := range map[string]any{
		"age_class": ageClass, "gender": gender,
		"dedicated_birth_year": birthYear, "team_id": teamID,
	} {
		if v != nil {
			t.Errorf("%s = %v, erwartet NULL", label, v)
		}
	}
}

// TestCreatePracticeGroup_KeinTeamZwilling: die Abwesenheit der teams-Zeile ist
// das Gate, über das alle team-gebundenen Flächen unerreichbar bleiben
// (proposal.md — Invariante 2). Deshalb wird sie hier gezählt, nicht erklärt.
func TestCreatePracticeGroup_KeinTeamZwilling(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	testutil.CreateSeason(t, db, "2025/26")

	var before int
	db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&before)

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Athletik"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", resp.StatusCode)
	}

	var after int
	db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&after)
	if after != before {
		t.Fatalf("teams-Zeilen: vorher %d, nachher %d — eine Übungsgruppe darf keinen Zwilling anlegen", before, after)
	}
}

// TestCreatePracticeGroup_NameDoppeltInSaison: der Partial-Unique-Index auf
// (season_id, name) WHERE kind='practice' trägt die Eindeutigkeit.
func TestCreatePracticeGroup_NameDoppeltInSaison(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	testutil.CreateSeason(t, db, "2025/26")

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	first := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Torwarttraining"})
	first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("erste Anlage: erwartet 201, bekommen %d", first.StatusCode)
	}
	second := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Torwarttraining"})
	defer second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("zweite Anlage: erwartet 409, bekommen %d", second.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE kind='practice'`).Scan(&n)
	if n != 1 {
		t.Fatalf("erwartet 1 Übungsgruppe, gefunden %d", n)
	}
}

// TestCreatePracticeGroup_NameInAndererSaisonErlaubt: die Eindeutigkeit gilt je
// Saison — „Torwarttraining 25/26" und „…26/27" sind zwei Gruppen.
func TestCreatePracticeGroup_NameInAndererSaisonErlaubt(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	oldSeason := testutil.CreateSeason(t, db, "2025/26")
	testutil.CreatePracticeGroup(t, db, oldSeason, "Torwarttraining")
	testutil.CreateSeason(t, db, "2026/27") // deaktiviert die alte Saison

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Torwarttraining"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", resp.StatusCode)
	}
}

// TestCreatePracticeGroup_OhneVorstand: das Gate liegt im Router-Tier.
func TestCreatePracticeGroup_OhneVorstand(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	testutil.CreateSeason(t, db, "2025/26")

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"trainer"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Athletik"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE kind='practice'`).Scan(&n)
	if n != 0 {
		t.Fatalf("trotz 403 wurden %d Gruppen angelegt", n)
	}
}

// TestCreatePracticeGroup_OhneAktiveSaison: kader.season_id ist NOT NULL — ohne
// aktive Saison gibt es keinen Ort, an dem die Gruppe leben könnte.
func TestCreatePracticeGroup_OhneAktiveSaison(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=0`)

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Post(t, srv, "/api/practice-groups", token, map[string]any{"name": "Athletik"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekommen %d", resp.StatusCode)
	}
}

// ── Pflege ────────────────────────────────────────────────────────────────────

// TestUpdatePracticeGroup_MitgliederUndTrainer: Mitglieder und Trainer landen in
// denselben Tabellen wie bei der Mannschaftsvariante, und die Mutation
// broadcastet.
func TestUpdatePracticeGroup_MitgliederUndTrainer(t *testing.T) {
	db := testutil.NewDB(t)
	eventHub := hub.NewHub()
	h := practicegroups.NewHandler(db, eventHub)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")

	playerID := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	trainerID := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	sub := eventHub.Subscribe()
	defer eventHub.Unsubscribe(sub)

	resp := testutil.Put(t, srv, "/api/practice-groups/"+strconv.Itoa(groupID), token, map[string]any{
		"members_add":  []int{playerID},
		"trainers_add": []int{trainerID},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", resp.StatusCode)
	}

	var members, trainers int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, groupID, playerID).Scan(&members)
	db.QueryRow(`SELECT COUNT(*) FROM kader_trainers WHERE kader_id=? AND member_id=?`, groupID, trainerID).Scan(&trainers)
	if members != 1 || trainers != 1 {
		t.Fatalf("members=%d trainers=%d, erwartet je 1", members, trainers)
	}
	if ev := waitForEvent(t, sub); ev != "practice-groups" {
		t.Fatalf("Broadcast = %q, erwartet practice-groups", ev)
	}
}

// TestListPracticeGroups_NurAktiveSaison: Gruppen abgelaufener Saisons bleiben
// in der DB, tauchen in der Liste aber nicht auf.
func TestListPracticeGroups_NurAktiveSaison(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	oldSeason := testutil.CreateSeason(t, db, "2025/26")
	testutil.CreatePracticeGroup(t, db, oldSeason, "Alte Gruppe")
	newSeason := testutil.CreateSeason(t, db, "2026/27")
	testutil.CreatePracticeGroup(t, db, newSeason, "Neue Gruppe")

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Get(t, srv, "/api/practice-groups", token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 || body.Items[0].Name != "Neue Gruppe" {
		t.Fatalf("erwartet nur „Neue Gruppe\", bekommen %+v", body.Items)
	}
}

// ── Löschen ───────────────────────────────────────────────────────────────────

// TestDeletePracticeGroup_MitTrainingsAbgelehnt: die Trainingshistorie überlebt
// jedes Aufräumen (proposal.md — Invariante 4).
func TestDeletePracticeGroup_MitTrainingsAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")

	for _, d := range []string{"2025-10-01", "2025-10-08", "2025-10-15"} {
		if _, err := db.Exec(
			`INSERT INTO training_sessions (kader_id, season_id, date, start_time, end_time, title)
			 VALUES (?, ?, ?, '18:00', '20:00', 'TW')`, groupID, seasonID, d); err != nil {
			t.Fatalf("Termin anlegen: %v", err)
		}
	}

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Delete(t, srv, "/api/practice-groups/"+strconv.Itoa(groupID), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekommen %d", resp.StatusCode)
	}
	var body struct {
		TrainingCount int `json:"training_count"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.TrainingCount != 3 {
		t.Fatalf("training_count = %d, erwartet 3", body.TrainingCount)
	}

	var groups, sessions int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE id=?`, groupID).Scan(&groups)
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE kader_id=?`, groupID).Scan(&sessions)
	if groups != 1 || sessions != 3 {
		t.Fatalf("Gruppe=%d Termine=%d — beides muss erhalten bleiben", groups, sessions)
	}
}

// TestDeletePracticeGroup_OhneTrainingsErfolg: ohne Historie ist die Gruppe
// löschbar.
func TestDeletePracticeGroup_OhneTrainingsErfolg(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Sichtung")

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	resp := testutil.Delete(t, srv, "/api/practice-groups/"+strconv.Itoa(groupID), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE id=?`, groupID).Scan(&n)
	if n != 0 {
		t.Fatalf("Gruppe existiert noch")
	}
}

// TestGetPracticeGroup_TeamKaderIstNichtAdressierbar: ein Mannschaftskader ist
// über die Übungsgruppen-Route nicht erreichbar — die Handler filtern
// kind='practice', nicht nur die Liste.
func TestGetPracticeGroup_TeamKaderIstNichtAdressierbar(t *testing.T) {
	db := testutil.NewDB(t)
	h := practicegroups.NewHandler(db, hub.NewHub())
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, routes(h))

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/practice-groups/" + strconv.Itoa(kaderID)},
		{http.MethodDelete, "/api/practice-groups/" + strconv.Itoa(kaderID)},
	} {
		resp := testutil.Do(t, srv, tc.method, tc.path, token, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: erwartet 404, bekommen %d", tc.method, tc.path, resp.StatusCode)
		}
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE id=?`, kaderID).Scan(&n)
	if n != 1 {
		t.Fatalf("der Mannschaftskader wurde über die Übungsgruppen-Route gelöscht")
	}
}
