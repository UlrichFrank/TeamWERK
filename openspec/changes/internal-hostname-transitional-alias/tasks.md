## 1. Backend — Config & Scheduler

- [ ] 1.1 `internal/config/config.go`: Default `BaseURL` von `https://internal.team-stuttgart.org` auf `https://teamwerk.team-stuttgart.org` umstellen (Zeile 85).
- [ ] 1.2 `internal/config/config_test.go` (neu oder erweitern): `TestConfig_BaseURLDefault` — ohne gesetzte `BASE_URL`-Env liefert `Load()` `BaseURL == "https://teamwerk.team-stuttgart.org"`.
- [ ] 1.3 `internal/scheduler/scheduler.go:840`: hartkodierte URL `https://internal.team-stuttgart.org/duty-board` durch `<BaseURL>/duty-board` ersetzen. `BaseURL` kommt aus dem bereits vorhandenen `cfg` im Scheduler (Konstruktor prüfen).
- [ ] 1.4 `internal/scheduler/scheduler_test.go` (erweitern): `TestScheduler_DutyReminder_UsesConfigBaseURL` — mit `BaseURL="https://example.test"` läuft der Reminder-Generator, resultierender Body enthält `https://example.test/duty-board`, keine `internal.*`-URL mehr.
- [ ] 1.5 `/usr/local/go/bin/go test ./internal/config/... ./internal/scheduler/...` grün.

## 2. Frontend — Transitional-Banner

- [ ] 2.1 Neue Komponente `web/src/components/TransitionalHostnameBanner.tsx`. Rendert `null`, wenn `window.location.host !== "internal.team-stuttgart.org"`. Sonst: Sticky-Top-Bar mit `bg-brand-info/10 border-b border-brand-info/30 text-brand-text text-sm`, Text „Wir sind umgezogen. Öffne **teamwerk.team-stuttgart.org**, installiere die PWA neu und logge dich einmal wieder ein.", primärer Button-Link (`Button Primary`-Klassen) auf `https://teamwerk.team-stuttgart.org${window.location.pathname}${window.location.search}`. Nicht dismissable. Icon `<AlertTriangle>` (lucide-react, `w-5 h-5`).
- [ ] 2.2 `web/src/components/AppShell.tsx`: `<TransitionalHostnameBanner />` **oberhalb** des Headers mounten (vor allen anderen Layout-Elementen, damit er auch bei Skeleton-Load bereits sichtbar ist).
- [ ] 2.3 `web/src/components/TransitionalHostnameBanner.test.tsx`: Component-Tests (vitest + @testing-library/react):
      - stubbt `window.location` auf `internal.team-stuttgart.org` → Banner sichtbar, CTA-`href` enthält `teamwerk.team-stuttgart.org`.
      - stubbt auf `teamwerk.team-stuttgart.org` → Component rendert `null`.
      - stubbt auf `localhost` → Component rendert `null`.
      - CTA-`href` preserved `pathname` + `search`.
- [ ] 2.4 `pnpm -C web test` grün.

## 3. Nginx — Dual-Serving auf dem neuen VPS

- [ ] 3.1 `git mv deploy/nginx-intern.conf deploy/nginx-teamwerk.conf`.
- [ ] 3.2 In `deploy/nginx-teamwerk.conf`: HTTP-Redirect-Block `server_name teamwerk.team-stuttgart.org internal.team-stuttgart.org;`. HTTPS-Block ebenso. Cert-Pfade auf `/etc/letsencrypt/live/teamwerk.team-stuttgart.org/{fullchain,privkey}.pem` setzen.
- [ ] 3.3 `deploy/setup-vps.sh`: Kommentar (Zeile 85) und Datei-Referenz (`nginx-intern.conf` → `nginx-teamwerk.conf`) aktualisieren.
- [ ] 3.4 Konfig lokal prüfen: `nginx -t -c deploy/nginx-teamwerk.conf` (soweit lokal möglich; Sanity-Check ohne echte Cert-Pfade).

## 4. Runbook + Betriebsschritte

Detaillierter phasenweiser Ablauf mit Rollback-Pfad steht in
[`deploy/internal-alias-cutover-runbook.md`](../../../deploy/internal-alias-cutover-runbook.md).
Diese Task-Liste ist die verkürzte Checkliste zum Abhaken während der
Umschaltung.

