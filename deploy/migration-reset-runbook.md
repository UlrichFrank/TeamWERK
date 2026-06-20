# Runbook: Migration-Reset (kollabierte Historie)

**Was passiert:** Die bisherigen 49 Migrationen wurden in eine einzige
`001_initial.up.sql` zusammengefasst (nur Schema, ohne Seeds). Damit
`make deploy` auf der Prod-DB (`schema_migrations.version = 49`) **nicht**
versucht, die Initial-Migration ein zweites Mal anzuwenden, wird die
`schema_migrations`-Tabelle vor dem Deploy zurückgesetzt.

## Voraussetzungen

- Lokal: Branch mit dem Reset gemerged in `main`.
- Prod-Backup ist frisch: `make backup`.
- VPS-SSH funktioniert (`ssh vServer`).

## Ablauf

```bash
# 1) Backup (lokal, schon vorhanden falls eben gemacht)
make backup

# 2) Service stoppen, damit nichts in die DB schreibt
ssh vServer "sudo systemctl stop teamwerk"

# 3) Aktuelles Binary deployen (build + scp) — OHNE migrate auf alten Pfad
make build
rsync -az bin/teamwerk vServer:/tmp/teamwerk.new
ssh vServer "sudo mv /tmp/teamwerk.new /usr/local/bin/teamwerk"

# 4) schema_migrations auf "1, dirty=0" zwingen.
#    migrate force markiert die angegebene Version als sauber angewendet,
#    ohne up/down auszuführen. Genau das wollen wir hier: Prod-Schema ist
#    schon im Stand der Initial-Migration, nur der Versionseintrag muss
#    angepasst werden.
ssh vServer "/usr/local/bin/teamwerk migrate force 1 --db /var/lib/teamwerk/teamwerk.db"

# 5) Verifikation: Version muss jetzt 1 sein, Tabellen unverändert.
ssh vServer "sqlite3 /var/lib/teamwerk/teamwerk.db 'SELECT version, dirty FROM schema_migrations;'"
ssh vServer "sqlite3 /var/lib/teamwerk/teamwerk.db 'SELECT count(*) FROM stammvereine;'"
# Erwartung: 1|0  bzw.  22

# 6) Service starten
ssh vServer "sudo systemctl start teamwerk"
ssh vServer "sudo systemctl status teamwerk --no-pager | head -12"
```

`make deploy` für künftige Releases läuft danach wieder normal — `migrate up`
findet `schema_migrations.version = 1` und wendet alle ab `002_*` ausstehenden
Migrationen an.

## Rollback (Notfall)

Falls nach `migrate force` etwas schief geht und der Service nicht startet:

```bash
# Backup einspielen
ssh vServer "sudo systemctl stop teamwerk"
scp ./teamwerk-backup.db vServer:/tmp/restore.db
ssh vServer "sudo mv /var/lib/teamwerk/teamwerk.db /var/lib/teamwerk/teamwerk.db.broken \
             && sudo mv /tmp/restore.db /var/lib/teamwerk/teamwerk.db \
             && sudo chown www-data:www-data /var/lib/teamwerk/teamwerk.db"
# Vorheriges Binary (aus Backup) zurücklegen, dann Service starten.
ssh vServer "sudo systemctl start teamwerk"
```

Da das Backup vor dem Reset gezogen wurde, sind `schema_migrations.version = 49`
und alle Daten wieder im Ausgangszustand — das alte Binary mit 49 Migrationen
funktioniert damit weiter.

## Hinweise

- **Seeds sind weg:** Die `001_initial.up.sql` enthält keine `INSERT`-Anweisungen
  mehr. Prod ist davon nicht betroffen (Daten bleiben unverändert).
  Für eine **frische** Installation (lokal, neuer VPS) müssen die Stammvereine
  (22) und Beitragssätze (4) entweder im Admin-UI gepflegt oder als
  Seed-SQL nachgezogen werden. Tests seeden über
  `internal/testutil/db.go:seedBaseData`.
- **Künftige neue Migrationen:** Nächste freie Nummer ist `002_*`. golang-migrate
  nimmt auf `schema_migrations.version = 1` direkt mit `002` weiter.
- **Down-Migration:** `001_initial.down.sql` droppt alle Tabellen/Views.
  Sinnvoll nur lokal; auf Prod nicht ausführen, sonst Datenverlust.
