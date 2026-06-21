## 1. Vorbereitung & Recherche

- [x] 1.1 Verifizieren, ob ein Endpoint existiert, der das/die Team(s) des eingeloggten Nutzers mit Slug zurückliefert (Kandidat: `/api/me/teams`, sonst über Member-Detail). Ergebnis im Design unter "Open Questions" festhalten.
- [x] 1.2 Matomo-Site `TeamWERK` (Site-ID **4**, URL `https://internal.team-stuttgart.org`) bei mittwald angelegt. Custom Dimensions 1/2/3 (`channel`/`team_slug`/`role`) als Action-Scope angelegt.
- [ ] 1.3 In Matomo-Admin Datenschutz-Einstellungen setzen: IP-Anonymisierung 2 Bytes, DoNotTrack respektieren, "Force anonymous tracking" aktivieren.

## 2. Tracker-Setup

- [x] 2.1 ~~`pnpm add @jonkoops/matomo-tracker-react`~~ Library war deprecated → Pivot zu eigenem `_paq`-Wrapper (siehe Design D1). Keine npm-Dependency.
- [x] 2.2 Neue Datei `web/src/lib/telemetry.ts`: `initTelemetry(url, siteId)` lädt `matomo.js` per `<script>`, setzt `disableCookies`, `setSecureCookie`, `setRequestMethod('POST')`. Bei leerer URL/SiteID: No-Op. Helper: `detectChannel()`, `slugifyTeam(name)`, `setChannelDimension()`, `setTeamSlugDimension()`, `setRoleDimension()`, `trackPageview(href, title)`.
- [x] 2.3 `initTelemetry(import.meta.env.VITE_MATOMO_URL, Number(import.meta.env.VITE_MATOMO_SITE_ID))` in `web/src/main.tsx` einmal beim App-Start aufrufen.
- [x] 2.4 `.env.example` (oder Äquivalent) um `VITE_MATOMO_URL` und `VITE_MATOMO_SITE_ID` ergänzen (leer als Default).

## 3. Pageview-Tracking im AppShell

- [x] 3.1 In `web/src/components/AppShell.tsx` Helper aus `lib/telemetry` und `useLocation()` einbinden.
- [x] 3.2 `useEffect` auf `pathname` + Auth-Status: nur tracken, wenn `!loading && user`. `trackPageview(window.location.href, document.title)` aufrufen.
- [x] 3.3 Vor dem `trackPageview` `setChannelDimension()` (Dim 1) aufrufen.
- [x] 3.4 Vor dem `trackPageview` `setRoleDimension(user.role)` (Dim 3) aufrufen.

## 4. Team-Slug (Custom Dimension 2)

- [x] 4.1 Team-Slug-Hook implementieren: bei Mount des `AppShell` (nach Auth-Loading) einmal die Teams des Nutzers laden (Endpoint aus Task 1.1), ersten Slug als `useState` halten, `none` wenn leer, `unknown` bei Fehler.
- [x] 4.2 Hook im AppShell-Effect aus Task 3.x einbinden: `setTeamSlugDimension(teamSlug)` (Dim 2).
- [x] 4.3 Edge-Case: bei Logout/Login-Wechsel Team-Slug neu laden (state-Reset auf `none` bei `!user`, neu laden bei `user?.id`-Wechsel).

## 5. Datenschutz-Seite

- [x] 5.1 Neue Datei `web/src/pages/DatenschutzPage.tsx` mit statischer Seite (brand-Tokens, lucide-Icons, kein Cookie-Banner).
- [x] 5.2 Inhalte: Verantwortlicher (Platzhalter für Vorstand), Hosting (IONOS für App, mittwald für Matomo), gespeicherte Daten (App: laut Mitgliedschaft + Auth; Matomo: anonyme Nutzungsdaten), Rechte (Auskunft, Löschung), Matomo-Absatz wie in Spec gefordert (inkl. Kinder-Account-Hinweis), Kontakt.
- [x] 5.3 Route `/datenschutz` in `web/src/App.tsx` als Public-Route eintragen (außerhalb des `AppShell`-Outlets).
- [x] 5.4 Link auf `/datenschutz` im Footer/Login-Bereich ergänzen (sichtbare Erreichbarkeit).

