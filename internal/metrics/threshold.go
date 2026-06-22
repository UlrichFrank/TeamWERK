package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Thresholds enthält obere Grenzwerte (max) bzw. untere Grenzwerte (min) je
// Kennzahl. Werte mit -1 sind „nicht gesetzt" und werden nicht geprüft.
type Thresholds struct {
	GoCycloMax        int
	GoCognitMax       int
	FunLenMax         int
	DuplGoMax         int
	DuplFrontendMaxPc float64 // %
	GoCoverageMinPct  float64
	FECoverageMinPct  float64
	LintIssuesMax     int
}

// Violation = eine konkrete Schwellwert-Verletzung im Gate-Lauf.
type Violation struct {
	Key    string
	Actual any
	Limit  any
	Reason string
}

func newEmptyThresholds() Thresholds {
	return Thresholds{
		GoCycloMax: -1, GoCognitMax: -1, FunLenMax: -1, DuplGoMax: -1,
		DuplFrontendMaxPc: -1, GoCoverageMinPct: -1, FECoverageMinPct: -1, LintIssuesMax: -1,
	}
}

// loadThresholds liest die simple key:value-YAML-Datei. Kommentare (#) und
// Leerzeilen werden ignoriert. Format: `key: <number>` pro Zeile.
func loadThresholds(path string) (Thresholds, error) {
	t := newEmptyThresholds()
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, valStr, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		valStr = strings.TrimSpace(valStr)
		if i := strings.Index(valStr, "#"); i >= 0 {
			valStr = strings.TrimSpace(valStr[:i])
		}
		if err := applyThresholdValue(&t, key, valStr); err != nil {
			return t, fmt.Errorf("thresholds.yml: %s: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return t, err
	}
	return t, nil
}

func applyThresholdValue(t *Thresholds, key, valStr string) error {
	switch key {
	case "gocyclo_max":
		return parseIntInto(valStr, &t.GoCycloMax)
	case "gocognit_max":
		return parseIntInto(valStr, &t.GoCognitMax)
	case "funlen_max":
		return parseIntInto(valStr, &t.FunLenMax)
	case "dupl_go_max":
		return parseIntInto(valStr, &t.DuplGoMax)
	case "dupl_frontend_max_pct":
		return parseFloatInto(valStr, &t.DuplFrontendMaxPc)
	case "go_coverage_min_pct":
		return parseFloatInto(valStr, &t.GoCoverageMinPct)
	case "frontend_coverage_min_pct":
		return parseFloatInto(valStr, &t.FECoverageMinPct)
	case "lint_issues_max":
		return parseIntInto(valStr, &t.LintIssuesMax)
	}
	// Unbekannte Keys werden ignoriert (vorwärtskompatibel).
	return nil
}

func parseIntInto(s string, dst *int) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*dst = v
	return nil
}

func parseFloatInto(s string, dst *float64) error {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*dst = v
	return nil
}

// compareThresholds liefert die Liste der Verletzungen für den Gate-Lauf.
func compareThresholds(rep Report, t Thresholds) []Violation {
	var v []Violation
	v = appendIntMax(v, "gocyclo", rep.Complexity.GoCyclo, t.GoCycloMax, "höchstens N Funktionen über Schwellwert")
	v = appendIntMax(v, "gocognit", rep.Complexity.GoCognit, t.GoCognitMax, "höchstens N Funktionen über Schwellwert")
	v = appendIntMax(v, "funlen", rep.Complexity.FunLen, t.FunLenMax, "höchstens N zu lange Funktionen")
	v = appendIntMax(v, "dupl_go", rep.Complexity.DuplBlocks, t.DuplGoMax, "höchstens N duplizierte Go-Blöcke")
	v = appendFloatMax(v, "dupl_frontend_pct", rep.Duplic.FrontendPercent, t.DuplFrontendMaxPc, "höchstens N % duplizierte Frontend-Zeilen")
	v = appendFloatMin(v, "go_coverage_pct", rep.Coverage.GoPercent, t.GoCoverageMinPct, "Go-Coverage muss mindestens N % betragen")
	if rep.Coverage.FrontendOK {
		v = appendFloatMin(v, "frontend_coverage_pct", rep.Coverage.FrontendPercent, t.FECoverageMinPct, "Frontend-Coverage muss mindestens N % betragen")
	}
	v = appendIntMax(v, "lint_issues", rep.LintDens.Issues, t.LintIssuesMax, "höchstens N Issues aus Haupt-.golangci.yml")
	return v
}

func appendIntMax(vs []Violation, key string, actual, limit int, reason string) []Violation {
	if limit < 0 {
		return vs
	}
	if actual > limit {
		return append(vs, Violation{Key: key, Actual: actual, Limit: limit, Reason: reason})
	}
	return vs
}

func appendFloatMax(vs []Violation, key string, actual, limit float64, reason string) []Violation {
	if limit < 0 {
		return vs
	}
	if actual > limit {
		return append(vs, Violation{Key: key, Actual: actual, Limit: limit, Reason: reason})
	}
	return vs
}

func appendFloatMin(vs []Violation, key string, actual, limit float64, reason string) []Violation {
	if limit < 0 {
		return vs
	}
	if actual < limit {
		return append(vs, Violation{Key: key, Actual: actual, Limit: limit, Reason: reason})
	}
	return vs
}
