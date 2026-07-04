## MODIFIED Requirements

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
