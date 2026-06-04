## ADDED Requirements

### Requirement: Spiel/Dienst-Toggle im Kalender-Header
Die KalenderPage SHALL einen Toggle „Spiel | Dienst" im Header anzeigen. Der Standard-Modus beim Laden der Seite ist „Dienst".

#### Scenario: Seite lädt im Dienst-Modus
- **WHEN** Nutzer die KalenderPage öffnet
- **THEN** ist der Toggle auf „Dienst" gesetzt und Spiel-Pill-Klicks navigieren zu `/kalender/{id}`

#### Scenario: Wechsel in den Spiel-Modus
- **WHEN** Nutzer auf „Spiel" im Toggle klickt
- **THEN** wechselt der Modus zu „Spiel" und Spiel-Pill-Klicks öffnen das GameModal

#### Scenario: Wechsel zurück in den Dienst-Modus
- **WHEN** Nutzer im Spiel-Modus auf „Dienst" klickt
- **THEN** wechselt der Modus zurück und Klicks navigieren wieder zu `/kalender/{id}`

### Requirement: Sonstiges-Filter im Spiel-Modus deaktiviert
Im Spiel-Modus SHALL der Sonstiges-Filter-Button visuell deaktiviert sein und nicht klickbar. Beim Aktivieren des Spiel-Modus SHALL ein aktiver Sonstiges-Filter automatisch entfernt werden.

#### Scenario: Sonstiges-Button im Spiel-Modus
- **WHEN** Spiel-Modus aktiv ist
- **THEN** hat der Sonstiges-Button `opacity-40 cursor-not-allowed` und ignoriert Klicks

#### Scenario: Automatische Deaktivierung beim Moduswechsel
- **WHEN** Sonstiges-Filter aktiv ist und Nutzer in Spiel-Modus wechselt
- **THEN** wird Sonstiges automatisch aus dem aktiven Filter-Set entfernt

### Requirement: GameModal für Admin und Trainer editierbar
Für Nutzer mit Rolle `admin` oder `trainer` SHALL das GameModal ein Formular mit den Feldern Datum, Uhrzeit, Gegner und Teams anzeigen. Speichern ruft `PUT /admin/games/{id}` auf.

#### Scenario: Admin öffnet GameModal im Spiel-Modus
- **WHEN** Admin auf ein Spiel-Pill klickt im Spiel-Modus
- **THEN** öffnet sich das GameModal mit einem Formular (Datum, Uhrzeit, Gegner, Teams)

#### Scenario: Erfolgreiches Speichern
- **WHEN** Admin Felder ändert und auf „Speichern" klickt
- **THEN** wird `PUT /admin/games/{id}` aufgerufen, das Modal schließt sich und der Kalender zeigt die aktualisierten Daten

#### Scenario: Fehlermeldung bei Speicherfehler
- **WHEN** der PUT-Request fehlschlägt
- **THEN** zeigt das Modal eine Fehlermeldung und bleibt geöffnet

### Requirement: GameModal für andere Rollen read-only
Für Nutzer ohne Admin- oder Trainer-Rolle SHALL das GameModal die Spieldaten (Datum, Uhrzeit, Gegner, Teams) als read-only anzeigen — ohne Formular, ohne Speichern-Button.

#### Scenario: Spieler öffnet GameModal im Spiel-Modus
- **WHEN** Spieler (Rolle `spieler` oder `elternteil`) auf ein Spiel-Pill klickt im Spiel-Modus
- **THEN** öffnet sich das GameModal mit read-only-Anzeige der Spieldaten und einem Schließen-Button

### Requirement: Abbrechen schließt das Modal ohne Änderungen
Das GameModal SHALL einen Abbrechen-/Schließen-Button haben, der das Modal ohne Speichern schließt.

#### Scenario: Schließen ohne Speichern
- **WHEN** Nutzer auf „Abbrechen" oder das X-Icon klickt
- **THEN** schließt sich das Modal ohne API-Aufruf
