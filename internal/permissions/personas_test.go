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

// Personas enthält die 12 kanonischen Test-Personas.
// Quelle der Wahrheit: openspec/specs/permissions/spec.md, Requirement „Persona-Definition".
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
	// medien (Migration 024): Spielbericht-Freigabe. Ohne diese Persona wäre das
	// Freigeber-Tier RequireClubFunction("medien","vorstand") nur über den
	// Admin-Bypass geprüft. Bewusst am Ende, damit die Token-User-IDs (i+100)
	// der bestehenden Personas stabil bleiben.
	{ID: "medien", Role: "standard", ClubFunctions: []string{"medien"}, IsParent: false},
}
