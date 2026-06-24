## MODIFIED Requirements

### Requirement: App-gehaltener Schlüssel mit Startup-Validierung

Das System SHALL den symmetrischen Schlüssel in folgender Reihenfolge laden und die erste nutzbare Quelle verwenden: (1) eine von systemd bereitgestellte Credential-Datei unter `$CREDENTIALS_DIRECTORY/field_key`, (2) die Umgebungsvariable `FIELD_ENCRYPTION_KEY`. Der Wert MUSS ein base64-kodierter 32-Byte-Schlüssel sein. Ist keine Quelle gesetzt oder der gefundene Wert ungültig, SHALL der Serverstart fehlschlagen. Das System SHALL beim Start eindeutig protokollieren, aus welcher Quelle der Schlüssel geladen wurde, OHNE den Schlüssel selbst zu protokollieren. Das System SHALL ein Subcommand `gen-encryption-key` bereitstellen, das einen gültigen Schlüssel erzeugt.

#### Scenario: Start ohne jede Schlüsselquelle wird verweigert
- **WHEN** der Server ohne Credential-Datei und ohne gesetztes `FIELD_ENCRYPTION_KEY` gestartet wird
- **THEN** bricht der Start mit einer eindeutigen Fehlermeldung ab und nimmt keine Requests an

#### Scenario: Ungültiger Schlüssel wird verweigert
- **WHEN** die genutzte Schlüsselquelle keinen gültigen base64-kodierten 32-Byte-Wert enthält
- **THEN** bricht der Start mit einer eindeutigen Fehlermeldung ab

#### Scenario: Credential-Datei hat Vorrang vor der Umgebung
- **WHEN** sowohl `$CREDENTIALS_DIRECTORY/field_key` als auch `FIELD_ENCRYPTION_KEY` gesetzt sind
- **THEN** wird der Schlüssel aus der Credential-Datei verwendet und die genutzte Quelle protokolliert

#### Scenario: Fallback auf die Umgebungsvariable
- **WHEN** keine Credential-Datei vorhanden, aber `FIELD_ENCRYPTION_KEY` gesetzt und gültig ist
- **THEN** startet der Server mit dem Schlüssel aus der Umgebung (Abwärtskompatibilität, lokale Entwicklung)

#### Scenario: Schlüsselerzeugung
- **WHEN** `teamwerk gen-encryption-key` ausgeführt wird
- **THEN** gibt es einen base64-kodierten 32-Byte-Schlüssel aus, der als Credential bzw. als `FIELD_ENCRYPTION_KEY` verwendbar ist

## ADDED Requirements

### Requirement: Schlüsselrotation

Das System SHALL ein Subcommand `rotate-key` bereitstellen, das alle mit dem alten Schlüssel verschlüsselten Bestandswerte (Präfix `"v1:"` bzw. Datei-Magic-Header) entschlüsselt und mit dem neuen Schlüssel im nächsthöheren versionierten Format (`"v2:"` bzw. entsprechender Datei-Header) zurückschreibt (Dateien via atomic rename). Während der Rotation MÜSSEN alter und neuer Schlüssel verfügbar sein. `Decrypt` SHALL die Format-Version am Präfix erkennen und den passenden Schlüssel wählen; Werte ohne bekanntes Präfix gelten weiterhin als Klartext (Passthrough). Wiederholte Ausführung SHALL bereits rotierte Werte überspringen (idempotent). Fehlt der alte Schlüssel oder lässt sich ein Bestandswert nicht entschlüsseln, SHALL `rotate-key` abbrechen, ohne teilweise inkonsistent zu schreiben.

#### Scenario: Bestand wird auf den neuen Schlüssel rotiert
- **WHEN** `teamwerk rotate-key` mit altem und neuem Schlüssel auf einer DB mit `"v1:"`-Werten ausgeführt wird
- **THEN** tragen die betroffenen Werte anschließend das `"v2:"`-Format und sind nur noch mit dem neuen Schlüssel entschlüsselbar

#### Scenario: Wiederholte Rotation ist idempotent
- **WHEN** `teamwerk rotate-key` ein zweites Mal mit demselben neuen Schlüssel ausgeführt wird
- **THEN** bleiben bereits auf `"v2:"` rotierte Werte unverändert

#### Scenario: Fehlender Alt-Schlüssel bricht ab
- **WHEN** `rotate-key` ausgeführt wird, aber der alte Schlüssel zum Entschlüsseln vorhandener `"v1:"`-Werte fehlt
- **THEN** bricht der Lauf mit einer eindeutigen Fehlermeldung ab und verändert keine Daten

#### Scenario: Gemischter Bestand bleibt lesbar
- **WHEN** eine Spalte sowohl `"v1:"`- als auch `"v2:"`-Werte enthält und beide Schlüssel konfiguriert sind
- **THEN** liefert der Lesepfad in beiden Fällen den korrekten Klartext
