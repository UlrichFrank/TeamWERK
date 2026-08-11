package games_test

// Tests für die Massen-Regeneration (duty-bulk-regen). Nutzt dieselben Helfer wie
// handler_test.go (insertDutyType, insertHeimTemplate, seedAgeClassRule, countRows).

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// --- gemeinsame Helfer -------------------------------------------------------------

func bulkFutureDate(t *testing.T, days int) string {
	t.Helper()
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func bulkVorstandToken(t *testing.T, db *sql.DB) string {
	t.Helper()
	uid := testutil.CreateUser(t, db, "standard")
	return testutil.Token(t, uid, "standard", []string{"vorstand"})
}

type bulkRegenTotalsDTO struct {
	Games           int `json:"games"`
	Created         int `json:"created"`
	Deleted         int `json:"deleted"`
	CustomKept      int `json:"custom_kept"`
	CustomDeleted   int `json:"custom_deleted"`
	AssignmentsKept int `json:"assignments_kept"`
	AssignmentsLost int `json:"assignments_lost"`
	Conflicts       int `json:"conflicts"`
	NotifiedUsers   int `json:"notified_users"`
}

type bulkRegenRowDTO struct {
	GameID          int    `json:"game_id"`
	Date            string `json:"date"`
	EffectiveAction string `json:"effective_action"`
	Excluded        bool   `json:"excluded"`
	Created         int    `json:"created"`
	DeletedAuto     int    `json:"deleted_auto"`
	DeletedCustom   int    `json:"deleted_custom"`
	AssignmentsKept int    `json:"assignments_kept"`
	AssignmentsLost int    `json:"assignments_lost"`
}

type bulkRegenResponseDTO struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`
	Rows     []bulkRegenRowDTO  `json:"rows"`
	Totals   bulkRegenTotalsDTO `json:"totals"`
	Warnings []string           `json:"warnings"`
	Applied  bool               `json:"applied"`
}

func decodeBulkRegen(t *testing.T, res *http.Response) bulkRegenResponseDTO {
	t.Helper()
	var got bulkRegenResponseDTO
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode bulk-regen response: %v", err)
	}
	return got
}

// dbFingerprint dumps every table's rows as a comparable string, used to prove a
// preview request wrote nothing (task 7.2). Column order comes from the driver, which
// is stable within one schema/db instance — good enough for a before/after diff.
func dbFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	tableRows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("dbFingerprint tables: %v", err)
	}
	var tables []string
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	tableRows.Close()

	out := ""
	for _, tbl := range tables {
		out += "## " + tbl + "\n"
		r, err := db.Query(`SELECT * FROM "` + tbl + `"`)
		if err != nil {
			t.Fatalf("dbFingerprint select %s: %v", tbl, err)
		}
		cols, _ := r.Columns()
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				r.Close()
				t.Fatalf("dbFingerprint scan %s: %v", tbl, err)
			}
			out += fmt.Sprintf("%v\n", vals)
		}
		r.Close()
	}
	return out
}

// --- 7.1 Routen-Matrix --------------------------------------------------------------

func TestBulkRegenPreview_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)

	gameDate := bulkFutureDate(t, 10)
	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", bulkVorstandToken(t, db), map[string]any{
		"date": gameDate, "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
	})
	createRes.Body.Close()

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", bulkVorstandToken(t, db), map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "template", "template_id": templateID}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)
	if got.Applied {
		t.Errorf("preview must not report applied=true")
	}
	if got.Totals.Games != 1 || got.Totals.Created != 1 {
		t.Errorf("expected 1 game / 1 created slot, got totals=%+v", got.Totals)
	}
}

func TestBulkRegenPreview_DefaultRangeWithoutFromTo(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)

	lastDate := bulkFutureDate(t, 45)
	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", bulkVorstandToken(t, db), map[string]any{
		"date": lastDate, "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
	})
	createRes.Body.Close()

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", bulkVorstandToken(t, db), map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)
	wantFrom := bulkFutureDate(t, 1)
	if got.Range.From != wantFrom {
		t.Errorf("expected default range.from=%s, got %s", wantFrom, got.Range.From)
	}
	if got.Range.To != lastDate {
		t.Errorf("expected default range.to=MAX(games.date)=%s, got %s", lastDate, got.Range.To)
	}
}

func TestBulkRegenPreview_OhneVorstand_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	srv := prodserver.New(t, db)
	uid := testutil.CreateUser(t, db, "standard")
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", testutil.Token(t, uid, "standard", nil), map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

func TestBulkRegenPreview_KeineAktiveSaison_400(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", bulkVorstandToken(t, db), map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}

func TestBulkRegenPreview_RangeInPast_400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", bulkVorstandToken(t, db), map[string]any{
		"from": bulkFutureDate(t, -1), "to": bulkFutureDate(t, 10),
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Error != "range_in_past" {
		t.Errorf("expected error 'range_in_past', got %q", body.Error)
	}
}

func TestBulkRegenPreview_InvalidTemplate_400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", bulkVorstandToken(t, db), map[string]any{
		"date": bulkFutureDate(t, 10), "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
	})
	createRes.Body.Close()

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", bulkVorstandToken(t, db), map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "template", "template_id": 999999}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Error != "invalid_template" {
		t.Errorf("expected error 'invalid_template', got %q", body.Error)
	}
}

func TestBulkRegenApply_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)

	token := bulkVorstandToken(t, db)
	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
		"date": bulkFutureDate(t, 10), "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
	})
	var created struct {
		ID int `json:"id"`
	}
	json.NewDecoder(createRes.Body).Decode(&created)
	createRes.Body.Close()

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "template", "template_id": templateID}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)
	if !got.Applied {
		t.Errorf("expected applied=true")
	}
	if got := countRows(t, db, "duty_slots", "game_id=? AND is_custom=0", created.ID); got != 1 {
		t.Errorf("expected 1 auto slot written, got %d", got)
	}
	var tpl sql.NullInt64
	db.QueryRow(`SELECT template_id FROM games WHERE id=?`, created.ID).Scan(&tpl)
	if !tpl.Valid || int(tpl.Int64) != templateID {
		t.Errorf("expected games.template_id=%d, got %+v", templateID, tpl)
	}
}

func TestBulkRegenApply_OhneVorstand_403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	srv := prodserver.New(t, db)
	uid := testutil.CreateUser(t, db, "standard")
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", testutil.Token(t, uid, "standard", nil), map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}

func TestBulkRegenApply_KeineAktiveSaison_400(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", bulkVorstandToken(t, db), map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}

func TestBulkRegenApply_RangeInPast_400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", bulkVorstandToken(t, db), map[string]any{
		"from": bulkFutureDate(t, 0), "to": bulkFutureDate(t, 10),
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}

// --- 7.2 Preview schreibt nicht ------------------------------------------------------

func TestPreviewBulkRegen_SchreibtNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)

	for i := 0; i < 3; i++ {
		res := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
			"date": bulkFutureDate(t, 10+i), "time": "14:00", "opponent": "FC Test",
			"team_ids": []int{teamID}, "event_type": "heim",
			"season_id": seasonID, "template_id": templateID,
		})
		res.Body.Close()
	}

	before := dbFingerprint(t, db)

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "purge"}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	after := dbFingerprint(t, db)
	if before != after {
		t.Errorf("preview must not change the database (purge over the whole range)")
	}

	// Poison-Sanity: der Vergleicher muss eine echte Änderung auch erkennen.
	db.Exec(`UPDATE games SET opponent='POISON' WHERE season_id=?`, seasonID)
	poisoned := dbFingerprint(t, db)
	if poisoned == after {
		t.Fatalf("dbFingerprint poison-sanity failed: comparator did not detect a real change")
	}
}

// --- 7.3 Preview sagt die Wahrheit ---------------------------------------------------

func TestBulkRegen_PreviewSagtDieWahrheit(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)

	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
		"date": bulkFutureDate(t, 10), "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
	})
	var created struct {
		ID int `json:"id"`
	}
	json.NewDecoder(createRes.Body).Decode(&created)
	createRes.Body.Close()

	body := map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "template", "template_id": templateID}},
	}

	srv := prodserver.New(t, db)
	previewRes := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", token, body)
	previewGot := decodeBulkRegen(t, previewRes)
	previewRes.Body.Close()

	applyRes := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, body)
	applyGot := decodeBulkRegen(t, applyRes)
	applyRes.Body.Close()

	if previewGot.Totals != applyGot.Totals {
		t.Errorf("preview and apply totals must be identical, got preview=%+v apply=%+v", previewGot.Totals, applyGot.Totals)
	}
	if got := countRows(t, db, "duty_slots", "game_id=? AND is_custom=0", created.ID); got != applyGot.Totals.Created {
		t.Errorf("actual duty_slots count %d does not match totals.created %d", got, applyGot.Totals.Created)
	}
}

// --- 7.4 purge vs none ---------------------------------------------------------------

func TestBulkRegen_PurgeVsNone(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)

	makeGame := func(date string) int {
		res := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
			"date": date, "time": "14:00", "opponent": "FC Test",
			"team_ids": []int{teamID}, "event_type": "heim",
			"season_id": seasonID, "template_id": templateID,
		})
		var created struct {
			ID int `json:"id"`
		}
		json.NewDecoder(res.Body).Decode(&created)
		res.Body.Close()
		// A hand-made (is_custom=1) slot on each game.
		db.Exec(`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, team_id, season_id, game_id, is_custom)
			VALUES ('Manuell', ?, ?, 1, ?, ?, ?, 1)`, date, dutyTypeID, teamID, seasonID, created.ID)
		return created.ID
	}

	noneGameID := makeGame(bulkFutureDate(t, 10))
	purgeGameID := makeGame(bulkFutureDate(t, 11))

	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"overrides": []map[string]any{
			{"game_id": noneGameID, "action": "none"},
			{"game_id": purgeGameID, "action": "purge"},
		},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	if got := countRows(t, db, "duty_slots", "game_id=? AND is_custom=1", noneGameID); got != 1 {
		t.Errorf("expected 'none' to keep the custom slot, got %d", got)
	}
	if got := countRows(t, db, "duty_slots", "game_id=? AND is_custom=1", purgeGameID); got != 0 {
		t.Errorf("expected 'purge' to delete the custom slot, got %d", got)
	}
}

// --- 7.5 notify:false ----------------------------------------------------------------

func TestBulkRegen_NotifyFalse_SendetNichts(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)
	assigneeID := testutil.CreateUser(t, db, "standard")

	createRes := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
		"date": bulkFutureDate(t, 10), "time": "14:00", "opponent": "FC Test",
		"team_ids": []int{teamID}, "event_type": "heim",
		"season_id": seasonID, "template_id": templateID,
	})
	var created struct {
		ID int `json:"id"`
	}
	json.NewDecoder(createRes.Body).Decode(&created)
	createRes.Body.Close()

	var slotID int
	db.QueryRow(`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, created.ID).Scan(&slotID)
	insertDutyAssignment(t, db, slotID, assigneeID, "assigned")

	cat := captureNotifyCategory(t)
	srv := prodserver.New(t, db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"defaults": map[string]any{"heim": map[string]any{"action": "none"}},
		"notify":   false,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)
	if got.Totals.NotifiedUsers != 1 {
		t.Errorf("expected totals.notified_users=1 even with notify:false, got %d", got.Totals.NotifiedUsers)
	}

	select {
	case c := <-cat:
		t.Errorf("expected no notify.Send call with notify:false, got category %q", c)
	case <-time.After(300 * time.Millisecond):
		// erwartet: Stille.
	}
}

