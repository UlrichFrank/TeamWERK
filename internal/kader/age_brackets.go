package kader

type ageBracketRef struct {
	AgeClass  string
	StartYear int
	EndYear   int
}

// Reference model for 2025/26 season (start year 2025).
var ageBracketRef2025 = []ageBracketRef{
	{AgeClass: "A-Jugend", StartYear: 2006, EndYear: 2007},
	{AgeClass: "B-Jugend", StartYear: 2007, EndYear: 2008},
	{AgeClass: "C-Jugend", StartYear: 2008, EndYear: 2009},
	{AgeClass: "D-Jugend", StartYear: 2009, EndYear: 2010},
}

// ComputeAgeBrackets returns birth year ranges for each age class given the
// season's start year (e.g. 2025 for season 2025/26).
func ComputeAgeBrackets(seasonStartYear int) map[string][2]int {
	offset := seasonStartYear - 2025
	result := make(map[string][2]int, len(ageBracketRef2025))
	for _, ref := range ageBracketRef2025 {
		result[ref.AgeClass] = [2]int{ref.StartYear + offset, ref.EndYear + offset}
	}
	return result
}

// BirthYearInBracket returns true if birthYear falls within the age bracket
// for the given ageClass and season start year.
func BirthYearInBracket(birthYear int, ageClass string, seasonStartYear int) bool {
	brackets := ComputeAgeBrackets(seasonStartYear)
	r, ok := brackets[ageClass]
	if !ok {
		return false
	}
	return birthYear >= r[0] && birthYear <= r[1]
}
