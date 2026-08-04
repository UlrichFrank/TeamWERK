package trainingdiary_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainingdiary"
)

// newDiaryServer baut einen Server nur mit den Tagebuch-Routen und liefert das
// Nachweis-Verzeichnis mit zurück — die Datei-Assertions (Orphan-Schutz,
// Ersetzen, Löschen) brauchen einen bekannten Pfad, den prodserver nicht
// herausgibt.
func newDiaryServer(t *testing.T, db *sql.DB) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	h := trainingdiary.NewHandler(db, hub.NewHub(), dir)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-diary", h.ListOwn)
		r.Post("/api/training-diary", h.CreateEntry)
		r.Put("/api/training-diary/{id}", h.UpdateEntry)
		r.Delete("/api/training-diary/{id}", h.DeleteEntry)
		r.Post("/api/training-diary/{id}/proof", h.UploadProof)
		r.Delete("/api/training-diary/{id}/proof", h.DeleteProof)
		r.Get("/api/training-diary/{id}/proof", h.ServeProof)
		r.Get("/api/members/{id}/training-diary", h.GetMemberDiary)
		r.Get("/api/teams/{id}/training-diary-stats", h.GetTeamStats)
	})
	return srv, dir
}

// player legt einen Standard-Nutzer mit Mitglied und Vereinsfunktion `spieler`
// an und liefert beides plus Token. Die Funktion ist Voraussetzung für die
// Neuanlage von Einträgen (siehe TestCreateEntry_NonPlayerForbidden).
func player(t *testing.T, db *sql.DB) (userID, memberID int, token string) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID = testutil.CreateMember(t, db, userID)
	testutil.AddClubFunction(t, db, memberID, "spieler")
	return userID, memberID, testutil.Token(t, userID, "standard", []string{"spieler"})
}

// trainerOf macht einen neuen Nutzer zum Trainer des übergebenen Kaders.
func trainerOf(t *testing.T, db *sql.DB, kaderID int) string {
	t.Helper()
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	testutil.AddClubFunction(t, db, mid, "trainer")
	testutil.AddKaderTrainer(t, db, kaderID, mid)
	return testutil.Token(t, uid, "standard", []string{"trainer"})
}

func validBody() map[string]any {
	return map[string]any{
		"trained_on":   time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		"kind":         "kraft",
		"duration_min": 45,
		"rpe":          7,
	}
}

func decodeEntry(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode entry: %v", err)
	}
	return m
}

// ---------------------------------------------------------------- CRUD

func TestCreateEntry_Success(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)

	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	e := decodeEntry(t, resp)
	if int(e["member_id"].(float64)) != memberID {
		t.Errorf("member_id = %v, want %d", e["member_id"], memberID)
	}
	// Saison-Anker: die AKTIVE Saison, nicht per Datumsvergleich ermittelt.
	if int(e["season_id"].(float64)) != seasonID {
		t.Errorf("season_id = %v, want %d", e["season_id"], seasonID)
	}
	if e["proof_status"] != "none" {
		t.Errorf("proof_status = %v, want none", e["proof_status"])
	}
}

// Ohne aktive Saison bleibt season_id NULL — der Eintrag ist trotzdem gültig
// und wird von der Retention nie angefasst.
func TestCreateEntry_NoActiveSeason(t *testing.T) {
	db := testutil.NewDB(t)
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if e := decodeEntry(t, resp); e["season_id"] != nil {
		t.Errorf("season_id = %v, want null", e["season_id"])
	}
}

// Eine im Body übergebene member_id darf den Eintrag nicht umhängen.
func TestCreateEntry_IgnoresBodyMemberID(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, ownMember, token := player(t, db)
	_, foreignMember, _ := player(t, db)

	body := validBody()
	body["member_id"] = foreignMember
	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp.Body.Close()
	if got := int(decodeEntry(t, resp)["member_id"].(float64)); got != ownMember {
		t.Errorf("member_id = %d, want %d (eigenes Mitglied)", got, ownMember)
	}
}

