// Package permissions enthält den Backend-Permission-Matrix-Test.
// Persona-Definitionen sind spiegelbildlich zu web/src/test/personas.ts —
// bei Änderungen beide Dateien aktualisieren.
package permissions_test

// Persona beschreibt einen Test-Nutzer mit System-Rolle, Vereinsfunktionen und Eltern-Status.
type Persona struct {
	ID            string
	Role          string
	ClubFunctions []string
	IsParent      bool
}

// Personas enthält die 11 kanonischen Test-Personas.
// Quelle der Wahrheit: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §1
var Personas = []Persona{
	{ID: "admin", Role: "admin", ClubFunctions: []string{}, IsParent: false},
	{ID: "vorstand", Role: "standard", ClubFunctions: []string{"vorstand"}, IsParent: false},
	{ID: "vorstand_elternteil", Role: "standard", ClubFunctions: []string{"vorstand"}, IsParent: true},
	{ID: "vorstand_beisitzer", Role: "standard", ClubFunctions: []string{"vorstand_beisitzer"}, IsParent: false},
	{ID: "kassierer", Role: "standard", ClubFunctions: []string{"kassierer"}, IsParent: false},
	{ID: "trainer", Role: "standard", ClubFunctions: []string{"trainer"}, IsParent: false},
	{ID: "trainer_elternteil", Role: "standard", ClubFunctions: []string{"trainer"}, IsParent: true},
	{ID: "sportliche_leitung", Role: "standard", ClubFunctions: []string{"sportliche_leitung"}, IsParent: false},
	{ID: "sportliche_leitung_elternteil", Role: "standard", ClubFunctions: []string{"sportliche_leitung"}, IsParent: true},
	{ID: "spieler", Role: "standard", ClubFunctions: []string{"spieler"}, IsParent: false},
	{ID: "elternteil", Role: "standard", ClubFunctions: []string{}, IsParent: true},
}
