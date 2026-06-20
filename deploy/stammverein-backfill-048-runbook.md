# Runbook: Stammverein-Backfill (Migrationen 048 + 049)

**Ziel:** Den Backfill `members.home_club` (Freitext) → `members.home_club_id`
(FK auf `stammvereine`) als versionierte Migrationen festschreiben — mit
**vorher manuell geprüfter** Zuordnung der Freitextwerte (kein Fuzzy-Matching
im Blindflug).

**Abweichung vom Ursprungsdesign (bewusst):** Migration 047 hielt den Backfill
absichtlich aus den Migrationen heraus. Hier wird stattdessen eine reproduzierbare
Migration gewählt — Voraussetzung ist, dass die Zuordnung **vor** dem Schreiben
der Migration anhand der echten Prod-Werte geprüft wurde (Schritte 2–3). Damit
bleibt die Review-Eigenschaft erhalten, nur das Ergebnis wird versioniert statt
manuell ausgeführt.

**Aufteilung in zwei Migrationen:**
- **048** seedt die zusätzlich benötigten Stammvereine (Schema-/Seed-Änderung).
- **049** macht den Backfill `home_club_id` und löscht nicht aufklärbare
  Freitexte (reine Datenmigration).

Diese Trennung erlaubt isoliertes Rollback der Datenmigration und macht
Schema-Erweiterungen unabhängig vom Mapping reviewbar.

---

## Quelle der Wahrheit: `deploy/stammverein-mapping-049.yaml`

Die YAML-Datei dokumentiert für jeden Stammverein:
- den kanonischen `name` (UNIQUE in `stammvereine`),
- ob er in 048 erstmals geseedet wird (`neu: true`),
- die `freitexte`-Liste — alle `members.home_club`-Werte, die auf diesen
  Stammverein gemappt werden.

Bei jeder Änderung am Mapping müssen **YAML und Migrations-SQL synchron**
gepflegt werden (bewusst nicht generiert, damit die Migration klartext-reviewbar
bleibt).

## Bestand nach 048 + 049 (Stand 2026-06-20)

22 Stammvereine in `stammvereine` (8 aus 047 + 14 aus 048).

| Quelle | Anzahl |
|--------|-------:|
| 8 bestehende (Migration 047) | 113 Mitglieder zugeordnet |
| 14 neue (Migration 048) | 53 Mitglieder zugeordnet |
| **Summe zugeordnet** | **166** |
| Freitext `Flüchtling` (kein Verein) | 1 (bleibt NULL) |
| Freitext `TS` (gelöscht) | 2 (`home_club` → NULL) |

---

## Schritt 1 — Aktuelle Prod-DB sichern

```bash
make backup        # read-only sqlite .backup → ./teamwerk-backup.db (+ ./backup/uploads/)
```

## Schritt 2 — Ist-Werte auslesen

```bash
sqlite3 teamwerk-backup.db "
SELECT home_club, COUNT(*) AS anzahl
FROM members
WHERE TRIM(COALESCE(home_club,'')) <> ''
GROUP BY home_club ORDER BY anzahl DESC;"
```

Erwartung (Stand 2026-06-20): 31 verschiedene Freitexte, 169 Mitglieder. Bei
Abweichung von dieser Verteilung **vor** dem Deploy die Differenz prüfen und
ggf. `deploy/stammverein-mapping-049.yaml` ergänzen.

## Schritt 3 — Zuordnung prüfen / aktualisieren

`deploy/stammverein-mapping-049.yaml` öffnen und für jeden Freitextwert aus
Schritt 2 die Zuordnung prüfen. Neue Freitexte → eintragen (entweder unter
einem bestehenden Verein oder als neuer Stammverein mit `neu: true`).

Falls neue Stammvereine hinzukommen oder Freitext-Listen geändert werden,
**beide** Migrations-SQL-Dateien synchron pflegen (siehe Hinweis oben).

## Schritt 4 — Auf einer Kopie verifizieren

```bash
cp teamwerk-backup.db /tmp/tw-test.db
go run ./cmd/teamwerk migrate --db /tmp/tw-test.db
sqlite3 /tmp/tw-test.db "
SELECT COUNT(home_club_id) AS gesetzt,
       SUM(CASE WHEN TRIM(COALESCE(home_club,''))<>'' AND home_club_id IS NULL THEN 1 ELSE 0 END) AS noch_offen
FROM members;"
# Erwartung: gesetzt = 166, noch_offen = 1 (nur 'Flüchtling').
```

Optional Detail-Verifikation pro Stammverein:

```bash
sqlite3 /tmp/tw-test.db "
SELECT s.sort_order, s.name, COUNT(m.id) AS anzahl
FROM stammvereine s
LEFT JOIN members m ON m.home_club_id = s.id
GROUP BY s.id ORDER BY s.sort_order;"
```

## Schritt 5 — Build / Test / Lint

```bash
go test -race ./...
make build
# /verify-change   # Pre-Completion-Checkliste (Build/Test/Lint + Invarianten)
```

## Schritt 6 — Deploy

```bash
git add internal/db/migrations/048_*.sql \
        internal/db/migrations/049_*.sql \
        deploy/stammverein-mapping-049.yaml \
        deploy/stammverein-backfill-048-runbook.md \
        internal/stammvereine/handler_test.go
git commit -m "feat(db): Backfill home_club_id (Migrationen 048+049)"
make deploy        # führt migrate up (046–049) automatisch auf dem VPS aus
```

## Schritt 7 — Nachbereitung

Verbleibende Mitglieder mit `home_club_id IS NULL` und nicht-leerem `home_club`
(aktuell: 1 Mitglied mit `home_club = 'Flüchtling'`) im Frontend unter
**Mitglied → Stammdaten** über das Stammverein-`<select>` zuweisen oder bewusst
als „aktiv_ohne" belassen.
