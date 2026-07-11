// Broadcast-Invariante als Test (Harness-Engineering, Säule 2 — mechanisch
// erzwungene Konventionen). CLAUDE.md-Hard-Rule: "Jede Mutations-Route ruft
// h.hub.Broadcast(...)". Dieser Test parst BuildRouter, sammelt alle mutierenden
// Routen (POST/PUT/PATCH/DELETE) und prüft, dass der zugehörige Handler-Rumpf
// einen Broadcast-artigen Aufruf enthält (direkt oder über einen Helfer). Nur
// stdlib (go/parser, go/ast) — konsistent mit arch_test.go.
//
// Ausnahmen ausschließlich über die begründete Allowlist unten. Ein
// Allowlist-Eintrag, der auf keine real registrierte Route zeigt, lässt den Test
// fehlschlagen (Anti-Verrottung).
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

// mutationVerbs sind die Chi-Router-Methoden, die eine schreibende Route
// registrieren. GET/HEAD/OPTIONS ändern keinen beobachtbaren Zustand.
var mutationVerbs = map[string]bool{"Post": true, "Put": true, "Patch": true, "Delete": true}

// broadcastAllowlist: mutierende Routen, die BEWUSST keinen Broadcast senden,
// weil sie keinen von anderen eingeloggten Clients live beobachtbaren Zustand
// ändern. Schlüssel = "<HandlersFeld>.<Methode>", Wert = Begründung.
// Jeder Eintrag MUSS auf eine real registrierte Mutations-Route zeigen (sonst
// schlägt TestBroadcastAllowlist_NoOrphans fehl).
var broadcastAllowlist = map[string]string{
	// Auth/Session: ändern Token/Session/Credentials des Aufrufers, keine
	// beobachtbare Domänenliste anderer Clients.
	"Auth.Login":                      "Session-Token-Ausgabe, kein geteilter State",
	"Auth.Logout":                     "beendet eigene Session",
	"Auth.Refresh":                    "rotiert eigenen Refresh-Token",
	"Auth.ForgotPassword":             "verschickt Reset-Mail, kein beobachtbarer State",
	"Auth.ResetPassword":              "setzt eigenes Passwort",
	"Auth.ChangePassword":             "ändert eigenes Passwort",
	"Auth.Register":                   "öffentlicher Beitrittsantrag (kein Broadcast-Empfänger vor Freigabe)",
	"Auth.RequestMembership":          "öffentlicher Beitrittsantrag",
	"Auth.RequestEmailChange":         "startet eigenen E-Mail-Wechsel (Bestätigung per Mail)",
	"Auth.RequestRecoveryEmailChange": "startet eigenen Recovery-Mail-Wechsel (Bestätigung per Mail)",
	"Auth.UpdateAccount":              "ändert eigene Account-Einstellungen",
	"Auth.Impersonate":                "erzeugt Admin-Token, kein Domänen-State",
	// Chat: eigener Realtime-Kanal (/api/chat/events + Push), nicht der Hub.
	"Chat.CreateConversation": "Chat nutzt eigenen SSE-Kanal /api/chat/events, nicht den Hub",
	"Chat.EditBroadcast":      "Chat nutzt eigenen SSE-Kanal /api/chat/events, nicht den Hub",
	"Chat.DeleteBroadcast":    "Chat nutzt eigenen SSE-Kanal /api/chat/events, nicht den Hub",
	// Push-Subscription / Benachrichtigungs-Präferenzen: pro Gerät/Nutzer, keine geteilte Live-Liste.
	"Notif.Subscribe":                     "Push-Subscription pro Gerät, kein geteilter State",
	"Notif.Unsubscribe":                   "Push-Subscription pro Gerät, kein geteilter State",
	"Notif.UpdateNotificationPreferences": "eigene Benachrichtigungs-Präferenzen, kein geteilter State",
	// Kalender-Feed-Token: pro Nutzer, kein beobachtbarer geteilter State.
	"Calendar.UpsertToken": "per-Nutzer ICS-Feed-Token, kein geteilter State",
	"Calendar.DeleteToken": "per-Nutzer ICS-Feed-Token, kein geteilter State",
	// Dokumente: die Dokumente-Seite hat kein Hub-Live-Update (kein 'files'-Event).
	"Files.CreateFolder":     "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.UploadFile":       "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Media.Upload":           "Upload-Vorstufe, kein Live-Update; der nachfolgende SendMessage/SendBroadcast broadcastet",
	"Files.DeleteFile":       "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.DeleteFolder":     "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.RenameFile":       "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.RenameFolder":     "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.AddPermission":    "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	"Files.DeletePermission": "kein 'files'-Hub-Event; Dokumente-Seite lädt ohne Live-Update",
	// Mailversand / einzelne Kassierer-Vorgänge ohne beobachtbare Live-Liste.
	"WelcomeEmail.Send":       "Mailversand, kein beobachtbarer State",
	"Beitragslauf.Confirm":    "einzelner Kassierer-Vorgang; schreibt append-only Protokoll, keine members-Änderung",
	"Beitragslauf.ExportData": "liefert Ciphertext für SEPA-Export, keine members-Änderung",
	// Datei-/Upload-Nebeneffekte ohne beobachtbare Live-Liste.
	"Upload.DeleteSepaMandat": "PII-Datei-Löschung, kein Live-Listen-State",
	// Reine Selbst-Präferenz, erscheint in keiner fremden Ansicht.
	"Members.UpdateReminderPreference": "eigene Dienst-Erinnerungs-Präferenz (users.duty_reminder_days), kein geteilter View",
}

