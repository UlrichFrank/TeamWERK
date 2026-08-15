package chat_test

import (
	"database/sql"
	"net/http"
	"slices"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// audienceWorld ist eine Fixture, in der sich die vier Zielgruppen sichtbar
// unterscheiden. Die Konstellation ist bewusst so gewählt, dass jede Zielgruppe
// mindestens einen User enthält, den eine andere NICHT enthält.
type audienceWorld struct {
	db *sql.DB

	sender      int // vorstand, Mitglied, keine Spieler-Funktion
	spielerUser int // Mitglied mit Vereinsfunktion 'spieler'
	spielerZwei int // Mitglied mit 'spieler' UND 'trainer' (Dedup-Fall)
	elternUser  int // reiner Elternaccount, kein eigener Mitglieds-Datensatz
	nurUser     int // User ohne Mitglieds-Datensatz und ohne Kind
}

func setupAudienceWorld(t *testing.T) *audienceWorld {
	t.Helper()
	db := testutil.NewDB(t)
	w := &audienceWorld{db: db}

	w.sender = testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, w.sender)

	w.spielerUser = testutil.CreateUser(t, db, "standard")
	mSpieler := testutil.CreateMember(t, db, w.spielerUser)
	addFunction(t, db, mSpieler, "spieler")

	// Zwei Vereinsfunktionen am selben Mitglied — muss trotzdem eine ID liefern.
	w.spielerZwei = testutil.CreateUser(t, db, "standard")
	mZwei := testutil.CreateMember(t, db, w.spielerZwei)
	addFunction(t, db, mZwei, "spieler")
	addFunction(t, db, mZwei, "trainer")

	// Elternteil mit ZWEI Kindern: Dedup-Fall für die eltern-Zielgruppe.
	// Beide Kinder sind Spieler — das Elternteil darf dadurch NICHT in
	// die spieler-Zielgruppe rutschen (keine Vererbung, design.md §3).
	w.elternUser = testutil.CreateUser(t, db, "standard")
	for range 2 {
		kind := testutil.CreateMember(t, db, 0)
		addFunction(t, db, kind, "spieler")
		linkParent(t, db, w.elternUser, kind)
	}

	w.nurUser = testutil.CreateUser(t, db, "standard")

	// Mitglied OHNE User-Account, mit Spieler-Funktion: über eine Mitteilung
	// strukturell nicht erreichbar.
	ohneZugang := testutil.CreateMember(t, db, 0)
	addFunction(t, db, ohneZugang, "spieler")

	return w
}

func addFunction(t *testing.T, db *sql.DB, memberID int, fn string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, fn); err != nil {
		t.Fatalf("addFunction(%d, %q): %v", memberID, fn, err)
	}
}

func linkParent(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, memberID); err != nil {
		t.Fatalf("linkParent(%d, %d): %v", parentUserID, memberID, err)
	}
}

