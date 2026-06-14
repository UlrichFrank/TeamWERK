## Why

Beim kontextuellen Navigieren innerhalb der App (z.B. Dashboard → Dienste, oder Sidebar → Termine → Sidebar → Mitfahrten) fehlt ein konsistentes „Zurück". Nutzer müssen den Browser-Zurück-Button oder die Sidebar bemühen — auf Mobile besonders umständlich, da die Sidebar hinter einem Hamburger-Menü versteckt ist.

## What Changes

- `AppShell` erhält einen globalen `← Zurück`-Button, der oberhalb des `<Outlet>`-Inhalts erscheint (Desktop) bzw. im Mobile-Header (zwischen Hamburger und Titel) — sichtbar sobald React Routers interner History-Index `> 0`
- Klick ruft `navigate(-1)` auf — kein eigenes Tracking, kein Location-State nötig
- Bestehende per-Page-Zurück-Buttons werden entfernt: `TermineDetailPage`, `MeinTeamPage`, `SpieltagDetailPage`, `MembersPage`

## Capabilities

### New Capabilities

- `global-back-navigation`: Globaler, generischer Zurück-Button in AppShell — history-basiert, ohne Page-spezifischen Code

### Modified Capabilities

_(keine — nur Entfernen redundanter lokaler Buttons)_

## Impact

- `web/src/components/AppShell.tsx` — Button-Logik und Rendering
- `web/src/pages/TermineDetailPage.tsx` — lokale Zurück-Buttons entfernen
- `web/src/pages/MeinTeamPage.tsx` — lokaler Zurück-Button entfernen
- `web/src/pages/SpieltagDetailPage.tsx` — lokaler Zurück-Link entfernen
- `web/src/pages/MembersPage.tsx` — lokaler Zurück-Button entfernen
- Keine neuen Dependencies, keine API-Änderungen, keine Migrationen
