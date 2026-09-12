// Fixture-Welt für TestObjectAuthorizationMatrix (object_matrix_test.go).
//
// Die Tier-Matrix (matrix_test.go) beweist, dass eine Route im richtigen
// Auth-Tier hängt — sie läuft auf leerer DB mit Pfad-ID 1 und kann deshalb
// grundsätzlich nur „alle Personas → 403/404" abbilden. Diese Datei liefert das
// fehlende Stück: echte Objekte, die *User B* gehören, gegen die *User A*
// anfragt.
//
// Drei Anfragende:
//   - A  (Standard-Rolle, Vereinsfunktion `spieler`, eigener Kader/Team) —
//     der Normalfall für alle Routen im Authenticated-Tier.
//   - T  (Standard-Rolle, Vereinsfunktion `trainer`, Trainer eines FREMDEN
//     Kaders) — nötig für Routen im Trainer-/Vorstand+Trainer-Tier: ohne die
//     Funktion antwortet schon die Middleware mit 403 und der Objekt-Check
//     im Handler wird nie erreicht.
//   - E  (Standard-Rolle, `IsParent`, Elternteil eines EIGENEN Kindes) — für
//     die /api/profile/kind/…-Routen: mit A (gar kein Elternteil) bewiese ein
//     403 nur „ist kein Elternteil"; E beweist die schärfere Aussage „ist
//     Elternteil, aber nicht dieses Kindes".
//   - B  (Eigentümer) — nur für die Fixture-Probe („existiert das Objekt
//     überhaupt?"), nie als Angreifer.
package permissions_test

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

var objCounter atomic.Int64

// objWorld hält die Grunddaten beider Seiten. Die konkreten Fixture-Objekte
// entstehen pro Route frisch (siehe die new*-Helfer), damit ein Befund
// (2xx statt 403) keine Folge-Route verfälscht.
type objWorld struct {
	t   *testing.T
	db  *sql.DB
	srv *httptest.Server

	seasonID int

	// B — Eigentümer aller Fixture-Objekte.
	bUserID, bMemberID, bTeamID, bKaderID int
	bChildUserID, bChildMemberID          int
	bPracticeGroupID                      int
	bToken                                string

	// A — Anfragender ohne jede Beziehung zu B.
	aUserID, aMemberID, aTeamID, aKaderID int
	aToken                                string

	// T — Anfragender mit Vereinsfunktion `trainer`, aber für einen fremden Kader.
	tUserID, tMemberID, tTeamID, tKaderID int
	tToken                                string

	// E — Elternteil eines eigenen Kindes, ohne jede Beziehung zu B's Kind.
	eUserID, eChildMemberID int
	eToken                  string
}

func newObjWorld(t *testing.T, db *sql.DB, srv *httptest.Server) *objWorld {
	t.Helper()
	w := &objWorld{t: t, db: db, srv: srv}

	w.seasonID = testutil.CreateSeason(t, db, "Objekt-Matrix-Saison")

	// ── B: Eigentümer ────────────────────────────────────────────────────────
	w.bUserID = testutil.CreateUser(t, db, "standard")
	w.bMemberID = testutil.CreateMember(t, db, w.bUserID)
	w.bTeamID = testutil.CreateTeam(t, db, "B-Mannschaft")
	w.bKaderID = testutil.CreateKader(t, db, w.bTeamID, w.seasonID)
	testutil.AddKaderMember(t, db, w.bKaderID, w.bMemberID)
	testutil.AddClubFunction(t, db, w.bMemberID, "spieler")
	w.bToken = testutil.Token(t, w.bUserID, "standard", []string{"spieler"})

	// Kind von B (family_links) — Zielobjekt der /api/profile/kind/…-Routen.
	w.bChildUserID = testutil.CreateUser(t, db, "standard")
	w.bChildMemberID = testutil.CreateMember(t, db, w.bChildUserID)
	testutil.AddFamilyLink(t, db, w.bUserID, w.bChildMemberID)
	testutil.AddKaderMember(t, db, w.bKaderID, w.bChildMemberID)

	// Übungsgruppe (Kader ohne teams-Zwilling) mit B als Mitglied.
	w.bPracticeGroupID = testutil.CreatePracticeGroup(t, db, w.seasonID, "B-Übungsgruppe")
	testutil.AddKaderMember(t, db, w.bPracticeGroupID, w.bMemberID)

	// ── A: Anfragender ───────────────────────────────────────────────────────
	w.aUserID = testutil.CreateUser(t, db, "standard")
	w.aMemberID = testutil.CreateMember(t, db, w.aUserID)
	w.aTeamID = testutil.CreateTeam(t, db, "A-Mannschaft")
	w.aKaderID = testutil.CreateKader(t, db, w.aTeamID, w.seasonID)
	testutil.AddKaderMember(t, db, w.aKaderID, w.aMemberID)
	testutil.AddClubFunction(t, db, w.aMemberID, "spieler")
	w.aToken = testutil.Token(t, w.aUserID, "standard", []string{"spieler"})

	// ── T: fremder Trainer ───────────────────────────────────────────────────
	w.tUserID = testutil.CreateUser(t, db, "standard")
	w.tMemberID = testutil.CreateMember(t, db, w.tUserID)
	w.tTeamID = testutil.CreateTeam(t, db, "T-Mannschaft")
	w.tKaderID = testutil.CreateKader(t, db, w.tTeamID, w.seasonID)
	testutil.AddKaderTrainer(t, db, w.tKaderID, w.tMemberID)
	testutil.AddClubFunction(t, db, w.tMemberID, "trainer")
	w.tToken = testutil.Token(t, w.tUserID, "standard", []string{"trainer"})

	// ── E: Elternteil eines fremden Kindes ───────────────────────────────────
	w.eUserID = testutil.CreateUser(t, db, "standard")
	eChildUserID := testutil.CreateUser(t, db, "standard")
	w.eChildMemberID = testutil.CreateMember(t, db, eChildUserID)
	testutil.AddFamilyLink(t, db, w.eUserID, w.eChildMemberID)
	testutil.AddKaderMember(t, db, w.aKaderID, w.eChildMemberID)
	w.eToken = testutil.TokenWithIsParent(t, w.eUserID, "standard", []string{}, true)

	return w
}

