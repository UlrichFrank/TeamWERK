## Why

Die Anwesenheits-Übersicht im Profil (eigenes Profil und Kind-Seite `/profil/kind/:id`)
zeigt je Trainings und Spiele eine einzige „Quote" `anwesend / (anwesend + fehlt)`.
Entschuldigte Termine fallen dabei aus dem Nenner — ein Spieler mit 5 Anwesenheiten und
15 Entschuldigungen steht mit 100 % da, genauso wie einer, der nie gefehlt hat. Anwesend
und Entschuldigt werden also faktisch gleich bewertet. Das ist nicht die Aussage, die
Spieler, Eltern und Trainer aus der Zahl lesen.

## What Changes

- Die Spieler-/Eltern-Sicht (`AttendanceStatsView`) ersetzt die eine Quote durch **drei
  Anteile am Gesamt** der gezählten Termine: `anwesend X %`, `entschuldigt Y %`,
  `fehlt Z %` (je mit Anzahl), für Trainings und Spiele getrennt. Die drei Anteile ergeben
  zusammen 100 % (Rundungsdifferenz ausgenommen) und passen zum bestehenden Balken.
- Backend unverändert: `GET /api/members/{id}/attendance-stats` liefert die drei Zähler
  schon heute.

## Nicht-Ziele

- Die Trainer-Sicht `/team/:id/anwesenheit` behält ihre kompakte Quote-Spalte (anderer
  Zweck: Vergleich vieler Spieler in einer Tabelle). Eine Angleichung wäre ein eigener
  Change.

## Capabilities

### Modified Capabilities

- `attendance-statistics`: Spieler-/Eltern-Sicht zeigt drei Anteile statt einer Quote.

## Impact

- `web/src/components/AttendanceStatsView.tsx` (+ Vitest).

## Test-Anforderungen

Keine neue Route. Vitest: `AttendanceStatsView` mit 5 anwesend / 3 entschuldigt / 2 fehlt
zeigt `50 %`, `30 %`, `20 %` und **keinen** Text „Quote"; ohne gezählte Termine erscheinen
keine Prozentwerte.
