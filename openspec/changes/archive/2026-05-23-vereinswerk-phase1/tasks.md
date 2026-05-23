## 1. Repository & Projektstruktur

- [x] 1.1 Neues Git-Repository `vereinswerk` anlegen (außerhalb von team-stuttgart-org)
- [x] 1.2 Go-Modulstruktur initialisieren (`go mod init github.com/teamstuttgart/vereinswerk`)
- [x] 1.3 Verzeichnisstruktur anlegen: `cmd/`, `internal/`, `web/` (React), `migrations/`
- [x] 1.4 Abhängigkeiten hinzufügen: `go-chi/chi`, `golang-jwt/jwt`, `modernc.org/sqlite`, `golang-migrate/migrate`
- [x] 1.5 Vite + React + Tailwind in `web/` initialisieren (`npm create vite@latest`)
- [x] 1.6 `Makefile` mit Targets `build`, `dev`, `deploy` anlegen
- [x] 1.7 `.gitignore` für Go-Binary, SQLite-Datei, `node_modules`, `.env` anlegen

## 2. VPS-Setup (IONOS)

- [ ] 2.1 SSH-Zugang zum IONOS VPS prüfen, Ubuntu-Version notieren — **PENDING: VPS nicht initialisiert**
- [ ] 2.2 Nginx installieren und aktivieren (`apt install nginx`)
- [ ] 2.3 Go 1.23+ auf dem VPS installieren (für lokales Debugging — Deployment ist Binary-Transfer)
- [ ] 2.4 DNS: Subdomain `intern.team-stuttgart.org` auf VPS-IP setzen
- [ ] 2.5 Nginx-Vhost für `intern.team-stuttgart.org` anlegen (Port 80, Proxy zu 8080)
- [ ] 2.6 SSL-Zertifikat via Certbot einrichten (`certbot --nginx -d intern.team-stuttgart.org`)
- [ ] 2.7 Nginx HTTP→HTTPS-Redirect konfigurieren
- [ ] 2.8 Verzeichnis `/var/lib/vereinswerk/` anlegen, Berechtigungen setzen
- [ ] 2.9 systemd-Service `vereinswerk.service` anlegen und aktivieren

## 3. Datenbank & Migrations-Setup

- [x] 3.1 `golang-migrate` CLI lokal installieren — **Optional: Migrationen laufen über Binary**
- [x] 3.2 Erste Migration: Tabellen `users`, `refresh_tokens`, `invitation_tokens`, `password_reset_tokens`
- [x] 3.3 Migration: Tabellen `clubs`, `seasons`, `teams`, `team_memberships`
- [x] 3.4 Migration: Tabellen `members`, `family_links`
- [x] 3.5 Migration: Tabellen `duty_types`, `duty_slots`, `duty_assignments`, `duty_accounts`
- [x] 3.6 `PRAGMA journal_mode=WAL` beim DB-Start in Go setzen
- [x] 3.7 `sqlc.yaml` konfigurieren, erste SQL-Queries für Auth schreiben und `sqlc generate` ausführen — **Optional: sqlc nicht verwendet, direkte SQL**

## 4. Backend: Auth (Chi + JWT)

