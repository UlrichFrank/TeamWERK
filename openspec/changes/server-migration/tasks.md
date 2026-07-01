## 1. Grundgerüst: Redirect-Config, `.env.example`, Runbook-Skelett

- [x] 1.1 `deploy/nginx-redirect.conf` schreiben: `server { listen 80/443; server_name intern.team-stuttgart.org; add_header Cache-Control "no-store"; return 301 https://{{NEW_DOMAIN}}$request_uri; }` inkl. TLS-Zertifikatspfaden (bestehende Certbot-Ausstellung wiederverwenden) und Kommentar, dass `{{NEW_DOMAIN}}` beim Deploy per `sed` ersetzt wird.
- [x] 1.2 `.env.example` um zwei auskommentierte Zeilen ergänzen: `# REMOTE_NEW=vServerNeu` und `# REMOTE_NEW_DIR=/usr/local/bin` mit Kommentar „nur während Server-Umzug setzen".
- [x] 1.3 `deploy/server-migration-runbook.md` als Datei anlegen mit sieben Abschnitten (Vorbereitung / Bootstrap / Testphase / DNS+Certbot / Cutover / Nachpflege / Rollback), zunächst als Skelett; wird in Task 6 gefüllt.

## 2. Makefile: gemeinsame Bausteine

- [x] 2.1 In `Makefile` oben bei den bestehenden Variablen `REMOTE_NEW`, `REMOTE_NEW_DIR` und `BASE_URL_NEW` per `grep '^…=' .env | cut -d= -f2-` einlesen; Standard für `REMOTE_NEW_DIR` bleibt leer → im Target auf `/usr/local/bin` fallen. `BASE_URL_NEW` behält Schema (`https://…`) und wird nirgends per CLI überschrieben.
- [x] 2.2 In `Makefile` interne Hilfsvariablen definieren: `NEW_REMOTE_RESOLVED := $(or $(NEW_REMOTE),$(REMOTE_NEW))` und `NEW_DOMAIN := $(patsubst https://%,%,$(BASE_URL_NEW))` (Domain ohne Schema, für Host-Header). Drei Prüf-Targets bauen: `_check-remote` (Quelle), `_check-new-remote` (Ziel-SSH-Alias), `_check-base-url-new` (`BASE_URL_NEW` in `.env` und mit `https://`-Präfix). Jedes bricht bei fehlendem Wert mit klarer Fehlermeldung inkl. `.env`-Beispiel ab.
- [x] 2.3 Alle drei Migrations-Targets als `.PHONY` deklarieren und im `help`-Kommentar erklären (`## Server-Umzug: initialen Zielhost aufsetzen` / `Testdaten überschreiben` / `Alt-Host auf 301 umschalten`).

## 3. `server-bootstrap`

