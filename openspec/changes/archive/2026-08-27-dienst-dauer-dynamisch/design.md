## Context

Siehe proposal.md — Why. Drei Eigenschaften des Ist-Zustands tragen den Entwurf:

1. **Die Anker-Auflösung existiert und ist korrekt.** `regen.go` (heute Z. 883) löst
   `anchor='end'` gegen `games.end_time` auf, falls gepflegt, und sonst gegen
   Anpfiff + `durationMins`. Diese Fallunterscheidung ist genau die, die eine dynamische
   Endzeit braucht — sie darf nicht ein zweites Mal geschrieben werden.
2. **`durationMins` ist immer bekannt, wenn Items verarbeitet werden.** Kann
   `effectiveEventDurationTx` die Spieldauer nicht bestimmen (Team ohne Altersklasse,
   Vorlage ohne `duration_minutes`), überspringt `runAutoRegen` das Spiel bereits
   vollständig (`continue`, Z. 227). Der dynamische Modus erbt damit **kein** neues
   Fehlerbild.
3. **Der Slot ist ein eingefrorener Schnappschuss.** `event_time`, `slots_total`,
   `audiences` und seit `dienst-dauer` auch `hours_value` werden beim Regen aufgelöst und
   geschrieben. Eine dynamische Dauer fügt sich ein, ohne dass `duty_slots` etwas Neues
   lernen muss.

## Goals / Non-Goals

**Goals**

- Ein Dienst kann „so lang wie das Spiel" sein, ohne je Altersklasse eine eigene Vorlage.
- Halbzeit-Dienste (beide Anker am Anpfiff) sind ausdrückbar.
- Eine fehlerhafte Definition führt nie zu einem fehlenden Dienst.

**Non-Goals**

- Keine dynamische Dauer an manuell angelegten Slots (`is_custom=1` — der Regen fasst sie
  nie an, das Etikett wäre eine Lüge).
- Keine über Mitternacht laufende Auflösung: Anker und Versätze werden wie heute im
  Tagesraster gerechnet.
- Keine Änderung an `duty_accounts.ist` — die aufgelöste Dauer ist weiterhin zugleich die
  Gutschrift (`dienst-dauer`, Decision 1 gilt unverändert).

## Decision 1 — Ein Modus-Feld, kein NULL-als-Signal

**Entscheidung:** `duration_mode TEXT CHECK(duration_mode IN ('absolut','dynamisch'))`,
Default `'absolut'`.

**Alternative (verworfen):** kein Modus-Feld, stattdessen `end_anchor IS NULL` = absolut.

**Begründung:** Ein Modus, den man lesen kann, schlägt einen, den man aus der
Abwesenheit dreier Felder erschließen muss — besonders in der Maske, die einen
Radio-Umschalter zeigt. Zudem bleiben die End-Felder beim Umschalten auf `absolut`
erhalten und sind beim Zurückschalten wieder da; mit NULL-als-Signal wären sie verloren.

SQLite erlaubt `CHECK`-Constraints auf per `ALTER TABLE ADD COLUMN` hinzugefügten Spalten
und setzt sie durch — nachgeprüft, entgegen einer falschen Annahme im Design von
`dienst-dauer` (dort korrigiert).

## Decision 2 — Die Anker-Auflösung wird ein Helfer

**Entscheidung:** Der Block aus `regenGameItems` wandert in

```go
func resolveAnchorTime(anchor string, offsetMinutes int, g dayGame, durationMins int) string
```

und wird zweimal aufgerufen: für den Start (unverändertes Verhalten) und für das Ende.

**Begründung:** Die Fallunterscheidung „`end_time` gepflegt oder Anpfiff + Spieldauer" ist
die eigentliche Fachlogik. Zweimal geschrieben würde sie irgendwann auseinanderlaufen —
und zwar still, weil beide Zweige plausible Uhrzeiten liefern.

**Nebeneffekt, ausdrücklich gewollt:** Der Start behält dadurch garantiert exakt sein
heutiges Verhalten; der Refactor ist verhaltensneutral und durch
`TestRegen_AbsoluterModusUnveraendert` abgesichert.

## Decision 3 — `hours_value` bleibt der Rückfall

**Entscheidung:** Im dynamischen Modus bleibt `hours_value` gepflegt. Der Regen rechnet

```
  start    = resolveAnchorTime(anchor,     offset_minutes,     …)
  ende     = resolveAnchorTime(end_anchor, end_offset_minutes, …)
  dauer    = ende − start
  hours    = dauer > 0 ? dauer/60 : hours_value      ← Rückfall
```

