package metrics

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxHotspots = 10

func renderStdout(w io.Writer, rep Report) {
	fmt.Fprintln(w, "TeamWERK Code Metrics")
	fmt.Fprintf(w, "  Stand: %s · Git: %s\n", rep.Timestamp.Format("2006-01-02 15:04:05Z"), rep.GitHash)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Größe")
	fmt.Fprintf(w, "    Go (prod):      %6d LOC\n", rep.Size.GoCode)
	fmt.Fprintf(w, "    Go (tests):     %6d LOC\n", rep.Size.GoTests)
	if rep.Size.GoCode > 0 {
		fmt.Fprintf(w, "    Go test ratio:  %6.2f\n", float64(rep.Size.GoTests)/float64(rep.Size.GoCode))
	}
	fmt.Fprintf(w, "    Frontend code:  %6d LOC (TS/TSX/CSS)\n", rep.Size.FrontendCode)
	fmt.Fprintf(w, "    Kommentar:      %6.1f %% (gesamt)\n", rep.Size.CommentRatio*100)

	fmt.Fprintln(w, "  Komplexität (gocyclo>15 / gocognit>20 / funlen>100L)")
	fmt.Fprintf(w, "    gocyclo:        %6d Funktionen\n", rep.Complexity.GoCyclo)
	fmt.Fprintf(w, "    gocognit:       %6d Funktionen\n", rep.Complexity.GoCognit)
	fmt.Fprintf(w, "    funlen:         %6d Funktionen\n", rep.Complexity.FunLen)

	fmt.Fprintln(w, "  Coverage")
	fmt.Fprintf(w, "    Go:             %6.1f %%\n", rep.Coverage.GoPercent)
	if rep.Coverage.FrontendOK {
		fmt.Fprintf(w, "    Frontend:       %6.1f %%\n", rep.Coverage.FrontendPercent)
	} else {
		fmt.Fprintln(w, "    Frontend:       n/a   (vitest --coverage übersprungen oder fehlgeschlagen)")
	}

	fmt.Fprintln(w, "  Lint-Dichte (.golangci.yml)")
	if rep.LintDens.Collected {
		fmt.Fprintf(w, "    Issues:         %6d  (%5.2f / kLOC)\n", rep.LintDens.Issues, rep.LintDens.IssuesPerKLOC)
	} else {
		fmt.Fprintln(w, "    Issues:         n/a")
	}

	fmt.Fprintln(w, "  Duplikation")
	fmt.Fprintf(w, "    Go (dupl):      %6d Blöcke\n", rep.Duplic.GoBlocks)
	if rep.Duplic.FrontendCollected {
		fmt.Fprintf(w, "    Frontend (jscpd): %4d Clones · %5.2f %% Lines\n", rep.Duplic.FrontendClones, rep.Duplic.FrontendPercent)
	} else {
		fmt.Fprintln(w, "    Frontend (jscpd): n/a")
	}
}

func writeMarkdownReport(path string, rep Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# TeamWERK Code Metrics\n\n")
	fmt.Fprintf(&b, "Stand: `%s` · Git: `%s`\n\n", rep.Timestamp.Format("2006-01-02 15:04:05Z"), rep.GitHash)
	fmt.Fprintf(&b, "Generiert von `make metrics`. Reines Reporting — keine Build-Aussage.\n\n")

	writeSizeSection(&b, rep.Size)
	writeComplexitySection(&b, rep.Complexity)
	writeCoverageSection(&b, rep.Coverage)
	writeLintSection(&b, rep.LintDens)
	writeDuplicationSection(&b, rep.Duplic)
	writeHotspotsSection(&b, rep.Complexity.Hotspots)

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSizeSection(b *strings.Builder, s SizeReport) {
	fmt.Fprintln(b, "## Größe")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Bereich | LOC | Dateien |")
	fmt.Fprintln(b, "|---|---:|---:|")
	fmt.Fprintf(b, "| Go (prod) | %d | – |\n", s.GoCode)
	fmt.Fprintf(b, "| Go (tests) | %d | – |\n", s.GoTests)
	fmt.Fprintf(b, "| Frontend (TS/TSX/CSS) | %d | – |\n", s.FrontendCode)
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "Per-Sprache (scc):")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Sprache | Dateien | Code | Kommentar | Blank |")
	fmt.Fprintln(b, "|---|---:|---:|---:|---:|")
	langs := append([]LanguageSize(nil), s.Languages...)
	sort.SliceStable(langs, func(i, j int) bool { return langs[i].Code > langs[j].Code })
	for _, l := range langs {
		fmt.Fprintf(b, "| %s | %d | %d | %d | %d |\n", l.Name, l.Files, l.Code, l.Comment, l.Blank)
	}
	fmt.Fprintln(b, "")
	if s.GoCode > 0 {
		fmt.Fprintf(b, "Go Test-Ratio: %.2f (Test-LOC ÷ Prod-LOC). Kommentar-Anteil gesamt: %.1f %%.\n\n",
			float64(s.GoTests)/float64(s.GoCode), s.CommentRatio*100)
	}
}

