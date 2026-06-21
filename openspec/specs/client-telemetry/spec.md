# client-telemetry Specification

## Purpose

Anonymes clientseitiges Pageview-Tracking via Matomo, inkl. Custom Dimensions für Channel (PWA/Browser), Team-Slug und Rolle. Datenschutzfreundlicher Default (cookieless, IP-anonymisiert, DoNotTrack-Respekt) — kein Cookie-Banner nötig.

## Requirements

### Requirement: Konfigurierbarer Matomo-Tracker

Die Frontend-Anwendung SHALL einen Matomo-Tracker initialisieren, dessen URL und Site-ID über Build-Zeit-Umgebungsvariablen (`VITE_MATOMO_URL`, `VITE_MATOMO_SITE_ID`) konfiguriert werden. Sind beide Variablen leer oder ungesetzt, MUSS der Tracker als deaktiviert initialisiert werden und DARF KEINE Netzwerkanfragen an Matomo senden.

#### Scenario: Tracking aktiv mit gültiger Konfiguration

- **WHEN** beim Vite-Build sowohl `VITE_MATOMO_URL` als auch `VITE_MATOMO_SITE_ID` gesetzt sind und die App im Browser geladen wird
- **THEN** der Matomo-Tracker wird mit diesen Werten initialisiert und sendet bei Routenwechseln Pageviews an die konfigurierte URL

#### Scenario: Tracking deaktiviert ohne Konfiguration

- **WHEN** `VITE_MATOMO_URL` leer/ungesetzt ist (z.B. lokaler Dev-Build)
- **THEN** der Tracker ist als `disabled: true` initialisiert und es werden keine Anfragen an Matomo gesendet — die App funktioniert vollständig ohne Matomo-Erreichbarkeit

#### Scenario: Tracking deaktiviert bei fehlender Site-ID

- **WHEN** `VITE_MATOMO_URL` gesetzt ist, `VITE_MATOMO_SITE_ID` aber leer/ungesetzt
- **THEN** der Tracker ist als deaktiviert initialisiert und es werden keine Anfragen gesendet

### Requirement: Pageview-Tracking bei Routenwechseln

Innerhalb authentifizierter Bereiche (im `AppShell`-Outlet) SHALL die Anwendung bei jedem React-Router-Routenwechsel einen Matomo-Pageview senden. Public-Routes (Login, Register, Passwort-Reset, Beitrittsantrag, öffentliche Datenschutz-Seite) MUSS die App in dieser Iteration NICHT tracken.

#### Scenario: Pageview wird bei Routenwechsel im AppShell gesendet

- **WHEN** ein eingeloggter Nutzer von `/dienstboerse` zu `/profil` navigiert
- **THEN** der Tracker sendet einen Pageview mit dem neuen Pfad `/profil` an Matomo

#### Scenario: Initialer Pageview nach Login

- **WHEN** ein Nutzer sich anmeldet und auf die Startseite (z.B. `/dashboard`) weitergeleitet wird
- **THEN** der Tracker sendet einen Pageview für `/dashboard`, sobald der `AppShell` mit fertig geladenem Auth-Zustand gerendert ist

#### Scenario: Public-Routes werden nicht getrackt

- **WHEN** ein Nutzer die Login-Seite `/login` öffnet
- **THEN** wird kein Pageview an Matomo gesendet

#### Scenario: Pageview wird nicht während Auth-Loading gesendet

- **WHEN** der `AppShell` initial gerendert wird und `useAuth().loading === true`
- **THEN** wird noch kein Pageview gesendet — erst nachdem `loading` auf `false` gewechselt hat und der Nutzer authentifiziert ist

### Requirement: Custom Dimension 1 — Channel

Jeder gesendete Pageview SHALL die Custom Dimension mit ID 1 (`channel`) mit dem Wert `pwa` oder `browser` enthalten. Als `pwa` MUSS gelten: `window.matchMedia('(display-mode: standalone)').matches === true` ODER `navigator.standalone === true` (iOS Safari Homescreen).

#### Scenario: Standalone-Display-Mode wird als pwa erkannt

- **WHEN** die App in einem Browser läuft, in dem `matchMedia('(display-mode: standalone)').matches` `true` ist
- **THEN** wird in jedem Pageview die Dimension 1 mit Wert `pwa` mitgeschickt

#### Scenario: iOS Homescreen wird als pwa erkannt

- **WHEN** die App auf iOS Safari aus dem Homescreen geöffnet wird (`navigator.standalone === true`)
- **THEN** wird in jedem Pageview die Dimension 1 mit Wert `pwa` mitgeschickt

#### Scenario: Normaler Browser wird als browser erkannt

