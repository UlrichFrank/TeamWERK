package metrics

import "testing"

func TestParseSize_LanguageAggregation(t *testing.T) {
	raw := []byte(`[
		{"Name":"Go","Count":3,"Code":1000,"Comment":100,"Blank":50,"Lines":1150,"Files":[
			{"Location":"internal/foo/handler.go","Code":700},
			{"Location":"internal/foo/handler_test.go","Code":200},
			{"Location":"cmd/teamwerk/main.go","Code":100}
		]},
		{"Name":"TypeScript","Count":2,"Code":500,"Comment":20,"Blank":30,"Lines":550,"Files":[]},
		{"Name":"CSS","Count":1,"Code":40,"Comment":2,"Blank":3,"Lines":45,"Files":[]}
	]`)
	got, err := parseSize(raw)
	if err != nil {
		t.Fatalf("parseSize: %v", err)
	}
	if got.GoCode != 800 {
		t.Errorf("GoCode prod LOC: got %d want 800", got.GoCode)
	}
	if got.GoTests != 200 {
		t.Errorf("GoTests LOC: got %d want 200", got.GoTests)
	}
	if got.FrontendCode != 540 {
		t.Errorf("FrontendCode (TS+CSS): got %d want 540", got.FrontendCode)
	}
	if got.TotalCode != 1540 {
		t.Errorf("TotalCode: got %d want 1540", got.TotalCode)
	}
	// 122 Kommentar / (1540 + 122) ≈ 0.0734
	if got.CommentRatio < 0.07 || got.CommentRatio > 0.08 {
		t.Errorf("CommentRatio: got %.4f want ~0.073", got.CommentRatio)
	}
}

func TestIsGoTestFile(t *testing.T) {
	cases := map[string]bool{
		"internal/foo/handler_test.go": true,
		"internal/foo/handler.go":      false,
		"cmd/teamwerk/main.go":         false,
		"some_test.go":                 true,
	}
	for in, want := range cases {
		if got := isGoTestFile(in); got != want {
			t.Errorf("isGoTestFile(%q): got %v want %v", in, got, want)
		}
	}
}