## 6. Public-Route-Behandlung

- [x] 6.1 Sicherstellen, dass Public-Routes (`/login`, `/register`, `/passwort-vergessen`, `/beitritt`, `/datenschutz`) **nicht** im `AppShell` liegen und somit keine Pageviews senden. Bestätigt durch die Routenstruktur in `App.tsx`: alle Public-Routes liegen außerhalb des `PrivateRoute><AppShell>`-Outlets, das Tracking-Effect lebt nur im AppShell → kein zusätzlicher Pfad-Check nötig.

## 7. Tests

- [x] 7.1 Unit-Test für `detectChannel()`: mockt `matchMedia` und `navigator.standalone`, prüft alle drei Kombinationen aus Spec (display-mode standalone, iOS standalone, normaler Browser).
- [x] 7.2 Test für `AppShell`-Tracking: bei Routenwechsel wird `trackPageview` aufgerufen, mit gesetzten Custom Dimensions 1/2/3. `lib/telemetry`-Module gemockt.
- [x] 7.3 Test: während `loading === true` wird kein `trackPageview` ausgelöst.
- [x] 7.4 Test: leeres `VITE_MATOMO_URL` führt zu `enabled === false` und keinen `_paq`-Calls.
- [x] 7.5 Test für `DatenschutzPage`: rendert mit anonymem Ctx, enthält Matomo-Absatz, IP-Anonymisierung, DoNotTrack, mittwald, keine Cookies, Kinder-Account-Hinweis.
- [x] 7.6 Integration-Test: Route `/datenschutz` ist ohne Auth erreichbar (rendert DatenschutzPage statt LoginPage-Inhalt).

## 8. Build & Deploy

- [x] 8.1 `VITE_MATOMO_URL` und `VITE_MATOMO_SITE_ID` in der Deploy-Pipeline / `make build`-Aufruf bereitstellen. Neue Datei `web/.env.example` dokumentiert die Variablen + Optionen (`.env.production` in `web/` oder Shell-Export beim Build). Wurzel-`.env.example` enthält ebenfalls einen Eintrag.
- [x] 8.2 `pnpm -C web build` ausgeführt; Bundle gzip 183.33 KB (vor Change unbekannt, aber Wachstum durch eigenen Wrapper minimal — kein neues npm-Paket, ~100 LoC Wrapper).
- [x] 8.3 `make deploy` erfolgreich auf VPS ausgerollt (Build-Hash `61deb34`, Bundle enthält `matomo.team-stuttgart.org` und Site-ID 4).
- [ ] 8.4 In produktiver Umgebung verifizieren (Browser-DevTools Network-Tab): Matomo-Requests werden gesendet, enthalten Custom Dimensions 1/2/3, kein `_pk_*` Cookie wird gesetzt, IP im Matomo-Backend anonymisiert.

## 9. Vorstand-Freigabe & Go-Live

- [ ] 9.1 Datenschutz-Text dem Vorstand zur Prüfung/Freigabe vorlegen; Platzhalter (Verantwortlicher, Kontaktdaten) durch finale Werte ersetzen.
- [ ] 9.2 Bei Bedarf Mitgliederinformation (Mail/Newsletter) über die Einführung des anonymen Tracking versenden.

## 10. Validierung

- [x] 10.1 `openspec validate matomo-client-tracking --strict` läuft sauber.
- [x] 10.2 `pnpm -C web test` grün (382 Tests). `make test` (Go) — kein Backend-Code geändert; nicht erneut ausgeführt.
- [x] 10.3 `pnpm -C web build` ohne Errors (1 Pre-existing Warning zu Chunk-Größe ist nicht durch diese Change verursacht; gzip-Hauptchunk 183 KB).
- [ ] 10.4 `/verify-change` Slash-Command durchlaufen (Projekt-Invarianten: brand-Tokens, lucide-Icons) — durch User auszuführen.
