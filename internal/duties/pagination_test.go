package duties_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type slotListResponse struct {
	Items []struct {
		ID        int    `json:"id"`
		EventDate string `json:"event_date"`
	} `json:"items"`
	Total int `json:"total"`
}

// TestListDutySlots_Paginated: Ergebnis ist auf limit begrenzt + total;
// über offset ist auch der älteste Slot erreichbar (event_date DESC).
func TestListDutySlots_Paginated(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Slots-Team")
	dutyTypeID := createDutyType(t, db, "Kuchen", 2)
	dates := []string{"2025-10-01", "2025-11-01", "2025-12-01"}
	for _, d := range dates {
		createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, d)
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/duty-slots", h.ListSlots)
	})

	resp := testutil.Get(t, srv, "/api/duty-slots?limit=2&offset=0", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var page1 slotListResponse
	if err := json.NewDecoder(resp.Body).Decode(&page1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}
	resp.Body.Close()

	if len(page1.Items) != 2 {
		t.Errorf("expected 2 items with limit=2, got %d", len(page1.Items))
	}
	if page1.Total != 3 {
		t.Errorf("expected total=3, got %d", page1.Total)
	}
	// Sortierung event_date DESC: neuester Slot zuerst.
	if len(page1.Items) > 0 && page1.Items[0].EventDate[:10] != "2025-12-01" {
		t.Errorf("expected newest slot (2025-12-01) first, got %q", page1.Items[0].EventDate)
	}

	// Ältester Slot über offset erreichbar.
	resp2 := testutil.Get(t, srv, "/api/duty-slots?limit=2&offset=2", token)
	var page2 slotListResponse
	if err := json.NewDecoder(resp2.Body).Decode(&page2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}
	resp2.Body.Close()
	if len(page2.Items) != 1 {
		t.Fatalf("expected 1 item on page 2, got %d", len(page2.Items))
	}
	if page2.Items[0].EventDate[:10] != "2025-10-01" {
		t.Errorf("expected oldest slot (2025-10-01) via offset, got %q", page2.Items[0].EventDate)
	}
}

// TestDutyBoard_NamesWithoutHeavyFields: Assignees tragen user_id + name inline,
// aber KEIN photo_url/Kontaktfeld — auch wenn die Person ein sichtbares Foto hat.
func TestDutyBoard_NamesWithoutHeavyFields(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Board-Team")
	dutyTypeID := createDutyType(t, db, "Verkauf", 2)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2099-01-01")

	// Assignee mit Namen, hinterlegtem Foto und photo_visible=1 — das Foto
	// dürfte nach altem Verhalten inline erscheinen, jetzt nicht mehr.
	assigneeID := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(`UPDATE users SET first_name='Foto', last_name='Sichtbar', photo_path='x.jpg' WHERE id=?`, assigneeID); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO user_visibility (user_id, photo_visible) VALUES (?, 1)`, assigneeID); err != nil {
		t.Fatalf("insert user_visibility: %v", err)
	}
	insertDutyAssignment(t, db, slotID, assigneeID, "assigned")

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testServer(t, h)

	resp := testutil.Get(t, srv, "/api/duty-board", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var groups []struct {
		Slots []struct {
			ID        int              `json:"id"`
			Assignees []map[string]any `json:"assignees"`
		} `json:"slots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}

	found := false
	for _, g := range groups {
		for _, s := range g.Slots {
			if s.ID != slotID {
				continue
			}
			found = true
			if len(s.Assignees) != 1 {
				t.Fatalf("expected 1 assignee, got %d", len(s.Assignees))
			}
			a := s.Assignees[0]
			if a["name"] != "Foto Sichtbar" {
				t.Errorf("expected assignee name inline, got %v", a["name"])
			}
			if _, ok := a["user_id"]; !ok {
				t.Error("expected user_id inline")
			}
			for _, heavy := range []string{"photo_url", "phones", "emails", "email", "address"} {
				if _, ok := a[heavy]; ok {
					t.Errorf("expected heavy field %q NOT inline in board assignee", heavy)
				}
			}
		}
	}
	if !found {
		t.Fatalf("slot %d not found on board", slotID)
	}
}

// TestDutyBoard_FromToWindow: optionales Datumsfenster begrenzt nur den Umfang.
func TestDutyBoard_FromToWindow(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Window-Team")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2)
	oldSlot := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2020-01-01")
	newSlot := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2099-01-01")

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testServer(t, h)

	collectSlotIDs := func(path string) map[int]bool {
		t.Helper()
		resp := testutil.Get(t, srv, path, token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, resp.StatusCode)
		}
		defer resp.Body.Close()
		var groups []struct {
			Slots []struct {
				ID int `json:"id"`
			} `json:"slots"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		ids := map[int]bool{}
		for _, g := range groups {
			for _, s := range g.Slots {
				ids[s.ID] = true
			}
		}
		return ids
	}

	all := collectSlotIDs("/api/duty-board")
	if !all[oldSlot] || !all[newSlot] {
		t.Fatalf("expected both slots without window, got %v", all)
	}
	windowed := collectSlotIDs("/api/duty-board?from=2098-01-01")
	if windowed[oldSlot] {
		t.Error("expected old slot filtered out by ?from")
	}
	if !windowed[newSlot] {
		t.Error("expected new slot within ?from window")
	}
}

// Fehlerfall: ohne Token antworten die Routen mit 401.
func TestListDutySlots_Unauthorized(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/duty-slots", h.ListSlots)
	})

	resp := testutil.Get(t, srv, "/api/duty-slots", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}