- [ ] 4.0 Runbook `deploy/internal-alias-cutover-runbook.md` reviewen und die
      TTL für `internal.team-stuttgart.org` im Mittwald-Panel **≥ 24 h vor
      dem Cutover** auf 300 s reduzieren.
- [ ] 4.1 **Phase A** — Code + Nginx (mit `server_name teamwerk.*` only,
      internal.* kommt in Phase D dazu) auf Neu-Host deployen; `BASE_URL` in
      `/etc/teamwerk/env` prüfen; Smoke-Test `teamwerk.*/api/healthz`.
- [ ] 4.2 **Phase B** — DNS Mittwald: `internal.team-stuttgart.org A → 31.70.110.19`. Propagation abwarten (`dig +short` liefert Neu-IP von zwei Resolvern).
- [ ] 4.3 **Phase C** — Cert erweitern: `certbot --nginx --expand -d teamwerk.team-stuttgart.org -d internal.team-stuttgart.org`. `certbot certificates` listet **ein** Zertifikat mit beiden SANs.
- [ ] 4.4 **Phase D** — Nginx-Config mit doppeltem `server_name` deployen, `nginx -t && reload`. Smoke-Test:
       - `https://teamwerk.team-stuttgart.org` → App, **kein** Banner.
       - `https://internal.team-stuttgart.org` → App, **Banner sichtbar**, CTA-Link stimmt.
       - `curl -sSI https://internal.team-stuttgart.org/api/healthz | head -1` → `HTTP/2 200` (nicht 301).
- [ ] 4.5 **Phase E** — Alt-VPS: `teamwerk`-Service und `nginx` stoppen und disablen, Scheduler-Cron entfernen. `curl` gegen Alt-IP → connection refused. VPS bleibt als Rollback-Reserve stehen, wird nicht gekündigt.

## 5. Doku

- [ ] 5.1 `docs/agent/01-overview.md`: URL-Referenz `https://internal.team-stuttgart.org` → `https://teamwerk.team-stuttgart.org` (Zeile 4). Kurzer Zusatz: „`internal.team-stuttgart.org` bleibt als Übergangs-Alias erreichbar; ein späterer Flip auf 301 ist möglich, aber nicht datiert."
- [ ] 5.2 `docs/agent/10-deployment.md`: URL-Referenzen aktualisieren, Verweis auf diese Change im Kontext „Dual-Serving-Übergang".
- [ ] 5.3 `docs/monitoring.md`: `BASE`/`HOST`-Beispielwerte, Prometheus-Target und curl-Beispiele auf `teamwerk.*`. Übergangs-Alias erwähnen (Monitoring pingt nur den Primärhost).
- [ ] 5.4 `web/public/benutzerhandbuch.html`: Chip-Text (Zeile 353) auf `teamwerk.team-stuttgart.org` aktualisieren.
- [ ] 5.5 `web/public/CHANGELOG.md`: neuer Eintrag „umzug auf teamwerk.team-stuttgart.org, internal.* bleibt als Alias erreichbar".
- [ ] 5.6 `docs/security/audit-2026-06-26.md`: **nicht anfassen** — historisches Audit-Dokument, `internal.*` beschreibt den damaligen Stand.

## 6. Follow-up (nicht Teil dieser Change, kein Datum, im Runbook als Option vermerken)

Der Flip von `internal.*` auf 301 kommt irgendwann — Zeitpunkt offen, Betreuerentscheidung (siehe design.md „Entscheidung 5"). Dieser Task-Block dokumentiert nur den mechanischen Ablauf für den späteren Follow-up-Change, damit er nicht neu recherchiert werden muss.

- [ ] 6.1 Neuer OpenSpec-Change `internal-hostname-hard-redirect`: `nginx-teamwerk.conf`-Block auf `server_name teamwerk.*;` reduzieren, separaten Redirect-Block `server_name internal.*; return 301 https://teamwerk.team-stuttgart.org$request_uri;` hinzufügen, `internal.*`-SAN im Cert behalten (Certbot renewals würden ihn sonst beim nächsten Cycle wegwerfen).
- [ ] 6.2 Danach: `TransitionalHostnameBanner` samt Mount und Test löschen.

## 7. Verifikation

- [ ] 7.1 `openspec validate internal-hostname-transitional-alias`.
- [ ] 7.2 `make test lint` grün.
- [ ] 7.3 `/verify-change` durchlaufen lassen.
