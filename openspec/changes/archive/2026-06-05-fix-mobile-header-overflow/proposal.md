## Why

Auf mobilen Geräten (< 640px) werden manche Seiten breiter als der Viewport dargestellt und lassen sich nicht horizontal scrollen — Inhalte am rechten Rand sind schlicht abgeschnitten. Zwei Ursachen: (1) Der Flex-Content-Container in AppShell fehlt `min-w-0`, sodass breiter Inhalt den Container aufzieht; (2) Page-Header auf mehreren Seiten stehen in einer starren `flex justify-between`-Zeile ohne Zeilenumbruch auf Mobile.

## What Changes

- **AppShell**: `min-w-0` am Flex-Content-Container ergänzen, damit der `overflow-auto` des `<main>`-Elements greift und horizontales Scrollen möglich ist
- **AdminUsersPage**: Header-Zeile (`h1` + Suchfeld + Button) erhält `flex-col sm:flex-row`-Stapelung; Tabellen-Container erhält `overflow-x-auto`
- **AdminDutyTypesPage**: Header-Zeile (`h1` + „+ Neu"-Button) auf responsive Stapelung umstellen
- **AdminDutyTemplatesPage**: Header-Zeile (`h1` + „+ Neue Vorlage"-Button) auf responsive Stapelung umstellen
- **KalenderPage**: Header-Zeile (`h1` + „Event anlegen"-Button) auf responsive Stapelung umstellen

Kein neues Verhalten, keine neuen Komponenten, keine Backend-Änderungen.

## Capabilities

### New Capabilities

*(keine — reine Darstellungskorrektur)*

### Modified Capabilities

*(keine Spec-level-Änderungen — die Funktionalität bleibt identisch)*

## Impact

- `web/src/components/AppShell.tsx` — eine Tailwind-Klasse ergänzt
- `web/src/pages/AdminUsersPage.tsx` — Header-Klassen + Tabellen-Wrapper
- `web/src/pages/AdminDutyTypesPage.tsx` — Header-Klassen
- `web/src/pages/AdminDutyTemplatesPage.tsx` — Header-Klassen
- `web/src/pages/KalenderPage.tsx` — Header-Klassen
- Kein Backend
