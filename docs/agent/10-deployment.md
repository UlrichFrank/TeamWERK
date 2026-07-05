# Deployment & VPS

IONOS VPS Linux XS · Binary `/usr/local/bin/teamwerk` · systemd-Service `teamwerk` · Nginx Reverse Proxy 443→8080 (Certbot). Config `/etc/teamwerk/env` (PORT, DB_PATH, JWT_SECRET, SMTP_*, VAPID_*, LOG_FORMAT, METRICS_TOKEN — **kein** `FIELD_ENCRYPTION_KEY` mehr, Zero-Knowledge). DB `/var/lib/teamwerk/teamwerk.db`. Scheduler-Cronjob `* * * * * /usr/local/bin/teamwerk-scheduler.sh` (Wrapper lädt Env, sendet Better-Stack-Heartbeat bei Erfolg). Erstaufbau: `deploy/vps-setup-runbook.md` (Schritte) + `deploy/setup-vps.sh` (idempotentes Script).

SSH-Alias `vServer` (in `.env`), direkt `https://217.160.118.39`. Domain + Certbot-Zertifikat noch ausstehend.

**Dual-Serving-Übergang:** Der Primärhost ist `teamwerk.team-stuttgart.org` (VPS `31.70.110.19`). Der Alt-Hostname `internal.team-stuttgart.org` wird als Übergangs-Alias aus **demselben** Nginx-`server`-Block bedient (ein Let's-Encrypt-Zertifikat mit beiden SANs, `deploy/nginx-teamwerk.conf`); ein UI-Banner (`TransitionalHostnameBanner`) weist Nutzer auf `internal.*` auf den Umzug hin. Ablauf + Rollback: `deploy/internal-alias-cutover-runbook.md`. Der spätere Flip von `internal.*` auf ein 301 ist ein eigener Follow-up-Change (`internal-hostname-hard-redirect`), Zeitpunkt bewusst offen.

```bash
make migrate-remote-up                               # Migrationen auf VPS
make create-admin-remote EMAIL=… PASSWORD=… NAME=…   # Admin anlegen
```

## Server-Umzug (VPS-Wechsel)

Wiederkehrender Ablauf zum Umzug einer TeamWERK-Instanz auf einen anderen VPS steckt in drei Makefile-Targets: `make server-bootstrap NEW_REMOTE=<alias>` (initialer Aufbau + Daten-Klon), `make server-sync-data NEW_REMOTE=<alias>` (beliebig oft wiederholbarer DB-/Storage-Sync während der Testphase), `make server-cutover NEW_REMOTE=<alias>` (Alt-Host auf 301-Redirect umschalten). Voraussetzungen (`REMOTE_NEW`, `REMOTE_NEW_DIR`, `BASE_URL_NEW` in `.env`) und alle manuellen Schritte (DNS, Certbot, Better-Stack-Umhängen, User-Kommunikation, PWA-Neuinstallation, Rollback) stehen in `deploy/server-migration-runbook.md`.

## Zero-Knowledge-Verschlüsselung der Bank-/SEPA-PII (Modell B)

Bank-/SEPA-Felder werden **clientseitig** verschlüsselt (`internal/crypto` ist serverseitig
nicht mehr am Lese-/Schreibpfad beteiligt; nur noch Client-Magic-Erkennung beim Upload). Der
Server speichert nur Ciphertext + gewrappte Schlüssel und **besitzt keinen
Entschlüsselungsschlüssel**:
- Vereins-**Keypair** (RSA-OAEP): öffentlicher Schlüssel `clubs.group_public_key` (nicht
  geheim) zum Schreiben; privater Schlüssel `clubs.group_private_key_enc` =
  `AES-GCM(PKCS8, PBKDF2(passphrase))`, entschlüsselbar nur mit der geteilten
  Tresor-Passphrase (Vorstand/Kassierer).
- **Einrichtung** (einmalig, über die UI „Tresor"): Passphrase setzen → Keypair erzeugen →
  `group_public_key` + `group_private_key_enc` + Salt + Key-Check werden gespeichert. Die
  Passphrase verlässt den Browser nie.

**Bestandsmigration: abgeschlossen.** Der einmalige `v1:`-Altbestand wurde über eine
temporäre Server-Brücke (`FIELD_ENCRYPTION_KEY`) im Browser eines Tresor-Inhabers nach
Modell B re-verschlüsselt; danach wurden der Schlüssel aus der Umgebung entfernt und die
Brücke (`internal/crypto`-Decrypt, Migrations-Endpoint, Legacy-Spalten `members.iban/
account_holder`, `clubs.glaeubiger_id/iban/bic/kontoinhaber`) per Migration `009` entfernt.
Der Server startet und läuft seitdem **ohne** Entschlüsselungsschlüssel (envelope-only). Das
Migrations-Werkzeug (Endpoint + UI + `make zk-finalize-remote`) existiert nicht mehr.

**Modellgrenze (ehrlich):** schützt gegen passive Kompromittierung (geleaktes DB-Backup,
Disk-Snapshot, neugieriger Hoster, lesender App-Admin), **nicht** gegen einen aktiv
bösartigen Server (liefert das JS aus) oder ein kompromittiertes Kassierer-Endgerät.

**Backup-/Recovery-Regel (kritisch):** Es gibt **kein serverseitiges Recovery**. Geht die
Tresor-Passphrase verloren (alle Inhaber), sind **alle Bank-/SEPA-Daten unwiederbringlich
verloren** (`group_private_key_enc` ist dann nicht mehr entschlüsselbar). Passphrase
mindestens zwei verantwortlichen Personen bekannt machen + sicher hinterlegen
(Passwort-Manager des Vorstands). **DB-Backup vor dem Migrationslauf** ziehen.
