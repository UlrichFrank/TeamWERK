# Deployment & VPS

IONOS VPS Linux XS · Binary `/usr/local/bin/teamwerk` · systemd-Service `teamwerk` · Nginx Reverse Proxy 443→8080 (Certbot). Config `/etc/teamwerk/env` (PORT, DB_PATH, JWT_SECRET, FIELD_ENCRYPTION_KEY, SMTP_*, VAPID_*, LOG_FORMAT, METRICS_TOKEN). DB `/var/lib/teamwerk/teamwerk.db`. Scheduler-Cronjob `* * * * * /usr/local/bin/teamwerk-scheduler.sh` (Wrapper lädt Env, sendet Better-Stack-Heartbeat bei Erfolg). Erstaufbau: `deploy/vps-setup-runbook.md` (Schritte) + `deploy/setup-vps.sh` (idempotentes Script).

SSH-Alias `vServer` (in `.env`), direkt `https://217.160.118.39`. Domain + Certbot-Zertifikat noch ausstehend.

```bash
make migrate-remote-up                               # Migrationen auf VPS
make create-admin-remote EMAIL=… PASSWORD=… NAME=…   # Admin anlegen
```

## Zero-Knowledge-Verschlüsselung der Bank-/SEPA-PII (Modell B)

> **Status:** Das Zero-Knowledge-Modell ist auf `feat/zero-knowledge-bank-vault`
> implementiert, aber **noch nicht in Produktion migriert**. Produktion läuft bis zur
> Migration (siehe unten) auf dem alten serverseitigen `FIELD_ENCRYPTION_KEY`-Modell.

Bank-/SEPA-Felder werden **clientseitig** verschlüsselt (`internal/crypto` ist serverseitig
nicht mehr am Lese-/Schreibpfad beteiligt). Der Server speichert nur Ciphertext + gewrappte
Schlüssel und **besitzt keinen Entschlüsselungsschlüssel**:
- Vereins-**Keypair** (RSA-OAEP): öffentlicher Schlüssel `clubs.group_public_key` (nicht
  geheim) zum Schreiben; privater Schlüssel `clubs.group_private_key_enc` =
  `AES-GCM(PKCS8, PBKDF2(passphrase))`, entschlüsselbar nur mit der geteilten
  Tresor-Passphrase (Vorstand/Kassierer).
- **Einrichtung** (einmalig, über die UI „Tresor"): Passphrase setzen → Keypair erzeugen →
  `group_public_key` + `group_private_key_enc` + Salt + Key-Check werden gespeichert. Die
  Passphrase verlässt den Browser nie.

**Rollout-Sequenz (zwei entkoppelte Deploys — minimales Brücken-Fenster):**

Das kritische, irreversible Fenster (Server hält gleichzeitig Brücken-Schlüssel **und**
liefert v1-Klartext aus) wird minimiert, indem die Startup-Toleranz (Server startet auch ohne
Schlüssel) bereits in Branch A enthalten ist. Der irreversible Moment ist dann nur noch eine
skriptbare Ops-Aktion, kein Code-Deploy.

```bash
# Branch A (feat/zk-migrate-bestand): Migrations-Endpoint + UI + tolerant startup.
#   FIELD_ENCRYPTION_KEY bleibt gesetzt (Brücke). Voll reversibel.
make deploy                          # Branch A ausrollen
# 1. Vorstand/Kassierer richtet den Tresor ein (UI „Tresor") und prüft alle Bank-Flows
#    im Browser (Member-Bank, Vereins-SEPA, Mandat-PDF, Fee-Run).
make backup                          # DB-Backup (kritisch, irreversibel ab Schritt 3)
# 2. Migrationslauf im Browser: UI „Datenmigration" (/migration) — lädt v1-Altbestand über
#    die Brücke, re-verschlüsselt clientseitig an den Gruppen-Public-Key, lädt hoch.
#    Idempotent; offene bankdaten-Anträge (Drafts) vorher annehmen/ablehnen.
# 3. Brücken-Schlüssel entfernen (sekundenschnell, kein Deploy):
make zk-finalize-remote              # prüft complete; entfernt FIELD_ENCRYPTION_KEY + Restart
# Branch B (feat/zk-remove-bridge): Migrations-Endpoint + Brücken-Code + Legacy-Spalten weg.
make deploy                          # Branch B als Hygiene, jederzeit später
```

**Modellgrenze (ehrlich):** schützt gegen passive Kompromittierung (geleaktes DB-Backup,
Disk-Snapshot, neugieriger Hoster, lesender App-Admin), **nicht** gegen einen aktiv
bösartigen Server (liefert das JS aus) oder ein kompromittiertes Kassierer-Endgerät.

**Backup-/Recovery-Regel (kritisch):** Es gibt **kein serverseitiges Recovery**. Geht die
Tresor-Passphrase verloren (alle Inhaber), sind **alle Bank-/SEPA-Daten unwiederbringlich
verloren** (`group_private_key_enc` ist dann nicht mehr entschlüsselbar). Passphrase
mindestens zwei verantwortlichen Personen bekannt machen + sicher hinterlegen
(Passwort-Manager des Vorstands). **DB-Backup vor dem Migrationslauf** ziehen.
