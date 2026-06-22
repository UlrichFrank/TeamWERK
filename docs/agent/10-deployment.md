# Deployment & VPS

IONOS VPS Linux XS · Binary `/usr/local/bin/teamwerk` · systemd-Service `teamwerk` · Nginx Reverse Proxy 443→8080 (Certbot). Config `/etc/teamwerk/env` (PORT, DB_PATH, JWT_SECRET, SMTP_*, VAPID_*, LOG_FORMAT, METRICS_TOKEN). DB `/var/lib/teamwerk/teamwerk.db`. Scheduler-Cronjob `* * * * * /usr/local/bin/teamwerk-scheduler.sh` (Wrapper lädt Env, sendet Better-Stack-Heartbeat bei Erfolg). Erstaufbau: `deploy/vps-setup-runbook.md` (Schritte) + `deploy/setup-vps.sh` (idempotentes Script).

SSH-Alias `vServer` (in `.env`), direkt `https://217.160.118.39`. Domain + Certbot-Zertifikat noch ausstehend.

```bash
make migrate-remote-up                               # Migrationen auf VPS
make create-admin-remote EMAIL=… PASSWORD=… NAME=…   # Admin anlegen
```
