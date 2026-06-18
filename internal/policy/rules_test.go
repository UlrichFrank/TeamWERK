package policy_test

import (
	"slices"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/policy"
)

func adminP() *policy.Principal    { return &policy.Principal{UserID: 1, Role: "admin"} }
func vorstandP() *policy.Principal { return &policy.Principal{UserID: 2, Role: "standard", ClubFunctions: []string{"vorstand"}} }
func trainerP() *policy.Principal  { return &policy.Principal{UserID: 3, Role: "standard", ClubFunctions: []string{"trainer"}} }
func slP() *policy.Principal {
	return &policy.Principal{UserID: 4, Role: "standard", ClubFunctions: []string{"sportliche_leitung"}}
}
func spielerP() *policy.Principal { return &policy.Principal{UserID: 5, Role: "standard", ClubFunctions: []string{"spieler"}} }

func TestIsTrainerLike(t *testing.T) {
	if !policy.IsTrainerLike(adminP()) {
		t.Error("admin should be trainer-like")
	}
	if !policy.IsTrainerLike(trainerP()) {
		t.Error("trainer should be trainer-like")
	}
	if !policy.IsTrainerLike(slP()) {
		t.Error("sportliche_leitung should be trainer-like")
	}
	if policy.IsTrainerLike(spielerP()) {
		t.Error("spieler should not be trainer-like")
	}
	if policy.IsTrainerLike(vorstandP()) {
		t.Error("vorstand alone should not be trainer-like")
	}
}

func TestIsVorstandLike(t *testing.T) {
	if !policy.IsVorstandLike(adminP()) {
		t.Error("admin should be vorstand-like")
	}
	if !policy.IsVorstandLike(vorstandP()) {
		t.Error("vorstand should be vorstand-like")
	}
	if policy.IsVorstandLike(trainerP()) {
		t.Error("trainer should not be vorstand-like")
	}
	if policy.IsVorstandLike(spielerP()) {
		t.Error("spieler should not be vorstand-like")
	}
}

func TestCanEditMember(t *testing.T) {
	const memberUserID = 5

	if !policy.CanEditMember(adminP(), memberUserID) {
		t.Error("admin can always edit")
	}
	if !policy.CanEditMember(vorstandP(), memberUserID) {
		t.Error("vorstand can always edit")
	}
	// Owner: spielerP has UserID=5, same as memberUserID
	if !policy.CanEditMember(spielerP(), memberUserID) {
		t.Error("owner can edit own member")
	}
	// Non-owner trainer cannot edit someone else's member
	if policy.CanEditMember(trainerP(), memberUserID) {
		t.Error("trainer (non-owner) should not edit foreign member")
	}
	// Unlinked member (memberUserID=0)
	if policy.CanEditMember(spielerP(), 0) {
		t.Error("spieler should not edit unlinked member")
	}
}

func TestCanDeleteMember(t *testing.T) {
	if !policy.CanDeleteMember(adminP()) {
		t.Error("admin can delete")
	}
	if !policy.CanDeleteMember(vorstandP()) {
		t.Error("vorstand can delete")
	}
	if policy.CanDeleteMember(trainerP()) {
		t.Error("trainer cannot delete members")
	}
	if policy.CanDeleteMember(spielerP()) {
		t.Error("spieler cannot delete members")
	}
}

func TestScopeMembersQuery(t *testing.T) {
	wideWhere, wideArg := policy.ScopeMembersQuery(adminP())
	if wideWhere != "1=1" || wideArg {
		t.Error("admin should get wide search (1=1, no arg)")
	}
	wideWhere, wideArg = policy.ScopeMembersQuery(vorstandP())
	if wideWhere != "1=1" || wideArg {
		t.Error("vorstand should get wide search")
	}
	narrowWhere, narrowArg := policy.ScopeMembersQuery(trainerP())
	if narrowWhere == "1=1" || !narrowArg {
		t.Error("trainer should get narrow search with user-id arg")
	}
	narrowWhere, narrowArg = policy.ScopeMembersQuery(spielerP())
	if narrowWhere == "1=1" || !narrowArg {
		t.Error("spieler should get narrow search with user-id arg")
	}
}

func TestNavFor_ProfileVisibility(t *testing.T) {
	// Admin should NOT see Mein Profil
	adminNav := policy.NavFor(adminP())
	for _, item := range adminNav {
		if item.Route == "/profil" {
			t.Error("admin should not see /profil in nav")
		}
	}
	// Trainer SHOULD see Mein Profil
	trainerNav := policy.NavFor(trainerP())
	found := false
	for _, item := range trainerNav {
		if item.Route == "/profil" {
			found = true
		}
	}
	if !found {
		t.Error("trainer should see /profil in nav")
	}
}