// ins führt ein INSERT aus und liefert die vergebene ID.
func (w *objWorld) ins(query string, args ...any) int {
	w.t.Helper()
	res, err := w.db.Exec(query, args...)
	if err != nil {
		w.t.Fatalf("Fixture-INSERT fehlgeschlagen (%s): %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		w.t.Fatalf("Fixture-INSERT LastInsertId (%s): %v", query, err)
	}
	return int(id)
}

func (w *objWorld) exec(query string, args ...any) {
	w.t.Helper()
	if _, err := w.db.Exec(query, args...); err != nil {
		w.t.Fatalf("Fixture-Exec fehlgeschlagen (%s): %v", query, err)
	}
}

func uniq(prefix string) string { return fmt.Sprintf("%s-%d", prefix, objCounter.Add(1)) }

// ── Chat ─────────────────────────────────────────────────────────────────────

// newConversation legt eine Gruppen-Konversation von B an, in der ausschließlich
// B Mitglied ist.
func (w *objWorld) newConversation() int {
	id := w.ins(`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`,
		uniq("B-Gruppe"), w.bUserID)
	w.exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, id, w.bUserID)
	return id
}

// newMessage legt eine Nachricht von B in einer frischen B-Konversation an.
func (w *objWorld) newMessage() (convID, msgID int) {
	convID = w.newConversation()
	msgID = w.ins(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
		convID, w.bUserID, "Nachricht von B")
	return convID, msgID
}

// newPoll legt eine Umfrage (Nachricht + chat_polls + zwei Optionen) von B an.
func (w *objWorld) newPoll() (msgID, optionID int) {
	_, msgID = w.newMessage()
	w.exec(`INSERT INTO chat_polls (message_id, allow_multiple) VALUES (?, 0)`, msgID)
	optionID = w.ins(`INSERT INTO chat_poll_options (message_id, position, label) VALUES (?, 1, 'Ja')`, msgID)
	w.ins(`INSERT INTO chat_poll_options (message_id, position, label) VALUES (?, 2, 'Nein')`, msgID)
	return msgID, optionID
}

// newBroadcast legt eine Mitteilung von B an die Spieler von B's Mannschaft an —
// A gehört dieser Mannschaft nicht an und ist damit kein Empfänger.
func (w *objWorld) newBroadcast() int {
	id := w.ins(`INSERT INTO broadcasts (sender_id, body) VALUES (?, ?)`, w.bUserID, "Mitteilung von B")
	w.exec(`INSERT INTO broadcast_targets (broadcast_id, kind, team_id) VALUES (?, 'team_spieler', ?)`,
		id, w.bTeamID)
	w.exec(`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`, id, w.bUserID)
	return id
}

// newMedia legt ein Medium von B an, das an einer Nachricht in B's privater
// Konversation hängt (media.Serve folgt der Sichtbarkeit des Referenzobjekts).
func (w *objWorld) newMedia() int {
	mediaID := w.ins(
		`INSERT INTO media (disk_name, mime_type, size, uploaded_by) VALUES (?, 'image/png', 4, ?)`,
		uniq("b-media")+".png", w.bUserID)
	convID := w.newConversation()
	w.ins(`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`,
		convID, w.bUserID, mediaID)
	return mediaID
}

