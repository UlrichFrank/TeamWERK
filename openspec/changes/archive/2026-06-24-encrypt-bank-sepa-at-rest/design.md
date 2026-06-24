## Context

Zahlungs-PII (Mitglieds-IBAN + Kontoinhaber, eingereichte Bankdaten-Entwürfe, SEPA-Mandat-PDFs, Vereins-SEPA-Stammdaten) liegt heute im Klartext in SQLite (`members`, `member_change_drafts`, `clubs`) und auf der Platte (`sepa_mandat_path`). Zugriff ist heute rein über Route-Authz beschränkt: `GET /api/members/{id}`, `POST /api/fee-run/export`, `GET /api/club` sind auf `vorstand`/`kassierer` (Admin umgeht) gegated. Eigentümer/Eltern können Bankdaten **schreiben** (change-request, `UpdateChildBank`), aber **nicht zurücklesen** — es existiert heute kein Lesepfad, der einem Mitglied seine eigene IBAN liefert.

Eine frühere clientseitige Envelope-Variante (Branch `origin/encryption`, archivierter Change `2026-06-05-member-data-encryption`, Specs `member-encryption` + `vorstand-vault`) wurde nie nach `main` übernommen. Sie führte eine separate `member_sensitive`-Tabelle, Dual-Key-DEK, einen WebCrypto-Vorstands-Tresor (`sessionStorage`) und einen clientseitigen CSV/Export ein. Der Branch ist zudem aus der Pre-SEPA-Beitragslauf-Ära (er löscht `internal/sepa/*`, `internal/upload/bulk_sepa*`, `sepa_token`) und damit ~120k Zeilen von `main` entfernt — nicht wiederverwendbar.

Constraints: Go 1.26, `modernc.org/sqlite` (pure Go, **kein CGo**), 1 GB RAM auf dem VPS, golang-migrate (nur SQL), Stateless-JWT (15-min-Access, 7-Tage-Refresh), zentrale Authz-Schicht `internal/policy` (Principal, `IsVorstandLike`, `IsKassiererLike`, `isParentOf`).

## Goals / Non-Goals

**Goals:**
- Bank-/SEPA-PII der vier Speicher ist **at-rest verschlüsselt** (AES-256-GCM); ein geleaktes Backup / eine gestohlene `.db` / ein Open-Source-Dump enthält keine Klartext-IBANs mehr.
- Entschlüsseln ist zentral autorisiert: Eigentümer ∨ Eltern ∨ admin/vorstand/kassierer.
- Eigentümer/Eltern können ihre Bankdaten erstmals **auch lesen** (neuer Lesepfad).
- Rollout ohne Downtime/Wartungsfenster und ohne Nutzer-Interaktion; Erstverschlüsselung idempotent.
- Keine neue externe Abhängigkeit, kein nennenswerter RAM-Mehrbedarf.

**Non-Goals:**
- Schutz gegen vollen Server-Compromise / Hoster-Zugriff (Key + DB am selben Ort) — siehe Decision Record A1 vs A2.
- Schutz gegen geklaute, autorisierte Sessions (Phishing eines Vorstands).
- Clientseitige Krypto / WebCrypto / Vorstands-Tresor (durch A1 abgelöst).
- IBAN-Suche/-Aggregation in SQL (entfällt bewusst).
- Verschlüsselung anderer PII (Geburtsdatum/Adresse) — bleibt außerhalb dieses Changes.

## Decisions

### D1 — A1 (app-gehaltener Schlüssel), NICHT A2 (Pro-Nutzer-/E2E)
**Entscheidung:** Ein serverseitiger, app-gehaltener Schlüssel (`FIELD_ENCRYPTION_KEY` aus der Umgebung) ver-/entschlüsselt die Felder. „Wer darf entschlüsseln" bleibt eine **Authz-Entscheidung** der App; die Verschlüsselung ist Defense-in-Depth gegen At-Rest-Lecks.

