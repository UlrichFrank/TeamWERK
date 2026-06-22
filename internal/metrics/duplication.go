package metrics

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

// DuplicationReport bündelt Backend- und Frontend-Duplikation.
// Go-Duplikation kommt aus complexity.DuplBlocks (dupl-Linter), wird hier nur
// gespiegelt, damit der Report-Renderer eine konsistente Struktur sieht.
type DuplicationReport struct {
	GoBlocks           int
	FrontendPercent    float64
	FrontendClones     int
	FrontendDupLines   int
	FrontendTotalLines int
	FrontendCollected  bool
}

type jscpdSummary struct {
	Statistics struct {
		Total struct {
			Clones          int     `json:"clones"`
			DuplicatedLines int     `json:"duplicatedLines"`
			Lines           int     `json:"lines"`
			Percentage      float64 `json:"percentage"`
		} `json:"total"`
	} `json:"statistics"`
}

func collectDuplication(repoRoot string, goBlocks int) DuplicationReport {
	rep := DuplicationReport{GoBlocks: goBlocks}
	outDir, err := os.MkdirTemp("", "metrics-jscpd-*")
	if err != nil {
		return rep
	}
	defer os.RemoveAll(outDir)

	cmd := exec.Command("pnpm", "-C", "web", "exec", "jscpd", "src",
		"--reporters", "json", "--output", outDir, "--silent")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return rep
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "jscpd-report.json"))
	if err != nil {
		return rep
	}
	var s jscpdSummary
	if err := json.Unmarshal(raw, &s); err != nil {
		return rep
	}
	rep.FrontendClones = s.Statistics.Total.Clones
	rep.FrontendDupLines = s.Statistics.Total.DuplicatedLines
	rep.FrontendTotalLines = s.Statistics.Total.Lines
	rep.FrontendPercent = s.Statistics.Total.Percentage
	rep.FrontendCollected = true
	return rep
}
