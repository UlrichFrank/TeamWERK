# Runbook — Cutover mit Dual-Serving (`internal.*` als Übergangs-Alias)

Anleitung für den finalen Umschalter auf die neue TeamWERK-Instanz
(`teamwerk.team-stuttgart.org` @ `31.70.110.19`), bei dem der bisherige
Hostname `internal.team-stuttgart.org` weiter erreichbar bleibt — aber **vom
neuen Host** ausgeliefert wird und **nicht** über einen dauerhaft laufenden
Alt-VPS mit 301.

Grundlage: die Änderungen aus dem OpenSpec-Change
`internal-hostname-transitional-alias` sind gemergt und im Binary/Bundle drin
(Config-Default, Scheduler-Fix, Banner, `deploy/nginx-teamwerk.conf`).

Dieses Runbook **ersetzt Kapitel 3 und 4** aus
[`server-migration-runbook.md`](server-migration-runbook.md). Kapitel 0–2
(Bootstrap + Testphase) und 5 (Nachpflege — Better-Stack, User-Kommunikation)
gelten weiter unverändert und werden hier nur referenziert.

---

## Voraussetzungen

- [ ] Bootstrap + Testphase gemäß Kap. 0–2 des `server-migration-runbook.md`
      abgeschlossen. Der neue VPS läuft, hat einen gültigen DB-/Storage-Snapshot,
      und `curl -k -H 'Host: teamwerk.team-stuttgart.org' https://localhost/api/healthz`
      auf dem Ziel-Host liefert `{"status":"ok",…}`.
- [ ] Change `internal-hostname-transitional-alias` implementiert und lokal
      grün (`make test lint`, `openspec validate`).
- [ ] DNS-TTL für `internal.team-stuttgart.org` im Mittwald-Panel **vorab**
      auf 300 s reduziert (mindestens 24 h vor dem Cutover, damit der Wechsel
      später zügig propagiert und ein Rollback ebenso zügig greift).
- [ ] Frisches Backup: `make backup && make backup-files` gegen die **alte**
      Produktion. Ist zusätzlich zu dem Snapshot aus `server-bootstrap`, weil
      seitdem Zeit vergangen ist.
- [ ] Ein Test-Nutzer-Account ohne kritische Berechtigungen, mit dem gleich
      der Cross-Origin-Login neu gemacht wird.

---

## Phase A — Code + neue Nginx-Config deployen (User-unsichtbar)

DNS für `internal.*` zeigt noch auf den Alt-VPS. Der Neu-Host bekommt nur
Traffic auf `teamwerk.*` (soweit der Betreuer testet) und ist damit
gefahrlos änderbar.

```bash
# 1. Binary + Bundle mit den neuen Code-Änderungen auf den Neu-Host
make deploy REMOTE=vServerNeu

# 2. Neue Nginx-Config auf den Neu-Host — server_name enthält vorerst
#    NUR teamwerk.*, weil DNS für internal.* noch nicht steht.
scp deploy/nginx-teamwerk.conf vServerNeu:/tmp/teamwerk.conf
ssh vServerNeu <<'REMOTE'
  # Übergangs-Zustand: server_name reduzieren auf teamwerk.* — internal.*
  # kommt in Phase C dazu, damit certbot --expand vorher nicht auf einen
  # 404 des Alt-Hosts läuft.
  sed -i 's/server_name teamwerk.team-stuttgart.org internal.team-stuttgart.org;/server_name teamwerk.team-stuttgart.org;/' /tmp/teamwerk.conf
  sudo mv /tmp/teamwerk.conf /etc/nginx/sites-available/teamwerk
  sudo nginx -t && sudo systemctl reload nginx
REMOTE

# 3. Env auf Neu-Host — BASE_URL zeigt auf Primärhost
ssh vServerNeu "sudo grep '^BASE_URL=' /etc/teamwerk/env"
# Sollte bereits durch server-bootstrap auf https://teamwerk.team-stuttgart.org
# stehen. Falls nicht, jetzt setzen und `sudo systemctl restart teamwerk`.

# 4. Smoke-Test Primärhost
curl -sSf https://teamwerk.team-stuttgart.org/api/healthz
curl -sSf https://teamwerk.team-stuttgart.org/                 # SPA-Shell
```

