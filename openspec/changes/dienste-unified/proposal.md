## Why

Die Dienstdomäne ist auf drei Seiten aufgeteilt (`/dienstboerse`, `/dienste`, `/dienstkonten`), die dieselbe Domäne fragmentieren — Organisatoren müssen zwischen Seiten wechseln um Dienste zu sehen und zu verwalten. Zusätzlich filtert die Dienstbörse über `team_memberships`, eine Tabelle die nie per Frontend befüllt wird, sodass die Börse für alle Nutzer eine leere Liste zeigt.

## What Changes

- **BREAKING** `GET /api/duty-board` — Filterlogik von `team_memberships` auf `kader_members` / `kader_trainers` umgestellt; Antwortstruktur um `claimed_by_me` pro Slot erweitert (war bereits vorhanden)
- **BREAKING** `POST /api/members/{id}/team-assignment` — Endpoint und Handler entfernt; war nie per Frontend aufgerufen
- Neue Route `GET /api/duty-board?view=mine` — filtert auf Slots mit eigener Zuteilung (für Meine/Alle-Toggle)
- Frontend `/dienstboerse` — Route und `DutyBoardPage` entfernt (kein Redirect)
- Frontend `/dienste` → `DutySlotsPage` — ersetzt durch neue `DutyPage` (vereinheitlicht)
- Neue vereinheitlichte `DutyPage` auf `/dienste` mit:
  - Für alle Rollen: Eintragen/Austragen, Vergangene ein-/ausblenden
  - Für Admin + Trainer: Toggle „Meine" (eigene Zuteilungen) / „Alle" (verwaltete Teams), Erfüllt, Geldersatz, Löschen mit Bestätigung
- AppShell-Navigation: „Dienstbörse" und „Dienst-Planung" entfallen, neuer Eintrag „Dienste" → `/dienste`
- `team_memberships`-Tabelle bleibt in der DB erhalten, wird aber nicht mehr beschrieben oder gelesen

## Capabilities

### New Capabilities

- `dienste-unified`: Vereinheitlichte Dienstseite — alle Rollen sehen und verwalten Dienste an einem Ort mit rollenabhängigen Aktionen und korrekter Kader-basierter Filterung

### Modified Capabilities

<!-- Keine bestehenden Specs betroffen -->

## Impact

- **Backend:** `internal/duties/handler.go` — `Board`-Query, neuer `view`-Parameter; `AssignTeam`-Handler entfernen
- **Backend:** `cmd/teamwerk/main.go` — Route `POST /api/members/{id}/team-assignment` entfernen
- **Frontend:** `web/src/pages/DutyBoardPage.tsx` — entfernen
- **Frontend:** `web/src/pages/DutySlotsPage.tsx` — entfernen
- **Frontend:** `web/src/pages/DutyPage.tsx` — neu anlegen
- **Frontend:** `web/src/App.tsx` — Routen `/dienstboerse` entfernen, `/dienste` auf neue DutyPage umbiegen
- **Frontend:** `web/src/components/AppShell.tsx` — Nav-Einträge anpassen
- **Keine Migration** erforderlich — `team_memberships` bleibt als leere Tabelle bestehen
