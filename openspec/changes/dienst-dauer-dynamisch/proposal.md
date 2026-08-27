## Why

Seit `dienst-dauer` trägt jeder Dienst eine Dauer (`hours_value`, Stunden) und die
Dienstbörse zeigt eine Zeitspanne. Diese Dauer ist eine **feste Zahl** — sie weiß nichts
vom Spiel, an dem der Dienst hängt.

Für einen Teil der Dienste ist das die falsche Aussage. Ein Zeitnehmer ist so lange
gebunden, wie das Spiel dauert; ein Abbau-Dienst endet, wenn die Halle leer ist. Die
Spieldauer schwankt aber je Altersklasse (`age_class_game_rules.half_duration_minutes` +
`break_minutes`) und je Vorlage (`game_templates.duration_minutes` bei generischen
Terminen). Wer heute „Zeitnehmer = 1 h" pflegt, hat ihn bei der E-Jugend zu lang und bei
den Herren zu kurz — und muss für jede Altersklasse eine eigene Vorlage bauen, nur damit
eine Zahl stimmt.

Der Startzeitpunkt kann das längst: `default_anchor` + `default_offset_minutes` positionieren
ihn relativ zum Anpfiff oder zum Spielende. Nur das Ende hat diese Möglichkeit nicht.

## What Changes

- Ein Diensttyp bekommt einen **Dauer-Modus**: `absolut` (heutiges Verhalten, `hours_value`)
  oder `dynamisch` (das Ende wird wie der Start über Anker + Versatz bestimmt).
- Der End-Anker nutzt **dieselben zwei Anker wie der Start** (`start` = Anpfiff,
  `end` = Spielende) mit Versatz in beide Richtungen. Damit sind sowohl „bis Spielende
  +15 min" als auch „bis Anpfiff +40 min" (Halbzeit-Dienste) ausdrückbar.
- Die Auflösung des End-Ankers ist **exakt die bestehende Start-Auflösung**
  (`regen.go`, heute Z. 883): `end` nimmt `games.end_time`, falls gepflegt, sonst
  Anpfiff + Spieldauer. Kein zweiter Rechenweg.
- **`hours_value` bleibt im dynamischen Modus gepflegt und dient als Rückfall.** Ergibt die
  dynamische Auflösung keine positive Dauer (Endzeit läge vor der Startzeit), entsteht der
  Slot trotzdem — mit der absoluten Dauer. Der Dienst fällt nie aus, weil jemand einen
  Versatz falsch gesetzt hat.
- Drei neue Spalten auf **zwei** Ebenen: `duty_types` und `game_template_items`
  (`duration_mode`, `end_anchor`, `end_offset_minutes`), Copy-on-pick wie die übrigen
  Item-Felder.
- **`duty_slots` bekommt keine neue Spalte.** Der Regen löst die dynamische Dauer in das
  vorhandene `hours_value` auf — der Slot ist wie bisher ein eingefrorener Schnappschuss
  und muss nicht wissen, woher seine Dauer kam.
- Manuell angelegte Slots bleiben **absolut**: sie tragen `is_custom=1`, der Regen fasst sie
  nie an, „folgt dem Spiel" könnte dort nichts bewirken. Das Modal „+ Dienst hinzufügen"
  darf die Dauer aus einer dynamischen Typ-Definition aber **ausrechnen** und vorbelegen.

- **Die Vorlagen-Detailseite entfällt.** `AdminDutyTemplateDetailPage` und ihre Route
  `/admin/dienstplan-vorlagen/{id}` sind gelöscht, der Item-Editor sitzt im Modal der
  Listenseite (`AdminDutyTemplatesPage`). Das war keine geplante Aufräumarbeit, sondern
  fiel bei Aufgabe 6 an: Detailseite und Listen-Modal pflegten dieselben Item-Felder in
  zwei Masken, die Modus-Maske hätte es doppelt gebraucht. Die Tests der Detailseite
  (Dauer, Rotations-Schalter, Team-Scope) sind auf den Modal-Editor portiert.
