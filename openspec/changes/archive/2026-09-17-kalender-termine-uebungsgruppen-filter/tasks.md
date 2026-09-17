## 1. Backend: Sichtbarkeits-Endpoint für Übungsgruppen

- [x] 1.1 `internal/practicegroups/handler.go`: neuen Handler `ListMine` (`GET /api/practice-groups/my`) ergänzen — liefert `[{id, name}]` der Übungsgruppen (`kader.kind='practice'`) der aktiven Saison, sortiert nach Name.
- [x] 1.2 Sichtbarkeitslogik in `ListMine` spiegeln (analog `ListTeamsForUser` in `internal/games/handler.go`, aber auf `kader` statt `teams`/`user_accessible_teams`): admin/vorstand/sportliche_leitung sehen alle; Trainer via `kader_trainers`; Spieler/Eltern via `kader_members`/`kader_extended_members`/`family_links`.
- [x] 1.3 Ohne aktive Saison: leeres Array zurückgeben (kein Fehler, kein 404).
- [x] 1.4 Route in `internal/app/router.go` im Authenticated-Tier registrieren (`r.Get("/api/practice-groups/my", h.PracticeGroups.ListMine)`) — bewusst nicht im Vorstand/Trainer/sportliche_leitung-Tier der bestehenden `/api/practice-groups`-Routen.
- [x] 1.5 Go-Tests in `internal/practicegroups/handler_test.go`: Happy-Path pro Rolle (admin sieht alle, Trainer sieht eigene, Spieler sieht eigene, Elternteil sieht Kind-Gruppe, fremder Spieler sieht nichts), 401 ohne Token, leeres Array ohne aktive Saison.

## 2. Frontend: gemeinsame Filter-Logik

- [x] 2.1 `web/src/lib/teamFilter.ts`: Hilfsfunktion `trainingFilterId({ team_id, kader_id }): number` ergänzen (`team_id > 0 ? team_id : -kader_id`), mit kurzem Kommentar zur negativen-ID-Konvention.
- [x] 2.2 `web/src/lib/teamFilter.test.ts`: Tests für `trainingFilterId` (Mannschafts-Training, Übungsgruppen-Training, verschiedene Übungsgruppen liefern verschiedene IDs).

## 3. Frontend: KalenderPage

- [x] 3.1 `KalenderPage.tsx`: `api.get('/practice-groups/my')` beim Laden abrufen (analog zum bestehenden `/teams`-Fetch), Ergebnis in State halten.
- [x] 3.2 `teamFilterOptions` um die Übungsgruppen-Einträge erweitern (`{ id: -pg.id, label: pg.name }`), zusätzlich zu `buildTeamOptions(teams...)`.
- [x] 3.3 `Training`-Interface um `kader_id: number` ergänzen; Trainings-Filterung (`matchesTeamFilter(filterTeamIds, [...])`) auf `trainingFilterId(t)` umstellen.
- [x] 3.4 Manuell/per Test prüfen, dass Spiele von einer Übungsgruppen-Auswahl unberührt bleiben (weiterhin nur `g.teams.map(t => t.id)`).

## 4. Frontend: TerminePage

- [x] 4.1 `TerminePage.tsx`: `api.get('/practice-groups/my')` beim Laden abrufen, Ergebnis in State halten.
- [x] 4.2 `teamOptions` um die Übungsgruppen-Einträge erweitern (gleiche Kodierung wie 3.2).
- [x] 4.3 `Session`-Interface um `kader_id: number` ergänzen; Trainings-Zweig der Filterung auf `trainingFilterId(t.data)` umstellen.
- [x] 4.4 `updateFilter`/`serializeTeamIds`-Aufrufe prüfen: `teams.length` als `optionCount` muss weiterhin die Gesamtzahl aller wählbaren Optionen (Teams + Übungsgruppen) sein, nicht nur `teams.length` — ggf. auf `teamOptions.length` umstellen, falls das noch nicht der Fall ist.

## 5. Frontend-Tests

- [x] 5.1 `KalenderPage.teamfilter.test.tsx`: Szenario ergänzen, in dem eine Übungsgruppe im Dropdown erscheint und das Filtern auf sie nur deren Trainings zeigt (Spiele und andere Trainings verschwinden).
- [x] 5.2 `TerminePage.teamfilter.test.tsx`: analoges Szenario für `/termine`, inklusive Kombination Mannschaft + Übungsgruppe (beide Ergebnismengen vereinigt sichtbar).

## 6. Verifikation

- [x] 6.1 `/verify-change` bzw. `make test`/`make lint` (Go + Vitest) grün.
- [x] 6.2 `openspec validate kalender-termine-uebungsgruppen-filter --strict` grün.
