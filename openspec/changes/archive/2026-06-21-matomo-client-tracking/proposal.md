## Why

Aktuell gibt es keine Sicht darauf, welche Features im TeamWERK tatsächlich von Mitgliedern genutzt werden. Damit fehlt die Grundlage für UX-Entscheidungen (z.B. welche Admin-Bereiche brauchen besseres Onboarding, ist die PWA-Adoption hoch, welche Team-Segmente nutzen welche Features). Eine Matomo-Instanz läuft bereits bei mittwald — die soll für anonyme Produkt-Nutzungs-Metriken eingebunden werden, ohne PII zu sammeln und ohne ein Cookie-Banner einführen zu müssen.

## What Changes

- Neue Frontend-Capability `client-telemetry`: anonymes clientseitiges Pageview-Tracking via externer Matomo-Instanz.
- React-Provider (`@jonkoops/matomo-tracker-react`) wird in `main.tsx` initialisiert; Tracker pusht bei jedem React-Router-Routenwechsel einen Pageview.
- Drei Custom Dimensions ohne PII:
  - **Dim 1 (`channel`):** `pwa` (Standalone-Display-Mode oder iOS `navigator.standalone`) oder `browser`.
  - **Dim 2 (`team_slug`):** Slug des Haupt-Teams aus dem Auth-Kontext (z.B. `h1`, `f-jugend`); `none` wenn keins.
  - **Dim 3 (`role`):** System-Rolle `admin` oder `standard` aus JWT-Claim.
- Datenschutz-konformer Default: **Cookieless Tracking** + **IP-Anonymisierung (2 Bytes)** + **DoNotTrack respektieren**. Damit kein Cookie-Banner nötig.
- Konfiguration über Vite-Env-Variablen `VITE_MATOMO_URL` und `VITE_MATOMO_SITE_ID`. Sind die leer → Tracking ist No-Op (z.B. lokale Dev-Umgebung).
- Erweiterung der Frontend-Seite `/datenschutz` um einen kurzen Absatz, der das Matomo-Tracking transparent erklärt (auch für Kinder-Accounts).
- Kein Backend-Code, keine neuen Routen, keine DB-Migration.

## Capabilities

### New Capabilities
- `client-telemetry`: Anonymes clientseitiges Pageview-Tracking mit Matomo, inkl. Custom Dimensions für Channel/Team/Rolle und datenschutzfreundlichem Default.

### Modified Capabilities
- Keine — `/datenschutz` ist eine reine Frontend-Seite und nicht durch eine eigene Capability spezifiziert.

## Impact

- **Frontend (`web/`):**
  - Neue Dependency: `@jonkoops/matomo-tracker-react` (~10 KB gz).
  - Neue Datei `web/src/lib/telemetry.ts` (Tracker-Setup, Custom-Dimension-Helper).
  - `web/src/main.tsx` — `MatomoProvider` einhängen.
  - `web/src/components/AppShell.tsx` — Route-Listener (`useLocation` + `trackPageView`), Custom-Dimensions setzen.
  - `web/src/pages/Datenschutz.tsx` (oder existierende Datenschutz-Seite) — Absatz zur Matomo-Nutzung.
  - `.env.example` (falls vorhanden) — `VITE_MATOMO_URL`, `VITE_MATOMO_SITE_ID` ergänzen.
- **Backend:** unverändert.
- **Deployment:** `VITE_MATOMO_URL` / `VITE_MATOMO_SITE_ID` müssen zum Build-Zeitpunkt gesetzt sein (in `make build` / CI / Deploy-Pipeline). Variablen werden in die statische Bundle eingebacken — kein Secret, daher unkritisch.
- **RAM-Footprint VPS:** unverändert (rein Client-Side).
- **Externe Dienste:** Matomo bei mittwald ist bereits in Betrieb (für die Vereins-Homepage) — keine neue externe Abhängigkeit, nur zusätzliche Site-ID.
- **DSGVO:** Anonymisierte Datenverarbeitung, kein Cookie-Banner. Transparenz über `/datenschutz`. Auftragsverarbeitungsvertrag mit mittwald existiert bereits.
