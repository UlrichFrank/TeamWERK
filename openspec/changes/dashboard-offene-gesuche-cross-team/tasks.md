# Tasks — dashboard-offene-gesuche-cross-team

> Baut auf `dashboard-offene-gesuche` auf. Ein Commit pro Task. Scope `dashboard`.

## 1. Backend: Kolokation + Pool-Gruppierung

- [ ] 1.1 `queryCarpoolingOpenRequests` erweitern: Anker = eigene nächste ≤3 Spiele; per `UNION` kolozierte Fremdspiele (gleicher `date`+`venue_id`, `venue_id IS NOT NULL`) dazu (siehe `design.md`).
- [ ] 1.2 Response-Gruppierung auf (Tag, Venue) umstellen: `CarpoolingOpenGroup` um `venueName` ergänzen, `CarpoolingOpenRequest` um Spiel-/Team-Kontext (`opponent`/`teamName`). Fallback `venue_id IS NULL` → Gruppe pro `game_id`.
- [ ] 1.3 `go vet` + `gofmt`.

  _Commit:_ `feat(dashboard): offene Gesuche teamübergreifend nach Tag und Ort poolen`

## 2. Backend-Tests

- [ ] 2.1 `TestDashboard_OffeneGesuche_CrossTeamSameVenue`.
- [ ] 2.2 `TestDashboard_OffeneGesuche_CrossTeamDifferentVenue`.
- [ ] 2.3 `TestDashboard_OffeneGesuche_NullVenueNoCrossMatch`.
- [ ] 2.4 `TestDashboard_OffeneGesuche_PoolMerge`.

  _Commit:_ `test(dashboard): Cross-Team-Pool – Kolokation, Null-Venue, Merge`

## 3. Frontend: Pool-Darstellung

- [ ] 3.1 `FahrgemeinschaftenSection` auf (Tag, Ort)-Gruppen umstellen: Header `Datum · Ortsname`, je Gesuch Name · „braucht N Plätze" + Spiel-/Team-Kontext; Venue-loser Fallback pro Spiel. `brand-*`-Tokens, `lucide-react`.
- [ ] 3.2 `pnpm -C web build` + `lint`.

  _Commit:_ `feat(dashboard): Dashboard poolt offene Gesuche teamübergreifend nach Tag und Ort`

## 4. Abschluss

- [ ] 4.1 `/verify-change`.
- [ ] 4.2 `openspec validate dashboard-offene-gesuche-cross-team --strict`.
- [ ] 4.3 Proposal archivieren (aktualisiert `specs/dashboard-offene-gesuche`).

  _Commit:_ `chore(dashboard): archiviere dashboard-offene-gesuche-cross-team`
