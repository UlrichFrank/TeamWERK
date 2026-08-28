## Why

`duty_slots.team_id` ist bei jedem spielgebundenen Slot eine Kopie von `game_teams` —
über den gesamten Bestand (620 Slots) trägt kein einziger Slot eine `team_id`, die nicht
das Team seines Termins ist. Zwei Quellen für dieselbe Tatsache, von denen nur eine
mitwächst: `game_teams` folgt Kaderwechseln sofort, die kopierte `team_id` friert den
Stand vom Anlegen ein.

Diese Redundanz hat bereits einen Produktionsfehler erzeugt. Das Spieltag-Detail-Modal
schrieb `teams[0].id` in neue Slots; an einem Termin mit drei Kadern sah nur der erste
die Dienste (Change `dienst-slot-mehr-team-sichtbarkeit`, elf Slots auf Prod korrigiert).
Der Fix hat die Stelle geheilt, nicht die Klasse: solange das Feld existiert und
geschrieben werden darf, ist derselbe Fehler an jeder neuen Schreibstelle wieder möglich.

Der Nutzen des Feldes ist dabei gleich null. Es erreicht das Frontend nie (`boardSlot`
hat kein Team-Feld), der Team-Filter der Dienstbörse arbeitet auf `boardGroup.team_ids`
aus `game_teams`, und der `teamIDs`-Loop im Regen fächert faktisch nie auf: die zehn
Vorlagen-Items mit Team-Allowlist hängen an den Vorlagen „Heimspiel"/„Auswärts", deren
151 Termine ausnahmslos genau ein Team haben. Die Allowlist wirkt dort als Prädikat
(„gilt dieses Item für diese Altersklasse?"), nicht als Multiplikator.

## What Changes

- Ein Dienst-Slot mit gesetztem `game_id` trägt **kein** `team_id` mehr. Sichtbarkeit,
  Benachrichtigung und Erinnerung lösen ausschließlich über `game_id` → `game_teams` auf.
- Die Spalte `duty_slots.team_id` **bleibt** — sie ist der einzige Geltungsbereich für
  game-lose Slots (Vereinsfest o. ä., aktuell eine Zeile). Kein Schema-Abbau.
- Der `teamIDs`-Loop in `regenGameItems` wird zum Prädikat: die Team-Allowlist eines
  Vorlagen-Items entscheidet, **ob** ein Slot entsteht, nicht mehr **wie viele**.
- **BREAKING (fachlich, heute unerreicht):** ein Heim-/Auswärtsspiel mit mehreren Teams
  erzeugt künftig **einen** Slot je Vorlagen-Item statt einen pro Team. Im Bestand
  existiert kein solches Spiel (0 von 151), die Änderung ist also aktuell nicht
  beobachtbar — sie ist trotzdem eine Semantikänderung und keine Umbenennung.
- Die Bewirtungsrotation schreibt keine `team_id` mehr. Die Zuordnung „welche Mannschaft
  bringt die Kuchen" bleibt erhalten, weil der Slot ohnehin am **eigenen** Anker-Heimspiel
  dieser Mannschaft hängt und Heimspiele genau ein Team tragen.
- `restoreAssignments`/`customKey` matchen für spielgebundene Slots künftig über
  `(duty_type_id, event_time)`. Damit der Übergang keine Zusagen verliert, wird das
  Matching **vor** der Migration übergangsfest gemacht (alt mit `team_id`, neu ohne, gilt
  als derselbe Slot).
- Migration setzt `team_id = NULL` für alle Slots mit `game_id IS NOT NULL`
  (Bestand: 600 Zeilen, davon 38 mit Zusagen in der Zukunft).

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `duties`: Slots mit `game_id` tragen generell kein Team mehr (ersetzt die Anforderung
  aus `dienst-slot-mehr-team-sichtbarkeit`, die das nur für Mehr-Team-Termine forderte)
- `bewirtungsrotation`: Rotations-Slot trägt kein `team_id`; die Mannschaft ergibt sich
  aus dem Anker-Spiel
- `duty-assignment-preservation`: Matching-Schlüssel ohne `team_id` für spielgebundene
  Slots, plus Übergangsregel für den Migrationslauf
- `duty-reminder-emails`: Empfängerauflösung nur noch `game_teams` bzw. vereinsweit
- `push-duties`: dieselbe Vereinfachung für die Slot-Anlage-Benachrichtigung

## Impact

- `internal/games/regen.go` — `regenGameItems` (Team-Loop → Prädikat), `buildRotationPlan`
  (kein `team_id` im Insert), `makeCustomKey`/`snapshotCustomSlots`/`restoreAssignments`
- `internal/duties/handler.go` — `CreateSlot` ignoriert `team_id` bei gesetztem `game_id`;
  Board-Query verliert den `ds.team_id IN (…)`-Zweig für Spiel-Slots
- `internal/scheduler/scheduler.go`, `internal/dashboard/handler.go`,
  `internal/hub/audience.go` — je ein Doppelzweig weniger
- `internal/db/migrations/0NN_duty_slots_team_id_nur_ohne_spiel.{up,down}.sql`
- Frontend: `SpieltagDetailModal.tsx` sendet `team_id` nicht mehr mit (der gestern
  eingeführte `slotTeamIdForGame`-Helfer entfällt ersatzlos)
- Reihenfolge ist Teil des Vertrags: erst übergangsfestes Matching deployen, dann
  migrieren, dann das Schreiben abstellen (siehe design.md)
