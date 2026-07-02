## Why

Der Team-Filter auf `/dienste` versteckt Dienste, statt sie zu filtern, und Dienste zeigen nicht, welches Team sie adressieren. Ursache: Anzeige und Filter hängen an `duty_slots.team_id`, das aber bei generischen Events und manuell angelegten Slots `NULL` ist — und bei Mehr-Team-Spielen kollabiert die Board-Gruppe auf das *erste* Team. Fachlich gehört ein Dienst jedoch zu den **Teams seines Termins** (`game_teams`, potenziell mehrere), nicht zu einem einzelnen `team_id`.

## What Changes

- Board-Gruppen von `GET /duty-board` tragen die **Teams ihres Termins** aus `game_teams` — einheitlich für `heim`/`auswärts`/`generisch` — als Menge (`team_ids: number[]` + `team_names: string[]`).
- **BREAKING (interne API):** Die Felder `team_id: number` und `team_name: string` je Board-Gruppe werden durch `team_ids`/`team_names` ersetzt.
- Game-lose Handslots (kein `game_id`) behalten ihr `ds.team_id` als einelementige Menge (`[team_id]` bzw. leer).
- Frontend `/dienste`: Team-Filter matcht, wenn das gewählte Team **in** der Team-Menge der Gruppe liegt (`includes`), statt strikter Gleichheit; der Gruppen-Header zeigt **alle** adressierten Teams.

## Capabilities

### New Capabilities
- `duty-board-team-filter`: Ein Dienst wird über die Teams seines Termins (`game_teams`) adressiert; `GET /duty-board` liefert diese Team-Menge je Gruppe, das Board zeigt alle adressierten Teams an und filtert per Zugehörigkeit.

### Modified Capabilities
<!-- keine bestehende Requirement-Änderung; team_id-Feldkontrakt war nie spezifiziert -->

## Impact

- **Backend:** `internal/duties/handler.go` (`Board` — Query/Struct `boardGroup`), bestehende Board-Tests (`internal/duties/handler_test.go`, u.a. der `team_id`-Kontrakttest).
- **Frontend:** `web/src/pages/DutyPage.tsx` (Interface `BoardGroup`, `visibleGroups`-Filter, Gruppen-Header).
- **Datenquelle:** `game_teams` (bestehend, keine Migration).
- Keine neuen Abhängigkeiten, keine DB-Migration, kein RAM-relevanter Zusatzaufwand.
