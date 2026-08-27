## Context

Siehe proposal.md — Why. Für den Entwurf zählen vier Eigenschaften des Ist-Zustands, die
alle vier Entwurfsentscheidungen tragen:

1. **`hours_value` heißt schon „Dauer".** `AdminDutyTypesPage.tsx:96` beschriftet das Feld
   so und nimmt `1h 30min` entgegen (`parseHoursInput`, Z. 49). Fachlich ist es die
   Gutschrift, die in `duty_accounts.ist` läuft. Die Doppelbedeutung existiert also bereits
   in der UI — dieser Change macht sie zur offiziellen Semantik, statt sie aufzulösen.
2. **`duty_slots` ist durchweg ein eingefrorener Schnappschuss.** `event_time`,
   `slots_total`, `audiences` werden beim Regen aus der Vorlage aufgelöst und in die Zeile
   geschrieben (`regen.go:938`). Kein Slot fragt zur Laufzeit die Vorlage. Eine Dauer, die
   sich anders verhielte, wäre der Fremdkörper.
3. **Die Vorlagen-Zeile erbt nicht, sie kopiert.** `game_template_items.anchor` und
   `.offset_minutes` sind `NOT NULL DEFAULT` (Migration `001`, Z. 187–188); der Editor füllt
   sie beim Auswählen des Diensttyps aus dessen Defaults
   (`AdminDutyTemplateDetailPage.tsx:247`). Es gibt im Codebase **kein** NULL-erbt-Muster.
4. **Die Dauer steht in keinem Regen-Schlüssel.** `makeCustomKey` und `restoreAssignments`
   matchen über `(duty_type_id, event_time, team_id)`. Eine Dauer verschiebt `event_time`
   nicht.

## Goals / Non-Goals

**Goals**

- Start- und Endzeit auf `/dienste` und im Spieltag-Detail-Modal.
- Dauer überschreibbar in der Vorlage und am einzelnen Slot.
- Genau **eine** Zahl je Slot, aus der sowohl die Anzeige als auch die Gutschrift folgt.

**Non-Goals**

- Kein zweites Feld zur Trennung von Anzeige und Anrechnung (bewusst verworfen, s. u.).
- Keine Reparatur der fehlenden `ist`-Buchung in `Fulfill` (eigener Change).
- Keine Endzeit in der Spieltag-Kopfzeile (`DutyPage.tsx:382` zeigt die Anstoßzeit des
  **Spiels**, nicht eines Slots).
- Kein Kalender-/ICS-Export der Dienstzeiten.

## Decision 1 — `hours_value` ist die Dauer, kein zweites Feld

**Entscheidung:** Anzeige und Anrechnung teilen sich eine Zahl. „8:00–9:00" *bedeutet*
„1,0 Stunden aufs Konto".

**Alternative (verworfen):** neue Spalte `duration_minutes` für die Anzeige, `hours_value`
bleibt die Gutschrift und wird auf `/diensttypen` in „Anrechnung" umbenannt.

**Begründung:** Die Trennung kauft Genauigkeit für zwei Fälle und kostet sie in allen
anderen. Der Vorstand müsste zwei Zahlen pflegen, die in der Praxis fast immer identisch
sind, und ein bekanntes Feld würde umbenannt. Der Preis der Kopplung — eine Zeitkorrektur
ändert die Gutschrift — ist die *gewollte* fachliche Aussage, nicht ein Nebeneffekt.

**Der Preis, ausgeschrieben:** Fristgebundene Dienste bekommen eine sinnlose Spanne. Der
Spielbericht-Dienst aus Migration `020` (`hours_value=0.5`, Anker Ende+24h) zeigt künftig
`20:00–20:30`, obwohl er 24 Stunden Zeit hat und keine Schicht ist. Ein Flag „als
Zeitspanne anzeigen" je Diensttyp wurde erwogen und verworfen: es hätte für einen
Einzelfall ein zweites Konzept in die Diensttyp-Maske gebracht. Sollten fristgebundene
Dienste zunehmen, ist das Flag der naheliegende Folge-Change — es ist additiv und bricht
nichts.

## Decision 2 — Copy-on-pick statt NULL-erben

**Entscheidung:** `game_template_items.hours_value` ist `NOT NULL`. Der Vorlagen-Editor
kopiert den Typ-Wert beim Auswählen des Diensttyps hinein; danach ist die Zeile
eigenständig.

**Alternative (verworfen):** `NULL` = „nimm den aktuellen Typ-Wert", aufgelöst bei jedem
Regen.

**Begründung:** Punkt 3 im Context. `anchor` und `offset_minutes` stehen in derselben
Zeilenmaske direkt daneben und verhalten sich als Copy-on-pick. Eine Dauer, die erbt, hätte
zwei Mechanismen in einer Maske nebeneinandergestellt, ohne dass die UI den Unterschied
zeigen kann.

