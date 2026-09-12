## ADDED Requirements

### Requirement: Deploy verifiziert den Prozessstart und hat einen Rückweg

`make deploy` MUST nach dem Neustart den Health-Endpunkt abfragen und bei Fehlschlag mit klarer Meldung abbrechen. Vor dem Austausch MUST das laufende Binary als Vorgängerversion gesichert werden; `make deploy-rollback` MUST diese Version zurückspielen, neu starten und ebenfalls den Health-Endpunkt prüfen. Der Server MUST ein tägliches, serverseitiges Backup von Datenbank und Storage-Pfaden mit 14 Tagen Aufbewahrung anlegen.

#### Scenario: Prozess kommt nicht hoch
- **WHEN** der neu gestartete Prozess innerhalb von 30 Sekunden nicht auf `/api/healthz` antwortet
- **THEN** meldet `make deploy` einen Fehler und nennt `make deploy-rollback`

#### Scenario: Rollback
- **WHEN** `make deploy-rollback` läuft
- **THEN** läuft danach das vorherige Binary und der Health-Endpunkt antwortet

#### Scenario: Tägliches Backup
- **WHEN** der Backup-Cron läuft
- **THEN** liegt unter `/var/backups/teamwerk/` ein datierter Ordner mit DB-Kopie und Storage-Archiv, ältere als 14 Tage sind entfernt
