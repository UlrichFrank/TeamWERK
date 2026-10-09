## ADDED Requirements

### Requirement: Spiegelung der Prod-Daten auf die Beta
`make mirror-beta FORCE=1` SHALL die Beta-Daten durch einen aktuellen Prod-Stand ersetzen: SQLite-DB, Dokumente (`files`), Beitragslauf-Protokolle und BWHV-Spielberichte. Videos und Bild-Ablagen (`uploads`, `media`, `training-diary`, `match-report-images`) werden NICHT gespiegelt; die entsprechenden Beta-Verzeichnisse werden geleert. Die Spiegelung läuft vollständig auf dem VPS; Daten verlassen den Server nicht.

#### Scenario: Ohne Bestätigung keine Spiegelung
- **WHEN** `make mirror-beta` ohne `FORCE=1` aufgerufen wird
- **THEN** bricht das Target ab, bevor eine Verbindung zum Server aufgebaut wird

#### Scenario: Beta trägt den Prod-Stand
- **WHEN** die Spiegelung erfolgreich war
- **THEN** kann sich ein Prod-Nutzer mit seinem Prod-Passwort auf der Beta anmelden und sieht seine Termine und Dienste

#### Scenario: Keine Bilder und Videos
- **WHEN** die Spiegelung erfolgreich war
- **THEN** enthalten die Beta-Verzeichnisse `uploads`, `media`, `training-diary`, `match-report-images` und `videos` keine Dateien (leere Unterverzeichnisse, die der Server selbst anlegt, sind erlaubt)

### Requirement: Spiegelung liest Prod nur
Die Spiegelung SHALL Prod ausschließlich lesen: die DB über die SQLite-Online-Backup-API aus einer `-readonly`-Verbindung, alle Lese- und Kopierschritte als `www-data` mit niedrigster CPU- und IO-Priorität. Jedes Schreibziel MUSS unter `/var/lib/teamwerk-beta` liegen, sonst bricht das Skript ab. Gestoppt und gestartet wird ausschließlich `teamwerk-beta`.

#### Scenario: Prod-Dienst läuft weiter
- **WHEN** die Spiegelung läuft
- **THEN** behält der Dienst `teamwerk` seine PID und antwortet durchgehend

#### Scenario: Schreibziel außerhalb der Beta
- **WHEN** ein Schreibziel des Skripts nicht unter `/var/lib/teamwerk-beta` liegt
- **THEN** bricht das Skript ab, bevor es schreibt

#### Scenario: Defekte Kopie
- **WHEN** die Integritätsprüfung der kopierten DB fehlschlägt
- **THEN** bleibt die bisherige Beta-DB unverändert in Betrieb

### Requirement: Bereinigte Kopie
Vor dem Einsetzen SHALL die Kopie bereinigt werden: alle Push-Abos und Refresh-Tokens gelöscht, Wartungsmodus aus.

#### Scenario: Keine Push-Abos in der Beta
- **WHEN** die Spiegelung erfolgreich war
- **THEN** enthält `push_subscriptions` der Beta keine Zeile
