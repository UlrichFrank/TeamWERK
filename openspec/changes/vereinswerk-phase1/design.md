## Context

Die bestehende Infrastruktur besteht aus einer TYPO3 14 Site auf Mittwald Webhosting L (PHP-only, kein persistenter Prozess) und einem IONOS VPS Linux XS (1 vCore, 1 GB RAM). Der VPS ist aktuell ungenutzt. Vereinsverwaltung läuft über manuelle Kanäle (Excel, WhatsApp).

VereinsWerk soll als eigenständige App auf dem VPS laufen — vollständig getrennt von der TYPO3-Site, die öffentlich bleibt und unverändert betrieben wird. Ressourceneffizienz ist eine harte Constraint: auf 1 GB RAM muss das gesamte System (OS, Webserver, App, DB) laufen.

## Goals / Non-Goals

**Goals:**
- Go + Chi Single Binary, die React-Build einbettet (`embed.FS`) — ein Prozess, ein Deployment-Artifact
- SQLite als Datenbank (in-process, kein separater DB-Prozess)
- JWT-Auth komplett unabhängig von TYPO3
- Drei Module Phase 1: CORE, MEMBERS, DUTIES
- HTTPS via Nginx + Let's Encrypt auf dem VPS
- Deployment: `git pull && go build && systemctl restart vereinswerk`

**Non-Goals:**
- SSO mit TYPO3 `fe_users` (zu komplex für Phase 1)
- WebSockets / Echtzeit-Features (nicht nötig für CORE/MEMBERS/DUTIES)
- Redis / Message Queue (Cronjob-Workaround reicht für MVP)
- Mobile Native App
- Multi-Tenancy (ein Verein, ein System)

## Decisions

### D1: Go + Chi statt Laravel/PHP

**Entscheidung:** Go mit Chi-Router als Backend.

**Warum:** PHP-FPM + MySQL auf 1 GB RAM verbraucht ~370 MB. Go-Binary + SQLite verbraucht ~50-70 MB — Faktor 5-7 effizienter. Außerdem: Single-Binary-Deployment ohne Dependency-Management auf dem Server.

**Alternativen erwogen:**
- Laravel/PHP: zu speicherintensiv für diesen VPS, Extbase-Boilerplate aufwändig
- Node.js: auf Mittwald Webhosting L nicht möglich (irrelevant für VPS, aber Go ist effizienter)

### D2: React + Tailwind als Frontend, via embed.FS ausgeliefert

**Entscheidung:** Vite baut React + Tailwind als statische Dateien, Go bettet das Build-Verzeichnis via `embed.FS` ein und liefert es aus.

**Warum:** Kein separater Node-Prozess oder CDN nötig. Ein Go-Binary ist das gesamte System. Entwicklung lokal mit Vite Dev Server + Proxy auf den Go-API-Port.

**Alternativen erwogen:**
- Inertia.js: Laravel-spezifisch, fällt weg mit Go
- Go HTML-Templates + htmx: einfacher, aber User will React-Erfahrung sammeln
- Separate Deployments (Go API + React auf CDN): mehr Infrastruktur, nicht nötig für MVP

### D3: SQLite statt PostgreSQL

**Entscheidung:** SQLite als primäre Datenbank, Zugriff via `sqlc` + `modernc.org/sqlite` (pure Go, kein CGo).

**Warum:** Kein separater DB-Prozess, kein RAM-Overhead (~200 MB MySQL entfallen). Für ~150 Vereinsmitglieder und niedrige parallele Schreiblast ist SQLite vollständig ausreichend. Migrations via `golang-migrate`.

**Alternativen erwogen:**
- PostgreSQL: korrekter für Produktion mit vielen parallelen Writes, aber hier unnötig
- GORM: zu viel Magic, sqlc gibt typsichere Queries aus SQL-Definitionen

**Migration zu PostgreSQL später:** sqlc und golang-migrate unterstützen beide PostgreSQL — der Wechsel ist möglich ohne Anwendungslogik zu ändern.

### D4: JWT-Auth (stateless) statt Sessions

**Entscheidung:** JWT-Tokens (Access Token 15 min, Refresh Token 7 Tage in HttpOnly Cookie).

**Warum:** Kein Redis / Session-Store nötig. Passt zur Single-Binary-Philosophie. React SPA verwaltet Access Token im Memory (nicht localStorage — XSS-Schutz).

**Alternativen erwogen:**
- Server-Side Sessions mit SQLite-Store: würde funktionieren, aber JWT ist Standard für React SPAs

### D5: Nginx als Reverse Proxy (nicht Go direkt auf Port 80/443)

**Entscheidung:** Nginx terminiert SSL, leitet zu Go auf Port 8080 weiter.

**Warum:** SSL-Zertifikat-Management via Certbot ist mit Nginx einfacher. Go auf privilegierten Ports laufen lassen ist ein Security-Anti-Pattern.

