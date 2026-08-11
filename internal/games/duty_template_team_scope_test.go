package games_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Tests für die optionale Team-Einschränkung eines Vorlagen-Items
// (game_template_items.team_ids). Leer/NULL = Item gilt für alle Teams des
// Spiels (Bestandsverhalten); nicht leer = Slot entsteht nur für die
// gelisteten Teams.

// seedTeamScopeTemplate legt eine Heim-Vorlage mit genau einem Item an und gibt
// (templateID, dutyTypeID) zurück. Anders als insertHeimTemplate trägt das Item
// hier eine Team-Allowlist, wenn teamIDs nicht leer ist.
func seedTeamScopeTemplate(t *testing.T, db *sql.DB, name string, dutyTypeID int, teamIDs []int) int {
	t.Helper()
	tr, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, 'heim', 75)`, name)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := tr.LastInsertId()
	var scope any
	if len(teamIDs) > 0 {
		b, _ := json.Marshal(teamIDs)
		scope = string(b)
	}
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order, team_ids)
		VALUES (?, ?, 'start', -60, 1, 0, ?)`, templateID, dutyTypeID, scope); err != nil {
		t.Fatalf("seed template item: %v", err)
	}
	return int(templateID)
}

// seedTwoTeamsSameAgeClass gibt zwei Teams derselben Altersklasse zurück, für die
// eine Spieldauer-Regel existiert — Voraussetzung dafür, dass der Regen für ein
// Spiel mit beiden Teams überhaupt Slots berechnen kann.
func seedTwoTeamsSameAgeClass(t *testing.T, db *sql.DB) (int, int) {
	t.Helper()
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	for _, id := range []int{teamA, teamB} {
		if _, err := db.Exec(`UPDATE teams SET age_class=? WHERE id=?`, "A-Jugend", id); err != nil {
			t.Fatalf("set age_class: %v", err)
		}
	}
	if _, err := db.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES (?, ?, ?)`,
		"A-Jugend", 30, 15); err != nil {
		t.Fatalf("seed age_class_game_rules: %v", err)
	}
	return teamA, teamB
}

// templateItemsFromAPI lädt die Items einer Vorlage über GET /api/duty-templates/{id}.
func templateItemsFromAPI(t *testing.T, srv *httptest.Server, id int, token string) []map[string]any {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-templates/"+strconv.Itoa(id), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET duty-template: expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode template: %v", err)
	}
	return body.Items
}

// TestUpdateTemplate_TeamIdsGespeichertUndGeladen: eine gespeicherte Team-Allowlist
// kommt beim erneuten Lesen unverändert zurück.
func TestUpdateTemplate_TeamIdsGespeichertUndGeladen(t *testing.T) {
	db := testutil.NewDB(t)
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)
	dutyTypeID := insertDutyType(t, db, "Kamera", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"team_ids": []int{teamA, teamB},
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	raw, ok := items[0]["team_ids"].([]any)
	if !ok {
		t.Fatalf("expected team_ids array in response, got %#v", items[0]["team_ids"])
	}
	got := []int{}
	for _, v := range raw {
		got = append(got, int(v.(float64)))
	}
	if len(got) != 2 || got[0] != teamA || got[1] != teamB {
		t.Errorf("expected team_ids [%d %d], got %v", teamA, teamB, got)
	}
}

// TestUpdateTemplate_LeereTeamIdsBedeutetAlleTeams: ohne team_ids bleibt die
// Spalte NULL — kein Default-Array, das versehentlich einschränken würde.
func TestUpdateTemplate_LeereTeamIdsBedeutetAlleTeams(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kasse", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stored sql.NullString
	if err := db.QueryRow(
		`SELECT team_ids FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read team_ids: %v", err)
	}
	if stored.Valid {
		t.Errorf("expected team_ids NULL, got %q", stored.String)
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if _, present := items[0]["team_ids"]; present {
		t.Errorf("expected team_ids omitted from response, got %#v", items[0]["team_ids"])
	}
}

