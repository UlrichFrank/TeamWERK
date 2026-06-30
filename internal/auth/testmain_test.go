package auth

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain senkt den bcrypt-Cost in Tests auf MinCost (4 statt 10). Hintergrund:
// jeder bcrypt.GenerateFromPassword/CompareHashAndPassword skaliert exponentiell
// mit dem Cost-Faktor und wird unter dem Race-Detector zusätzlich ~40× langsamer
// — Auth-Tests gingen damit von 9 s auf 5 min hoch. Produktion bleibt unangetastet.
func TestMain(m *testing.M) {
	bcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}
