package attendance

import "testing"

func ptrBool(b bool) *bool { return &b }

func TestClassify_PresentTrue(t *testing.T) {
	if got := Classify(ptrBool(true), false, false); got != CategoryPresent {
		t.Errorf("expected present, got %s", got)
	}
}

func TestClassify_PresentFalse(t *testing.T) {
	if got := Classify(ptrBool(false), false, false); got != CategoryMissed {
		t.Errorf("expected missed, got %s", got)
	}
}

func TestClassify_DeclinedWithAbsence(t *testing.T) {
	if got := Classify(nil, true, true); got != CategoryExcused {
		t.Errorf("expected excused, got %s", got)
	}
}

func TestClassify_DeclinedWithoutAbsenceIsUnknown(t *testing.T) {
	if got := Classify(nil, true, false); got != CategoryUnknown {
		t.Errorf("declined ohne absence_id ist keine Entschuldigung, expected unknown, got %s", got)
	}
}

func TestClassify_NoResponseNoAttendance(t *testing.T) {
	if got := Classify(nil, false, false); got != CategoryUnknown {
		t.Errorf("expected unknown for data hole, got %s", got)
	}
}

func TestClassify_AttendanceOverridesAutoDecline(t *testing.T) {
	// Spieler war doch da, obwohl eine Abwesenheit vorliegt → present gewinnt.
	if got := Classify(ptrBool(true), true, true); got != CategoryPresent {
		t.Errorf("attendance must win over auto-decline, got %s", got)
	}
}