**Alternative A2 (verworfen):** Pro-Nutzer-Schlüsselpaare mit Envelope-Verschlüsselung (DEK pro Datensatz, gewrappt für Eigentümer-, Eltern- und Rollen-Public-Key; Privatschlüssel passwort-abgeleitet).
- A2-Mehrwert gegenüber A1: **ausschließlich** Schutz gegen vollen Server-Compromise/Hoster. Gegen geleaktes Backup schützt A1 gleichwertig; gegen geklaute autorisierte Session schützt auch A2 nicht.
- A2-Kosten, die zur Verwerfung führten:
  1. Mitglieder müssen ihre **eigenen** Daten lesen → faktisch wird **jeder** Nutzer Schlüsselträger; eine kryptografische Beschränkung auf 3 Rollen ist damit unhaltbar.
  2. Kollidiert mit Stateless-JWT: der entpackte Schlüssel müsste entweder in den Server-RAM (dann wieder server-compromise-anfällig — A2-Nutzen futsch) oder ausschließlich in den Browser (echtes E2E).
  3. Echtes E2E macht den Server blind → der gesamte SEPA-XML-Builder (pain.001.002.03), IBAN-Validierung/-Normalisierung und die Preview-/Ausschlusslogik müssten in den Browser portiert werden.
  4. Passwort-Reset braucht einen Key-Recovery-Flow, sonst Datenverlust.
  5. Vereinsfunktion/Elternteil zuweisen würde von einem DB-INSERT zu einer asynchronen Krypto-Wrapping-Operation.
  6. Neue Totalverlust-Risiken (alle Rollenträger verlieren Passwort = alle Bankdaten verloren).
- **Re-Evaluierung nur**, falls „Hoster/Server-Admin darf prinzipiell nicht entschlüsseln" zur harten Anforderung wird. Die frühere A2-Umsetzung (Branch `encryption`) bleibt als Referenz im Git-Verlauf.

