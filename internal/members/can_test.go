package members_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TC-MCAN-01: Vorstand list — all items carry can.edit=true, can.delete=true
func TestList_VorstandCanFlags(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})

	testutil.CreateMember(t, database, 0)
	testutil.CreateMember(t, database, 0)

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members", tok)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		Items []struct {
			Can struct {
				Edit   bool `json:"edit"`
				Delete bool `json:"delete"`
			} `json:"can"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total == 0 {
		t.Fatal("expected at least one member")
	}
	for i, item := range body.Items {
		if !item.Can.Edit {
			t.Errorf("item[%d]: expected can.edit=true for vorstand", i)
		}
		if !item.Can.Delete {
			t.Errorf("item[%d]: expected can.delete=true for vorstand", i)
		}
	}
}

// TC-MCAN-02: Trainer is gated out of /api/members (403) — scope applies when gate opens
func TestList_TrainerGated(t *testing.T) {
	database := testutil.NewDB(t)
	trainerUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members", tok)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for trainer on /api/members, got %d", res.StatusCode)
	}
}

// TC-MCAN-03: GET /api/members/{id} — vorstand sees can.edit=true, can.delete=true
func TestGetMember_VorstandCanFlags(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})

	memberUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, memberUserID)

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members/"+itoa(memberID), tok)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Can struct {
			Edit   bool `json:"edit"`
			Delete bool `json:"delete"`
		} `json:"can"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Can.Edit {
		t.Error("vorstand should see can.edit=true")
	}
	if !body.Can.Delete {
		t.Error("vorstand should see can.delete=true")
	}
}

// TC-MCAN-04: GET /api/members/{id} — admin viewing member owned by a different user
// sees can.edit=true (admin pass-through). Policy unit tests cover the owner/non-owner
// distinction for non-admin callers (see internal/policy/rules_test.go:TestCanEditMember).
func TestGetMember_AdminCanFlags(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	otherUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, otherUserID)

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members/"+itoa(memberID), tok)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Can struct {
			Edit   bool `json:"edit"`
			Delete bool `json:"delete"`
		} `json:"can"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Can.Edit {
		t.Error("admin should see can.edit=true")
	}
	if !body.Can.Delete {
		t.Error("admin should see can.delete=true")
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
