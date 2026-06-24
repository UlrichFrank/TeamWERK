# Deployment & VPS

IONOS VPS Linux XS · Binary `/usr/local/bin/teamwerk` · systemd-Service `teamwerk` · Nginx Reverse Proxy 443→8080 (Certbot). Config `/etc/teamwerk/env` (PORT, DB_PATH, JWT_SECRET, FIELD_ENCRYPTION_KEY, SMTP_*, VAPID_*, LOG_FORMAT, METRICS_TOKEN). DB `/var/lib/teamwerk/teamwerk.db`. Scheduler-Cronjob `* * * * * /usr/local/bin/teamwerk-scheduler.sh` (Wrapper lädt Env, sendet Better-Stack-Heartbeat bei Erfolg). Erstaufbau: `deploy/vps-setup-runbook.md` (Schritte) + `deploy/setup-vps.sh` (idempotentes Script).

SSH-Alias `vServer` (in `.env`), direkt `https://217.160.118.39`. Domain + Certbot-Zertifikat noch ausstehend.

```bash
make migrate-remote-up                               # Migrationen auf VPS
make create-admin-remote EMAIL=… PASSWORD=… NAME=…   # Admin anlegen
```

## At-Rest-Verschlüsselung der Bank-/SEPA-PII (`FIELD_ENCRYPTION_KEY`)

Bank-/SEPA-Felder (Mitglieds-IBAN/Kontoinhaber, `member_change_drafts`-Bankdaten,
Vereins-SEPA-Stammdaten, SEPA-Mandat-PDFs) werden serverseitig AES-256-GCM
verschlüsselt gespeichert (`internal/crypto`). Der Schlüssel liegt in
`FIELD_ENCRYPTION_KEY` (32 Byte, base64). **Ohne gültigen Schlüssel startet der
Server nicht** (Startup-Check in `serve()`).

**Rollout-Sequenz (Zero-Downtime):**

```bash
go run ./cmd/teamwerk gen-encryption-key   # FIELD_ENCRYPTION_KEY=… → in /etc/teamwerk/env (chmod 600)
make deploy                                # Binary + restart; ab jetzt wird jeder Schreibvorgang verschlüsselt,
                                           #   Lesen versteht Klartext + Ciphertext (toleranter Decrypt)
ssh vServer /usr/local/bin/teamwerk encrypt-pii   # einmalige Erstverschlüsselung des Bestands (idempotent)
```

`encrypt-pii` ist idempotent (bereits verschlüsselte Werte werden übersprungen)
und bei Abbruch wiederholbar. Spiegelbild für Rollback/Schlüsselrotation:
`teamwerk decrypt-pii` (vor einem Code-Downgrade nach erfolgtem `encrypt-pii`).

**Backup-Regel (kritisch):** Den Schlüssel **niemals** im selben Backup wie die
DB ablegen — sonst ist die Verschlüsselung gegen ein geleaktes Backup wirkungslos.
Schlüssel separat sichern (Passwort-Manager). **Schlüsselverlust = Datenverlust**
der Bank-/SEPA-Felder (kein Recovery möglich). DB-Backup **vor** dem ersten
`encrypt-pii`-Lauf ziehen.
