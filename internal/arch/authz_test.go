// Authz-Drift-Detektor als Test (Harness-Engineering, Säule 2 — mechanisch
// erzwungene Konventionen). Analog zum Broadcast-Gate (broadcast_test.go), aber
// für die Autorisierungs-Dimension: Er stellt sicher, dass jede Route, die in
// BuildRouter hinter einem RequireRole/RequireClubFunction-Gate liegt, auch in
// den hand-gepflegten Persona-Erwartungen der Permission-Matrix
// (internal/permissions/matrix_test.go) erfasst ist — oder mit Begründung in der
// authzAllowlist unten steht.
//
// Warum ein zweiter, statischer Detektor neben TestPermissionMatrix_Backend?
// Der Matrix-Test walkt den LAUFZEIT-Router (prodserver). Handler-Felder, die
// prodserver (noch) nicht verdrahtet, werden über `if h.X != nil { ... }`
// übersprungen — ihre gated Routen tauchen im Laufzeit-Router NICHT auf und
// entgehen damit dem Matrix-Drift-Check (z. B. Match-Reports, Admin-
// Wartungsmodus). Dieser Detektor liest den ROUTER-QUELLTEXT via go/ast
// (statisch), sieht also auch diese Routen und zwingt jede bewusst nicht in der
// Matrix erfasste gated Route in eine dokumentierte Allowlist.
//
// Nur stdlib (go/parser, go/ast) — konsistent mit arch_test.go/broadcast_test.go.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// httpVerbMethods bildet die chi-Router-Registrierungsmethoden auf ihr
// HTTP-Verb ab. r.Group/r.Route/r.Use/r.Handle sind KEINE Verb-Registrierungen
// und tauchen hier bewusst nicht auf.
var httpVerbMethods = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Patch": "PATCH",
	"Delete": "DELETE", "Options": "OPTIONS", "Head": "HEAD",
	"Connect": "CONNECT", "Trace": "TRACE",
}

// roleConstants ist die Alias-Tabelle, die die auth.Role*-Konstanten
// (SelectorExpr `auth.RoleX`) auf ihre String-Werte auflöst. Quelle:
// internal/auth/roles.go. Muss synchron zu roles.go gehalten werden — ein neuer
// Rollen-Konstant, der hier fehlt, führt lediglich zu einem "auth.RoleX"-
// Platzhalter in der Gate-Beschreibung, nie zu einem falsch-negativen Match
// (die Membership-Prüfung nutzt nur method+path).
var roleConstants = map[string]string{
	"RoleAdmin":     "admin",
	"RoleStandard":  "standard",
	"RolePressTeam": "presseteam",
}

