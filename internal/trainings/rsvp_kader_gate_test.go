package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// Tests zum Change `rsvp-kader-gate`: eine training_responses-Zeile entsteht nur
// für Beteiligte des Termins (Stammkader, erweiterter Kader, Trainer).
//
// Das Risiko dieses Changes liegt auf der MANNSCHAFTSSEITE — dort ändert sich
// Verhalten, das es vorher gab. Die Übungsgruppe ist nur der Fall, der die Lücke
// sichtbar gemacht hat.

// gateFixture: ein Mannschaftstermin mit seinem Kader.
type gateFixture struct {
	db        *sql.DB
	srv       interface{ Close() }
	sessionID int
	kaderID   int
	teamID    int
	seasonID  int
}

func setupGate(t *testing.T) (*gateFixture, func(token string, body any) *http.Response) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	// Vor dem Cutoff — sonst maskiert ein 422 die Aussage über das Gate.
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-14 12:00")))
	srv := testServer(t, h)

	post := func(token string, body any) *http.Response {
		return testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token, body)
	}
	return &gateFixture{db: db, srv: srv, sessionID: sessionID, kaderID: kaderID, teamID: teamID, seasonID: seasonID}, post
}

func (f *gateFixture) responseCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=?`, f.sessionID).Scan(&n); err != nil {
		t.Fatalf("responseCount: %v", err)
	}
	return n
}

// TestRespond_FremderAbgelehnt: der Kern des Changes. Ein Nutzer mit
// Mitglieds-Datensatz, der zum Termin nicht gehört, wird abgelehnt — vorher
// wurde seine Zusage gespeichert und zählte in confirmed_count mit.
func TestRespond_FremderAbgelehnt(t *testing.T) {
	f, post := setupGate(t)
	uid := testutil.CreateUser(t, f.db, "standard")
	testutil.CreateMember(t, f.db, uid)

	res := post(testutil.Token(t, uid, "standard", []string{"spieler"}), map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", res.StatusCode)
	}
	if n := f.responseCount(t); n != 0 {
		t.Fatalf("trotz 403 wurden %d Zeilen geschrieben", n)
	}
}

// TestRespond_StammkaderErlaubt / _ErweiterterKaderErlaubt / _TrainerErlaubt:
// alle drei Zweige der Beteiligung. Ohne diese drei wäre ein zu enges Gate
// (etwa nur kader_members) von einem richtigen nicht zu unterscheiden.
func TestRespond_StammkaderErlaubt(t *testing.T) {
	f, post := setupGate(t)
	uid := testutil.CreateUser(t, f.db, "standard")
	mid := testutil.CreateMember(t, f.db, uid)
	testutil.AddKaderMember(t, f.db, f.kaderID, mid)

	res := post(testutil.Token(t, uid, "standard", []string{"spieler"}), map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d", res.StatusCode)
	}
	if n := f.responseCount(t); n != 1 {
		t.Fatalf("erwartet 1 Zeile, gefunden %d", n)
	}
}

func TestRespond_ErweiterterKaderErlaubt(t *testing.T) {
	f, post := setupGate(t)
	uid := testutil.CreateUser(t, f.db, "standard")
	mid := testutil.CreateMember(t, f.db, uid)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderID, mid)

	res := post(testutil.Token(t, uid, "standard", []string{"spieler"}), map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d — der erweiterte Kader ist Beteiligter, nicht Fremder", res.StatusCode)
	}
}

func TestRespond_TrainerErlaubt(t *testing.T) {
	f, post := setupGate(t)
	uid := testutil.CreateUser(t, f.db, "standard")
	mid := testutil.CreateMember(t, f.db, uid)
	testutil.AddKaderTrainer(t, f.db, f.kaderID, mid)

	res := post(testutil.Token(t, uid, "standard", []string{"trainer"}), map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d", res.StatusCode)
	}
}

// TestRespond_UebungsgruppeFremderAbgelehnt: der Fall, den `uebungsgruppen`
// nicht abschließen konnte. Derselbe Codepfad, deshalb dasselbe Ergebnis.
func TestRespond_UebungsgruppeFremderAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	// Der Fremde ist Spieler einer regulären Mannschaft — kein Unbeteiligter im
	// System, nur keiner dieser Gruppe.
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.AddKaderMember(t, db, kaderID, testutil.CreateMember(t, db, uid))

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-14 12:00")))
	srv := testServer(t, h)

	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID),
		testutil.Token(t, uid, "standard", []string{"spieler"}), map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=?`, sessionID).Scan(&n)
	if n != 0 {
		t.Fatalf("trotz 403 wurden %d Zeilen geschrieben", n)
	}
}

// TestRespond_ElternteilFuerFremdesKindAbgelehnt: unverändertes
// Bestandsverhalten — das Ownership-Gate greift vor dem Kader-Gate.
func TestRespond_ElternteilFuerFremdesKindAbgelehnt(t *testing.T) {
	f, post := setupGate(t)
	// Das Kind gehört zum Termin, der Elternteil aber nicht zum Kind.
	fremdesKind := testutil.CreateMember(t, f.db, 0)
	testutil.AddKaderMember(t, f.db, f.kaderID, fremdesKind)

	parentUID := testutil.CreateUser(t, f.db, "standard")
	eigenesKind := testutil.CreateMember(t, f.db, 0)
	testutil.AddFamilyLink(t, f.db, parentUID, eigenesKind)

	res := post(testutil.TokenWithIsParent(t, parentUID, "standard", nil, true),
		map[string]any{"status": "confirmed", "member_id": fremdesKind})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", res.StatusCode)
	}
}

