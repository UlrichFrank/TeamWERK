## Why

Der Monatskalender (`KalenderPage`) ist auf Mobile (< 640px) nicht nutzbar: Bei 375px Viewport-Breite sind die 7 Spalten des Grids nur ~49px breit. Event-Texte werden abgeschnitten, Trainings sind kaum erkennbar, und der „+"-Button zum Anlegen neuer Events funktioniert nur per Hover — auf Touch-Geräten also gar nicht.

## What Changes

- `KalenderPage` erhält auf Mobile (`sm:hidden`) eine **Agenda-View**: scrollbare chronologische Liste aller Spiele und Trainings des aktuell gewählten Monats, gruppiert nach Datum
- Der bestehende 7-Spalten-Desktop-Kalender bleibt unverändert (`hidden sm:block`)
- Die Monatswechsel-Navigation (◀ / ▶) ist auf beiden Views identisch nutzbar
- Admins und Trainer bekommen auf Mobile einen **Floating Action Button (FAB)** zum Anlegen neuer Events — ersetzt den nicht-funktionierenden `group-hover`-Button der Grid-Zellen
- Tap auf ein Event navigiert zur jeweiligen Detailseite (`SpieltagDetailPage` / `TrainingsDetailPage`) — identisches Verhalten wie im Desktop-Grid

## Capabilities

### New Capabilities

- `kalender-agenda-view`: Mobile Agenda-Darstellung des Monatskalenders mit FAB für berechtigte Nutzer

### Modified Capabilities

*(keine — Desktop-Kalender und Datenstruktur bleiben unverändert)*

## Impact

- `web/src/pages/KalenderPage.tsx` — neue Agenda-View-Sektion, FAB-Logik
- Keine Backend-Änderungen (dieselben API-Endpunkte, dieselben Daten)
- Keine neuen Abhängigkeiten