// authzAllowlist: gated Routen, die BEWUSST nicht in der Permission-Matrix
// (internal/permissions/matrix_test.go) stehen. Schlüssel = "<METHOD> <path>",
// Wert = Begründung. Jeder Eintrag MUSS auf eine real gated Route in BuildRouter
// zeigen (sonst schlägt TestArch_AuthzMatrix_NoOrphans fehl).
//
// Aktueller Grund für alle Einträge: Das zugehörige Handler-Feld wird vom
// Test-Router (internal/testutil/prodserver) noch nicht verdrahtet, die Route
// steht daher unter `if h.X != nil { ... }` und fehlt im LAUFZEIT-Router — der
// Matrix-Drift-Check kann sie folglich nicht erzwingen. Sobald prodserver die
// Felder verdrahtet (separater Slice), gehören diese Routen in die Matrix und
// die Einträge hier werden entfernt.
var authzAllowlist = map[string]string{
	// Admin-Wartungsmodus — RequireRole("admin"), nur wenn h.Settings != nil.
	// prodserver setzt h.Settings (noch) nicht → nicht im Laufzeit-Router / in der Matrix.
	"GET /api/admin/maintenance-mode":  "RequireRole(admin); prodserver verdrahtet h.Settings (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	"POST /api/admin/maintenance-mode": "RequireRole(admin); prodserver verdrahtet h.Settings (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	// Match-Reports — unter `if h.MatchReports != nil`. prodserver verdrahtet
	// MatchReports (noch) nicht → nicht im Laufzeit-Router / in der Matrix.
	// Autor-Gate: RequireRole(presseteam, admin).
	"GET /api/match-reports/my":                      "RequireRole(presseteam,admin); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	"POST /api/match-reports":                        "RequireRole(presseteam,admin); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	"DELETE /api/match-reports/{id}":                 "RequireRole(presseteam,admin); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	"POST /api/match-reports/{id}/submit-for-review": "RequireRole(presseteam,admin); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	// Freigeber-Gate: RequireClubFunction(medien, vorstand).
	"GET /api/match-reports/pending":       "RequireClubFunction(medien,vorstand); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
	"POST /api/match-reports/{id}/publish": "RequireClubFunction(medien,vorstand); prodserver verdrahtet h.MatchReports (noch) nicht → fehlt im Laufzeit-Router/Matrix",
}

// authzGate beschreibt ein einzelnes RequireRole/RequireClubFunction-Gate mit
// seinen aufgelösten String-Argumenten.
type authzGate struct {
	kind string   // "role" | "clubfunc"
	args []string // aufgelöste String-Werte (auth.Role*-Konstanten via roleConstants)
}

func (g authzGate) String() string {
	fn := "RequireClubFunction"
	if g.kind == "role" {
		fn = "RequireRole"
	}
	return fn + "(" + strings.Join(g.args, ",") + ")"
}

// gatedRoute ist eine in BuildRouter registrierte Route samt der beim
// Registrierungspunkt aktiven (umschließenden) Gates.
type gatedRoute struct {
	method string
	path   string
	gates  []authzGate
}

func (r gatedRoute) key() string { return r.method + " " + r.path }

func gatesString(gs []authzGate) string {
	parts := make([]string, 0, len(gs))
	for _, g := range gs {
		parts = append(parts, g.String())
	}
	return strings.Join(parts, " ∧ ")
}

// authzWalker läuft rekursiv und scope-bewusst durch BuildRouter. Er hält den
// Import-Alias des internal/auth-Packages (zur Auflösung von auth.RequireRole/
// RequireClubFunction) und sammelt alle registrierten Routen mit ihren aktiven
// Gates.
type authzWalker struct {
	authAlias string
	routes    []gatedRoute
}

// collectGatedRoutes parst BuildRouter und liefert ausschließlich Routen, die
// unter mindestens einem RequireRole/RequireClubFunction-Gate stehen.
func collectGatedRoutes(f *ast.File) []gatedRoute {
	authAlias := "auth"
	for alias, dir := range importAliasToDir(f) {
		if dir == "auth" {
			authAlias = alias
		}
	}
	var buildRouter *ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "BuildRouter" {
			buildRouter = fd
			break
		}
	}
	if buildRouter == nil {
		return nil
	}
	w := &authzWalker{authAlias: authAlias}
	// Top-Level-Gates (r.Use im Rumpf von BuildRouter): InFlight/Recoverer/
	// CleanPath/securityHeaders/cors/Maintenance — keine Authz-Gates, ergibt leer.
	base := w.scanUseGates(buildRouter.Body)
	w.walk(buildRouter.Body.List, base, "")

	var out []gatedRoute
	for _, rt := range w.routes {
		if len(rt.gates) > 0 {
			out = append(out, rt)
		}
	}
	return out
}

// walk verarbeitet eine Statement-Liste unter dem gegebenen Gate-Stack und
// Pfad-Präfix. `if`-Blöcke (z. B. `if h.X != nil { ... }`) ändern die
// Middleware-Kette nicht → gleiche Gates/Präfix. Neue Scopes entstehen nur durch
// r.Group/r.Route (handleCall).
func (w *authzWalker) walk(stmts []ast.Stmt, gates []authzGate, prefix string) {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ast.IfStmt:
			if s.Body != nil {
				w.walk(s.Body.List, gates, prefix)
			}
			switch e := s.Else.(type) {
			case *ast.BlockStmt:
				w.walk(e.List, gates, prefix)
			case *ast.IfStmt:
				w.walk([]ast.Stmt{e}, gates, prefix)
			}
		case *ast.ExprStmt:
			if call, ok := s.X.(*ast.CallExpr); ok {
				w.handleCall(call, gates, prefix)
			}
		}
	}
}

