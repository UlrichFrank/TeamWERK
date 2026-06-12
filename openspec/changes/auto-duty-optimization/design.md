## Context

`internal/games/handler.go` enthält bereits die komplette skip/reduce-Logik in den Helpern `loadSameDayContext`, `classifySlotPosition` und `applyBehavior` (Zeilen 118–316). Die Helper sind privat zum Package und werden ausschließlich von `RegenerateDaySlots` und `RegenerateSlots` aufgerufen — beide sind HTTP-Endpunkte, die der Vorstand explizit per Knopfdruck triggert.

Dieses Change verschiebt den Entry Point: statt durch User-Klick getriggert zu werden, läuft die Logik implizit nach jeder relevanten Mutation an Heim-/Auswärtsspielen. Die Helper bleiben unverändert; neu ist die Orchestrierung um sie herum.

Drei Spannungsfelder treiben die Entscheidungen:

1. **Datenkonsistenz vs. UX-Stabilität.** Heute sind befüllte Slots eine harte Garantie — sie verschwinden nie ohne expliziten User-Akt. Das Change bricht damit, weil ein konsistenter Dienstplan unter dynamischen Nachbarschaftsregeln nicht anders machbar ist (siehe Decision: „Befüllte Slots werden bei skip/reduce gelöscht").
2. **Template-Wahrheit vs. inline Edits.** Wenn das Backend autoritativ aus dem Template ableitet, müssen manuelle Slot-Edits geschützt werden, sonst gehen sie beim nächsten Spielplan-Touch verloren.
3. **Performance vs. Korrektheit der Drei-Tage-Logik.** Die Adjacency hängt nur von direkten Nachbartagen ab, also reicht ein Fenster von ±1. Aber: bei einem Datum-Move muss sowohl das alte als auch das neue Fenster regeneriert werden — sechs Tage in einem Worst Case.

## Goals / Non-Goals

**Goals:**

- Der Vorstand kann ein Heim-/Auswärtsspiel anlegen, verschieben, löschen — und der Dienstplan der drei betroffenen Tage ist sofort konsistent mit den `same_day_behavior`- und `adjacent_day_behavior`-Regeln der `duty_types`.
- Manuell hinzugefügte oder editierte Slots überleben Auto-Regen unverändert.
- Die UI auf `/kalender` und `/kalender/{id}` enthält keinen „Dienste generieren"-Knopf mehr. Der Vorgang ist unsichtbar.
- Helfer, deren Dienst durch Auto-Regen entfällt oder die Variante wechselt, bekommen eine Push-Benachrichtigung in Kategorie `duties`.

**Non-Goals:**

- Keine Migration von Template-/DutyType-Änderungen — wenn der Vorstand `same_day_behavior` eines duty_types ändert, wird kein Auto-Regen über die Saison hinweg getriggert. Dafür kommt später eine separate „Saison-Dienstplan optimieren"-Funktion auf anderer Ebene.
- Keine Konfliktauflösung von doppelten Slots zwischen Spielen am selben Tag — `regen_summary.conflicts` listet sie nur, die Lösung bleibt beim Vorstand (wie heute).
- Keine Slot-Übernahme bei `reduced`-Variante: wenn „Kassendienst Voll" zu „Kassendienst Reduziert" wird, wird der alte Slot gelöscht und ein neuer angelegt; der Helfer wird benachrichtigt, dass er sich neu eintragen kann/muss. Automatisches „Mit-Migrieren" der `duty_assignment`-Zeile wäre überraschendes Verhalten — der Helfer könnte den neuen Slot gar nicht mehr leisten wollen.

## Decisions

### Decision: Befüllte Slots werden bei skip/reduce gelöscht, mit Push an den Helfer

**Was:** Die heutige Logik in `RegenerateDaySlots` (Zeile 1485-1486) schützt befüllte Slots (`slots_filled > 0`) — sie werden nicht aus dem Template neu erzeugt, sondern beibehalten. Diese Schutzlogik entfällt. Auto-Regen löscht alle Slots mit `is_custom=0`, unabhängig von `slots_filled`, und legt sie laut aktuellem Template + Adjacency neu an. Vor dem Delete werden die `user_id`s der `duty_assignments` der gelöschten Slots gesammelt und nach Commit per `notify.Send(db, cfg, uids, "duties", "Dienst angepasst", "...", "/dienste")` benachrichtigt.

**Warum:** Konsistenz mit dem Template ist nicht optional — wenn der Vorstand einen Spielplan ändert, MUSS der Dienstplan der Adjacency-Regel folgen, sonst ist die ganze Auto-Logik wertlos. Die heutige Schutzlogik existiert nur, weil „Dienste generieren" ein expliziter User-Akt ist, bei dem niemand erwartet, dass Helfer rausfliegen. Sobald die Regeneration implizit läuft, ist die Erwartungshaltung anders — der Vorstand erwartet, dass das System „richtig" reagiert. Push-Benachrichtigung kompensiert die Überraschung für den Helfer.

**Alternativen verworfen:**

- *Befüllte Slots schützen, Inkonsistenz akzeptieren.* Würde die Auto-Logik unzuverlässig machen — nach drei Spielplan-Änderungen wäre der Dienstplan ein Flickenteppich aus alten und neuen Regeln.
- *Conflict-Marker setzen statt löschen.* Würde eine neue UI-Schicht für „dieser Slot ist veraltet, will der Helfer migrieren?" erfordern. Zu viel Komplexität für seltene Fälle.

### Decision: Auto-Regen-Fenster ist Event-Datum ± 1 Tag

**Was:** Wenn ein Event an Datum D angelegt/geändert/gelöscht wird, regeneriert das Backend Dienstpläne für D-1, D, D+1. Bei UpdateGame mit Datums-Move von D_alt nach D_neu wird das Fenster sowohl `{D_alt-1, D_alt, D_alt+1}` als auch `{D_neu-1, D_neu, D_neu+1}` verarbeitet (Set-Union, damit gemeinsame Tage nicht doppelt regeneriert werden).

**Warum:** `loadSameDayContext` (Zeile 281) lädt `allGameTimes` für genau das Event-Datum und prüft `prev_count`/`next_count` für direkt benachbarte Tage. Die skip/reduce-Logik kennt keine weiteren Datums-Abhängigkeiten. Ein größeres Fenster wäre Verschwendung; ein kleineres ginge an `adjacent_day_behavior` vorbei.

**Alternativen verworfen:**

- *Nur das Event-Datum regenerieren.* Würde `adjacent_day_behavior` an Nachbartagen ignorieren — z.B. das alte Heimspiel an N-1 hätte weiterhin „normal"-Dienste, obwohl jetzt ein zweites Heimspiel an N steht.
- *Ganze Saison regenerieren.* Zu teuer, vor allem auf VPS XS (1 GB RAM, SQLite). Bei 30+ Spielen pro Saison sind das im Worst Case 90 Slot-Inserts pro Mutation.

### Decision: Auto-Regen schont Slots mit `is_custom=1`

**Was:** Neue Spalte `duty_slots.is_custom INTEGER NOT NULL DEFAULT 0`. Wer einen Slot manuell über `POST /api/duty-slots` oder `PUT /api/duty-slots/{id}` anlegt/ändert, schreibt `is_custom=1`. Auto-Regen löscht nur Slots mit `is_custom=0` und überschreibt nie `is_custom=1`-Slots. Wenn ein Auto-Regen-Slot zeitlich/typgleich auf einen `is_custom=1`-Slot fällt, wird der Auto-Slot nicht angelegt (silent skip, taucht in `regen_summary.conflicts` auf).

**Warum:** Manuelle Slot-Edits sind seltene Sonderfälle (z.B. „diesmal brauchen wir 4 statt 2 Helfer"). Sie haben eine andere Wahrheitsquelle als das Template — der Vorstand hat sie bewusst gesetzt. Auto-Regen darf sie nicht überschreiben.

**Bestandsbehandlung:** Migration 037 setzt `is_custom=0` für alle existierenden Slots. Das ist konservativ falsch für manuell editierte Bestandsslots, aber sicherer als der umgekehrte Default — beim ersten Auto-Regen nach Deploy werden sie regeneriert, was bei laufender Saison maximal eine Welle Push-Benachrichtigungen auslöst. Vor Deploy: Vorstand wird gebeten, manuell-editierte Slots zu identifizieren und mit `UPDATE duty_slots SET is_custom=1 WHERE id IN (…)` zu schützen (siehe tasks.md 5.1).

**Alternativen verworfen:**

- *`is_custom=1` als neuen Default für Bestand.* Würde dauerhaft den Auto-Regen für alle Saison-Bestandsspiele blockieren — die Feature wäre tot bei Launch.
- *Separate `manual_duty_slots`-Tabelle.* Doppelte Schema-Komplexität für einen Edge Case.

### Decision: CreateGame-Request schickt kein `slots[]` mehr für Heim/Auswärts

**Was:** Der Request-Body von `POST /api/admin/games` enthält weiterhin `template_id` (optional, sonst per `findTemplateForGame` ermittelt) — aber kein `slots[]` für `event_type ∈ {heim, auswärts}`. Für `event_type=generisch` bleibt `slots[]` erhalten und wird mit `is_custom=1` persistiert (kein Template existiert).

**Warum:** Eine Wahrheitsquelle. Wenn das Frontend Slots vorberechnen würde, müsste es die Adjacency-Logik kennen, die im Backend lebt. Race-Conditions („zwei Vorstände speichern gleichzeitig, Slot-Vorberechnung divergent") wären die Folge.

**Wizard-UX:** Vor dem Save bietet der Wizard einen Read-only-Preview „So sieht der Dienstplan aus" — implementiert über einen neuen Endpoint `POST /api/admin/games/preview-duties` (Body identisch zu `CreateGame`, Response identisch zu `regen_summary`, aber ohne Persistenz und ohne Notification). Der Preview-Endpoint ist ein zusätzlicher Service; er ist nicht zwingend nötig, hilft aber dem Vorstand vor Save zu sehen, was passieren wird. Implementierung folgt in Phase 2 (siehe tasks.md).

**Alternativen verworfen:**

- *`slots[]` weiterhin akzeptieren als „Overrides".* Würde die Auto-Logik untergraben — der Vorstand könnte versehentlich alte Slot-Daten mitschicken und das Template aushebeln.
- *Frontend Slots vorberechnen.* Duplizierte Logik in Go und TypeScript, Inkonsistenzrisiko.

### Decision: Bestehende Endpoints `regenerate-day` und `{id}/regenerate` bleiben als interne Wrapper

**Was:** Die HTTP-Handler `RegenerateDaySlots` und `RegenerateSlots` bleiben registriert. Sie rufen intern `runAutoRegen(ctx, date, seasonID)` auf. Ihre HTTP-Routen bleiben in `main.go` — sie sind aktuell auf vorstand/trainer/admin geschützt. Frontend nutzt sie nicht mehr; sie sind dann „dead routes" bis eine spätere „Saison-Optimieren"-Funktion sie aufgreift.

**Warum:** Geringere Diff-Größe, weniger Risiko von Regressionen in den Helpern, klare Rückfallebene falls Auto-Regen unerwartet versagt (Vorstand kann via curl manuell triggern).

**Alternativen verworfen:**

- *Endpoints komplett entfernen.* Verlust einer Notfall-Repair-Möglichkeit. Auch späterer „Optimieren"-Bulk müsste sie neu erfinden.

### Decision: `runAutoRegen` läuft in der gleichen Transaction wie die Mutation

**Was:** `CreateGame`, `UpdateGame`, `DeleteGame` öffnen eine `tx`, mutieren das Game, rufen `runAutoRegen(ctx, tx, …)` in derselben tx auf, committen. Notification-Versand passiert NACH Commit per `notify.Send` als Goroutine (Status quo für alle `notify.Send`-Calls in dem Package).

**Warum:** Atomarität — entweder ist das ganze Drei-Tage-Fenster konsistent, oder die Mutation rollt zurück. Halbgare Zwischenzustände sind das Schlimmste, was passieren kann.

**Performance-Risiko:** Auto-Regen kann pro Mutation 3 Tage × ~5 Spiele/Tag × ~6 Slots/Spiel = ~90 INSERTs + DELETEs auslösen. SQLite WAL packt das problemlos. Sollte sich das in der Praxis als Bottleneck erweisen, kann die Notification-Sammlung weiter aus der tx raus gezogen werden (sie ist es schon).

**Alternativen verworfen:**

- *Auto-Regen async per Goroutine.* Race-Conditions mit gleichzeitig laufenden Mutations, plus die Mutation-Response könnte kein `regen_summary` liefern.

### Decision: Response-Schema `regen_summary` ist Best-Effort, nicht garantiert

**Was:** `regen_summary` enthält `created`, `reduced`, `skipped`, `notified_users`, `conflicts`. Bei sehr großen Fenstern (>20 Slots im Diff) wird die Liste in die ersten 20 Einträge pro Kategorie gekappt; das Frontend zeigt „… und N weitere Änderungen".

**Warum:** Eine UI mit 200 Zeilen Diff-Liste ist unbrauchbar. Wer Details will, kann den Kalender öffnen.

**Alternativen verworfen:**

- *Volle Liste immer.* Risiko sehr großer Responses (mehrere KB) ohne Mehrwert.

## Risks / Trade-offs

- **Risiko: Push-Sturm bei Bulk-Spielplan-Import.** Wenn der Vorstand 30 Heimspiele via CSV-Import anlegt (Feature noch nicht da, aber denkbar), würden 30 × ~5 betroffene Helfer = 150 Push-Notifications fliegen. Mitigation: Auto-Regen für CSV-Bulk-Pfade explizit ausschalten und am Ende einmal über die ganze Saison laufen lassen. Wird in tasks.md als Folgearbeit markiert.
- **Risiko: `is_custom=0` als Default für Bestand löst Welle aus.** Beim ersten Game-Update nach Deploy regeneriert das System die drei Tage und entfernt ggf. manuell-editierte Bestandsslots. Mitigation: Migration 037 enthält keine Daten-Updates; Vorstand wird per CLAUDE.md / Release-Notes gebeten, vor Deploy zu identifizieren, was geschützt werden soll.
- **Trade-off: Helfer können zwischen „Eingetragen" und „Ausgetragen" hin- und herwippen.** Bei iterativen Spielplan-Edits ändert sich die Adjacency wiederholt, der Helfer bekommt mehrere Pushs. Akzeptiert — der Vorstand sollte Spielpläne in einem Schwung pflegen.
- **Trade-off: Generische Events werden vom Auto-Regen ignoriert (für ihre eigenen Slots), erscheinen aber in `allGameTimes` der Same-Day-Logik.** Heißt: ein generisches Event an Tag N kann die Klassifikation der Slots eines Heimspiels am selben Tag beeinflussen (`isBetweenGames` für einen Slot zwischen Heim- und generic-Zeit). Das ist Status quo aus `loadSameDayContext` (Zeile 286 fragt ohne `is_home`-Filter) — wir ändern es nicht.

## Open Questions

- **Soll der Preview-Endpoint `POST /api/admin/games/preview-duties` schon in Phase 1 mitkommen, oder erst nach erstem Praxistest?** → Default: erst Phase 2. Wizard zeigt zunächst „Dienstplan wird beim Speichern automatisch erzeugt" als Info-Hinweis.
- **Wie heißt die Notification-Variante für „Slot wurde reduziert" vs. „Slot wurde gelöscht"?** Vorschlag: ein einheitlicher Titel „Dienst angepasst", Body unterscheidet („… wurde aufgrund einer Spielplanänderung entfernt." / „… wurde zur Variante {neue Variante} geändert. Bitte überprüfe deinen Dienstplan."). Final wird in Implementation festgezurrt.