**Erwartung:** `teamwerk.*` läuft normal. Kein User bemerkt etwas
(Alt-Host serviert weiter `internal.*`).

---

## Phase B — DNS `internal.*` umziehen (User-sichtbar, aber unkritisch)

Ab jetzt zeigen sowohl `teamwerk.*` als auch `internal.*` auf die Neu-IP.
Für den kurzen Moment, in dem DNS schon steht, aber die Nginx-Config noch
kein `server_name internal.*;` hat, antwortet Nginx auf `internal.*` mit
dem Default-Server (`teamwerk`-Block) und liefert die App aus — nicht
korrekt gebrandet, aber funktional. Deshalb Phase C direkt hinterher.

```bash
# 1. Mittwald-Panel: A-Record internal.team-stuttgart.org → 31.70.110.19

# 2. Propagation abwarten (2× hintereinander gleiche Antwort reicht meist)
dig +short internal.team-stuttgart.org
# → 31.70.110.19

# Ggf. zweiter Resolver zur Sicherheit
dig +short internal.team-stuttgart.org @1.1.1.1
```

**Wenn dig noch die Alt-IP liefert:** einfach warten und Phase B nicht
verlassen — auf gar keinen Fall Phase C starten, sonst schlägt Certbot
fehl.

---

## Phase C — Zertifikat um `internal.*` erweitern

```bash
ssh vServerNeu 'certbot --nginx --expand \
  -d teamwerk.team-stuttgart.org \
  -d internal.team-stuttgart.org \
  --non-interactive --agree-tos -m vorstand@team-stuttgart.org'

# Verifikation: genau ein Zertifikat, beide SANs
ssh vServerNeu "sudo certbot certificates"
# Erwartet:
#   Certificate Name: teamwerk.team-stuttgart.org
#     Domains: teamwerk.team-stuttgart.org internal.team-stuttgart.org
#     Expiry Date: … (VALID)
```

