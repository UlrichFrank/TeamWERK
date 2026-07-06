package members_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestList_LimitClamp verifies the pagination clamp: a non-positive limit
// (?limit=0 or ?limit=-1) must not degrade into "empty list" (LIMIT 0) or
// "unbounded" (SQLite treats negative LIMIT as no limit). Both are clamped to
// the default 50, and an over-large limit is capped at 200.
func TestList_LimitClamp(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	// 60 members: more than the default page (50) but fewer than the cap (200).
	const created = 60
	for i := 0; i < created; i++ {
		testutil.CreateMember(t, database, 0)
	}

	srv := newMembersServer(t, database)

	cases := []struct {
		name  string
		query string
	}{
		{"limit=0", "/api/members?limit=0"},
		{"limit=-1", "/api/members?limit=-1"},
		{"limit=99999", "/api/members?limit=99999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.Get(t, srv, tc.query, tok)
			if res.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", res.StatusCode)
			}
			lr := decodeList(t, res)
			if lr.Total != created {
				t.Errorf("expected total=%d, got %d", created, lr.Total)
			}
			// limit=0 must NOT yield an empty page despite total>0.
			if len(lr.Items) == 0 {
				t.Errorf("%s: expected a non-empty page, got 0 items", tc.name)
			}
			// A non-positive limit clamps to the default 50 (not unbounded → not 60).
			// An over-large limit caps at 200 but there are only 60 members, so it
			// returns all 60. Either way the page is bounded and non-empty.
			if len(lr.Items) > 200 {
				t.Errorf("%s: page must be bounded to <=200, got %d items", tc.name, len(lr.Items))
			}
		})
	}
}

// TestList_LimitClampDefault asserts the concrete default: with 60 members and a
// clamped limit the returned page holds exactly the default 50, proving the
// negative/zero limit did NOT fall through to SQLite's unbounded behavior (which
// would return all 60).
func TestList_LimitClampDefault(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	const created = 60
	for i := 0; i < created; i++ {
		testutil.CreateMember(t, database, 0)
	}

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?limit=-1", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != created {
		t.Errorf("expected total=%d, got %d", created, lr.Total)
	}
	if len(lr.Items) != 50 {
		t.Errorf("limit=-1 must clamp to default 50, got %d items (unbounded would be %d)", len(lr.Items), created)
	}
}
