## Why

Auf der Spieltag-Detailseite (`/spielplan/:id`) gibt es keine Möglichkeit, ein Event zu löschen. Der Backend-Endpunkt `DELETE /api/admin/games/{id}` existiert bereits, ist aber im Frontend nicht angebunden.

## What Changes

- **Löschen-Button** auf `SpieltagDetailPage.tsx` für berechtigte Rollen (admin, vorstand, trainer)
- **Bestätigungs-Dialog** vor dem Löschen mit Hinweis auf mitgelöschte Dienste
- **Nach Löschen**: Redirect zur Spielplan-Übersicht (`/spielplan`)
- `isAdmin`-Check in der Detailseite auf alle berechtigten Rollen ausweiten (admin + vorstand + trainer)

## Capabilities

### New Capabilities

- `spieltag-delete`: Event löschen über die Detailseite mit Bestätigung und Redirect

### Modified Capabilities

## Impact

- **Frontend only**: `web/src/pages/SpieltagDetailPage.tsx`
- Kein Backend-Change nötig (`DELETE /api/admin/games/{id}` existiert)
- Kein DB-Schema-Change
