## Why

Der `FIELD_ENCRYPTION_KEY` (AES-256-GCM, At-Rest-Verschlüsselung der Bank-/SEPA-PII aus `encrypt-bank-sepa-at-rest`) liegt heute als **Klartext-Zeile** in `/etc/teamwerk/env` und wird via systemd `EnvironmentFile=` in die **Prozess-Umgebung** geladen. Dadurch steht er in `/proc/<pid>/environ` — eine App-RCE als `www-data` liest ihn direkt aus, und ein kopierter `env`-Datei-Inhalt ist auf jedem anderen Host sofort brauchbar. Außerdem gibt es heute keinen Weg, einen kompromittierten Schlüssel auszutauschen, ohne den Bestand manuell zu de-/re-verschlüsseln.

Ziel ist eine **verhältnismäßige** Härtung für einen einzelnen 1-GB-VPS: den Schlüssel verschlüsselt + host-gebunden ablegen und aus der Prozess-Umgebung entfernen, sowie Schlüsselrotation als First-Class-Operation. Bewusst **kein** Vault/KMS auf derselben Maschine (verschiebt nur das Bootstrap-Problem, hoher Betriebsaufwand/RAM-Druck) — das wird mit Begründung als verworfene Alternative dokumentiert.

## What Changes

- **systemd-Credentials statt Env (primär):** `teamwerk.service` liefert den Schlüssel über `LoadCredentialEncrypted=`/`SetCredentialEncrypted=` (erzeugt mit `systemd-creds encrypt`, host- bzw. `host+tpm2`-gebunden). Der Schlüssel liegt damit **verschlüsselt** auf der Platte (auf einem anderen Host wertlos) und der Klartext erscheint nur im per-Service-tmpfs unter `$CREDENTIALS_DIRECTORY` — **nicht mehr** in `/proc/environ`.
- **Schlüsselquelle mit Fallback:** `internal/crypto` liest den Schlüssel bevorzugt aus `$CREDENTIALS_DIRECTORY/field_key` (Datei), mit Fallback auf die Umgebungsvariable `FIELD_ENCRYPTION_KEY`. Das erlaubt Zero-Downtime-Migration (erst Code, dann Credential umstellen) und lässt lokale Entwicklung weiter per `.env` laufen.
- **Schlüsselrotation:** Neues Subcommand `rotate-key` (alt entschlüsseln → mit neuem Schlüssel als versioniertes Format `"v2:"` neu verschlüsseln), idempotent, auf Basis des vorhandenen `encrypt-pii`/`decrypt-pii`-Mechanismus.
- **Deployment:** `deploy/teamwerk.service`, `deploy/setup-vps.sh` und `deploy/deploy-encryption.sh` provisionieren die systemd-Credential (mit TPM2-Verfügbarkeitsprüfung; Fallback host-key-Binding über `/var/lib/systemd/credential.secret`).
- **Doku:** Rollout-, Rotations- und Backup-Hinweise in `10-deployment.md`; Schlüsselquellen-Reihenfolge in `03-go.md`.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `bank-data-at-rest-encryption`: Die Anforderung „App-gehaltener Schlüssel" wird verschärft — Schlüsselquelle ist primär eine **systemd-Credential** (verschlüsselt, host-/TPM-gebunden, nicht in der Prozess-Umgebung), mit Env-Fallback; zusätzlich wird **Schlüsselrotation** (`rotate-key`, versioniertes `"v2:"`-Format) als Anforderung ergänzt. Ver-/Entschlüsselungs-Semantik und Berechtigungsregel (`policy.CanDecryptBankData`) bleiben unverändert.

## Impact

- **Code:** `internal/crypto` (Schlüsselquelle Credential-Datei + Env-Fallback; `"v2:"`-Format/Rotation), `cmd/teamwerk` (`rotate-key`-Subcommand; Startup-Check liest neue Quelle).
- **Deployment/Betrieb:** `deploy/teamwerk.service` (`LoadCredentialEncrypted=`), `deploy/setup-vps.sh` + `deploy/deploy-encryption.sh` (Credential erzeugen/provisionieren statt/zusätzlich zur `env`-Zeile), TPM2-Verfügbarkeitsprüfung.
- **Doku:** `docs/agent/10-deployment.md`, `docs/agent/03-go.md`. `.env.example` bleibt für lokale Entwicklung (Env-Fallback).
- **Daten:** keine Schemaänderung. Eine Rotation überführt vorhandene `"v1:"`-Werte nach `"v2:"`; `Decrypt` bleibt tolerant (versioniertes Präfix + Klartext-Passthrough).
- **Bedrohungsmodell:** schließt die `/proc/environ`-Exposition und „env-Datei-portierbar"-Lücke; schützt **weiterhin nicht** gegen Live-Root/RCE auf dem laufenden Host (serverseitige Entschlüsselung bleibt nötig) — bewusst akzeptierte Grenze.
- **Kompatibilität:** Env-Fallback erhält lokale Dev + ermöglicht stufenweisen Rollout; kein Wartungsfenster.
