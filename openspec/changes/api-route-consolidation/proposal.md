## Why

Der `/admin`-Präfix in API-Routen vermittelt falsches Sicherheitsgefühl: Die eigentliche Zugriffskontrolle liegt ausschließlich in Chi-Middleware-Gruppen, nicht im Pfad. Historisch gewachsen führt das zu inkonsistenten Ressource-Pfaden (`/api/kalender` vs. `/api/admin/kalender` für dieselbe Ressource je nach HTTP-Verb) und erschwert das Verständnis des Routings.

## What Changes

- **BREAKING**: Alle `/api/admin/*`-Routen werden auf ihre kanonischen Ressource-Pfade umbenannt (z.B. `/api/admin/club` → `/api/club`)
- **BREAKING**: `GET /api/admin/teams` und `GET /api/teams` werden zu einem einzigen Endpoint zusammengeführt; die Antwort wird rollenabhängig gefiltert (vorstand/admin → alle Teams inkl. inaktiv; andere → Kader-gefilterter View)
- Frontend-Calls in allen Pages und `lib/api.ts`-Nutzungen werden auf die neuen Pfade aktualisiert
- CLAUDE.md API-Übersicht wird aktualisiert
- Keine Änderungen an Middleware-Gruppen, Berechtigungslogik oder Datenbankschema

## Capabilities

### New Capabilities

_Keine — dies ist eine reine Refaktorierung ohne neue Features._

### Modified Capabilities

_Keine Spec-Level-Änderungen. Das Verhalten aus Nutzersicht bleibt identisch; nur die URL-Struktur ändert sich._

## Impact

- **Backend**: `cmd/teamwerk/main.go` — alle Route-Registrierungen; `internal/games/handler.go` — `ListTeamsForUser` wird um Vorstand-Check erweitert (oder `ListTeams` aus config übernimmt die Logik)
- **Frontend**: alle `web/src/pages/*.tsx` die `api.get/post/put/delete('/admin/...')` aufrufen; betrifft ca. 15–20 Seiten
- **Kein Breaking Change für Endnutzer**: Da Frontend und Backend gleichzeitig deployed werden (Hard-Cut, keine Redirects)
- **CLAUDE.md**: API-Routen-Übersicht