- [x] 3.1 Target-Signatur: `server-bootstrap: _check-remote _check-new-remote _check-base-url-new build`. Kein Fetch vor Prüfungen.
- [x] 3.2 Schritt A: `setup-vps` auf Ziel — `rsync -az deploy/ $(NEW_REMOTE_RESOLVED):/tmp/teamwerk-deploy/` und `ssh $(NEW_REMOTE_RESOLVED) "cd /tmp/teamwerk-deploy && sudo bash setup-vps.sh"`.
- [x] 3.3 (entfallen — Domain kommt aus `BASE_URL_NEW` via 2.2)
- [x] 3.4 Schritt C: Env-Klonen per Pipe `ssh $(REMOTE) "sudo cat /etc/teamwerk/env" | sed -E "s|^BASE_URL=.*|BASE_URL=$(BASE_URL_NEW)|" | ssh $(NEW_REMOTE_RESOLVED) "sudo tee /etc/teamwerk/env > /dev/null && sudo chmod 600 /etc/teamwerk/env"` — keine lokale Zwischendatei.
- [x] 3.5 Schritt D: Better-Stack-Config-Dateien klonen (`heartbeat-url`, `betterstack-logs-token`, `betterstack-metrics-token`, `betterstack-metrics-endpoint`) per gleiches Pipe-Muster mit `chmod 600`.
- [x] 3.6 Schritt E: Zielhost-Service (falls existent) stoppen `ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl stop teamwerk 2>/dev/null || true"`.
- [x] 3.7 Schritt F: DB-Snapshot — `ssh $(REMOTE) "sqlite3 $(DB_PATH) '.backup /tmp/teamwerk-migration.db'"`, dann `ssh $(REMOTE) "sudo cat /tmp/teamwerk-migration.db" | ssh $(NEW_REMOTE_RESOLVED) "sudo tee /var/lib/teamwerk/teamwerk.db > /dev/null"` und `rm -f /tmp/teamwerk-migration.db` auf der Quelle. Alternative: `scp` via `-3` (Direktkopie zwischen Remotes) — bevorzugt, wenn SSH-Konfigurations-Setup es zulässt.
- [x] 3.8 Schritt G: Storage-Ordner per rsync direkt zwischen den Remotes (`rsync -az -e ssh $(REMOTE):$(UPLOAD_DIR_REMOTE)/ $(NEW_REMOTE_RESOLVED):$(UPLOAD_DIR_REMOTE)/` — falls Remote-zu-Remote nicht funktioniert, Fallback über Entwickler-Disk mit Warnhinweis „~X GB werden transient auf laptop gepuffert"). Analog für `files`, `videos`, `beitragslauf-protokolle`.
- [x] 3.9 Schritt H: Owner-Fix auf Ziel `ssh $(NEW_REMOTE_RESOLVED) "sudo chown -R www-data:www-data /var/lib/teamwerk /storage"`.
- [x] 3.10 Schritt I: Binary deployen — bestehendes `deploy`-Target aufrufen, aber mit umgebogenem `REMOTE`: `$(MAKE) deploy REMOTE=$(NEW_REMOTE_RESOLVED) REMOTE_DIR=$(REMOTE_NEW_DIR)`. Das umfasst `migrate up` und Service-Start.
- [x] 3.11 Schritt J: Smoke-Test — `ssh $(NEW_REMOTE_RESOLVED) "curl -k -s -H 'Host: $(NEW_DOMAIN)' https://localhost/api/healthz"` (NEW_DOMAIN = BASE_URL_NEW ohne Schema, siehe 2.2) und prüfen, dass `"status":"ok"` und `"db":"ok"` enthalten sind. Bei Fail Exit 1 mit Response-Body.
- [x] 3.12 Schritt K: Grep auf Ziel-Env absetzen und Ausgabe zeigen: `ssh $(NEW_REMOTE_RESOLVED) "sudo grep '^BASE_URL=' /etc/teamwerk/env"` → sichtbare Bestätigung der `BASE_URL`.

## 4. `server-sync-data`

- [x] 4.1 Target-Signatur: `server-sync-data: _check-remote _check-new-remote _check-base-url-new build`.
- [x] 4.2 Bestätigungsdialog: `printf "server-sync-data überschreibt DB und Storage auf $(NEW_REMOTE_RESOLVED) mit einem frischen Snapshot von $(REMOTE). Testdaten auf Ziel gehen verloren. Fortfahren? [y/N] "; read ans; case "$$ans" in y|Y) ;; *) echo "Abgebrochen." ; exit 1;; esac`.
- [x] 4.3 Schritt A: Ziel-Service stoppen `ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl stop teamwerk"`.
- [x] 4.4 Schritt B: DB-Snapshot + Übertragung (dieselbe Sequenz wie 3.7).
- [x] 4.5 Schritt C: Storage-Ordner-Sync (dieselbe Sequenz wie 3.8).
- [x] 4.6 Schritt D: Owner-Fix (dieselbe Sequenz wie 3.9).
- [x] 4.7 Schritt E: `migrate up` auf Ziel — `ssh $(NEW_REMOTE_RESOLVED) "$(REMOTE_NEW_DIR)/$(BINARY) migrate up --db $(DB_PATH)"`.
- [x] 4.8 Schritt F: Ziel-Service starten `ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl start teamwerk"`.
- [x] 4.9 Schritt G: Smoke-Test wie 3.11.

## 5. `server-cutover`

- [x] 5.1 Target-Signatur: `server-cutover: _check-remote _check-new-remote _check-base-url-new`.
- [x] 5.2 Bestätigungsdialog mit deutlichem Warntext: „`server-cutover` stoppt teamwerk auf $(REMOTE) und schaltet den Alt-Host auf 301 → $(BASE_URL_NEW). Fortfahren? [y/N]". Abbruch wie 4.2.
- [x] 5.3 Schritt A: `$(MAKE) server-sync-data NEW_REMOTE=$(NEW_REMOTE_RESOLVED) MAKE_CONFIRMED=1`. Das Sub-Target liest `BASE_URL_NEW` selbstständig aus `.env`. Der Bestätigungsdialog wird via `MAKE_CONFIRMED=1` übersprungen (sauberer als `yes y |`-Pipe; auch in `server-sync-data` implementieren: `if [ "$$MAKE_CONFIRMED" = "1" ]; then ans=y; else printf …; read ans; fi`).
- [x] 5.4 Schritt B: Alt-Host-Service stoppen und disablen `ssh $(REMOTE) "sudo systemctl stop teamwerk && sudo systemctl disable teamwerk"`.
- [x] 5.5 Schritt C: Nginx-Config-Backup auf Alt-Host: `ssh $(REMOTE) "sudo cp /etc/nginx/sites-available/teamwerk /etc/nginx/sites-available/teamwerk.$(TS).bak"` (`TS` existiert schon als Makefile-Variable).
- [x] 5.6 Schritt D: Redirect-Config deployen: `sed "s|{{NEW_BASE_URL}}|$(BASE_URL_NEW)|g; s|{{NEW_DOMAIN}}|$(NEW_DOMAIN)|g" deploy/nginx-redirect.conf | ssh $(REMOTE) "sudo tee /etc/nginx/sites-available/teamwerk > /dev/null"`.
- [x] 5.7 Schritt E: `nginx -t` prüfen (`ssh $(REMOTE) "sudo nginx -t"`), bei Erfolg reloaden (`sudo systemctl reload nginx`); bei Fehler zurückrollen (Backup aus 5.5 wiederherstellen) und Exit 1.
- [x] 5.8 Schritt F: Alt-Host `curl -sI https://$(REMOTE_DOMAIN)/api/healthz`-Prüfung (soll `301` liefern, nicht `200`); wenn `200`, Fehlermeldung „Redirect nicht aktiv, prüfen".
- [x] 5.9 Schritt G: Abschluss-Ausgabe mit Nachpflege-Checkliste: Better-Stack-Monitor auf `$(BASE_URL_NEW)/api/healthz` umhängen, Better-Stack-Heartbeat-URL nichts ändern (dieselbe Datei auf Ziel), User informieren (Push-Broadcast oder Vorstandsansage) mit PWA-Neuinstallations-Hinweis.

## 6. Runbook (`deploy/server-migration-runbook.md`)

- [x] 6.1 Abschnitt „0. Vorbereitung": SSH-Alias für Ziel-VPS in `~/.ssh/config`, `REMOTE_NEW`, `REMOTE_NEW_DIR` und `BASE_URL_NEW` in `.env`, Ziel-VPS mit `setup-vps.sh` initial provisioniert (falls nicht: erledigt `server-bootstrap` mit).
- [x] 6.2 Abschnitt „1. Bootstrap": `make server-bootstrap NEW_REMOTE=…` (Domain aus `BASE_URL_NEW`), Ergebnis-Kontrollen (`curl` via IP+Host-Header, `journalctl -u teamwerk`).
- [x] 6.3 Abschnitt „2. Testphase": Lokale `/etc/hosts`-Zeile `31.70.110.19 teamwerk.team-stuttgart.org` empfehlen, damit nur der Betreuer auf den neuen Host läuft. Wiederholtes `make server-sync-data` mit erwartetem Datenverlust auf Ziel.
- [x] 6.4 Abschnitt „3. DNS + Certbot": DNS-A-Record im Provider-Panel; Wartezeit; `certbot --nginx -d teamwerk.team-stuttgart.org` auf Zielhost.
- [x] 6.5 Abschnitt „4. Cutover": `make server-cutover NEW_REMOTE=…` (Domain aus `BASE_URL_NEW`), Erwartungen, Verifikation (Alt-Host liefert 301).
- [x] 6.6 Abschnitt „5. Nachpflege": Better-Stack-HTTP-Monitor auf `https://teamwerk.team-stuttgart.org/api/healthz` umhängen; entscheiden, ob User-Broadcast per Push (`push_subscriptions` sind zwar veraltet, ein letzter Broadcast versucht es trotzdem → HTTP 410 räumt sie danach ab) oder per E-Mail; PWA-Neuinstallations-Hinweis, „alte Bookmarks per 301 umgeleitet, aber PWA-Homescreen-Icon zeigt weiter auf alt".
- [x] 6.7 Abschnitt „6. Wenn was schiefgeht" (Rollback): Nginx-Backup wiederherstellen (`sudo cp /etc/nginx/sites-available/teamwerk.<timestamp>.bak /etc/nginx/sites-available/teamwerk && sudo nginx -t && sudo systemctl reload nginx`), `sudo systemctl enable teamwerk && sudo systemctl start teamwerk`, DNS zurückstellen falls nötig, Better-Stack zurückhängen.
- [x] 6.8 Abschnitt „Was nicht automatisiert ist": Explizit auflisten (DNS-Wechsel, Certbot, Better-Stack-Monitor, User-Info, PWA-Neuinstallation).

## 7. Doku-Anpassung

- [x] 7.1 `docs/agent/10-deployment.md`: unter „Deployment & VPS" einen Absatz „Server-Umzug" ergänzen mit Verweis auf `deploy/server-migration-runbook.md` und Nennung der drei Targets — 3 Sätze reichen.
- [x] 7.2 `CLAUDE.md` selbst bleibt unverändert (der Verweis liegt im importierten `10-deployment.md`).

## 8. Cutover-Vorprüfung (kein dritter VPS verfügbar → Prüfung statt Trockenlauf)

Kein Wegwerf-VPS vorhanden, gegen den man den Ablauf komplett üben könnte.
`server-bootstrap` und `server-sync-data` sind nicht destruktiv gegen die
Produktion (nur der neue Host wird geschrieben) → der neue Server ist selbst
das Verifikations-System für diese beiden Targets. `server-cutover` läuft
zwangsläufig als „First-Time-Run gegen Prod"; die Sicherheiten sind
Nginx-Backup mit Timestamp, `nginx -t` vor Reload, dokumentiertes Rollback.
Diese Tasks verifizieren die riskanten Annahmen von `server-cutover`
**bevor** er läuft.

- [ ] 8.1 Alt-Host: `ssh $(REMOTE) "ls /etc/nginx/sites-available/"` — Dateiname der Prod-Config muss zu `SOURCE_DOMAIN` (= `BASE_URL` aus `.env` ohne `https://`-Präfix) passen; genau diese Datei überschreibt `server-cutover`. Bei Abweichung entweder `BASE_URL` in `.env` an den Dateinamen anpassen oder die Datei umbenennen — sonst schreibt der Cutover ins Leere.
- [ ] 8.2 Alt-Host: `ssh $(REMOTE) "sudo certbot certificates"` — Cert-Pfad `/etc/letsencrypt/live/<SOURCE_DOMAIN>/` muss existieren (die Redirect-Config referenziert `fullchain.pem` und `privkey.pem` unter diesem Pfad). Bei abweichendem Ordnernamen (z. B. `-0001`-Suffix) die Redirect-Config vor Deploy anpassen.
- [ ] 8.3 Redirect-Config lokal ausrendern und inspizieren: `sed "s|{{SOURCE_DOMAIN}}|<SOURCE_DOMAIN>|g; s|{{NEW_BASE_URL}}|<BASE_URL_NEW>|g" deploy/nginx-redirect.conf | less` — beide Platzhalter ersetzt, `server_name` sinnvoll, `return 301` zeigt auf neue Domain inkl. Schema, Cert-Pfade zeigen auf existierenden Cert. Diff gegen die aktuelle Prod-Config (`ssh $(REMOTE) "sudo cat /etc/nginx/sites-available/<SOURCE_DOMAIN>"`) — nur `return 301` + Cache-Control sollten neu sein, alles andere übernehmbar.
- [ ] 8.4 Extra DB-/Storage-Backup direkt vor dem Cutover: `make backup && make backup-files` (die eingebauten `.bak`-Snapshots von `server-cutover` sichern die Nginx-Config, nicht die DB). Backup-Ordner-Pfad notieren, damit Rollback ihn schnell findet.

## 9. Echter Umzug (führt der Betreuer manuell durch, mit Runbook)

- [ ] 9.1 `.env` um `REMOTE_NEW=vServerNeu` (SSH-Alias auf 31.70.110.19), `REMOTE_NEW_DIR=/usr/local/bin` und `BASE_URL_NEW=https://teamwerk.team-stuttgart.org` ergänzen.
- [ ] 9.2 `make server-bootstrap NEW_REMOTE=vServerNeu` ausführen.
- [ ] 9.3 Testphase mit lokaler `/etc/hosts`-Zeile, gemäß Runbook.
- [ ] 9.4 DNS-A-Record + Certbot auf Zielhost gemäß Runbook.
- [ ] 9.5 `make server-cutover NEW_REMOTE=vServerNeu` ausführen.
- [ ] 9.6 Better-Stack-HTTP-Monitor auf `https://teamwerk.team-stuttgart.org/api/healthz` umhängen.
- [ ] 9.7 Push-Broadcast oder E-Mail an alle Nutzer mit PWA-Neuinstallations-Hinweis.
