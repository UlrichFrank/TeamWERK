## REMOVED Requirements

### Requirement: Zentrale Autorisierung der Entschlüsselung

**Grund:** Der Server entschlüsselt Bank-/SEPA-PII nicht mehr (Zero-Knowledge at rest).
`policy.CanDecryptBankData` und alle serverseitigen `crypto.Decrypt`-Lesepfade für diese
Felder entfallen; Berechtigung wird durch Schlüsselbesitz (Eigentümer-/Gruppen-Wrap)
ausgedrückt, nicht durch eine server-seitige Authz-Regel. Ersetzt durch die Anforderung
„Berechtigtes Lesen ausschließlich durch Eigentümer und Finance-Gruppe" in
`client-side-bank-encryption`.

### Requirement: Eigentümer- und Eltern-Lesen der eigenen Bankdaten

**Grund:** Server-Endpoints liefern keinen Klartext mehr. Eigentümer lesen ihre Daten
clientseitig über ihren Eigentümer-Wrap; das Eltern-Lesen entfällt ersatzlos (Kinderdaten
sind nur an die Finance-Gruppe gewrappt — siehe `client-side-bank-encryption`).

## MODIFIED Requirements

### Requirement: App-gehaltener Schlüssel nur als Migrations-Brücke

Das System SHALL den symmetrischen Schlüssel `FIELD_ENCRYPTION_KEY` **ausschließlich für
die einmalige Migration** des `"v1:"`-Bestands auf das clientseitige Envelope-Format
laden. Nach abgeschlossener Migration SHALL der Schlüssel aus der Betriebsumgebung
entfernt werden und der Server SHALL ohne ihn starten können. Solange die Migration nicht
abgeschlossen ist, SHALL der bestehende toleranter `Decrypt` (Klartext-Passthrough,
`"v1:"`-Entschlüsselung) **nur im Migrationspfad** verfügbar sein, nicht in regulären
Lesepfaden.

#### Scenario: Server startet nach Migration ohne Schlüssel
- **WHEN** die Migration abgeschlossen ist und `FIELD_ENCRYPTION_KEY` nicht gesetzt ist
- **THEN** startet der Server normal und bedient alle Bank-/SEPA-Routen (nur Ciphertext)

#### Scenario: Regulärer Lesepfad entschlüsselt nicht serverseitig
- **WHEN** eine reguläre Route Bank-/SEPA-Felder ausliefert
- **THEN** verwendet sie keinen `crypto.Decrypt`-Aufruf und liefert ausschließlich Ciphertext

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
