## ADDED Requirements

### Requirement: Modus-Toggle auf KalenderPage

Die KalenderPage SHALL einen segmentierten Toggle `[Dienste | Termine]` oben rechts im Header anzeigen. Der aktive Modus wird mit `bg-brand-yellow text-brand-black font-medium` hervorgehoben, der inaktive mit `text-brand-text-muted hover:bg-brand-border-subtle`.

#### Scenario: Standard-Modus ist Dienste

- **WHEN** ein User `/kalender` aufruft
- **THEN** ist der Toggle auf „Dienste" voreingestellt

#### Scenario: Toggle wechselt Modus

- **WHEN** ein User auf „Termine" klickt
- **THEN** wechselt der aktive Modus zu „Termine" ohne Seitenneuladen

#### Scenario: Visueller Zustand des Toggles

- **WHEN** der Modus „Dienste" aktiv ist
- **THEN** ist „Dienste"-Button gelb hinterlegt; „Termine"-Button ist nicht hervorgehoben

---

### Requirement: Klick-Verhalten im Dienste-Modus

Im Dienste-Modus SHALL ein Klick auf einen Spieltag zu `/kalender/:id` navigieren. Trainings SHALL im Dienste-Modus nicht klickbar sein (kein onClick, kein Hover-Effekt, `cursor-default`).

#### Scenario: Spieltag-Klick im Dienste-Modus

- **WHEN** der Modus „Dienste" aktiv ist
- **WHEN** ein User auf einen Spieltag im Kalender klickt
- **THEN** navigiert die App zu `/kalender/:id`

#### Scenario: Training-Klick im Dienste-Modus

- **WHEN** der Modus „Dienste" aktiv ist
- **WHEN** ein User auf ein Training im Kalender klickt
- **THEN** passiert nichts (kein Navigation, kein Modal)

#### Scenario: Training-Darstellung im Dienste-Modus

- **WHEN** der Modus „Dienste" aktiv ist
- **THEN** haben Trainings keinen Hover-Hintergrund-Effekt im Kalender-Tile

---

### Requirement: Klick-Verhalten im Termine-Modus für berechtigte Rollen

Im Termine-Modus SHALL ein Klick auf einen Spieltag durch einen User mit Rolle admin, trainer, vorstand oder sportliche_leitung das `GameEditModal` öffnen. Ein Klick auf ein Training SHALL das `TrainingEditModal` öffnen.

#### Scenario: Spieltag-Klick Termine-Modus als Trainer

- **WHEN** der Modus „Termine" aktiv ist
- **WHEN** ein User mit Rolle trainer auf einen Spieltag klickt
- **THEN** öffnet sich das `GameEditModal` mit den Daten des Spieltags

#### Scenario: Training-Klick Termine-Modus als Trainer

- **WHEN** der Modus „Termine" aktiv ist
- **WHEN** ein User mit Rolle trainer auf ein Training klickt
- **THEN** öffnet sich das `TrainingEditModal` mit den Daten des Trainings

---

### Requirement: Klick-Verhalten im Termine-Modus für nicht-berechtigte Rollen

Im Termine-Modus SHALL ein Klick auf einen Spieltag oder ein Training durch einen User mit Rolle spieler oder elternteil das `EventInfoModal` (schreibgeschützt) öffnen.

#### Scenario: Spieltag-Klick Termine-Modus als Spieler

- **WHEN** der Modus „Termine" aktiv ist
- **WHEN** ein User mit Rolle spieler auf einen Spieltag klickt
- **THEN** öffnet sich das `EventInfoModal` mit den schreibgeschützten Details des Spieltags

#### Scenario: Training-Klick Termine-Modus als Elternteil

- **WHEN** der Modus „Termine" aktiv ist
- **WHEN** ein User mit Rolle elternteil auf ein Training klickt
- **THEN** öffnet sich das `EventInfoModal` mit den schreibgeschützten Details des Trainings
