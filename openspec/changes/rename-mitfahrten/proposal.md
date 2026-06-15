## Why

Im Code, in der DB und in der UI wird das Feature inkonsistent als „Mitfahrgelegenheiten", „Mitfahrten" und „Mitfahrt-Paarungen" bezeichnet. Im Navigations-Label heißt es bereits „Mitfahrten", während Routen, Tabellen und Page-Komponente noch das längere „Mitfahrgelegenheiten" tragen. Ein einheitlicher Name reduziert kognitive Last für künftige Entwicklung und vereinfacht Suche, Refactoring und Spec-Pflege.

## What Changes

- **BREAKING** API-Routen umbenannt: `/api/mitfahrgelegenheiten*` → `/api/mitfahrten*`
- **BREAKING** Frontend-Route umbenannt: `/mitfahrgelegenheiten` → `/mitfahrten`
- **BREAKING** DB-Tabelle umbenannt: `mitfahrgelegenheiten` → `mitfahrten` (Migration 043, inkl. Index- und FK-Anpassungen)
- Page-Komponente und Datei umbenannt: `MitfahrgelegenheitenPage.tsx` → `MitfahrtenPage.tsx`
- SSE-Event-Name umbenannt: `mitfahrgelegenheiten` → `mitfahrten`
- Push-Deep-Links und Notification-Titel angepasst: `/mitfahrgelegenheiten` → `/mitfahrten`, Titel „Mitfahrgelegenheit" → „Mitfahrt"
- Bestehende Specs umbenannt: `mitfahrgelegenheiten-board` → `mitfahrten-board`, `mitfahrgelegenheiten-team-filter` → `mitfahrten-team-filter`, `mitfahrgelegenheiten-nav` → `mitfahrten-nav`
- Doku (`CLAUDE.md`, `docs/anleitung-*.md`, `CHANGELOG.md`) entsprechend nachgezogen

Nicht im Scope: Go-Package-Name (`internal/carpooling/`) bleibt — englischer Domain-Begriff, keine fachliche Verwirrung.

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

- `mitfahrgelegenheiten-board`: Wird komplett zu `mitfahrten-board` umbenannt; alle Requirement-Referenzen auf „Mitfahrgelegenheiten" werden zu „Mitfahrten", Routen zu `/api/mitfahrten` und `/mitfahrten`.
- `mitfahrgelegenheiten-team-filter`: Wird komplett zu `mitfahrten-team-filter` umbenannt; Route-Referenzen und Bezeichner aktualisiert.
- `mitfahrgelegenheiten-nav`: Wird komplett zu `mitfahrten-nav` umbenannt; Frontend-Pfad-Referenzen aktualisiert.
- `mitfahrt-paarungen`: Tabellen-Referenz `mitfahrgelegenheiten` in der Spec ändert sich auf `mitfahrten` (Pairings-API-Pfade waren bereits korrekt benannt).
- `carpooling-notifications`: Push-Deep-Link und Event-Bezeichner werden auf neue Namen aktualisiert.
- `dashboard-migration`: Falls Referenzen auf Routen oder Event-Name vorhanden, diese ebenfalls anpassen.

## Impact

- **Backend:** `cmd/teamwerk/main.go` (Routen-Registrierung), `internal/carpooling/handler.go`, `internal/carpooling/paarungen_handler.go`, `internal/carpooling/handler_test.go`, `internal/dashboard/handler.go`, `internal/scheduler/scheduler.go` (SQL-Strings, SSE-Broadcast-Events, Push-Pfade).
- **DB:** Neue Migration `043_rename_mitfahrgelegenheiten.up.sql` + `.down.sql`. Tabelle wird per `ALTER TABLE ... RENAME TO` umbenannt (SQLite ≥ 3.25 unterstützt das mit automatischer FK-Anpassung; `mitfahrt_paarungen.biete_id`/`suche_id` referenzieren `mitfahrgelegenheiten(id)` und müssen ggf. via Tabellen-Neuanlage migriert werden).
- **Frontend:** `web/src/App.tsx` (Route + Import), `web/src/components/AppShell.tsx` (Nav-Link), `web/src/pages/MitfahrgelegenheitenPage.tsx` (Rename + API-Pfade), `web/src/pages/DashboardPage.tsx` (Links + Event-Name in `useLiveUpdates`).
- **Specs:** Drei `openspec/specs/`-Verzeichnisse werden umbenannt; archivierte Changes bleiben unverändert (historisch).
- **Push/SSE:** Bestehende Web-Push-Subscriptions sind nicht betroffen (URL wird beim Klick zur Server-Route navigiert — Route existiert nach Deploy).
- **Externe Konsumenten:** Keine. App ist intern, kein öffentliches API.
- **Breaking Change Risiko:** Während des Deploys kurzes Zeitfenster, in dem Frontend-Build und Backend-Build kohärent sein müssen. Da Deploy single-binary ist (`make deploy` baut Frontend embedded), ist das atomar.
