> Keine Migration. Keine Mutation (kein Broadcast).

## 1. Backend

- [x] 1.1 `internal/attendance/matrix.go`: Handler `GetTeamRSVPMatrix` (Parameter-Validierung, 404/403, Spalten, Zeilen, Zellen inkl. Voreinstellung, Serien-Abmeldung, `present` nur für `canSeeTeamStats`)
- [x] 1.2 Route `GET /api/teams/{id}/rsvp-matrix` in `internal/app/router.go` (Authenticated-Tier), Tier- und Objektrechte-Matrix nachziehen
- [x] 1.3 Tests `internal/attendance/matrix_test.go` gemäß Test-Anforderungen

## 2. Frontend

- [x] 2.1 `web/src/lib/terminMatrix.ts`: Typen + reine Funktionen (Spaltenfilter, Teilnahme-Quote) mit Vitest
- [x] 2.2 `web/src/components/TerminMatrix.tsx`: Tabelle (sticky Namensspalte, Symbole, Legende, Kopf-Links) mit Vitest
- [x] 2.3 `TerminePage.tsx`: Umschalter Liste/Tabelle (`view=tabelle`), Mannschafts-Einfachauswahl, Laden der Matrix, Live-Updates

## 3. Zu-/Absage und geteilte Teilnahme

- [x] 3.1 RSVP-Fristen nach `internal/policy`, Matrix liefert `rsvp_locks_at`, `rsvp_require_reason`, `is_self`, `can_respond`, `locked`, eigene Gründe (+ Tests)
- [x] 3.2 `terminMatrix.ts`: Teilnahme getrennt nach Bisher/Geplant (+ Vitest)
- [x] 3.3 `TerminMatrix` + `TerminePage`: antippbare Zellen, Dialog mit den Listen-Funktionen (+ Vitest)

## 4. Abschluss

- [ ] 4.1 `make test` / `pnpm -C web build test lint` / `openspec validate` grün
