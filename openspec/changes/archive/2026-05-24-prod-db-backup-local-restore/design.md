## Context

Die Produktionsdatenbank liegt auf dem VPS unter `/var/lib/teamwerk/teamwerk.db` (WAL-Mode, SQLite).
Lokal liegt die Entwicklungsdatenbank unter `./teamwerk.db` (aus `.env`: `DB_PATH=./teamwerk.db`).
SSH-Verbindung zum VPS läuft über den Alias `$(REMOTE)` (aus `.env`).

Das `sqlite3`-CLI-Tool ist auf dem VPS installiert (wird für Migrationen genutzt).
Der systemd-Service läuft während des Backups weiter — ein Online-Backup ist nötig.

## Goals / Non-Goals

**Goals:**
- Sicheres Online-Backup der laufenden SQLite-Datenbank auf dem VPS (ohne Service-Stop)
- Lokales Einspielen des Backups als Entwicklungsdatenbank
- Alles über `make`-Targets — kein zusätzliches Skript, keine neue Dependency

**Non-Goals:**
- Automatisches/geplantes Backup (kein Cron-Job)
- Verschlüsselung des Backups
- Backup-Rotation oder Archivierung

## Decisions

### SQLite Online-Backup via `.backup`-Befehl

**Entscheidung:** `sqlite3 /var/lib/teamwerk/teamwerk.db ".backup /tmp/teamwerk-backup.db"` auf dem VPS ausführen.

**Warum:** SQLite's `.backup`-Befehl ist der offizielle Weg für Online-Backups bei laufendem WAL-Mode. Er ist atomar und konsistent — kein Risiko von halbfertigen Writes. Alternativen wären `cp` (unsicher bei WAL, kann inkonsistenten Zustand kopieren) oder `VACUUM INTO` (SQL-Befehl, braucht Go-Code).

### Download via `scp`

**Entscheidung:** `scp $(REMOTE):/tmp/teamwerk-backup.db ./teamwerk-backup.db`

**Warum:** `scp` ist immer verfügbar wenn SSH funktioniert. `rsync` wäre für eine einzige kleine Datei Overkill.

### Backup-Datei lokal als `teamwerk-backup.db` ablegen

**Entscheidung:** Backup landet als `./teamwerk-backup.db`, separate von der aktiven `./teamwerk.db`.

**Warum:** Versehentliches Überschreiben der laufenden lokalen DB vermeiden. `make restore-local` macht dann explizit `cp teamwerk-backup.db teamwerk.db`.

### Zusammengeführtes Target `make pull-db`

**Entscheidung:** Ein kombiniertes Target `pull-db` führt backup → download → restore in einem Schritt aus.

**Warum:** Das ist der häufigste Use-Case. Die einzelnen Targets bleiben für manuelle Nutzung erhalten.

## Risks / Trade-offs

- [Backup-Datei auf `/tmp`] → Mitigation: Temporäre Datei auf VPS nach Download löschen (`ssh $(REMOTE) rm -f /tmp/teamwerk-backup.db`)
- [Bestehende lokale DB wird überschrieben] → Mitigation: `make restore-local` gibt eine Warnung aus und fragt nach Bestätigung (`read -p`)
- [`sqlite3` muss auf VPS verfügbar sein] → Mitigation: VPS-Setup installiert es bereits; Fehlermeldung ist selbsterklärend falls nicht vorhanden
