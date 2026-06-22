package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ComplexityReport hält die Go-Komplexitäts-Kennzahlen aus der
// .golangci.metrics.yml-Ausführung.
type ComplexityReport struct {
	GoCyclo    int // Anzahl gocyclo-Issues
	GoCognit   int // Anzahl gocognit-Issues
	FunLen     int // Anzahl funlen-Verstöße
	DuplBlocks int // dupl-Treffer (Go-Code-Duplikation)
	Hotspots   []Hotspot
}

// Hotspot ist eine komplexe Stelle aus dem Komplexitäts-Lauf.
type Hotspot struct {
	Linter   string
	File     string
	Line     int
	Function string
	Text     string
}

type lintIssue struct {
	FromLinter string `json:"FromLinter"`
	Text       string `json:"Text"`
	Pos        struct {
		Filename string `json:"Filename"`
		Line     int    `json:"Line"`
	} `json:"Pos"`
}

type lintReport struct {
	Issues []lintIssue `json:"Issues"`
}

func collectComplexity(repoRoot string) (ComplexityReport, error) {
	tmp, err := os.CreateTemp("", "metrics-complexity-*.json")
	if err != nil {
		return ComplexityReport{}, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	cmd := exec.Command("golangci-lint", "run",
		"-c", ".golangci.metrics.yml",
		"--issues-exit-code", "0",
		"--show-stats=false",
		"--output.text.path=/dev/null",
		"--output.json.path="+tmp.Name(),
		"./...")
	cmd.Dir = repoRoot
	// Stderr von golangci-lint ist informativ (Linter-Banner), ignoriert.
	if err := cmd.Run(); err != nil {
		return ComplexityReport{}, fmt.Errorf("golangci-lint metrics run: %w", err)
	}
	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return ComplexityReport{}, fmt.Errorf("read metrics json: %w", err)
	}
	return parseComplexity(raw)
}

func parseComplexity(raw []byte) (ComplexityReport, error) {
	var rep lintReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return ComplexityReport{}, fmt.Errorf("parse golangci json: %w", err)
	}
	out := ComplexityReport{}
	for _, iss := range rep.Issues {
		switch iss.FromLinter {
		case "gocyclo":
			out.GoCyclo++
		case "gocognit":
			out.GoCognit++
		case "funlen":
			out.FunLen++
		case "dupl":
			out.DuplBlocks++
		}
		if iss.FromLinter == "gocyclo" || iss.FromLinter == "gocognit" || iss.FromLinter == "funlen" {
			out.Hotspots = append(out.Hotspots, Hotspot{
				Linter:   iss.FromLinter,
				File:     normalizeRepoPath(iss.Pos.Filename),
				Line:     iss.Pos.Line,
				Function: extractFuncName(iss.Text),
				Text:     iss.Text,
			})
		}
	}
	sort.SliceStable(out.Hotspots, func(i, j int) bool {
		return hotspotScore(out.Hotspots[i].Text) > hotspotScore(out.Hotspots[j].Text)
	})
	return out, nil
}

// extractFuncName zieht den Funktionsnamen aus typischen Linter-Texten:
//   - funlen:   `Function 'NAME' is too long (...)`
//   - gocognit: `cognitive complexity N of func ` + "`" + `NAME` + "`" + ` is ...`
//   - gocyclo:  `cyclomatic complexity N of func ` + "`" + `NAME` + "`" + ` is ...`
func extractFuncName(text string) string {
	if name := between(text, "Function '", "'"); name != "" {
		return name
	}
	if name := between(text, "of func `", "`"); name != "" {
		return name
	}
	return ""
}

// normalizeRepoPath schneidet ggf. führende `../` und absolute Präfixe ab und
// liefert einen Pfad relativ zur Repo-Wurzel (erstes bekanntes Top-Level-Dir).
func normalizeRepoPath(p string) string {
	p = filepath.ToSlash(p)
	for _, top := range []string{"internal/", "cmd/", "web/", "deploy/", "openspec/", "scripts/"} {
		if i := strings.Index(p, top); i >= 0 {
			return p[i:]
		}
	}
	return p
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	rest := s[i+len(start):]
	j := strings.Index(rest, end)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// hotspotScore parsiert die erste Zahl aus dem Issue-Text als Sortierkriterium.
func hotspotScore(text string) int {
	score := 0
	inNum := false
	for _, r := range text {
		if r >= '0' && r <= '9' {
			score = score*10 + int(r-'0')
			inNum = true
		} else if inNum {
			break
		}
	}
	return score
}