// --- 7.6 Ein Broadcast pro Lauf --------------------------------------------------------

func TestBulkRegenApply_EinBroadcastProLauf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)

	srv, sharedHub := prodserver.NewWithHub(t, db)
	token := bulkVorstandToken(t, db)

	// 40 Termine an 25 Tagen (mehrere Spiele an manchen Tagen).
	dayCount := 25
	gameCount := 40
	for i := 0; i < gameCount; i++ {
		date := bulkFutureDate(t, 10+(i%dayCount))
		res := testutil.Post(t, srv, "/api/games", token, map[string]any{
			"date": date, "time": fmt.Sprintf("%02d:00", 10+(i/dayCount)), "opponent": fmt.Sprintf("Gegner %d", i),
			"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
		})
		res.Body.Close()
	}

	dutiesCh := sharedHub.SubscribeUser(999001)
	gamesCh := sharedHub.SubscribeUser(999001)
	defer sharedHub.UnsubscribeUser(999001, dutiesCh)

	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 60),
		"defaults": map[string]any{"heim": map[string]any{"action": "template", "template_id": templateID}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	dutiesCount, gamesCount := 0, 0
	deadline := time.After(2 * time.Second)
	quiet := time.NewTimer(400 * time.Millisecond)
	defer quiet.Stop()
loop:
	for {
		select {
		case ev := <-gamesCh:
			switch ev {
			case "duties":
				dutiesCount++
			case "games":
				gamesCount++
			}
			if !quiet.Stop() {
				<-quiet.C
			}
			quiet.Reset(400 * time.Millisecond)
		case <-quiet.C:
			break loop
		case <-deadline:
			break loop
		}
	}

	if dutiesCount != 1 {
		t.Errorf("expected exactly 1 'duties' broadcast, got %d", dutiesCount)
	}
	if gamesCount != 1 {
		t.Errorf("expected exactly 1 'games' broadcast, got %d", gamesCount)
	}
}

