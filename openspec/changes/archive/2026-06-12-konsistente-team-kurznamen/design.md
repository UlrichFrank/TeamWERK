## Context

Team-Namen werden auf zwei Arten berechnet: `buildTeamShortNames` (z.B. "mA2") und `buildTeamDisplayNames` (z.B. "A-Jugend männlich 2"). Beide Funktionen in `teamName.ts` berechnen Namen aus `{age_class, gender, team_number, group_count}` — diese Felder stammen aus der DB, nicht aus dem `name`-Feld der `teams`-Tabelle.

Das Problem: `GET /api/teams` (→ `ListTeamsForUser`) filtert rollenabhängig — Spieler und Elternteile sehen nur ihre eigenen Kader-Teams. Folge: die `shortNames`-Map ist für diese Nutzer unvollständig. Bei Games mit mehreren Teams (z.B. generische Events) greift der Fallback auf `t.name` (DB-Rohname wie "A-Jugend männlich 2").

## Goals / Non-Goals

**Goals:**
- Alle eingeloggten User sehen für alle Teams konsistent den berechneten Kurznamen
- Keine sensiblen Daten im neuen Endpoint (nur Metadaten für Namensberechnung)
- `GET /api/teams` bleibt unverändert (rollenabhängig, für Filter-Dropdown)

**Non-Goals:**
- DB-Migrationen (kein Anfassen des `name`-Feldes in `teams`)
- Änderung der Rollenlogik für andere Endpoints
- Neue Langname-Stellen einführen

## Decisions

**1. Neuer Endpoint statt Änderung an `ListTeamsForUser`**

`GET /api/teams` hat eine klar definierte Semantik (Kader-View je Rolle). Diese zu ändern würde andere Konsumenten brechen. Ein separater `GET /api/teams/names` ist semantisch klar und minimal.

Response-Shape: `[{id, age_class, gender, team_number, group_count}]` — kein `name`-Feld nötig, da das Frontend die Namen selbst berechnet.

**2. Endpoint im `games`-Handler**

`ListTeamsForUser` sitzt bereits im `games`-Handler (der die `kader`-Joins für `team_number`/`group_count` kennt). Der neue Handler `ListTeamNames` folgt demselben Query-Muster ohne Rollenfilter.

**3. buildTeamDisplayNames entfernen**

Kein aktiver Abnehmer mehr nach dieser Änderung. Totes Code entfernen. Falls zukünftig Langnamen an einer Stelle benötigt werden, kann die Funktion neu eingeführt werden.

**4. KalenderPage: zwei separate Loads**

- `/api/teams` → `teams` (für Filter-Dropdown, role-gefiltert, bleibt wie bisher)
- `/api/teams/names` → `allTeamNames` (für `shortNames`-Map, immer vollständig)

`shortNames` wird aus `allTeamNames` berechnet. `displayNames` entfällt.

## Risks / Trade-offs

- [Minimaler Extra-Request] KalenderPage macht jetzt einen zusätzlichen API-Call → vernachlässigbar (kleine JSON-Antwort, einmalig beim Mount, parallel zu anderen Calls)
- [Andere Pages laden weiterhin role-gefiltertes `/api/teams`] Für GameEditModal, AdminTrainingsPage, TerminePage, ChatPage sind Spieler/Elternteile entweder kein Use-Case oder sehen dort nur eigene Teams — kein Bug, da diese Pages nur eigene Teams zeigen. Nur `buildTeamDisplayNames` → `buildTeamShortNames` ersetzen reicht dort.
