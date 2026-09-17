package chat_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func listConversations(t *testing.T, srv *httptest.Server, token string) []convListItem {
	t.Helper()
	res := testutil.Get(t, srv, "/api/chat/conversations", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/chat/conversations: expected 200, got %d", res.StatusCode)
	}
	var items []convListItem
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return items
}

func pinConv(t *testing.T, srv *httptest.Server, convID int, token string) {
	t.Helper()
	res := testutil.Put(t, srv, "/api/chat/conversations/"+strconv.Itoa(convID)+"/pin", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT .../pin (conv %d): expected 204, got %d", convID, res.StatusCode)
	}
}

// TestPin_HappyPath deckt tasks.md 6.1: Pin durch ein aktives Mitglied setzt
// pinned=true und die Konversation erscheint vor allen ungepinnten.
func TestPin_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	convOld := createGroupConv(t, db, "Alt", owner)
	convToPin := createGroupConv(t, db, "Wird gepinnt", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, convToPin, token)

	items := listConversations(t, srv, token)
	if len(items) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(items))
	}
	if items[0].ID != convToPin || !items[0].Pinned {
		t.Errorf("expected convToPin (%d) pinned at index 0, got %+v", convToPin, items[0])
	}
	if items[1].ID != convOld || items[1].Pinned {
		t.Errorf("expected convOld (%d) unpinned at index 1, got %+v", convOld, items[1])
	}
}

// TestPin_NonMemberForbidden deckt tasks.md 6.2 (zusätzlich zur
// Objektrechte-Matrix): ein Nicht-Mitglied kann nicht pinnen.
func TestPin_NonMemberForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "Fremd", owner)
	outsiderToken := testutil.Token(t, outsider, "standard", nil)

	res := testutil.Put(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/pin", outsiderToken, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member pin, got %d", res.StatusCode)
	}
}

// TestUnpin_ResetsToActivitySort deckt tasks.md 6.7.
func TestUnpin_ResetsToActivitySort(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "Chat", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, conv, token)

	res := testutil.Delete(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/pin", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE .../pin: expected 204, got %d", res.StatusCode)
	}

	items := listConversations(t, srv, token)
	if len(items) != 1 || items[0].Pinned {
		t.Fatalf("expected unpinned conversation after unpin, got %+v", items)
	}
}

// TestPin_Idempotent_DoesNotMovePinOrder deckt tasks.md 6.5: erneutes Pinnen
// einer bereits gepinnten Konversation ändert die Reihenfolge nicht.
func TestPin_Idempotent_DoesNotMovePinOrder(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	convA := createGroupConv(t, db, "A", owner)
	convB := createGroupConv(t, db, "B", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, convA, token)
	pinConv(t, srv, convB, token)
	// Erneutes Pinnen von A darf es nicht hinter B schieben.
	pinConv(t, srv, convA, token)

	items := listConversations(t, srv, token)
	if len(items) != 2 || items[0].ID != convA || items[1].ID != convB {
		t.Fatalf("expected order [A, B] unchanged after re-pin, got %+v", items)
	}
}

// TestPin_AppendsToEndOfPinnedOrder deckt tasks.md 6.4.
func TestPin_AppendsToEndOfPinnedOrder(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	convA := createGroupConv(t, db, "A", owner)
	convB := createGroupConv(t, db, "B", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, convA, token)
	pinConv(t, srv, convB, token)

	items := listConversations(t, srv, token)
	if len(items) != 2 || items[0].ID != convA || items[1].ID != convB {
		t.Fatalf("expected order [A, B] (B appended after A), got %+v", items)
	}
}

// TestPin_NotVisibleToOtherMembers deckt tasks.md 6.6.
func TestPin_NotVisibleToOtherMembers(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "Gemeinsam", userA, userB)
	tokenA := testutil.Token(t, userA, "standard", nil)
	tokenB := testutil.Token(t, userB, "standard", nil)

	pinConv(t, srv, conv, tokenA)

	itemsA := listConversations(t, srv, tokenA)
	if len(itemsA) != 1 || !itemsA[0].Pinned {
		t.Fatalf("expected pinned=true for user A, got %+v", itemsA)
	}

	itemsB := listConversations(t, srv, tokenB)
	if len(itemsB) != 1 || itemsB[0].Pinned {
		t.Fatalf("expected pinned=false for user B (private pin), got %+v", itemsB)
	}
}

// TestReorderPinned_HappyPath deckt tasks.md 6.3 (Erfolgsfall).
func TestReorderPinned_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	convA := createGroupConv(t, db, "A", owner)
	convB := createGroupConv(t, db, "B", owner)
	convC := createGroupConv(t, db, "C", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, convA, token)
	pinConv(t, srv, convB, token)
	pinConv(t, srv, convC, token)

	res := testutil.Put(t, srv, "/api/chat/conversations/pinned-order", token,
		map[string]any{"order": []int{convC, convA, convB}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT .../pinned-order: expected 204, got %d", res.StatusCode)
	}

	items := listConversations(t, srv, token)
	want := []int{convC, convA, convB}
	if len(items) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(items))
	}
	for i, id := range want {
		if items[i].ID != id {
			t.Errorf("index %d: expected conv %d, got %d", i, id, items[i].ID)
		}
	}
}

// TestDeleteConversation_StillWorksAlongsidePinRoutes deckt tasks.md 6.8: das
// Löschen (DELETE /api/chat/conversations/{id}) bleibt unverändert funktions-
// fähig, insbesondere kollidiert es nicht mit der neuen Route
// DELETE /api/chat/conversations/{id}/pin (reiner Regressionsschutz für den
// Router-Umbau in diesem Change — der Handler selbst wurde nicht angefasst).
func TestDeleteConversation_StillWorksAlongsidePinRoutes(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "Zu löschen", owner)
	token := testutil.Token(t, owner, "standard", nil)

	res := testutil.Delete(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/pin", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE .../pin (unpinned conv, no-op): expected 204, got %d", res.StatusCode)
	}

	res = testutil.Delete(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /api/chat/conversations/{id}: expected 204, got %d", res.StatusCode)
	}

	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = ?`, conv).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Errorf("expected conversation to be deleted (no members left), got %d rows", remaining)
	}
}

// TestReorderPinned_MismatchedSetRejected deckt tasks.md 6.3 (Fehlerfall).
func TestReorderPinned_MismatchedSetRejected(t *testing.T) {
	db := testutil.NewDB(t)
	_, mount := newChatServer(t, db)
	srv := testutil.NewServer(t, func(r chi.Router) { mount(r) })

	owner := testutil.CreateUser(t, db, "standard")
	convA := createGroupConv(t, db, "A", owner)
	convB := createGroupConv(t, db, "B", owner)
	convUnpinned := createGroupConv(t, db, "Ungepinnt", owner)
	token := testutil.Token(t, owner, "standard", nil)

	pinConv(t, srv, convA, token)
	pinConv(t, srv, convB, token)

	// Enthält eine nicht gepinnte Konversation statt convB.
	res := testutil.Put(t, srv, "/api/chat/conversations/pinned-order", token,
		map[string]any{"order": []int{convA, convUnpinned}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for mismatched pin set, got %d", res.StatusCode)
	}

	// pin_order darf durch den abgelehnten Request nicht verändert worden sein.
	items := listConversations(t, srv, token)
	if len(items) != 3 || items[0].ID != convA || items[1].ID != convB {
		t.Fatalf("expected unchanged order [A, B, Unpinned...] after rejected reorder, got %+v", items)
	}
}
