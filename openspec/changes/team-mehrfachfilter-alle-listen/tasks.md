## 1. Backend: `team_id` als ID-Liste

- [x] 1.1 `internal/absences/handler.go` — `parseTeamIDList` + `placeholders`; `Calendar` filtert über `IN (…)`, die beiden Query-Varianten (mit/ohne Filter) werden zu einer mit optionaler Zusatzbedingung
- [x] 1.2 Tests: Liste, Einzel-ID, unbrauchbarer Wert (`internal/absences/teamfilter_test.go`)

## 2. Gemeinsame Filter-Semantik

- [x] 2.1 `web/src/lib/teamFilter.ts` — `parseTeamIds`, `effectiveTeamIds`, `toggleTeamId`, `serializeTeamIds`, `matchesTeamFilter`, `buildTeamOptions`; `TeamFilterOption` zieht aus der Komponente hierher
- [x] 2.2 Unit-Tests der Semantik (`web/src/lib/teamFilter.test.ts`)
- [x] 2.3 `TerminePage.tsx` verhaltensgleich auf die lib umstellen (Bestandstests müssen unverändert grün bleiben)

## 3. Die drei übrigen Listen

- [x] 3.1 `DutyPage.tsx` — `team` als Menge, `TeamFilter` statt `<select>` (ab zwei Mannschaften), `matchesTeamFilter`, `otherFiltersActive`/`resetFilters`
- [x] 3.2 `DutyPage.tsx` — Fokus endet bei Team-/Typ-Filteränderung (wie TerminePage)
- [x] 3.3 `MitfahrgelegenheitenPage.tsx` — `team` als Menge, `TeamFilter`, clientseitiges Filterprädikat; `load()` ohne `team_id` und ohne Filter-Abhängigkeit
- [x] 3.4 `KalenderPage.tsx` — `filterTeamIds` als Set im State, `TeamFilter` (auch auf Mobile), `matchesTeamFilter` für Spiele und Trainings
- [x] 3.5 `KalenderPage.tsx` — `loadAbsences` schickt die ID-Liste; der Query-Wert ist zugleich die stabile Effekt-Abhängigkeit

## 4. Tests

- [x] 4.1 `DutyPage.teamfilter.test.tsx` — Liste, Einzel-ID, Abwählen, Deselektion aller, Fokus-Ende
- [x] 4.2 `MitfahrgelegenheitenPage.teamfilter.test.tsx` — Liste, Einzel-ID, Filtern ohne Neuladen, kein `team_id` in der Anfrage
- [x] 4.3 `KalenderPage.teamfilter.test.tsx` — Gitter folgt der Auswahl, Deselektion aller, Abwesenheiten mit ID-Liste

## 5. Abschluss

- [x] 5.1 `make test` + `golangci-lint` + `pnpm -C web build/test/lint` grün
- [x] 5.2 `openspec validate team-mehrfachfilter-alle-listen --strict`