- **WHEN** die App in einem normalen Browser-Tab läuft (weder Display-Mode standalone noch iOS-standalone)
- **THEN** wird in jedem Pageview die Dimension 1 mit Wert `browser` mitgeschickt

### Requirement: Custom Dimension 2 — Team-Slug

Wenn der eingeloggte Nutzer einem oder mehreren Teams zugeordnet ist, SHALL der Tracker die Custom Dimension mit ID 2 (`team_slug`) als Slug des ersten Teams mitschicken. Hat der Nutzer kein Team, MUSS der Wert `none` sein. Kann der Team-Kontext nicht ermittelt werden (z.B. Endpoint-Fehler), MUSS die Dimension weggelassen oder mit `unknown` belegt werden — der Tracker DARF in diesem Fall nicht abbrechen.

#### Scenario: Nutzer mit Team

- **WHEN** der eingeloggte Nutzer Mitglied von Team `H1` (Slug `h1`) ist
- **THEN** wird in jedem Pageview die Dimension 2 mit Wert `h1` mitgeschickt

#### Scenario: Nutzer ohne Team

- **WHEN** der eingeloggte Nutzer keinem Team zugeordnet ist
- **THEN** wird in jedem Pageview die Dimension 2 mit Wert `none` mitgeschickt

#### Scenario: Team-Kontext nicht ladbar

- **WHEN** der Endpoint zum Ermitteln der Teams einen Fehler liefert
- **THEN** wird die Dimension 2 weggelassen oder mit `unknown` belegt; Pageviews werden trotzdem normal gesendet

### Requirement: Custom Dimension 3 — Rolle

Jeder Pageview eines eingeloggten Nutzers SHALL die Custom Dimension mit ID 3 (`role`) mit dem Wert `admin` oder `standard` enthalten, abgeleitet aus dem JWT-Claim `role` im `AuthContext`.

#### Scenario: Admin-Nutzer

- **WHEN** der eingeloggte Nutzer die Rolle `admin` hat
- **THEN** wird in jedem Pageview die Dimension 3 mit Wert `admin` mitgeschickt

#### Scenario: Standard-Nutzer

- **WHEN** der eingeloggte Nutzer die Rolle `standard` hat
- **THEN** wird in jedem Pageview die Dimension 3 mit Wert `standard` mitgeschickt

### Requirement: Datenschutzfreundlicher Default

Der Tracker SHALL clientseitig keine identifizierenden Cookies setzen. Er MUSS mit `disableCookies: true` initialisiert werden und KEINE der folgenden Datenpunkte versenden: User-ID, E-Mail-Adresse, Mitglieds-ID, Spitzname, Klartextnamen oder andere PII.

#### Scenario: Keine Matomo-Cookies im Browser

- **WHEN** die App geladen ist und Pageviews gesendet werden
- **THEN** sind im `document.cookie` keine Cookies mit Prefix `_pk_` vorhanden

#### Scenario: Keine PII in Tracker-Requests

- **WHEN** ein Pageview an Matomo gesendet wird
- **THEN** enthält der HTTP-Request weder `user_id`, noch E-Mail, noch Mitglieds-ID, noch Klartextnamen — nur Pfad, Custom Dimensions (Channel/Team-Slug/Rolle) und vom Browser implizit gesetzte Felder (User-Agent, Sprache, Auflösung)

### Requirement: Öffentliche Datenschutz-Seite mit Matomo-Hinweis

Die Anwendung SHALL eine öffentlich (ohne Login) erreichbare Route `/datenschutz` bereitstellen, die u.a. einen Absatz zur Matomo-Nutzung enthält: gemessene Daten, Custom Dimensions, Anonymisierung, Hosting-Ort (mittwald) und Hinweis auf DoNotTrack-Respektierung. Der Absatz MUSS auch erklären, dass Kinder-Accounts gleich behandelt werden.

#### Scenario: Datenschutz-Seite ist ohne Login erreichbar

- **WHEN** ein nicht-authentifizierter Browser `/datenschutz` öffnet
- **THEN** wird die Datenschutz-Seite angezeigt, ohne dass ein Redirect auf `/login` erfolgt

#### Scenario: Datenschutz-Seite erwähnt Matomo

- **WHEN** die Datenschutz-Seite gerendert ist
- **THEN** enthält sie einen klar erkennbaren Abschnitt, der Matomo, die gemessenen Daten (Pageviews, Channel, Team, Rolle), Anonymisierung (IP, Cookieless) und Hosting bei mittwald erwähnt

#### Scenario: Datenschutz-Seite erwähnt Behandlung von Kinder-Accounts

- **WHEN** die Datenschutz-Seite gerendert ist
- **THEN** wird explizit erwähnt, dass Kinder-Accounts genauso anonym getrackt werden wie Erwachsenen-Accounts und keine personenbezogenen Daten gesendet werden