// TestRespond_StaffFuerNichtKadermitgliedAbgelehnt: die Verschärfung aus
// design.md — Entscheidung 3. Der Vorstand darf für ein Mitglied antworten,
// aber nicht auf einem Termin, mit dem dieses Mitglied nichts zu tun hat.
func TestRespond_StaffFuerNichtKadermitgliedAbgelehnt(t *testing.T) {
	f, post := setupGate(t)
	vorstandUID := testutil.CreateUser(t, f.db, "standard")
	testutil.CreateMember(t, f.db, vorstandUID)
	fremdesMitglied := testutil.CreateMember(t, f.db, 0) // in keinem Kader

	res := post(testutil.Token(t, vorstandUID, "standard", []string{"vorstand"}),
		map[string]any{"status": "confirmed", "member_id": fremdesMitglied})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, bekommen %d", res.StatusCode)
	}
	if n := f.responseCount(t); n != 0 {
		t.Fatalf("trotz 403 wurden %d Zeilen geschrieben", n)
	}
}

// TestRespond_AnzeigeUndAntwortrechtStimmenUeberein hält Invariante 2 fest:
// „darf ich antworten?" (Respond, Go-Helfer) und „bin ich beteiligt?"
// (am_i_participant, SQL) sind dieselbe Frage an zwei Fundstellen. Driften sie
// auseinander, entsteht ein Termin, der als „du bist dabei" angezeigt wird,
// dessen Zusage aber 403 liefert — oder umgekehrt.
//
// Geprüft wird über alle drei Beteiligungszweige plus den Fremden: für jeden
// muss `am_i_participant` und der Ausgang der Antwort dasselbe sagen.
func TestRespond_AnzeigeUndAntwortrechtStimmenUeberein(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	mkUser := func(join func(kaderID, memberID int)) (uid int) {
		uid = testutil.CreateUser(t, db, "standard")
		mid := testutil.CreateMember(t, db, uid)
		if join != nil {
			join(kaderID, mid)
		}
		return uid
	}
	cases := []struct {
		name string
		uid  int
	}{
		{"stammkader", mkUser(func(k, m int) { testutil.AddKaderMember(t, db, k, m) })},
		{"erweiterter kader", mkUser(func(k, m int) { testutil.AddExtendedKaderMember(t, db, k, m) })},
		{"trainer", mkUser(func(k, m int) { testutil.AddKaderTrainer(t, db, k, m) })},
		{"fremder", mkUser(nil)},
	}

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-14 12:00")))
	srv := testServer(t, h)

	for _, c := range cases {
		token := testutil.Token(t, c.uid, "standard", nil)

		lr := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", token)
		var body struct {
			Items []struct {
				ID             int  `json:"id"`
				AmIParticipant bool `json:"am_i_participant"`
			} `json:"items"`
		}
		json.NewDecoder(lr.Body).Decode(&body)
		lr.Body.Close()

		// listed und flag getrennt: fehlt das Item ganz, ist das ein anderer
		// Befund (Sichtbarkeitsfilter) als ein am_i_participant=false auf einem
		// gelieferten Termin — beides bricht die Invariante, aber an
		// verschiedenen Stellen.
		listed, flag := false, false
		for _, it := range body.Items {
			if it.ID == sessionID {
				listed, flag = true, it.AmIParticipant
			}
		}

		rr := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
			map[string]any{"status": "confirmed"})
		rr.Body.Close()
		mayAnswer := rr.StatusCode == http.StatusNoContent

		if flag != mayAnswer {
			t.Errorf("%s: gelistet=%v, am_i_participant=%v, Antwort=%d — Anzeige und Antwortrecht widersprechen sich",
				c.name, listed, flag, rr.StatusCode)
		}
	}
}

// TestListSessions_KaderTrainerOhneVereinsfunktion: der Befund, den der
// Konsistenztest oben zutage gefördert hat. Der Trainer-Zweig des
// Sichtbarkeitsfilters in ListSessions hing an der Vereinsfunktion `trainer`
// UND der Eintragung in kader_trainers; das Antwortrecht (isKaderParticipant)
// hängt nur an der Eintragung. Wer als Trainer eines Kaders geführt wird, ohne
// die Vereinsfunktion zu tragen, durfte den Termin damit verwalten und
// beantworten, sah ihn aber nicht.
//
// Die Konstellation ist keine Theorie: KaderTrainerSearch bietet den
// Funktionsfilter als abwählbare Checkbox an — ein Übungsgruppenleiter ohne
// Vereinsfunktion ist ausdrücklich vorgesehen.
func TestListSessions_KaderTrainerOhneVereinsfunktion(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	uid := testutil.CreateUser(t, db, "standard")
	testutil.AddKaderTrainer(t, db, groupID, testutil.CreateMember(t, db, uid))

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-14 12:00")))
	srv := testServer(t, h)

	// Token ohne jede Vereinsfunktion — die Zugehörigkeit steht allein in
	// kader_trainers.
	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31",
		testutil.Token(t, uid, "standard", nil))
	var body struct {
		Items []struct {
			ID             int  `json:"id"`
			AmIParticipant bool `json:"am_i_participant"`
		} `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	for _, it := range body.Items {
		if it.ID == sessionID {
			if !it.AmIParticipant {
				t.Fatalf("Termin gelistet, aber am_i_participant=false")
			}
			return
		}
	}
	t.Fatalf("Termin %d fehlt in der Liste — der Kader-Trainer sieht seinen eigenen Termin nicht", sessionID)
}
