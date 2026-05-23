## Context

TeamWERK hat bisher keine Responsive-Unterstützung und ist nicht als PWA installierbar. Die AppShell besteht aus einer fixen `w-56`-Sidebar (224px), die auf Mobilgeräten über die Hälfte des Bildschirms einnimmt. Die Pages nutzen ausschließlich Desktop-Layouts: Tabellen ohne Breakpoints, Button-Gruppen ohne Wrapping, `p-8` Padding das auf 375px zu eng wird. Zusätzlich lädt `MembersPage` alle Mitglieder als einmaligen Client-Array – bei bis zu 1000 Einträgen ein Performance-Problem auf Mobile (1000 Cards = ~4000 DOM-Knoten, ~60.000px Scrollhöhe).

Die Änderung ist überwiegend frontend-seitig mit einer leichtgewichtigen Backend-Erweiterung (Paginierung).

## Goals / Non-Goals

**Goals:**
- Vollständig bedienbare App auf 375–430px (iPhone SE, iPhone 12/14)
- Navigation per Hamburger-Menü (Overlay-Drawer) auf Mobile
- Alle 6 Tabellen-Seiten als Cards auf Mobile, mit ⋮-Aktionen-Menü und Edit-Modal
- Serverseitige Paginierung + Suche für Members und Admin-Users
- Alle 14 Grid/Form-Seiten responsive angepasst (Padding, Button-Sizing, Stacking)
- Touch-Targets ≥ 44px auf allen interaktiven Elementen
- Installierbare PWA (Add to Homescreen, Offline-Shell)

**Non-Goals:**
- Native App-Feeling (keine Bottom Navigation, kein Swipe-Gesture)
- Landscape-Optimierungen (Landscape funktioniert, ist aber nicht gesondert optimiert)
- iPad/Tablet-spezifische Layouts (Tablet nutzt Desktop-Layout ab 640px)
- Dark Mode
- Push-Benachrichtigungen
- Vollständige Offline-Funktionalität (nur Shell, keine Offline-Dateneingabe)

## Decisions

### 1. Overlay-Sidebar statt Push-Layout
Die Sidebar öffnet sich als Fixed-Position-Overlay (z-50) über dem Content, mit einem halbtransparenten Backdrop. Klick außerhalb oder nach Nav-Klick schließt sie. Dekorationsklassen des Main-Content (`rounded-tl-3xl rounded-bl-3xl border-l-4 border-brand-yellow`) nur auf Desktop aktiv (`sm:`-Präfix), da sie ohne sichtbare Sidebar optisch sinnlos sind.

**Alternativen:**
- *Push-Layout*: Content wird nach rechts geschoben — komplexer, schlechtere UX bei schmalem Screen
- *Bottom Navigation*: Passt nicht zur bestehenden hierarchischen Nav-Struktur

### 2. Mobile Card-Layout für Tabellen (kein `overflow-x: scroll`)
Auf Mobile (<640px) wird jede Tabellenzeile als eigenständige Card gerendert. Die Card zeigt primäre Info und einen ⋮-Button für Aktionen.

**Alternativen:**
- *Horizontales Scrollen*: Zugänglich, aber ungewohnt
- *`hidden sm:table-cell` pro Spalte*: Einfacher, aber Cards bieten besseres Tap-Target-Handling

### 3. Edit-Modal für Inline-Edit-Formulare auf Mobile
Seiten mit Inline-Edit (AdminDutyTypesPage: 5 Felder in Tabellenzeile) öffnen auf Mobile über das ⋮-Menü ein Modal. Das Modal enthält exakt dieselben Felder wie die Desktop-Inline-Zeile.

**Alternativen:**
- *Inline-Card-Expansion*: Tabellenzeile expandiert zu Formular-Card — komplexer State, visuell unübersichtlich bei mehreren gleichzeitig
- *Separate Edit-Route*: Zu viel Navigation für kleine Formulare

### 4. Serverseitige Paginierung + Suche für Members und Admin-Users
`GET /api/members` und `GET /api/admin/users` erhalten Query-Parameter `search`, `limit` (default 50), `offset` (default 0). Response-Format: `{ items: T[], total: int }`. Die bestehende clientseitige `filter()`-Logik in MembersPage entfällt.

