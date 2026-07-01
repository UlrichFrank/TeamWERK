# Server-Migrations-Runbook

Wiederkehrende Anleitung für den Umzug einer TeamWERK-Instanz von einem VPS
(**Quelle**, `REMOTE`) auf einen anderen VPS (**Ziel**, `REMOTE_NEW`) — konkret
z. B. für den Wechsel `intern.team-stuttgart.org` → `teamwerk.team-stuttgart.org`,
aber generisch verwendbar für Provider-Wechsel, Disaster-Recovery-Übung oder
Staging-Auffrischung.

Der Ablauf ist in drei Skript-Phasen und dazwischen manuelle Schritte
gegliedert. Alle Skript-Teile stecken in drei Makefile-Targets:

- `make server-bootstrap NEW_REMOTE=<alias>` — einmaliger Aufbau des Ziel-Hosts
- `make server-sync-data NEW_REMOTE=<alias>` — beliebig oft wiederholbarer DB-/Storage-Sync
- `make server-cutover NEW_REMOTE=<alias>` — finaler Umschalter (Alt-Host → 301)

---

## 0. Vorbereitung

**Ziel-VPS**: Ubuntu 24.04 LTS, mind. 1 GB RAM, ausreichend Disk (siehe
`deploy/vps-setup-runbook.md`, Storage-Erweiterung — 10 GB reichen NICHT für
produktiven Video-Betrieb), Root-SSH offen.

**Lokal (Entwickler-Laptop):**

1. SSH-Alias für Ziel-VPS in `~/.ssh/config`:
   ```
   Host vServerNeu
     HostName 31.70.110.19
     User root
     IdentityFile ~/.ssh/id_ed25519
   ```
   Test: `ssh vServerNeu "uname -a"`.

2. `.env` ergänzen um:
   ```
   REMOTE_NEW=vServerNeu
   REMOTE_NEW_DIR=/usr/local/bin
   BASE_URL_NEW=https://teamwerk.team-stuttgart.org
   ```
   `REMOTE`, `REMOTE_DIR`, `BASE_URL` (die Quelle) bleiben wie sie sind — sie
   werden bis zum Cutover weiter für Prod-Deploys genutzt.

**Ziel-VPS provisioniert?** Wenn nein, egal — `server-bootstrap` führt
`setup-vps.sh` idempotent mit aus.

---

## 1. Bootstrap (einmalig)

```bash
make server-bootstrap NEW_REMOTE=vServerNeu
```

Was passiert:

1. `setup-vps.sh` läuft auf Ziel (Nginx, systemd, Vector, Cron, Self-signed Cert).
2. `/etc/teamwerk/env` wird von Quelle nach Ziel geklont, `BASE_URL` auf
   `BASE_URL_NEW` umgeschrieben. Alle Secrets (`JWT_SECRET`, `VAPID_*`,
   `SMTP_*`, `VIDEO_STREAM_SECRET`, `METRICS_TOKEN`) bleiben identisch.
3. Better-Stack-Konfigurationsdateien werden mitkopiert (`heartbeat-url`,
   `betterstack-logs-token`, `betterstack-metrics-token`,
   `betterstack-metrics-endpoint`).
4. Ziel-Service wird gestoppt (falls bereits vorhanden).
5. `sqlite3 .backup` auf Quelle → Snapshot per SSH-Pipe nach Ziel-DB-Pfad.
6. Storage-Ordner (`uploads`, `files`, `videos`, `beitragslauf-protokolle`) via
   rsync direkt zwischen Remotes (Fallback: über Entwickler-Disk).
7. Owner-Fix `chown -R www-data:www-data` auf Ziel.
8. `make deploy` gegen Ziel-Host (Binary, `migrate up`, Service-Start).
9. Smoke-Test `/api/healthz` über IP + Host-Header.

**Ergebnis prüfen:**

```bash
# Health über IP + Host-Header (DNS ist noch nicht umgestellt)
ssh vServerNeu "curl -k -s -H 'Host: teamwerk.team-stuttgart.org' https://localhost/api/healthz"
# → {"status":"ok","db":"ok",...}

# Env-Rewrite
ssh vServerNeu "sudo grep '^BASE_URL=' /etc/teamwerk/env"
# → BASE_URL=https://teamwerk.team-stuttgart.org

# Journal
ssh vServerNeu "journalctl -u teamwerk -n 30 --no-pager"
```

---

## 2. Testphase (Tage bis Wochen)

Ziel: der Betreuer testet die neue Instanz gründlich, während die produktive
Nutzung noch auf der alten läuft. Voraussetzung: **normale Nutzer landen nicht
zufällig auf dem Ziel-Host** — sonst driften die DBs auseinander.

**Empfehlung: lokale `/etc/hosts`-Zeile beim Betreuer**, damit der Browser des
Betreuers auf die IP zeigt, aber sonst niemand:

```
31.70.110.19  teamwerk.team-stuttgart.org
```

DNS-A-Record im Provider-Panel **noch nicht** setzen. Kein Domain-Sharing mit
Kollegen.

**Beim Testen frische Produktions-Daten holen** (jederzeit wiederholbar, plättet
alle Testdaten auf dem Ziel):

