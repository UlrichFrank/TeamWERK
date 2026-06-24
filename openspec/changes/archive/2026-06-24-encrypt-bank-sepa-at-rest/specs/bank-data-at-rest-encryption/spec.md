## ADDED Requirements

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

`Decrypt` SHALL einen Wert ohne `"v1:"`-Prefix unverändert zurückgeben (Behandlung als noch nicht migrierter Klartext) und einen Wert mit `"v1:"`-Prefix mit dem konfigurierten Schlüssel entschlüsseln. Schlägt die Authentifizierung des Ciphertexts fehl, SHALL `Decrypt` einen Fehler zurückgeben und keinen Klartext liefern.

#### Scenario: Gemischter Bestand wird korrekt gelesen
- **WHEN** eine Spalte teils Klartext (vor Migration), teils `"v1:"`-Ciphertext (nach Migration) enthält
- **THEN** liefert der Lesepfad in beiden Fällen den korrekten Klartextwert

#### Scenario: Manipulierter Ciphertext wird abgewiesen
- **WHEN** ein `"v1:"`-Wert nachträglich verändert wurde (gebrochene GCM-Authentifizierung)
- **THEN** liefert `Decrypt` einen Fehler statt eines Klartextwerts

### Requirement: App-gehaltener Schlüssel mit Startup-Validierung

Das System SHALL den symmetrischen Schlüssel aus der Umgebungsvariable `FIELD_ENCRYPTION_KEY` (32 Byte, base64-kodiert) laden. Fehlt der Schlüssel oder ist er ungültig, SHALL der Serverstart fehlschlagen. Das System SHALL ein Subcommand `gen-encryption-key` bereitstellen, das einen gültigen Schlüssel erzeugt.

#### Scenario: Start ohne Schlüssel wird verweigert
- **WHEN** der Server ohne gesetztes `FIELD_ENCRYPTION_KEY` gestartet wird
- **THEN** bricht der Start mit einer eindeutigen Fehlermeldung ab und nimmt keine Requests an

#### Scenario: Ungültiger Schlüssel wird verweigert
- **WHEN** `FIELD_ENCRYPTION_KEY` gesetzt, aber kein gültiger base64-kodierter 32-Byte-Wert ist
- **THEN** bricht der Start mit einer eindeutigen Fehlermeldung ab

#### Scenario: Schlüsselerzeugung
- **WHEN** `teamwerk gen-encryption-key` ausgeführt wird
- **THEN** gibt es einen base64-kodierten 32-Byte-Schlüssel aus, der als `FIELD_ENCRYPTION_KEY` verwendbar ist

### Requirement: Zentrale Autorisierung der Entschlüsselung

Das System SHALL Klartext-Bankdaten eines Mitglieds nur dann ausliefern, wenn der Aufrufer berechtigt ist: `admin` ODER Vereinsfunktion `vorstand` ODER `kassierer` ODER der Eigentümer (das verknüpfte Mitglied selbst) ODER ein über `family_links` verbundenes Elternteil. Diese Regel SHALL zentral in `policy.CanDecryptBankData` implementiert und von jedem Lesepfad aufgerufen werden. Nicht-berechtigte Aufrufer SHALL keine entschlüsselten Bankdaten erhalten.

#### Scenario: Vorstand/Kassierer/Admin liest jedes Mitglied
- **WHEN** ein Nutzer mit Rolle `admin` oder Vereinsfunktion `vorstand`/`kassierer` `GET /api/members/{id}` aufruft
- **THEN** enthält die Antwort IBAN und Kontoinhaber im Klartext

#### Scenario: Trainer erhält keine Bankdaten
- **WHEN** ein Nutzer ausschließlich mit Funktion `trainer` Bankdaten eines Mitglieds anfordert
- **THEN** erhält er keine entschlüsselten Bankdaten (Feld leer/weggelassen bzw. HTTP 403 am dedizierten Endpoint)

#### Scenario: Fremdes Mitglied erhält keine Bankdaten
- **WHEN** ein Mitglied die Bankdaten eines anderen, nicht verbundenen Mitglieds anfordert
- **THEN** erhält es keine entschlüsselten Bankdaten

### Requirement: Eigentümer- und Eltern-Lesen der eigenen Bankdaten

Das System SHALL einem Mitglied die eigenen Bankdaten (IBAN, Kontoinhaber) entschlüsselt über `GET /api/profile/me` liefern und einem Elternteil die Bankdaten seines verbundenen Kindes über `GET /api/profile/kind/{id}`. Das Frontend SHALL diese Werte in Profil bzw. Kind-Profil anzeigen.

#### Scenario: Mitglied liest eigene IBAN
- **WHEN** ein Mitglied mit verknüpftem Nutzerkonto `GET /api/profile/me` aufruft
- **THEN** enthält die Antwort die eigene IBAN und den Kontoinhaber im Klartext

#### Scenario: Elternteil liest Bankdaten des Kindes
- **WHEN** ein über `family_links` verbundenes Elternteil `GET /api/profile/kind/{id}` für sein Kind aufruft
- **THEN** enthält die Antwort die IBAN und den Kontoinhaber des Kindes im Klartext

#### Scenario: Elternteil eines fremden Kindes
- **WHEN** ein Nutzer `GET /api/profile/kind/{id}` für ein nicht mit ihm verbundenes Mitglied aufruft
- **THEN** antwortet der Server mit HTTP 403 und liefert keine Bankdaten

### Requirement: Idempotente Erstverschlüsselung des Bestands

Das System SHALL ein Subcommand `encrypt-pii` bereitstellen, das alle Bestandswerte der vier Speicher ohne `"v1:"`-Prefix (bzw. PDFs ohne Magic-Header) verschlüsselt und zurückschreibt (Dateien via atomic rename). Wiederholte Ausführung SHALL bereits verschlüsselte Werte überspringen und keinen doppelt verschlüsselten Wert erzeugen.

#### Scenario: Bestand wird verschlüsselt
- **WHEN** `teamwerk encrypt-pii` auf einer DB mit Klartext-Bankdaten ausgeführt wird
- **THEN** tragen anschließend alle betroffenen Werte den `"v1:"`-Prefix und PDFs den Magic-Header

#### Scenario: Wiederholter Lauf ist idempotent
- **WHEN** `teamwerk encrypt-pii` ein zweites Mal ausgeführt wird
- **THEN** bleiben bereits verschlüsselte Werte unverändert (kein Doppel-Encrypt)
