## Why

Elternteile können Mitfahrgelegenheiten weder im Namen ihrer Kinder eintragen noch bestehende Kind-Einträge verwalten oder bestätigte Paarungen ihrer Kinder im Dashboard sehen. Das widerspricht dem etablierten Proxy-Account-Pattern, das bei Spielzusagen und Trainings bereits vollständig umgesetzt ist.

## What Changes

- `POST /api/mitfahrgelegenheiten` akzeptiert optionalen `forUserId`-Parameter; Elternteil kann Eintrag für Kind anlegen
- `DELETE /api/mitfahrgelegenheiten/{id}` erlaubt Elternteil das Löschen von Kind-Einträgen
- `POST /api/mitfahrt-paarungen` erlaubt Elternteil das Stellen einer Paarungsanfrage für Kind-Einträge
- `POST /api/mitfahrt-paarungen/{id}/confirm` und `/reject` erlauben Elternteil das Bestätigen/Ablehnen im Namen des Kindes
- `GET /api/mitfahrgelegenheiten` gibt `childUserIds` zurück und setzt `isOwn`/`bieteIsOwn`/`sucheIsOwn` auch für Kind-Einträge
- `GET /api/dashboard` liefert in `carpoolingConfirmed` auch bestätigte Paarungen der Kinder
- Frontend `FormModal` erhält „Für wen?"-Dropdown (nur sichtbar für Elternteile mit ≥1 Kind)

Die Logik gilt unabhängig davon ob das Kind `can_login = 0` oder `can_login = 1` hat — `family_links` ist die einzige Berechtigungsquelle.

## Capabilities

### New Capabilities

- `carpooling-elternzugang`: Elternzugang für Mitfahrgelegenheiten — lesen, anlegen, löschen und Paarungen verwalten im Namen von Kindern

### Modified Capabilities

- `mitfahrgelegenheiten-board`: `isOwn`-Semantik erweitert um Kind-Einträge; `ListResponse` erhält `childUserIds`; `forUserId` in Upsert
- `mitfahrt-paarungen`: RequestPairing, ConfirmPairing, RejectPairing erlauben Kind-Stellvertretung
- `dashboard-carpooling-hint`: `carpoolingConfirmed` schließt Kind-Paarungen ein

## Impact

- `internal/carpooling/handler.go` — alle 5 Endpoints + 2 Query-Hilfsfunktionen
- `internal/dashboard/handler.go` — `queryCarpoolingConfirmed()`
- `web/src/pages/MitfahrgelegenheitenPage.tsx` — FormModal, Upsert-Call
- Kein Datenbankschema-Change, keine neue Migration
