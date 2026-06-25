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

**Rollout-Sequenz (geplant, Sektion 6 — noch offen):**

```bash
# Vorbedingung: feat/zero-knowledge-bank-vault deployt; FIELD_ENCRYPTION_KEY noch gesetzt
#   (Migrations-Brücke). DB-Backup ziehen (kritisch, irreversibel ab Schritt 3).
# 1. Vorstand/Kassierer richtet den Tresor ein (UI „Tresor") und prüft alle Bank-Flows
#    im Browser (Member-Bank, Vereins-SEPA, Mandat-PDF, Fee-Run).
# 2. Einmaliger Migrationslauf: der Browser eines Tresor-Inhabers entschlüsselt den
#    v1-Altbestand über die Server-Brücke, re-verschlüsselt clientseitig an den
#    Gruppen-Public-Key und lädt die Envelopes hoch.
# 3. FIELD_ENCRYPTION_KEY aus /etc/teamwerk/env entfernen + Server-Neustart — ab jetzt
#    kann der Server Bank-/SEPA-PII prinzipiell nicht mehr entschlüsseln.
```

**Modellgrenze (ehrlich):** schützt gegen passive Kompromittierung (geleaktes DB-Backup,
Disk-Snapshot, neugieriger Hoster, lesender App-Admin), **nicht** gegen einen aktiv
bösartigen Server (liefert das JS aus) oder ein kompromittiertes Kassierer-Endgerät.

**Backup-/Recovery-Regel (kritisch):** Es gibt **kein serverseitiges Recovery**. Geht die
Tresor-Passphrase verloren (alle Inhaber), sind **alle Bank-/SEPA-Daten unwiederbringlich
verloren** (`group_private_key_enc` ist dann nicht mehr entschlüsselbar). Passphrase
mindestens zwei verantwortlichen Personen bekannt machen + sicher hinterlegen
(Passwort-Manager des Vorstands). **DB-Backup vor dem Migrationslauf** ziehen.
