## Context

Siehe `proposal.md` — Why. Die relevanten Bausteine liegen alle in `internal/games/regen.go`:

- `buildRotationPlan` läuft einmal pro Tag, vor der Pro-Spiel-Schleife, gruppiert nach
  `duty_type_id`, und liefert heute `rotationPlan = map[dutyTypeID]map[gameID]rotationAssignment`
  mit `{HasSlot bool, TeamID sql.NullInt64}`.
- `regenGameItems` liest im Rotations-Zweig `rotation[it.DutyTypeID][g.ID]` und legt bei
  `HasSlot` einen Slot mit `slots_total = it.SlotsCount` an.
- `makeCustomKey` lässt für rotations-aktive Duty-Types `team_id` aus dem Schlüssel fallen —
  benutzt sowohl von der `is_custom=1`-Konflikterkennung als auch von `restoreAssignments`.

Randbedingungen, die der Umbau nicht verletzen darf: der Plan wird innerhalb derselben `tx`
gebaut wie der Rest des Regens; das Ausrichter-Gate (`itemPassesAusrichterGate`) filtert die
rotations-aktiven Items schon beim Sammeln, also vor der Bedarfsrechnung; die Preview
(`ApplyBulkRegen` mit `Rollback`) ist derselbe Codepfad wie der Apply.

## Goals / Non-Goals

**Goals:**

- Die Zuteilungseinheit ist der Kuchen, nicht der Slot. Ein Slot bündelt die Kuchen **einer**
  Mannschaft.
- Ein Rotations-Slot ist wieder ein Dienst einer Mannschaft zu ihrem eigenen Spiel — damit
  fällt die Sonderbehandlung im Match-Key weg.
- Kein neues Schema, keine Migration, keine neue Route.

**Non-Goals:**

- Kein saisonweiter Rotationszustand (unverändertes Non-Goal aus `kuchendienst-rotation`).
- Keine Verteilung der Kuchen einer Mannschaft über mehrere ihrer Spiele. Eine Mannschaft
  bekommt genau einen Slot pro Tag und Duty-Type.
- Keine Nachrüstung bestehender Slots. Wirkung erst beim nächsten Regen-Lauf des Tages.

## Decisions

### Decision 1: Der Plan wird team-zentriert statt spiel-zentriert

`rotationPlan` wird zu `map[dutyTypeID]map[gameID]rotationAssignment` mit
`rotationAssignment{TeamID int, Cakes int}` — ein Eintrag pro **Anker-Spiel**, nicht mehr pro
Heimspiel bis zum Bedarf. `regenGameItems` bleibt damit strukturell unverändert (es fragt
weiterhin `rotation[dutyTypeID][g.ID]`), erhält aber höchstens einen Eintrag je Mannschaft und
nutzt `Cakes` als `slots_total`.

**Alternative:** Den Plan nach `teamID` schlüsseln und in `regenGameItems` rückwärts suchen.
Verworfen — die Pro-Spiel-Schleife müsste dann wissen, welches Spiel Anker ist, und die
Zuordnung wäre an zwei Stellen formuliert.

### Decision 2: Das Anker-Spiel ist das Spiel, das die Warteschlangen-Position bestimmt hat

`rotationGroup` merkt sich beim Aufbau der Warteschlange pro Team zusätzlich die `game_id`, an
der es eingetreten ist (`anchorByTeam map[int]int`). Damit ist der Anker per Konstruktion
dasselbe Spiel, das die Reihenfolge festlegt — es kann nicht auseinanderlaufen.

Wichtig: Anker-Kandidaten sind nur Spiele, die dieses Item überhaupt tragen (Vorlage enthält
es) **und** das Ausrichter-Gate passieren. Ein Team, dessen frühestes Heimspiel eine andere
Vorlage nutzt, tritt erst mit seinem nächsten passenden Spiel in die Warteschlange ein — und
genau dort hängt dann auch sein Slot. Beides fällt in dieselbe Schleife, es braucht keine
zweite Auflösung.

### Decision 3: `slots_count` wird für Rotations-Items ignoriert statt validiert

Der Vorlagen-Editor deaktiviert das Feld, der Server prüft es aber nicht (kein
`400`-Sonderfall). Grund: `slots_count` ist NOT NULL mit sinnvollem Default; ein Item, das
zwischen Rotation an/aus wechselt, würde sonst beim Umschalten einen Validierungsfehler
werfen, obwohl der Wert für den anderen Modus korrekt ist. Der Wert bleibt erhalten und wirkt
wieder, sobald die Rotation abgeschaltet wird.

