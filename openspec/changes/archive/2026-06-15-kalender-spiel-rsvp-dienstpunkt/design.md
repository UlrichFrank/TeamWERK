## Context

`KalenderPage.tsx` rendert Spiel-Kacheln (heim, auswärts, generisch) in einem 7-Spalten-Monats-Kalender. Jede Zelle ist ein `@container`-Element mit Container-Queries (`@tile-sm` = 80 px, `@tile-md` = 120 px). Trainings zeigen bereits `confirmed_count`/`declined_count` mit `hidden @tile-sm:inline-flex`. Spiele zeigen den Dienst-Punkt inline in der Uhrzeitzeile, aber keine RSVP-Zahlen.

Alle nötigen Daten kommen bereits vom Backend: `confirmed_count`, `declined_count`, `slot_count`, `filled_count`, `total_count` sind im `Game`-Interface und in der API-Response vorhanden.

## Goals / Non-Goals

**Goals:**
- Spiel-Kacheln zeigen RSVP-Zähler analog zu Trainings
- Dienst-Punkt wird in die Teamname-Zeile (rechts) verschoben — sichtbarer, weniger gedrängt

**Non-Goals:**
- Kein Backend-Change
- Keine Änderung an Trainings-Kacheln
- Kein `maybe_count` anzeigen (zu wenig Platz, zu wenig Relevanz)
- Kein Redesign der Kalender-Zellen

## Decisions

**Dienst-Punkt in Teamname-Zeile:** Die erste Zeile (`flex items-center gap-1`) wird zu `flex items-center gap-1` mit `flex-1` auf dem Teamname-`<span>`. Der Punkt (`w-1.5 h-1.5 rounded-full`) hängt am Ende, versteckt via `hidden @tile-sm:block`. Bedingung bleibt `g.slot_count > 0`.

**RSVP in Uhrzeitzeile:** Nach der Uhrzeit kommen `✓{g.confirmed_count}` und `✗{g.declined_count}` mit `hidden @tile-sm:inline-flex items-center gap-0.5`, identisch zum Trainings-Pattern. Farben: `text-green-600` für Zusagen, `text-brand-danger` für Absagen.

**`rsvp_opt_out`-Games:** Zähler werden immer gezeigt — das Backend berechnet bereits korrekte Werte (Spieler ohne Antwort zählen als confirmed). Kein Sonderfall im Frontend nötig.

## Risks / Trade-offs

- Auf sehr schmalen Kacheln (Mobile < 80 px) bleibt alles unsichtbar wie bisher — bewusste Entscheidung, konsistent mit Training
- Bei Spielen ohne Dienst-Slots (`slot_count = 0`) kein Punkt → kein visuelles Rauschen
