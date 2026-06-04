## Why

Die Begründung für eine Absage oder ein „Vielleicht" wird zwar vom Backend entgegen genommen, aber auf der Termine-Seite nicht zuverlässig erfasst: Die bisherigen Inline-Inputs sind mit dem State nicht verdrahtet (`reasons` fehlt), sodass Begründungen nie übermittelt werden. Trainer haben dadurch keine Planungsgrundlage. Statt eines unübersichtlichen Inline-Felds soll ein Modal die Begründung abfragen – so bleibt sie verpflichtend und kann nicht versehentlich leer übermittelt werden.

## What Changes

- Entfernen der kaputten Inline-`<input>`-Felder unter den RSVP-Buttons in `TerminePage.tsx`
- Neues Modal erscheint, wenn der Nutzer auf **Vielleicht** oder **Absagen** klickt
- Das Modal enthält ein Textfeld für die Begründung (Pflichtfeld, OK ist disabled solange leer)
- Klick auf **Abbrechen** schließt das Modal ohne Zustandsänderung
- Klick auf **OK** sendet die RSVP-Antwort mit Begründung und schließt das Modal
- Gilt für eigene RSVPs und für Eltern-Kind-RSVPs
- Gilt für Training- und Spieltermine

## Capabilities

### New Capabilities

- `rsvp-reason-modal`: Modal-Dialog zur Eingabe einer Pflichtbegründung bei Absage/Vielleicht-RSVP

### Modified Capabilities

- `rsvp`: Das bestehende RSVP-Verhalten für Training und Spiele ändert sich: Begründung wird jetzt vor dem Absenden abgefragt (statt optionalem Inline-Input danach)

## Impact

- `web/src/pages/TerminePage.tsx`: Umbau der RSVP-Interaktion (State-Verwaltung, Button-Handler, Modal-Rendering)
- Keine Backend-Änderungen (API nimmt `reason` bereits entgegen)
- Keine neuen Abhängigkeiten
