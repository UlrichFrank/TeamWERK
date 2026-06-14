## Why

Trainer und Sportliche Leiter können keine Dienst-Vorlagen lesen und keine Dienst-Typen abrufen, weil diese Lese-Endpunkte fälschlicherweise in der vorstand-only Gruppe gesperrt sind. Dadurch erscheint beim Anlegen von Kalender-Ereignissen "keine passende Vorlage" und auf der Termin-Detailseite ist das Dropdown für Dienst-Typen leer.

## What Changes

- `GET /api/duty-types` wird von der vorstand-only Gruppe in die vorstand+trainer+sportliche_leitung Gruppe verschoben (Lese-Zugriff für Trainer)
- `GET /api/duty-templates`, `GET /api/duty-templates/{id}` und `GET /api/duty-templates/{id}/preview` werden ebenfalls in die vorstand+trainer+sportliche_leitung Gruppe verschoben
- Schreib-Operationen (POST/PUT/DELETE) auf duty-types und duty-templates bleiben vorstand-only
- Frontend `SpieltagDetailPage.tsx`: `canEdit`-Check wird von `user.role`-Vergleich auf `hasFunction()`-Prüfung umgestellt, damit Nutzer mit `role=spieler` und `club_functions=["trainer"]` korrekt als berechtigt erkannt werden

## Capabilities

### New Capabilities

Keine neuen Capabilities — dies ist ausschließlich eine Berechtigungskorrektur.

### Modified Capabilities

- `duty-read-access`: Lese-Zugriff auf Dienst-Typen und Dienst-Templates wird auf Trainer und Sportliche Leiter ausgeweitet

## Impact

- `cmd/teamwerk/main.go`: Router-Gruppen umstrukturieren (4 GET-Routen verschieben)
- `web/src/pages/SpieltagDetailPage.tsx`: `canEdit`-Berechtigungslogik korrigieren
- Keine Datenbankmigrationen, keine neuen Endpunkte, kein Schema-Änderung