### D6: Rollenmodell in JWT-Claims, nicht DB-Lookup per Request

**Entscheidung:** Rolle wird beim Login in JWT-Claim eingebettet. Rollenwechsel erfordern Re-Login.

**Rollen:** `admin`, `trainer`, `elternteil`, `spieler` (kein separater `teamleiter` — Trainer übernimmt Teamverwaltung).

**Warum:** Kein DB-Lookup bei jedem Request. Ausreichend für einen Verein, wo Rollenänderungen selten sind.

### D7: SMTP via Mittwald-Mailaccount

**Entscheidung:** E-Mail-Versand über den bestehenden Mittwald-Mailaccount (`vorstand@team-stuttgart.org`) via SMTP. Zugangsdaten in `.env` auf dem VPS.

**Warum:** Kein zusätzlicher externer Dienst (Postmark, Resend), keine extra Kosten. Mittwald stellt SMTP-Zugang für gehostete Domains bereit.

**Alternativen erwogen:**
- Postmark/Resend: bessere Zustellbarkeit und Tracking, aber Mehrkosten und weiterer Account
- Eigener Postfix auf VPS: zu viel Ops-Aufwand, Spam-Risiko ohne SPF/DKIM-Setup

### D8: Zweistufiger Einladungsflow

**Entscheidung:** Registrierung erfolgt über zwei Wege — Eigenantrag mit Genehmigung oder Direkteinladung durch Berechtigte.

**Weg A (Eigenantrag):** Nutzer stellt Anfrage für eine Mannschaft → Admin, Trainer oder Vorstand genehmigt oder lehnt ab → bei Genehmigung erhält der Nutzer einen Registrierungslink.

**Weg B (Direkteinladung):** Admin, Trainer oder Vorstand lädt eine E-Mail-Adresse direkt ein → Nutzer erhält sofort den Registrierungslink.

**Warum:** Reduziert Admin-Aufwand (Trainer können ihre eigene Mannschaft selbst einladen), verhindert aber unkontrollierte Selbstregistrierung.

## Risks / Trade-offs

**SQLite Write-Concurrency** → Mehrere gleichzeitige Schreibvorgänge (z.B. 20 Eltern melden sich gleichzeitig für Dienste an) können zu Lock-Timeouts führen.
Mitigation: WAL-Mode aktivieren (`PRAGMA journal_mode=WAL`) — erlaubt parallele Reads während eines Writes.

**1 GB RAM — kein Headroom für Spitzen** → Bei unerwartet hoher Last (z.B. Turnier mit vielen gleichzeitigen Zugriffen) könnte OOM auftreten.
Mitigation: Go ist sehr speichereffizient. Monitoring via `htop`. Bei Bedarf VPS-Upgrade auf XS+ (~2 GB).

**Kein Background Job Worker** → E-Mail-Erinnerungen und Scheduler-Tasks laufen via Cronjob (`php artisan schedule:run`-Äquivalent: `./vereinswerk scheduler`). Latenz bis zu 1 Minute, kein Retry bei Fehlern.
Mitigation: Für MVP (Dienst-Erinnerungen, Lizenz-Checks) ist 1-Minuten-Latenz akzeptabel. Fehler werden geloggt.

**JWT-Invalidierung** → Kompromittierte Tokens sind bis zum Ablauf (15 min) gültig. Kein Token-Blacklisting ohne Redis.
Mitigation: Kurze Access-Token-Lebensdauer (15 min). Refresh Token in HttpOnly Cookie ist schwerer zu stehlen.

**Kein SSO mit TYPO3** → Nutzer müssen sich separat in VereinsWerk einloggen. Zwei Passwörter möglich.
Mitigation: Phase 1 Akzeptanz. In Phase 2 kann TYPO3-Button auf VereinsWerk-Login mit "Eingeloggt bleiben" (langer Refresh Token) den Friction reduzieren. Echtes SSO ist Phase-3-Thema.

## Migration Plan

1. IONOS VPS: Ubuntu 24.04, Nginx installieren, Certbot einrichten
2. Go-Build-Pipeline: `Makefile` mit `make build`, `make deploy` (rsync Binary + restart systemd)
3. Subdomain `intern.team-stuttgart.org` auf VPS-IP zeigen (DNS-Eintrag)
4. SSL-Zertifikat via Certbot für `intern.team-stuttgart.org`
5. systemd-Service `vereinswerk.service` für das Go-Binary
6. SQLite-Datenbankdatei in `/var/lib/vereinswerk/vereinswerk.db` (außerhalb des Binary-Pfads)
7. TYPO3: Link in `Page.html` auf `https://intern.team-stuttgart.org` ergänzen

**Rollback:** systemd-Service stoppen, altes Binary wiederherstellen (Binary-Versionierung im Deployment).

## Open Questions

Alle initialen offenen Fragen sind geklärt (D7, D8, Subdomain). Keine offenen Punkte.
