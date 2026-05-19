## 1. Backend: Serverseitige Paginierung

- [x] 1.1 `internal/members/handler.go`: Query-Parameter `search`, `limit`, `offset` für `GET /api/members` auslesen; SQL-Query mit `LIKE`-Filter und `LIMIT`/`OFFSET` erweitern
- [x] 1.2 `internal/members/handler.go`: Response-Format von `[]Member` auf `{ items: []Member, total: int }` umstellen (COUNT-Query für total)
- [x] 1.3 `internal/auth/handler.go` (o.ä.): `GET /api/admin/users` analog mit `search`, `limit`, `offset` und `{ items, total }`-Response paginieren (nur registrierte Nutzer; Einladungen + Anfragen bleiben unpaginiert)

## 2. Foundation: AppShell & Navigation

- [x] 2.1 `useMediaQuery`-Hook in `web/src/lib/useMediaQuery.ts` erstellen
- [x] 2.2 AppShell: Hamburger-State (`sidebarOpen`) und Toggle-Logik hinzufügen
- [x] 2.3 AppShell: Mobile Kopfzeile (`<header>`) mit Hamburger-Button (☰) und App-Name (nur `sm:hidden`)
- [x] 2.4 AppShell: Sidebar auf Desktop via `hidden sm:flex` und als Fixed-Overlay (`fixed inset-y-0 left-0 z-50`) auf Mobile implementieren
- [x] 2.5 AppShell: Backdrop (`fixed inset-0 z-40 bg-black/40`) hinter Overlay-Sidebar
- [x] 2.6 AppShell: Schließen-Button (✕) in der Overlay-Sidebar oben rechts
- [x] 2.7 AppShell: Sidebar automatisch schließen nach Nav-Link-Klick (NavLink `onClick`)
- [x] 2.8 AppShell: Main-Content Padding `px-4 py-4 sm:p-8`; Dekorationsklassen nur auf Desktop: `sm:rounded-tl-3xl sm:rounded-bl-3xl sm:border-l-4 sm:border-brand-yellow`

## 3. Shared Utility: Wiederverwendbare Mobile-Komponenten

- [x] 3.1 `MobileCard`-Komponente in `web/src/components/MobileCard.tsx` erstellen (Props: title, subtitle, badge, children für Actions)
- [x] 3.2 `ActionMenu`-Komponente in `web/src/components/ActionMenu.tsx` erstellen (⋮-Button + Dropdown, schließt bei Außen-Klick via `useEffect` + document-EventListener)
- [x] 3.3 `EditModal`-Komponente in `web/src/components/EditModal.tsx` erstellen (Fixed-Overlay, schließt bei Backdrop-Klick, Speichern/Abbrechen-Buttons, children für Formularfelder)
- [x] 3.4 `usePaginatedFetch`-Hook in `web/src/lib/usePaginatedFetch.ts` erstellen (state: items, total, offset, loading; actions: loadMore, setSearch mit Debounce 300ms)

## 4. Tabellen-Seiten: Card-Layout + Paginierung

- [x] 4.1 `MembersPage`: Clientseitige `filter()`-Logik durch `usePaginatedFetch('/members')` ersetzen; Suchleiste `sticky top-0 z-10` und full-width auf Mobile; Card-Liste auf Mobile (Name + Position · Status-Badge); „Mehr laden"-Button wenn `items.length < total`
- [x] 4.2 `AdminUsersPage`: `usePaginatedFetch('/admin/users')` für registrierte Nutzer; Card-Liste auf Mobile (Name + E-Mail · Rolle-Badge + ⋮-Menü mit Löschen/Einladung-Widerruf/Anfrage-Aktionen); Einladungen + Anfragen-Cards ohne Paginierung (kleiner Datensatz)
- [x] 4.3 `AdminTeamsPage`: Card-Liste auf Mobile (Teamname + Altersklasse · Status-Badge; keine Aktionen nötig)
- [x] 4.4 `AdminDutyTypesPage`: Card-Liste auf Mobile (Name + Stunden · Geldersatz + ⋮-Menü); ⋮-„Bearbeiten" öffnet `EditModal` mit Name/Stunden/Geldersatz/Anker/Versatz-Feldern
- [ ] 4.5 `DutyAccountsPage`: Card-Liste auf Mobile (Name + Soll/Ist · Differenz-Badge; keine Aktionen)
- [ ] 4.6 `DutySlotsPage`: Card-Liste auf Mobile (Event + Datum · Belegung) mit Accordion-Expand für Zuteilungen als flache Liste (Name · Status-Badge · Aktions-Button wenn pending)

