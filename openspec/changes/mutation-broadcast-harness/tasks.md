# Tasks — mutation-broadcast-harness

> Stdlib-AST-Test, Vorbild `internal/arch/arch_test.go`. Erst das Gate bauen, dann die von ihm aufgedeckten Rest-Verstöße beheben. Kein Produktpfad-Code außer nachgezogenen Broadcasts.

## 1. Router-Parsing

- [x] 1.1 Helfer, der `internal/app/router.go` (`BuildRouter`) per `go/parser` einliest und alle `r.Post`/`r.Put`/`r.Patch`/`r.Delete`-Registrierungen extrahiert (Feld→Package über Handlers-Struct + Import-Aliase; rekursiv in Group/Route-Closures via ast.Inspect).
- [x] 1.2 Sanity-Asserts im Gate: `len(routes) > 0` (Parser defekt sonst) und `unresolved == 0` (jeder Handler-Ausdruck auflösbar). `TestBroadcastAllowlist_NoOrphans` übt `collectMutationRoutes` zusätzlich.

## 2. Handler-Rumpf-Prüfung

- [x] 2.1 Für `(package, method)` die Methode in `internal/<pkg>/*.go` per AST finden und den Rumpf auf einen `CallExpr` mit `broadcast` (case-insensitive) im Bezeichner absuchen — erkennt `h.hub.Broadcast`/`BroadcastToUsers` UND Helfer wie `broadcastMembers`.
- [x] 2.2 Robustheit: Methode nicht gefunden / Handler-Ausdruck nicht auflösbar → aussagekräftiger Testfehler (kein stilles Überspringen).

## 3. Gate + Allowlist

- [x] 3.1 Test `TestEveryMutationRouteBroadcasts` in `internal/arch/broadcast_test.go`: mutierend ∧ kein Broadcast ∧ nicht Allowlist → `t.Errorf` mit Package/Methode/Route.
- [x] 3.2 Explizite `broadcastAllowlist` (Route → Begründung): Auth/Session, Chat (eigener Kanal), Push-Subscription, Kalender-Feed-Token, Dokumente (kein files-Event), Mailversand, Beitragslauf, SEPA-Mandat-Delete, Reminder-Präferenz.
- [x] 3.3 `TestBroadcastAllowlist_NoOrphans`: jeder Allowlist-Eintrag muss auf eine registrierte Mutations-Route zeigen.

## 4. Rest-Verstöße beheben

- [x] 4.1 Gate ausgeführt: 53 Verstöße aufgedeckt → 22 allowlisted (mit Begründung), 31 Broadcasts nachgezogen.
- [x] 4.2 Broadcasts nachgezogen + Tests: auth (11 Routen → users/members via broadcastFinance), members (10 → members via broadcastMembers), upload (6 Foto-Routen → members), games (2 Regen → duties), duties (SetSeasonTargets → duties), videos (CreateUpload → video-queued).
- [x] 4.3 `make test` grün (Broadcast-Gate + alle neuen Broadcast-Tests).

## 5. Doku

- [x] 5.1 `docs/agent/08-verification.md` um das Broadcast-Gate ergänzt (neben Architektur-Test und `/verify-change`).
- [ ] 5.2 `openspec validate mutation-broadcast-harness --strict` grün; Change nach Merge archivieren.
