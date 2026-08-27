## Context

Siehe proposal.md — Why. Für den Entwurf zählen drei Eigenschaften des Ist-Zustands:

1. **Die Sichtbarkeitsabfrage hat schon beide Zweige.** `duties/handler.go` prüft
   `ds.team_id IN (meine Teams)` **oder** `ds.team_id IS NULL AND ds.game_id IN (…)`.
   Der zweite Zweig ist die Zielsemantik, der erste der abzubauende.
2. **Alle Regen-Schlüssel sind pro Spiel skopiert.** `snapshotCustomSlots` und
   `loadNewAutoSlotsKeyed` laden ausschließlich Slots **eines** `game_id`. Innerhalb
   eines Spiels mit genau einem Team kann `team_id` deshalb nie zwei Slots
   unterscheiden — es ist im Schlüssel `(duty_type_id, event_time, team_id)` eine
   Konstante.
3. **38 Zusagen liegen in der Zukunft** und laufen bei jedem Regen-Lauf durch
   `restoreAssignments`. Sie sind das einzige, was bei einem Fehler wirklich kaputtgeht.

## Goals / Non-Goals

**Goals:**

- Der Übergang kommt ohne ein Zeitfenster aus, in dem Slots unsichtbar sind oder
  Zusagen verloren gehen — auch wenn Migration und Deploy auseinanderfallen.
- Rollback bleibt jederzeit möglich, auch **nach** der Migration.
- `team_id` bleibt als Geltungsbereich für game-lose Slots unangetastet.

**Non-Goals:**

- Die Spalte löschen. Sie trägt den game-losen Fall weiter.
- Eine Team-Teilmenge pro Slot (`duty_slots.team_ids`-Allowlist). Eigener Change,
  falls der Bedarf je auftritt.
- `game_template_items.team_ids` anfassen. Die Allowlist bleibt, ihre Rolle wird nur
  von „Multiplikator" auf „Prädikat" präzisiert — was sie faktisch längst ist.

## Decisions

### 1. Erst aufhören zu lesen, dann aufhören zu schreiben, dann migrieren

Die naheliegende Reihenfolge — migrieren, dann Code umstellen — erzwingt ein Fenster, in
dem beides zusammenpassen muss. Stattdessen wird der **Lesepfad zuerst tolerant**: die
Board-, Dashboard-, Scheduler- und Audience-Abfragen lösen für **jeden** Slot mit
`game_id` über `game_teams` auf und schauen `ds.team_id` gar nicht mehr an. Bedingung
`ds.team_id IS NULL AND ds.game_id IN (…)` wird zu `ds.game_id IN (…)`.

Damit ist ein Bestands-Slot mit `team_id = 13` **und** ein neuer mit `NULL` identisch
sichtbar. Die Migration verliert ihren Charakter als Voraussetzung und wird reine
Hygiene: sie kann Tage später laufen, in Teilen, oder gar nicht.

*Alternative:* Migration und Deploy koppeln (Wartungsmodus, ein Fenster). Verworfen —
teurer und riskanter für einen Gewinn, den die tolerante Leseseite gratis liefert.

### 2. Der Matching-Schlüssel ignoriert `team_id` bei spielgebundenen Slots

`makeCustomKey` setzt `HasTeam=false`, sobald der Slot ein `game_id` trägt. Wegen
Context-Punkt 2 ist das innerhalb eines Spiels informationserhaltend. Der Effekt: ein
Bestands-Slot (`team_id=13`) und der beim nächsten Regen-Lauf an seiner Stelle
entstehende neue Slot (`NULL`) fallen auf denselben Schlüssel — `restoreAssignments`
matcht sie, die Zusage überlebt. Ohne diesen Schritt wäre genau das der Punkt, an dem
die 38 Zusagen fielen.

Diese Änderung **muss im selben Deploy** liegen wie „Inserts schreiben NULL" (Decision 3).
Getrennt deployt wäre entweder der Schlüssel inkonsistent zur Insert-Seite oder umgekehrt.

*Alternative:* Fuzzy-Match mit Fallback auf den alten Schlüssel. Verworfen — zwei
Matching-Regeln nebeneinander sind dauerhaft schwerer zu verstehen als eine, und der
Sonderfall wäre nie wieder zu entfernen.

### 3. Der Team-Loop wird zum Prädikat