// ── Mitglieder / Profil ──────────────────────────────────────────────────────

// newPhone legt eine Telefonnummer am Mitglied von B an.
func (w *objWorld) newPhone(memberID int) int {
	return w.ins(`INSERT INTO member_phones (member_id, label, number) VALUES (?, 'privat', ?)`,
		memberID, "0711"+uniq(""))
}

// newChangeDraft legt einen Änderungsantrag auf B's Mitglied an.
//
// field_name/new_value müssen zu applyDraftToMember passen ('address' mit
// JSON-Objekt), sonst scheitert die Accept-Route mit 500 — ein Fixture-Artefakt
// statt eines Objektrechte-Befunds.
//
// old_value/new_value gehen als []byte (BLOB) in die Spalte, nicht als String:
// AcceptDraft scannt new_value in eine json.RawMessage, und der SQLite-Treiber
// liefert einen TEXT-Wert als `string` — das scheitert beim Scan ("unsupported
// Scan"). Der Produktivpfad schreibt dort eine json.RawMessage, also ebenfalls
// BLOB.
func (w *objWorld) newChangeDraft() (memberID, draftID int) {
	memberID = testutil.CreateMember(w.t, w.db, w.bUserID)
	draftID = w.ins(
		`INSERT INTO member_change_drafts (member_id, field_name, old_value, new_value, created_by_user_id)
		 VALUES (?, 'address', ?, ?, ?)`,
		memberID,
		[]byte(`{"street":"Altweg 1","zip":"70173","city":"Stuttgart"}`),
		[]byte(`{"street":"Neuweg 2","zip":"70174","city":"Stuttgart"}`),
		w.bUserID)
	return memberID, draftID
}

// newCalendarToken legt B einen Kalender-Token an (calendar_tokens ist
// UNIQUE je User, deshalb idempotent).
func (w *objWorld) newCalendarToken(userID int) string {
	token := uniq("caltoken")
	w.exec(`INSERT INTO calendar_tokens (user_id, token) VALUES (?, ?)
	        ON CONFLICT(user_id) DO UPDATE SET token = excluded.token`, userID, token)
	return token
}

// ── Termine / Dienste ────────────────────────────────────────────────────────

// newGame legt ein Spiel von B's Mannschaft an.
func (w *objWorld) newGame() int {
	return testutil.CreateGame(w.t, w.db, w.seasonID, w.bTeamID, "2026-03-14")
}

// newTrainingSession legt einen Trainingstermin in B's Kader an.
func (w *objWorld) newTrainingSession() int {
	return testutil.CreateTrainingSession(w.t, w.db, w.bTeamID, w.seasonID, "2026-03-14")
}

// newTrainingSeries legt eine Trainingsserie in B's Kader an.
func (w *objWorld) newTrainingSeries() int {
	return testutil.CreateTrainingSeries(w.t, w.db, w.bTeamID, w.seasonID, w.bUserID)
}

// newDutySlot legt einen Dienst-Slot an B's Mannschaft an (ohne Zuweisung).
func (w *objWorld) newDutySlot() int {
	typeID := testutil.CreateDutyType(w.t, w.db, uniq("Diensttyp"), 2)
	return testutil.CreateDutySlot(w.t, w.db, typeID, w.seasonID, w.bTeamID, 0, "2026-03-14")
}

// newDutyAssignment legt einen Slot mit Zuweisung an B an.
func (w *objWorld) newDutyAssignment() (slotID, assignmentID int) {
	slotID = w.newDutySlot()
	assignmentID = w.ins(
		`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, 'assigned')`,
		slotID, w.bUserID)
	w.exec(`UPDATE duty_slots SET slots_filled = slots_filled + 1 WHERE id = ?`, slotID)
	return slotID, assignmentID
}

// ── Mitfahrgelegenheiten ─────────────────────────────────────────────────────

