## Why

`/dienste` braucht mehrere Sekunden bis zur ersten Anzeige. Gemessen gegen eine lokale Kopie
mit realistischem Bestand (566 kommende Slots, 258 Zusagen, 248 `game_teams`-Zeilen):
`GET /api/duty-board` antwortet für Admin/Vorstand/Trainer in **3,8–3,9 s**, für Eltern in
**2,6 s**, mit „Vergangene anzeigen" bis **5,1 s** allein in einer Query. Die Zeit steckt fast
vollständig in **einer** Unterabfrage: dem Aushilfe-Kennzeichen je Eingetragenem (seit
`dienste-erweiterter-kader`). Der Wert wächst mit jeder Zusage über die Saison und liegt auf
dem VPS (Linux XS) eher höher. Ein Sweep über alle übrigen GET-Routen (fünf Personas,
~110 Routen) fand keinen zweiten Ausreißer — alles andere bleibt unter ~110 ms, der
überwiegende Teil unter 30 ms.

## What Changes

- **Team-Prädikat der Dienst-Slots umformulieren** (`slotInTeamsSQL` in `internal/duties`,
  `slotInTeams` in `internal/dashboard`): statt
  `ds.game_id IN (SELECT game_id FROM game_teams WHERE team_id IN (…))` die korrelierte Form
  `EXISTS (SELECT 1 FROM game_teams WHERE game_id = ds.game_id AND team_id IN (…))`. Sie
  nutzt den vorhandenen Primärschlüssel `(game_id, team_id)` und ist unabhängig von
  Planer-Statistik und Indizes schnell (gemessen 9 ms statt 1,3–2,9 s, identisches
  Ergebnis).
- **Regressionsschutz:** ein Test prüft per `EXPLAIN QUERY PLAN`, dass die Assignee-Query der
  Dienstbörse `game_teams` nicht vollständig scannt, und ein Laufzeit-unabhängiger
  Ergebnisvergleich hält die Semantik des Prädikats fest.
- **Keine Migration, kein neuer Index.** Ein Index `game_teams(team_id)` hilft nur, solange
  SQLite keine Statistik (`sqlite_stat1`) hat — nach `ANALYZE` wählt der Planer wieder den
  Scan (gemessen 1,3 s). Er ist deshalb bewusst **nicht** Teil der Lösung.
- Keine Änderung an API-Form, Sichtbarkeit, Aushilfe-Regel oder Frontend.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `duties`: neue Anforderung an die Antwortzeit der Dienstbörse — das Aushilfe- und
  Team-Prädikat darf nicht mit (Zusagen × Termin-Teams) skalieren und darf nicht von
  Planer-Statistik abhängen.

## Test-Anforderungen

Keine neue Route. Geänderte Route `GET /api/duty-board` (Antwortform unverändert):

| Test | Erwartung | Invariante |
|---|---|---|
| `TestBoardAssigneesPlan_KeinScanAufGameTeams` | Plan ohne `SCAN` auf `game_teams`, auch nach `ANALYZE` | Aufwand je Zusage unabhängig von der Anzahl der Termin-Team-Zuordnungen |
| `TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` (bestehend) | grün | Board und Dienst-Bilanz werten die Aushilfe-Regel gleich aus |
| `board_aushilfe_test.go`, Dashboard-Aushilfe-Tests (bestehend) | grün | Sichtbarkeit und `aushilfe`-Kennzeichen unverändert |
| Board-Tests bei fehlender Berechtigung (bestehend, 401) | grün | Auth-Tier unverändert |

## Impact

- **Code:** `internal/duties/handler.go` (`slotInTeamsSQL`, Assignee-Query in `Board` ggf. als
  benannte Funktion herausgezogen, damit der Test sie erklären kann),
  `internal/dashboard/handler.go` (`slotInTeams`).
- **Tests:** neuer Plan-Test in `internal/duties`; bestehende Aushilfe-Tests
  (`board_aushilfe_test.go`, insbesondere
  `TestAushilfePraedikat_BoardUndBilanzDeckungsgleich`) und Dashboard-Tests müssen
  unverändert grün bleiben — sie sind der Nachweis, dass die Semantik gleich bleibt.
- **Berechtigungsmodell:** unberührt — dieselben Team-Mengen (`appdb.UserTeamsSQL`), dieselbe
  Audience-Logik, nur die SQL-Form des Mengenvergleichs ändert sich.
- **Betrieb:** keine Migration, kein Deploy-Sonderschritt, kein RAM-Mehrbedarf.
