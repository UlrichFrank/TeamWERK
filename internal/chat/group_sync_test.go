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