func writeComplexitySection(b *strings.Builder, c ComplexityReport) {
	fmt.Fprintln(b, "## Komplexität")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "Aus `.golangci.metrics.yml` (gocyclo>15, gocognit>20, funlen>100L, dupl>150T).")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Linter | Verstöße |")
	fmt.Fprintln(b, "|---|---:|")
	fmt.Fprintf(b, "| gocyclo | %d |\n", c.GoCyclo)
	fmt.Fprintf(b, "| gocognit | %d |\n", c.GoCognit)
	fmt.Fprintf(b, "| funlen | %d |\n", c.FunLen)
	fmt.Fprintf(b, "| dupl | %d |\n", c.DuplBlocks)
	fmt.Fprintln(b, "")
}

func writeCoverageSection(b *strings.Builder, c CoverageReport) {
	fmt.Fprintln(b, "## Coverage")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Stack | Wert |")
	fmt.Fprintln(b, "|---|---:|")
	fmt.Fprintf(b, "| Go (`go test -cover ./internal/...`) | %.1f %% |\n", c.GoPercent)
	if c.FrontendOK {
		fmt.Fprintf(b, "| Frontend (`vitest --coverage`) | %.1f %% |\n", c.FrontendPercent)
	} else {
		fmt.Fprintln(b, "| Frontend (`vitest --coverage`) | n/a |")
	}
	fmt.Fprintln(b, "")
}

func writeLintSection(b *strings.Builder, l LintDensityReport) {
	fmt.Fprintln(b, "## Lint-Dichte")
	fmt.Fprintln(b, "")
	if !l.Collected {
		fmt.Fprintln(b, "n/a — Haupt-`.golangci.yml`-Lauf nicht erfolgreich.")
		fmt.Fprintln(b, "")
		return
	}
	fmt.Fprintln(b, "| Kennzahl | Wert |")
	fmt.Fprintln(b, "|---|---:|")
	fmt.Fprintf(b, "| Issues (Haupt-`.golangci.yml`) | %d |\n", l.Issues)
	fmt.Fprintf(b, "| Go-Code | %.1f kLOC |\n", l.GoCodeKLOC)
	fmt.Fprintf(b, "| Issues / kLOC | %.2f |\n", l.IssuesPerKLOC)
	fmt.Fprintln(b, "")
}

func writeDuplicationSection(b *strings.Builder, d DuplicationReport) {
	fmt.Fprintln(b, "## Duplikation")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Stack | Wert |")
	fmt.Fprintln(b, "|---|---:|")
	fmt.Fprintf(b, "| Go (`dupl` aus Komplexitäts-Config) | %d Blöcke |\n", d.GoBlocks)
	if d.FrontendCollected {
		fmt.Fprintf(b, "| Frontend (`jscpd`) | %d Clones · %d/%d duplicated lines · %.2f %% |\n",
			d.FrontendClones, d.FrontendDupLines, d.FrontendTotalLines, d.FrontendPercent)
	} else {
		fmt.Fprintln(b, "| Frontend (`jscpd`) | n/a |")
	}
	fmt.Fprintln(b, "")
}

func writeHotspotsSection(b *strings.Builder, hs []Hotspot) {
	fmt.Fprintln(b, "## Top-Hotspots (Go)")
	fmt.Fprintln(b, "")
	if len(hs) == 0 {
		fmt.Fprintln(b, "Keine Funktionen oberhalb der Schwellwerte.")
		fmt.Fprintln(b, "")
		return
	}
	fmt.Fprintln(b, "Die komplexesten Funktionen aus dem Komplexitäts-Lauf.")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| Linter | Funktion | Datei:Zeile | Detail |")
	fmt.Fprintln(b, "|---|---|---|---|")
	limit := maxHotspots
	if limit > len(hs) {
		limit = len(hs)
	}
	for _, h := range hs[:limit] {
		fn := h.Function
		if fn == "" {
			fn = "(unbenannt)"
		}
		fmt.Fprintf(b, "| %s | `%s` | `%s:%d` | %s |\n", h.Linter, fn, h.File, h.Line, escapeMD(h.Text))
	}
	fmt.Fprintln(b, "")
}

func escapeMD(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
