## Context

Die `games`-Tabelle modelliert alle Kalendereinträge (Heim-, Auswärtsspiele, generische Events). Bisher hat jedes Event genau ein `date`-Feld. Turniere und Trainingslager dauern typischerweise mehrere Tage; bisher musste pro Tag ein separates Event angelegt werden.

Die Absenz-Funktion löst dasselbe Problem bereits mit `start_date`/`end_date` und Banner-Rendering im Kalender. Für Events wird bewusst ein anderer Rendering-Ansatz gewählt (Pill statt Banner), da Events anklickbar bleiben müssen und die bestehende Pill-Darstellung beibehalten werden soll.

## Goals / Non-Goals

**Goals:**
- Ein einziges `end_date`-Feld zu `games` hinzufügen (nullable)
- Mehrtägige Events in jeder betroffenen Tageszelle als Pill anzeigen
- Backend und Frontend minimal anpassen, alle bestehenden Features (RSVP, Slots, Mitfahrten) unverändert lassen

**Non-Goals:**
- Neuer `event_type` (kein `turnier`, kein `trainingslager`)
- Per-Tag-RSVP oder tagesweise Teilnahme
- Einzelspiele innerhalb eines Turniers erfassen
- Abweichende Darstellung für verschiedene Typen mehrtägiger Events

## Decisions

### end_date statt Dauer in Tagen

`end_date DATE` (nullable) statt `duration_days INT`. Begründung: Explizite Datumsangabe ist fehlertoleranter, verständlicher in SQL-Queries und konsistent mit dem `start_date`/`end_date`-Muster der Absenzen.

### Kein neuer event_type

Turniere nutzen `heim`/`auswärts` mit `end_date`. Trainingslager nutzen `generisch` mit `end_date`. Die Semantik des event_type bleibt unverändert. Kein zusätzlicher CHECK-Constraint, keine Frontend-Fallunterscheidung nötig.

### Pill-Wiederholung statt Banner

Events mit `end_date` erscheinen als identische Pill in jeder Tageszelle des Bereichs — nicht als über Tage gespannter Banner (wie Absenzen). Begründung: Pills sind direkt anklickbar, zeigen alle Event-Infos im gewohnten Format und erfordern keine Layout-Umstrukturierung der Kalender-Grid.

### Enddatum nur im Wizard für `generisch`

Im Event-Wizard-Formular wird das Enddatum-Feld nur bei `event_type === 'generisch'` angezeigt. Für `heim`/`auswärts`-Events kann `end_date` über das Edit-Modal gesetzt werden, falls ein Turnier mehrere Tage dauert. Dies vermeidet eine zu frühe UX-Komplexität im Wizard.

### Filterlogik im Frontend

Die Zuordnung von Events zu Tageszellen erfolgt im Frontend. Aktuell: `games.filter(g => g.date.slice(0,10) === dateStr)`. Neu: Event auch anzeigen wenn `end_date` gesetzt und `date <= dateStr <= end_date`.

## Risks / Trade-offs

**Pill-Duplikate bei langen Events** → Bei 7-Tage-Trainingslager erscheint dieselbe Pill 7x. Für typische Fälle (2-5 Tage) akzeptabel; wird bei Bedarf revisited.

**end_date ohne Validierung gegen date** → Backend muss sicherstellen, dass `end_date >= date`. Einfache Validierung in `CreateGame`/`UpdateGame`.

**Migration ist additive** → `end_date` ist nullable, alle bestehenden Rows bleiben unverändert. Rollback: Column droppen (SQLite erlaubt kein DROP COLUMN vor 3.35 — Migration muss als neue Tabelle rewritten werden falls Rollback nötig). Praktisch irrelevant da additive nullable Column.

## Migration Plan

1. Migration `0NN_games_end_date.up.sql`: `ALTER TABLE games ADD COLUMN end_date DATE`
2. Migration `0NN_games_end_date.down.sql`: keine sinnvolle Umkehrung (SQLite); leer lassen oder Tabelle neu erstellen
3. Deploy: `make deploy` führt migrate up automatisch aus
