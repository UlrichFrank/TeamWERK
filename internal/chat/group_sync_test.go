package chat_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// syncFixture: Trainer von T1 hat eine Gruppe aus „Spieler T1" angelegt.
type syncFixture struct {
	*tgFixture
	hub    *hub.EventHub
	server *httptest.Server
	convID int
	owner  string // Token des Erstellers
}

func setupSync(t *testing.T) *syncFixture {
	t.Helper()
	f, _ := setupTwoTeams(t)
	hb := hub.NewHub()
	h := chat.NewHandler(f.db, hb, testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations", h.CreateConversation)
		r.Post("/api/chat/conversations/{id}/sync/preview", h.SyncPreview)
		r.Post("/api/chat/conversations/{id}/sync/apply", h.SyncApply)
	})
	sf := &syncFixture{tgFixture: f, hub: hb, server: srv, owner: testutil.Token(t, f.trainerU1, "standard", nil)}
	res := testutil.Post(t, srv, "/api/chat/conversations", sf.owner, map[string]any{
		"type": "group", "name": "Spieler T1",
		"memberIds": []int{f.playerU1, f.extPlayerU1},
		"sources":   []map[string]any{{"groupType": "team", "refId": f.team1, "kind": "spieler"}},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Gruppe anlegen: %d", res.StatusCode)
	}
	sf.convID = decodeJSON[struct{ ID int }](t, res).ID
	return sf
}

func (sf *syncFixture) preview(t *testing.T, token string, body any) (*http.Response, chat.SyncPreview) {
	t.Helper()
	res := testutil.Post(t, sf.server, fmt.Sprintf("/api/chat/conversations/%d/sync/preview", sf.convID), token, body)
	if res.StatusCode != http.StatusOK {
		return res, chat.SyncPreview{}
	}
	return res, decodeJSON[chat.SyncPreview](t, res)
}

func ids(list []chat.TeamGroupMember) map[int]bool { return memberIDs(list) }

func (sf *syncFixture) addPlayerToT1(t *testing.T) int {
	t.Helper()
	u := testutil.CreateUser(t, sf.db, "standard")
	testutil.AddKaderMember(t, sf.db, sf.kader1, testutil.CreateMember(t, sf.db, u))
	return u
}

func (sf *syncFixture) addManually(t *testing.T, uid int) {
	t.Helper()
	if _, err := sf.db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, sf.convID, uid); err != nil {
		t.Fatal(err)
	}
}

func TestSyncPreview_NeuerSpielerInAdd(t *testing.T) {
	sf := setupSync(t)
	neu := sf.addPlayerToT1(t)
	_, p := sf.preview(t, sf.owner, nil)
	if !ids(p.Add)[neu] || len(p.Add) != 1 {
		t.Errorf("add = %v, erwartet genau den neuen Spieler %d", p.Add, neu)
	}
	if len(p.Remove) != 0 {
		t.Errorf("remove = %v, erwartet leer", p.Remove)
	}
	if len(p.Sources) != 1 || p.Sources[0].Label == "" || p.Blocked {
		t.Errorf("sources = %+v, blocked=%v", p.Sources, p.Blocked)
	}
}

func TestSyncPreview_AusgetretenerInRemove(t *testing.T) {
	sf := setupSync(t)
	setMemberStatusByUser(t, sf.tgFixture, sf.playerU1, "ausgetreten")
	_, p := sf.preview(t, sf.owner, nil)
	if !ids(p.Remove)[sf.playerU1] {
		t.Errorf("remove = %v, erwartet den Ausgetretenen %d", p.Remove, sf.playerU1)
	}
}

func TestSyncPreview_ManuellHinzugefuegterInRemove(t *testing.T) {
	sf := setupSync(t)
	sf.addManually(t, sf.parentU1)
	_, p := sf.preview(t, sf.owner, nil)
	if !ids(p.Remove)[sf.parentU1] {
		t.Errorf("remove = %v, erwartet den manuell Hinzugefügten", p.Remove)
	}
}

