# bank-data-at-rest-encryption

## Purpose

Bank-/SEPA-PII wird at-rest mit AES-256-GCM verschlüsselt; der Bestand wird idempotent erstverschlüsselt. Entschlüsselung erfolgt ausschließlich clientseitig durch berechtigte Finance-Gruppen-Inhaber (Zero-Knowledge, Modell B) — der Server besitzt keinen Entschlüsselungsschlüssel. `FIELD_ENCRYPTION_KEY` war nur als einmalige Migrations-Brücke vorhanden und ist nach Abschluss der Migration entfernt.

## Requirements

### Requirement: At-Rest-Verschlüsselung der Bank-/SEPA-PII

Das System SHALL Bank-/SEPA-PII der folgenden vier Speicher ausschließlich als AES-256-GCM-Ciphertext im versionierten Format `"v1:" + base64(nonce ‖ ciphertext)` ablegen: (1) `members.iban` und `members.account_holder`, (2) `member_change_drafts` mit `field_name='bankdaten'` (`new_value`), (3) `clubs.iban`, `clubs.bic`, `clubs.glaeubiger_id`, `clubs.kontoinhaber`, (4) SEPA-Mandat-PDFs (Dateiinhalt unter `members.sepa_mandat_path`). Klartext dieser Felder SHALL die Datenbank bzw. die Platte nach abgeschlossener Migration nicht mehr verlassen.

#### Scenario: Bankdaten werden verschlüsselt gespeichert
- **WHEN** ein berechtigter Nutzer über `PUT /api/members/{id}/bank-details` IBAN und Kontoinhaber setzt
- **THEN** stehen in den Spalten `members.iban` und `members.account_holder` Werte mit `"v1:"`-Prefix, nicht der eingegebene Klartext

#### Scenario: SEPA-Mandat-PDF wird verschlüsselt abgelegt
- **WHEN** ein berechtigter Nutzer über `POST /api/upload/sepa-mandat/{id}` ein Mandat hochlädt
- **THEN** wird der Dateiinhalt unter `sepa_mandat_path` verschlüsselt (Magic-Header) gespeichert, nicht im Klartext

#### Scenario: Vereins-SEPA-Stammdaten werden verschlüsselt gespeichert
- **WHEN** ein Vorstand über `PUT /api/club` IBAN/BIC/Gläubiger-ID/Kontoinhaber setzt
- **THEN** stehen diese vier Felder in der `clubs`-Zeile mit `"v1:"`-Prefix

### Requirement: Toleranter Decrypt für gemischte Bestände

`Decrypt` SHALL **nur während der Migration** einen `"v1:"`-Wert mit dem konfigurierten
Schlüssel entschlüsseln und einen Wert ohne Prefix als Klartext durchreichen. Nach der
Migration existiert kein `"v1:"`- und kein Klartext-Bestand mehr in den Bank-/SEPA-Feldern;
diese tragen ausschließlich das clientseitige Envelope-Format.

#### Scenario: Migration liest Altbestand korrekt
- **WHEN** der Migrationslauf einen `"v1:"`-Wert oder einen Klartext-Altwert verarbeitet
- **THEN** liefert der Decrypt-Pfad den korrekten Klartext zur clientseitigen Re-Verschlüsselung

#### Scenario: Nach Migration kein Altformat mehr
- **WHEN** die Migration abgeschlossen ist
- **THEN** enthalten die Bank-/SEPA-Felder weder `"v1:"`- noch Klartextwerte

### Requirement: App-gehaltener Schlüssel nur als Migrations-Brücke

Das System SHALL den symmetrischen Schlüssel `FIELD_ENCRYPTION_KEY` **ausschließlich für
die einmalige Migration** des `"v1:"`-Bestands auf das clientseitige Envelope-Format
laden. Nach abgeschlossener Migration SHALL der Schlüssel aus der Betriebsumgebung
entfernt werden und der Server SHALL ohne ihn starten können. Solange die Migration nicht
abgeschlossen ist, SHALL der bestehende tolerante `Decrypt` (Klartext-Passthrough,
`"v1:"`-Entschlüsselung) **nur im Migrationspfad** verfügbar sein, nicht in regulären
Lesepfaden.

#### Scenario: Server startet nach Migration ohne Schlüssel
- **WHEN** die Migration abgeschlossen ist und `FIELD_ENCRYPTION_KEY` nicht gesetzt ist
- **THEN** startet der Server normal und bedient alle Bank-/SEPA-Routen (nur Ciphertext)

#### Scenario: Regulärer Lesepfad entschlüsselt nicht serverseitig
- **WHEN** eine reguläre Route Bank-/SEPA-Felder ausliefert
- **THEN** verwendet sie keinen `crypto.Decrypt`-Aufruf und liefert ausschließlich Ciphertext

### Requirement: Idempotente Erstverschlüsselung des Bestands

Das System SHALL ein Subcommand `encrypt-pii` bereitstellen, das alle Bestandswerte der vier Speicher ohne `"v1:"`-Prefix (bzw. PDFs ohne Magic-Header) verschlüsselt und zurückschreibt (Dateien via atomic rename). Wiederholte Ausführung SHALL bereits verschlüsselte Werte überspringen und keinen doppelt verschlüsselten Wert erzeugen.

#### Scenario: Bestand wird verschlüsselt
- **WHEN** `teamwerk encrypt-pii` auf einer DB mit Klartext-Bankdaten ausgeführt wird
- **THEN** tragen anschließend alle betroffenen Werte den `"v1:"`-Prefix und PDFs den Magic-Header

#### Scenario: Wiederholter Lauf ist idempotent
- **WHEN** `teamwerk encrypt-pii` ein zweites Mal ausgeführt wird
- **THEN** bleiben bereits verschlüsselte Werte unverändert (kein Doppel-Encrypt)
