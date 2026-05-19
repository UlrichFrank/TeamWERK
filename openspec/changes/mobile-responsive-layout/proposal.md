## Why

TeamWERK ist aktuell ausschließlich für Desktop-Nutzung ausgelegt. Die Seitenleiste nimmt auf Mobilgeräten (~375–430px) über 50% des Bildschirms ein, sämtliche Tabellen und Formulare besitzen keine Responsive-Breakpoints, und die App ist nicht installierbar. Damit die Plattform auch unterwegs vollständig nutzbar ist – für Trainer, Spieler und Eltern – und sich auf dem Homescreen wie eine native App verhält, muss das gesamte Frontend mobile-first umgebaut und als PWA ausgeliefert werden.

## What Changes

- **AppShell**: Sidebar wird auf Mobilgeräten (<640px) versteckt; stattdessen Hamburger-Button (☰) mit Overlay-Drawer
- **Tabellen → Mobile Cards**: Alle 6 tabellenbasierten Seiten erhalten auf Mobile ein Card-Layout mit ⋮-Aktionen-Dropdown; Inline-Edit-Formulare öffnen ein Modal
- **Serverseitige Paginierung**: `GET /api/members` und `GET /api/admin/users` werden paginiert (`limit`/`offset`/`search`) — Response-Format `{ items, total }`. Frontend: „Mehr laden"-Button + serverseitige Suche
- **Grid/Form-Seiten**: Responsive Padding, Button-Sizing (mind. 44px Tap-Target) und vertikales Stacking auf Mobile für alle verbleibenden Seiten
- **PWA**: App ist als Progressive Web App installierbar (vite-plugin-pwa, Service Worker, Manifest, Icons)

## Capabilities

### New Capabilities

- `mobile-navigation`: Hamburger-Menü mit Overlay-Sidebar auf Mobilgeräten – schließt bei Klick außerhalb oder nach Navigation
- `mobile-table-cards`: Responsive Darstellung von Tabellendaten als Card-Layout mit ⋮-Aktionen-Dropdown und Edit-Modal auf Mobile; paginierte Listen mit serverseitiger Suche
- `pwa-support`: Installierbare Progressive Web App mit Service Worker, Web App Manifest und Offline-Shell

### Modified Capabilities

*(keine bestehenden Specs betroffen – rein UI-seitige und leichtgewichtige Backend-Erweiterung)*

## Impact

- **web/src/components/AppShell.tsx**: Hamburger-State, Overlay-Logik, bedingte Sidebar-Anzeige, Mobile-Header
- **web/src/components/**: Neue Komponenten `MobileCard`, `ActionMenu`, `EditModal`
- **web/src/pages/*.tsx** (alle 20 Seiten): Tailwind Responsive-Klassen, Button-Sizing, Grid-Stacking
- **web/src/pages/MembersPage.tsx** + **AdminUsersPage.tsx**: Clientseitige `filter()` entfällt, ersetzt durch serverseitige Suche + Pagination-State
- **internal/members/handler.go**: `GET /api/members` erhält `search`, `limit`, `offset` Query-Parameter, gibt `{ items, total }` zurück
- **internal/auth/handler.go** (oder config): `GET /api/admin/users` erhält analoge Paginierung
- **web/vite.config.ts**: Integration `vite-plugin-pwa`
- **web/public/**: `manifest.json`, `icons/` (192×192, 512×512 PNG)
- **Neue npm-Dependency**: `vite-plugin-pwa`
