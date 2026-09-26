## Context

Warum die Änderung nötig ist: siehe proposal.md. Hier nur, was den Ansatz bestimmt.

`GET /api/duty-board` (`duties.Board`) setzt drei Queries ab. Gemessen auf einer lokalen
Kopie mit Migrationsstand 070 kosten sie:

```
1. Slot-Query (Gruppen + Slots)           5-12 ms
2. Assignee-Query (Namen + aushilfe)   1.500-2.900 ms   (mit Vergangenheit bis 5.100 ms)
3. game_teams je Gruppe nachladen            ~1 ms
```

Die Assignee-Query wertet je Zusage das Aushilfe-Prädikat aus:

```
CASE WHEN slotInTeamsSQL(UserTeamsSQL(Extended, "da.user_id"))
     AND NOT slotInTeamsSQL(UserTeamsSQL(Stamm, "da.user_id"))
```

Mit `slotInTeamsSQL(t)` heute:

```
((ds.game_id IS NULL AND ds.team_id IN (t))
 OR ds.game_id IN (SELECT gt_s.game_id FROM game_teams gt_s WHERE gt_s.team_id IN (t)))
```

Weil `t` über `da.user_id` korreliert ist, baut SQLite die Team-Menge je Zeile neu auf.
`game_teams` hat nur den PK `(game_id, team_id)` und keinen Index auf `team_id`. Der Planer
geht deshalb je Zusage und je Prädikat (2× erweitert, 2× Stamm) **alle** `game_teams`-Zeilen
durch (`SCAN gt_s`) und prüft für jede die korrelierte Team-Menge. Der Aufwand ist grob
Zusagen × `game_teams` × 4, und beide Faktoren wachsen über die Saison.

Dieselbe Hilfsfunktion steckt in der Slot-Query (Team-Quelle für Nicht-Vorstand,
Eltern-Zielgruppe, Aushilfe-Kennzeichen der Gruppe). Dort ist `t` mit `?` parametrisiert,
also unkorreliert, und deshalb billig. Eine Kopie derselben Form liegt in
`dashboard.slotInTeams`; der Block „Aushilfe" (`queryMeineDiensteAushilfe`) nutzt sie mit
`da.user_id`, beschränkt auf die eigenen Zusagen und `LIMIT 5`.

## Goals / Non-Goals

**Goals:**
- Die Assignee-Query der Dienstbörse ist unabhängig von der Anzahl der `game_teams`-Zeilen und
  von Planer-Statistik im niedrigen zweistelligen Millisekundenbereich.
- Board und Dashboard werten das Team-Prädikat weiterhin **wortgleich** aus
  (`TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` bleibt die Klammer).
- Ein Test macht ein Zurückfallen auf den Scan sichtbar.

**Non-Goals:**
- Keine weiteren Indizes. Der Sweep über alle GET-Routen fand keinen zweiten Hotspot.
  Unindizierte Fremdschlüssel kosten bei den heutigen Mengen (max. ~1.800 Zeilen) messbar
  nichts.
- Keine Umstellung der Aushilfe-Berechnung nach Go (siehe Entscheidung 2).
- Kein Frontend-Umbau (Virtualisierung der Liste o. Ä.). Das Rendering wurde nicht gemessen;
  es wird nach dem Fix einmal im Browser geprüft (Task 3.2).
- `/api/games/my` ohne Zeitfenster (~235 ms): das Frontend sendet immer `from`/`to` und
  braucht dann ~20 ms.

## Decisions

### 1. Korrelation über `ds.game_id` statt Mengenvergleich über `team_id`

`slotInTeamsSQL` und `dashboard.slotInTeams` werden zu:

```
((ds.game_id IS NULL AND ds.team_id IN (t))
 OR EXISTS (SELECT 1 FROM game_teams gt_s
            WHERE gt_s.game_id = ds.game_id AND gt_s.team_id IN (t)))
```

Je Slot ist das ein PK-Lookup auf die 1–3 Teams seines Termins. Die logische Aussage bleibt
gleich: Es gibt ein Team des Termins, das in `t` liegt.

Gemessen (Assignee-Query, 178 Zusagen, sqlite3-CLI):

| Variante | ohne Index | mit `game_teams(team_id)` | mit Index + `ANALYZE` |
|---|---|---|---|
| heute (`IN`) | 1.486 ms | 13 ms | 1.346 ms |
| `EXISTS` | 9 ms | 9 ms | 8 ms |

Beide Varianten liefern identische Zeilen und Aushilfe-Summen.

