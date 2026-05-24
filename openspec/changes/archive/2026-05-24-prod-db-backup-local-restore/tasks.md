## 1. Makefile-Targets

- [x] 1.1 `backup`-Target hinzufügen: SSH-Befehl `sqlite3 $(DB_PATH) ".backup /tmp/teamwerk-backup.db"` auf VPS, dann `scp` nach `./teamwerk-backup.db`, dann Cleanup auf VPS
- [x] 1.2 `restore-local`-Target hinzufügen: Warnung ausgeben, Bestätigung per `read -p` abfragen, bei `y` `./teamwerk-backup.db` nach `./teamwerk.db` kopieren
- [x] 1.3 `pull-db`-Target hinzufügen: kombiniert `backup` und `restore-local`

## 2. Verifikation

- [ ] 2.1 `make backup` manuell gegen VPS testen — `./teamwerk-backup.db` erscheint lokal, keine Datei bleibt auf `/tmp` des VPS
- [ ] 2.2 `make restore-local` testen — bestätigen und abbrechen durchspielen, lokale DB korrekt ersetzt
- [ ] 2.3 `make pull-db` end-to-end testen — Go-Backend startet mit der restored DB ohne Fehler