```bash
make server-sync-data NEW_REMOTE=vServerNeu
# Bestätigungsdialog: „…überschreibt Testdaten…, ok? [y/N]"
```

**Was wird getestet:**

- Login mit realem Account (JWT bleibt gültig, weil `JWT_SECRET` identisch).
- **Bankdaten-Tresor** entsperren und einen Datensatz entschlüsseln. Wenn das
  klappt, sitzt die Zero-Knowledge-Verschlüsselung (`clubs.group_private_key_enc`
  ist byte-genau übertragen). Wenn nicht: **stopp**, nachprüfen ob der
  DB-Snapshot vollständig war und ob der Tresor mit derselben Passphrase
  aufgeht wie auf der Produktion.
- Fee-Run (SEPA-XML) im Browser durchspielen (nicht bestätigen).
- Kalender, Dienstbörse, Chat einmal laden.
- Ein Test-Push senden: `make push-test-remote USER=<id> TITLE=Test BODY=Testcut REMOTE=vServerNeu` (kommt beim User an? Bei alter PWA vermutlich **nicht**, weil dessen Subscription auf die alte Origin registriert ist — das ist erwartet und wird nach Cutover per HTTP-410-Cleanup aufgeräumt).

---

## 3. DNS-Umstellung und TLS

**Wenn Testphase abgeschlossen ist** — nicht früher, sonst laufen Nutzer schon
auf den Ziel-Host und ihre Änderungen gehen beim `server-sync-data` verloren.

1. Provider-Panel: A-Record `teamwerk.team-stuttgart.org` → `31.70.110.19`
   setzen (TTL kann klein bleiben, wenn Provider das erlaubt — 300s reichen).
2. Warten auf DNS-Propagation, in der Regel 15 min bis 1h.
   ```bash
   dig +short teamwerk.team-stuttgart.org
   # → 31.70.110.19
   ```
3. Certbot auf Ziel:
   ```bash
   ssh vServerNeu "certbot --nginx -d teamwerk.team-stuttgart.org --non-interactive --agree-tos -m vorstand@team-stuttgart.org"
   ```
4. Test:
   ```bash
   curl -sSf https://teamwerk.team-stuttgart.org/api/healthz
   # → {"status":"ok",…}
   ```

Die Instanz ist jetzt öffentlich unter der neuen Domain erreichbar. Die alte
Domain zeigt (bis zum Cutover) weiter die Produktion.

Optional: **lokale `/etc/hosts`-Zeile jetzt löschen**, damit der Betreuer auch
über echten DNS testet.

---

## 4. Cutover (finaler Umschalter)

**Vorprüfung** — kein Wegwerf-VPS für Trockenlauf verfügbar, deshalb vor dem
`server-cutover`-Lauf einmal manuell verifizieren:

```bash
# 4a. Nginx-Config-Dateiname auf Alt-Host — muss zu SOURCE_DOMAIN passen
ssh <alter-remote> "ls /etc/nginx/sites-available/"
# → intern.team-stuttgart.org (die Datei, die überschrieben wird)

# 4b. Cert-Pfad existiert
ssh <alter-remote> "sudo certbot certificates"
# → /etc/letsencrypt/live/intern.team-stuttgart.org/fullchain.pem
#   (bei -0001-Suffix vor Deploy die Redirect-Config anpassen)

# 4c. Redirect-Config lokal ausrendern und lesen
sed 's|{{SOURCE_DOMAIN}}|intern.team-stuttgart.org|g; s|{{NEW_BASE_URL}}|https://teamwerk.team-stuttgart.org|g' deploy/nginx-redirect.conf | less

# 4d. Extra DB-Backup direkt vor dem Umschalten
make backup && make backup-files
```

Erst wenn 4a–4d ok sind:

```bash
make server-cutover NEW_REMOTE=vServerNeu
```

Was passiert (nach Bestätigung `[y/N]`):

1. Frischer `server-sync-data`-Lauf (letzte Änderungen von Quelle → Ziel).
2. Alt-Host: `teamwerk`-Service stoppen und disable.
3. Alt-Host: Backup der Nginx-Config nach
   `/etc/nginx/sites-available/<domain>.<ISO-timestamp>.bak`.
4. Alt-Host: `deploy/nginx-redirect.conf` deployen (mit ersetzten Platzhaltern).
5. `nginx -t` + `systemctl reload nginx`.
6. Verifikation: `curl -sI` auf Alt-Host liefert `301` mit `Location:` neue Domain.

**Verifikation von außen:**

```bash
curl -sI https://intern.team-stuttgart.org/beliebiger/pfad
# → HTTP/2 301
# → location: https://teamwerk.team-stuttgart.org/beliebiger/pfad
# → cache-control: no-store

curl -sI https://intern.team-stuttgart.org/api/healthz
# → HTTP/2 301  (nicht 200 — sonst greift Redirect nicht für API)
```

---

## 5. Nachpflege