func TestCreateEntry_CustomKindRequiresText(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	body := validBody()
	body["kind"] = "sonstiges"
	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}

	body["kind_custom"] = "Schwimmen"
	resp2 := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("mit Freitext: status = %d, want 201", resp2.StatusCode)
	}
	if got := decodeEntry(t, resp2)["kind_custom"]; got != "Schwimmen" {
		t.Errorf("kind_custom = %v, want Schwimmen", got)
	}
}

func TestCreateEntry_UnknownKind(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	body := validBody()
	body["kind"] = "yoga"
	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreateEntry_InvalidRPE(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	for _, rpe := range []int{0, 11} {
		body := validBody()
		body["rpe"] = rpe
		resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("rpe=%d: status = %d, want 400", rpe, resp.StatusCode)
		}
	}
	// Die Ränder müssen durchgehen.
	for _, rpe := range []int{1, 10} {
		body := validBody()
		body["rpe"] = rpe
		resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("rpe=%d: status = %d, want 201", rpe, resp.StatusCode)
		}
	}
}

func TestCreateEntry_InvalidDuration(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	for _, d := range []int{0, 1000} {
		body := validBody()
		body["duration_min"] = d
		resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("duration=%d: status = %d, want 400", d, resp.StatusCode)
		}
	}
}

func TestCreateEntry_FutureDate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	body := validBody()
	body["trained_on"] = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

// Heute muss erlaubt sein — sonst kann abends niemand die Einheit von heute
// eintragen.
func TestCreateEntry_TodayAllowed(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	body := validBody()
	body["trained_on"] = time.Now().Format("2006-01-02")
	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
}

func TestCreateEntry_NoMemberForUser(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	userID := testutil.CreateUser(t, db, "standard") // bewusst ohne Mitglied
	// Spieler-Funktion im Token, aber kein Mitglieds-Datensatz — prüft gezielt
	// den resolveOwnMember-Pfad und nicht die vorgelagerte Spieler-Prüfung.
	token := testutil.Token(t, userID, "standard", []string{"spieler"})

	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_diary_entries`).Scan(&n)
	if n != 0 {
		t.Errorf("Einträge = %d, want 0", n)
	}
}

// Ein Mitglieds-Datensatz allein berechtigt nicht zur Erfassung: das Tagebuch
// gehört Spielern. Trainer, Vorstand und Eltern mit eigenem Mitglied bekommen
// 403 — sonst wäre die versteckte Nav-Zeile bloße Kosmetik.
func TestCreateEntry_NonPlayerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	testutil.AddClubFunction(t, db, memberID, "trainer")
	token := testutil.Token(t, userID, "standard", []string{"trainer"})

	resp := testutil.Do(t, srv, http.MethodPost, "/api/training-diary", token, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_diary_entries`).Scan(&n)
	if n != 0 {
		t.Errorf("Einträge = %d, want 0", n)
	}
}

func TestListOwn_OnlyOwnEntries(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, mineID, token := player(t, db)
	_, foreignID, _ := player(t, db)

	testutil.CreateTrainingDiaryEntry(t, db, mineID, seasonID, "2026-05-01", 30, 5)
	testutil.CreateTrainingDiaryEntry(t, db, mineID, seasonID, "2026-05-03", 60, 8)
	testutil.CreateTrainingDiaryEntry(t, db, foreignID, seasonID, "2026-05-02", 45, 6)

	resp := testutil.Do(t, srv, http.MethodGet, "/api/training-diary", token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(out.Items))
	}
	for _, it := range out.Items {
		if int(it["member_id"].(float64)) != mineID {
			t.Errorf("fremder Eintrag in eigener Liste: %v", it["member_id"])
		}
	}
	// Absteigend nach Datum.
	if out.Items[0]["trained_on"].(string)[:10] != "2026-05-03" {
		t.Errorf("Reihenfolge falsch: %v", out.Items[0]["trained_on"])
	}
}

func TestUpdateEntry_Success(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	body := validBody()
	body["duration_min"] = 90
	resp := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/training-diary/%d", id), token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := int(decodeEntry(t, resp)["duration_min"].(float64)); got != 90 {
		t.Errorf("duration_min = %d, want 90", got)
	}
}