// recipientsOf sendet eine Mitteilung an die Zielgruppe und liefert die
// tatsächlich erzeugten broadcast_reads-Empfänger sowie den gemeldeten Zähler.
// Bewusst über die Route statt über den unexportierten Resolver: die Zusage der
// Spec ist "wer bekommt eine Zeile", nicht "was gibt eine Funktion zurück".
func recipientsOf(t *testing.T, w *audienceWorld, target string) (ids []int, reported int) {
	t.Helper()
	srv := broadcastSrv(t, w.db)

	token := testutil.Token(t, w.sender, "standard", []string{"vorstand"})
	res := testutil.Post(t, srv, "/api/chat/broadcasts", token,
		map[string]any{"body": "Test " + target, "targetType": target})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("targetType %q: status %d, want 201", target, res.StatusCode)
	}
	body := decodeJSON[sendResult](t, res)

	rows, err := w.db.Query(
		`SELECT user_id FROM broadcast_reads WHERE broadcast_id = ? ORDER BY user_id`, body.ID)
	if err != nil {
		t.Fatalf("read broadcast_reads: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids, body.Recipients
}

// assertAudience prüft die Zielgruppen-Auflösung gegen die erwartete Menge.
//
// broadcast_reads enthält immer zusätzlich den Absender — SendBroadcast schreibt
// ihm unabhängig von der Zielgruppe eine Zeile mit gesetztem read_at, damit die
// eigene Mitteilung nicht als ungelesen erscheint. Die Zeilenmenge ist deshalb
// `Zielgruppe ∪ {Absender}`, der gemeldete recipients-Zähler dagegen die
// Zielgruppe OHNE ihn.
func assertAudience(t *testing.T, target string, sender int, gotRows []int, gotReported int, audience []int) {
	t.Helper()

	wantRows := append(slices.Clone(audience), sender)
	slices.Sort(wantRows)
	wantRows = slices.Compact(wantRows)
	got := slices.Clone(gotRows)
	slices.Sort(got)
	if !slices.Equal(got, wantRows) {
		t.Errorf("Zielgruppe %q: broadcast_reads = %v, want %v", target, got, wantRows)
	}

	wantReported := 0
	for _, id := range audience {
		if id != sender {
			wantReported++
		}
	}
	if gotReported != wantReported {
		t.Errorf("Zielgruppe %q: recipients = %d, want %d", target, gotReported, wantReported)
	}
}

// TC: Jede Zielgruppe löst auf die EXAKTE erwartete Menge auf — nicht nur auf
// "nicht leer". Genau diese Assertion hätte den Bug verhindert, bei dem
// targetType 'role' gegen users.role auflöste und immer null Empfänger traf,
// ohne dass irgendetwas fehlschlug.
func TestZielgruppen_LoesenAufDieExakteMengeAuf(t *testing.T) {
	tests := []struct {
		target string
		// want liefert die reine Zielgruppe (ohne die stets zusätzlich
		// geschriebene Absenderzeile).
		want func(w *audienceWorld) []int
	}{
		{"users", func(w *audienceWorld) []int {
			return []int{w.sender, w.spielerUser, w.spielerZwei, w.elternUser, w.nurUser}
		}},
		{"members", func(w *audienceWorld) []int {
			// nurUser und elternUser haben keinen Mitglieds-Datensatz.
			return []int{w.sender, w.spielerUser, w.spielerZwei}
		}},
		{"spieler", func(w *audienceWorld) []int {
			// sender ist Mitglied ohne Spieler-Funktion, elternUser erbt nicht.
			return []int{w.spielerUser, w.spielerZwei}
		}},
		{"eltern", func(w *audienceWorld) []int {
			return []int{w.elternUser}
		}},
	}

	for _, tc := range tests {
		t.Run(tc.target, func(t *testing.T) {
			w := setupAudienceWorld(t)
			rows, reported := recipientsOf(t, w, tc.target)
			assertAudience(t, tc.target, w.sender, rows, reported, tc.want(w))
		})
	}
}

// TC: Eltern erben die Vereinsfunktion ihrer Kinder NICHT — anders als in der
// Ordner-ACL (policy.FolderAccess, case "club_function"), wo genau das passiert.
// Ohne diese Trennung wäre 'eltern' eine Teilmenge von 'spieler' und die
// Auswahl im Composer irreführend.
func TestZielgruppe_SpielerErbtNichtAnEltern(t *testing.T) {
	w := setupAudienceWorld(t)

	spieler, _ := recipientsOf(t, w, "spieler")
	if slices.Contains(spieler, w.elternUser) {
		t.Error("Elternteil eines Spielers darf NICHT in der Zielgruppe 'spieler' sein")
	}

	eltern, _ := recipientsOf(t, w, "eltern")
	if !slices.Contains(eltern, w.elternUser) {
		t.Error("Elternteil fehlt in der Zielgruppe 'eltern'")
	}
}

// TC: Mehrfachzugehörigkeit erzeugt genau eine Zeile. Ohne DISTINCT läge der
// Fehler nicht im Fan-out (INSERT OR IGNORE fängt ihn), sondern im gemeldeten
// recipients-Zähler — der Absender bekäme eine zu hohe Zahl zu sehen.
func TestZielgruppen_MehrfachzugehoerigkeitZaehltEinmal(t *testing.T) {
	w := setupAudienceWorld(t)

	// Elternteil mit zwei Kindern: genau eine Zeile, genau einmal gezählt.
	eltern, reported := recipientsOf(t, w, "eltern")
	elternRows := 0
	for _, id := range eltern {
		if id == w.elternUser {
			elternRows++
		}
	}
	if elternRows != 1 {
		t.Errorf("Elternteil zweier Kinder hat %d broadcast_reads-Zeilen, want 1", elternRows)
	}
	if reported != 1 {
		t.Errorf("Elternteil zweier Kinder: recipients = %d, want 1", reported)
	}

	// Mitglied mit zwei Vereinsfunktionen.
	members, _ := recipientsOf(t, w, "members")
	count := 0
	for _, id := range members {
		if id == w.spielerZwei {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Mitglied mit zwei Vereinsfunktionen erscheint %dx in 'members', want 1", count)
	}
}

// TC: Ein Mitglied ohne User-Account ist über keine Zielgruppe erreichbar. Die
// Lücke ist bewusst (broadcast_reads.user_id ist der Anker), sie darf aber nicht
// still zu einer falschen Empfängerzahl führen.
func TestZielgruppen_MitgliedOhneZugangIstNichtErreichbar(t *testing.T) {
	w := setupAudienceWorld(t)

	// Sanity: die Fixture enthält Spieler-Mitglieder mit UND ohne Zugang.
	var spielerMitglieder, ohneZugang int
	if err := w.db.QueryRow(`
		SELECT COUNT(*), SUM(CASE WHEN m.user_id IS NULL THEN 1 ELSE 0 END)
		  FROM members m
		  JOIN member_club_functions mcf ON mcf.member_id = m.id AND mcf.function = 'spieler'`).
		Scan(&spielerMitglieder, &ohneZugang); err != nil {
		t.Fatalf("count spieler: %v", err)
	}
	if ohneZugang == 0 {
		t.Fatal("Fixture kaputt: kein Spieler-Mitglied ohne User-Account vorhanden")
	}

	// Die Zielgruppe zählt nur die erreichbaren — und meldet das ehrlich, statt
	// den nicht erreichbaren Rest stillschweigend mitzuzählen.
	_, reported := recipientsOf(t, w, "spieler")
	wantReachable := spielerMitglieder - ohneZugang
	if reported != wantReachable {
		t.Errorf("recipients = %d, want %d (von %d Spieler-Mitgliedern haben %d keinen Zugang)",
			reported, wantReachable, spielerMitglieder, ohneZugang)
	}
}