### D2 — AES-256-GCM aus der stdlib, versioniertes Format
**Entscheidung:** `crypto/aes` + `crypto/cipher` (GCM). Format `"v1:" + base64(nonce ‖ ciphertext)` mit zufälligem 12-Byte-Nonce pro Verschlüsselung. Kein CGo, keine neue Abhängigkeit, GCM liefert Authentizität (Manipulationserkennung).
**Warum versioniert:** Der `"v1:"`-Prefix macht (a) die Migration idempotent („schon verschlüsselt?"), (b) spätere Schlüsselrotation/Re-Encrypt möglich, ohne das Schema zu ändern.
**Alternative (verworfen):** SQLCipher / Full-DB-Encryption — bräuchte CGo, kollidiert mit `modernc.org/sqlite`.

### D3 — Toleranter `Decrypt` für Zero-Downtime-Rollout
**Entscheidung:** `Decrypt(v)` gibt Werte **ohne** `"v1:"`-Prefix unverändert zurück (= noch Klartext). Dadurch liest neuer Code eine gemischte DB korrekt, und „Code deployen" ist von „Daten verschlüsseln" entkoppelt — kein Wartungsfenster.
**Trade-off:** Ein theoretisch denkbarer Klartext-Wert, der zufällig mit `"v1:"` beginnt, würde fehlinterpretiert. Praktisch ausgeschlossen für IBAN/BIC/Kontoinhaber; bei PDFs wird stattdessen ein Magic-Header genutzt.

### D4 — In-place-Verschlüsselung in bestehenden Spalten (keine `member_sensitive`-Tabelle)
**Entscheidung:** Ciphertext wird in den vorhandenen `TEXT`-Spalten gespeichert (`members.iban/account_holder`, `clubs.*`, `member_change_drafts.new_value`), PDFs als verschlüsselte Datei am selben Pfad. Keine Schema-Migration für den Speicherort nötig.
**Warum:** A1 braucht keine Wrapped-DEK-Spalten (das war A2-spezifisch). Minimaler Eingriff, kein Datenmodell-Umbau.

### D5 — Zentrale Autorisierung `policy.CanDecryptBankData`
**Entscheidung:** Eine Funktion in `internal/policy`: `CanDecryptBankData(p *Principal, memberUserID int) bool` = `IsVorstandLike(p) ∨ IsKassiererLike(p) ∨ admin ∨ (p.UserID == memberUserID) ∨ isParentOf(p.UserID, member)`. Jeder Lesepfad ruft sie auf, bevor `Decrypt` ausgegeben wird.
**Warum:** Verhindert, dass ein einzelner Handler die Regel versehentlich falsch/lückenhaft implementiert (IDOR). Eltern-Prüfung braucht DB-Zugriff (`family_links`) — die Signatur erlaubt eine DB-gestützte Variante analog zu `FolderAccess`.

### D6 — Erstverschlüsselung als Go-Subcommand `encrypt-pii`, nicht als SQL-Migration
**Entscheidung:** Ein idempotentes Subcommand iteriert die vier Speicher, verschlüsselt jeden Wert ohne `"v1:"`-Prefix (PDFs ohne Magic-Header) und schreibt zurück (Dateien via atomic rename).
**Warum:** AES gehört nicht in `.up.sql`; golang-migrate kann es nicht. Idempotenz erlaubt Wiederholung nach Abbruch.

## Risks / Trade-offs

- **Schlüsselverlust = Datenverlust** → Schlüssel separat vom DB-Backup sichern (Passwort-Manager); Startup-Check verhindert versehentlichen Betrieb ohne Schlüssel; Doku in `10-deployment.md`.
- **Key + DB am selben Server** (A1-Grenze, kein Schutz gegen Server-Compromise) → bewusst akzeptiert; im Security-Abschnitt dokumentiert, Fokus auf Backup-Trennung + Patch-/Least-Privilege-Hygiene.
- **IDOR an einem vergessenen Lesepfad** → größtes Restrisiko; Mitigation: zentrale `CanDecryptBankData` + Test pro Lesepfad (auch Negativfälle: Trainer, fremdes Mitglied).
- **IBAN nicht mehr SQL-durchsuchbar** → heute irrelevant (Bulk-Match über Namen); festgehalten, falls je IBAN-Suche gewünscht.
- **Falsche Rollout-Reihenfolge** (Code vor Key) → Startup-Check bootet ohne Key gar nicht; toleranter Decrypt macht „encrypt-pii später" unkritisch.
- **PDF-Migration bricht ab** → Magic-Header-Erkennung + atomic rename → Wiederholung überspringt bereits verschlüsselte Dateien.

## Migration Plan

1. **Schlüssel erzeugen:** `teamwerk gen-encryption-key` → `FIELD_ENCRYPTION_KEY` in `/etc/teamwerk/env` (chmod 600). **Separat sichern** (nicht ins DB-Backup).
2. **Deploy:** `make deploy` (Binary + `systemctl restart`). Startup-Check validiert den Schlüssel. Ab jetzt: jeder Schreibvorgang verschlüsselt; Lesen versteht Klartext + Ciphertext (toleranter Decrypt).
3. **Erstverschlüsselung (einmalig, bei laufender App):** `teamwerk encrypt-pii`. Idempotent; bei Abbruch erneut ausführbar.
4. **Verifikation:** Stichprobe in der DB zeigt `"v1:"`-Prefixe; SEPA-Export und Profil-Lesen funktionieren.

**Rollback:** Da `Decrypt` tolerant ist und der alte Code Klartext erwartet, ist ein reiner Code-Rollback **nach** `encrypt-pii` nicht trivial (alter Code liest dann Ciphertext). Rollback-Strategie: Code-Version mit `internal/crypto` behalten und nur `encrypt-pii` aussetzen; im Notfall ein `decrypt-pii`-Lauf (spiegelbildlich, ebenfalls idempotent) vor dem Downgrade. DB-Backup vor Schritt 3 ziehen.

## Security (Threat Model)

| Angriff | Durch A1 geschützt? |
|---|---|
| Gestohlenes DB-Backup / DB-File allein | ✓ JA (Kerngewinn) |
| Verlorenes Laptop-/Cloud-Backup | ✓ JA |
| Open-Source-Dump leakt PII | ✓ JA |
| Neugieriger Trainer/Mitglied via API (IDOR) | ⚠ über Authz (`CanDecryptBankData`), nicht Krypto |
| Geklaute Vorstands-/Kassierer-Session (Phishing) | ✗ NEIN (darf legitim entschlüsseln) |
| Voller Server-Compromise (RCE/root) | ✗ NEIN (Key + DB am selben Ort) |
| Memory-/Core-Dump des Prozesses | ✗ NEIN |
| Logs | ✓ heute kein IBAN-Logging (verifiziert) |
| SEPA-XML-Export | nur HTTP-Response an Browser, nicht auf Server-Platte (verifiziert) |
| Beitragslauf-Protokoll | nur MemberNumber/Name/BetragCent/Success — keine IBAN (verifiziert) |

**Größtes Restrisiko ist nicht die Krypto, sondern (a) die App-Authz-Oberfläche (IDOR) und (b) die Key/Backup-Trennung.** Dort liegt der Test-/Ops-Fokus.

## Open Questions

- Soll `encrypt-pii` auch ein spiegelbildliches `decrypt-pii` (für Rollback/Schlüsselrotation) mitbringen, oder reicht zunächst nur Vorwärts-Migration? (Empfehlung: `decrypt-pii` gleich mitliefern, geringer Mehraufwand, rettet den Rollback-Fall.)
- Eltern-Prüfung in `CanDecryptBankData`: DB-gestützt (Query pro Aufruf) vs. vorab im `Principal` gecachte Kind-IDs — abhängig davon, ob `claims`/`Principal` die family-Kontexte bereits trägt.