`regenGameItems` iteriert heute `for _, tid := range teamIDs` und legt je Team einen Slot
an. Künftig entscheidet `itemAppliesToAnyTeam(it.TeamIDs, teamIDs)` einmal, ob das Item
für diesen Termin gilt, und legt **einen** Slot mit `team_id = NULL` an. Die Funktion
existiert bereits — sie ist heute die Vorschau-Variante (`PreviewSlots`), womit Vorschau
und Erzeugung nebenbei auf dieselbe Bedingung zusammenfallen.

Bei genau einem Team des Termins ist das Verhalten bitgleich zu heute. Bei mehreren
Teams entsteht ein geteilter Slot statt N paralleler — die gewollte Semantik laut
proposal.md.

### 4. Rotations-Slots verlieren die Team-Zuordnung, behalten die Zuteilung

`buildRotationPlan` bleibt unverändert: es baut weiter eine Team-Warteschlange, rechnet
den Kuchenbedarf und weist jeder Mannschaft ihre Kuchenzahl an **ihrem** Anker-Heimspiel
zu. Nur der Insert schreibt kein `team_id` mehr. Die Aussage „Mannschaft X bringt drei
Kuchen" bleibt lesbar — sie steht im Anker-Spiel, und ein Heimspiel trägt ein Team.

Bekommt ein Heimspiel je zwei Teams, treten beide an derselben Ankerposition in die
Warteschlange ein und ihre Zuteilungen werden zu **einem** Slot mit summierten Kuchen
verschmolzen (statt wie heute zwei Slots mit je eigener `team_id`). Das ist konsistent
mit Decision 3, kostet aber die Pro-Mannschaft-Auskunft in genau diesem Fall. Siehe
Risiken.

### 5. Migration ist ein reines `UPDATE`, ohne `down`-Datenverlust

`UPDATE duty_slots SET team_id = NULL WHERE game_id IS NOT NULL`. Das `down`-Skript kann
den alten Wert nicht rekonstruieren — muss es aber auch nicht: unter dem alten Code ist
`team_id IS NULL` bei gesetztem `game_id` ein vollständig unterstützter Zustand (der
`IS NULL`-Zweig existiert dort schon). Das `down` bleibt deshalb leer, mit Kommentar.

## Risks / Trade-offs

- **Ein Heimspiel mit zwei Teams verliert die Pro-Mannschaft-Kuchenzuteilung** (Decision 4)
  → Heute 0 von 151 Fällen; `buildRotationPlan` filtert ohnehin auf `event_type='heim'`,
  und ein Heimspiel ist fachlich das Spiel *einer* Mannschaft. Ein Test pinnt das
  Verschmelzungsverhalten, damit es eine bewusste Zusage ist und kein Zufall. Wer die
  Pro-Team-Auskunft braucht, braucht den `duty_slots.team_ids`-Change (Non-Goal).
- **Regen-Läufe schreiben Bestands-Slots stillschweigend um** → Sobald ein Termin aus
  anderem Anlass regeneriert wird, verschwindet dessen `team_id` schon vor der Migration.
  Das ist unschädlich (Decision 1), macht aber „wie viele Zeilen sind noch alt?" zu einer
  wandernden Zahl. Die Migration ist idempotent und deckt den Rest ab.
- **Die tolerante Leseseite verbirgt eine kaputte Schreibseite** → Schriebe jemand künftig
  wieder eine `team_id` an einen spielgebundenen Slot, fiele es nicht mehr auf, weil
  niemand sie liest. Deshalb prüft ein Test, dass `CreateSlot` und der Regen bei gesetztem
  `game_id` NULL schreiben — nicht nur, dass die Sichtbarkeit stimmt.
- **Kein `down`-Pfad für die Daten** (Decision 5) → Akzeptiert, weil der Zielzustand unter
  altem wie neuem Code gültig ist. Vor der Migration trotzdem `make backup`.

## Migration Plan

1. **Deploy A** (ein Schritt, alles zusammen): Lesepfade ignorieren `ds.team_id` bei
   gesetztem `game_id`; `makeCustomKey` ignoriert es ebenso; Regen und `CreateSlot`
   schreiben NULL; Team-Loop → Prädikat. Ab hier ist der Bestand irrelevant.
2. **Migration** (jederzeit danach): `UPDATE duty_slots SET team_id = NULL WHERE game_id
   IS NOT NULL`. Vorher `make backup`.
3. **Rollback**: Binary zurückrollen. Auch mit bereits migrierten Daten funktioniert der
   alte Code, weil dessen `team_id IS NULL AND game_id`-Zweig genau diesen Zustand
   abdeckt. Ein Datenrollback ist nicht nötig und nicht vorgesehen.