func TestSyncPreview_ErstellerNieInRemove(t *testing.T) {
	sf := setupSync(t)
	_, p := sf.preview(t, sf.owner, nil)
	if ids(p.Remove)[sf.trainerU1] {
		t.Error("Ersteller steht in remove")
	}
}

func TestSyncPreview_Ueberlappung(t *testing.T) {
	sf := setupSync(t)
	sf.addPlayerToT1(t)
	_, p := sf.preview(t, sf.owner, nil)
	if p.Sources[0].Total != 3 || p.Sources[0].AlreadyIn != 2 {
		t.Errorf("total=%d alreadyIn=%d, erwartet 3/2", p.Sources[0].Total, p.Sources[0].AlreadyIn)
	}
	var found bool
	for _, s := range p.Suggestions {
		if s.GroupType == "team" && s.RefID == sf.team1 && s.Kind == "spieler" {
			found = s.Total == 3 && s.AlreadyIn == 2
		}
	}
	if !found {
		t.Errorf("Vorschlag Spieler T1 mit 2 von 3 fehlt: %+v", p.Suggestions)
	}
}

func TestSyncPreview_LeereSourcesAendernNichts(t *testing.T) {
	sf := setupSync(t)
	sf.addPlayerToT1(t)
	_, p := sf.preview(t, sf.owner, map[string]any{"sources": []any{}})
	if len(p.Add) != 0 || len(p.Remove) != 0 || len(p.Sources) != 0 {
		t.Errorf("leere Herkunft: add=%v remove=%v sources=%v", p.Add, p.Remove, p.Sources)
	}
	if len(p.Suggestions) == 0 {
		t.Error("suggestions leer, erwartet die sichtbaren Kacheln")
	}
}

func TestSyncPreview_LeererKaderSperrt(t *testing.T) {
	sf := setupSync(t)
	if _, err := sf.db.Exec(`DELETE FROM kader_members WHERE kader_id = ?`, sf.kader1); err != nil {
		t.Fatal(err)
	}
	if _, err := sf.db.Exec(`DELETE FROM kader_extended_members WHERE kader_id = ?`, sf.kader1); err != nil {
		t.Fatal(err)
	}
	_, p := sf.preview(t, sf.owner, nil)
	if !p.Blocked || p.Sources[0].Problem != "empty" || len(p.Remove) != 0 {
		t.Errorf("blocked=%v problem=%q remove=%v, erwartet Sperre ohne Entfernungen", p.Blocked, p.Sources[0].Problem, p.Remove)
	}
}

func TestSyncPreview_UnsichtbareKachelSperrt(t *testing.T) {
	sf := setupSync(t)
	// Übergabe an den Spieler von T2, der T1 nicht sehen darf.
	if _, err := sf.db.Exec(`UPDATE conversations SET created_by = ? WHERE id = ?`, sf.playerU2, sf.convID); err != nil {
		t.Fatal(err)
	}
	sf.addManually(t, sf.playerU2)
	res, p := sf.preview(t, testutil.Token(t, sf.playerU2, "standard", nil), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, erwartet 200", res.StatusCode)
	}
	if !p.Blocked || p.Sources[0].Problem != "not_visible" || p.Sources[0].Label == "" || len(p.Remove) != 0 {
		t.Errorf("blocked=%v source=%+v remove=%v", p.Blocked, p.Sources[0], p.Remove)
	}
}

func TestSyncPreview_NichtErsteller403(t *testing.T) {
	sf := setupSync(t)
	res, _ := sf.preview(t, testutil.Token(t, sf.playerU1, "standard", nil), nil)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, erwartet 403", res.StatusCode)
	}
}

func TestSyncPreview_Unbekannt404(t *testing.T) {
	sf := setupSync(t)
	res := testutil.Post(t, sf.server, "/api/chat/conversations/999999/sync/preview", sf.owner, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, erwartet 404", res.StatusCode)
	}
}

func TestSyncPreview_Direkt400(t *testing.T) {
	sf := setupSync(t)
	res := testutil.Post(t, sf.server, "/api/chat/conversations", sf.owner, map[string]any{"type": "direct", "userId": sf.playerU1})
	direct := decodeJSON[struct{ ID int }](t, res).ID
	res = testutil.Post(t, sf.server, fmt.Sprintf("/api/chat/conversations/%d/sync/preview", direct), sf.owner, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, erwartet 400", res.StatusCode)
	}
}

