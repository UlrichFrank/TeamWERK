# Runbook: Stammverein-Backfill als Migration 048

**Ziel:** Den Backfill `members.home_club` (Freitext) → `members.home_club_id`
(FK auf `stammvereine`) als versionierte Migration **048** festschreiben — mit
**vorher manuell geprüfter** Zuordnung der Freitextwerte (kein Fuzzy-Matching im
Blindflug).

**Abweichung vom Ursprungsdesign (bewusst):** Migration 047 hielt den Backfill
absichtlich aus den Migrationen heraus (reviewbar via `deploy/stammverein-mapping-*.sql`).
Hier wird stattdessen eine reproduzierbare Migration gewählt — Voraussetzung ist,
dass die Zuordnung **vor** dem Schreiben der Migration anhand der echten Prod-Werte
geprüft wurde (Schritte 2–3). Damit bleibt die Review-Eigenschaft erhalten, nur das
Ergebnis wird versioniert statt manuell ausgeführt.

---

## Ziel-Vereine (Seed aus Migration 047)

| ID | name |
|----|------|
| 1 | SKG Gablenberg 1884 |
| 2 | SKG Stuttgart Max-Eyth-See 1898 |
| 3 | SportKultur Stuttgart |
| 4 | Spvgg 1897 Cannstatt |
| 5 | TB Gaisburg 1886 |
| 6 | TB Untertürkheim 1888 |
| 7 | TSV Stuttgart-Münster 1875/99 |
| 8 | TV Cannstatt 1846 |

> Im Migrations-SQL **nie über die rohe ID** mappen, sondern per Name-Subquery
> (`SELECT id FROM stammvereine WHERE name='…'`) — robust gegen abweichende
> AUTOINCREMENT-Reihenfolge.

---

## Schritt 1 — Aktuelle Prod-DB sichern

```bash
make backup        # read-only sqlite .backup → ./teamwerk-backup.db (+ ./backup/uploads/)
```

## Schritt 2 — Ist-Werte auslesen (keine Migration nötig, Spalte existiert schon)

```bash
sqlite3 teamwerk-backup.db "
SELECT home_club, COUNT(*) AS anzahl
FROM members
WHERE TRIM(COALESCE(home_club,'')) <> ''
GROUP BY home_club ORDER BY anzahl DESC;"
```

Optional zum Abgleich, welche Werte der normalisierte Auto-Match treffen würde
(zeigt `exakt` vs. `UNMATCHED`) — dafür braucht die Kopie 047:

```bash
cp teamwerk-backup.db /tmp/tw-mapping.db
/usr/local/go/bin/go run ./cmd/teamwerk migrate --db /tmp/tw-mapping.db
sqlite3 /tmp/tw-mapping.db < deploy/stammverein-mapping-preview.sql
```

## Schritt 3 — Zuordnungstabelle festlegen

Für **jeden** Freitextwert aus Schritt 2 entscheiden: welcher der 8 Vereine — oder
bewusst **kein** Stammverein (`NULL` = `aktiv_ohne`). Ergebnis hier dokumentieren:

| home_club (Freitext) | Anzahl | → Verein (Name) oder NULL |
|----------------------|--------|---------------------------|
| _(aus Schritt 2 eintragen)_ | | |

## Schritt 4 — Migration 048 schreiben

Nächste freie Nummer prüfen (`ls internal/db/migrations/ | tail`); aktuell ist 047
das Höchste, also **048**. Zwei Dateien anlegen:

`internal/db/migrations/048_stammverein_backfill.up.sql`:

```sql
-- Backfill home_club -> home_club_id anhand der geprüften Zuordnung (Runbook
-- deploy/stammverein-backfill-048-runbook.md, Stand <DATUM>). Nur exakt geprüfte
-- Zuordnungen; UNMATCHED bleiben NULL und werden im Frontend zugewiesen.
-- Idempotent (nur wo home_club_id noch NULL) und über Name-Subquery (robust).

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name='SKG Gablenberg 1884')
  WHERE home_club_id IS NULL AND home_club IN ('<Freitext A>', '<Freitext B>');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name='TV Cannstatt 1846')
  WHERE home_club_id IS NULL AND home_club IN ('<Freitext C>');

-- … je Verein einen Block, mit den in Schritt 3 zugeordneten Freitextwerten.
```

`internal/db/migrations/048_stammverein_backfill.down.sql`:

```sql
-- Rücknahme: nur die von dieser Migration gesetzten Zuordnungen wieder lösen.
-- (Setzt voraus, dass vor 048 alle home_club_id NULL waren — gilt nach 047.)
UPDATE members SET home_club_id = NULL
  WHERE home_club IN (
    '<Freitext A>', '<Freitext B>', '<Freitext C>' /* … alle in 048 gemappten Werte */
  );
```

> Hinweis: Falls einzelne Mitglieder ihre Zuordnung schon manuell im Frontend
> gesetzt haben, schützt `home_club_id IS NULL` im up sie vor Überschreiben.

## Schritt 5 — Auf einer Kopie verifizieren

```bash
cp teamwerk-backup.db /tmp/tw-048.db
/usr/local/go/bin/go run ./cmd/teamwerk migrate --db /tmp/tw-048.db
sqlite3 /tmp/tw-048.db "
SELECT COUNT(home_club_id) AS gesetzt,
       SUM(CASE WHEN TRIM(COALESCE(home_club,''))<>'' AND home_club_id IS NULL THEN 1 ELSE 0 END) AS noch_offen
FROM members;"
# 'gesetzt' muss der Summe der in Schritt 3 zugeordneten Zeilen entsprechen.
```

## Schritt 6 — Build/Test/Verify

```bash
/usr/local/go/bin/go test ./...        # bzw. make test
/verify-change                          # Migrationsnummer, Build/Test/Lint, openspec validate
```

## Schritt 7 — Commit & Deploy

```bash
git add internal/db/migrations/048_stammverein_backfill.*.sql deploy/stammverein-backfill-048-runbook.md
git commit -m "feat(db): Migration 048 — Backfill home_club_id aus geprüfter Zuordnung"
make deploy                             # führt migrate up (048) automatisch auf dem VPS aus
```

Nach dem Deploy verbleibende `noch_offen`-Mitglieder im Frontend unter
**Mitglied → Stammdaten** über das Stammverein-`<select>` zuweisen.
