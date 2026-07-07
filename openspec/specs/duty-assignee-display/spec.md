# duty-assignee-display Specification

## Purpose

Diese Spezifikation beschreibt die Capability `duty-assignee-display`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Assignee-Namen im Duty-Slot sichtbar
Das System SHALL für jeden Duty-Slot die Namen der eingetragenen Personen anzeigen. Diese Information ist für alle authentifizierten Nutzer sichtbar, die den Slot sehen können.

#### Scenario: Namen werden unter dem Slot angezeigt
- **WHEN** ein eingeloggter Nutzer die Dienstbörse oder eine Event-Detail-Seite öffnet
- **THEN** werden unter jedem Slot die Namen der eingetragenen Personen angezeigt

#### Scenario: Slot ohne Assignees
- **WHEN** ein Slot noch keine Einträge hat
- **THEN** wird keine Assignee-Zeile angezeigt (der Slot erscheint wie bisher)

#### Scenario: Slot vollständig besetzt
- **WHEN** alle Plätze eines Slots belegt sind
- **THEN** werden alle Namen angezeigt, auch wenn keine Vakanzen mehr vorhanden sind

### Requirement: Profilbild im Assignee-Eintrag

Das System SHALL das Profilbild einer Person als Avatar neben dem Namen anzeigen, sofern die Person `photo_visible` freigegeben hat. In der Dienstbörsen-**Liste** (`GET /api/duty-board`) SHALL das Profilbild jedoch NICHT inline pro Assignee ausgeliefert werden; `photo_url` wird stattdessen bei Bedarf (beim Öffnen eines Slots/Assignees) über einen On-Demand-Pfad nachgeladen. Die sichtbaren **Namen** bleiben inline Teil der Board-Antwort.

#### Scenario: Namen inline, Avatar on-demand

- **WHEN** ein eingeloggter Nutzer die Dienstbörse öffnet
- **THEN** enthält jeder Slot die Namen der eingetragenen Personen inline
- **AND** die Board-Antwort enthält KEIN `photo_url` pro Assignee

#### Scenario: Profilbild sichtbar freigegeben (on-demand)

- **WHEN** ein Nutzer einen Slot/Assignee öffnet und die Person `photo_visible = true` gesetzt und ein Bild hinterlegt hat
- **THEN** wird das Bild als kleiner Avatar neben dem Namen nachgeladen und angezeigt

#### Scenario: Profilbild nicht freigegeben

- **WHEN** eine Person `photo_visible = false` gesetzt hat oder kein Bild hinterlegt ist
- **THEN** wird kein Avatar angezeigt; nur der Name erscheint

### Requirement: Kontaktdaten-Tooltip
Das System SHALL einen Tooltip pro Assignee bereitstellen, der auf Desktop per Hover und auf Mobile per Tap geöffnet wird. Der Tooltip zeigt Kontaktdaten gemäß den individuellen Freigaben der Person.

#### Scenario: Tooltip mit Telefonnummer(n)
- **WHEN** ein Nutzer den Tooltip einer Person öffnet, die `phones_visible = true` gesetzt hat
- **THEN** werden alle hinterlegten Telefonnummern mit Label im Tooltip angezeigt

#### Scenario: Tooltip ohne Telefonnummer
- **WHEN** eine Person `phones_visible = false` gesetzt hat oder keine Telefonnummern hinterlegt hat
- **THEN** erscheint im Tooltip kein Telefon-Abschnitt

#### Scenario: Tooltip mit Adresse
- **WHEN** ein Nutzer den Tooltip einer Person öffnet, die `address_visible = true` gesetzt hat
- **THEN** wird die vollständige Adresse (Straße, PLZ, Ort) im Tooltip angezeigt

#### Scenario: Tooltip ohne Adresse
- **WHEN** eine Person `address_visible = false` gesetzt hat oder keine Adresse hinterlegt hat
- **THEN** erscheint im Tooltip kein Adress-Abschnitt

#### Scenario: Tooltip ohne freigegebene Daten
- **WHEN** eine Person weder Telefon noch Adresse freigegeben hat
- **THEN** zeigt der Tooltip nur den Namen der Person

#### Scenario: Tooltip auf Mobile schließen
- **WHEN** ein Nutzer auf Mobile einen Tooltip geöffnet hat und außerhalb tippt
- **THEN** schließt sich der Tooltip

