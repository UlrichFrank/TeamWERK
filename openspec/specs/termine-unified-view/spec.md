### Requirement: Chronologisch gemischte Termine-Liste

Die `/termine`-Seite SHALL eine chronologisch sortierte Liste aus Trainings und Spielen
des eigenen Teams anzeigen, zeitlich gefiltert auf zukünftige Termine (+ optional vergangene).

#### Scenario: Spieler sieht eigene Trainings und Spiele gemischt
- **WHEN** ein User mit Rolle `spieler` die `/termine`-Seite aufruft
- **THEN** werden Trainings und Spiele seines Teams chronologisch gemischt angezeigt
- **THEN** sind vergangene Termine standardmäßig ausgeblendet (Checkbox „Vergangene anzeigen")

#### Scenario: Leere Liste wenn keine Termine
- **WHEN** ein User keine Team-Zugehörigkeit hat oder keine Termine im Zeitraum existieren
- **THEN** wird ein leerer Zustand mit erklärendem Text angezeigt

#### Scenario: Termin-Typ ist visuell erkennbar
- **WHEN** Trainings und Spiele in der Liste angezeigt werden
- **THEN** ist der Typ jedes Termins durch ein Icon unterscheidbar (Dumbbell für Training, Home/MapPin für Spiel)

---

### Requirement: RSVP-Buttons in der Termine-Liste

Jeder aktive Termin in der Liste SHALL für Spieler und Eltern RSVP-Buttons (Zusagen/Vielleicht/Absagen)
mit optionalem Grund-Textfeld direkt in der Karte anzeigen.

#### Scenario: RSVP-Buttons sichtbar für Spieler
- **WHEN** ein User mit Rolle `spieler` oder `elternteil` die `/termine`-Seite aufruft
- **THEN** sind RSVP-Buttons für jeden aktiven Termin sichtbar

#### Scenario: RSVP-Buttons nicht sichtbar für Trainer
- **WHEN** ein User mit Trainer-Funktion die `/termine`-Seite aufruft
- **THEN** werden keine RSVP-Buttons angezeigt (Trainer klicken auf den Termin für die Übersicht)

#### Scenario: Aktiver RSVP-Status visuell hervorgehoben
- **WHEN** ein User bereits eine Rückmeldung abgegeben hat
- **THEN** ist der entsprechende Button visuell aktiv (farblich hervorgehoben)

#### Scenario: Abgesagter Termin zeigt keine RSVP-Buttons
- **WHEN** ein Training den Status `cancelled` hat
- **THEN** werden keine RSVP-Buttons angezeigt, und der Termin ist als abgesagt markiert

---

### Requirement: Trainer-Link zur Detailübersicht

Für Trainer SHALL jede Termin-Karte anklickbar sein und zur Detailseite führen.

#### Scenario: Trainer klickt auf Trainingstermin
- **WHEN** ein User mit Trainer-Funktion auf eine Trainingskarte klickt
- **THEN** wird zur Route `/termine/training/:id` navigiert

#### Scenario: Trainer klickt auf Spieltermin
- **WHEN** ein User mit Trainer-Funktion auf eine Spielkarte klickt
- **THEN** wird zur Route `/termine/spiel/:id` navigiert

---

### Requirement: Detailseite zeigt Rückmeldungs-Tabelle

`/termine/training/:id` und `/termine/spiel/:id` SHALL für Trainer eine Übersichtstabelle
aller Teammitglieder mit RSVP-Status und Grund anzeigen.

#### Scenario: Trainer sieht alle Mitglieder mit Status
- **WHEN** ein Trainer `/termine/training/:id` oder `/termine/spiel/:id` aufruft
- **THEN** wird eine Tabelle mit allen Kader-Mitgliedern, RSVP-Status und Grund angezeigt
- **THEN** sind Mitglieder ohne Rückmeldung als „–" (kein Status) gelistet

#### Scenario: Anwesenheits-Tracking nur für Trainings
- **WHEN** ein Trainer `/termine/training/:id` für einen vergangenen Termin aufruft
- **THEN** ist eine Anwesenheits-Spalte mit Checkboxen sichtbar
- **WHEN** ein Trainer `/termine/spiel/:id` aufruft
- **THEN** gibt es keine Anwesenheits-Spalte

---

### Requirement: Navigation „Termine" ersetzt „Trainings"

Der Nav-Eintrag „Trainings" SHALL durch „Termine" ersetzt werden.
Die alten Routes `/trainings` und `/trainings/:id` SHALL auf `/termine` redirecten.

#### Scenario: Alter Trainings-Link wird weitergeleitet
- **WHEN** ein User `/trainings` aufruft
- **THEN** wird er zu `/termine` weitergeleitet (HTTP 301 oder client-seitiger Redirect)

#### Scenario: Nav zeigt „Termine" für alle Rollen
- **WHEN** ein User eingeloggt ist
- **THEN** ist „Termine" in der Navigation sichtbar (für alle Rollen gleich wie vorher „Trainings")
