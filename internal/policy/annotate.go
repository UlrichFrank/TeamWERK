package policy

// CanFlags holds the per-resource action flags included in API responses as "can".
type CanFlags struct {
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// GameCanFlags extends CanFlags with game-specific actions.
type GameCanFlags struct {
	Edit         bool `json:"edit"`
	Delete       bool `json:"delete"`
	ManageLineup bool `json:"manage_lineup"`
}

// DutyCanFlags holds action flags for duty slots.
type DutyCanFlags struct {
	Edit    bool `json:"edit"`
	Delete  bool `json:"delete"`
	Fulfill bool `json:"fulfill"`
}

// MemberCan returns the CanFlags for a member resource.
// memberUserID is the user_id column of the member row (0 if unlinked).
func MemberCan(p *Principal, memberUserID int) CanFlags {
	return CanFlags{
		Edit:   CanEditMember(p, memberUserID),
		Delete: CanDeleteMember(p),
	}
}

// GameCan returns the GameCanFlags for a game resource.
func GameCan(p *Principal) GameCanFlags {
	canEdit := CanEditGame(p)
	return GameCanFlags{
		Edit:         canEdit,
		Delete:       canEdit,
		ManageLineup: canEdit,
	}
}

// DutyCan returns the DutyCanFlags for a duty slot resource.
func DutyCan(p *Principal) DutyCanFlags {
	return DutyCanFlags{
		Edit:    CanEditDutySlot(p),
		Delete:  CanEditDutySlot(p),
		Fulfill: CanFulfillAssignment(p),
	}
}