- **Die Dauer eines Diensttyps wird validiert.** `POST`/`PUT /api/duty-types` weisen eine
  explizit gesendete `hours_value` ≤ 0 mit HTTP 400 ab; ein fehlendes Feld ergibt den
  Spalten-Default 1.0 statt der stillen Null des leeren Requests. Ohne das ist die
  garantierte Invariante („ein Slot trägt nach jedem Regen-Lauf eine Dauer > 0") über den
  Diensttyp umgehbar — Slot- und Vorlagen-Route prüfen nur ihre eigene Eingabe, eine per
  Copy-on-pick geerbte Dauer sendet niemand explizit.

## Capabilities

### Modified Capabilities

- `duties`: Die Dauer eines Dienstes kann relativ zum Spiel definiert werden statt als
  feste Zahl; der Regen löst sie je Termin auf. Die Dauer eines Diensttyps muss positiv
  sein.
- `sse-live-updates`: Die Zeile `AdminDutyTemplateDetailPage` in der Seiten-Tabelle
  entfällt (Seite gelöscht).

## Test-Anforderungen

| Route / Pfad | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| — (Regen) | `TestRegen_DynamischeDauerFolgtSpieldauer` | Zwei Spiele verschiedener Altersklassen mit derselben Vorlage erzeugen Slots **unterschiedlicher** Dauer. |
| — (Regen) | `TestRegen_DynamischeDauerNutztEndTime` | Ist `games.end_time` gepflegt, rechnet der End-Anker `end` dagegen — nicht gegen Anpfiff + Spieldauer. |
| — (Regen) | `TestRegen_DynamischeDauerAnkerStart` | End-Anker `start` (Halbzeit-Dienst) ergibt die Differenz zweier Anpfiff-Versätze, unabhängig von der Spieldauer. |
| — (Regen) | `TestRegen_DynamischeDauerFaelltAufAbsoluteZurueck` | Löst das Ende vor den Start auf, trägt der Slot `hours_value` — und der Slot entsteht. |
| — (Regen) | `TestRegen_AbsoluterModusUnveraendert` | `duration_mode='absolut'` verhält sich exakt wie vor diesem Change. |
| `PUT /api/duty-types/{id}` | `TestUpdateType_SpeichertDauerModus` | Modus, End-Anker und End-Versatz werden gespeichert und zurückgeliefert. |
| `PUT /api/duty-types/{id}` | `TestUpdateType_UngueltigerDauerModus_400` | `duration_mode` außerhalb `absolut\|dynamisch` → 400, nichts persistiert. |
| `PUT /api/duty-templates/{id}` | `TestUpdateTemplate_SpeichertDauerModusJeItem` | Die Vorlagen-Zeile trägt ihren eigenen Modus, unabhängig vom aktuellen Typ-Wert. |
| `POST /api/duty-types` | `TestCreateType_DauerNullWirdAbgewiesen` | `hours_value: 0` → 400, kein Diensttyp angelegt. |
| `POST /api/duty-types` | `TestCreateType_OhneDauerNutztDBDefault` | Fehlendes Feld ergibt 1,0 — nicht die Null des leeren Requests. |
| `PUT /api/duty-types/{id}` | `TestUpdateType_DauerNullWirdAbgewiesen` | `hours_value: 0` → 400, Bestand (inkl. Name) unverändert. |

**Garantierte Invariante:** Ein Slot trägt nach jedem Regen-Lauf eine Dauer > 0 — unabhängig
davon, wie Anker und Versätze gesetzt sind.

## Impact

- `internal/db/migrations/053_dienst_dauer_dynamisch.{up,down}.sql` — je drei Spalten auf
  `duty_types` und `game_template_items`, mit `CHECK`-Constraints (SQLite erlaubt das auf
  hinzugefügten Spalten, nachgeprüft).
- `internal/games/regen.go` — Anker-Auflösung in einen Helfer ziehen und für den End-Anker
  wiederverwenden; `resultHours` aus der Differenz statt aus dem Item.
- `internal/games/handler.go` — `templateItemRow` + Template-CRUD um die drei Felder.
- `internal/duties/handler.go` — `ListTypes`/`CreateType`/`UpdateType` um die drei Felder,
  Validierung von `duration_mode` und `end_anchor`.
- `web/src/pages/AdminDutyTypesPage.tsx` — Modus-Umschalter.
- `web/src/pages/AdminDutyTemplatesPage.tsx` — dieselbe Maske je Vorlagen-Zeile.
- `web/src/lib/dutyTemplateItems.ts` — `refreshItemsFromDutyTypes` muss die drei neuen
  Felder mitziehen, sonst frischt „Aus Diensttypen auffrischen" unvollständig auf.
- `web/src/components/SpieltagDetailModal.tsx` — Vorbelegung der Dauer aus einer dynamischen
  Typ-Definition.
- **Unberührt:** `duty_slots` (kein Schema-Change), `restoreAssignments`/`makeCustomKey`
  (die Dauer steht in keinem Match-Schlüssel), `duty_accounts.ist` (liest weiter
  `ds.hours_value`).