// resolveHandlerPackages liest den Handlers-Struct und die Import-Aliase aus
// router.go und liefert: fieldName -> (pkgDir, receiverType).
type handlerRef struct {
	pkgDir   string // Verzeichnis unter internal/ (z. B. "config")
	recvType string // Empfänger-Typname (z. B. "Handler", "WelcomeEmailHandler")
}

func routerFilePath() string { return filepath.Join("..", "app", "router.go") }

func parseRouterFile(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, routerFilePath(), nil, 0)
	if err != nil {
		t.Fatalf("parse router.go: %v", err)
	}
	return fset, f
}

// importAliasToDir: Alias (bzw. Paketname) -> letztes Pfadsegment (= internal-Dir).
func importAliasToDir(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(path, modulePrefix) {
			continue
		}
		dir := path[strings.LastIndex(path, "/")+1:]
		alias := dir
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		out[alias] = dir
	}
	return out
}

// handlerFieldRefs: Feldname im Handlers-Struct -> handlerRef, für Felder vom
// Typ *<internalPkg>.<Type>. Nicht-Domänen-Felder (http.Handler, primitive)
// werden übersprungen.
func handlerFieldRefs(f *ast.File, aliasToDir map[string]string) map[string]handlerRef {
	out := map[string]handlerRef{}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Handlers" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			if len(field.Names) == 0 {
				continue
			}
			// Erwartet *pkg.Type
			star, ok := field.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				continue
			}
			dir, ok := aliasToDir[pkgIdent.Name]
			if !ok {
				continue // z. B. http.Handler oder externes Paket
			}
			out[field.Names[0].Name] = handlerRef{pkgDir: dir, recvType: sel.Sel.Name}
		}
		return true
	})
	return out
}

// mutationRoute: eine im Router registrierte schreibende Route.
type mutationRoute struct {
	verb   string
	field  string // Handlers-Feld (z. B. "Members")
	method string // Handler-Methode (z. B. "Update")
}

func (m mutationRoute) key() string { return m.field + "." + m.method }

// collectMutationRoutes durchläuft BuildRouter (inkl. verschachtelter
// Group/Route-Closures — ast.Inspect rekursiert in FuncLits) und sammelt alle
// r.<Verb>(path, h.<Field>.<Method>)-Registrierungen. handlerExpr-Formen, die
// nicht h.Field.Method sind, landen in unresolved.
func collectMutationRoutes(f *ast.File) (routes []mutationRoute, unresolved []string) {
	var buildRouter *ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "BuildRouter" {
			buildRouter = fd
			break
		}
	}
	if buildRouter == nil {
		return nil, []string{"BuildRouter nicht gefunden"}
	}
	ast.Inspect(buildRouter.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !mutationVerbs[sel.Sel.Name] {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		handler := call.Args[len(call.Args)-1]
		field, method, ok := handlerFieldMethod(handler)
		if !ok {
			unresolved = append(unresolved, "unaufgelöster Handler bei "+sel.Sel.Name)
			return true
		}
		routes = append(routes, mutationRoute{verb: sel.Sel.Name, field: field, method: method})
		return true
	})
	return routes, unresolved
}

