## Why

`GET /api/mitfahrgelegenheiten` liefert aktuell alle zukünftigen Spiele an alle eingeloggten Nutzer — ohne Team-Zugehörigkeit zu prüfen. Damit sehen Elternteile und Spieler Spiele fremder Mannschaften, und der "Alle"-Toggle auf der Seite ist irreführend, da er nicht "mein Team" bedeutet.

## What Changes

- **Backend**: `GET /api/mitfahrgelegenheiten` filtert Spiele nach Team-Zugehörigkeit des Nutzers (analog zur Dashboard-Logik). Admins und Vorstände sehen weiterhin alle Spiele.
- **Frontend**: Der Toggle "Alle" / "Meine" wird zu **"Team" / "Meine"** umbenannt. "Team" zeigt alle Spiele der eigenen Mannschaft(en); "Meine" zeigt nur Spiele, bei denen der Nutzer selbst einen Eintrag (biete/suche) oder eine Paarung hat.

## Capabilities

### New Capabilities

- `carpooling-team-filter`: Rollen-abhängige Filterung der Mitfahrgelegenheiten-Liste nach Team-Zugehörigkeit; Toggle-Umbenennung auf "Team"/"Meine"

### Modified Capabilities

<!-- keine bestehenden Specs ändern sich inhaltlich -->

## Impact

- **Backend**: `internal/carpooling/handler.go` — `List()`-Funktion erhält Team-Subquery analog zu `internal/dashboard/handler.go:teamQueryForUser()`
- **Frontend**: `web/src/pages/MitfahrgelegenheitenPage.tsx` — Button-Label, kein Logik-Change
- **Keine DB-Migration nötig** — bestehende Tabellen reichen aus
- **Kein Breaking Change** für andere Endpunkte
