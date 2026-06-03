## Why

Auf mobilen Displays (≤ 375px, z.B. iPhone 7) überlaufen Seitenheader horizontal, weil Titel und Controls in einer starren Flex-Zeile stehen. Die AdminUsersPage ist konkret gemeldet; dasselbe Muster tritt auf zwei weiteren Admin-Seiten auf.

## What Changes

- `AdminUsersPage`: Header-Zeile (`h1` + Suchfeld + Button) erhält `flex-col sm:flex-row`-Stapelung auf Mobile
- `AdminDutyTypesPage`: Header-Zeile (`h1` + „+ Neu"-Button) erhält `flex-col sm:flex-row`-Stapelung auf Mobile
- `AdminDutyTemplatesPage`: Header-Zeile (`h1` + „+ Neue Vorlage"-Button) erhält `flex-col sm:flex-row`-Stapelung auf Mobile

Kein neues Verhalten, keine neuen Komponenten, keine Backend-Änderungen.

## Capabilities

### New Capabilities

*(keine — reine Darstellungskorrektur)*

### Modified Capabilities

*(keine Spec-level-Änderungen — die Funktionalität bleibt identisch)*

## Impact

- `web/src/pages/AdminUsersPage.tsx` — Header-Klassen
- `web/src/pages/AdminDutyTypesPage.tsx` — Header-Klassen
- `web/src/pages/AdminDutyTemplatesPage.tsx` — Header-Klassen
