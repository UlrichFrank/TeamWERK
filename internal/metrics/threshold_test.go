package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadThresholds_ParseAndIgnoreUnknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "thresholds.yml")
	content := `# Kommentar wird ignoriert
gocyclo_max: 12
gocognit_max:   35   # inline comment
funlen_max: 8
dupl_go_max: 4
dupl_frontend_max_pct: 5.0
go_coverage_min_pct: 40.5
frontend_coverage_min_pct: 15
lint_issues_max: 7
unknown_key: 99
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadThresholds(path)
	if err != nil {
		t.Fatalf("loadThresholds: %v", err)
	}
	want := Thresholds{
		GoCycloMax: 12, GoCognitMax: 35, FunLenMax: 8, DuplGoMax: 4,
		DuplFrontendMaxPc: 5.0, GoCoverageMinPct: 40.5, FECoverageMinPct: 15,
		LintIssuesMax: 7,
	}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestCompareThresholds_AllGreen(t *testing.T) {
	rep := Report{
		Complexity: ComplexityReport{GoCyclo: 5, GoCognit: 5, FunLen: 2, DuplBlocks: 1},
		Coverage:   CoverageReport{GoPercent: 60, FrontendPercent: 30, FrontendOK: true},
		Duplic:     DuplicationReport{FrontendPercent: 2.0},
		LintDens:   LintDensityReport{Issues: 1},
	}
	t1 := Thresholds{
		GoCycloMax: 10, GoCognitMax: 10, FunLenMax: 5, DuplGoMax: 3,
		DuplFrontendMaxPc: 5, GoCoverageMinPct: 50, FECoverageMinPct: 20,
		LintIssuesMax: 5,
	}
	if vs := compareThresholds(rep, t1); len(vs) != 0 {
		t.Errorf("want no violations, got %d: %+v", len(vs), vs)
	}
}

func TestCompareThresholds_Violations(t *testing.T) {
	rep := Report{
		Complexity: ComplexityReport{GoCyclo: 20, GoCognit: 5, FunLen: 2, DuplBlocks: 1},
		Coverage:   CoverageReport{GoPercent: 30, FrontendPercent: 10, FrontendOK: true},
		Duplic:     DuplicationReport{FrontendPercent: 12.0},
		LintDens:   LintDensityReport{Issues: 50},
	}
	t1 := Thresholds{
		GoCycloMax: 10, GoCognitMax: -1, FunLenMax: -1, DuplGoMax: -1,
		DuplFrontendMaxPc: 5, GoCoverageMinPct: 50, FECoverageMinPct: 20,
		LintIssuesMax: 5,
	}
	vs := compareThresholds(rep, t1)
	gotKeys := map[string]bool{}
	for _, v := range vs {
		gotKeys[v.Key] = true
	}
	wantKeys := []string{"gocyclo", "dupl_frontend_pct", "go_coverage_pct", "frontend_coverage_pct", "lint_issues"}
	for _, k := range wantKeys {
		if !gotKeys[k] {
			t.Errorf("missing violation for key %q (got %v)", k, gotKeys)
		}
	}
	// Keys mit limit=-1 dürfen NICHT auftauchen.
	for _, k := range []string{"gocognit", "funlen", "dupl_go"} {
		if gotKeys[k] {
			t.Errorf("unexpected violation for key %q (limit was -1)", k)
		}
	}
}

func TestCompareThresholds_FrontendCoverageSkippedWhenNotCollected(t *testing.T) {
	rep := Report{Coverage: CoverageReport{GoPercent: 60, FrontendOK: false}}
	t1 := Thresholds{GoCycloMax: -1, FECoverageMinPct: 99, GoCoverageMinPct: -1}
	for _, v := range compareThresholds(rep, t1) {
		if v.Key == "frontend_coverage_pct" {
			t.Errorf("frontend coverage check should be skipped when not collected")
		}
	}
}
