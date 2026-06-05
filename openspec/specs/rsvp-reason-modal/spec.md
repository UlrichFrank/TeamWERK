# rsvp-reason-modal Specification

## Purpose
TBD - created by archiving change rsvp-reason-modal. Update Purpose after archive.
## Requirements
### Requirement: Begründungsmodal bei Absage und Vielleicht
Das System SHALL beim Klick auf „Absagen" oder „Vielleicht" ein Modal-Dialog öffnen, das den Nutzer zur Eingabe einer Begründung auffordert, bevor die RSVP-Antwort übermittelt wird. Das Textfeld MUSS nicht leer sein, um den OK-Button zu aktivieren.

#### Scenario: Modal öffnet sich bei Klick auf Absagen
- **WHEN** ein Nutzer auf den „Absagen"-Button eines aktiven Termins klickt
- **THEN** öffnet sich ein Modal mit einem Textfeld für die Begründung und den Buttons „OK" (disabled) und „Abbrechen"

#### Scenario: Modal öffnet sich bei Klick auf Vielleicht
- **WHEN** ein Nutzer auf den „Vielleicht"-Button eines aktiven Termins klickt
- **THEN** öffnet sich ein Modal mit einem Textfeld für die Begründung und den Buttons „OK" (disabled) und „Abbrechen"

#### Scenario: OK wird aktiv nach Texteingabe
- **WHEN** der Nutzer im Modal-Textfeld mindestens ein Zeichen eingibt
- **THEN** wird der OK-Button aktiviert

### Requirement: RSVP-Übermittlung mit Begründung
Das System SHALL die RSVP-Antwort erst nach Bestätigung im Modal an das Backend übermitteln. Die eingegebene Begründung MUSS als `reason`-Feld im Request-Body enthalten sein.

#### Scenario: Erfolgreiches Absenden nach Begründung
- **WHEN** der Nutzer eine Begründung eingibt und auf OK klickt
- **THEN** wird `POST /api/training-sessions/{id}/respond` oder `POST /api/games/{id}/respond` mit `{ status, reason }` aufgerufen, das Modal geschlossen und der lokale RSVP-Status aktualisiert

#### Scenario: Abbrechen erhält vorherigen Zustand
- **WHEN** der Nutzer im Modal auf „Abbrechen" klickt
- **THEN** wird kein API-Aufruf gemacht, das Modal geschlossen und der vorherige RSVP-Status bleibt erhalten

### Requirement: Eltern-Kind-RSVP im Modal
Das System SHALL das Begründungsmodal auch für Eltern-Kind-RSVPs verwenden. Das Modal MUSS den Namen des Kindes im Titel anzeigen.

#### Scenario: Modal für Kind-RSVP
- **WHEN** ein Elternteil auf „Absagen" oder „Vielleicht" für ein Kind klickt
- **THEN** öffnet sich das Modal mit dem Kindnamen im Titel und verhält sich identisch zum eigenen RSVP-Modal

### Requirement: Entfernen der Inline-Begründungsfelder
Das System SHALL keine Inline-Textfelder für Begründungen unterhalb der RSVP-Buttons anzeigen.

#### Scenario: Kein Inline-Input sichtbar
- **WHEN** ein Nutzer die Terminliste öffnet
- **THEN** sind keine Textfelder unterhalb der RSVP-Buttons sichtbar — die Begründungseingabe erfolgt ausschließlich über das Modal

