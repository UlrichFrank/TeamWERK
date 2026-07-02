## Why

Im Team-Gruppen-Picker des Gruppenchats werden Teams mit der falschen Kurzform angezeigt: Ein Team `mB2` erscheint als `mB`, sobald der aufrufende Nutzer nur Zugriff auf **eines** der Teams seiner Altersklasse+Geschlecht-Gruppe hat. Überall sonst in der App (Kalender, Termine, Dienstbörse, Broadcast-Fenster) wird korrekt `mB2` angezeigt. Der Chat-Picker ist die einzige Ausnahme und widerspricht damit der kanonischen Standard-Kurzform der Teams.

## What Changes

- Der Endpoint `GET /api/chat/team-groups` liefert pro Eintrag ein neues Feld `displayShort` mit der **kanonischen, saisonweit** berechneten Team-Kurzform (über den geteilten SQL-Helper `db.TeamDisplayShort`). Die Team-Nummer wird angehängt, wenn in der aktiven Saison mehrere Teams derselben Altersklasse+Geschlecht existieren — **unabhängig davon, welche Teams der Caller sehen darf**.
- Das bisher unbenutzte, nie befüllte Feld `teamName` sowie das nur clientseitig verwendete `groupCount` entfallen aus der Response.
- Das Frontend (`web/src/pages/ChatPage.tsx`) übernimmt `displayShort` direkt, statt die Kurzform selbst aus `groupCount` neu zu berechnen (`buildTeamShortNames`). Damit verschwindet die divergierende Zweitberechnung, die den Fehler verursacht hat.
- **Unverändert (bewusst caller-scoped):** das `count`-Feld (Mitgliederzahl ohne Caller) und die Sichtbarkeitsregel des Endpoints. Nur die Disambiguierung des Anzeige-Labels wird saisonweit.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `chat-team-groups`: Der Listen-Endpoint liefert die kanonische Team-Kurzform als `displayShort` (saisonweite Disambiguierung) statt roher Felder, aus denen der Client die Kurzform caller-scoped neu berechnet.

## Impact

- **Backend:** `internal/chat/team_groups.go` (`TeamGroup`-Struct, `ListTeamGroups`-Query, neuer Import `internal/db`), Tests in `internal/chat/team_groups_test.go`.
- **Frontend:** `web/src/pages/ChatPage.tsx` (`TeamGroup`-Interface, `teamGroupShortNames`, Picker-Anzeige/-Filter). `buildTeamShortNames` in `web/src/lib/teamName.ts` bleibt (weiter vom Broadcast-Fenster genutzt).
- **API:** additiv+entfernend an einer read-only Route; kein Schema-, kein Migrations-, kein Broadcast-Eingriff.