**Paginierungs-UI**: „Mehr laden"-Button (Load More) statt Seitennavigation:
- Einfacher State (nur `offset`)
- Mobile-freundlicher als Seitennavigation
- Kein Positions-Verlust beim Tippen auf „Mehr laden"

**Suche**: Debounced (300ms), setzt `offset` auf 0 bei neuer Suchanfrage.

**Admin-Users Besonderheit**: Das Endpoint `/api/admin/users` gibt nur registrierte Nutzer zurück. Einladungen und Anfragen bleiben unpaginiert (realistisch < 20 Einträge).

**Alternativen:**
- *Client-seitiges Laden aller Einträge + Virtualisierung (react-window)*: Neue Dependency, erfordert feste Höhen, komplexer als Paginierung
- *Cursor-basierte Paginierung*: Sauberer für sehr große Datasets, aber overhead für 1000 Einträge

### 5. `sm:` (640px) als einziger Mobile-Breakpoint
Kein `md:`-Einsatz für Mobile-Logik. Einfach und konsistent.

### 6. PWA via vite-plugin-pwa
`vite-plugin-pwa` ist die Standardlösung für Vite-basierte PWAs. Sie generiert Service Worker und kann das Manifest verarbeiten.

**Service Worker Strategie:**
- `/api/*`: NetworkFirst (immer frische Daten, bei Offline Cache-Fallback)
- Statische Assets (JS, CSS, Fonts): CacheFirst (schnelles Laden bei wiederkehrenden Besuchen)
- HTML-Shell: NetworkFirst mit Offline-Fallback auf `/offline.html`

**Manifest (web/public/manifest.json):**
```json
{
  "name": "TeamWERK",
  "short_name": "TeamWERK",
  "theme_color": "#000000",
  "background_color": "#FFFFFF",
  "display": "standalone",
  "start_url": "/",
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable" }
  ]
}
```

Icons werden aus dem vorhandenen Logo-SVG als PNG gerendert.

**Alternativ:** Manueller Service Worker — mehr Kontrolle, aber mehr Wartungsaufwand. vite-plugin-pwa ist für Vite der Standard.

### 7. useMediaQuery-Hook für React-State
Für Logik die JS-State benötigt (Dropdown automatisch schließen bei Resize über 640px) wird ein `useMediaQuery('(max-width: 639px)')` Hook eingesetzt. Visuelle Responsive-Logik bleibt ausschließlich in Tailwind-Klassen.

## Risks / Trade-offs

- **MembersPage mit 1000 Einträgen** → Serverseitige Paginierung löst das DOM-Problem. Risiko: Suchfeld benötigt Debounce damit keine API-Flut entsteht. Mitigation: 300ms Debounce im Hook.
- **AdminDutyTypesPage Nested-Edit** → Edit-Modal auf Mobile braucht separaten State für welche Zeile offen ist. Dasselbe State-Pattern wie beim bestehenden `editId` — kein neuer Ansatz nötig.
- **DutySlotsPage Nested-Table** → Zuteilungen bleiben als flache Liste (kein Card-in-Card), nur visuell eingerückt.
- **PWA-Icons** → Logo-SVG muss zu PNG gerendert werden. Kein automatischer Build-Schritt — einmalige manuelle Erstellung oder Skript. Maskable Icon braucht 10% Safe-Zone Padding.
- **vite-plugin-pwa Breaking Changes** → Plugin ist aktiv gepflegt; Version vor Installation prüfen. Service Worker kann bei Update-Problemen gecacht bleiben (Workbox-Strategie setzt skipWaiting).
- **Viele Dateien gleichzeitig** → 20 Pages + AppShell. Mitigation: nach Page-Typ gruppiert, nach jedem Block manuell testen.

## Migration Plan

1. Backend-Paginierung (Members + Admin-Users Handler) — unabhängig deploybar
2. AppShell (Navigation muss überall funktionieren bevor Pages getestet werden)
3. Shared Komponenten (MobileCard, ActionMenu, EditModal)
4. Tabellen-Seiten der Reihe nach
5. Grid/Form-Seiten (risikoärmer, kleinere Änderungen)
6. PWA-Setup (vite-plugin-pwa, Manifest, Icons, Offline-Page)
7. Manueller Test auf echtem Mobilgerät + Installierbarkeit prüfen
8. Rollback: Git revert — kein Deployment-Risiko, rein CSS/React/leichte API-Erweiterung
