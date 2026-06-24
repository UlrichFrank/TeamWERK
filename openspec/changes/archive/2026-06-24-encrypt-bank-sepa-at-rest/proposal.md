## Why

Bankdaten (IBAN, Kontoinhaber), eingereichte Bankdaten-Entwürfe, SEPA-Mandat-PDFs und die SEPA-Stammdaten des Vereins liegen heute im Klartext in SQLite bzw. auf der Platte. Ein geleaktes DB-Backup, eine gestohlene `.db`-Datei oder ein versehentlich veröffentlichter Dump (relevant für die geplante Open-Source-Stellung) gibt damit sämtliche Zahlungs-PII preis. Eine frühere, clientseitige Envelope-Variante (Branch `encryption`, Specs `member-encryption`/`vorstand-vault`) wurde nie nach `main` übernommen und ist durch ihren Umfang (WebCrypto-Tresor, separate Tabelle, blinder Server) für den Betrieb zu schwer. Wir brauchen einen pragmatischen, serverseitigen Schutz, der gegen das häufigste reale Leck (Backup/DB-Datei) wirkt, ohne SEPA-Export und Self-Service umzubauen.

## What Changes

- **Neues Package `internal/crypto`**: AES-256-GCM (stdlib, kein CGo) mit versioniertem Format `"v1:" + base64(nonce‖ciphertext)`. `Decrypt` ist **tolerant** — Werte ohne `"v1:"`-Prefix gelten als (noch) Klartext und werden unverändert zurückgegeben. Das ermöglicht Zero-Downtime-Rollout und eine idempotente Migration.
- **Schlüssel aus Umgebung**: `FIELD_ENCRYPTION_KEY` (32 Byte, base64) in `/etc/teamwerk/env`. Startup-Check verweigert den Boot bei fehlendem/ungültigem Schlüssel. Neues Subcommand `gen-encryption-key`.
- **At-Rest-Verschlüsselung von vier Speichern** (in-place, keine neue Tabelle): (1) `members.iban` + `account_holder`, (2) `member_change_drafts` mit `field_name='bankdaten'`, (3) `clubs.iban/bic/glaeubiger_id/kontoinhaber`, (4) SEPA-Mandat-PDFs (Dateiinhalt, Pfad `sepa_mandat_path`).
- **Zentrale Entschlüsselungs-Autorisierung**: neue Funktion `policy.CanDecryptBankData(p, memberUserID)` = Eigentümer (Mitglied selbst) ∨ Eltern (`family_links`) ∨ `admin`/`vorstand`/`kassierer`. Jeder Lesepfad ruft sie auf.
- **NEU: Eigentümer-/Eltern-Lesen** (existiert heute nicht): `GET /api/profile/me` und `GET /api/profile/kind/{id}` liefern dem Berechtigten die eigene IBAN + Kontoinhaber entschlüsselt zurück, inkl. Frontend-Anzeige.
- **Einmalige Daten-Migration als Go-Subcommand `encrypt-pii`** (nicht als SQL-Migration — AES gehört nicht in `.up.sql`): verschlüsselt alle Bestands-Zeilen der vier Speicher und bestehende PDF-Dateien, idempotent über den `"v1:"`-Prefix bzw. einen Magic-Header.
- **BREAKING (Spec-Ebene): Ablösung der A2-Specs** `member-encryption` und `vorstand-vault` — die clientseitige Envelope-/Tresor-Variante wird durch A1 ersetzt.
- **Deployment-Doku + Backup-Regel**: Schlüssel und DB-Backup nie am selben Ort; Schlüsselverlust = Datenverlust.

## Capabilities

### New Capabilities
- `bank-data-at-rest-encryption`: Serverseitige AES-256-GCM-Verschlüsselung der Bank-/SEPA-PII (vier Speicher), versioniertes Ciphertext-Format, app-gehaltener Schlüssel aus der Umgebung, Startup-Validierung, idempotente Erstverschlüsselung via `encrypt-pii`, sowie die zentrale Entschlüsselungs-Autorisierung (Eigentümer ∨ Eltern ∨ admin/vorstand/kassierer) inkl. neuem Eigentümer-/Eltern-Lesen.

### Modified Capabilities
- `member-encryption`: **REMOVED** — die clientseitige WebCrypto-Envelope-Verschlüsselung mit `member_sensitive`-Tabelle und Dual-Key-DEK wird durch die serverseitige A1-Variante ersetzt (Begründung in `design.md`, Decision Record A1 vs A2).
- `vorstand-vault`: **REMOVED** — der browserseitige Vorstands-Tresor (Gruppenschlüssel in `sessionStorage`) entfällt mit dem Wechsel auf einen app-gehaltenen Schlüssel.

## Impact

- **Code:** neues `internal/crypto`; `internal/policy` (neue Regel + Tests); Schreibpfade in `internal/members` (`UpdateBankdaten`, `UpdateChildBank`, change-request-Create), `internal/config` (`UpdateClub`), `internal/upload` (SEPA-Upload + Bulk-Import, PDF-Inhalt); Lesepfade in `internal/members` (`Get`, `GetProfile`, `GetChildProfile`, change-drafts), `internal/beitragslauf` (`Export`), `internal/config` (`GetClub`), `internal/upload` (SEPA-Download); `cmd/teamwerk` (Subcommands `gen-encryption-key`, `encrypt-pii`, Startup-Check); `web/` (Eigentümer-/Eltern-Lesen-Anzeige in Profil/Kind-Profil).
- **Konfiguration/Betrieb:** neue Env-Variable `FIELD_ENCRYPTION_KEY`; Rollout-Sequenz und Backup-Trennung in `docs/agent/10-deployment.md`.
- **Daten:** keine neue Tabelle und kein neuer Spaltentyp; bestehende Spalten halten künftig Ciphertext. Einmaliger `encrypt-pii`-Lauf konvertiert Bestand.
- **Specs:** `member-encryption` und `vorstand-vault` werden entfernt; der archivierte Change `2026-06-05-member-data-encryption` und der Branch `encryption` gelten als abgelöst.
- **Bekannte Folge:** IBAN ist nach Verschlüsselung nicht mehr SQL-durchsuchbar/-vergleichbar (heute irrelevant — SEPA-Bulk-Match läuft über Namen).
- **RAM/VPS:** vernachlässigbar (stdlib-Krypto, keine neue Abhängigkeit).
