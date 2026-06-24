// Package metrics implementiert das `teamwerk metrics` Tool (siehe
// openspec/changes/code-metrics-target). Erhebt Größe, Komplexität, Coverage,
// Duplikation und Lint-Dichte für Go-Backend und TS/React-Frontend, rendert
// stdout-Tabelle + metrics/REPORT.md und prüft optional gegen Schwellwerte.
package metrics

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Options steuert den Lauf. Gate=true vergleicht gegen thresholds.yml.
type Options struct {
	RepoRoot string
	Gate     bool
	Out      io.Writer
}

// Report ist die strukturierte Ergebnismenge eines Laufs.
type Report struct {
	GitHash    string
	Timestamp  time.Time
	Size       SizeReport
	Complexity ComplexityReport
	Coverage   CoverageReport
	LintDens   LintDensityReport
	Duplic     DuplicationReport
}

// Run erhebt alle Kennzahlen, rendert Report und (falls Gate) prüft Schwellwerte.
// Default-Lauf liefert immer Exit-Code 0; Gate-Lauf liefert 1 bei Verletzung.
func Run(opts Options) int {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.RepoRoot == "" {
		root, err := repoRoot()
		if err != nil {
			fmt.Fprintf(opts.Out, "metrics: cannot determine repo root: %v\n", err)
			return 0
		}
		opts.RepoRoot = root
	}

	if err := checkTools(opts.RepoRoot); err != nil {
		fmt.Fprintf(opts.Out, "metrics: %v\n", err)
		return 0
	}

	rep, err := collect(opts.RepoRoot)
	if err != nil {
		fmt.Fprintf(opts.Out, "metrics: collection failed: %v\n", err)
		return 0
	}

	renderStdout(opts.Out, rep)
	reportPath := filepath.Join(opts.RepoRoot, "metrics", "REPORT.md")
	if err := writeMarkdownReport(reportPath, rep); err != nil {
		fmt.Fprintf(opts.Out, "metrics: writing REPORT.md failed: %v\n", err)
		return 0
	}
	fmt.Fprintf(opts.Out, "\nReport geschrieben: %s\n", reportPath)

	if !opts.Gate {
		return 0
	}

	thresholdsPath := filepath.Join(opts.RepoRoot, "metrics", "thresholds.yml")
	thresholds, err := loadThresholds(thresholdsPath)
	if err != nil {
		fmt.Fprintf(opts.Out, "metrics-gate: cannot read %s: %v\n", thresholdsPath, err)
		return 1
	}
	violations := compareThresholds(rep, thresholds)
	if len(violations) == 0 {
		fmt.Fprintln(opts.Out, "metrics-gate: alle Schwellwerte eingehalten.")
		return 0
	}
	fmt.Fprintln(opts.Out, "metrics-gate: Schwellwert-Verletzungen:")
	for _, v := range violations {
		fmt.Fprintf(opts.Out, "  - %s: Ist=%v, Schwellwert=%v (%s)\n", v.Key, v.Actual, v.Limit, v.Reason)
	}
	return 1
}

func collect(repoRoot string) (Report, error) {
	rep := Report{
		Timestamp: time.Now().UTC(),
		GitHash:   gitShortHash(repoRoot),
	}
	size, err := collectSize(repoRoot)
	if err != nil {
		return rep, fmt.Errorf("size: %w", err)
	}
	rep.Size = size

	complexity, err := collectComplexity(repoRoot)
	if err != nil {
		return rep, fmt.Errorf("complexity: %w", err)
	}
	rep.Complexity = complexity

	coverage, err := collectCoverage(repoRoot)
	if err != nil {
		return rep, fmt.Errorf("coverage: %w", err)
	}
	rep.Coverage = coverage

	rep.LintDens = collectLintDensity(repoRoot, size.GoCode)
	rep.Duplic = collectDuplication(repoRoot, complexity.DuplBlocks)
	return rep, nil
}

// checkTools verifiziert dass scc, golangci-lint, jscpd und go vorhanden sind
// und liefert sonst einen Hinweis mit Installbefehl.
func checkTools(repoRoot string) error {
	checks := []struct {
		name    string
		args    []string
		install string
		dir     string
	}{
		{"go", []string{"tool", "scc", "--version"}, "go get -tool github.com/boyter/scc/v3@latest", repoRoot},
		{"golangci-lint", []string{"--version"}, "siehe https://golangci-lint.run/welcome/install/", repoRoot},
		{"pnpm", []string{"-C", "web", "exec", "jscpd", "--version"}, "pnpm -C web add -D jscpd", repoRoot},
	}
	for _, c := range checks {
		cmd := exec.Command(c.name, c.args...)
		cmd.Dir = c.dir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("benötigtes Werkzeug %q nicht verfügbar (%w) — installieren mit: %s", c.name+" "+c.args[0], err, c.install)
		}
	}
	return nil
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return string(trimNewline(out)), nil
}

func gitShortHash(repoRoot string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(trimNewline(out))
}

func trimNewline(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// ErrToolMissing wird zurückgegeben, wenn ein benötigtes Werkzeug fehlt.
var ErrToolMissing = errors.New("required metrics tool missing")