- **Better-Stack:**
  - HTTP-Monitor umhängen: URL `https://intern.team-stuttgart.org/api/healthz`
    → `https://teamwerk.team-stuttgart.org/api/healthz`. Erwartungs-Keyword
    `"db":"ok"` unverändert.
  - Heartbeat-URL: **nichts ändern** — dieselbe Datei `/etc/teamwerk/heartbeat-url`
    liegt auf dem Ziel und der Scheduler-Cron dort meldet sich beim selben
    Monitor.
  - Logs- und Metrics-Sources: **nichts ändern** — dieselben Tokens sind auf
    dem Ziel, dieselben Sources füllen sich mit dem neuen Hostname im `host`-Feld.
- **User-Kommunikation** (via bevorzugtem Kanal — Vorstandsansage oder
  Push-Broadcast von der neuen Domain aus):
  - „TeamWERK ist umgezogen: neue URL ist **teamwerk.team-stuttgart.org**."
  - „Bookmarks werden automatisch weitergeleitet."
  - „Wichtig für Smartphone-Nutzer mit App-Icon: die alte PWA im Homescreen
    zeigt weiter auf die alte URL. Bitte **die alte PWA löschen**, die neue
    URL im Browser öffnen und **‚Zum Homescreen hinzufügen'** erneut. Danach
    Push-Benachrichtigungen neu erlauben."
- **Push-Subscriptions:** die alten Endpoints in `push_subscriptions` sind auf
  die alte Origin registriert und werden beim nächsten Sende-Versuch mit
  HTTP 410 abgelehnt — das eingebaute Cleanup in `internal/notifications`
  räumt sie automatisch ab. Alternativ manuell per SQL leeren, wenn der
  Log-Spam stört.

---

## 6. Wenn was schiefgeht (Rollback)

Wenn nach dem Cutover ein Problem auftritt, das nicht schnell zu fixen ist:

1. **Alt-Host: Nginx-Config zurücksichern**
   ```bash
   ssh <alter-remote> "ls /etc/nginx/sites-available/ | grep .bak"
   # → intern.team-stuttgart.org.2026-…-…-…-….bak
   ssh <alter-remote> "sudo cp /etc/nginx/sites-available/<domain>.<timestamp>.bak /etc/nginx/sites-available/<domain>"
   ssh <alter-remote> "sudo nginx -t && sudo systemctl reload nginx"
   ```
2. **Alt-Host: teamwerk-Service wieder starten**
   ```bash
   ssh <alter-remote> "sudo systemctl enable teamwerk && sudo systemctl start teamwerk"
   ssh <alter-remote> "curl -s http://localhost:8080/api/healthz"
   ```
3. **DNS zurückstellen** (nur wenn schon umgehängt): A-Record im Provider-Panel
   auf alte IP. Warten auf Propagation.
4. **Better-Stack**: HTTP-Monitor-URL zurück auf alte Domain umhängen.
5. **Datenlage prüfen:** in der Zeit zwischen Cutover und Rollback ausgeführte
   Schreibaktionen auf der neuen Instanz sind auf dem Alt-Host nicht sichtbar.
   Ob rüber-migriert werden muss, hängt vom Zeitraum ab. Bei Minuten meistens
   verschmerzbar; bei Stunden manuell sichten und per SQL-Import zurücksichern.

---

## 7. Was nicht automatisiert ist

Bewusst außerhalb der Skript-Automatisierung, weil provider- oder
menschenspezifisch:

- **DNS-A-Record-Wechsel** — Provider-Panel.
- **Certbot-Erstlauf** auf Ziel — braucht bereits gesetzten DNS.
- **Better-Stack-Monitor-URLs umhängen** — externe API, Handarbeit im Panel.
- **User-Kommunikation** (Push-Broadcast, E-Mail, Vorstandsansage).
- **PWA-Neuinstallation** durch die Nutzer — Origin-gebunden im Browser, kein
  Server-Trick.
- **Cleanup des Alt-Hosts** (VPS abbauen beim Provider) — wenn und wann der
  Alt-Host ausgemustert wird, entscheidet der Betreuer.

---

## Anhang: konkrete Umzugs-Session (Vorlage)

Für den TeamStuttgart-Umzug (`intern.` → `teamwerk.`):

```bash
# 0. .env-Ergänzung
cat >> .env <<'EOF'
REMOTE_NEW=vServerNeu
REMOTE_NEW_DIR=/usr/local/bin
BASE_URL_NEW=https://teamwerk.team-stuttgart.org
EOF

# 1. Bootstrap
make server-bootstrap NEW_REMOTE=vServerNeu

# 2. Testphase — lokale /etc/hosts-Zeile
echo "31.70.110.19  teamwerk.team-stuttgart.org" | sudo tee -a /etc/hosts

# ... tagelanges Testen, bei Bedarf:
make server-sync-data NEW_REMOTE=vServerNeu

# 3. DNS im Provider-Panel setzen, warten, /etc/hosts-Zeile entfernen
ssh vServerNeu "certbot --nginx -d teamwerk.team-stuttgart.org --non-interactive --agree-tos -m vorstand@team-stuttgart.org"

# 4. Cutover
make server-cutover NEW_REMOTE=vServerNeu

# 5. Better-Stack-Monitor umhängen, User informieren.
```
