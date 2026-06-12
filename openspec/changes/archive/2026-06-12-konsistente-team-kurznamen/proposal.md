## Why

Team-Namen werden inkonsistent angezeigt: Admin sieht überall berechnete Kurznamen (mA2, mB1 etc.), Spieler/Elternteile sehen für Teams außerhalb ihres Kaders den rohen DB-Namen (z.B. "A-Jugend männlich 2"). Ursache ist dass `GET /api/teams` rollenabhängig gefiltert wird, `buildTeamShortNames` aber alle Teams kennen muss um eine vollständige Map aufzubauen.

## What Changes

- **Neuer Backend-Endpoint** `GET /api/teams/names` gibt alle aktiven Teams als `{id, age_class, gender, team_number, group_count}` zurück — für alle eingeloggten User, keine sensiblen Daten
- **KalenderPage** lädt `/api/teams/names` für den shortNames-Aufbau; das role-gefilterte `/api/teams` bleibt für den Filter-Dropdown erhalten
- **buildTeamDisplayNames** wird in allen Verwendungsstellen durch `buildTeamShortNames` ersetzt: `KalenderPage`, `GameEditModal`, `AdminTrainingsPage`, `TerminePage`, `ChatPage`
- **buildTeamDisplayNames** wird aus `teamName.ts` entfernt (kein Abnehmer mehr)

## Capabilities

### New Capabilities

- `team-names-endpoint`: Leichtgewichtiger Endpoint der alle aktiven Team-Metadaten für die clientseitige Namenberechnung liefert, unabhängig von der Rolle des aufrufenden Users

### Modified Capabilities

<!-- keine Spec-Level-Verhaltensänderungen an bestehenden Capabilities -->

## Impact

**Backend:**
- `internal/games/handler.go` — neuer Handler `ListTeamNames`
- `cmd/teamwerk/main.go` — neue Route `GET /api/teams/names` im authenticated-Block

**Frontend:**
- `web/src/lib/teamName.ts` — `buildTeamDisplayNames` entfernen
- `web/src/pages/KalenderPage.tsx` — zweiter Teams-Load für shortNames-Basis; `displayNames` entfernen
- `web/src/components/GameEditModal.tsx` — `buildTeamDisplayNames` → `buildTeamShortNames`
- `web/src/pages/AdminTrainingsPage.tsx` — `buildTeamDisplayNames` → `buildTeamShortNames`
- `web/src/pages/TerminePage.tsx` — `buildTeamDisplayNames` → `buildTeamShortNames`
- `web/src/pages/ChatPage.tsx` — `buildTeamDisplayNames` → `buildTeamShortNames`
