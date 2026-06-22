package metrics

import (
	"encoding/json"
	"os"
	"os/exec"
)

// LintDensityReport zählt Issues aus der Haupt-.golangci.yml (Gate) und
// rechnet sie pro 1000 Go-Code-LOC um. Aussage: "wieviel Restschuld pro kLOC".
type LintDensityReport struct {
	Issues        int
	GoCodeKLOC    float64
	IssuesPerKLOC float64
	Collected     bool
}

func collectLintDensity(repoRoot string, goCodeLOC int) LintDensityReport {
	rep := LintDensityReport{}
	if goCodeLOC > 0 {
		rep.GoCodeKLOC = float64(goCodeLOC) / 1000.0
	}

	tmp, err := os.CreateTemp("", "metrics-lint-*.json")
	if err != nil {
		return rep
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	cmd := exec.Command("golangci-lint", "run",
		"--issues-exit-code", "0",
		"--show-stats=false",
		"--output.text.path=/dev/null",
		"--output.json.path="+tmp.Name(),
		"./...")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return rep
	}
	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return rep
	}
	var lr lintReport
	if err := json.Unmarshal(raw, &lr); err != nil {
		return rep
	}
	rep.Issues = len(lr.Issues)
	rep.Collected = true
	if rep.GoCodeKLOC > 0 {
		rep.IssuesPerKLOC = float64(rep.Issues) / rep.GoCodeKLOC
	}
	return rep
}