// Auch der Trainer des Kaders darf fremde Einträge NICHT ändern — Schreibrecht
// hat ausschließlich der Eigentümer.
func TestUpdateEntry_ForeignEntry(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, ownerMember, _ := player(t, db)
	testutil.AddKaderMember(t, db, kaderID, ownerMember)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	trainerToken := trainerOf(t, db, kaderID)

	resp := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/training-diary/%d", id), trainerToken, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var duration int
	db.QueryRow(`SELECT duration_min FROM training_diary_entries WHERE id = ?`, id).Scan(&duration)
	if duration != 30 {
		t.Errorf("duration_min = %d, want unverändert 30", duration)
	}
}

// Fremde IDs dürfen nicht per Statuscode enumerierbar sein: nicht existierende
// ID → 404, nicht 403.
func TestUpdateEntry_NotFoundBeforeForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	resp := testutil.Do(t, srv, http.MethodPut, "/api/training-diary/9999", token, validBody())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestDeleteEntry_Success(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, dir := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	// Nachweis auf der Platte, damit die Mitlöschung prüfbar ist.
	diskName := "proof-delete.jpg"
	if err := os.WriteFile(filepath.Join(dir, diskName), []byte("x"), 0644); err != nil {
		t.Fatalf("write proof: %v", err)
	}
	testutil.SetTrainingDiaryProof(t, db, id, diskName, "image/jpeg")

	resp := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/training-diary/%d", id), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_diary_entries WHERE id = ?`, id).Scan(&n)
	if n != 0 {
		t.Errorf("Eintrag existiert noch")
	}
	if _, err := os.Stat(filepath.Join(dir, diskName)); !os.IsNotExist(err) {
		t.Errorf("Nachweisdatei wurde nicht entfernt")
	}
}

func TestDeleteEntry_ForeignEntry(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, ownerMember, _ := player(t, db)
	_, _, otherToken := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)

	resp := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/training-diary/%d", id), otherToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_diary_entries WHERE id = ?`, id).Scan(&n)
	if n != 1 {
		t.Errorf("Eintrag wurde gelöscht")
	}
}

// ---------------------------------------------------------- Sichtbarkeit

// kaderPlayer legt einen Spieler an und hängt ihn in den Kader.
func kaderPlayer(t *testing.T, db *sql.DB, kaderID int) (userID, memberID int, token string) {
	t.Helper()
	userID, memberID, token = player(t, db)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	return userID, memberID, token
}

func TestMemberDiary_Owner(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", memberID), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMemberDiary_ParentAccess(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, childMember, _ := player(t, db)
	testutil.CreateTrainingDiaryEntry(t, db, childMember, seasonID, "2026-05-01", 30, 5)

	parentUser := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUser, childMember)
	parentToken := testutil.TokenWithIsParent(t, parentUser, "standard", nil, true)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", childMember), parentToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// Kern-Invariante: Spieler sehen einander nicht — auch nicht im selben Kader.
func TestMemberDiary_OtherPlayer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, targetMember, _ := kaderPlayer(t, db, kaderID)
	_, _, mateToken := kaderPlayer(t, db, kaderID)
	testutil.CreateTrainingDiaryEntry(t, db, targetMember, seasonID, "2026-05-01", 30, 5)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), mateToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestMemberDiary_TrainerOfKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, targetMember, _ := kaderPlayer(t, db, kaderID)
	trainerToken := trainerOf(t, db, kaderID)
	testutil.CreateTrainingDiaryEntry(t, db, targetMember, seasonID, "2026-05-01", 30, 5)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), trainerToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// Auch der erweiterte Kader zählt.