// handleCall behandelt eine einzelne r.<X>(...)-Registrierung: Verb-Routen
// werden mit den aktiven Gates gespeichert; r.Group/r.Route öffnen einen neuen
// Scope (eigene r.Use-Gates werden auf den Stack gelegt).
func (w *authzWalker) handleCall(call *ast.CallExpr, gates []authzGate, prefix string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	name := sel.Sel.Name
	switch name {
	case "Group":
		fl := funcLitArg(call.Args)
		if fl == nil {
			return
		}
		child := append(cloneGates(gates), w.scanUseGates(fl.Body)...)
		w.walk(fl.Body.List, child, prefix)
	case "Route":
		if len(call.Args) < 2 {
			return
		}
		fl, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			return
		}
		sub := prefix + stringLit(call.Args[0])
		child := append(cloneGates(gates), w.scanUseGates(fl.Body)...)
		w.walk(fl.Body.List, child, sub)
	default:
		verb, ok := httpVerbMethods[name]
		if !ok || len(call.Args) < 1 {
			return
		}
		p := stringLit(call.Args[0])
		if p == "" {
			return
		}
		w.routes = append(w.routes, gatedRoute{
			method: verb,
			path:   prefix + p,
			gates:  cloneGates(gates),
		})
	}
}

// scanUseGates sammelt die RequireRole/RequireClubFunction-Gates aus den
// r.Use(...)-Aufrufen im DIREKTEN Rumpf eines Scopes. Es rekursiert in
// `if`-Blöcke (r.Use kann unter `if h.X != nil` stehen), steigt aber NICHT in
// verschachtelte r.Group/r.Route-FuncLits ab — deren r.Use gehören zum
// Kind-Scope und werden dort erneut gescannt.
func (w *authzWalker) scanUseGates(body *ast.BlockStmt) []authzGate {
	var gates []authzGate
	var visit func(stmts []ast.Stmt)
	visit = func(stmts []ast.Stmt) {
		for _, st := range stmts {
			switch s := st.(type) {
			case *ast.IfStmt:
				if s.Body != nil {
					visit(s.Body.List)
				}
				if e, ok := s.Else.(*ast.BlockStmt); ok {
					visit(e.List)
				}
			case *ast.ExprStmt:
				call, ok := s.X.(*ast.CallExpr)
				if !ok {
					continue
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Use" {
					continue
				}
				for _, a := range call.Args {
					if g, ok := w.gateFromUseArg(a); ok {
						gates = append(gates, g)
					}
				}
			}
		}
	}
	visit(body.List)
	return gates
}

// gateFromUseArg erkennt `auth.RequireRole(...)` / `auth.RequireClubFunction(...)`
// als Argument eines r.Use(...)-Aufrufs und löst die String-Argumente auf.
func (w *authzWalker) gateFromUseArg(arg ast.Expr) (authzGate, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return authzGate{}, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return authzGate{}, false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != w.authAlias {
		return authzGate{}, false
	}
	var kind string
	switch sel.Sel.Name {
	case "RequireRole":
		kind = "role"
	case "RequireClubFunction":
		kind = "clubfunc"
	default:
		return authzGate{}, false
	}
	g := authzGate{kind: kind}
	for _, a := range call.Args {
		g.args = append(g.args, w.resolveStringArg(a))
	}
	return g, true
}

// resolveStringArg löst ein Gate-Argument zu seinem String-Wert auf: String-
// Literal direkt, auth.Role*-Konstante via roleConstants (Alias-Tabelle).
func (w *authzWalker) resolveStringArg(a ast.Expr) string {
	switch e := a.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			if s, err := strconv.Unquote(e.Value); err == nil {
				return s
			}
		}
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok && x.Name == w.authAlias {
			if v, ok := roleConstants[e.Sel.Name]; ok {
				return v
			}
			return "auth." + e.Sel.Name // unbekannte Konstante — nur Anzeige
		}
	case *ast.Ident:
		return e.Name
	}
	return "?"
}