// newCarpoolOffer legt ein „biete" von B zu einem Spiel von B's Mannschaft an.
func (w *objWorld) newCarpoolOffer() int {
	gameID := w.newGame()
	return w.ins(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt)
		 VALUES (?, ?, 'biete', 3, 'Halle')`, gameID, w.bUserID)
}

// newCarpoolPairing legt eine Paarung zwischen zwei fremden Nutzern an — weder
// die Bieter- noch die Sucher-Seite gehört A.
func (w *objWorld) newCarpoolPairing() int {
	gameID := w.newGame()
	bieteID := w.ins(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`,
		gameID, w.bUserID)
	otherUser := testutil.CreateUser(w.t, w.db, "standard")
	sucheID := w.ins(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ) VALUES (?, ?, 'suche')`,
		gameID, otherUser)
	return w.ins(
		`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status)
		 VALUES (?, ?, 'suche', 'pending')`, bieteID, sucheID)
}

// ── Abwesenheiten / Trainingstagebuch ────────────────────────────────────────

func (w *objWorld) newAbsence() int {
	return testutil.CreateAbsence(w.t, w.db, w.bMemberID, "vacation", "2026-03-01", "2026-03-10", w.bUserID)
}

func (w *objWorld) newDiaryEntry() int {
	id := testutil.CreateTrainingDiaryEntry(w.t, w.db, w.bMemberID, w.seasonID, "2026-03-01", 60, 5)
	testutil.SetTrainingDiaryProof(w.t, w.db, id, uniq("proof")+".jpg", "image/jpeg")
	return id
}

// ── Dokumente ────────────────────────────────────────────────────────────────

// newFolder legt einen Ordner von B an — ohne jede folder_permissions-Zeile für
// A. Eigentümer-Vorrang (file_folders.created_by) gibt B Lese- und Schreibrecht.
func (w *objWorld) newFolder() int {
	return testutil.CreateFolder(w.t, w.db, uniq("B-Ordner"), 0, w.bUserID)
}

func (w *objWorld) newFolderPermission(folderID int) int {
	return w.ins(
		`INSERT INTO folder_permissions (folder_id, principal_type, principal_ref, can_read, can_write)
		 VALUES (?, 'user', ?, 1, 1)`, folderID, fmt.Sprintf("%d", w.bUserID))
}

func (w *objWorld) newFile() (folderID, fileID int) {
	folderID = w.newFolder()
	fileID = testutil.CreateFile(w.t, w.db, folderID, w.bUserID, uniq("B-Datei")+".pdf")
	return folderID, fileID
}

// newEmptyKader legt einen Kader ohne Mitglieder an (eigene Mannschaft, damit
// die Team-Zuordnung stimmt). Nötig für DELETE /api/kader/{id}: ein besetzter
// Kader endet schon an der Mitglieder-Zählung mit 409 und verdeckt damit, dass
// gar kein Objektrecht geprüft wird.
func (w *objWorld) newEmptyKader() int {
	teamID := testutil.CreateTeam(w.t, w.db, uniq("B-Mannschaft"))
	return testutil.CreateKader(w.t, w.db, teamID, w.seasonID)
}

// ── Videos ───────────────────────────────────────────────────────────────────

func (w *objWorld) newVideo() int {
	return testutil.CreateVideo(w.t, w.db, w.bTeamID, w.seasonID, w.bUserID, "ready")
}

// ── Spielberichte ────────────────────────────────────────────────────────────

func (w *objWorld) newMatchReport() int {
	gameID := w.newGame()
	return testutil.CreateMatchReport(w.t, w.db, gameID, w.bUserID, 0)
}

func (w *objWorld) newMatchReportImage() (reportID, imageID int) {
	reportID = w.newMatchReport()
	imageID = w.ins(
		`INSERT INTO match_report_images (report_id, position, caption, storage_path)
		 VALUES (?, 1, 'Bild', ?)`, reportID, uniq("mr-img")+".jpg")
	return reportID, imageID
}

// ── Team-interne Objekte (Aufgaben, Strafen, Kasse) ──────────────────────────

func (w *objWorld) newResponsibilityType() int {
	return testutil.AddResponsibilityType(w.t, w.db, w.bKaderID, uniq("Aufgabe"))
}

func (w *objWorld) newResponsibility() int {
	return testutil.AssignResponsibility(w.t, w.db, w.bKaderID, w.bMemberID, uniq("Aufgabe"))
}

func (w *objWorld) newPenaltyType() int {
	return testutil.AddPenaltyType(w.t, w.db, w.bKaderID, uniq("Grund"), 500)
}

func (w *objWorld) newPenalty() int {
	return testutil.CreatePenalty(w.t, w.db, w.bKaderID, w.bMemberID, 500, uniq("Grund"), w.bMemberID)
}

func (w *objWorld) newCashbookEntry() int {
	return testutil.CreateCashbookEntry(w.t, w.db, w.bKaderID, w.bMemberID, 500, "Test")
}

// newStrafenwart / newKassenwart: B ist Amtsträger seines eigenen Kaders —
// Zielobjekt der DELETE-Routen auf penalty-wardens/treasurers.
func (w *objWorld) newStrafenwart() int {
	testutil.AppointStrafenwart(w.t, w.db, w.bKaderID, w.bMemberID)
	return w.bMemberID
}

func (w *objWorld) newKassenwart() int {
	testutil.AppointKassenwart(w.t, w.db, w.bKaderID, w.bMemberID)
	return w.bMemberID
}