## 5. Grid/Form-Seiten: Responsive Anpassungen

- [x] 5.1 `AdminClubPage`: Padding `px-4 sm:px-6`, Button-Sizing `py-2.5 sm:py-1.5`
- [x] 5.2 `AdminSeasonsPage`: Grid/Form-Stacking auf Mobile, Button-Sizing
- [x] 5.3 `AdminTeamsPage`: Form-Bereich Padding + Button-Sizing (Grid bereits `grid-cols-1 lg:grid-cols-2`)
- [x] 5.4 `AdminDutyTypesPage`: Form-Bereich Padding + Button-Sizing
- [x] 5.5 `AdminGameTemplatePage`: Template-Items-Liste responsive, Button-Gruppen `flex-wrap`
- [x] 5.6 `MemberDetailPage`: Formular-Grid einspaltig auf Mobile, Button-Gruppen `flex-wrap`
- [x] 5.7 `ProfilePage`: Formular responsive, Button-Sizing
- [x] 5.8 `SpielplanPage`: Spielplan-Liste/Grid auf Mobile einspaltig
- [x] 5.9 `SpieltagDetailPage`: Slot-Übersicht responsive, Action-Buttons `flex-wrap`
- [x] 5.10 `DutyBoardPage`: Padding `px-4 sm:px-6`, Button-Sizing (Cards bereits responsive)
- [x] 5.11 `MembershipRequestsPage`: Anfragen-Liste responsive, Button-Gruppen `flex-wrap`
- [x] 5.12 `LoginPage`: Padding responsive, Button `w-full sm:w-auto`
- [x] 5.13 `RegisterPage`: Padding responsive, Button `w-full sm:w-auto`
- [x] 5.14 `ForgotPasswordPage` + `ResetPasswordPage`: Padding responsive

## 6. PWA: Manifest, Service Worker, Icons

- [x] 6.1 `vite-plugin-pwa` als Dev-Dependency installieren (`pnpm add -D vite-plugin-pwa`)
- [x] 6.2 `web/vite.config.ts`: VitePWA-Plugin konfigurieren (registerType: 'autoUpdate', Workbox NetworkFirst für `/api/*`, CacheFirst für Assets)
- [x] 6.3 `web/public/manifest.json` erstellen (name, short_name, theme_color `#000000`, background_color `#FFFFFF`, display `standalone`, start_url `/`, icons)
- [x] 6.4 PWA-Icons generieren: `web/public/icons/icon-192.png` und `icon-512.png` aus Logo-SVG (schwarzer Hintergrund, 10% Safe-Zone-Padding für Maskable)
- [x] 6.5 `web/public/offline.html` erstellen (TeamWERK-Branding, „Sie sind offline"-Hinweis, Link zum Reload)
- [x] 6.6 `web/index.html`: `<link rel="manifest">` und `<meta name="theme-color">` eintragen (falls nicht automatisch von vite-plugin-pwa gesetzt)

## 7. Qualitätssicherung (Manuell zu testen)

- [ ] 7.1 Vite Dev-Server starten und alle Seiten auf 375px (Chrome DevTools) testen
- [ ] 7.2 Hamburger-Menü: Öffnen, Schließen (Backdrop / ✕ / Nav-Link) testen
- [ ] 7.3 ⋮-Dropdown auf allen 6 Tabellen-Seiten testen (Öffnen, Außen-Klick, Aktion)
- [ ] 7.4 Edit-Modal auf AdminDutyTypesPage testen (Öffnen, Speichern, Abbrechen)
- [ ] 7.5 Paginierung auf MembersPage testen: erste Seite, „Mehr laden", Suche, Suche + laden
- [ ] 7.6 Paginierung auf AdminUsersPage testen
- [ ] 7.7 DutySlotsPage Accordion auf Mobile testen
- [ ] 7.8 Alle Formular-Seiten auf Mobile ausfüllen und absenden
- [ ] 7.9 PWA-Installierbarkeit in Chrome DevTools prüfen (Lighthouse PWA-Audit)
- [ ] 7.10 Offline-Fallback testen (DevTools → Network → Offline)
- [ ] 7.11 Desktop-Layout bei 1280px auf Regression prüfen (kein Umbau Desktop-Ansicht)