func TestMemberDiary_TrainerOfExtendedKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, targetMember, _ := player(t, db)
	testutil.AddExtendedKaderMember(t, db, kaderID, targetMember)
	trainerToken := trainerOf(t, db, kaderID)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), trainerToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMemberDiary_TrainerForeignTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamA := testutil.CreateTeam(t, db, "Herren 1")
	teamB := testutil.CreateTeam(t, db, "Herren 2")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, targetMember, _ := kaderPlayer(t, db, kaderA)
	foreignTrainer := trainerOf(t, db, kaderB)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), foreignTrainer, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestMemberDiary_SportlicheLeitung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, targetMember, _ := player(t, db)
	testutil.CreateTrainingDiaryEntry(t, db, targetMember, seasonID, "2026-05-01", 30, 5)

	slUser := testutil.CreateUser(t, db, "standard")
	slMember := testutil.CreateMember(t, db, slUser)
	testutil.AddClubFunction(t, db, slMember, "sportliche_leitung")
	slToken := testutil.Token(t, slUser, "standard", []string{"sportliche_leitung"})

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), slToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// vorstand ist bewusst NICHT leseberechtigt — gleichgezogen mit der
// Anwesenheitsstatistik.
func TestMemberDiary_VorstandForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, targetMember, _ := player(t, db)
	testutil.CreateTrainingDiaryEntry(t, db, targetMember, seasonID, "2026-05-01", 30, 5)

	vUser := testutil.CreateUser(t, db, "standard")
	vMember := testutil.CreateMember(t, db, vUser)
	testutil.AddClubFunction(t, db, vMember, "vorstand")
	vToken := testutil.Token(t, vUser, "standard", []string{"vorstand"})

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/members/%d/training-diary", targetMember), vToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// ------------------------------------------------------------ Team-Stats