// handlerFieldMethod extrahiert (Field, Method) aus h.<Field>.<Method>.
func handlerFieldMethod(expr ast.Expr) (field, method string, ok bool) {
	outer, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	inner, ok := outer.X.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	if base, ok := inner.X.(*ast.Ident); !ok || base.Name != "h" {
		return "", "", false
	}
	return inner.Sel.Name, outer.Sel.Name, true
}

// methodBroadcasts prüft, ob die Methode recvType.method im Package unter
// internal/<pkgDir> einen Broadcast-artigen Aufruf enthält. found=false, wenn
// die Methode nicht gefunden wurde.
func methodBroadcasts(pkgDir, recvType, method string) (broadcasts, found bool) {
	dir := filepath.Join("..", pkgDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, false
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != method {
				continue
			}
			if !recvTypeMatches(fd.Recv, recvType) {
				continue
			}
			return bodyHasBroadcast(fd.Body), true
		}
	}
	return false, false
}

func recvTypeMatches(recv *ast.FieldList, recvType string) bool {
	if recv == nil || len(recv.List) == 0 {
		return false
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.Name == recvType
}

// bodyHasBroadcast: enthält der Rumpf einen CallExpr, dessen aufgerufener
// Bezeichner "broadcast" (case-insensitive) enthält? Deckt h.hub.Broadcast,
// BroadcastToUsers UND Helfer wie broadcastMembers/broadcastGame ab.
func bodyHasBroadcast(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var name string
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			name = fn.Sel.Name
		case *ast.Ident:
			name = fn.Name
		}
		if strings.Contains(strings.ToLower(name), "broadcast") {
			found = true
			return false
		}
		return true
	})
	return found
}

func TestEveryMutationRouteBroadcasts(t *testing.T) {
	_, f := parseRouterFile(t)
	aliasToDir := importAliasToDir(f)
	fields := handlerFieldRefs(f, aliasToDir)
	routes, unresolved := collectMutationRoutes(f)

	if len(unresolved) > 0 {
		t.Errorf("konnte %d Mutations-Handler nicht auflösen (Parser erweitern oder Router prüfen): %s",
			len(unresolved), strings.Join(unresolved, "; "))
	}
	if len(routes) == 0 {
		t.Fatal("keine Mutations-Routen gefunden — Parser defekt?")
	}

	for _, rt := range routes {
		if _, ok := broadcastAllowlist[rt.key()]; ok {
			continue
		}
		ref, ok := fields[rt.field]
		if !ok {
			t.Errorf("Feld %q (Route %s %s) nicht im Handlers-Struct auf ein internes Package abbildbar",
				rt.field, rt.verb, rt.key())
			continue
		}
		broadcasts, found := methodBroadcasts(ref.pkgDir, ref.recvType, rt.method)
		if !found {
			t.Errorf("Methode %s.%s (internal/%s) nicht gefunden — Router/Handler inkonsistent",
				ref.recvType, rt.method, ref.pkgDir)
			continue
		}
		if !broadcasts {
			t.Errorf("Mutations-Route %s %s ruft keinen Broadcast — entweder h.hub.Broadcast/BroadcastToUsers/Helfer ergänzen oder mit Begründung in broadcastAllowlist aufnehmen",
				rt.verb, rt.key())
		}
	}
}

// TestBroadcastAllowlist_NoOrphans stellt sicher, dass jeder Allowlist-Eintrag
// auf eine real registrierte Mutations-Route zeigt (verhindert Verrottung).
func TestBroadcastAllowlist_NoOrphans(t *testing.T) {
	_, f := parseRouterFile(t)
	routes, _ := collectMutationRoutes(f)
	registered := map[string]bool{}
	for _, rt := range routes {
		registered[rt.key()] = true
	}
	for key := range broadcastAllowlist {
		if !registered[key] {
			t.Errorf("Allowlist-Eintrag %q zeigt auf keine registrierte Mutations-Route (veraltet? Tippfehler?)", key)
		}
	}
}