func TestSyncPreview_UngueltigeKachel400(t *testing.T) {
	sf := setupSync(t)
	res, _ := sf.preview(t, sf.owner, map[string]any{"sources": []map[string]any{{"groupType": "team", "refId": sf.team1, "kind": "foobar"}}})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, erwartet 400", res.StatusCode)
	}
}

func TestSyncPreview_SchreibtNichts(t *testing.T) {
	sf := setupSync(t)
	sf.addPlayerToT1(t)
	count := func() string {
		return fmt.Sprintf("%d/%d/%d",
			countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_members`),
			countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM messages`),
			countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_sources`))
	}
	before := count()
	sf.preview(t, sf.owner, map[string]any{"sources": []map[string]any{{"groupType": "team", "refId": sf.team1, "kind": "eltern"}}})
	if after := count(); after != before {
		t.Errorf("Vorschau hat geschrieben: %s → %s", before, after)
	}
}

func (sf *syncFixture) apply(t *testing.T, token string, body any) *http.Response {
	t.Helper()
	return testutil.Post(t, sf.server, fmt.Sprintf("/api/chat/conversations/%d/sync/apply", sf.convID), token, body)
}

func (sf *syncFixture) isActive(t *testing.T, uid int) bool {
	t.Helper()
	return countFxRows(t, sf.tgFixture,
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		sf.convID, uid) == 1
}

func drain(ch chan string) []string {
	var out []string
	for {
		select {
		case ev := <-ch:
			out = append(out, ev)
		default:
			return out
		}
	}
}

func TestSyncApply_WendetAuswahlAn(t *testing.T) {
	sf := setupSync(t)
	neu := sf.addPlayerToT1(t)
	sf.addManually(t, sf.parentU1)
	res := sf.apply(t, sf.owner, map[string]any{"addUserIds": []int{neu}, "removeUserIds": []int{sf.parentU1}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, erwartet 200", res.StatusCode)
	}
	got := decodeJSON[map[string]int](t, res)
	if got["added"] != 1 || got["removed"] != 1 {
		t.Errorf("Antwort %v, erwartet added=1 removed=1", got)
	}
	if !sf.isActive(t, neu) || sf.isActive(t, sf.parentU1) {
		t.Error("Mitgliedschaften nicht wie gewählt")
	}
	if n := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND is_system = 1 AND ((sender_id = ? AND body = 'wurde hinzugefügt') OR (sender_id = ? AND body = 'wurde entfernt'))`, sf.convID, neu, sf.parentU1); n != 2 {
		t.Errorf("%d Systemnachrichten, erwartet 2", n)
	}
	if n := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_sources WHERE conversation_id = ?`, sf.convID); n != 1 {
		t.Errorf("Herkunft hat %d Zeilen, erwartet 1", n)
	}
}

func TestSyncApply_AbgewaehlteBleibt(t *testing.T) {
	sf := setupSync(t)
	sf.addManually(t, sf.parentU1)
	sf.addManually(t, sf.extParentU1)
	if res := sf.apply(t, sf.owner, map[string]any{"removeUserIds": []int{sf.parentU1}}); res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !sf.isActive(t, sf.extParentU1) {
		t.Error("abgewählte Person wurde entfernt")
	}
}

func TestSyncApply_FremdeID409OhneAenderung(t *testing.T) {
	sf := setupSync(t)
	res := sf.apply(t, sf.owner, map[string]any{
		"sources":    []map[string]any{{"groupType": "team", "refId": sf.team1, "kind": "eltern"}},
		"addUserIds": []int{sf.playerU2}, // steht in keinem Diff
	})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, erwartet 409", res.StatusCode)
	}
	if sf.isActive(t, sf.playerU2) {
		t.Error("fremde ID wurde hinzugefügt")
	}
	if n := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_sources WHERE conversation_id = ? AND kind = 'eltern'`, sf.convID); n != 0 {
		t.Error("Herkunft wurde trotz 409 geändert")
	}
}