**Verworfen: Index `game_teams(team_id)`.** Ohne Statistik wirkt er (HTTP end-to-end:
Admin 3,8 s → 32 ms). Nach `ANALYZE` hält der Planer den Scan aber wieder für günstiger,
weil `game_teams` klein ist, und die Zeit springt zurück auf über eine Sekunde. Heute läuft
nirgends `ANALYZE`/`PRAGMA optimize`. Der Fix hinge also an einem Zustand, den ein
SQLite-Upgrade (modernc) oder eine spätere Wartungsmaßnahme still kippt. Die `EXISTS`-Form
trägt ihren Zugriffspfad selbst.

**Verworfen: beides.** Der Index würde nur anderen `game_teams … team_id IN`-Stellen helfen,
die laut Sweep nicht langsam sind. Er wäre eine Migration ohne Befund.

### 2. SQL-Form beibehalten, keine Go-seitige Aushilfe-Berechnung

Alternative: je distinktem Eingetragenen die Stamm-/Erweitert-Mengen einmal laden und in Go
gegen die (ohnehin nachgeladenen) Termin-Teams der Gruppe prüfen. Das wäre ebenfalls
planerunabhängig, würde aber eine zweite Formulierung der Aushilfe-Regel neben
`appdb.UserTeamsSQL` schaffen. Genau diese Doppelung soll
`TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` verhindern. Die Umformulierung ist eine
Zeile je Hilfsfunktion und lässt die Regel an einer Stelle.

### 3. Beide Hilfsfunktionen gleichzeitig ändern

`duties.slotInTeamsSQL` und `dashboard.slotInTeams` sind zwei Kopien derselben Form. Sie
liegen in zwei Domänen-Packages, die sich nicht importieren dürfen (Arch-Test). Eine
Zusammenlegung nach `internal/db` neben `UserTeamsSQL` wäre möglich, weil sie reiner
SQL-Baustein und damit Foundation ist. Sie ist aber nicht nötig und bleibt ausdrücklich
optional (Task 1.3). Pflicht ist nur, dass beide dieselbe Form tragen.

### 4. Regressionstest über `EXPLAIN QUERY PLAN`

Damit der Test die echte Query erklären kann, wird der SQL-Text der Assignee-Query aus
`Board` in eine unexportierte Funktion herausgezogen (`boardAssigneesSQL(n int) string`,
`n` = Anzahl Platzhalter). Der Test
- legt über `testutil` eine kleine Saison mit Spiel, `game_teams`, Slot und Zusage an,
- führt `EXPLAIN QUERY PLAN` mit Dummy-Argumenten aus und verlangt, dass keine Zeile
  `SCAN gt_s` bzw. `SCAN game_teams` enthält,
- führt anschließend `ANALYZE` aus und prüft erneut (Szenario „Plan bleibt nach
  Statistik-Erhebung gleich").

Geprüft wird bewusst nur das Fehlen des Scans, nicht der vollständige Plan. Andere
Planänderungen (Reihenfolge, Covering-Index) sollen den Test nicht rot färben. Ein
Laufzeit-Schwellwert im Test wurde verworfen: Er wäre auf CI-Runnern flaky und sähe den
Fehler auf kleinen Testdaten ohnehin nicht.

## Risks / Trade-offs

- **[Slot-Query wird korreliert]** Mit `?`-Parametern war das Prädikat bisher unkorreliert;
  die `EXISTS`-Form ist je Slot korreliert. Bei 566 Slots sind das 566 PK-Lookups, die
  innere unkorrelierte Team-Menge baut SQLite einmal auf. → Task 3.1 misst die Slot-Query
  für Nicht-Vorstand-Personas vor und nach dem Umbau. Wird sie messbar langsamer (> 20 ms),
  bleibt für die parametrisierten Aufrufe die alte Form, und nur die korrelierten Aufrufe
  (Assignee-Query, Dashboard-Aushilfe) bekommen `EXISTS`.
- **[Plan-Test hängt am Planer]** Ein SQLite-Upgrade könnte auch für die `EXISTS`-Form einen
  Scan wählen. → Dann ist der Test genau das gewünschte Signal, kein Fehlalarm: die
  Antwortzeit würde tatsächlich wieder wachsen.
- **[Semantik-Drift]** `IN (SELECT …)` und `EXISTS (…)` unterscheiden sich bei NULL-Werten.
  `gt_s.game_id`/`team_id` sind `NOT NULL`, und `ds.game_id IS NULL` fängt der erste Zweig
  ab. → Die bestehenden Aushilfe-/Sichtbarkeitstests plus der Personas-Vergleich (Task 2.3)
  sichern das ab.

## Migration Plan

Keine Migration. Normales `make deploy`; Rollback über `make deploy-rollback` ist gefahrlos,
weil sich an Schema und Daten nichts ändert.
