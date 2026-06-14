## Why

Die Mitfahrgelegenheiten-Seite hat drei reproduzierbare Bugs und eine Darstellungslücke bei generischen Events mit mehreren Mannschaften. Die Bugs führen zu doppelten Datenbankeinträgen, falsch angezeigten Daten und verfälschten Formularfeldern — was das Feature für Nutzer in der Praxis unbrauchbar macht.

## What Changes

- **Bug 1 – `suche`-Duplikate:** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` macht immer ein blindes INSERT. Mehrfaches Öffnen und Speichern des Modals erzeugt beliebig viele Duplikate. → UPSERT-Logik wie bei `biete` + UNIQUE-Index via Migration
- **Bug 2 – Modal-State:** Wechsel zwischen „Ich biete Mitfahrt" / „Ich suche Mitfahrt" im FormModal setzt `treffpunkt` und `notiz` nicht zurück → Felder aus vorherigem Kontext werden mitgeschickt. → Felder bei Typ-Wechsel resetten
- **Bug 3 – Team-Filter fehlt:** `GET /mitfahrgelegenheiten` gibt alle Einträge aller zugänglichen Teams zurück. Nutzer in mehreren Teams sehen gemischte Daten. → Team-Dropdown auf der Page; Frontend schickt `?team_id=X`
- **Feature – generische Events einmalig:** Generische Events mit mehreren Teams erscheinen mehrfach in der Liste (je Team ein Row). → DISTINCT auf `game_id`, Team-Namen komma-separiert anzeigen

## Capabilities

### New Capabilities

- `mitfahrgelegenheiten-team-filter`: Team-Dropdown auf der Mitfahrgelegenheiten-Seite filtert die angezeigten Events und Einträge per API

### Modified Capabilities

- `mitfahrgelegenheiten-board`: UPSERT für `suche`, Modal-State-Reset, Deduplizierung generischer Events

## Impact

- **DB:** Migration — UNIQUE INDEX auf `(game_id, user_id) WHERE typ='suche'`; ggf. bestehende Duplikate bereinigen
- **Backend:** `internal/carpooling/handler.go` — `Upsert()`-Handler; `List()`-Query für generische Events
- **Frontend:** `web/src/pages/MitfahrgelegenheitenPage.tsx` — FormModal-State, Team-Dropdown
- **Keine neuen Abhängigkeiten**
