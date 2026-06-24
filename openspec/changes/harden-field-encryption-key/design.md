## Context

`encrypt-bank-sepa-at-rest` führte serverseitige AES-256-GCM-Verschlüsselung der Bank-/SEPA-PII ein. Der Schlüssel (`FIELD_ENCRYPTION_KEY`, 32 Byte base64) liegt als Klartext-Zeile in `/etc/teamwerk/env` und wird via systemd `EnvironmentFile=/etc/teamwerk/env` in die Umgebung des `teamwerk`-Service (User `www-data`) geladen. `internal/crypto.InitFromEnv()` liest ihn mit `os.Getenv`. Der Startup-Check verweigert den Boot ohne gültigen Schlüssel.

Zwei konkrete Schwächen des Ist-Zustands:
1. **Prozess-Umgebung:** Über `EnvironmentFile` steht der Klartext-Schlüssel in `/proc/<pid>/environ` — eine Codeausführung als `www-data` (App-RCE) liest ihn direkt.
2. **Portierbarer Klartext + keine Rotation:** Die `env`-Datei ist unverschlüsselt (auf jedem Host brauchbar), und ein kompromittierter Schlüssel lässt sich nur über manuelles `decrypt-pii`→Key-Tausch→`encrypt-pii` ersetzen.

Constraints: einzelner IONOS VPS Linux XS (1 GB RAM), systemd, Go 1.26, `modernc.org/sqlite`. Serverseitige Entschlüsselung ist funktional zwingend (SEPA-XML-Builder `pain.008`, IBAN-Validierung, Fee-Run) — ein Live-Root-Angreifer auf dem laufenden Host ist damit prinzipiell nicht abwehrbar (dokumentierte Modellgrenze, kein Ziel dieses Changes).

## Goals / Non-Goals

**Goals:**
- Schlüssel **verschlüsselt + host-gebunden** auf der Platte (kopierter Blob auf anderem Host wertlos) und **nicht** in der Prozess-Umgebung (`/proc/environ`).
- **Schlüsselrotation** als idempotentes Subcommand (`rotate-key`), Format-versioniert (`"v2:"`).
- **Zero-Downtime + Abwärtskompatibilität:** Env-Fallback erhält lokale Dev und erlaubt stufenweisen Rollout (erst Code, dann Credential).
- Keine neue externe Abhängigkeit, kein nennenswerter RAM-Mehrbedarf, kein zusätzlicher Dienst.

**Non-Goals:**
- Schutz gegen Live-Root/RCE/Hoster/Memory-Dump des laufenden Hosts (Modellgrenze; serverseitige Entschlüsselung bleibt nötig).
- Off-box-Vault/KMS oder echtes E2E (siehe Decisions — verworfen/Eskalation).
- Änderung der Krypto-Primitive, der Verschlüsselungs-Stores oder der Berechtigungsregel `policy.CanDecryptBankData`.

## Decisions

### D1 — systemd-Credentials als primäre Schlüsselquelle
**Entscheidung:** `teamwerk.service` liefert den Schlüssel über `LoadCredentialEncrypted=field_key:/etc/teamwerk/field_key.cred` (bzw. `SetCredentialEncrypted=`). Der Blob wird mit `systemd-creds encrypt` erzeugt und ist host- bzw. `host+tpm2`-gebunden. systemd entschlüsselt ihn beim Start und legt den Klartext als Datei in einem per-Service-**tmpfs** unter `$CREDENTIALS_DIRECTORY` ab (Mode 0400, nur für den Service sichtbar) — **nicht** in der Umgebung.
**Warum:** entfernt die `/proc/environ`-Exposition und macht die at-rest-Ablage host-gebunden, ohne neuen Dienst/RAM-Bedarf. systemd ist bereits die Service-Verwaltung.

### D2 — Schlüsselquelle mit Fallback-Reihenfolge
**Entscheidung:** `internal/crypto` liest in dieser Reihenfolge: (1) Datei `$CREDENTIALS_DIRECTORY/field_key`, falls vorhanden; (2) `FIELD_ENCRYPTION_KEY` aus der Umgebung. Erste nutzbare Quelle gewinnt; Validierung (base64, 32 Byte) bleibt unverändert.
**Warum:** Zero-Downtime-Migration (neuer Code läuft mit alter Env weiter; Credential wird unabhängig umgestellt) und lokale Entwicklung bleibt per `.env` möglich (kein systemd nötig). Der Startup-Check meldet beim Boot, welche Quelle genutzt wurde.

