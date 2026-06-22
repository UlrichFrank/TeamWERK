# VPS-Setup-Runbook

Schritt-für-Schritt-Anleitung für einen frischen TeamWERK-VPS (z. B. nach
Provider-Wechsel oder Server-Verlust). Komplementiert `deploy/setup-vps.sh`,
das den mechanischen Teil übernimmt.

Zielsystem: Ubuntu 24.04 LTS, mindestens 1 GB RAM, 10 GB Disk, root-SSH.

---

## 0. Vorab — externe Konten

Diese Dinge müssen **vor** dem Setup existieren, sonst hängt das Script.

| Was | Wo | Wofür |
|---|---|---|
| SSH-Zugang als root | Provider-Panel | gesamtes Setup |
| Domain + DNS-A-Record | Provider-DNS | Nginx + Certbot |
| SMTP-Zugangsdaten | mailagenturserver oder Ersatz | `SMTP_*` in Env |
| Better Stack Heartbeat-Monitor | uptime.betterstack.com | Scheduler-Totmann |
| Better Stack Logs-Source | logs.betterstack.com | Vector → Log-Stream |
| Better Stack HTTP-Monitor | uptime.betterstack.com | `/api/healthz` |

**Better-Stack-Vorbereitung:**
1. Monitors → New → **HTTP** → `https://<DOMAIN>/api/healthz`, Keyword `"db":"ok"`, Intervall 1 min
2. Monitors → New → **Heartbeat** → Period `2 min`, Grace `1 min` → URL kopieren
3. Logs → Sources → New → Ubuntu → Source-Token kopieren

---

## 1. SSH-Alias setzen (lokal)

```bash
# ~/.ssh/config
Host vServer
  HostName <VPS-IP>
  User root
  IdentityFile ~/.ssh/id_ed25519
```

Test: `ssh vServer "uname -a"`.

---

## 2. Setup-Script auf VPS ausführen

```bash
# Vom lokalen Repo aus
rsync -az deploy/ vServer:/tmp/deploy/
ssh vServer "cd /tmp/deploy && bash setup-vps.sh"
```

Das Script ist idempotent (kann mehrfach laufen). Es legt an:

- `/etc/teamwerk/env` mit Platzhaltern (`REPLACE_*`)
- `/etc/teamwerk/heartbeat-url`, `/etc/teamwerk/betterstack-logs-token`
- `/usr/local/bin/teamwerk-scheduler.sh` (Cron-Wrapper)
- systemd-Service, Nginx-vhost, self-signed Cert
- Vector (installiert, aber nicht gestartet bis Token gesetzt)
- Logrotate-Config
- Cronjob (`* * * * *`)

---

## 3. Geheimnisse einsetzen

Auf dem VPS:

```bash
# 3a. Env-Datei
vim /etc/teamwerk/env
# Setzen:
#   BASE_URL=https://<DOMAIN>
#   SMTP_USER, SMTP_PASS
#   VAPID_* — bleibt erstmal Platzhalter (siehe Schritt 5)

# 3b. Better-Stack-Heartbeat
echo "https://uptime.betterstack.com/api/v1/heartbeat/XXXX" > /etc/teamwerk/heartbeat-url
chmod 600 /etc/teamwerk/heartbeat-url

# 3c. Better-Stack-Logs-Token
echo "XXXXXXXXXXXXX" > /etc/teamwerk/betterstack-logs-token
chmod 600 /etc/teamwerk/betterstack-logs-token
```

---

## 4. Binary deployen

Lokal:

```bash
make deploy
```

Dies baut den Linux-Binary, kopiert nach `/usr/local/bin/teamwerk`, führt
`migrate up` aus und startet den Service.

---

## 5. VAPID-Keys erzeugen (Push-Notifications)

```bash
ssh vServer "/usr/local/bin/teamwerk gen-vapid"
# Output:
#   VAPID_PUBLIC_KEY=…
#   VAPID_PRIVATE_KEY=…
ssh vServer "vim /etc/teamwerk/env"   # Werte eintragen
ssh vServer "systemctl restart teamwerk"
```

---

## 6. Admin anlegen + Daten zurückspielen

Frische Box, ohne Backup:

```bash
make create-admin-remote EMAIL=admin@example.org PASSWORD='…' NAME='Admin'
```

Mit Backup einer alten DB:

```bash
ssh vServer "systemctl stop teamwerk"
scp teamwerk.db.backup vServer:/var/lib/teamwerk/teamwerk.db
ssh vServer "chown www-data:www-data /var/lib/teamwerk/teamwerk.db && \
             /usr/local/bin/teamwerk migrate up --db /var/lib/teamwerk/teamwerk.db && \
             systemctl start teamwerk"
```

Beitragslauf-Protokolle: `/var/lib/teamwerk/beitragslauf-protokolle/` mit dem
Backup nachziehen (Append-only-Textdateien — siehe `CLAUDE.md`).

---

## 7. Vector starten (Log-Stream)

Nach Schritt 3c:

```bash
ssh vServer "systemctl restart vector && systemctl is-active vector"
```

Verifizieren: Logs erscheinen innerhalb von ~1 min in Better Stack → Logs.

---

## 8. TLS (Certbot)

Sobald Domain auf VPS-IP zeigt:

```bash
ssh vServer "apt-get install -y certbot python3-certbot-nginx && \
             certbot --nginx -d <DOMAIN> --non-interactive --agree-tos -m <EMAIL>"
```

Certbot ersetzt den self-signed Cert in der Nginx-Config automatisch.
Auto-Renewal läuft via systemd-Timer (`systemctl list-timers | grep certbot`).

---

## 9. Smoke-Test

```bash
# Health (public)
curl -s https://<DOMAIN>/api/healthz
# → {"status":"ok","db":"ok","disk_free_pct":NN,"scheduler_age_sec":NN}

# Metrics (Token aus /etc/teamwerk/env)
curl -s -H "Authorization: Bearer $METRICS_TOKEN" https://<DOMAIN>/api/metrics
# → teamwerk_up 1, teamwerk_db_up 1, …

# Scheduler-Heartbeat (innerhalb 2 min ankommen)
# → Better Stack Heartbeat-Monitor zeigt "Up"

# Logs in Better Stack → Logs erscheinen (JSON, Feld msg, level, time)
```

---

## 10. Nachpflege

| Datei | Was tun |
|---|---|
| `~/.ssh/config` lokal | Alias `vServer` auf neue IP umstellen |
| `Makefile` | Falls SSH-Host oder Pfade abweichen, Targets prüfen |
| `CLAUDE.md` | IP-Adresse aktualisieren |

---

## Was das Script **nicht** macht

- DNS einrichten (Provider-spezifisch)
- TLS-Cert (Schritt 8 manuell, weil Domain-DNS Voraussetzung)
- Backup-Wiederherstellung (Schritt 6 manuell, weil Quelle variiert)
- Better-Stack-Accounts/Monitore (Schritt 0 manuell — externe API)

---

## Bei Verlust dieses Runbooks

Quelle der Wahrheit für das Setup ist `deploy/setup-vps.sh`. Es enthält alle
Pfade, Permissions und Konfig-Inhalte; das Runbook fasst nur die Reihenfolge
und manuellen Schritte zusammen.
