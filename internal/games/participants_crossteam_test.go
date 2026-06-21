package games_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// crossTeamFixture richtet ein generisches Event mit zwei Teams ein. Liefert
// game_id sowie die Member-IDs in Team A (Spieler+Extended) und Team B.
type crossTeamFixture struct {
	gameID                  int
	teamA, teamB            int
	memberTeamA             int // regulär
	memberTeamAExtended     int
	memberTeamB             int // regulär
	memberTeamBExtended     int
	memberTeamBExtendedOpt  int // mit cross_team_visible=1
	userOfMemberA, userOfMB int
}

func newCrossTeamFixture(t *testing.T, db *sql.DB) crossTeamFixture {
	t.Helper()
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	tA := mkTeamCustom(t, db, "TS A", "Erwachsene", "m")
	tB := mkTeamCustom(t, db, "TS B", "Erwachsene", "m")
	kA := mkKaderCustomReturn(t, db, seasonID, tA, "Erwachsene", "m", 1)
	kB := mkKaderCustomReturn(t, db, seasonID, tB, "Erwachsene", "m", 2)

	uA := testutil.CreateUser(t, db, "standard")
	mA := testutil.CreateMember(t, db, uA)
	mAExt := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	uB := testutil.CreateUser(t, db, "standard")
	mB := testutil.CreateMember(t, db, uB)
	mBExt := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	mBExtOpt := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))

	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kA, mA)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kA, mAExt)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kB, mB)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kB, mBExt)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kB, mBExtOpt)
	db.Exec(`UPDATE members SET cross_team_visible=1 WHERE id=?`, mBExtOpt)

	gameID := testutil.CreateGame(t, db, seasonID, tA, "2026-04-12")
	db.Exec(`UPDATE games SET event_type='generisch' WHERE id=?`, gameID)
	db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, tB)

	return crossTeamFixture{
		gameID: gameID, teamA: tA, teamB: tB,
		memberTeamA: mA, memberTeamAExtended: mAExt,
		memberTeamB: mB, memberTeamBExtended: mBExt, memberTeamBExtendedOpt: mBExtOpt,
		userOfMemberA: uA, userOfMB: uB,
	}
}

