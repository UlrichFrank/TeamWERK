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

func TestClaims_CanOverrideRSVPCutoff(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		functions []string
		want      bool
	}{
		{"admin without functions", "admin", nil, true},
		{"admin with kassierer", "admin", []string{"kassierer"}, true},
		{"standard + vorstand", "standard", []string{"vorstand"}, true},
		{"standard + trainer", "standard", []string{"trainer"}, true},
		{"standard + sportliche_leitung", "standard", []string{"sportliche_leitung"}, true},
		{"standard + vorstand_beisitzer", "standard", []string{"vorstand_beisitzer"}, false},
		{"standard + kassierer only", "standard", []string{"kassierer"}, false},
		{"standard + spieler", "standard", []string{"spieler"}, false},
		{"standard without functions", "standard", nil, false},
		{"standard + multiple incl. trainer", "standard", []string{"spieler", "trainer"}, true},
		{"standard + multiple excl. override roles", "standard", []string{"spieler", "kassierer"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Claims{Role: tc.role, ClubFunctions: tc.functions}
			if got := c.CanOverrideRSVPCutoff(); got != tc.want {
				t.Errorf("CanOverrideRSVPCutoff() = %v, want %v", got, tc.want)
			}
		})
	}
}
