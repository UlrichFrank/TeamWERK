package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// --- spiel-treffpunkt: PUT /api/games/{id}/meeting --------------------------

type meetingFixture struct {
	db       *sql.DB
	seasonID int
	teamID   int
	gameID   int // Anwurf 18:00 (testutil.CreateGame)
}

func newMeetingFixture(t *testing.T) meetingFixture {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-01-15")
	return meetingFixture{db, seasonID, teamID, gameID}
}

func putMeeting(t *testing.T, f meetingFixture, token string, body map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	srv := testServer(t, f.db)
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/games/%d/meeting", f.gameID), token, body)
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func storedMeeting(t *testing.T, f meetingFixture) (sql.NullInt64, string) {
	t.Helper()
	var off sql.NullInt64
	var place string
	if err := f.db.QueryRow(`SELECT meet_offset_minutes, meet_place FROM games WHERE id=?`, f.gameID).Scan(&off, &place); err != nil {
		t.Fatalf("read meeting: %v", err)
	}
	return off, place
}

func TestUpdateGameMeeting_TrainerSetztTreffzeit(t *testing.T) {
	f := newMeetingFixture(t)
	trainer := makeTrainer(t, f.db, f.teamID, f.seasonID)
	token := testutil.Token(t, trainer, "standard", []string{"trainer"})

	res, out := putMeeting(t, f, token, map[string]string{"meet_time": "16:30", "meet_place": "  Parkplatz Vereinsheim "})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%v)", res.StatusCode, out)
	}
	if out["meet_time"] != "16:30" || out["meet_date"] != "2099-01-15" || out["meet_place"] != "Parkplatz Vereinsheim" {
		t.Errorf("unexpected response %v", out)
	}
	off, place := storedMeeting(t, f)
	if !off.Valid || off.Int64 != 90 || place != "Parkplatz Vereinsheim" {
		t.Errorf("stored offset=%v place=%q, want 90 / Parkplatz Vereinsheim", off, place)
	}
	var prev sql.NullInt64
	var n int
	f.db.QueryRow(`SELECT COUNT(*), MAX(prev_offset) FROM pending_game_meeting_push WHERE game_id=?`, f.gameID).Scan(&n, &prev)
	if n != 1 || prev.Valid {
		t.Errorf("pending row: n=%d prev=%v, want 1 row with prev NULL", n, prev)
	}

}

// Der Ausgangsstand des Debounce-Fensters bleibt bei weiteren Korrekturen stehen.
func TestUpdateGameMeeting_KorrekturBehaeltAusgangsstand(t *testing.T) {
	f := newMeetingFixture(t)
	f.db.Exec(`UPDATE games SET meet_offset_minutes=60, meet_place='Halle' WHERE id=?`, f.gameID)
	admin := testutil.CreateUser(t, f.db, "admin")
	token := testutil.Token(t, admin, "admin", nil)

	putMeeting(t, f, token, map[string]string{"meet_time": "16:30"})
	putMeeting(t, f, token, map[string]string{"meet_time": "16:00"})

	var prev sql.NullInt64
	var prevPlace string
	f.db.QueryRow(`SELECT prev_offset, prev_place FROM pending_game_meeting_push WHERE game_id=?`, f.gameID).Scan(&prev, &prevPlace)
	if !prev.Valid || prev.Int64 != 60 || prevPlace != "Halle" {
		t.Errorf("prev=%v %q, want 60 / Halle", prev, prevPlace)
	}
}