**Der Preis, ausgeschrieben:** Ändert der Vorstand die Dauer eines Diensttyps, wirkt das
**nicht** auf bestehende Vorlagen. Er muss sie in jeder Vorlage nachziehen. Das ist heute
für Anker und Versatz schon so und war bisher kein gemeldetes Problem — aber die Dauer wird
sichtbarer sein als der Versatz, und damit fällt es eher auf.

## Decision 3 — Bei `reduced` gewinnt die Variante

**Entscheidung:** Liefert `applyBehavior` eine andere `duty_type_id` als die der
Vorlagen-Zeile (`same_day_behavior`/`adjacent_day_behavior` = `reduced`), kommt die Dauer
des Slots aus dem **Varianten-Typ**. Die kopierte Dauer der Vorlagen-Zeile wird verworfen.

**Begründung:** Eine Variante ist kein Rabatt, sondern ein **anderer Dienst**. An einem
Mehrfachspieltag braucht das Folgespiel eine andere Arbeit als das erste, und `reduced`
wählt die richtige aus. Damit ergibt sich die Zuständigkeit ohne Sonderregel:

| Eigenschaft | Quelle | weil |
|---|---|---|
| `anchor`, `offset_minutes` | Vorlagen-Zeile | **wann** im Tagesablauf |
| `slots_count` | Vorlagen-Zeile | **wie viele** Personen |
| `hours_value` | Varianten-Typ bei `reduced`, sonst Vorlagen-Zeile | **welche Arbeit** |

Die Vorlage bestimmt Position und Umfang, der Diensttyp bestimmt die Art der Arbeit — und
die Dauer ist eine Eigenschaft der Arbeit. Dass die Variantenstunden meist niedriger sind,
ist Folge, nicht Zweck.

**Implementierung:** `regenGameItems` holt bei `isReduce` heute schon den Namen der Variante
(`lookupDutyTypeNameTx`, Z. 906). Der Helfer liefert künftig Name **und** `hours_value` in
einem Query — kein zusätzlicher Roundtrip.

**Der Preis, ausgeschrieben:** Wer die Dauer in der Vorlage anpasst, sieht sie an
Mehrfachspieltagen nicht wirken. Die bestehende „Variante geändert"-Meldung in
`RegenSummary` macht sichtbar, dass dort ein anderer Typ greift; ein eigener Hinweis wird
nicht ergänzt.

## Decision 4 — Der Slot ist die Quelle für `duty_accounts.ist`

**Entscheidung:** Die Neuberechnung in `DeleteGame` (`games/handler.go:1532`) wechselt von
`SUM(dt.hours_value)` auf `SUM(ds.hours_value)`; der JOIN auf `duty_types` entfällt dort.

**Begründung:** Folgt zwingend aus Decision 1 + der Materialisierung. Sobald ein Slot seine
Dauer überschreiben darf und die Dauer die Gutschrift *ist*, wäre eine Aggregation über den
Typ schlicht falsch — sie ignorierte genau die Korrektur, die der Vorstand vorgenommen hat.
Der Slot-Wert enthält bereits die Varianten-Auflösung aus Decision 3, ohne dass die Abfrage
davon wissen muss.

**Angrenzender, hier nicht behobener Fehler:** `Fulfill` (`duties/handler.go:1172`) setzt
`duty_assignments.status='fulfilled'`, erhöht aber `duty_accounts.ist` nicht. Die Spalte
wird im gesamten Code nur an zwei Stellen geschrieben: `INSERT OR IGNORE (…, 0)` beim Ziehen
(Z. 1136) und die Neuberechnung in `DeleteGame`. Das Konto steht damit faktisch auf 0, bis
zufällig ein Termin gelöscht wird. Dieser Change wechselt die Quelle der einen
funktionierenden Aggregation und lässt die fehlende Buchung unangetastet — ihn hier
mitzuflicken würde die Änderung fachlich verdoppeln und den Test-Umfang sprengen. Er ist
als Folge-Change festzuhalten.

## Decision 5 — Einheit: `REAL` in Stunden, nicht `INTEGER` in Minuten

**Entscheidung:** Alle drei Ebenen führen `hours_value REAL`, wie `duty_types` es heute tut.

**Begründung:** Die neuen Spalten *überschreiben* `duty_types.hours_value` und fließen in
dieselbe `SUM()`. Minuten hätten jede Aggregation um ein `/60.0` erweitert und zwei
Einheiten für eine Größe eingeführt. `parseHoursInput`/`hoursToDisplay` arbeiten bereits auf
`REAL`-Stunden und werden unverändert wiederverwendet.

**Der Preis:** `1/3` Stunde ist in `REAL` nicht exakt. Für Dienstdauern (Vielfache von
5 Minuten) ist der Rundungsfehler in `Math.round(h * 60)` folgenlos; ein `20min`-Dienst
speichert `0.3333…` und zeigt wieder `20min`.

## Decision 6 — Anzeige-Details

- **Trennzeichen:** `8:00–9:00` (U+2013, Halbgeviertstrich), ohne nachgestelltes „Uhr".
- **Mitternacht:** `23:30–00:30`, kein Datumszusatz. Ein Dienst, der über Mitternacht läuft,
  ist selten genug, dass die Verkürzung zumutbar ist; `24:30` wäre falsch, ein Datum wäre in
  der Tabellenspalte zu breit.
