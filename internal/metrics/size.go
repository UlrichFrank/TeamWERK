package metrics

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// SizeReport entspricht den Größen-Kennzahlen aus scc.
type SizeReport struct {
	Languages    []LanguageSize
	GoCode       int // Code-LOC (ohne Tests; siehe Tests-Feld)
	GoTests      int
	FrontendCode int // TS + TSX + CSS
	TotalCode    int
	CommentRatio float64 // (comments / (code+comments)) gesamt
}

// LanguageSize ist eine flache Zeile aus scc-JSON.
type LanguageSize struct {
	Name    string
	Files   int
	Code    int
	Comment int
	Blank   int
	Lines   int
}

type sccLang struct {
	Name    string `json:"Name"`
	Count   int    `json:"Count"`
	Code    int    `json:"Code"`
	Comment int    `json:"Comment"`
	Blank   int    `json:"Blank"`
	Lines   int    `json:"Lines"`
	Files   []struct {
		Location string `json:"Location"`
		Code     int    `json:"Code"`
	} `json:"Files"`
}

func collectSize(repoRoot string) (SizeReport, error) {
	cmd := exec.Command("go", "tool", "scc", "--format", "json", "--no-cocomo", "--by-file",
		"--exclude-dir", "node_modules,dist,bin,storage,backup,coverage,metrics,tmp",
		"./internal", "./cmd", "./web/src")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return SizeReport{}, fmt.Errorf("scc: %w", err)
	}
	return parseSize(out)
}

func parseSize(jsonOut []byte) (SizeReport, error) {
	var langs []sccLang
	if err := json.Unmarshal(jsonOut, &langs); err != nil {
		return SizeReport{}, fmt.Errorf("parse scc json: %w", err)
	}
	rep := SizeReport{Languages: make([]LanguageSize, 0, len(langs))}
	var totalCode, totalComment int
	for _, l := range langs {
		rep.Languages = append(rep.Languages, LanguageSize{
			Name: l.Name, Files: l.Count, Code: l.Code,
			Comment: l.Comment, Blank: l.Blank, Lines: l.Lines,
		})
		totalCode += l.Code
		totalComment += l.Comment
		if l.Name == "Go" {
			for _, f := range l.Files {
				if isGoTestFile(f.Location) {
					rep.GoTests += f.Code
				} else {
					rep.GoCode += f.Code
				}
			}
		}
		if l.Name == "TypeScript" || l.Name == "TSX" || l.Name == "CSS" {
			rep.FrontendCode += l.Code
		}
	}
	rep.TotalCode = totalCode
	if totalCode+totalComment > 0 {
		rep.CommentRatio = float64(totalComment) / float64(totalCode+totalComment)
	}
	return rep, nil
}

func isGoTestFile(loc string) bool {
	return len(loc) > 8 && loc[len(loc)-8:] == "_test.go"
}
