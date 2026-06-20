## Why

Passive Mitglieder werden im Beitragslauf **gar nicht erkannt** — sie fallen mit der Begründung `kein_beitragssatz` komplett aus dem Lauf.

Die Ursache ist **kein** Fehler im Status-Mapping (`internal/beitragslauf/compute.go:16` bildet `pausiert`/`passiv` korrekt auf die Gruppe `passiv` ab), sondern ein **Datums-Bug in der Beitragsmatrix**:

```
Seed (043_sepa_beitragslauf.up.sql:26):  ('passiv', 6000, '2027-01-01')
Stichtag im Lauf ist IMMER der 01.07. der Saison.
```

`LookupBetragCent` (`internal/beitragslauf/query.go:99`) sucht den neuesten Satz mit `valid_from <= Stichtag`. Für jede Saison, die am **01.07.2026** beginnt, ist `2027-01-01 > 2026-07-01` → es wird **kein gültiger Passiv-Satz gefunden** → das Mitglied wird ausgeschlossen. Erst ab Saison 2027/28 (Stichtag 01.07.2027) greift der Satz.

Die `aktiv_ohne`- und `aktiv_mit`-Sätze derselben Beitragsordnung (beschlossen 22.04.2026) gelten bereits ab `2026-07-01`. Der abweichende Passiv-Stichtag `2027-01-01` ist nicht beabsichtigt — alle drei Kategorien stammen aus derselben Anlage 1 und sollen zum selben Saisonstart greifen.

Dieser Fix ist bewusst **klein und unabhängig** vom geplanten Stammverein-Feature und kann sofort deployt werden.

## What Changes

**DB-Migration (neu, 046):**
- Neuer `beitrags_saetze`-Eintrag `('passiv', 6000, '2026-07-01')`, sodass passive Mitglieder ab Saisonstart 2026/27 erkannt werden.
- Der bestehende Satz `('passiv', 6000, '2027-01-01')` bleibt unangetastet (Historie); da Betrag identisch, ändert sich am späteren Stichtag nichts.
- `INSERT OR IGNORE`, damit die Migration idempotent ist.

**Keine Code-Änderung** — die Berechnungslogik ist korrekt; nur die Datenbasis wird ergänzt.

## Impact

- Betroffene Specs: `sepa-beitragslauf` (Beitragsberechnung)
- Betroffener Code: `internal/db/migrations/046_passiv_beitragssatz_saisonstart.{up,down}.sql`
- Betroffene Tests: `internal/beitragslauf/handler_test.go` — neuer Fall „passives Mitglied in Saison 2026/27 wird mit 60 € einbezogen"
- Kein Frontend-, Router- oder API-Change → kein neuer `Broadcast`/`useLiveUpdates`-Bedarf.

Beiträge gelten grundsätzlich pro Saison (01.07.–30.06.); die erste relevante Saison beginnt am 01.07.2026, daher ist `valid_from = 2026-07-01` der korrekte Geltungsbeginn für den Passiv-Satz.