// decodeParticipants entpackt die {items, hidden_team_ids}-Antwort.
func decodeParticipants(t *testing.T, res *http.Response) (items []map[string]any, hidden []int) {
	t.Helper()
	defer res.Body.Close()
	var resp struct {
		Items         []map[string]any `json:"items"`
		HiddenTeamIDs []int            `json:"hidden_team_ids"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Items, resp.HiddenTeamIDs
}

func memberIDsByTeam(items []map[string]any) map[int][]int {
	out := map[int][]int{}
	for _, it := range items {
		tid := int(it["team_id"].(float64))
		mid := int(it["member_id"].(float64))
		out[tid] = append(out[tid], mid)
	}
	return out
}

// TestGetParticipants_MultiTeam_SpielerSiehtNurEigenesTeam — Spieler in Team A
// bei Event mit Teams A+B sieht nur Member aus Team A (+ Opt-In aus B).
func TestGetParticipants_MultiTeam_SpielerSiehtNurEigenesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	// Spieler ist in Team A
	srv := testServer(t, db)
	token := testutil.Token(t, fx.userOfMemberA, "standard", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	items, hidden := decodeParticipants(t, res)
	by := memberIDsByTeam(items)

	// Team A: regulär + extended sichtbar.
	if got := len(by[fx.teamA]); got != 2 {
		t.Errorf("Team A: erwartet 2 Member, got %d", got)
	}
	// Team B: nur das Opt-In-Mitglied sichtbar.
	if got := len(by[fx.teamB]); got != 1 {
		t.Errorf("Team B: erwartet 1 (nur Opt-In), got %d", got)
	}
	if len(by[fx.teamB]) == 1 && by[fx.teamB][0] != fx.memberTeamBExtendedOpt {
		t.Errorf("Team B sichtbares Mitglied muss Opt-In sein, got %v", by[fx.teamB])
	}
	// Team B muss als "hidden" markiert sein (es wurden Mitglieder verborgen).
	foundHidden := false
	for _, h := range hidden {
		if h == fx.teamB {
			foundHidden = true
		}
	}
	if !foundHidden {
		t.Errorf("Team B sollte in hidden_team_ids stehen, got %v", hidden)
	}
}

// TestGetParticipants_MultiTeam_OptInMachtFremdSichtbar — wenn alle Member aus
// Team B Opt-In haben, ist die Liste vollständig und Team B nicht als hidden
// markiert.
func TestGetParticipants_MultiTeam_OptInMachtFremdSichtbar(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	// Alle Team-B-Member auf Opt-In setzen.
	db.Exec(`UPDATE members SET cross_team_visible=1 WHERE id IN (?,?,?)`,
		fx.memberTeamB, fx.memberTeamBExtended, fx.memberTeamBExtendedOpt)

	srv := testServer(t, db)
	token := testutil.Token(t, fx.userOfMemberA, "standard", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	items, hidden := decodeParticipants(t, res)
	by := memberIDsByTeam(items)
	if got := len(by[fx.teamB]); got != 3 {
		t.Errorf("Team B: alle Opt-In, erwartet 3, got %d", got)
	}
	for _, h := range hidden {
		if h == fx.teamB {
			t.Errorf("Team B sollte NICHT als hidden markiert sein, wenn nichts gefiltert wurde")
		}
	}
}

// TestGetParticipants_MultiTeam_ElternSehenTeamsDerKinder — Elternteil (kein
// eigenes Member) eines Kindes in Team A sieht Team A (+ Opt-In aus B).
func TestGetParticipants_MultiTeam_ElternSehenTeamsDerKinder(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	parentUID := testutil.CreateUser(t, db, "standard")
	// Eltern-Link: parent → child member (mA in Team A)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUID, fx.memberTeamA); err != nil {
		t.Fatalf("family_links insert: %v", err)
	}

	srv := testServer(t, db)
	token := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	items, _ := decodeParticipants(t, res)
	by := memberIDsByTeam(items)
	if got := len(by[fx.teamA]); got != 2 {
		t.Errorf("Eltern: Team A erwartet 2 Member, got %d", got)
	}
	if got := len(by[fx.teamB]); got != 1 {
		t.Errorf("Eltern: Team B erwartet 1 (Opt-In), got %d", got)
	}
}

// TestGetParticipants_MultiTeam_KindIn2TeamsSiehtBeide — ein Member, das in
// Kader von A UND B steht, sieht beide Sektionen vollständig.
func TestGetParticipants_MultiTeam_KindIn2TeamsSiehtBeide(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	// memberTeamA zusätzlich in Kader von Team B aufnehmen.
	var kBID int
	db.QueryRow(`SELECT id FROM kader WHERE team_id=?`, fx.teamB).Scan(&kBID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kBID, fx.memberTeamA); err != nil {
		t.Fatalf("multi-kader insert: %v", err)
	}

	srv := testServer(t, db)
	token := testutil.Token(t, fx.userOfMemberA, "standard", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	items, hidden := decodeParticipants(t, res)
	by := memberIDsByTeam(items)
	if got := len(by[fx.teamA]); got != 2 {
		t.Errorf("Team A: erwartet 2, got %d", got)
	}
	// Team B nun vollständig sichtbar — 4 Zeilen: original 3 Member + memberTeamA
	// (steht zusätzlich im Kader von Team B, taucht daher dort als Zeile auf).
	if got := len(by[fx.teamB]); got != 4 {
		t.Errorf("Team B (eigenes Team auch): erwartet 4, got %d", got)
	}
	for _, h := range hidden {
		if h == fx.teamB {
			t.Errorf("Team B sollte NICHT hidden sein, wenn der Caller darin Mitglied ist")
		}
	}
}

// TestGetParticipants_MultiTeam_TrainerSiehtAlles — Funktionsträger
// (trainer) bypassen den Filter komplett.
func TestGetParticipants_MultiTeam_TrainerSiehtAlles(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	trainerUID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, trainerUID, "standard", []string{"trainer"})

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	items, hidden := decodeParticipants(t, res)
	if len(items) != 5 {
		t.Errorf("Trainer: erwartet 5 Member (alle), got %d", len(items))
	}
	if len(hidden) != 0 {
		t.Errorf("Trainer: hidden_team_ids muss leer sein, got %v", hidden)
	}
}

// TestGetParticipants_MultiTeam_VorstandSiehtAlles — analog für Vorstand.
func TestGetParticipants_MultiTeam_VorstandSiehtAlles(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	vUID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, vUID, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID), token)
	items, _ := decodeParticipants(t, res)
	if len(items) != 5 {
		t.Errorf("Vorstand: erwartet 5 Member, got %d", len(items))
	}
}

// TestGetParticipants_SingleTeam_KeinFilter — Single-Team-Event: kein Filter
// auch für Standard-Spieler, cross_team_visible irrelevant.
func TestGetParticipants_SingleTeam_KeinFilter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := mkTeamCustom(t, db, "TS Solo", "Erwachsene", "m")
	kaderID := mkKaderCustomReturn(t, db, seasonID, teamID, "Erwachsene", "m", 1)
	uid := testutil.CreateUser(t, db, "standard")
	m1 := testutil.CreateMember(t, db, uid)
	m2 := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, m1)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, m2)
	// m2 explizit auf cross_team_visible=0 — sollte irrelevant sein.

	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-04-12")
	// Bewusst NUR ein Eintrag in game_teams (kein zweites Team).

	// Ein Außenstehender (weder in diesem Kader noch Funktionsträger).
	otherUID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)

	// Caller A: Mitglied im einzigen Team — sieht alles.
	tokA := testutil.Token(t, uid, "standard", nil)
	resA := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), tokA)
	itemsA, hiddenA := decodeParticipants(t, resA)
	if len(itemsA) != 2 {
		t.Errorf("Single-Team, Mitglied: erwartet 2 Member, got %d", len(itemsA))
	}
	if len(hiddenA) != 0 {
		t.Errorf("Single-Team: hidden_team_ids muss leer sein, got %v", hiddenA)
	}

	// Caller B: außerhalb des Teams — single-team, also kein Filter; sieht ebenfalls alles.
	tokB := testutil.Token(t, otherUID, "standard", nil)
	resB := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), tokB)
	itemsB, _ := decodeParticipants(t, resB)
	if len(itemsB) != 2 {
		t.Errorf("Single-Team, Außenstehender: erwartet 2 Member (kein Filter), got %d", len(itemsB))
	}
}
