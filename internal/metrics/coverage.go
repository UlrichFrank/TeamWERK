package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CoverageReport listet Go- und Frontend-Coverage separat.
type CoverageReport struct {
	GoPercent       float64
	FrontendPercent float64
	FrontendOK      bool // false wenn vitest --coverage nicht lief
}

func collectCoverage(repoRoot string) (CoverageReport, error) {
	rep := CoverageReport{}
	goPct, err := runGoCoverage(repoRoot)
	if err != nil {
		return rep, err
	}
	rep.GoPercent = goPct

	fePct, ok := runFrontendCoverage(repoRoot)
	rep.FrontendPercent = fePct
	rep.FrontendOK = ok
	return rep, nil
}

func runGoCoverage(repoRoot string) (float64, error) {
	profile, err := os.CreateTemp("", "metrics-cover-*.out")
	if err != nil {
		return 0, err
	}
	profile.Close()
	defer os.Remove(profile.Name())

	cmd := exec.Command("go", "test", "-coverprofile="+profile.Name(), "./internal/...")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("go test -cover: %w", err)
	}
	cmd = exec.Command("go", "tool", "cover", "-func="+profile.Name())
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("go tool cover: %w", err)
	}
	return parseGoCoverageTotal(out), nil
}

func parseGoCoverageTotal(out []byte) float64 {
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pct := strings.TrimSuffix(fields[len(fields)-1], "%")
		if v, err := strconv.ParseFloat(pct, 64); err == nil {
			return v
		}
	}
	return 0
}

type vitestSummary struct {
	Total struct {
		Lines struct {
			Pct float64 `json:"pct"`
		} `json:"lines"`
	} `json:"total"`
}

func runFrontendCoverage(repoRoot string) (float64, bool) {
	cmd := exec.Command("pnpm", "-C", "web", "exec", "vitest", "run",
		"--coverage",
		"--coverage.reporter=json-summary",
		"--coverage.reporter=text-summary")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return 0, false
	}
	summaryPath := filepath.Join(repoRoot, "web", "coverage", "coverage-summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		return 0, false
	}
	var s vitestSummary
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, false
	}
	return s.Total.Lines.Pct, true
}
