## Why

Teamnamen werden in TeamWERK aktuell in drei konkurrierenden Formaten dargestellt — Kurzform („mA1"), Langform („B-Jugend 2 männlich") und Rohname aus `teams.name` („TS Stuttgart B-Jugend männlich") — und Multi-Team-Spiele zeigen je nach Ort mal alle Teams, mal nur den String „Mehrere", mal (Dashboard) nur das erste Team aufgrund eines `MIN()`-Bugs. Die Inkonsistenz ist besonders auffällig zwischen Termine-Seite (Rohname für Spiele, Langform für Trainings) und Kalender (Kurzform), und macht Spielgemeinschaften/Doppelheimspiele schwer lesbar.

## What Changes

- **Backend wird Single Source of Truth** für Display-Strings: neuer SQL-Helper `TeamDisplayShort(alias)` parallel zum bestehenden `TeamDisplayName(alias)`.
- **Alle Listen-Endpoints** für Spiele/Trainings/Dienste/Dashboard liefern künftig sowohl `team_display_short` als auch `team_display_long` (komma-getrennt bei mehreren Teams) plus `team_ids[]`.
- **Frontend-Helper** `formatTeamList(teams, mode)` zentralisiert die Render-Entscheidung; alle Seiten verwenden ihn.
- **„Mehrere"-String überlebt ausschließlich im Kalender-Tile** (Inline-Label und Tooltip) als bewusste Ausnahme wegen Platzmangels — überall sonst werden tatsächliche Teamnamen aufgelistet.
- **DashboardPage-Bug behoben**: `MIN(...)` in `dashboard/handler.go:176` wird durch `GROUP_CONCAT(...)` ersetzt — bisher verschwindet bei Doppelheimspielen das zweite Team.
- **SpieltagDetailPage-Bug behoben**: aktuell wird `team_name` referenziert, das in der API-Response gar nicht existiert (rendert leer); Frontend liest künftig `team_display_long` aus dem `teams[]`-Array.
- **MeinTeamPage**: Rohname → Langform.
- **DutyPage**: Langform → Kurzform.

## Capabilities

### New Capabilities

- `team-name-display`: Einheitliche Regeln für die Darstellung von Teamnamen in API-Responses und im Frontend

### Modified Capabilities

*(keine bestehenden Capabilities ändern ihre Verträge — die Endpoints erhalten zusätzliche Felder, vorhandene Felder bleiben rückwärtskompatibel; `team_names` und Rohnamen bleiben erhalten und werden parallel gepflegt, bis das Frontend umgestellt ist)*

## Impact

**Backend**
- Neu: `internal/db/team_display_short.go`
- Geändert: `internal/games/handler.go` (ListGames, GetGame, ListMyGames), `internal/duties/handler.go` (DutyBoard), `internal/dashboard/handler.go` (UpcomingEvents)

**Frontend**
- Erweitert: `web/src/lib/teamName.ts` (neuer `formatTeamList`-Helper)
- Geändert: `KalenderPage.tsx`, `TerminePage.tsx`, `SpieltagDetailPage.tsx`, `TermineDetailPage.tsx`, `DutyPage.tsx`, `DashboardPage.tsx`, `MeinTeamPage.tsx`, `EventInfoModal.tsx`, `AdminTrainingsPage.tsx`

**Migrationen**
- Keine.

**API-Verträge**
- Additiv: neue Felder `team_display_short`, `team_display_short_csv`, `team_display_long`, `team_display_long_csv` in den betroffenen Responses. Bestehende Felder (`team_names`, `team_name`, `teams[]`) bleiben erhalten.

## Test-Anforderungen

- **`TeamDisplayShort` Unit-Test**: deckt Single-Team und Multi-Team derselben age_class+gender ab, alle drei Gender-Werte (`m`, `f`, `mixed`), unbekannte age_class
- **`formatTeamList` Snapshot-Test**: prüft alle drei Modi (`short`, `long`, `kalender`) inkl. Single-/Multi-Team und Kalender-Sonderfall („Mehrere")
- **`GET /api/games` (ListGames)**: Happy-Path mit Doppelheimspiel (2 Teams) → prüft `team_display_short_csv` enthält beide Kurznamen und `team_display_long_csv` beide Langnamen
- **`GET /api/games/my` (ListMyGames)**: Happy-Path mit Doppelheimspiel → prüft beide Display-Felder
- **`GET /api/dashboard`**: Doppelheimspiel im Time-Window → prüft, dass beide Teams aufgelistet sind (Regression-Test für `MIN`→`GROUP_CONCAT`-Fix)
- **`GET /api/duty-board`**: Slot mit team_id → prüft `team_display_short` ist gesetzt
- **Invariante**: Server-Kurzform und Frontend-`buildTeamShortNames` produzieren für dieselben Eingabedaten denselben String (Parity-Test, der beide Quellen vergleicht)