**Alternative (verworfen):** den Slot überspringen und die Fehldefinition in
`RegenSummary` melden.

**Begründung:** Ein fehlender Dienst ist teurer als ein zu langer. Fällt ein Zeitnehmer aus
dem Plan, merkt es niemand bis zum Spieltag; eine falsche Spanne fällt beim Eintragen auf.
Der Rückfall macht die Zusage-Kette (`restoreAssignments`) außerdem unempfindlich: ein Slot
verschwindet nicht mitsamt seinen Zusagen, weil jemand einen Versatz vertippt hat.

**Der Preis, ausgeschrieben:** Die Fehldefinition bleibt unsichtbar — der Slot sieht
plausibel aus, obwohl seine dynamische Regel nie greift. Ein Hinweis in `RegenSummary`
(„Dienst X: dynamische Dauer nicht auflösbar, absolute verwendet") wäre die naheliegende
Ergänzung und ist bewusst **nicht** Teil dieses Changes — er verlangt eine neue
Meldungsart in der Bilanz-Karte, und der Rückfall ist ohne sie schon sicher.

## Decision 4 — `duty_slots` bleibt unverändert

**Entscheidung:** Kein `duration_mode` am Slot. Der Regen materialisiert nur das Ergebnis.

**Begründung:** Folgt aus Context 3. Ein Modus am Slot wäre entweder redundant (er wird nie
neu ausgewertet) oder irreführend (bei `is_custom=1` würde er „folgt dem Spiel"
behaupten, obwohl der Regen den Slot nie wieder anfasst).

**Konsequenz für die Anzeige:** Die Dienstbörse kann nicht sagen „diese Spanne folgt dem
Spiel". Das ist akzeptiert — für den Eltern-Blick auf `/dienste` zählt die konkrete
Uhrzeit, nicht ihre Herkunft.

## Decision 5 — Beide Anker auch für das Ende

**Entscheidung:** `end_anchor` nimmt dieselben Werte wie `default_anchor` (`start`|`end`).

**Begründung:** Kostet nichts (derselbe Helfer) und deckt den Halbzeit-Fall ab: Start
Anpfiff +25 min, Ende Anpfiff +40 min. Eine auf „nur Spielende" reduzierte Variante hätte
diese Dienste weiter in den absoluten Modus gezwungen — und genau dort ist die feste Zahl
sogar richtig, aber der Vorstand müsste den Unterschied kennen, statt einfach beide Enden
gleich zu denken.

## Migrationsplan

`053_dienst_dauer_dynamisch.up.sql` — je drei Spalten auf `duty_types` und
`game_template_items`:

```sql
ALTER TABLE duty_types ADD COLUMN duration_mode TEXT NOT NULL DEFAULT 'absolut'
  CHECK(duration_mode IN ('absolut','dynamisch'));
ALTER TABLE duty_types ADD COLUMN end_anchor TEXT NOT NULL DEFAULT 'end'
  CHECK(end_anchor IN ('start','end'));
ALTER TABLE duty_types ADD COLUMN end_offset_minutes INTEGER NOT NULL DEFAULT 0;
-- dieselben drei auf game_template_items
```

Kein Backfill nötig: `'absolut'` ist das heutige Verhalten, und `hours_value` ist überall
gepflegt. Die End-Felder tragen sinnvolle Startwerte (`end` + 0 = „bis Spielende"), damit
das Umschalten auf `dynamisch` sofort etwas Vernünftiges ergibt.

`.down.sql` droppt alle sechs Spalten.

## Risks / Trade-offs

- **Der Rückfall verbirgt Fehldefinitionen.** Siehe Decision 3. Mitigation ist die Maske:
  sie zeigt beim Bearbeiten eine Beispielrechnung („bei 60 min Spieldauer: 09:30–11:45"),
  damit ein negativer Versatz sofort auffällt.
- **Copy-on-pick betrifft jetzt sechs Felder statt drei.** `refreshItemsFromDutyTypes`
  (`web/src/lib/dutyTemplateItems.ts`) muss mitwachsen, sonst frischt „Aus Diensttypen
  auffrischen" den Modus **nicht** mit auf und hinterlässt eine Zeile, deren Modus und
  Dauer aus verschiedenen Ständen stammen. Das ist der wahrscheinlichste Fehler in diesem
  Change und deshalb ein eigener Task mit eigenem Test.
- **Die Spieldauer ist nur so gut wie `age_class_game_rules`.** Fehlt die Regel, überspringt
  der Regen das Spiel schon heute komplett — der dynamische Modus macht diesen
  Bestandsmangel sichtbarer, weil dann auch die Dauer davon abhängt.
