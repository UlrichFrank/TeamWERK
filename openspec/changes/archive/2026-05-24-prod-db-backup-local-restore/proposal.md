## Why

Für lokale Entwicklung und Debugging ist es hilfreich, einen aktuellen Datenbankstand von Produktion lokal einspielen zu können. Bisher gibt es keinen definierten Weg, dies sicher und reproduzierbar zu tun.

## What Changes

- Neues `make backup` Target: SSH-Verbindung zum VPS, SQLite-Backup erstellen, lokal herunterladen
- Neues `make restore-local` Target: heruntergeladenes Backup als lokale Entwicklungsdatenbank einspielen
- Kombiniertes `make pull-db` Target: backup + restore in einem Schritt

## Capabilities

### New Capabilities

- `db-backup-restore`: Make-Targets zum Sichern der Produktionsdatenbank und lokalen Einspielen für Entwicklungszwecke

### Modified Capabilities

## Impact

- `Makefile`: neue Targets
- Keine Code-Änderungen, keine Migrationen, keine API-Änderungen
- Setzt voraus: SSH-Alias `vServer` konfiguriert (bereits in CLAUDE.md dokumentiert), SQLite auf dem VPS verfügbar