func TestTeamStats_TrainerOwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, activeMember, _ := kaderPlayer(t, db, kaderID)
	_, idleMember, _ := kaderPlayer(t, db, kaderID)
	testutil.CreateTrainingDiaryEntry(t, db, activeMember, seasonID, "2026-05-01", 60, 6)
	testutil.CreateTrainingDiaryEntry(t, db, activeMember, seasonID, "2026-05-03", 30, 8)
	trainerToken := trainerOf(t, db, kaderID)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/teams/%d/training-diary-stats", teamID), trainerToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Items []struct {
			MemberID int     `json:"member_id"`
			Entries  int     `json:"entries"`
			Minutes  int     `json:"minutes"`
			AvgRPE   float64 `json:"avg_rpe"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&out)

	byID := map[int]int{}
	for i, it := range out.Items {
		byID[it.MemberID] = i
	}
	a, ok := byID[activeMember]
	if !ok {
		t.Fatalf("aktives Mitglied fehlt in der Übersicht")
	}
	if out.Items[a].Entries != 2 || out.Items[a].Minutes != 90 || out.Items[a].AvgRPE != 7 {
		t.Errorf("Aggregat = %+v, want entries=2 minutes=90 avg=7", out.Items[a])
	}
	// Mitglieder ohne Einträge müssen mit Nullwerten erscheinen.
	i, ok := byID[idleMember]
	if !ok {
		t.Fatalf("Mitglied ohne Einträge fehlt in der Übersicht")
	}
	if out.Items[i].Entries != 0 || out.Items[i].Minutes != 0 {
		t.Errorf("Nullzeile = %+v, want entries=0 minutes=0", out.Items[i])
	}
}

func TestTeamStats_PlayerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)
	_, _, playerToken := kaderPlayer(t, db, kaderID)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/teams/%d/training-diary-stats", teamID), playerToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestTeamStats_TrainerForeignTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamA := testutil.CreateTeam(t, db, "Herren 1")
	teamB := testutil.CreateTeam(t, db, "Herren 2")
	testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	srv, _ := newDiaryServer(t, db)
	foreignTrainer := trainerOf(t, db, kaderB)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/teams/%d/training-diary-stats", teamA), foreignTrainer, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// Ohne aktive Saison: leere Liste statt Fehler.
func TestTeamStats_NoActiveSeason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)
	// Über admin geprüft: die Trainer-Auflösung selbst braucht eine aktive
	// Saison und würde hier zwangsläufig 403 liefern — geprüft werden soll
	// aber der leere Auswertungszeitraum, nicht die ACL.
	adminUser := testutil.CreateUser(t, db, "admin")
	adminToken := testutil.Token(t, adminUser, "admin", nil)
	if _, err := db.Exec(`UPDATE seasons SET is_active = 0`); err != nil {
		t.Fatalf("deactivate season: %v", err)
	}

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/teams/%d/training-diary-stats", teamID), adminToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Items) != 0 {
		t.Errorf("items = %d, want 0", len(out.Items))
	}
}

// -------------------------------------------------------------- Nachweis

// uploadProof sendet eine multipart-Anfrage mit den übergebenen Bytes.
func uploadProof(t *testing.T, srv *httptest.Server, entryID int, token string, filename string, content []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("proof", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/training-diary/%d/proof", srv.URL, entryID), &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do upload: %v", err)
	}
	return resp
}

// jpegBytes liefert Bytes, die http.DetectContentType als image/jpeg erkennt.
func jpegBytes(padding int) []byte {
	b := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	return append(b, bytes.Repeat([]byte{0x00}, padding)...)
}

func TestUploadProof_Success(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, dir := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	resp := uploadProof(t, srv, id, token, "beleg.jpg", jpegBytes(100))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201 (%s)", resp.StatusCode, body)
	}
	e := decodeEntry(t, resp)
	if e["proof_status"] != "present" {
		t.Errorf("proof_status = %v, want present", e["proof_status"])
	}

	var diskName string
	db.QueryRow(`SELECT proof_disk_name FROM training_diary_entries WHERE id = ?`, id).Scan(&diskName)
	if diskName == "" {
		t.Fatalf("proof_disk_name leer")
	}
	if _, err := os.Stat(filepath.Join(dir, diskName)); err != nil {
		t.Errorf("Datei fehlt auf der Platte: %v", err)
	}
}

func TestUploadProof_UnsupportedType(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, dir := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	// HEIC-Signatur (ftypheic im ISOBMFF-Header) — genau der Fall, den die
	// clientseitige Kompression auf Chrome/Firefox nicht wandeln kann.
	heic := append([]byte{0x00, 0x00, 0x00, 0x18}, []byte("ftypheic")...)
	resp := uploadProof(t, srv, id, token, "foto.heic", append(heic, bytes.Repeat([]byte{0}, 64)...))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("Datei wurde trotz Ablehnung geschrieben: %v", entries)
	}
}

func TestUploadProof_TooLarge(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	resp := uploadProof(t, srv, id, token, "gross.jpg", jpegBytes(2<<20))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
}

func TestUploadProof_ForeignEntry(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, dir := newDiaryServer(t, db)

	_, ownerMember, _ := kaderPlayer(t, db, kaderID)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	trainerToken := trainerOf(t, db, kaderID)

	resp := uploadProof(t, srv, id, trainerToken, "beleg.jpg", jpegBytes(100))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("Datei wurde geschrieben: %v", entries)
	}
}

func TestUploadProof_ReplacesOld(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, dir := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	first := uploadProof(t, srv, id, token, "a.jpg", jpegBytes(100))
	first.Body.Close()
	var firstName string
	db.QueryRow(`SELECT proof_disk_name FROM training_diary_entries WHERE id = ?`, id).Scan(&firstName)

	second := uploadProof(t, srv, id, token, "b.jpg", jpegBytes(200))
	second.Body.Close()
	var secondName string
	db.QueryRow(`SELECT proof_disk_name FROM training_diary_entries WHERE id = ?`, id).Scan(&secondName)

	if firstName == secondName {
		t.Fatalf("Dateiname unverändert — Ersetzen hat nicht gegriffen")
	}
	if _, err := os.Stat(filepath.Join(dir, firstName)); !os.IsNotExist(err) {
		t.Errorf("alte Datei liegt noch da (Waisen-Blob)")
	}
	if _, err := os.Stat(filepath.Join(dir, secondName)); err != nil {
		t.Errorf("neue Datei fehlt: %v", err)
	}
}

// Ein erneuter Upload nach der Retention hebt den purged-Marker auf.
func TestUploadProof_ResetsPurgedMarker(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)
	if _, err := db.Exec(
		`UPDATE training_diary_entries SET proof_purged_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		t.Fatalf("mark purged: %v", err)
	}

	resp := uploadProof(t, srv, id, token, "neu.jpg", jpegBytes(100))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if got := decodeEntry(t, resp)["proof_status"]; got != "present" {
		t.Errorf("proof_status = %v, want present", got)
	}
}