- [x] 4.1 Chi-Router-Grundgerüst mit Middleware-Stack aufbauen (Logger, Recoverer, CORS)
- [x] 4.2 `POST /api/auth/login` — Credentials prüfen, Access + Refresh Token ausstellen
- [x] 4.3 `POST /api/auth/refresh` — Refresh Token rotieren, neuen Access Token ausstellen
- [x] 4.4 `POST /api/auth/logout` — Refresh Token Cookie löschen, DB-Eintrag invalidieren
- [x] 4.5 JWT-Auth-Middleware für geschützte Routen implementieren
- [x] 4.6 Rollen-Middleware (`RequireRole`) für Admin-geschützte Routen
- [x] 4.7 `POST /api/auth/request-membership` — Beitrittsantrag stellen (Name, E-Mail, Team; öffentlich, kein Login)
- [x] 4.8 `GET /api/admin/membership-requests` — Offene Anträge auflisten (Admin + Trainer des jeweiligen Teams)
- [x] 4.9 `POST /api/admin/membership-requests/:id/approve` — Antrag genehmigen: Registrierungstoken generieren, E-Mail senden
- [x] 4.10 `POST /api/admin/membership-requests/:id/reject` — Antrag ablehnen: Ablehnungs-E-Mail senden
- [x] 4.11 `POST /api/auth/invite` — Direkteinladung durch Admin oder Trainer (E-Mail + Team + Rolle angeben)
- [x] 4.12 `POST /api/auth/register` — Registrierungstoken validieren, Nutzer anlegen, Token invalidieren
- [x] 4.13 `POST /api/auth/forgot-password` — Reset-Link per E-Mail senden
- [x] 4.14 `POST /api/auth/reset-password` — Passwort aktualisieren, alle Refresh Tokens invalidieren
- [x] 4.15 E-Mail-Versand via SMTP konfigurieren: Mittwald-Mailaccount `vorstand@team-stuttgart.org` (`.env`: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS)

## 5. Backend: CORE (Club-Config)

- [x] 5.1 `GET/PUT /api/admin/club` — Vereinsstammdaten lesen und aktualisieren
- [x] 5.2 `GET/POST /api/admin/seasons` — Saisons auflisten und anlegen
- [x] 5.3 `PUT /api/admin/seasons/:id/activate` — Saison als aktiv markieren
- [x] 5.4 `GET/POST /api/admin/teams` — Teams auflisten und anlegen
- [x] 5.5 `PUT /api/admin/teams/:id` — Team bearbeiten (Name, Altersklasse, Status)
- [x] 5.6 `POST /api/admin/teams/:id/assign-trainer` — Trainer einem Team zuweisen

## 6. Backend: MEMBERS

- [x] 6.1 `GET /api/members` — Mitgliederliste (gefiltert nach Rolle: Admin sieht alle, Trainer nur eigenes Team)
- [x] 6.2 `POST /api/members` — Neues Mitglied anlegen
- [x] 6.3 `GET/PUT /api/members/:id` — Mitglied lesen und aktualisieren
- [x] 6.4 `PUT /api/members/:id/status` — Mitgliedsstatus ändern
- [x] 6.5 `POST /api/members/:id/team-assignment` — Mannschaftszugehörigkeit zuweisen
- [x] 6.6 `POST /api/admin/family-links` — Elternteil mit Spielerprofil verknüpfen
- [x] 6.7 `GET /api/members/export` — CSV-Export der Mitgliederliste (Admin only)
- [x] 6.8 Fahrzeuginformation: `GET/PUT /api/profile/vehicle` — eigene Fahrzeugdaten pflegen

## 7. Backend: DUTIES

- [x] 7.1 `GET/POST /api/admin/duty-types` — Diensttypen auflisten und anlegen
- [x] 7.2 `PUT /api/admin/duty-types/:id` — Diensttyp bearbeiten
- [x] 7.3 `GET/POST /api/duty-slots` — Dienst-Slots auflisten und anlegen
- [x] 7.4 `PUT /api/duty-slots/:id` — Dienst-Slot bearbeiten
- [x] 7.5 `GET /api/duty-board` — Offene Dienst-Slots anzeigen (sortiert nach Event-Datum)
- [x] 7.6 `POST /api/duty-board/:slotId/claim` — Dienst-Slot beanspruchen
- [x] 7.7 `POST /api/duty-assignments/:id/fulfill` — Diensterfüllung durch Admin/Teamleiter bestätigen
- [x] 7.8 `POST /api/duty-assignments/:id/cash-substitute` — Geldersatz erfassen
- [x] 7.9 `GET /api/duty-accounts` — Dienstkonten anzeigen (eigenes für Elternteil, alle für Admin)
- [x] 7.10 `GET /api/admin/duty-accounts/export` — CSV-Export Dienstkonten (Admin only)
- [x] 7.11 Admin: Saison-Soll je Diensttyp konfigurieren (`PUT /api/admin/seasons/:id/duty-targets`)

