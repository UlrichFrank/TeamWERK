package kader

type ageBracketRef struct {
	AgeClass  string
	StartYear int
	EndYear   int
}

// Reference model for 2025/26 season (start year 2025).
var ageBracketRef2025 = []ageBracketRef{
	{AgeClass: "A-Jugend", StartYear: 2007, EndYear: 2008},
	{AgeClass: "B-Jugend", StartYear: 2009, EndYear: 2010},
	{AgeClass: "C-Jugend", StartYear: 2011, EndYear: 2012},
	{AgeClass: "D-Jugend", StartYear: 2013, EndYear: 2014},
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

// trainingGroupYearCount is how many birth years are offered for training-group
// kader (Förderkader/Perspektivkader). The list starts at the YOUNGER D-Jugend
// year: the Perspektivkader deliberately overlaps D-Jugend there (a single
// Jahrgang), the Förderkader continue with the years below. Extra buffer years
// cover even younger groups; the concrete Jahrgang is still chosen per kader via
// dedicated_birth_year.
const trainingGroupYearCount = 6

// TrainingGroupCandidateYears returns the selectable birth years for
// training-group kader, computed relative to the D-Jugend bracket so they shift
// with the season like A–D. The first year is the younger D-Jugend Jahrgang
// (Perspektivkader — overlaps D-Jugend), the rest are progressively younger
// (Förderkader 1/2, …). For 2025/26 (D-Jugend = 2013/2014) this yields
// 2014, 2015, 2016, … (Perspektivkader = 2014, Förderkader 1 = 2015, 2 = 2016).
func TrainingGroupCandidateYears(seasonStartYear int) []int {
	dYoungest := ComputeAgeBrackets(seasonStartYear)["D-Jugend"][1] // e.g. 2014 for 2025/26
	years := make([]int, trainingGroupYearCount)
	for i := range years {
		years[i] = dYoungest + i
	}
	return years
}