`--expand` erweitert das bestehende `teamwerk.*`-Zertifikat um den zweiten
SAN, statt ein separates zu ordern — genau das Ziel (siehe design.md
„Entscheidung 1").

**Wenn Certbot mit `Detail: … 404` scheitert:** DNS ist noch nicht durch.
Zurück in Phase B, warten, wiederholen. Kein Grund für Panik — der
Primärhost bleibt in dieser Zeit unbetroffen erreichbar.

---

## Phase D — Nginx auf Dual-Serving flippen

```bash
# Neue Config mit beiden server_name-Einträgen deployen (die Original-Fassung,
# nicht die in Phase A reduzierte).
scp deploy/nginx-teamwerk.conf vServerNeu:/tmp/teamwerk.conf
ssh vServerNeu <<'REMOTE'
  sudo cp /etc/nginx/sites-available/teamwerk \
          /etc/nginx/sites-available/teamwerk.pre-alias.bak
  sudo mv /tmp/teamwerk.conf /etc/nginx/sites-available/teamwerk
  sudo nginx -t && sudo systemctl reload nginx
REMOTE

# Smoke-Test beide Hostnames — von außen, nicht vom Neu-Host aus
curl -sSf https://teamwerk.team-stuttgart.org/api/healthz
curl -sSf https://internal.team-stuttgart.org/api/healthz    # muss 200 sein, NICHT 301
curl -sSI https://internal.team-stuttgart.org/dashboard | head -1
# Erwartung: HTTP/2 200 (SPA-Shell) — kein 301, kein 502
```

Dann im Browser (nicht incognito, um Bestandsverhalten zu sehen):

- `https://teamwerk.team-stuttgart.org` → App lädt, **kein Banner sichtbar**.
- `https://internal.team-stuttgart.org` → App lädt, **Banner sichtbar** oben,
  CTA-Link zeigt auf `https://teamwerk.team-stuttgart.org/<derselbe Pfad>`.
- Als Test-User auf `internal.*` einloggen: klappt (Cookie wird auf
  internal.* gesetzt). Anschließend Banner-CTA klicken: auf teamwerk.* muss
  einmal neu eingeloggt werden (das ist der bewusste Trade-off, siehe
  design.md).

**Wenn der Banner auf `teamwerk.*` erscheint:** Regression in
`TransitionalHostnameBanner.tsx` — der Host-Check ist verhauen. Nginx
zurückrollen (`/etc/nginx/sites-available/teamwerk.pre-alias.bak`), Change
lokal debuggen, kein Rollback der DNS-Umstellung nötig.

---

## Phase E — Alt-VPS abschalten

Der Alt-VPS bekommt jetzt keinen legitimen Traffic mehr. Vor dem endgültigen
Ausschalten die Zugriffslogs kurz prüfen — falls doch irgendein Kanal
(externer Service, ical-Feed-Konsument, alter Push-Endpoint) noch auf die
Alt-IP zugreift.

```bash
# 1. Nachlaufender Traffic auf Alt-Host (letzte 30 min)
ssh vServer 'sudo tail -n 5000 /var/log/nginx/access.log | \
             awk "{print \$1, \$7}" | sort -u | head -50'

# 2. teamwerk-Service auf Alt-Host stoppen und disablen
ssh vServer 'sudo systemctl stop teamwerk && sudo systemctl disable teamwerk'
ssh vServer 'sudo systemctl status teamwerk --no-pager | head -5'
# Erwartet: inactive (dead), disabled

# 3. Cron-Wrapper deaktivieren, damit Scheduler nicht mehr auf toter DB tickt
ssh vServer 'sudo crontab -l | grep -v teamwerk-scheduler | sudo crontab -'
ssh vServer 'sudo crontab -l'

# 4. Nginx auf Alt-Host stoppen (Zertifikate lassen wir stehen für Rollback)
ssh vServer 'sudo systemctl stop nginx && sudo systemctl disable nginx'

# 5. Sanity: Alt-Host antwortet nicht mehr auf HTTP/HTTPS
curl -sS -o /dev/null -w '%{http_code}\n' --resolve \
  internal.team-stuttgart.org:443:217.160.118.39 \
  https://internal.team-stuttgart.org/api/healthz
# Erwartet: 000 (connection refused) oder Timeout
```

**Der VPS wird noch nicht beim Provider gekündigt** — er bleibt als
Rollback-Reserve stehen, bis der neue Host über einen Übungszeitraum
(Größenordnung 1–2 Wochen) stabil läuft. Danach IONOS-Kündigung als
separater Ops-Schritt außerhalb dieses Runbooks.

---

## Was danach kommt (nicht Teil dieses Cutovers)

- **Better-Stack-Monitor umhängen:** identisch zu Kap. 5 des großen
  Runbooks. Monitor-URL auf `https://teamwerk.team-stuttgart.org/api/healthz`.
  Heartbeat + Logs bleiben unverändert.
- **User-Kommunikation:** *entfällt weitgehend*, weil der Banner sie
  automatisiert. Für Push-Nutzer trotzdem einmalig via Broadcast (aus der
  neuen Instanz gesendet, damit die noch-registrierten Alt-Origin-Subs die
  Nachricht bekommen): „Bitte PWA auf `teamwerk.team-stuttgart.org` neu
  installieren und Push dort neu erlauben."
- **301-Flip:** irgendwann als eigener Change (`internal-hostname-hard-redirect`),
  wenn der Betreuer entscheidet — kein Datum, siehe Change-Design
  „Entscheidung 5". Bis dahin: Dual-Serving läuft weiter.

---

## Rollback

Wenn während Phase B–D irgendetwas klemmt und schnell zurückgerollt werden
muss:

### Rollback A: nur Nginx-Config falsch (Banner, Config-Fehler)

```bash
ssh vServerNeu 'sudo cp /etc/nginx/sites-available/teamwerk.pre-alias.bak \
                        /etc/nginx/sites-available/teamwerk && \
                 sudo nginx -t && sudo systemctl reload nginx'
```

DNS bleibt bestehen, Bestandsnutzer sehen kurz ein Zwischen-Rendering.
Debuggen, neu deployen, zurück in Phase D.

### Rollback B: alles zurück, Alt-Host wieder aktiv

Wenn der neue Host als Ganzes zickt und ein Fix > 30 min dauern würde:

```bash
# 1. DNS internal.* zurück auf Alt-IP (Mittwald-Panel)
#    internal.team-stuttgart.org  A → 217.160.118.39

# 2. Alt-Host wieder hochfahren
ssh vServer 'sudo systemctl enable teamwerk && sudo systemctl start teamwerk'
ssh vServer 'sudo systemctl enable nginx && sudo systemctl start nginx'
ssh vServer 'curl -s http://localhost:8080/api/healthz'   # → status:ok
ssh vServer 'sudo crontab -e'   # Scheduler-Cron wieder einhängen — siehe Backup

# 3. DNS-Propagation abwarten
until [ "$(dig +short internal.team-stuttgart.org)" = "217.160.118.39" ]; do
  sleep 30; done
```

**Datenlage-Warnung:** Schreibaktionen, die zwischen Cutover und Rollback
über `teamwerk.*` (nicht `internal.*`) auf dem Neu-Host passiert sind, sind
danach auf dem Alt-Host **nicht** sichtbar. Wenn das relevant ist, DB-Diff
manuell abgleichen bevor die alte DB weiterschreiben darf — bei kurzem
Cutover-Fenster (Minuten) meist verschmerzbar.

### Rollback C: 301 statt Dual-Serving, doch kein Alt-Host mehr

Alternative, falls Phase D scheitert und ein sofortiges 301 auf teamwerk.*
akzeptabel ist (Bulk-Logout in Kauf nehmen): auf dem **Neu-Host** einen
zweiten `server`-Block anlegen mit `server_name internal.team-stuttgart.org;`
und `return 301 https://teamwerk.team-stuttgart.org$request_uri;`. Das ist
faktisch der Zielzustand des späteren Follow-up-Changes, nur früher
gezogen.

---

## Kommando-Zusammenfassung (Copy-Paste-Vorlage)

```bash
# Phase A
make deploy REMOTE=vServerNeu
scp deploy/nginx-teamwerk.conf vServerNeu:/tmp/teamwerk.conf
ssh vServerNeu 'sed -i "s/server_name teamwerk.team-stuttgart.org internal.team-stuttgart.org;/server_name teamwerk.team-stuttgart.org;/" /tmp/teamwerk.conf && \
                sudo mv /tmp/teamwerk.conf /etc/nginx/sites-available/teamwerk && \
                sudo nginx -t && sudo systemctl reload nginx'
curl -sSf https://teamwerk.team-stuttgart.org/api/healthz

# Phase B — DNS im Mittwald-Panel setzen, dann warten:
until [ "$(dig +short internal.team-stuttgart.org)" = "31.70.110.19" ]; do sleep 30; done

# Phase C
ssh vServerNeu 'certbot --nginx --expand \
  -d teamwerk.team-stuttgart.org -d internal.team-stuttgart.org \
  --non-interactive --agree-tos -m vorstand@team-stuttgart.org'

# Phase D
scp deploy/nginx-teamwerk.conf vServerNeu:/tmp/teamwerk.conf
ssh vServerNeu 'sudo cp /etc/nginx/sites-available/teamwerk /etc/nginx/sites-available/teamwerk.pre-alias.bak && \
                sudo mv /tmp/teamwerk.conf /etc/nginx/sites-available/teamwerk && \
                sudo nginx -t && sudo systemctl reload nginx'
curl -sSf https://teamwerk.team-stuttgart.org/api/healthz
curl -sSf https://internal.team-stuttgart.org/api/healthz

# Phase E
ssh vServer 'sudo systemctl stop teamwerk && sudo systemctl disable teamwerk && \
             sudo systemctl stop nginx && sudo systemctl disable nginx'
```
