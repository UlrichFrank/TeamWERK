# Tasks — mutation-broadcast-harness

> Stdlib-AST-Test, Vorbild `internal/arch/arch_test.go`. Erst das Gate bauen, dann die von ihm aufgedeckten Rest-Verstöße beheben. Kein Produktpfad-Code außer nachgezogenen Broadcasts.

## 1. Router-Parsing

- [ ] 1.1 Helfer, der `internal/app/router.go` (`BuildRouter`) per `go/parser` einliest und alle `r.Post`/`r.Put`/`r.Patch`/`r.Delete`-Registrierungen als `{method, path, handlerRecv, handlerMethod}` extrahiert (Router-Var → Domänen-Package über die lokalen Deklarationen in `BuildRouter` auflösen).
- [ ] 1.2 Unit-Test des Extraktors gegen einen kleinen, bekannten Ausschnitt (Anzahl Mutations-Routen > 0, eine erwartete Route ist enthalten).

## 2. Handler-Rumpf-Prüfung

- [ ] 2.1 Für `(package, method)` die Methode in `internal/<pkg>/*.go` per AST finden und den Rumpf auf ein `CallExpr` mit `Broadcast` im Bezeichner absuchen (Selektor- und Ident-Namen; erkennt Helfer).
- [ ] 2.2 Robustheit: Methode nicht gefunden / Handler-Ausdruck nicht auflösbar → aussagekräftiger Testfehler (kein stilles Überspringen).

## 3. Gate + Allowlist

- [ ] 3.1 Test `TestEveryMutationRouteBroadcasts` in `internal/arch/` (oder neuem `internal/harness/`): mutierend ∧ kein Broadcast ∧ nicht Allowlist → `t.Errorf` mit Package/Methode/Route.
- [ ] 3.2 Explizite Allowlist `map[…]string` (Route → Begründung) mit den bekannten legitimen Ausnahmen (`/api/auth/*`, Impersonation, Push-Subscription-Registrierung, reine Datei-/Export-Downloads).
- [ ] 3.3 Verwaiste-Allowlist-Prüfung: jeder Allowlist-Eintrag muss auf eine real registrierte Route zeigen, sonst Testfehler.

## 4. Rest-Verstöße beheben

- [ ] 4.1 Gate ausführen, aufgedeckte Verstöße auflisten (erwartet u. a. diverse `auth`-Mutationen: `CreateUser`, `UpdateUser`, `UpdateUserRole`, `Invite`, `ImportCSV`, adult-Zweig `ApproveMembershipRequest`; `members.DeleteMember`, `members.CreateMemberFromUser`; `duties.SetSeasonTargets`).
- [ ] 4.2 Je Verstoß entscheiden: Broadcast nachziehen (Regelfall, passenden Event-String + Audience wählen — Vorbild `absences.broadcastMemberEvents`) **oder** begründet auf die Allowlist. Backend-Tests für nachgezogene Broadcasts ergänzen (Happy-Path je Route).
- [ ] 4.3 `make test` grün.

## 5. Doku

- [ ] 5.1 `docs/agent/08-verification.md` um das neue Gate ergänzen (neben Architektur-Test und `/verify-change`).
- [ ] 5.2 `openspec validate mutation-broadcast-harness --strict` grün; Change nach Umsetzung archivieren.