**Alternative:** `slots_count` beim Speichern auf `1` normalisieren. Verworfen — zerstört den
Wert für den Fall, dass die Rotation später abgeschaltet wird.

### Decision 4: Restbedarf verfällt, ohne Auffang-Slot

Entschieden vom Vorstand (siehe `proposal.md`). Technisch bedeutet das: `UnassignedEntry`
verliert `GameID` und bekommt `Count int` (Anzahl nicht zugeteilter Kuchen); der Eintrag
entsteht **einmal pro Tag und Duty-Type** in `buildRotationPlan`, nicht mehr pro Slot in
`regenGameItems`. Weil `buildRotationPlan` das Datum nicht als Parameter hat, wird
`UnassignedEntry` künftig in `regenSingleDay` mit dem dort bekannten `date` befüllt — der Plan
liefert nur die Zahl.

Damit verschwindet auch der einzige Weg, auf dem ein Rotations-Slot ohne `team_id` entstehen
konnte; der `team_id IS NULL`-Fallback der Dienstbörse bleibt für andere Fälle bestehen, wird
von der Rotation aber nicht mehr benutzt.

### Decision 5: `makeCustomKey` verliert den Rotations-Sonderfall vollständig

Der Parameter `rotationTypes` entfällt aus `makeCustomKey`, `snapshotCustomSlots` und
`restoreAssignments`; `regenSingleDay` baut die Menge nicht mehr auf. Das ist eine echte
Rückabwicklung von `kuchendienst-rotation` Decision 5, keine Erweiterung — die Begründung von
damals („team_id ist aus der Tagesreihenfolge abgeleitet und damit verschiebbar") gilt nach
Decision 2 nicht mehr, und die Ausnahme wäre jetzt sogar schädlich: zwei Mannschaften mit
gleichzeitigem Heimspiel bekämen Slots mit identischem `(duty_type_id, event_time)` und
würden im Restore verwechselt.

### Decision 6: Bedarf ohne Deckelung auf die Spieleanzahl

`demand = ceil(games × verhaeltnis)`, entschieden vom Vorstand. Die alte Deckelung war eine
Folge davon, dass pro Spiel höchstens ein Slot entstehen konnte. Da eine Mannschaft jetzt
mehrere Kuchen in einem Slot trägt, ist ein Verhältnis > 1 sinnvoll ausdrückbar. Die untere
Grenze bleibt implizit: `verhaeltnis > 0` erzwingt bereits die Settings-Route.

## Risks / Trade-offs

- **Bestehende Rotations-Slots werden erst beim nächsten Regen korrigiert** → Nach dem Deploy
  einen Massenlauf („Dienste aktualisieren") über den betroffenen Zeitraum fahren. Die
  Vorschau zeigt die Bilanz vorher; der Zeitraum ist ohnehin auf `[morgen, …]` geklemmt.
- **Zusagen gehen verloren, wo Slots wegfallen** → Unvermeidbar: aus fünf Slots werden drei.
  Betroffene laufen regulär in die „entfernt"-Benachrichtigung. Deshalb den Massenlauf
  bewusst und angekündigt fahren, nicht nebenbei.
- **Eine Spielplanänderung kann die Zuteilung verschieben** → Tritt eine Mannschaft mit einem
  früheren Heimspiel neu in die Warteschlange ein, rutschen die hinteren Positionen und die
  letzte Mannschaft kann herausfallen. Das war vorher durch den team-losen Match-Key teilweise
  abgefedert; jetzt wird es sichtbar als „entfernt". Bewusst akzeptiert — die Meldung ist
  ehrlicher als eine Zusage, die stillschweigend zu einer anderen Mannschaft wandert.
- **Verhältnis > 1 kann mehr Kuchen fordern, als der Cap hergibt** → Der Rest verfällt und
  steht in der Zusammenfassung. Wer das nicht will, hebt den Cap.
- **Zwei gleichzeitige Heimspiele derselben Mannschaft** (Dubletten aus dem H4A-Import) zählen
  doppelt in die Bedarfsrechnung → Kein Sonderfall in der Engine; Dubletten sind Datenpflege
  und werden vom Import bereits als solche gemeldet.

## Migration Plan

Kein Schema-Wechsel, keine Datenmigration. Nach dem Deploy:

1. Einstellungen → Bewirtung prüfen (Verhältnis, Max. Kuchen pro Mannschaft).
2. Massenlauf „Dienste aktualisieren" über den Zeitraum mit Heimspieltagen, Vorschau prüfen,
   dann anwenden.

Rollback: Binary zurückrollen und denselben Massenlauf erneut fahren — die Slots werden aus
der Vorlage neu erzeugt, es gibt keinen persistierten Rotationszustand, der driften könnte.
