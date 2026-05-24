## ADDED Requirements

### Requirement: Produktionsdatenbank sichern
Der Entwickler SHALL mit einem einzigen `make`-Befehl ein konsistentes Backup der laufenden Produktionsdatenbank erstellen und lokal herunterladen können, ohne den Produktionsservice zu unterbrechen.

#### Scenario: Backup erstellen und herunterladen
- **WHEN** der Entwickler `make backup` ausführt
- **THEN** wird per SSH auf dem VPS `sqlite3 <DB_PATH> ".backup /tmp/teamwerk-backup.db"` ausgeführt
- **THEN** wird die Backup-Datei per `scp` nach `./teamwerk-backup.db` heruntergeladen
- **THEN** wird die temporäre Backup-Datei auf dem VPS gelöscht

### Requirement: Backup lokal einspielen
Der Entwickler SHALL ein heruntergeladenes Backup als lokale Entwicklungsdatenbank einspielen können, mit einer expliziten Bestätigungsabfrage vor dem Überschreiben.

#### Scenario: Restore mit Bestätigung
- **WHEN** der Entwickler `make restore-local` ausführt
- **THEN** wird eine Warnung ausgegeben, dass `./teamwerk.db` überschrieben wird
- **THEN** wird eine Bestätigung (`y/N`) abgefragt
- **THEN** bei Bestätigung wird `./teamwerk-backup.db` nach `./teamwerk.db` kopiert

#### Scenario: Restore abgebrochen
- **WHEN** der Entwickler `make restore-local` ausführt und die Bestätigung verneint
- **THEN** wird keine Änderung an `./teamwerk.db` vorgenommen

### Requirement: Kombiniertes Pull-Target
Der Entwickler SHALL mit `make pull-db` backup, download und lokales Einspielen in einem Schritt ausführen können.

#### Scenario: pull-db kombiniert alle Schritte
- **WHEN** der Entwickler `make pull-db` ausführt
- **THEN** wird `backup` ausgeführt (Online-Backup + Download + Cleanup auf VPS)
- **THEN** wird `restore-local` ausgeführt (mit Bestätigungsabfrage)
