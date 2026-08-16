// Push-Fan-out-Invariante als Test (Harness-Engineering, Säule 2 — mechanisch
// erzwungene Konventionen). event-log-Change: notify.Send ist die einzige
// Stelle, die den Event-Log aus der UNGEFILTERTEN Empfängerliste schreibt,
// bevor Push/Email-Präferenzen gefiltert werden (design.md Decision 1). Ein
// Aufrufer, der stattdessen push.SendToUsers (oder push.SendToUserWithBadge)
// direkt nutzt, umgeht den Log — genau das Muster, das die sechs Bypässe vor
// diesem Change waren. Dieser Test parst alle internal/-Packages (nur stdlib
// go/parser, go/ast — konsistent mit arch_test.go/broadcast_test.go) und
// meldet jede Verwendung von push.SendToUsers/push.SendToUserWithBadge
// außerhalb von internal/notify und internal/push selbst.
//
// Erfasst wird jede Verwendung des Bezeichners (SelectorExpr), nicht nur ein
// direkter Aufruf: internal/chat übergibt push.SendToUserWithBadge als
// Funktionswert (pushFn: push.SendToUserWithBadge) und ruft ihn erst über
// h.pushFn(...) auf — ein reiner CallExpr-Scan auf "push.SendToUserWithBadge(...)"
// würde das übersehen.
//
// Ausnahmen ausschließlich über die begründete Allowlist unten. Ein
// Allowlist-Eintrag ohne realen Fundort lässt den Test fehlschlagen
// (Anti-Verrottung, wie TestBroadcastAllowlist_NoOrphans).
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

// pushFanoutFuncs sind die push-Paket-Funktionen, die NUR aus internal/notify
// (bzw. internal/push selbst) aufgerufen werden dürfen.
var pushFanoutFuncs = map[string]bool{
	"SendToUsers":         true,
	"SendToUserWithBadge": true,
}

// pushFanoutAllowlist: Paket -> Funktion, die BEWUSST direkt auf den push-Fan-out
// zugreift statt über notify.Send. Schlüssel = "<Paket>.<Funktion>", Wert =
// Begründung. Jeder Eintrag MUSS auf einen real gefundenen Aufrufer zeigen
// (sonst schlägt TestArchitecture_PushAllowlistOhneWaisen fehl).
var pushFanoutAllowlist = map[string]string{
	// Chat hat einen eigenen Realtime-Kanal (/api/chat/events) und einen eigenen
	// App-Badge (siehe docs/agent/06-gotchas.md „App-Icon-Badge") — bewusst nicht
	// im Event-Log (design.md „Chat bleibt draußen").
	"chat.SendToUserWithBadge": "eigener Kanal + eigener Badge, bewusst nicht im Event-Log",
}

// pushFanoutOccurrence: ein gefundener Zugriff auf eine der pushFanoutFuncs.
type pushFanoutOccurrence struct {
	pkg  string // internal/-Package, z. B. "chat"
	fn   string // z. B. "SendToUserWithBadge"
	file string // relativer Pfad, für Fehlermeldungen
	line int
}

func (o pushFanoutOccurrence) key() string { return o.pkg + "." + o.fn }

// collectPushFanoutOccurrences durchläuft alle internal/-Packages (Produktionscode,
// _test.go ausgeschlossen — Test-Doubles wie `push.SendToUsers = func(...)` sind
// kein Fan-out-Bypass) außer internal/notify und internal/push selbst, und
// sammelt jede Verwendung von push.<pushFanoutFuncs>.
func collectPushFanoutOccurrences(t *testing.T) []pushFanoutOccurrence {
	t.Helper()
	root := internalRoot(t)
	var occs []pushFanoutOccurrence

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
		if pkg == "notify" || pkg == "push" {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		pushAlias := ""
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if p != modulePrefix+"push" {
				continue
			}
			pushAlias = "push"
			if imp.Name != nil {
				pushAlias = imp.Name.Name
			}
		}
		if pushAlias == "" {
			return nil // Datei importiert internal/push gar nicht.
		}

		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != pushAlias {
				return true
			}
			if !pushFanoutFuncs[sel.Sel.Name] {
				return true
			}
			occs = append(occs, pushFanoutOccurrence{
				pkg:  pkg,
				fn:   sel.Sel.Name,
				file: rel,
				line: fset.Position(sel.Pos()).Line,
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

func TestArchitecture_KeinDirekterPushFanout(t *testing.T) {
	occs := collectPushFanoutOccurrences(t)
	for _, o := range occs {
		if _, ok := pushFanoutAllowlist[o.key()]; ok {
			continue
		}
		t.Errorf("%s:%d: internal/%s greift direkt auf push.%s zu — über notify.Send (mit notify.NoEmail()/notify.SkipPushPref() falls nötig) umleiten oder mit Begründung in pushFanoutAllowlist aufnehmen",
			o.file, o.line, o.pkg, o.fn)
	}
}

// TestArchitecture_PushAllowlistOhneWaisen stellt sicher, dass jeder
// pushFanoutAllowlist-Eintrag auf einen real gefundenen Aufrufer zeigt
// (verhindert Verrottung, analog TestBroadcastAllowlist_NoOrphans).
func TestArchitecture_PushAllowlistOhneWaisen(t *testing.T) {
	occs := collectPushFanoutOccurrences(t)
	found := map[string]bool{}
	for _, o := range occs {
		found[o.key()] = true
	}
	for key := range pushFanoutAllowlist {
		if !found[key] {
			t.Errorf("pushFanoutAllowlist-Eintrag %q zeigt auf keinen gefundenen push-Zugriff (veraltet? Tippfehler?)", key)
		}
	}
}
