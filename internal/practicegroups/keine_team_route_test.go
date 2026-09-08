package practicegroups_test

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestPracticeGroup_KeineTeamRoute hält die strukturelle Aussage dieses Changes
// fest: eine Übungsgruppe hat keine `teams`-Zeile, und **deshalb** — nicht wegen
// eines Filters — bleiben Strafen, Kasse, Aufgaben, Warte, Statistiken und
// Roster für sie unerreichbar (design.md — Entscheidung 1, „fails closed").
//
// **Das Kriterium ist der Inhalt, nicht der Status-Code.** Ein Teil dieser
// Listen-Routen antwortet auf eine unbekannte `teams.id` mit `200 []` statt mit
// 404 — sie lösen über `WHERE k.team_id = ?` auf, finden nichts und geben die
// leere Liste zurück. Das ist Bestandsverhalten und keine Aussage über
// Übungsgruppen; ein Test auf „kein 2xx" würde es fälschlich zu einer machen.
// Was die Spec zusagt, ist „kein 2xx **für die Gruppe**": unter
// `/api/teams/{id}/…` darf nichts von ihr sichtbar werden.
//
// Der Test läuft deshalb gegen den **Produktions-Router** mit einem
// **Admin**-Token — scheiterte eine Route an der Berechtigung statt an der
// fehlenden `teams`-Zeile, bewiese sie nichts — und sucht die eindeutigen Namen
// der Gruppe und ihres Mitglieds in **jedem** Antwort-Body.
//
// Er ist bewusst eine Charakterisierung, kein Arch-Gate: weil es genau ein
// anwendungsseitiges Gate gibt (`kader_extended_members`), braucht es keine
// Allowlist im Stil von `broadcastAllowlist`. Eine neue `/api/teams/{id}/…`-Route
// deckt dieser Test nicht automatisch ab — er hält die Aussage fest, er
// erzwingt sie nicht.
func TestPracticeGroup_KeineTeamRoute(t *testing.T) {
	const groupName = "ZzTorwartgruppeXy"
	const memberLast = "ZzUebungsmitgliedXy"

	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, groupName)

	memberID := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	if _, err := db.Exec(`UPDATE members SET last_name=? WHERE id=?`, memberLast, memberID); err != nil {
		t.Fatalf("Mitgliedsname setzen: %v", err)
	}
	testutil.AddKaderMember(t, db, groupID, memberID)

	// Kader-IDs und Team-IDs sind getrennte Sequenzen. Ohne diese Zusicherung
	// könnte die Gruppen-ID zufällig auf ein existierendes Team zeigen und der
	// Test seine eigene Aussage verlieren.
	var teamWithSameID int
	if err := db.QueryRow(`SELECT COUNT(*) FROM teams WHERE id = ?`, groupID).Scan(&teamWithSameID); err != nil {
		t.Fatalf("teams prüfen: %v", err)
	}
	if teamWithSameID != 0 {
		t.Fatalf("Vorbedingung verletzt: es gibt bereits ein Team mit id=%d", groupID)
	}

	srv := prodserver.New(t, db)
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	id := strconv.Itoa(groupID)
	paths := []string{
		"/api/teams/" + id + "/attendance-stats",
		"/api/teams/" + id + "/attendance-open",
		"/api/teams/" + id + "/training-diary-stats",
		"/api/teams/" + id + "/roster",
		"/api/teams/" + id + "/responsibility-types",
		"/api/teams/" + id + "/responsibilities",
		"/api/teams/" + id + "/penalties",
		"/api/teams/" + id + "/penalty-types",
		"/api/teams/" + id + "/penalty-wardens",
		"/api/teams/" + id + "/penalty-settings",
		"/api/teams/" + id + "/cashbook",
		"/api/teams/" + id + "/treasurers",
	}
	for _, p := range paths {
		res := testutil.Get(t, srv, p, token)
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatalf("GET %s: Body lesen: %v", p, err)
		}
		for _, needle := range []string{groupName, memberLast} {
			if strings.Contains(string(body), needle) {
				t.Errorf("GET %s (%d): Antwort enthält %q — die Übungsgruppe ist unter /api/teams/{id}/… sichtbar geworden",
					p, res.StatusCode, needle)
			}
		}
	}

	// Gegenprobe: dieselben Namen sind über die Übungsgruppen-Route sehr wohl
	// zu sehen. Sonst wäre der Test auch dann grün, wenn die Fixture nichts
	// angelegt hätte.
	res := testutil.Get(t, srv, "/api/practice-groups/"+id, token)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, needle := range []string{groupName, memberLast} {
		if !strings.Contains(string(body), needle) {
			t.Fatalf("GET /api/practice-groups/%s enthält %q nicht — die Fixture trägt nicht", id, needle)
		}
	}
}
