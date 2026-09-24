package dashboard_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Aushilfe-Block und getrennte Bilanz (Change dienste-erweiterter-kader).

type aushilfeBody struct {
	MeineDienste struct {
		NextGame *struct {
			ID int `json:"id"`
		} `json:"nextGame"`
		MySlots []struct {
			DutyTypeName string `json:"dutyTypeName"`
		} `json:"mySlots"`
		Aushilfe *struct {
			NextGame *struct {
				ID int `json:"id"`
			} `json:"nextGame"`
			TeamLabel      string `json:"teamLabel"`
			OpenSlotsCount int    `json:"openSlotsCount"`
			MySlots        []struct {
				DutyTypeName string `json:"dutyTypeName"`
				Date         string `json:"date"`
			} `json:"mySlots"`
		} `json:"aushilfe"`
		DutyAccount []struct {
			TeamID    int     `json:"teamId"`
			Geleistet float64 `json:"geleistet"`
		} `json:"dutyAccount"`
		DutyAccountAushilfe []map[string]any `json:"dutyAccountAushilfe"`
	} `json:"meineDienste"`
}

func loadDashboard(t *testing.T, db *sql.DB, userID int) aushilfeBody {
	t.Helper()
	srv := testServer(t, dashboard.NewHandler(db))
	res := testutil.Get(t, srv, "/api/dashboard", testutil.Token(t, userID, "standard", []string{"spieler"}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body aushilfeBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
}

type aushilfeSetup struct {
	db                   *sql.DB
	season               int
	stammTeam, erwTeam   int
	stammKader, erwKader int
	user, member         int
}

func newAushilfeSetup(t *testing.T) *aushilfeSetup {
	t.Helper()
	db := testutil.NewDB(t)
	s := &aushilfeSetup{db: db}
	s.season = testutil.CreateSeason(t, db, "2025/26")
	s.stammTeam = testutil.CreateTeam(t, db, "mC1")
	s.erwTeam = testutil.CreateTeam(t, db, "mB1")
	s.stammKader = testutil.CreateKader(t, db, s.stammTeam, s.season)
	s.erwKader = testutil.CreateKader(t, db, s.erwTeam, s.season)
	s.user = testutil.CreateUser(t, db, "standard")
	s.member = testutil.CreateMember(t, db, s.user)
	testutil.AddKaderMember(t, db, s.stammKader, s.member)
	testutil.AddExtendedKaderMember(t, db, s.erwKader, s.member)
	return s
}

func dayOffset(n int) string { return time.Now().AddDate(0, 0, n).Format("2006-01-02") }

func TestDashboard_MeineDienste_AushilfeBlock(t *testing.T) {
	s := newAushilfeSetup(t)
	dt := testutil.CreateDutyType(t, s.db, "Bewirtung", 1.0)

	erwGame := testutil.CreateGame(t, s.db, s.season, s.erwTeam, dayOffset(1))
	stammGame := testutil.CreateGame(t, s.db, s.season, s.stammTeam, dayOffset(7))
	testutil.CreateDutySlot(t, s.db, dt, s.season, s.erwTeam, erwGame, dayOffset(1)) // 2 Plätze
	testutil.CreateDutySlot(t, s.db, dt, s.season, s.stammTeam, stammGame, dayOffset(7))

	// Eigene Aushilfe-Zusage auf einem weiteren Spiel des erweiterten Teams.
	erwGame2 := testutil.CreateGame(t, s.db, s.season, s.erwTeam, dayOffset(3))
	taken := testutil.CreateDutySlot(t, s.db, dt, s.season, s.erwTeam, erwGame2, dayOffset(3))
	s.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?)`, taken, s.user)
	s.db.Exec(`UPDATE duty_slots SET slots_filled=2 WHERE id=?`, taken)

	body := loadDashboard(t, s.db, s.user)
	md := body.MeineDienste

	if md.NextGame == nil || md.NextGame.ID != stammGame {
		t.Fatalf("Stamm-Block muss beim Stammteam bleiben, nextGame=%+v", md.NextGame)
	}
	if len(md.MySlots) != 0 {
		t.Errorf("Aushilfe-Zusage darf nicht im Stamm-Block stehen: %+v", md.MySlots)
	}
	if md.Aushilfe == nil {
		t.Fatal("Aushilfe-Block fehlt")
	}
	if md.Aushilfe.NextGame == nil || md.Aushilfe.NextGame.ID != erwGame {
		t.Errorf("aushilfe.nextGame = %+v, want Spiel %d", md.Aushilfe.NextGame, erwGame)
	}
	if md.Aushilfe.OpenSlotsCount != 2 {
		t.Errorf("aushilfe.openSlotsCount = %d, want 2", md.Aushilfe.OpenSlotsCount)
	}
	if md.Aushilfe.TeamLabel == "" {
		t.Error("aushilfe.teamLabel leer")
	}
	if len(md.Aushilfe.MySlots) != 1 || md.Aushilfe.MySlots[0].Date != dayOffset(3) {
		t.Errorf("aushilfe.mySlots = %+v, want eine Zusage am %s", md.Aushilfe.MySlots, dayOffset(3))
	}
}

func TestDashboard_MeineDienste_OhneErweitertenKaderKeinBlock(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "mC1")
	kader := testutil.CreateKader(t, db, team, season)
	user := testutil.CreateUser(t, db, "standard")
	testutil.AddKaderMember(t, db, kader, testutil.CreateMember(t, db, user))
	dt := testutil.CreateDutyType(t, db, "Kasse", 1.0)
	game := testutil.CreateGame(t, db, season, team, dayOffset(2))
	testutil.CreateDutySlot(t, db, dt, season, team, game, dayOffset(2))

	body := loadDashboard(t, db, user)
	if body.MeineDienste.Aushilfe != nil {
		t.Errorf("ohne erweiterten Kader muss aushilfe null sein, bekam %+v", body.MeineDienste.Aushilfe)
	}
	if len(body.MeineDienste.DutyAccountAushilfe) != 0 {
		t.Errorf("dutyAccountAushilfe muss leer sein, bekam %v", body.MeineDienste.DutyAccountAushilfe)
	}
}

func TestDashboard_DutyAccountAushilfe(t *testing.T) {
	s := newAushilfeSetup(t)
	dt := testutil.CreateDutyType(t, s.db, "Kasse", 1.0)
	past := testutil.CreateGame(t, s.db, s.season, s.erwTeam, dayOffset(-3))
	slot := testutil.CreateDutySlot(t, s.db, dt, s.season, s.erwTeam, past, dayOffset(-3))
	s.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?)`, slot, s.user)

	md := loadDashboard(t, s.db, s.user).MeineDienste
	for _, e := range md.DutyAccount {
		if e.Geleistet != 0 {
			t.Errorf("Stamm-Bilanz Team %d geleistet = %v, want 0", e.TeamID, e.Geleistet)
		}
	}
	if len(md.DutyAccountAushilfe) != 1 {
		t.Fatalf("dutyAccountAushilfe = %v, want eine Position", md.DutyAccountAushilfe)
	}
	e := md.DutyAccountAushilfe[0]
	if e["geleistet"] != 1.0 || int(e["teamId"].(float64)) != s.erwTeam {
		t.Errorf("Aushilfe-Position = %v, want geleistet 1 für Team %d", e, s.erwTeam)
	}
	if _, hasSoll := e["soll"]; hasSoll {
		t.Error("Aushilfe-Position darf kein soll tragen")
	}
}