// --- 7.7 Vergangenheit unerreichbar -----------------------------------------------------

func TestBulkRegen_VergangenheitUnerreichbar(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	token := bulkVorstandToken(t, db)

	// A past game with a slot + assignment, created directly (bypassing the API's
	// own protections against past dates, to simulate pre-existing history).
	pastDate := bulkFutureDate(t, -5)
	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type) VALUES (?,?,?,?,1,'heim')`,
		seasonID, "Vergangen", pastDate, "14:00")
	if err != nil {
		t.Fatalf("seed past game: %v", err)
	}
	pastGameID64, _ := res.LastInsertId()
	pastGameID := int(pastGameID64)
	db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, pastGameID, teamID)
	slotRes, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, slots_total, team_id, season_id, game_id)
		 VALUES ('Test', ?, '13:00', ?, 1, ?, ?, ?)`, pastDate, dutyTypeID, teamID, seasonID, pastGameID)
	if err != nil {
		t.Fatalf("seed past slot: %v", err)
	}
	pastSlotID64, _ := slotRes.LastInsertId()
	pastSlotID := int(pastSlotID64)
	assigneeID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, pastSlotID, assigneeID, "fulfilled")

	srv := prodserver.New(t, db)
	res2 := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
	})
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (future-only range with an unrelated past game), got %d", res2.StatusCode)
	}

	// The past slot and its assignment must be completely untouched.
	if got := countRows(t, db, "duty_slots", "id=?", pastSlotID); got != 1 {
		t.Errorf("expected the past slot to still exist unchanged, got count=%d", got)
	}
	if got := countRows(t, db, "duty_assignments", "duty_slot_id=? AND user_id=?", pastSlotID, assigneeID); got != 1 {
		t.Errorf("expected the past assignment to still exist unchanged, got count=%d", got)
	}

	// A request whose range itself starts in the past must be rejected outright.
	res3 := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, map[string]any{
		"from": pastDate, "to": bulkFutureDate(t, 30),
	})
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 range_in_past for a from-date in the past, got %d", res3.StatusCode)
	}
}