func TestUpdateGameMeeting_FremderTrainer(t *testing.T) {
	f := newMeetingFixture(t)
	other := testutil.CreateTeam(t, f.db, "Team B")
	trainer := makeTrainer(t, f.db, other, f.seasonID)
	token := testutil.Token(t, trainer, "standard", []string{"trainer"})

	res, _ := putMeeting(t, f, token, map[string]string{"meet_time": "16:30"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if off, _ := storedMeeting(t, f); off.Valid {
		t.Errorf("offset changed to %d", off.Int64)
	}
}

func TestUpdateGameMeeting_SpielerVerboten(t *testing.T) {
	f := newMeetingFixture(t)
	user := testutil.CreateUser(t, f.db, "standard")
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res, _ := putMeeting(t, f, token, map[string]string{"meet_time": "16:30"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestUpdateGameMeeting_SportlicheLeitung(t *testing.T) {
	f := newMeetingFixture(t)
	user := testutil.CreateUser(t, f.db, "standard")
	token := testutil.Token(t, user, "standard", []string{"sportliche_leitung"})
	res, _ := putMeeting(t, f, token, map[string]string{"meet_time": "16:30"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestUpdateGameMeeting_Validierung(t *testing.T) {
	cases := []struct {
		name string
		body map[string]string
		code string
	}{
		{"NachAnwurf", map[string]string{"meet_time": "18:30"}, "meet_after_start"},
		{"ZuWeitVorher", map[string]string{"meet_time": "05:59"}, "meet_offset_out_of_range"},
		{"OrtOhneZeit", map[string]string{"meet_time": "", "meet_place": "Halle"}, "meet_place_without_time"},
		{"OrtZuLang", map[string]string{"meet_time": "17:00", "meet_place": strings.Repeat("ä", 101)}, "meet_place_too_long"},
		{"Format", map[string]string{"meet_time": "17"}, "validation"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newMeetingFixture(t)
			f.db.Exec(`UPDATE games SET meet_offset_minutes=30, meet_place='Halle' WHERE id=?`, f.gameID)
			admin := testutil.CreateUser(t, f.db, "admin")
			token := testutil.Token(t, admin, "admin", nil)

			res, out := putMeeting(t, f, token, c.body)
			if res.StatusCode != http.StatusBadRequest || out["error"] != c.code {
				t.Fatalf("expected 400 %s, got %d %v", c.code, res.StatusCode, out)
			}
			off, place := storedMeeting(t, f)
			if !off.Valid || off.Int64 != 30 || place != "Halle" {
				t.Errorf("stored values changed: %v %q", off, place)
			}
			var n int
			f.db.QueryRow(`SELECT COUNT(*) FROM pending_game_meeting_push`).Scan(&n)
			if n != 0 {
				t.Errorf("pending row written on rejected request")
			}
		})
	}
}

func TestUpdateGameMeeting_Entfernen(t *testing.T) {
	f := newMeetingFixture(t)
	f.db.Exec(`UPDATE games SET meet_offset_minutes=30, meet_place='Halle' WHERE id=?`, f.gameID)
	admin := testutil.CreateUser(t, f.db, "admin")
	token := testutil.Token(t, admin, "admin", nil)

	res, out := putMeeting(t, f, token, map[string]string{"meet_time": "", "meet_place": ""})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if out["meet_time"] != nil || out["meet_place"] != "" {
		t.Errorf("unexpected response %v", out)
	}
	off, place := storedMeeting(t, f)
	if off.Valid || place != "" {
		t.Errorf("stored %v %q, want NULL / empty", off, place)
	}
}

func TestUpdateGameMeeting_Unbekannt(t *testing.T) {
	f := newMeetingFixture(t)
	admin := testutil.CreateUser(t, f.db, "admin")
	token := testutil.Token(t, admin, "admin", nil)
	srv := testServer(t, f.db)
	res := testutil.Do(t, srv, http.MethodPut, "/api/games/99999/meeting", token, map[string]string{"meet_time": "17:00"})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// Die Treffzeit hängt am Anwurf: eine Verlegung über PUT /api/games/{id}
// verschiebt sie mit, ohne dass jemand sie neu setzt. Liste und Detail liefern
// denselben abgeleiteten Wert.
func TestMeetingFolgtVerlegtemAnwurf(t *testing.T) {
	f := newMeetingFixture(t)
	admin := testutil.CreateUser(t, f.db, "admin")
	token := testutil.Token(t, admin, "admin", nil)

	if res, _ := putMeeting(t, f, token, map[string]string{"meet_time": "16:30", "meet_place": "Bus"}); res.StatusCode != http.StatusOK {
		t.Fatalf("set meeting: %d", res.StatusCode)
	}
	// Anwurf wie ein H4A-Import direkt verlegen (der Import schreibt nur time).
	if _, err := f.db.Exec(`UPDATE games SET time='20:00' WHERE id=?`, f.gameID); err != nil {
		t.Fatal(err)
	}

	srv := testServer(t, f.db)
	get := func(path string) []byte {
		res := testutil.Do(t, srv, http.MethodGet, path, token, nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: %d", path, res.StatusCode)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	var wrap struct {
		Game map[string]any `json:"game"`
	}
	json.Unmarshal(get(fmt.Sprintf("/api/games/%d", f.gameID)), &wrap)
	detail := wrap.Game
	if detail["meet_time"] != "18:30" || detail["meet_place"] != "Bus" {
		t.Errorf("detail: meet_time=%v meet_place=%v, want 18:30 / Bus", detail["meet_time"], detail["meet_place"])
	}

	var list struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(get("/api/games?season_id="+fmt.Sprint(f.seasonID)), &list)
	found := false
	for _, it := range list.Items {
		if int(it["id"].(float64)) == f.gameID {
			found = true
			if it["meet_time"] != "18:30" {
				t.Errorf("list: meet_time=%v, want 18:30", it["meet_time"])
			}
		}
	}
	if !found {
		t.Errorf("game not in list")
	}
}
