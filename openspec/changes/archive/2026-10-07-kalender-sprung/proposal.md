# Proposal

## Why

Der Kalender-Dialog (`EventInfoModal`) führt mit „In Terminen öffnen“ und „In Diensten öffnen“ von einem Termin auf `/termine` bzw. `/dienste`. Den Rückweg gibt es nicht: Wer auf `/termine` oder `/dienste` einen Termin vor sich hat und ihn im Monatszusammenhang sehen will (was ist am selben Wochenende noch, wer richtet den Spieltag aus, Hinweis und Treffzeit pflegen), muss in den Kalender wechseln, den Monat von Hand suchen und den Termin anklicken.

## What Changes

- **`/termine`**: Jede Termin-Karte (Spiel/Event und Training) bekommt eine Icon-Aktion „Im Kalender öffnen“ (`CalendarDays`) rechts in der Zeile der Zusage-Zähler. Der Klick löst nicht den Karten-Klick (Detailseite) aus.
- **`/dienste`**: Jeder Dienst-Block, der an einem Termin hängt (`game_id` gesetzt), bekommt dieselbe Icon-Aktion im Kopf des Blocks rechts neben den Team-Namen. Blöcke ohne Termin (termin-lose Hand-Slots) bekommen keine Aktion.
- **`/kalender`**: Neuer Query-Parameter `focus=<game|training>-<id>`, kombiniert mit dem bestehenden `date=YYYY-MM-DD`. Der Kalender öffnet den Monat aus `date`, öffnet für den fokussierten Termin den Termin-Dialog (`EventInfoModal`) und hebt die Termin-Kachel im Monatsgitter kurz hervor. Danach wird `focus` aus der URL entfernt (Zurück-Navigation und Neuladen öffnen den Dialog nicht erneut).
- Ist der Termin nicht auffindbar (gelöscht, keine Sichtbarkeit), zeigt der Kalender den Hinweis „Dieser Termin ist nicht verfügbar“ und sonst den Monat normal.
- Kein Backend-Change: der Sprung nutzt bestehende Lese-Routen.

## Capabilities

### New Capabilities
- `kalender-sprung`: Aktion „Im Kalender öffnen“ auf den Termin-Karten von `/termine` und den Dienst-Blöcken von `/dienste`, inklusive Ziel-URL und Sichtbarkeitsregeln der Aktion.

### Modified Capabilities
- `kalender-date-param`: Der Kalender akzeptiert zusätzlich `focus=<game|training>-<id>` und öffnet den fokussierten Termin; Verhalten bei unbekanntem Termin und ungültigem Format.

## Impact

- Frontend: `web/src/pages/TerminePage.tsx` (Karten), `web/src/pages/DutyPage.tsx` (Block-Kopf), `web/src/pages/KalenderPage.tsx` (Fokus-Auswertung, Dialog öffnen, Hervorhebung), ein kleiner Helfer für die Ziel-URL (z. B. `web/src/lib/calendarLink.ts`).
- Backend: keine Änderung. Gelesen wird über `GET /api/games/{id}` bzw. `GET /api/training-sessions/{id}`, falls der Termin nicht in den bereits geladenen Monatsdaten steckt.
- Berechtigungen unverändert: die Aktion erscheint nur an Terminen, die der Nutzer ohnehin sieht; der Kalender zeigt nur, was die bestehenden Lese-Routen ihm liefern.
- Tests: Vitest für TerminePage, DutyPage und KalenderPage (Fokus-Deep-Link).