## 8. Frontend: Grundgerüst & Auth

- [x] 8.1 React Router einrichten (öffentliche Routes: Login, Register, Passwort-Reset; geschützte Routes: App-Bereich)
- [x] 8.2 Auth-Context mit Access Token im Memory und automatischem Refresh implementieren
- [x] 8.3 Axios-Instanz mit Auth-Interceptor (Token anhängen, 401 → Refresh → Retry)
- [x] 8.4 Login-Seite (E-Mail + Passwort, Fehleranzeige)
- [x] 8.5 Beitrittsantrag-Seite (öffentlich: Name, E-Mail, Mannschaft auswählen, absenden)
- [x] 8.6 Registrierungs-Seite (aus Einladungs-/Genehmigungslink: Passwort setzen)
- [x] 8.7 Passwort-vergessen-Seite und Passwort-reset-Seite
- [x] 8.7 App-Shell: Navigation (Sidebar/Header), Logout-Button, aktiver Nutzer anzeigen

## 9. Frontend: CORE

- [x] 9.1 Vereinseinstellungen-Seite (Vereinsname, Saisons, Teams) — Admin only
- [x] 9.2 Team-Verwaltungsseite: Teams anlegen, Teamleiter/Trainer zuweisen
- [x] 9.3 Nutzerverwaltungsseite: Einladungen versenden, Rollen anzeigen

## 10. Frontend: MEMBERS

- [x] 10.1 Mitgliederliste mit Suche und Team-Filter
- [x] 10.2 Mitglied-Detailseite: Stammdaten, Status, Mannschaftszugehörigkeit
- [x] 10.3 Mitglied anlegen/bearbeiten Formular
- [x] 10.4 Familien-Verlinkung: Elternteil → Spielerprofil zuweisen (Admin)
- [x] 10.5 Eigenes Profil: Elternteil/Spieler sieht nur verknüpfte Kinder / eigenes Profil
- [x] 10.6 CSV-Export-Button (Admin)

## 11. Frontend: DUTIES

- [x] 11.1 Dienstbörse-Seite: offene Slots, nach Datum sortiert, mit Claim-Button
- [x] 11.2 Diensttypen-Verwaltung (Admin): anlegen, bearbeiten
- [x] 11.3 Dienst-Slots-Verwaltung (Admin/Teamleiter): anlegen, bearbeiten, Erfüllung bestätigen
- [x] 11.4 Dienstkonten-Übersicht: eigenes Konto (Elternteil/Spieler), alle Konten (Admin)
- [x] 11.5 Geldersatz-Erfassung Formular (Admin)
- [x] 11.6 CSV-Export Dienstkonten (Admin)

## 12. embed.FS Integration & Deployment

- [x] 12.1 `go:embed web/dist` Directive in Go-Binary einbauen, statische Files servieren
- [x] 12.2 SPA-Fallback: alle nicht-API-Routes liefern `index.html`
- [x] 12.3 `make build` in Makefile: erst `npm run build`, dann `go build` mit eingebettetem Frontend
- [x] 12.4 `make deploy` in Makefile: Binary via rsync übertragen, systemd-Service restarten, Migrationen ausführen
- [x] 12.5 Scheduler-Subcommand implementieren (`./vereinswerk scheduler:run`) — Einladungstoken bereinigen
- [ ] 12.6 Cronjob auf VPS einrichten: `* * * * * /usr/local/bin/vereinswerk scheduler:run` — **PENDING: VPS nicht initialisiert**

## 13. TYPO3-Integration

- [x] 13.1 In `team-stuttgart-site/Resources/Private/Layouts/Page.html` Link/Button „Mitgliederbereich" auf `https://intern.team-stuttgart.org` ergänzen
- [ ] 13.2 `make sync-ext` ausführen und Änderung lokal prüfen — **PENDING: VPS DNS nicht konfiguriert**
- [ ] 13.3 Änderung auf Mittwald deployen (`make push`) — **PENDING: VPS DNS nicht konfiguriert**