func TestNavFor_MitgliederVisibility(t *testing.T) {
	// Vorstand sees /mitglieder
	vorstandNav := policy.NavFor(vorstandP())
	found := false
	for _, item := range vorstandNav {
		if item.Route == "/mitglieder" {
			found = true
		}
	}
	if !found {
		t.Error("vorstand should see /mitglieder in nav")
	}
	// Trainer does NOT see /mitglieder
	trainerNav := policy.NavFor(trainerP())
	for _, item := range trainerNav {
		if item.Route == "/mitglieder" {
			t.Error("trainer should not see /mitglieder in nav")
		}
	}
}

func TestCapabilities(t *testing.T) {
	caps := policy.Capabilities(vorstandP())
	found := false
	for _, c := range caps {
		if c == policy.CapManageMembers {
			found = true
		}
	}
	if !found {
		t.Error("vorstand should have manage_members capability")
	}

	caps = policy.Capabilities(spielerP())
	for _, c := range caps {
		if c == policy.CapManageMembers {
			t.Error("spieler should not have manage_members capability")
		}
	}

	caps = policy.Capabilities(adminP())
	found = false
	for _, c := range caps {
		if c == policy.CapImpersonate {
			found = true
		}
	}
	if !found {
		t.Error("admin should have impersonate capability")
	}
}

func hasCap(caps []string, want string) bool {
	return slices.Contains(caps, want)
}

func TestCapabilities_TrainerLike(t *testing.T) {
	// Trainer and sportliche_leitung get training/fulfill/broadcast, but NOT broadcast_all or manage_members.
	for _, p := range []*policy.Principal{trainerP(), slP()} {
		caps := policy.Capabilities(p)
		if !hasCap(caps, policy.CapManageTrainings) {
			t.Errorf("trainer-like %v should have manage_trainings", p.ClubFunctions)
		}
		if !hasCap(caps, policy.CapFulfillDuties) {
			t.Errorf("trainer-like %v should have fulfill_duties", p.ClubFunctions)
		}
		if !hasCap(caps, policy.CapBroadcast) {
			t.Errorf("trainer-like %v should have broadcast_messages", p.ClubFunctions)
		}
		if hasCap(caps, policy.CapBroadcastAll) {
			t.Errorf("trainer-like %v should NOT have broadcast_all", p.ClubFunctions)
		}
		if hasCap(caps, policy.CapManageMembers) {
			t.Errorf("trainer-like %v should NOT have manage_members", p.ClubFunctions)
		}
	}
}

func TestCapabilities_Vorstand(t *testing.T) {
	caps := policy.Capabilities(vorstandP())
	// Pure vorstand may broadcast (incl. broadcast_all) but is not trainer-like.
	if !hasCap(caps, policy.CapBroadcast) {
		t.Error("vorstand should have broadcast_messages")
	}
	if !hasCap(caps, policy.CapBroadcastAll) {
		t.Error("vorstand should have broadcast_all")
	}
	if hasCap(caps, policy.CapManageTrainings) {
		t.Error("pure vorstand should NOT have manage_trainings (trainer + sportliche_leitung only)")
	}
	if hasCap(caps, policy.CapFulfillDuties) {
		t.Error("pure vorstand should NOT have fulfill_duties")
	}
	if hasCap(caps, policy.CapManageDocuments) {
		t.Error("pure vorstand should NOT have manage_documents (admin only)")
	}
}

func TestCapabilities_AdminOnly(t *testing.T) {
	caps := policy.Capabilities(adminP())
	for _, c := range []string{policy.CapManageDocuments, policy.CapModerateChat, policy.CapBroadcastAll, policy.CapManageTrainings} {
		if !hasCap(caps, c) {
			t.Errorf("admin should have %q", c)
		}
	}
}

func TestCapabilities_Spieler(t *testing.T) {
	caps := policy.Capabilities(spielerP())
	for _, c := range []string{
		policy.CapManageTrainings, policy.CapFulfillDuties, policy.CapBroadcast,
		policy.CapBroadcastAll, policy.CapManageDocuments, policy.CapModerateChat,
	} {
		if hasCap(caps, c) {
			t.Errorf("spieler should NOT have %q", c)
		}
	}
}
