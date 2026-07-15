## Why

Die Produktions-Instanz zieht auf `teamwerk.team-stuttgart.org`; der bisherige Hostname `internal.team-stuttgart.org` soll **weiter erreichbar** sein, damit Bestandsnutzer, alte Bookmarks, ausgelieferte Passwort-Reset-/Einladungs-Mails und PWA-Homescreen-Icons nicht zeitgleich brechen. Die naheliegende Lösung — der bereits vorbereitete harte 301 auf dem Alt-VPS (`deploy/nginx-redirect.conf`, `make server-cutover`) — bindet den alten IONOS-Host dauerhaft als Redirect-Server. Das ist unnötige Betriebslast (Systemd, Nginx, Certbot-Renewal auf einer zweiten Maschine) und verhindert die Abschaltung.

Zweite Reibung: ein hartes 301 auf einen anderen Origin **loggt alle Bestandsnutzer synchron aus** (Refresh-Token-Cookie ist an `internal.*` gebunden, kommt bei `teamwerk.*` nicht mit) — kein Datenverlust, aber Support-Wochenende. Ein sanfter Übergang mit **beiden Hostnames am selben (neuen) Host** und einem UI-Banner verteilt den Login-Refresh auf die Bestandsnutzer selbst.

## What Changes

- **DNS** (Mittwald, manuell): `internal.team-stuttgart.org  A → 31.70.110.19` (neue VPS-IP). Alt-Host kann anschließend abgeschaltet werden.
- **Nginx auf dem neuen VPS** serviert beide Hostnames aus **einem** `server`-Block (`server_name teamwerk.team-stuttgart.org internal.team-stuttgart.org`). Ein Zertifikat mit beiden SANs via Certbot. Konfig-Datei umbenannt: `deploy/nginx-intern.conf` → `deploy/nginx-teamwerk.conf`.
- **Backend-Code**: die hartkodierte URL `https://internal.team-stuttgart.org/duty-board` in `internal/scheduler/scheduler.go` (Reminder-Mail-Body) wird auf `cfg.BaseURL + "/duty-board"` refactored. Default von `BaseURL` in `internal/config/config.go` wird auf `https://teamwerk.team-stuttgart.org` gesetzt. `.env` auf VPS: `BASE_URL=https://teamwerk.team-stuttgart.org`.
- **Frontend-Banner**: neuer persistenter Banner im `AppShell`, der **nur** auf `window.location.host === "internal.team-stuttgart.org"` sichtbar ist. Text: „Wir sind umgezogen. Bitte öffne **teamwerk.team-stuttgart.org**, installiere die PWA dort neu und logge dich einmal wieder ein." + primärer CTA-Button (Link auf `https://teamwerk.team-stuttgart.org$path`). Nicht dismissable — bleibt bis DNS/Origin gewechselt ist.
- **Nicht Teil dieser Änderung**:
  - Endgültiges Flippen von `internal.*` auf 301 → `teamwerk.*`. Kommt irgendwann als eigener Follow-up-Change, Zeitpunkt ist bewusst offen (siehe design.md „Entscheidung 5"). Der Dual-Serving-Zustand ist tragbarer Dauerbetrieb, kein Provisorium mit Ablaufdatum.
  - Migration der Bestandsdaten oder VPS-Wechsel selbst (siehe archivierte Change `2026-07-03-server-migration` bzw. `deploy/server-migration-runbook.md` — das läuft parallel und ist Voraussetzung).
  - Automatische Abmeldung der `internal.*`-Push-Subscriptions (bleiben liegen, bereinigen sich beim PWA-Neu-Install oder via HTTP-410 aus dem Endpoint).
  - Multi-Origin-CORS-Whitelist (Details im Design; same-origin-Flow ist ausreichend).

## Capabilities

### Modified Capabilities
- `vps-deployment`: HTTPS-Zugang serviert Primärhostname `teamwerk.team-stuttgart.org` und Übergangs-Alias `internal.team-stuttgart.org` aus derselben Instanz.

### New Capabilities
- (keine — Übergangs-Banner und Alias sind Erweiterung an `vps-deployment`, nicht eigene Capability.)

## Impact

- **`internal/config/config.go`**: 1-Zeilen-Änderung (Default `BaseURL`).
- **`internal/scheduler/scheduler.go`**: 1-Zeilen-Änderung + 1 Testcase (URL kommt aus `cfg.BaseURL`).
- **`web/src/components/AppShell.tsx`** (oder neuer Komponent `web/src/components/TransitionalHostnameBanner.tsx`): Banner-Komponente + Mount.
- **`deploy/nginx-intern.conf` → `deploy/nginx-teamwerk.conf`**: umbenannt, `server_name` erweitert, Cert-Pfade auf `teamwerk.*`.
- **`deploy/internal-alias-cutover-runbook.md`** (neu): Phasen-orientierter Deploy-/Cutover-Ablauf für 1b (Dual-Serving) — ersetzt die Kap. 3+4 aus `server-migration-runbook.md` für den konkreten Umzug, referenziert Kap. 0–2 und 5 wieder.
- **`deploy/setup-vps.sh`**: Kommentar + kopierte Datei-Referenz.
- **`docs/agent/01-overview.md`**, **`docs/monitoring.md`**, **`docs/agent/10-deployment.md`**: URL-Referenzen aktualisiert; Übergangs-Alias erwähnt.
- **`web/public/benutzerhandbuch.html`**, **`web/public/CHANGELOG.md`**: URL-Referenzen aktualisiert.
- **CSP / CORS / Cookie-Domain**: unverändert. `Access-Control-Allow-Origin` bleibt exakt auf `BaseURL` (teamwerk.*). Same-origin-Requests von internal.* → internal.* durchlaufen ohne CORS-Prüfung; Cross-Origin von internal.* → teamwerk.* ist kein Anwendungsfall. Refresh-Token-Cookies bleiben an ihrer jeweiligen Origin gebunden — dokumentierter Trade-off, den der Banner adressiert.
- **Betrieb**: Alt-VPS (IONOS `217.160.118.39`) kann nach DNS-TTL abgeschaltet werden — kein Redirect-Server nötig. Certbot-Renewal auf neuem VPS deckt via SAN beide Hostnames.
- **Sicherheit**: keine Änderung am Auth-, Krypto- oder Rate-Limit-Modell.

## Test-Anforderungen

| Route / Ort | Testname | Erwarteter Status / Invariante |
|---|---|---|
| `scheduler` reminder-mail | `TestScheduler_DutyReminder_UsesConfigBaseURL` | Mail-Body enthält `cfg.BaseURL + "/duty-board"`, nicht die frühere hartkodierte URL. Getestet mit `BaseURL="https://example.test"`. |
| `internal/config/config.go` | `TestConfig_BaseURLDefault` | Ohne gesetzte `BASE_URL`-Env liefert `Load()` `BaseURL == "https://teamwerk.team-stuttgart.org"`. |
| Frontend `TransitionalHostnameBanner` | Unit-/Component-Test | Rendert sichtbar bei `window.location.host === "internal.team-stuttgart.org"`, rendert `null` bei allen anderen Hosts (u. a. `teamwerk.team-stuttgart.org` und `localhost`). |
| Frontend Banner-CTA | Unit-/Component-Test | CTA-Link zeigt auf `https://teamwerk.team-stuttgart.org` + `window.location.pathname + window.location.search`. |

Garantierte fachliche Invariante: **Nach der Bestandsnutzung von `internal.*` verweist keine vom Backend generierte Direktlink-URL mehr auf `internal.team-stuttgart.org`**, und **jeder Aufruf an `internal.*` zeigt einen sichtbaren Migrations-Hinweis**, ohne dass die App unbenutzbar wird.