// --- 13.2 Idempotenz (manueller End-to-End-Test, hier automatisiert nachgezogen) ------

// TestBulkRegen_ZweiterIdentischerLaufIstIdempotent: ein Lauf mit gemischten Zuständen
// (template/none/purge, dazu ein Termin mit bestehender Zuweisung), unmittelbar gefolgt
// von einem zweiten Lauf mit demselben Body, darf beim zweiten Mal nichts mehr wirklich
// verändern: jede Zeile erzeugt genauso viel, wie sie löscht, und keine Zuweisung geht
// verloren (der Slot kommt mit identischem Schlüssel zurück, siehe restoreAssignments).
func TestBulkRegen_ZweiterIdentischerLaufIstIdempotent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRule(t, db, teamID)
	dutyTypeID := insertDutyType(t, db, "Aufbau", 2.0)
	templateID := insertHeimTemplate(t, db, dutyTypeID, -60)
	token := bulkVorstandToken(t, db)
	assigneeID := testutil.CreateUser(t, db, "standard")

	makeGame := func(date string) int {
		res := testutil.Post(t, prodserver.New(t, db), "/api/games", token, map[string]any{
			"date": date, "time": "14:00", "opponent": "FC Test",
			"team_ids": []int{teamID}, "event_type": "heim", "season_id": seasonID,
		})
		var created struct {
			ID int `json:"id"`
		}
		json.NewDecoder(res.Body).Decode(&created)
		res.Body.Close()
		return created.ID
	}

	templateGameID := makeGame(bulkFutureDate(t, 10))
	noneGameID := makeGame(bulkFutureDate(t, 11))
	purgeGameID := makeGame(bulkFutureDate(t, 12))
	db.Exec(`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, team_id, season_id, game_id, is_custom)
		VALUES ('Manuell', ?, ?, 1, ?, ?, ?, 1)`, bulkFutureDate(t, 12), dutyTypeID, teamID, seasonID, purgeGameID)

	body := map[string]any{
		"from": bulkFutureDate(t, 1), "to": bulkFutureDate(t, 30),
		"overrides": []map[string]any{
			{"game_id": templateGameID, "action": "template", "template_id": templateID},
			{"game_id": noneGameID, "action": "none"},
			{"game_id": purgeGameID, "action": "purge"},
		},
	}

	srv := prodserver.New(t, db)
	res1 := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, body)
	got1 := decodeBulkRegen(t, res1)
	res1.Body.Close()
	if res1.StatusCode != http.StatusOK {
		t.Fatalf("first run: expected 200, got %d", res1.StatusCode)
	}

	// The templated game's one auto slot gets an assignment before the second run.
	var slotID int
	db.QueryRow(`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, templateGameID).Scan(&slotID)
	insertDutyAssignment(t, db, slotID, assigneeID, "assigned")

	res2 := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", token, body)
	got2 := decodeBulkRegen(t, res2)
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("second run: expected 200, got %d", res2.StatusCode)
	}

	if got2.Totals.Created != got2.Totals.Deleted {
		t.Errorf("second (idempotent) run: expected created == deleted, got created=%d deleted=%d",
			got2.Totals.Created, got2.Totals.Deleted)
	}
	if got2.Totals.AssignmentsLost != 0 {
		t.Errorf("second (idempotent) run: expected assignments_lost=0, got %d", got2.Totals.AssignmentsLost)
	}
	if got2.Totals.AssignmentsKept != 1 {
		t.Errorf("second (idempotent) run: expected the assignment made after run 1 to be kept, got assignments_kept=%d", got2.Totals.AssignmentsKept)
	}
	if got1.Totals.Games != got2.Totals.Games {
		t.Errorf("expected the same number of games in both runs, got %d vs %d", got1.Totals.Games, got2.Totals.Games)
	}
	// purge is idempotent too: still nothing there the second time around.
	if got := countRows(t, db, "duty_slots", "game_id=?", purgeGameID); got != 0 {
		t.Errorf("expected purge game to have 0 slots after both runs, got %d", got)
	}
}
