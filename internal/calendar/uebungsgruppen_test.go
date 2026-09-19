package calendar_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// uebungsgruppenFixture baut einen Nutzer, der zugleich an einer Mannschaft und
// an einer Übungsgruppe hängt, mit je einem Trainingstermin. Der
// Mannschaftstermin ist in jedem Test der Gegenbeleg: ohne ihn wäre ein leerer
// Feed von einem korrekt gefilterten nicht zu unterscheiden.
//
// Rückgabe: Server, User-Token, ID des Mannschaftstermins, ID des
// Übungsgruppen-Termins.
func uebungsgruppenFixture(t *testing.T) (srv *httptest.Server, userToken string, teamSessionID, groupSessionID int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	teamSessionID = testutil.CreateTrainingSessionForKader(t, db, kaderID, seasonID, "2026-05-04", "Mannschaftstraining")

	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	testutil.AddKaderMember(t, db, groupID, memberID)
	// Der Termin-Titel weicht bewusst vom Gruppennamen ab: der Feed muss den
	// Gruppennamen ausgeben, nicht den (pro Termin frei gepflegten) Titel.
	groupSessionID = testutil.CreateTrainingSessionForKader(t, db, groupID, seasonID, "2026-05-05", "TW-Einheit 3")

	srv = prodserver.New(t, db)
	userToken = testutil.Token(t, userID, "standard", nil)
	return srv, userToken, teamSessionID, groupSessionID
}

// feedWithToggles legt ein Token mit den übergebenen Schaltern an und liefert
// den entfalteten Feed-Text.
func feedWithToggles(t *testing.T, srv *httptest.Server, userToken string, toggles map[string]any) string {
	t.Helper()
	tok := postToken(t, srv, userToken, toggles)
	res := testutil.Get(t, srv, "/api/calendar/feed/"+tok["token"].(string), "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", res.StatusCode)
	}
	return unfoldICS(readBody(t, res.Body))
}

func trainingUID(id int) string { return "training-" + strconv.Itoa(id) }

// TestIcalFeed_MitUebungsgruppe ist die Umkehrung des früheren
// TestIcalFeed_OhneUebungsgruppe: der Ausschluss der Übungsgruppen war eine
// Folge der Auflösung über `ts.team_id` (bei einer Übungsgruppe NULL). Seit
// `fetchTrainings` über `ts.kader_id` auflöst, gehören die Termine in den Feed.
func TestIcalFeed_MitUebungsgruppe(t *testing.T) {
	srv, userToken, teamSessionID, groupSessionID := uebungsgruppenFixture(t)

	body := feedWithToggles(t, srv, userToken, allTogglesOn())

	if !strings.Contains(body, trainingUID(teamSessionID)) {
		t.Fatalf("Mannschaftstermin (%s) fehlt im Feed — der Gegenbeleg trägt nicht", trainingUID(teamSessionID))
	}
	if !strings.Contains(body, trainingUID(groupSessionID)) {
		t.Errorf("Übungsgruppen-Termin (%s) fehlt im iCal-Feed", trainingUID(groupSessionID))
	}
}

// TestIcalFeed_UebungsgruppeTraegtGruppennamen hält fest, dass die Beschriftung
// aus `kader.name` kommt und nicht aus `training_sessions.title`. Der Titel ist
// pro Termin frei und driftet innerhalb derselben Serie; ein Kalendereintrag
// soll über die Saison hinweg wiedererkennbar bleiben.
func TestIcalFeed_UebungsgruppeTraegtGruppennamen(t *testing.T) {
	srv, userToken, _, _ := uebungsgruppenFixture(t)

	body := feedWithToggles(t, srv, userToken, allTogglesOn())

	if !strings.Contains(body, "SUMMARY:Training: Torwarttraining") {
		t.Errorf("SUMMARY trägt nicht den Gruppennamen; Feed:\n%s", body)
	}
	if strings.Contains(body, "TW-Einheit 3") {
		t.Errorf("SUMMARY trägt den Termin-Titel statt des Gruppennamens")
	}
}

// TestIcalFeed_PracticeGroupsToggleAus prüft den neuen Schalter: er filtert
// ausschließlich die Übungsgruppen heraus, das Mannschaftstraining bleibt.
func TestIcalFeed_PracticeGroupsToggleAus(t *testing.T) {
	srv, userToken, teamSessionID, groupSessionID := uebungsgruppenFixture(t)

	toggles := allTogglesOn()
	toggles["include_practice_groups"] = false
	body := feedWithToggles(t, srv, userToken, toggles)

	if !strings.Contains(body, trainingUID(teamSessionID)) {
		t.Errorf("Mannschaftstermin (%s) fehlt, obwohl include_training=true", trainingUID(teamSessionID))
	}
	if strings.Contains(body, trainingUID(groupSessionID)) {
		t.Errorf("Übungsgruppen-Termin (%s) steht im Feed trotz include_practice_groups=false", trainingUID(groupSessionID))
	}
}

// TestIcalFeed_TrainingToggleAusLaesstUebungsgruppe zeigt die Gegenrichtung:
// die beiden Schalter sind unabhängig, include_training betrifft nur Termine
// mit gesetztem team_id.
func TestIcalFeed_TrainingToggleAusLaesstUebungsgruppe(t *testing.T) {
	srv, userToken, teamSessionID, groupSessionID := uebungsgruppenFixture(t)

	toggles := allTogglesOn()
	toggles["include_training"] = false
	body := feedWithToggles(t, srv, userToken, toggles)

	if strings.Contains(body, trainingUID(teamSessionID)) {
		t.Errorf("Mannschaftstermin (%s) steht im Feed trotz include_training=false", trainingUID(teamSessionID))
	}
	if !strings.Contains(body, trainingUID(groupSessionID)) {
		t.Errorf("Übungsgruppen-Termin (%s) fehlt, obwohl include_practice_groups=true", trainingUID(groupSessionID))
	}
}

// TestCalendarToken_PracticeGroupsRoundtrip sichert, dass der sechste Schalter
// über POST und GET denselben Wert führt — ohne das wäre die Oberfläche
// stillschweigend wirkungslos.
func TestCalendarToken_PracticeGroupsRoundtrip(t *testing.T) {
	srv, userToken, _, _ := uebungsgruppenFixture(t)

	toggles := allTogglesOn()
	toggles["include_practice_groups"] = false
	if got := postToken(t, srv, userToken, toggles)["include_practice_groups"]; got != false {
		t.Fatalf("POST-Antwort: include_practice_groups=%v, erwartet false", got)
	}

	res := testutil.Get(t, srv, "/api/calendar/token", userToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/calendar/token: erwartet 200, bekommen %d", res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["include_practice_groups"] != false {
		t.Errorf("GET-Antwort: include_practice_groups=%v, erwartet false", out["include_practice_groups"])
	}
}