func TestSyncApply_ErstellerEntfernen409(t *testing.T) {
	sf := setupSync(t)
	if res := sf.apply(t, sf.owner, map[string]any{"removeUserIds": []int{sf.trainerU1}}); res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, erwartet 409", res.StatusCode)
	}
	if !sf.isActive(t, sf.trainerU1) {
		t.Error("Ersteller wurde entfernt")
	}
}

func TestSyncApply_Gesperrt409(t *testing.T) {
	sf := setupSync(t)
	sf.db.Exec(`DELETE FROM kader_members WHERE kader_id = ?`, sf.kader1)
	sf.db.Exec(`DELETE FROM kader_extended_members WHERE kader_id = ?`, sf.kader1)
	if res := sf.apply(t, sf.owner, map[string]any{}); res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, erwartet 409 sync_blocked", res.StatusCode)
	}
	if !sf.isActive(t, sf.playerU1) {
		t.Error("Mitglied trotz Sperre entfernt")
	}
}

func TestSyncApply_NichtErsteller403(t *testing.T) {
	sf := setupSync(t)
	neu := sf.addPlayerToT1(t)
	if res := sf.apply(t, testutil.Token(t, sf.playerU1, "standard", nil), map[string]any{"addUserIds": []int{neu}}); res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, erwartet 403", res.StatusCode)
	}
	if sf.isActive(t, neu) {
		t.Error("Nicht-Ersteller hat hinzugefügt")
	}
}

func TestSyncApply_ReaktiviertAusgetretenesMitglied(t *testing.T) {
	sf := setupSync(t)
	sf.db.Exec(`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP WHERE conversation_id = ? AND user_id = ?`, sf.convID, sf.playerU1)
	if res := sf.apply(t, sf.owner, map[string]any{"addUserIds": []int{sf.playerU1}}); res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !sf.isActive(t, sf.playerU1) {
		t.Error("left_at nicht zurückgesetzt")
	}
}

func TestSyncApply_NurHerkunftFestlegen(t *testing.T) {
	sf := setupSync(t)
	sf.db.Exec(`DELETE FROM conversation_sources WHERE conversation_id = ?`, sf.convID)
	before := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_members WHERE left_at IS NULL`)
	res := sf.apply(t, sf.owner, map[string]any{
		"sources": []map[string]any{{"groupType": "team", "refId": sf.team1, "kind": "spieler"}},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if n := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_sources WHERE conversation_id = ?`, sf.convID); n != 1 {
		t.Errorf("Herkunft hat %d Zeilen, erwartet 1", n)
	}
	if after := countFxRows(t, sf.tgFixture, `SELECT COUNT(*) FROM conversation_members WHERE left_at IS NULL`); after != before {
		t.Errorf("Mitglieder geändert: %d → %d", before, after)
	}
}

func TestSyncApply_Events(t *testing.T) {
	sf := setupSync(t)
	neu := sf.addPlayerToT1(t)
	sf.addManually(t, sf.parentU1)
	sf.addManually(t, sf.extParentU1)
	chPlayer := sf.hub.SubscribeUser(sf.playerU1)
	chRemoved := sf.hub.SubscribeUser(sf.parentU1)
	defer sf.hub.UnsubscribeUser(sf.playerU1, chPlayer)
	defer sf.hub.UnsubscribeUser(sf.parentU1, chRemoved)

	res := sf.apply(t, sf.owner, map[string]any{"addUserIds": []int{neu}, "removeUserIds": []int{sf.parentU1, sf.extParentU1}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	newMsg := fmt.Sprintf("chat:new-message:%d", sf.convID)
	left := fmt.Sprintf("chat:member-left:%d", sf.convID)
	got := drain(chPlayer)
	if len(got) != 2 || got[0] != newMsg || got[1] != left {
		t.Errorf("aktives Mitglied bekam %v, erwartet je ein %q und %q", got, newMsg, left)
	}
	if got := drain(chRemoved); len(got) != 1 || got[0] != left {
		t.Errorf("Entfernter bekam %v, erwartet genau ein %q", got, left)
	}
}