// ── Hilfsfunktionen (AST) ─────────────────────────────────────────────────────

func funcLitArg(args []ast.Expr) *ast.FuncLit {
	for _, a := range args {
		if fl, ok := a.(*ast.FuncLit); ok {
			return fl
		}
	}
	return nil
}

func stringLit(e ast.Expr) string {
	if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		if s, err := strconv.Unquote(bl.Value); err == nil {
			return s
		}
	}
	return ""
}

func cloneGates(g []authzGate) []authzGate {
	out := make([]authzGate, len(g))
	copy(out, g)
	return out
}

// ── Matrix-Extraktion (statisch aus matrix_test.go) ───────────────────────────

func matrixFilePath() string { return filepath.Join("..", "permissions", "matrix_test.go") }

// parseMatrixRouteKeys parst `var matrix = []endpointCase{...}` und liefert die
// Menge der "<METHOD> <path>"-Schlüssel. Statisch (go/ast) — unabhängig von der
// prodserver-Laufzeitverdrahtung, konsistent mit dem Router-Parsing.
func parseMatrixRouteKeys(t *testing.T) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, matrixFilePath(), nil, 0)
	if err != nil {
		t.Fatalf("parse matrix_test.go: %v", err)
	}
	keys := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range vs.Names {
			if name.Name != "matrix" || i >= len(vs.Values) {
				continue
			}
			cl, ok := vs.Values[i].(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, elt := range cl.Elts {
				ec, ok := elt.(*ast.CompositeLit)
				if !ok {
					continue
				}
				var method, path string
				for _, e := range ec.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok {
						continue
					}
					switch key.Name {
					case "method":
						method = stringLit(kv.Value)
					case "path":
						path = stringLit(kv.Value)
					}
				}
				if method != "" && path != "" {
					keys[method+" "+path] = true
				}
			}
		}
		return true
	})
	return keys
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestArch_AuthzGatesMatchMatrix stellt sicher, dass jede in BuildRouter hinter
// einem RequireRole/RequireClubFunction-Gate liegende Route in den Persona-
// Erwartungen der Permission-Matrix erfasst ist — oder mit Begründung in der
// authzAllowlist steht.
func TestArch_AuthzGatesMatchMatrix(t *testing.T) {
	_, f := parseRouterFile(t)
	routes := collectGatedRoutes(f)
	if len(routes) == 0 {
		t.Fatal("keine gated Routen in BuildRouter gefunden — Walker defekt?")
	}
	matrixKeys := parseMatrixRouteKeys(t)
	if len(matrixKeys) == 0 {
		t.Fatal("keine Matrix-Routen aus matrix_test.go geparst — Parser defekt?")
	}

	for _, rt := range routes {
		if _, ok := authzAllowlist[rt.key()]; ok {
			continue
		}
		if !matrixKeys[rt.key()] {
			t.Errorf("gated Route %q (Gates: %s) fehlt in der Permission-Matrix "+
				"(internal/permissions/matrix_test.go) — Persona-Erwartungen dort ergänzen "+
				"oder mit Begründung in authzAllowlist aufnehmen",
				rt.key(), gatesString(rt.gates))
		}
	}
}

// TestArch_AuthzMatrix_NoOrphans stellt sicher, dass jeder authzAllowlist-Eintrag
// auf eine real gated Route in BuildRouter zeigt (verhindert Verrottung: eine
// entfernte oder entgatete Route lässt den Test fehlschlagen).
func TestArch_AuthzMatrix_NoOrphans(t *testing.T) {
	_, f := parseRouterFile(t)
	gatedKeys := map[string]bool{}
	for _, rt := range collectGatedRoutes(f) {
		gatedKeys[rt.key()] = true
	}
	for key := range authzAllowlist {
		if !gatedKeys[key] {
			t.Errorf("authzAllowlist-Eintrag %q zeigt auf keine gated Route in BuildRouter "+
				"(veraltet? Tippfehler? Route entfernt oder entgatet?)", key)
		}
	}
}
