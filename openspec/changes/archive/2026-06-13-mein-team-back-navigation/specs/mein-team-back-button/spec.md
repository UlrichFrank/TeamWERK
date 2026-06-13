## ADDED Requirements

### Requirement: Zurück-Button bei gefilterter Team-Ansicht
MeinTeamPage SHALL einen Zurück-Button anzeigen, wenn der Query-Parameter `team` in der URL gesetzt ist (`focusTeamId != null`). Ein Klick auf den Button MUSS `navigate(-1)` auslösen. Ist kein `?team`-Parameter gesetzt, DARF der Button nicht gerendert werden.

#### Scenario: Zurück-Button sichtbar bei ?team=X
- **WHEN** User MeinTeamPage mit URL `/mein-team?team=20` öffnet
- **THEN** erscheint oben links ein Button mit `ChevronLeft`-Icon und Text „Zurück"

#### Scenario: Klick navigiert zurück
- **WHEN** User auf den Zurück-Button klickt
- **THEN** wird `navigate(-1)` ausgelöst und der User gelangt zur vorherigen Seite

#### Scenario: Kein Zurück-Button ohne ?team-Parameter
- **WHEN** User MeinTeamPage mit URL `/mein-team` (ohne Query-Parameter) öffnet
- **THEN** wird kein Zurück-Button gerendert
