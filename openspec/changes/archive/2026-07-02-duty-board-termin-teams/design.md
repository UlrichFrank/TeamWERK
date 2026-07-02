## Context

`GET /duty-board` (`internal/duties/handler.go`, `Board`) baut Board-Gruppen in Go auf. Für game-basierte Gruppen (`game-{id}`) wird `boardGroup.TeamID`/`TeamName` aktuell aus der **ersten** gescannten Zeile gesetzt (`COALESCE(ds.team_id, 0)`, ~Zeile 530/597–609). Auto-Dienste für Spiele setzen `ds.team_id` pro `game_teams`-Eintrag (`internal/games/regen.go:342`); generische Events setzen `ds.team_id = NULL` (`internal/games/handler.go:903`). Die tatsächlichen Teams eines Termins stehen in `game_teams` — von Create- und Update-Pfad einheitlich für alle Event-Typen befüllt (`handler.go:874–877`, `1050–1056`).

Das Frontend (`web/src/pages/DutyPage.tsx`) filtert clientseitig mit `g.team_id !== filterTeamId` (Zeile 148) und zeigt `g.team_name` (Zeile 265). Dadurch: NULL-`team_id` → nie gematcht + kein Label; Mehr-Team-Spiel → nur erstes Team sichtbar/filterbar.

## Goals / Non-Goals

**Goals:**
- Board-Gruppen tragen die **Menge** der Termin-Teams (`team_ids`/`team_names`) aus `game_teams`.
- Team-Filter matcht per Zugehörigkeit; Header zeigt alle Teams.
- Game-lose Handslots behalten `[ds.team_id]`.

**Non-Goals:**
- Kein Umbau des serverseitigen Sichtbarkeits-Filters (Board bleibt für Nicht-Privilegierte auf eigene Teams gescoped; admin/vorstand/sL sehen alle — bestätigt „Filter tut").
- Kein Umbau des `/api/teams`-Dropdown-Scopings.
- Keine DB-Migration, kein Schema-Wechsel an `duty_slots`.

## Decisions

**1. Team-Menge nach dem Gruppieren per Zusatz-Query anhängen (statt SQL-Aggregation im Haupt-Query).**
Nach dem bestehenden Scan-Loop stehen die `game_id`s aller Gruppen fest. Eine zweite Query
`SELECT gt.game_id, gt.team_id, <TeamDisplayShort> FROM game_teams gt JOIN teams t ON t.id = gt.team_id WHERE gt.game_id IN (...) ORDER BY gt.game_id, t.age_class, t.gender` liefert die Teams je Spiel; das Ergebnis wird per `map[gameID][]team` an die Gruppen gehängt. Begründung: Der Haupt-Query ist bereits komplex; eine separate Query hält ihn lesbar und vermeidet Zeilen-Multiplikation (JOIN auf `game_teams` würde Slots × Teams vervielfachen). Analog zum bestehenden Assignee-Nachladen (`handler.go:649ff`) — gleiches Muster, gleiche RAM-Charakteristik (nur Ergebnis-IDs).

**2. `boardGroup`-Kontrakt: `TeamID *int`/`TeamName string` → `TeamIDs []int` + `TeamNames []string`.**
Positionsgleiche Arrays statt Objektliste — minimaler JSON-Diff, im Frontend trivial per Index korreliert. Kein `omitempty` auf `TeamIDs` (immer als Array serialisieren, ggf. leer), damit das Frontend `includes()` ohne Null-Guard nutzen kann.

**3. Game-lose Gruppen: `TeamIDs = [team_id]` falls `>0`, sonst `[]`.**
Die vorhandene `teamID > 0`-Logik bleibt, füllt aber die Menge statt eines Skalars. Kurzname aus der bereits im Haupt-Query gejointen `teams`-Zeile.

**4. Frontend: `includes`-Filter + Multi-Team-Header.**
`BoardGroup.team_id`/`team_name` → `team_ids: number[]`/`team_names: string[]`. Filter: `filterTeamId !== null && !g.team_ids.includes(filterTeamId)` → ausblenden. Header (Zeile 265): `team_names` als komma-separierte Kurznamen (bei leer: nichts). Kurznamen kommen bereits serverseitig aus `TeamDisplayShort`; das clientseitige `buildTeamShortNames`/`teamShortNames` bleibt fürs Dropdown, wird für den Header nicht mehr gebraucht.

## Risks / Trade-offs

- **Interner API-Bruch** (`team_id`/`team_name` je Gruppe entfällt): Nur `DutyPage.tsx` konsumiert das Feld; Kalender-Dienste-Panel prüfen (`kalender-dienste-panel`). → Task deckt Konsumenten-Scan ab.
- **Bestehender Kontrakttest** (`handler_test.go:466`, erwartet numerisches `team_id`) muss auf `team_ids` umgestellt werden — bewusst Teil des Change, keine Abschwächung.
- **Zweite Query pro Board-Load**: vernachlässigbar (ein `IN (...)` über wenige `game_id`s), gleiches Muster wie Assignee-Nachladen.
- **Mehr-Team-Header-Länge**: bei vielen Teams könnte der Header lang werden — in der Praxis 1–2 Teams; Kurznamen halten es kompakt.
