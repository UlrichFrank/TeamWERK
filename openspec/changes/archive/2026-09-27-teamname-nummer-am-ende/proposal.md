# Proposal: Teamname mit Nummer am Ende — ein Schema überall

## Why

Ein Jahrgang mit zwei Mannschaften derselben Kombination (z. B. zwei C-Jugenden männlich) erscheint heute in drei Schreibweisen, und das auf ein und derselben Seite: „Mein Team" zeigt eingeklappt den gespeicherten `teams.name` („C-Jugend männlich" für Mannschaft 1, „C-Jugend männlich 2" für Mannschaft 2), aufgeklappt den serverseitigen Langnamen „C-Jugend 1 männlich". Die erste Mannschaft trägt eingeklappt gar keine Nummer, obwohl es eine zweite gibt. Erwartet ist ein einziges Schema: **„C-Jugend männlich 1" / „C-Jugend männlich 2"**.

## What Changes

- Die serverseitige Langform (`display_long`, SQL-Helfer `appdb.TeamDisplayName`) stellt die Mannschaftsnummer ans **Ende**: `<Altersklasse> <Geschlecht> <Nummer>` („C-Jugend männlich 1"). Gibt es in der aktiven Saison nur eine Mannschaft dieser Kombination, entfällt die Nummer weiterhin („C-Jugend männlich"). Das betrifft alle Stellen, die `display_long`/`team_display_long_csv` ausliefern (Teams-Roster, Spiele, Trainings).
- `GET /api/teams/my` liefert zusätzlich `display_short`/`display_long` (heute nur `name`). „Mein Team" zeigt den Langnamen damit **schon im eingeklappten Zustand** — eingeklappt und aufgeklappt sind identisch.
- Client-seitige Namensbauer, die heute eigene Varianten erzeugen, nutzen einen gemeinsamen Helfer mit demselben Schema: Kader-Verwaltung (`AdminKaderPage`: Kartentitel und Lösch-Bestätigung) und `AutoAssignModal` (lässt die Nummer heute ganz weg, zwei Mannschaften sind dort nicht unterscheidbar). Der Kopier-Dialog (`CopyKaderModal`) arbeitet je Kombination und legt eine Mannschaft an; er zeigt deshalb eine Zeile je Kombination mit Hinweis statt zweier Zeilen mit geteiltem Häkchen.
- **Nicht** geändert: der gespeicherte `teams.name`. Er ist Identitätsschlüssel (`kader.ensureTeam` sucht über ihn, der H4A-Import liest die Nummer aus „… männlich 2") und Fallback, wenn ein Team in der aktiven Saison keinen Kader hat. Sein Schema („… männlich" / „… männlich 2") passt bereits zur neuen Reihenfolge.
- Kurzform (`display_short`, „mC1") bleibt unverändert.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `team-name-display`: Die Langform hat eine festgelegte Wortreihenfolge (Nummer am Ende), `GET /api/teams/my` liefert Display-Strings, und „Mein Team" rendert eingeklappt wie aufgeklappt denselben Langnamen; client-seitig gebaute Kader-Titel folgen demselben Schema.

## Impact

- **Backend:** `internal/db/team_display_name.go` (Reihenfolge), `internal/teams/handler.go` (`ListMyTeams` liefert Display-Felder). Betroffene Ausgaben: `GET /api/teams/{id}/roster`, `GET /api/games`, `GET /api/games/{id}`, Trainings-Liste (über `TeamDisplayName`). Keine Migration, keine Schema-Änderung, keine neue Route.
- **Frontend:** `web/src/lib/teamName.ts` (neuer Helfer für die Langform aus Kader-Feldern), `MeinTeamPage.tsx`, `AdminKaderPage.tsx`, `CopyKaderModal.tsx`, `AutoAssignModal.tsx`.
- **Tests:** Erwartungswerte in `internal/games/handler_test.go` (`team_display_long_csv`) und ein Kommentar in `internal/duties/handler_test.go` ändern sich.
- **Berechtigungen:** keine Änderung — `GET /api/teams/my` behält Tier und Filterlogik.
- **Keine Mutation** → kein neuer Broadcast nötig.