### D3 — Versioniertes Format `"v2:"` + `rotate-key`
**Entscheidung:** Rotation erzeugt einen neuen Schlüssel, entschlüsselt jeden `"v1:"`-Wert mit dem alten und schreibt ihn als `"v2:"` mit dem neuen zurück (Dateien analog via Magic-Header v2, atomic rename). `Decrypt` unterscheidet die Version am Präfix und wählt den passenden Schlüssel; Klartext-Passthrough bleibt. Während der Rotation müssen **beide** Schlüssel verfügbar sein (alt + neu).
**Warum:** Austausch eines kompromittierten Schlüssels ohne Schema-Änderung; idempotent (bereits `"v2:"` wird übersprungen) und auf dem bestehenden `encrypt-pii`/`decrypt-pii`-Iterationsmechanismus aufgebaut.
**Offen (für Tasks):** Übergabe des Alt-Schlüssels an `rotate-key` (zweite Credential `field_key_old` bzw. Flag/Env `FIELD_ENCRYPTION_KEY_OLD`).

### D4 — TPM2 optional, host-key als Fallback
**Entscheidung:** Beim Provisionieren `systemd-creds encrypt --with-key=auto` (nutzt TPM2, falls vorhanden, sonst host-key `/var/lib/systemd/credential.secret`). Das Deploy-Skript prüft TPM-Verfügbarkeit und protokolliert die gewählte Bindung.
**Warum:** IONOS-VPS hat evtl. kein TPM2; host-key-Binding ist bereits ein Gewinn (Blob nicht portierbar, nicht in Env) und bleibt wartungsarm.

### D5 — Kein Vault/KMS auf demselben Host (verworfen, Option B dokumentiert)
**Entscheidung:** Kein OpenBao/HashiCorp Vault/Cloud-KMS in diesem Change.
**Begründung:** Ein Vault auf **demselben** VPS verschiebt nur das Bootstrap-Problem (Unseal-Key/Token liegt wieder lokal) bei hohem Betriebsaufwand und RAM-Druck (1 GB). Echter Gewinn nur **off-box** (zweiter Host / SaaS-Free-Tier wie HCP Vault Secrets, Infisical Cloud) — Preis: neue Infrastruktur bzw. Drittanbieter-Trust + Netzabhängigkeit beim Boot. Cloud-KMS hat keinen echten Free-Tier. **Re-Evaluierung**, falls „Master-Key muss off-box liegen" zur harten Anforderung wird.

### D6 — Kein Passphrase-beim-Start, kein E2E
**Entscheidung:** Beide verworfen. Passphrase-Eintrag bei jedem Start tötet die unbeaufsichtigte Verfügbarkeit (Reboot/Crash → manuelles Unseal). Echtes E2E/Pro-Nutzer-Schlüssel (= das in `encrypt-bank-sepa-at-rest` verworfene A2) nähme den Server aus der Trust-Grenze, bricht aber serverseitigen SEPA-Builder/Self-Service/Passwort-Reset — bleibt dokumentierte Eskalation für „Hoster darf prinzipiell nie entschlüsseln".

## Risks / Trade-offs

- **systemd-Version/TPM:** `LoadCredentialEncrypted=` braucht systemd ≥ 250; verschlüsselte Credentials ≥ 251. Mitigation: Deploy-Skript prüft `systemd --version` und TPM, fällt sonst auf host-key zurück bzw. meldet, dass der Env-Fallback (D2) aktiv bleibt.
- **Host-Bindung = kein Restore auf neuem Host:** Ein nur host-key-gebundener Blob lässt sich auf einer **neuen** Maschine nicht entschlüsseln. Mitigation: weiterhin den **rohen** Schlüssel separat sichern (Passwort-Manager); der Blob ist Komfort/Härtung, nicht das Backup. In `10-deployment.md` betonen.
- **Rotation braucht zwei Schlüssel gleichzeitig:** Fehlbedienung (alter Schlüssel weg) ⇒ `"v1:"`-Werte unlesbar. Mitigation: `rotate-key` bricht ab, wenn der Alt-Schlüssel fehlt oder `"v1:"`-Werte nicht entschlüsselbar sind; DB-Backup vor Rotation (Skript erzwingt `make backup`).
- **Restrisiko unverändert:** Live-Root liest den Klartext aus dem tmpfs/RAM — durch dieses Modell nicht abwehrbar (akzeptiert). Der Gewinn ist „nicht in `/proc/environ`" + „Blob nicht portierbar" + „rotierbar".
- **Zwei Schlüsselquellen = Verwechslungsgefahr:** Mitigation: Startup-Check loggt eindeutig die aktive Quelle; Doku beschreibt die Reihenfolge.
