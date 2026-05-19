package kader

import "testing"

func TestComputeAgeBrackets(t *testing.T) {
	tests := []struct {
		seasonStartYear int
		ageClass        string
		wantMin         int
		wantMax         int
	}{
		// Reference season 2025/26
		{2025, "A-Jugend", 2006, 2007},
		{2025, "B-Jugend", 2007, 2008},
		{2025, "C-Jugend", 2008, 2009},
		{2025, "D-Jugend", 2009, 2010},
		// Next season 2026/27 — each class shifts by +1
		{2026, "A-Jugend", 2007, 2008},
		{2026, "B-Jugend", 2008, 2009},
		{2026, "C-Jugend", 2009, 2010},
		{2026, "D-Jugend", 2010, 2011},
		// 2024/25 — each class shifts by -1
		{2024, "A-Jugend", 2005, 2006},
		{2024, "D-Jugend", 2008, 2009},
	}

	for _, tc := range tests {
		brackets := ComputeAgeBrackets(tc.seasonStartYear)
		got, ok := brackets[tc.ageClass]
		if !ok {
			t.Errorf("season %d: missing bracket for %s", tc.seasonStartYear, tc.ageClass)
			continue
		}
		if got[0] != tc.wantMin || got[1] != tc.wantMax {
			t.Errorf("season %d, %s: got [%d,%d], want [%d,%d]",
				tc.seasonStartYear, tc.ageClass, got[0], got[1], tc.wantMin, tc.wantMax)
		}
	}
}

func TestBirthYearInBracket(t *testing.T) {
	// 2025/26: A-Jugend = 2006-2007
	if !BirthYearInBracket(2006, "A-Jugend", 2025) {
		t.Error("2006 should be in A-Jugend 2025/26")
	}
	if !BirthYearInBracket(2007, "A-Jugend", 2025) {
		t.Error("2007 should be in A-Jugend 2025/26")
	}
	if BirthYearInBracket(2008, "A-Jugend", 2025) {
		t.Error("2008 should NOT be in A-Jugend 2025/26")
	}
	if BirthYearInBracket(2005, "A-Jugend", 2025) {
		t.Error("2005 should NOT be in A-Jugend 2025/26")
	}
	// Unknown class
	if BirthYearInBracket(2007, "E-Jugend", 2025) {
		t.Error("unknown class should return false")
	}
}
