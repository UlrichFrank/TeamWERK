package chat_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// pgFixture: eine Übungsgruppe der aktiven Saison mit Trainer, Spieler und
// dessen Elternteil — dazu ein Fremder, der zu keiner Gruppe gehört.
type pgFixture struct {
	db       *sql.DB
	season   int
	groupID  int
	trainerU int
	playerU  int
	parentU  int
	fremdU   int
}

func setupPracticeGroup(t *testing.T) (*pgFixture, *chi.Mux) {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, season, "Torwarttraining")

	trainerU := testutil.CreateUser(t, db, "standard")
	testutil.AddKaderTrainer(t, db, groupID, testutil.CreateMember(t, db, trainerU))

	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddKaderMember(t, db, groupID, playerM)

	parentU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentU, playerM)

	// Der Fremde ist Spieler einer regulären Mannschaft — er ist kein
	// Unbeteiligter im System, nur keiner dieser Gruppe.
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, season)
	fremdU := testutil.CreateUser(t, db, "standard")
	testutil.AddKaderMember(t, db, kaderID, testutil.CreateMember(t, db, fremdU))

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	r := chi.NewRouter()
	r.Get("/api/chat/team-groups", h.ListTeamGroups)
	r.Get("/api/chat/practice-groups/{id}/{kind}/members", h.ResolvePracticeGroup)

	return &pgFixture{
		db: db, season: season, groupID: groupID,
		trainerU: trainerU, playerU: playerU, parentU: parentU, fremdU: fremdU,
	}, r
}

// TestListTeamGroups_EnthaeltUebungsgruppe: die Sichtbarkeit läuft über
// Mitgliedschaft/Trainer/Eltern, nicht über `user_accessible_teams` — die View
// filtert `k.team_id IS NOT NULL` und kennt Übungsgruppen gar nicht.
func TestListTeamGroups_EnthaeltUebungsgruppe(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.playerU, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	var found []chat.TeamGroup
	for _, g := range groups {
		if g.GroupType == "practice" && g.TeamID == f.groupID {
			found = append(found, g)
		}
	}
	if len(found) == 0 {
		t.Fatalf("Mitglied sieht seine Übungsgruppe nicht, bekommen: %+v", groups)
	}
	for _, g := range found {
		if g.DisplayShort != "Torwarttraining" {
			t.Errorf("DisplayShort = %q, erwartet Torwarttraining", g.DisplayShort)
		}
	}
}

// TestListTeamGroups_FremderSiehtUebungsgruppeNicht: der Gegentest — sonst wäre
// eine Gruppe, die alle sehen, von einer korrekt gefilterten nicht zu
// unterscheiden.
func TestListTeamGroups_FremderSiehtUebungsgruppeNicht(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.fremdU, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	for _, g := range groups {
		if g.GroupType == "practice" && g.TeamID == f.groupID {
			t.Fatalf("Fremder sieht die Übungsgruppe: %+v", g)
		}
	}
}

// TestResolvePracticeGroup_Mitglieder: die Auflösung liefert die Nutzer des
// kinds, der Aufrufer selbst wird herausgefiltert.
func TestResolvePracticeGroup_Mitglieder(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	// Zweites Gruppenmitglied, damit „Aufrufer herausgefiltert" prüfbar ist,
	// ohne dass die Liste leer wird.
	otherU := testutil.CreateUser(t, f.db, "standard")
	testutil.AddKaderMember(t, f.db, f.groupID, testutil.CreateMember(t, f.db, otherU))

	res := testutil.Get(t, srv,
		fmt.Sprintf("/api/chat/practice-groups/%d/spieler/members", f.groupID),
		testutil.Token(t, f.playerU, "standard", nil))
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("erwartet 200, bekommen %d", res.StatusCode)
	}
	members := decodeJSON[[]chat.TeamGroupMember](t, res)

	if len(members) != 1 || members[0].ID != otherU {
		t.Fatalf("erwartet genau den anderen Spieler (%d), bekommen %+v", otherU, members)
	}
}

// TestResolvePracticeGroup_Fremder: 403 für einen Nutzer ohne Zugehörigkeit.
func TestResolvePracticeGroup_Fremder(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		fmt.Sprintf("/api/chat/practice-groups/%d/spieler/members", f.groupID),
		testutil.Token(t, f.fremdU, "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", res.StatusCode)
	}
}

// TestResolvePracticeGroup_TeamKaderIst404: ein Mannschaftskader ist über die
// Übungsgruppen-Route nicht adressierbar. 404, nicht 403 — die Route kennt
// dieses Objekt nicht, statt den Zugriff darauf zu verweigern.
func TestResolvePracticeGroup_TeamKaderIst404(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	teamID := testutil.CreateTeam(t, f.db, "Team B")
	kaderID := testutil.CreateKader(t, f.db, teamID, f.season)
	uid := testutil.CreateUser(t, f.db, "standard")
	testutil.AddKaderMember(t, f.db, kaderID, testutil.CreateMember(t, f.db, uid))

	// Auch für ein Mitglied genau dieses Kaders: die Route ist nicht der Weg.
	res := testutil.Get(t, srv,
		fmt.Sprintf("/api/chat/practice-groups/%d/spieler/members", kaderID),
		testutil.Token(t, uid, "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("erwartet 404, bekommen %d", res.StatusCode)
	}
}

// TestResolvePracticeGroup_SpielerOhneErweitertenKader: Übungsgruppen haben
// keinen erweiterten Kader (PUT /api/kader/{id} lehnt ihn mit 409 ab). Das kind
// `spieler` darf deshalb — anders als bei einer Mannschaft — keine Union mit
// kader_extended_members tragen. Der Test schreibt die Zeile direkt in die DB,
// am Gate vorbei: geprüft wird die Query, nicht das Gate.
func TestResolvePracticeGroup_SpielerOhneErweitertenKader(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	extU := testutil.CreateUser(t, f.db, "standard")
	extM := testutil.CreateMember(t, f.db, extU)
	testutil.AddExtendedKaderMember(t, f.db, f.groupID, extM)

	res := testutil.Get(t, srv,
		fmt.Sprintf("/api/chat/practice-groups/%d/spieler/members", f.groupID),
		testutil.Token(t, f.trainerU, "standard", nil))
	members := decodeJSON[[]chat.TeamGroupMember](t, res)

	for _, m := range members {
		if m.ID == extU {
			t.Fatalf("erweiterter Kader taucht im kind spieler auf: %+v", members)
		}
	}
}
