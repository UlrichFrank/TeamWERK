# Tasks — dashboard-offene-gesuche

> Ein Commit pro Task. Conventional Commits, Scope `dashboard`.

## 1. Backend: offene Gesuche der eigenen Teams abfragen

- [x] 1.1 In `internal/dashboard/handler.go` Response-Typen ergänzen: `CarpoolingOpenGroup { gameId, date, title, requests[] }` und `CarpoolingOpenRequest { sucheId, requesterName, plaetze, treffpunkt }`; Feld `CarpoolingOpenGroups []CarpoolingOpenGroup json:"carpoolingOpenGroups"` an `Response` anhängen und in `Get` initialisieren.
- [x] 1.2 Methode `queryCarpoolingOpenRequests(r, userID, seasonID)`: nächste max. 3 künftige Spiele der eigenen Teams (`teamQueryForUser` / `user_accessible_teams`, alle `event_type`), je Spiel die `suche`-Einträge ohne `confirmed`-Paarung (`NOT EXISTS (… mitfahrt_paarungen … status='confirmed')`). In `Get` aufrufen und `resp.CarpoolingOpenGroups` setzen.
- [x] 1.3 `go vet` + `gofmt`.

  _Commit:_ `feat(dashboard): offene Mitfahr-Gesuche der eigenen Teams in API`

## 2. Backend-Tests

- [x] 2.1 `TestDashboard_OffeneGesuche_OwnTeam` (Happy-Path: erscheint).
- [x] 2.2 `TestDashboard_OffeneGesuche_ConfirmedExcluded` (confirmed → nicht offen, aber in `carpoolingConfirmed`).
- [x] 2.3 `TestDashboard_OffeneGesuche_PendingStillOpen` (pending → bleibt offen).
- [x] 2.4 `TestDashboard_OffeneGesuche_OtherTeamExcluded` (fremdes Team → nicht sichtbar).

  _Commit:_ `test(dashboard): offene Gesuche – Happy-Path und Abgrenzungen`

## 3. Frontend: Block „Offene Gesuche"

- [x] 3.1 In `web/src/pages/DashboardPage.tsx` Response-Typ um `carpoolingOpenGroups` erweitern.
- [x] 3.2 In `FahrgemeinschaftenSection` unter den bestätigten Paarungen einen Block „Offene Gesuche" rendern: pro Gruppe Datum + Titel, je Gesuch Name · „braucht N Plätze" (· Treffpunkt), `lucide-react`-Icon (`Search`), `brand-*`-Tokens, Leerzustand sauber. Live-Update läuft über bestehendes `useLiveUpdates('mitfahrgelegenheiten')`.
- [x] 3.3 `pnpm -C web build` (inkl. `tsc -b`) grün. Hinweis: `pnpm lint` ist repo-weit nicht lauffähig (keine `eslint.config.js`/`.eslintrc` im Repo, ESLint v9) — vorbestehend, unabhängig von dieser Änderung; ersatzweise `tsc --noEmit` ohne Fehler.

  _Commit:_ `feat(dashboard): Dashboard zeigt offene Mitfahr-Gesuche der eigenen Teams`

## 4. Abschluss

- [x] 4.1 Verifikation: `go vet ./...` sauber, `go test -race ./...` grün (inkl. Architektur-Test), `tsc -b`/`tsc --noEmit` ohne Fehler. Invarianten geprüft (Route→Tests ✓, keine Mutation → kein Broadcast ✓, brand-Tokens ✓, lucide `Search` ✓, keine Migration ✓). `pnpm lint` repo-weit nicht lauffähig (keine ESLint-Config eingecheckt) — vorbestehend.
- [x] 4.2 `openspec validate dashboard-offene-gesuche --strict` → valid.
- [ ] 4.3 Proposal archivieren (offen — bewusst dem Nutzer überlassen, nach Commit/Review).

  _Commit:_ `chore(dashboard): archiviere dashboard-offene-gesuche`
