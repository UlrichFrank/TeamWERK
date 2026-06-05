## ADDED Requirements

### Requirement: EventInfoModal zeigt schreibgeschützte Details eines Kalendereintrags

Das `EventInfoModal` SHALL Spieltag- oder Trainingsdaten schreibgeschützt anzeigen. Es enthält keine Bearbeiten-Buttons. Es verwendet das Standard-Modal-Design (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`).

#### Scenario: Modal für Spieltag

- **WHEN** das `EventInfoModal` für einen Spieltag geöffnet wird
- **THEN** zeigt es: Event-Typ-Icon, Gegner, Datum, Uhrzeit, RSVP-Zahlen (confirmed/declined/maybe)

#### Scenario: Modal für Training

- **WHEN** das `EventInfoModal` für ein Training geöffnet wird
- **THEN** zeigt es: Training-Icon (Dumbbell), Titel, Datum, Startzeit–Endzeit, Ort, RSVP-Zahlen

#### Scenario: Kein Bearbeiten-Button

- **WHEN** das `EventInfoModal` für einen beliebigen Eintrag geöffnet wird
- **THEN** enthält es keinen Button zum Bearbeiten oder Löschen

---

### Requirement: EventInfoModal schließt sich via Escape oder Schließen-Button

Das Modal SHALL einen Schließen-Button (`<X>`-Icon) oben rechts haben und sich bei Escape-Taste schließen (via `useEscapeKey`-Hook).

#### Scenario: Schließen via Button

- **WHEN** ein User auf den `<X>`-Button klickt
- **THEN** schließt sich das Modal

#### Scenario: Schließen via Escape

- **WHEN** das Modal offen ist und der User Escape drückt
- **THEN** schließt sich das Modal

---

### Requirement: Kein eigener API-Call

Das `EventInfoModal` SHALL keine eigenen API-Aufrufe machen. Alle anzuzeigenden Daten (RSVP-Zahlen, Ort, Datum etc.) sind bereits im jeweiligen `Game`- oder `Training`-Objekt vorhanden.

#### Scenario: Öffnen ohne Netzwerkanfrage

- **WHEN** das `EventInfoModal` geöffnet wird
- **THEN** werden keine API-Calls ausgeführt
- **THEN** werden alle Daten aus den Props gerendert