// TestUpdateTemplate_UnbekannteTeamId: eine nicht existierende teams.id wird mit
// 400 invalid_team abgelehnt, die gespeicherte Vorlage bleibt unverändert.
func TestUpdateTemplate_UnbekannteTeamId(t *testing.T) {
	db := testutil.NewDB(t)
	teamA, _ := seedTwoTeamsSameAgeClass(t, db)
	dutyTypeID := insertDutyType(t, db, "Kamera", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, []int{teamA})

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Umbenannt", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"team_ids": []int{teamA, 999999},
		}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "invalid_team" {
		t.Errorf("expected error invalid_team, got %q", errBody.Error)
	}

	// Vorlage darf nicht angefasst worden sein (weder Name noch Items).
	var name string
	db.QueryRow(`SELECT name FROM game_templates WHERE id=?`, templateID).Scan(&name)
	if name != "Heim" {
		t.Errorf("expected template name unchanged (Heim), got %q", name)
	}
	var stored sql.NullString
	db.QueryRow(`SELECT team_ids FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored)
	if !stored.Valid || stored.String != "["+strconv.Itoa(teamA)+"]" {
		t.Errorf("expected team_ids unchanged ([%d]), got valid=%v %q", teamA, stored.Valid, stored.String)
	}
}

// TestUpdateTemplate_StandardNutzerVerboten: ohne Vereinsfunktion vorstand ist
// PUT /api/duty-templates/{id} gesperrt (Gegenstück zum bestehenden
// TestCreateDutyTemplate_TrainerForbidden).
func TestUpdateTemplate_StandardNutzerVerboten(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kasse", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	userID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Gekapert", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// TestRegen_TeamEingeschraenktesItemNurFuerGelisteteTeams: bei einem Spiel mit
// zwei Teams erzeugt ein auf Team A eingeschränktes Item ausschließlich für A
// einen Slot — B bekommt keinen, obwohl es am selben Spiel hängt. Ein zweites,
// uneingeschränktes Item derselben Vorlage bleibt davon unberührt.
func TestRegen_TeamEingeschraenktesItemNurFuerGelisteteTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)

	kameraID := insertDutyType(t, db, "Kamera", 2.0)
	kasseID := insertDutyType(t, db, "Kasse", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", kameraID, []int{teamA})
	// Zweites Item ohne Einschränkung in derselben Vorlage.
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, 'start', -30, 1, 1)`, templateID, kasseID); err != nil {
		t.Fatalf("seed second item: %v", err)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Post(t, srv, "/api/games", token, map[string]any{
		"date": "2026-06-13", "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamA, teamB}, "event_type": "heim",
		"season_id": seasonID, "template_id": templateID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create game: expected 201, got %d", res.StatusCode)
	}

	teamsForDutyType := func(dutyTypeID int) []int {
		rows, err := db.Query(
			`SELECT team_id FROM duty_slots WHERE duty_type_id=? AND is_custom=0 ORDER BY team_id`, dutyTypeID)
		if err != nil {
			t.Fatalf("query slots: %v", err)
		}
		defer rows.Close()
		var ids []int
		for rows.Next() {
			var id sql.NullInt64
			rows.Scan(&id)
			ids = append(ids, int(id.Int64))
		}
		return ids
	}

	kamera := teamsForDutyType(kameraID)
	if len(kamera) != 1 || kamera[0] != teamA {
		t.Errorf("expected Kamera-Slot nur für Team A (%d), got team_ids %v", teamA, kamera)
	}
	kasse := teamsForDutyType(kasseID)
	if len(kasse) != 2 {
		t.Errorf("expected Kasse-Slots für beide Teams, got team_ids %v", kasse)
	}
}

// TestRegen_ItemOhneTeamIdsGiltFuerAlleTeams: Rückwärtskompatibilität — ohne
// team_ids entsteht der Slot weiterhin für jedes Team des Spiels.
func TestRegen_ItemOhneTeamIdsGiltFuerAlleTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)

	dutyTypeID := insertDutyType(t, db, "Kasse", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Post(t, srv, "/api/games", token, map[string]any{
		"date": "2026-06-13", "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamA, teamB}, "event_type": "heim",
		"season_id": seasonID, "template_id": templateID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create game: expected 201, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM duty_slots WHERE duty_type_id=? AND is_custom=0`, dutyTypeID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 Slots (ein Team je Slot), got %d", count)
	}
}

// TestRegen_ZaehlungBeruecksichtigtTeamFilter: RegenSummary.Created[].Count meldet
// die Zahl der tatsächlich getroffenen Teams, nicht die Teamzahl des Spiels.
func TestRegen_ZaehlungBeruecksichtigtTeamFilter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)

	dutyTypeID := insertDutyType(t, db, "Kamera", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, []int{teamA})

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Post(t, srv, "/api/games", token, map[string]any{
		"date": "2026-06-13", "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamA, teamB}, "event_type": "heim",
		"season_id": seasonID, "template_id": templateID,
	})
	if res.StatusCode != http.StatusCreated {
		res.Body.Close()
		t.Fatalf("create game: expected 201, got %d", res.StatusCode)
	}
	summary := readRegenSummary(t, res)
	res.Body.Close()

	found := false
	for _, c := range summary.Created {
		if c.DutyType != "Kamera" {
			continue
		}
		found = true
		if c.Count != 1 {
			t.Errorf("expected Created count 1 (nur Team A), got %d", c.Count)
		}
	}
	if !found {
		t.Errorf("expected Created entry for Kamera, got %+v", summary.Created)
	}
}

// ── Vorschau (GET /api/duty-templates/{id}/preview) ──────────────────────────
//
// Die Vorschau muss dieselbe Team-Einschränkung anwenden wie der Regen: ein
// Eintrag erscheint genau dann, wenn für mindestens eines der übergebenen Teams
// real ein Slot entstünde. Ohne Team-Angabe bleibt sie ungefiltert, bei
// generischen Vorlagen immer (dort ignoriert auch der Regen team_ids).

// previewDutyTypes ruft die Vorschau ab und gibt die Diensttyp-Namen zurück.
func previewDutyTypes(t *testing.T, srv *httptest.Server, templateID int, query, token string) []string {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-templates/"+strconv.Itoa(templateID)+"/preview?time=14:00"+query, token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("preview: expected 200, got %d", res.StatusCode)
	}
	var items []struct {
		DutyTypeName string `json:"duty_type_name"`
	}
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	names := []string{}
	for _, it := range items {
		names = append(names, it.DutyTypeName)
	}
	return names
}

func contains(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// seedPreviewFixture: Heim-Vorlage mit "Kamera" (nur teamA) und "Kasse" (alle Teams).
func seedPreviewFixture(t *testing.T, db *sql.DB, teamA int) int {
	t.Helper()
	kamera := insertDutyType(t, db, "Kamera", 2.0)
	kasse := insertDutyType(t, db, "Kasse", 2.0)
	templateID := seedTeamScopeTemplate(t, db, "Heim", kamera, []int{teamA})
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order)
		VALUES (?, ?, 'start', -30, 1, 1)`, templateID, kasse); err != nil {
		t.Fatalf("seed second item: %v", err)
	}
	return templateID
}

func TestPreview_TeamEingeschraenktesItemFehltBeiFremdemTeam(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)
	templateID := seedPreviewFixture(t, db, teamA)

	userID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "admin", []string{"vorstand"})

	names := previewDutyTypes(t, srv, templateID, "&team_ids="+strconv.Itoa(teamB), token)
	if contains(names, "Kamera") {
		t.Errorf("Kamera darf für Team B nicht in der Vorschau stehen, got %v", names)
	}
	if !contains(names, "Kasse") {
		t.Errorf("Kasse (ohne Einschränkung) muss in der Vorschau stehen, got %v", names)
	}
}

