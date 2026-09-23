> Keine Migration. Keine Mutation (kein Broadcast).

## 1. Backend

- [x] 1.1 `internal/attendance/matrix.go`: Handler `GetTeamRSVPMatrix` (Parameter-Validierung, 404/403, Spalten, Zeilen, Zellen inkl. Voreinstellung, Serien-Abmeldung, `present` nur für `canSeeTeamStats`)
- [x] 1.2 Route `GET /api/teams/{id}/rsvp-matrix` in `internal/app/router.go` (Authenticated-Tier), Tier- und Objektrechte-Matrix nachziehen
- [x] 1.3 Tests `internal/attendance/matrix_test.go` gemäß Test-Anforderungen

## 2. Frontend

- [ ] 2.1 `web/src/lib/terminMatrix.ts`: Typen + reine Funktionen (Spaltenfilter, Teilnahme-Quote) mit Vitest
- [ ] 2.2 `web/src/components/TerminMatrix.tsx`: Tabelle (sticky Namensspalte, Symbole, Legende, Kopf-Links) mit Vitest
- [ ] 2.3 `TerminePage.tsx`: Umschalter Liste/Tabelle (`view=tabelle`), Mannschafts-Einfachauswahl, Laden der Matrix, Live-Updates

## 3. Abschluss

- [ ] 3.1 `make test` / `pnpm -C web build test lint` / `openspec validate` grün
