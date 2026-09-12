// Goroutine-Invariante als Test (Harness-Engineering, Säule 2 — mechanisch
// erzwungene Konventionen, analog broadcast_test.go/pushfanout_test.go).
// Betriebshärtung Welle 2 (design.md Entscheidung 2): ein nackter `go`-Start
// im Domänencode kann panicken und — unabgefangen — den gesamten
// Serverprozess mitreißen. Jede Goroutine MUSS deshalb über
// internal/background.Go (oder die dünne internal/notify.SendAsync-Fassade
// darüber) laufen, die den Panic abfängt, loggt und zählt.
//
// Dieser Test parst alle Nicht-Test-Dateien unter internal/ (ohne
// internal/background und internal/hub — siehe unten) und sammelt jeden
// *ast.GoStmt. Erlaubt sind nur go-Starts, deren aufgerufene Funktion
// background.Go oder notify.SendAsync ist. Alles andere ist ein Fehler, außer
// die Fundstelle steht mit Begründung in goroutineAllowlist — ein
// Allowlist-Eintrag ohne reale Fundstelle lässt den Test ebenfalls
// fehlschlagen (Anti-Verrottung, wie TestBroadcastAllowlist_NoOrphans).
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goroutineExemptDirs sind Top-Level-Packages unter internal/, die dieser Test
// NICHT scannt:
//   - background selbst (der Wrapper startet die geschützte Goroutine intern
//     nackt — das ist der Mechanismus, nicht ein Umgehungsfall).
//   - hub: reiner Kanal-Fan-out (Subscribe/Broadcast) ohne eigene go-Starts im
//     Produktionscode (Stand dieses Change — Zukunftssicherung analog design.md
//     Entscheidung 2, falls das Package künftig eigene Goroutinen bekommt).
var goroutineExemptDirs = map[string]bool{
	"background": true,
	"hub":        true,
}

// goroutineAllowFuncs: Paket -> erlaubte Funktion(en), die einen go-Start
// panic-sicher machen.
var goroutineAllowFuncs = map[string]map[string]bool{
	"background": {"Go": true},
	"notify":     {"SendAsync": true},
}

// goroutineAllowlist: "<Paket>.<einschließende Funktion>" -> Begründung, für
// bewusste Ausnahmen außerhalb von background.Go/notify.SendAsync. Aktuell
// leer — die einzigen im Betrieb bewusst nackten Goroutinen (Hub-Fan-out, der
// Video-Transcode-Worker-Start in cmd/teamwerk/main.go) liegen außerhalb des
// gescannten Baums (main.go ist Composition Root, kein internal/-Package;
// hub ist oben von der Prüfung ausgenommen).
var goroutineAllowlist = map[string]string{}

// goroutineOccurrence: ein gefundener *ast.GoStmt außerhalb von
// background.Go/notify.SendAsync.
type goroutineOccurrence struct {
	pkg  string // internal/-Package, z. B. "chat"
	fn   string // einschließende Funktion, z. B. "SendBroadcast"
	file string // relativer Pfad, für Fehlermeldungen
	line int
}

func (o goroutineOccurrence) key() string { return o.pkg + "." + o.fn }

// isAllowedGoCall meldet, ob call (das Argument eines go-Statements) direkt
// background.Go(...) oder notify.SendAsync(...) aufruft — über die
// Importalias-Auflösung importAliasToPkg (Alias -> internal/-Package).
func isAllowedGoCall(call *ast.CallExpr, importAliasToPkg map[string]string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkg, ok := importAliasToPkg[ident.Name]
	if !ok {
		return false
	}
	return goroutineAllowFuncs[pkg][sel.Sel.Name]
}

// fileImportAliasToPkg liefert Alias (bzw. Paketname) -> internal/-Package für
// alle internal/-Imports einer Datei.
func fileImportAliasToPkg(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(p, modulePrefix) {
			continue
		}
		pkg := strings.TrimPrefix(p, modulePrefix)
		alias := pkg
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		out[alias] = pkg
	}
	return out
}

// enclosingFuncName findet den Namen der Top-Level-Funktion/Methode, die pos
// umschließt (für die Allowlist-/Fehlermeldung). Liefert "" wenn pos in
// keinem *ast.FuncDecl liegt (z. B. eine package-level var-Initialisierung).
func enclosingFuncName(f *ast.File, pos token.Pos) string {
	name := ""
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		if fd.Body.Pos() <= pos && pos <= fd.Body.End() {
			name = fd.Name.Name
		}
	}
	return name
}

// collectGoroutineOccurrences durchläuft alle internal/-Packages (Produktionscode,
// _test.go und goroutineExemptDirs ausgeschlossen) und sammelt jeden *ast.GoStmt,
// dessen Aufruf NICHT background.Go/notify.SendAsync ist.
func collectGoroutineOccurrences(t *testing.T) []goroutineOccurrence {
	t.Helper()
	root := internalRoot(t)
	var occs []goroutineOccurrence

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg := strings.SplitN(rel, "/", 2)[0]
		if goroutineExemptDirs[pkg] {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		aliasToPkg := fileImportAliasToPkg(f)

		ast.Inspect(f, func(n ast.Node) bool {
			gs, ok := n.(*ast.GoStmt)
			if !ok {
				return true
			}
			if isAllowedGoCall(gs.Call, aliasToPkg) {
				return true
			}
			occs = append(occs, goroutineOccurrence{
				pkg:  pkg,
				fn:   enclosingFuncName(f, gs.Pos()),
				file: rel,
				line: fset.Position(gs.Pos()).Line,
			})
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/: %v", err)
	}
	return occs
}

func TestArchitecture_KeineNacktenGoroutinen(t *testing.T) {
	occs := collectGoroutineOccurrences(t)
	for _, o := range occs {
		if _, ok := goroutineAllowlist[o.key()]; ok {
			continue
		}
		t.Errorf("%s:%d: internal/%s startet in %s eine nackte Goroutine — über background.Go(\"<name>\", func(){...}) "+
			"bzw. notify.SendAsync(...) panic-sicher machen, oder mit Begründung in goroutineAllowlist aufnehmen",
			o.file, o.line, o.pkg, o.fn)
	}
}

// TestArchitecture_GoroutineAllowlistOhneWaisen stellt sicher, dass jeder
// goroutineAllowlist-Eintrag auf eine real gefundene nackte Goroutine zeigt
// (verhindert Verrottung, analog TestBroadcastAllowlist_NoOrphans /
// TestArchitecture_PushAllowlistOhneWaisen).
func TestArchitecture_GoroutineAllowlistOhneWaisen(t *testing.T) {
	occs := collectGoroutineOccurrences(t)
	found := map[string]bool{}
	for _, o := range occs {
		found[o.key()] = true
	}
	for key := range goroutineAllowlist {
		if !found[key] {
			t.Errorf("goroutineAllowlist-Eintrag %q zeigt auf keine gefundene nackte Goroutine (veraltet? Tippfehler?)", key)
		}
	}
}
