package auth

import "testing"

func TestClaims_HasAnyFunction(t *testing.T) {
	c := &Claims{ClubFunctions: []string{"trainer", "spieler"}}

	if c.HasAnyFunction() {
		t.Error("empty argument list must return false")
	}
	if c.HasAnyFunction("vorstand", "kassierer") {
		t.Error("no overlap must return false")
	}
	if !c.HasAnyFunction("trainer") {
		t.Error("single matching function must return true")
	}
	if !c.HasAnyFunction("vorstand", "spieler", "kassierer") {
		t.Error("any overlap must return true")
	}

	empty := &Claims{ClubFunctions: nil}
	if empty.HasAnyFunction("trainer") {
		t.Error("empty ClubFunctions must return false")
	}
}
