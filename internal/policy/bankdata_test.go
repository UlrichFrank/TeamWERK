package policy_test

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/policy"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func p(userID int, role string, fns ...string) *policy.Principal {
	return &policy.Principal{UserID: userID, Role: role, ClubFunctions: fns}
}

func TestCanDecryptBankData(t *testing.T) {
	db := testutil.NewDB(t)

	adminID := testutil.CreateUser(t, db, "admin")
	vorstandID := testutil.CreateUser(t, db, "standard")
	kassiererID := testutil.CreateUser(t, db, "standard")
	trainerID := testutil.CreateUser(t, db, "standard")
	ownerUserID := testutil.CreateUser(t, db, "standard")
	parentUserID := testutil.CreateUser(t, db, "standard")
	strangerID := testutil.CreateUser(t, db, "standard")

	// Mitglied gehört ownerUserID; childMember ist account-los und mit parentUserID verlinkt.
	ownMember := testutil.CreateMember(t, db, ownerUserID)
	childMember := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`,
		parentUserID, childMember); err != nil {
		t.Fatal(err)
	}

	allowed := []struct {
		name string
		p    *policy.Principal
		mem  int
	}{
		{"admin", p(adminID, "admin"), ownMember},
		{"vorstand", p(vorstandID, "standard", "vorstand"), ownMember},
		{"kassierer", p(kassiererID, "standard", "kassierer"), ownMember},
		{"Eigentümer", p(ownerUserID, "standard", "spieler"), ownMember},
		{"Elternteil", p(parentUserID, "standard"), childMember},
	}
	for _, c := range allowed {
		if !policy.CanDecryptBankData(db, c.p, c.mem) {
			t.Errorf("%s sollte entschlüsseln dürfen", c.name)
		}
	}

	denied := []struct {
		name string
		p    *policy.Principal
		mem  int
	}{
		{"Trainer", p(trainerID, "standard", "trainer"), ownMember},
		{"fremdes Mitglied", p(strangerID, "standard", "spieler"), ownMember},
		{"fremdes Elternteil", p(parentUserID, "standard"), ownMember},
		{"Nichtberechtigter für fremdes Kind", p(strangerID, "standard"), childMember},
	}
	for _, c := range denied {
		if policy.CanDecryptBankData(db, c.p, c.mem) {
			t.Errorf("%s sollte NICHT entschlüsseln dürfen", c.name)
		}
	}
}