func TestDeleteProof_Success(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, dir := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, token, "a.jpg", jpegBytes(100))
	up.Body.Close()
	var diskName string
	db.QueryRow(`SELECT proof_disk_name FROM training_diary_entries WHERE id = ?`, id).Scan(&diskName)

	resp := testutil.Do(t, srv, http.MethodDelete, fmt.Sprintf("/api/training-diary/%d/proof", id), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, diskName)); !os.IsNotExist(err) {
		t.Errorf("Datei liegt noch da")
	}
	// Der Eintrag selbst bleibt unangetastet.
	var duration int
	db.QueryRow(`SELECT duration_min FROM training_diary_entries WHERE id = ?`, id).Scan(&duration)
	if duration != 30 {
		t.Errorf("duration_min = %d, want 30", duration)
	}
}

func TestServeProof_Owner(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, token, "a.jpg", jpegBytes(100))
	up.Body.Close()

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("nosniff-Header fehlt")
	}
}

func TestServeProof_TrainerOfKader(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, ownerMember, ownerToken := kaderPlayer(t, db, kaderID)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, ownerToken, "a.jpg", jpegBytes(100))
	up.Body.Close()
	trainerToken := trainerOf(t, db, kaderID)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), trainerToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// Eltern sehen den Nachweis ihres Kindes — der Fall hinter der Kind-Profilseite
// (/profil/kind/{id}, Tab „Trainingstagebuch"). Nicht nur die Metadaten aus
// GetMemberDiary, sondern die Datei selbst.
func TestServeProof_Parent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, childMember, childToken := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, childMember, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, childToken, "a.jpg", jpegBytes(100))
	up.Body.Close()

	parentUser := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUser, childMember)
	parentToken := testutil.TokenWithIsParent(t, parentUser, "standard", nil, true)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), parentToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
}

// Kern-Invariante auf dem Datei-Pfad: Mannschaftskameraden sehen den Nachweis nicht.
func TestServeProof_OtherPlayer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, ownerMember, ownerToken := kaderPlayer(t, db, kaderID)
	_, _, mateToken := kaderPlayer(t, db, kaderID)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, ownerToken, "a.jpg", jpegBytes(100))
	up.Body.Close()

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), mateToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestServeProof_TrainerOtherTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	teamA := testutil.CreateTeam(t, db, "Herren 1")
	teamB := testutil.CreateTeam(t, db, "Herren 2")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	srv, _ := newDiaryServer(t, db)

	_, ownerMember, ownerToken := kaderPlayer(t, db, kaderA)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	up := uploadProof(t, srv, id, ownerToken, "a.jpg", jpegBytes(100))
	up.Body.Close()
	foreignTrainer := trainerOf(t, db, kaderB)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), foreignTrainer, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// 410 grenzt „von der Retention gelöscht" von 404 („nie einer da") ab.
func TestServeProof_Purged(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)
	if _, err := db.Exec(
		`UPDATE training_diary_entries SET proof_purged_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		t.Fatalf("mark purged: %v", err)
	}

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("status = %d, want 410", resp.StatusCode)
	}
}

// Die ACL greift VOR der 410/404-Unterscheidung — sonst verriete der
// Statuscode Unberechtigten den Zustand eines fremden Eintrags.
func TestServeProof_PurgedButForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, ownerMember, _ := player(t, db)
	_, _, otherToken := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, ownerMember, seasonID, "2026-05-01", 30, 5)
	if _, err := db.Exec(
		`UPDATE training_diary_entries SET proof_purged_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		t.Fatalf("mark purged: %v", err)
	}

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), otherToken, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (nicht 410 — kein Zustandsleck)", resp.StatusCode)
	}
}

// Eine nicht existierende Member-ID liefert 403, nicht 500 und nicht 404.
func TestMemberDiary_UnknownMember(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, _, token := player(t, db)

	resp := testutil.Do(t, srv, http.MethodGet, "/api/members/99999/training-diary", token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestServeProof_NeverHadOne(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "25/26")
	srv, _ := newDiaryServer(t, db)
	_, memberID, token := player(t, db)
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 30, 5)

	resp := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/training-diary/%d/proof", id), token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
