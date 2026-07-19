package kader

import "testing"

func TestComputeAgeBrackets(t *testing.T) {
	tests := []struct {
		seasonStartYear int
		ageClass        string
		wantMin         int
		wantMax         int
	}{
		// Reference season 2025/26 — DHB-konform, non-overlapping 2-year ranges
		{2025, "A-Jugend", 2007, 2008},
		{2025, "B-Jugend", 2009, 2010},
		{2025, "C-Jugend", 2011, 2012},
		{2025, "D-Jugend", 2013, 2014},
		// Next season 2026/27 — each class shifts by +1
		{2026, "A-Jugend", 2008, 2009},
		{2026, "B-Jugend", 2010, 2011},
		{2026, "C-Jugend", 2012, 2013},
		{2026, "D-Jugend", 2014, 2015},
		// Prior season 2024/25 — each class shifts by -1
		{2024, "A-Jugend", 2006, 2007},
		{2024, "D-Jugend", 2012, 2013},
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
	// 2025/26: A-Jugend = 2007-2008
	if !BirthYearInBracket(2007, "A-Jugend", 2025) {
		t.Error("2007 should be in A-Jugend 2025/26")
	}
	if !BirthYearInBracket(2008, "A-Jugend", 2025) {
		t.Error("2008 should be in A-Jugend 2025/26")
	}
	if BirthYearInBracket(2009, "A-Jugend", 2025) {
		t.Error("2009 should NOT be in A-Jugend 2025/26")
	}
	if BirthYearInBracket(2006, "A-Jugend", 2025) {
		t.Error("2006 should NOT be in A-Jugend 2025/26")
	}
	// Unknown class
	if BirthYearInBracket(2007, "E-Jugend", 2025) {
		t.Error("unknown class should return false")
	}
}

func TestTrainingGroupCandidateYears(t *testing.T) {
	// 2025/26: D-Jugend = 2013/2014. Perspektivkader overlaps the younger D year
	// (2014), Förderkader 1/2 continue below → 2014..2019 for count=6.
	got := TrainingGroupCandidateYears(2025)
	want := []int{2014, 2015, 2016, 2017, 2018, 2019}
	if len(got) != len(want) {
		t.Fatalf("2025: got %d years %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("2025: got[%d]=%d, want %d (full: %v)", i, got[i], want[i], got)
		}
	}

	dBracket := ComputeAgeBrackets(2025)["D-Jugend"] // [2013, 2014]
	// Perspektivkader = younger D-Jugend Jahrgang (overlaps D-Jugend).
	if got[0] != dBracket[1] {
		t.Errorf("Perspektivkader: got %d, want %d (younger D-Jugend year)", got[0], dBracket[1])
	}
	if got[1] != 2015 { // Förderkader 1
		t.Errorf("Förderkader 1: got %d, want 2015", got[1])
	}
	if got[2] != 2016 { // Förderkader 2
		t.Errorf("Förderkader 2: got %d, want 2016", got[2])
	}

	// The Perspektivkader Jahrgang must lie INSIDE the D-Jugend bracket (overlap),
	// but the candidates must never include the older D-Jugend year.
	if got[0] < dBracket[0] || got[0] > dBracket[1] {
		t.Errorf("Perspektivkader year %d must be within D-Jugend [%d,%d]", got[0], dBracket[0], dBracket[1])
	}
	for _, y := range got {
		if y == dBracket[0] {
			t.Errorf("candidate year %d must not be the older D-Jugend year", y)
		}
	}

	// Shifts with the season by +1 (2026/27: D-Jugend = 2014/2015 → Perspektiv 2015).
	if got26 := TrainingGroupCandidateYears(2026); got26[0] != 2015 {
		t.Errorf("2026/27: Perspektivkader got %d, want 2015", got26[0])
	}
}

func TestNoBracketOverlap(t *testing.T) {
	for _, year := range []int{2024, 2025, 2026} {
		brackets := ComputeAgeBrackets(year)
		seen := map[int]string{}
		for class, r := range brackets {
			for by := r[0]; by <= r[1]; by++ {
				if prev, ok := seen[by]; ok {
					t.Errorf("season %d: birth year %d appears in both %s and %s", year, by, prev, class)
				}
				seen[by] = class
			}
		}
	}
}