- **Ohne Uhrzeit:** trägt ein Slot kein `event_time` (heute `'—'`), bleibt es bei `'—'` —
  eine Dauer ohne Startzeitpunkt ergibt keine Spanne.

## Migrationsplan

`052_dienst_dauer.up.sql`:

```sql
ALTER TABLE game_template_items ADD COLUMN hours_value REAL NOT NULL DEFAULT 1.0;
ALTER TABLE duty_slots          ADD COLUMN hours_value REAL NOT NULL DEFAULT 1.0;

UPDATE game_template_items
   SET hours_value = (SELECT hours_value FROM duty_types WHERE id = duty_type_id);
UPDATE duty_slots
   SET hours_value = (SELECT hours_value FROM duty_types WHERE id = duty_type_id);
```

Der Backfill wendet Copy-on-pick rückwirkend an: jede Bestandszeile bekommt den Wert des
Typs, den sie heute effektiv hätte. Keine Zeile bleibt auf dem Default stehen (jede trägt
eine gültige `duty_type_id`, beide Spalten sind `NOT NULL REFERENCES`).

`.down.sql` droppt beide Spalten. SQLite kann `DROP COLUMN` seit 3.35; `modernc.org/sqlite`
ist neuer. Bestehende Down-Migrationen im Projekt nutzen das bereits.

**Verifiziert per Hand, nicht über `make migrate-down`:** dieses Target rollt nichts zurück
— `runMigrate()` (`cmd/teamwerk/main.go:531`) liest das `up`/`down`-Argument gar nicht aus
und ruft in beiden Fällen `db.Migrate` (= up), quittiert aber mit „migrations applied". Ein
vermeintlicher Rollback lässt die Schema-Version unverändert. Vorbestehender Tooling-Fehler,
nicht Teil dieses Changes; der Down/Up-Round-Trip wurde stattdessen direkt per `sqlite3` auf
einer Kopie der Dev-DB geprüft (620 Slots, 0 Abweichungen nach dem erneuten Backfill).

**Deploy-Reihenfolge ist unkritisch** — anders als bei `dienst-slot-team-id-ausbauen` gibt
es keine Übergangsphase: die Spalten sind vom ersten Moment an gefüllt, und kein Lesepfad
kennt einen Zustand „Dauer fehlt". Ein Alt-Client, der `hours_value` nicht sendet, bekommt
serverseitig den Typ-Wert eingesetzt (siehe Spec, `POST /api/duty-slots`).

## Risks / Trade-offs

- **Sichtbare Änderung für alle Nutzer ohne Vorwarnung.** Nach dem Deploy trägt jeder Dienst
  eine Spanne. Ist ein Typ-Wert unrealistisch gepflegt (weil ihn bisher niemand als Uhrzeit
  gelesen hat), fällt das jetzt sofort auf. **Mitigation:** vor dem Deploy die
  `hours_value`-Liste auf `/diensttypen` einmal durchsehen — es sind wenige Zeilen.
  **Ergebnis der Durchsicht (12 Diensttypen, Stand Umsetzung):** die Werte lesen sich
  durchweg als echte Schichtlängen — Aufbau/Abbau klein 0,25 h, Aufbau Groß 0,75 h, Abbau
  Groß und Einkauf 0,5 h, Verkauf/Kamera/Zeitnehmer/Sekretär/Kuchen 1,0 h, Grill 2,0 h.
  Kein Wert muss vor dem Deploy korrigiert werden. Einzige Auffälligkeit ist der bereits
  in Decision 1 benannte `Spielbericht` (1,0 h, Anker Ende **+2880 min = 48 h**, nicht
  24 h wie in docs/agent/06-gotchas.md geschrieben) — er zeigt künftig eine
  Ein-Stunden-Spanne zwei Tage nach Spielende. Bewusst akzeptiert.
- **`is_custom=1` als stiller Nebeneffekt.** `UpdateSlot` setzt das Flag bedingungslos
  (`handler.go:600`); ein Slot mit korrigierter Dauer ist damit dauerhaft vom Regen
  ausgenommen. Bewusst unverändert übernommen (konsistent mit Uhrzeit- und
  Personen-Korrektur) und **ohne** Hinweis in der UI. **Trade-off akzeptiert:** wird die
  Dauer zum häufigsten Korrekturgrund, klinken sich mehr Slots aus der Vorlagen-Pflege aus
  als bisher. Der Massenlauf (`/api/duty-slots/bulk-regen`) bleibt der Weg zurück — sein
  `purge`-Zustand löscht auch `is_custom=1`-Slots.
- **Kein Schutz gegen `hours_value <= 0` im Bestand.** Die Validierung greift nur an den
  Schreibrouten. Eine per SQL gesetzte 0 zeigt `8:00–8:00`. Kein CHECK-Constraint, weil er
  bei einem `ALTER TABLE ADD COLUMN` in SQLite einen Tabellen-Rebuild erzwänge.