// Beide Blöcke liefern die Slot-ID (Sprungziel /dienste?focus=slot-<id>) und
// die Mannschaft des Slots — auch der Stamm-Block, nicht nur die Aushilfe.
func TestDashboard_MeineDienste_SlotIDUndMannschaft(t *testing.T) {
	s := newAushilfeSetup(t)
	dt := testutil.CreateDutyType(t, s.db, "Bewirtung", 1.0)

	stammGame := testutil.CreateGame(t, s.db, s.season, s.stammTeam, dayOffset(2))
	stammSlot := testutil.CreateDutySlot(t, s.db, dt, s.season, s.stammTeam, stammGame, dayOffset(2))
	erwGame := testutil.CreateGame(t, s.db, s.season, s.erwTeam, dayOffset(3))
	erwSlot := testutil.CreateDutySlot(t, s.db, dt, s.season, s.erwTeam, erwGame, dayOffset(3))
	for _, id := range []int{stammSlot, erwSlot} {
		if _, err := s.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?)`, id, s.user); err != nil {
			t.Fatal(err)
		}
	}

	srv := testServer(t, dashboard.NewHandler(s.db))
	res := testutil.Get(t, srv, "/api/dashboard", testutil.Token(t, s.user, "standard", []string{"spieler"}))
	defer res.Body.Close()
	var body struct {
		MeineDienste struct {
			MySlots []struct {
				SlotID    int    `json:"slotId"`
				TeamLabel string `json:"teamLabel"`
			} `json:"mySlots"`
			Aushilfe *struct {
				MySlots []struct {
					SlotID    int    `json:"slotId"`
					TeamLabel string `json:"teamLabel"`
				} `json:"mySlots"`
			} `json:"aushilfe"`
		} `json:"meineDienste"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	md := body.MeineDienste
	if len(md.MySlots) != 1 || md.MySlots[0].SlotID != stammSlot || md.MySlots[0].TeamLabel == "" {
		t.Errorf("Stamm-Slot = %+v, want slotId=%d mit Mannschaft", md.MySlots, stammSlot)
	}
	if md.Aushilfe == nil || len(md.Aushilfe.MySlots) != 1 || md.Aushilfe.MySlots[0].SlotID != erwSlot {
		t.Fatalf("Aushilfe-Slot fehlt oder falsche ID: %+v", md.Aushilfe)
	}
	if md.MySlots[0].TeamLabel == md.Aushilfe.MySlots[0].TeamLabel {
		t.Errorf("Stamm- und Aushilfe-Slot tragen dieselbe Mannschaft %q", md.MySlots[0].TeamLabel)
	}
}
