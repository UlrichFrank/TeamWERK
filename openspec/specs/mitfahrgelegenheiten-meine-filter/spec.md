### Requirement: Meine-Filter-Toggle
Die Seite SHALL einen Toggle „Team | Meine" oben rechts neben der `<h1>` anzeigen. Im Modus „Team" werden alle Spiele der eigenen Mannschaft(en) angezeigt (Standardansicht). Im Modus „Meine" werden nur Spiele angezeigt, bei denen der eingeloggte Nutzer mindestens einen Eintrag hat (biete oder suche) oder in einer Paarung beteiligt ist. Der Filter ist für alle Rollen sichtbar und aktiv.

#### Scenario: Standard-Ansicht zeigt Team-Spiele
- **WHEN** Nutzer die Seite öffnet
- **THEN** Toggle steht auf „Team" und alle Spiele der eigenen Mannschaft(en) werden angezeigt

#### Scenario: Wechsel zu „Meine"
- **WHEN** Nutzer auf „Meine" klickt
- **THEN** es werden nur noch Spiele angezeigt, bei denen `isOwn === true` auf mindestens einem Eintrag steht oder der Nutzer in einer Paarung (`bieteIsOwn || sucheIsOwn`) beteiligt ist

#### Scenario: Keine eigenen Einträge
- **WHEN** Nutzer auf „Meine" klickt und keine eigenen Einträge hat
- **THEN** alle Tab-Listen sind leer; passende Leer-Meldung wird angezeigt

### Requirement: Tab-Counts spiegeln den aktiven Filter
Die Spiel-Counts in den Event-Typ-Tabs (Auswärtsspiele / Heimspiele / Events) SHALL die Anzahl der im aktuell aktiven Filter sichtbaren Spiele zeigen.

#### Scenario: Tab-Count im Meine-Modus
- **WHEN** Filter auf „Meine" steht und Nutzer hat 2 eigene Auswärtsspiele
- **THEN** Tab „Auswärtsspiele" zeigt `(2)` statt der Gesamtzahl

#### Scenario: Tab-Count im Team-Modus
- **WHEN** Filter auf „Team" steht
- **THEN** Tab-Counts zeigen die Gesamtzahl aller Spiele des eigenen Teams
