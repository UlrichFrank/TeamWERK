package metrics

import "testing"

func TestParseComplexity_CountsAndHotspots(t *testing.T) {
	raw := []byte(`{"Issues":[
		{"FromLinter":"gocyclo","Text":"cyclomatic complexity 22 of func ` + "`" + `(*Handler).Foo` + "`" + ` is high","Pos":{"Filename":"internal/foo/handler.go","Line":42}},
		{"FromLinter":"gocognit","Text":"cognitive complexity 88 of func ` + "`" + `(*Handler).Big` + "`" + ` is high (> 20)","Pos":{"Filename":"internal/foo/handler.go","Line":100}},
		{"FromLinter":"funlen","Text":"Function 'Calendar' is too long (130 > 100)","Pos":{"Filename":"internal/x/handler.go","Line":7}},
		{"FromLinter":"dupl","Text":"dup block","Pos":{"Filename":"internal/y.go","Line":1}}
	]}`)
	got, err := parseComplexity(raw)
	if err != nil {
		t.Fatalf("parseComplexity: %v", err)
	}
	if got.GoCyclo != 1 || got.GoCognit != 1 || got.FunLen != 1 || got.DuplBlocks != 1 {
		t.Errorf("counts: cyclo=%d cognit=%d funlen=%d dupl=%d (want 1/1/1/1)",
			got.GoCyclo, got.GoCognit, got.FunLen, got.DuplBlocks)
	}
	// dupl-Issues kommen NICHT in die Hotspots-Liste
	if len(got.Hotspots) != 3 {
		t.Fatalf("hotspots len: got %d want 3", len(got.Hotspots))
	}
	// Sortierung: gocognit (88) > funlen (130) > gocyclo (22)
	// Sortiert wird per hotspotScore — der zieht die erste Zahl im Text.
	// gocognit "cognitive complexity 88" → 88
	// funlen "is too long (130" → 130 (NEIN — "is too long" hat keine Zahl davor; "Function 'Calendar' is too long (130 > 100)" → erste Zahl ist 130)
	// gocyclo "cyclomatic complexity 22" → 22
	wantOrder := []string{"funlen", "gocognit", "gocyclo"}
	for i, want := range wantOrder {
		if got.Hotspots[i].Linter != want {
			t.Errorf("hotspot[%d] linter: got %s want %s", i, got.Hotspots[i].Linter, want)
		}
	}
}

func TestExtractFuncName(t *testing.T) {
	cases := map[string]string{
		"Function 'BuildRouter' is too long (299 > 100)":               "BuildRouter",
		"cognitive complexity 177 of func `(*Handler).Import` is high": "(*Handler).Import",
		"cyclomatic complexity 22 of func `applyBehavior` is high":     "applyBehavior",
		"some unrelated text": "",
	}
	for in, want := range cases {
		if got := extractFuncName(in); got != want {
			t.Errorf("extractFuncName(%q): got %q want %q", in, got, want)
		}
	}
}

func TestNormalizeRepoPath(t *testing.T) {
	cases := map[string]string{
		"../../dev/teamwerk/internal/x/y.go": "internal/x/y.go",
		"/abs/path/cmd/teamwerk/main.go":     "cmd/teamwerk/main.go",
		"internal/x/y.go":                    "internal/x/y.go",
		"./web/src/foo.ts":                   "web/src/foo.ts",
		"unknown/path.go":                    "unknown/path.go",
	}
	for in, want := range cases {
		if got := normalizeRepoPath(in); got != want {
			t.Errorf("normalizeRepoPath(%q): got %q want %q", in, got, want)
		}
	}
}
