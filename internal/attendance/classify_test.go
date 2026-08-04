package attendance

import "testing"

func ptrBool(b bool) *bool { return &b }

func TestClassify_PresentTrue(t *testing.T) {
	if got := Classify(ptrBool(true), false); got != CategoryPresent {
		t.Errorf("expected present, got %s", got)
	}
}

func TestClassify_PresentFalse(t *testing.T) {
	if got := Classify(ptrBool(false), false); got != CategoryMissed {
		t.Errorf("expected missed, got %s", got)
	}
}

func TestClassify_DeclinedWithAbsence(t *testing.T) {
	// Automatische Absage aus einer erfassten Abwesenheit (absence_id gesetzt).
	if got := Classify(nil, true); got != CategoryExcused {
		t.Errorf("expected excused, got %s", got)
	}
}

func TestClassify_DeclinedWithoutAbsenceIsExcused(t *testing.T) {
	// Manuelle Absage des Mitglieds ohne hinterlegte Abwesenheit: zählt
	// fachlich ebenso als entschuldigtes Fehlen wie die automatische Absage.
	if got := Classify(nil, true); got != CategoryExcused {
		t.Errorf("manuelle Absage ohne absence_id ist entschuldigt, expected excused, got %s", got)
	}
}

func TestClassify_NoResponseNoAttendance(t *testing.T) {
	if got := Classify(nil, false); got != CategoryUnknown {
		t.Errorf("expected unknown for data hole, got %s", got)
	}
}

func TestClassify_AttendanceOverridesAutoDecline(t *testing.T) {
	// Spieler war doch da, obwohl eine Absage vorliegt → present gewinnt.
	if got := Classify(ptrBool(true), true); got != CategoryPresent {
		t.Errorf("attendance must win over auto-decline, got %s", got)
	}
}

func TestClassify_MissedOverridesDecline(t *testing.T) {
	// Explizit erfasstes Fehlen gewinnt ebenfalls über eine Absage. Dass ein
	// solcher Datensatz für abgesagte Mitglieder gar nicht erst entsteht,
	// stellen die SaveAttendances-Handler sicher (games/trainings).
	if got := Classify(ptrBool(false), true); got != CategoryMissed {
		t.Errorf("explicit present=false must win over decline, got %s", got)
	}
}