func TestPreview_TeamEingeschraenktesItemSichtbarBeiTreffer(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)
	templateID := seedPreviewFixture(t, db, teamA)

	userID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "admin", []string{"vorstand"})

	// Mehrteam-Event: teamA trifft die Allowlist → Eintrag bleibt sichtbar.
	// Trennzeichen bewusst prozent-kodiert (%2C) — genau so schickt es der Wizard,
	// seit buildPreviewUrl die Query über URLSearchParams baut.
	q := "&team_ids=" + strconv.Itoa(teamB) + "%2C" + strconv.Itoa(teamA)
	names := previewDutyTypes(t, srv, templateID, q, token)
	if !contains(names, "Kamera") {
		t.Errorf("Kamera muss bei Treffer sichtbar sein, got %v", names)
	}
}

func TestPreview_OhneTeamAngabeUngefiltert(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamA, _ := seedTwoTeamsSameAgeClass(t, db)
	templateID := seedPreviewFixture(t, db, teamA)

	userID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "admin", []string{"vorstand"})

	names := previewDutyTypes(t, srv, templateID, "", token)
	if !contains(names, "Kamera") {
		t.Errorf("ohne team_ids muss die Vorschau ungefiltert bleiben, got %v", names)
	}
}

func TestPreview_TeamsAusGameIdAbgeleitet(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)
	templateID := seedPreviewFixture(t, db, teamA)
	// Spiel hängt nur an teamB → Kamera (nur teamA) darf nicht erscheinen.
	gameID := testutil.CreateGame(t, db, seasonID, teamB, "2026-06-13")

	userID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "admin", []string{"vorstand"})

	names := previewDutyTypes(t, srv, templateID, "&game_id="+strconv.Itoa(gameID), token)
	if contains(names, "Kamera") {
		t.Errorf("Teams aus game_id müssen den Filter speisen, got %v", names)
	}
	if !contains(names, "Kasse") {
		t.Errorf("Kasse muss sichtbar bleiben, got %v", names)
	}
}

func TestPreview_GenerischeVorlageWirdNichtGefiltert(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamA, teamB := seedTwoTeamsSameAgeClass(t, db)

	kamera := insertDutyType(t, db, "Kamera", 2.0)
	tr, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES ('Turnier', 'generisch', 120)`)
	if err != nil {
		t.Fatalf("seed template: %v", err)
	}
	templateID, _ := tr.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO game_template_items (template_id, duty_type_id, anchor, offset_minutes, slots_count, sort_order, team_ids)
		VALUES (?, ?, 'start', -60, 1, 0, ?)`, templateID, kamera, "["+strconv.Itoa(teamA)+"]"); err != nil {
		t.Fatalf("seed item: %v", err)
	}

	userID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "admin", []string{"vorstand"})

	// teamB trifft die Allowlist nicht — bei generisch ignoriert der Regen sie aber,
	// die Vorschau muss den Eintrag also trotzdem zeigen.
	names := previewDutyTypes(t, srv, int(templateID), "&team_ids="+strconv.Itoa(teamB), token)
	if !contains(names, "Kamera") {
		t.Errorf("generische Vorlage darf nicht team-gefiltert werden, got %v", names)
	}
}
