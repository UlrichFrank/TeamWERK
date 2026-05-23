## Why

Vereinsverwaltung bei Team Stuttgart läuft über Excel, WhatsApp und E-Mail-Ketten — Dienste, Termine und Zu-/Absagen sind unübersichtlich und fehleranfällig. Eine eigene Webanwendung bündelt die wichtigsten administrativen Aufgaben an einem Ort, ohne die bestehende öffentliche TYPO3-Site anzufassen.

## What Changes

- Neue eigenständige Web-App `VereinsWerk` auf dem bestehenden IONOS VPS Linux XS
- Go + Chi Backend (Single Binary, ~50 MB RAM) statt PHP — passt auf den ressourcenknappen VPS
- React + Tailwind Frontend, vom Go-Binary via `embed.FS` ausgeliefert (kein separater Webserver nötig)
- SQLite als Datenbank (kein separater DB-Prozess, geringer RAM-Verbrauch)
- JWT-basierte Authentifizierung, vollständig unabhängig von TYPO3 `fe_users`
- TYPO3-Integration: Link/Button auf der bestehenden team-stuttgart.org-Site verweist auf `app.team-stuttgart.org`
- Drei Module in Phase 1: CORE (Auth, Rollen, Vereinskonfiguration), MEMBERS (Spielerprofile, Familien), DUTIES (Dienste, Dienstbörse, Dienstkonten)

## Capabilities

### New Capabilities

- `auth`: Registrierung, Login, JWT-Session, Rollenmodell (Admin, Trainer, Elternteil, Spieler)
- `club-config`: Vereinsstammdaten, Saison-Konfiguration, Teams und Altersklassen
- `members`: Spielerprofile mit Stammdaten, Eltern/Kind-Verknüpfung, Mitgliedsstatus, Mannschaftszugehörigkeit
- `duties`: Diensttypen definieren, Dienste an Events knüpfen, Dienstbörse, Dienstkonten je Familie (Soll/Ist)
- `vps-deployment`: Go-Binary-Deployment via systemd auf IONOS VPS, HTTPS via Let's Encrypt, Cronjob für Scheduler

### Modified Capabilities

## Impact

- Neues Git-Repository `vereinswerk` (separates Repo, nicht im team-stuttgart-org Monorepo)
- TYPO3-Site erhält einen neuen Link/Button im Header oder Footer (minimale Änderung an `Resources/Private/Layouts/Page.html`)
- IONOS VPS: Go-Prozess als systemd-Service, Port 8080, Nginx als Reverse Proxy mit SSL-Termination
- Keine Änderungen an Mittwald Webhosting oder bestehender TYPO3-Konfiguration
