## ADDED Requirements

### Requirement: Meine-Filter-Toggle
Die Seite SHALL einen Toggle „Alle | Meine" oben rechts neben der `<h1>` anzeigen. Im Modus „Meine" werden nur Spiele angezeigt, bei denen der eingeloggte Nutzer mindestens einen Eintrag hat (biete oder suche). Der Filter ist für alle Rollen sichtbar und aktiv.

#### Scenario: Standard-Ansicht zeigt alle Spiele
- **WHEN** Nutzer die Seite öffnet
- **THEN** Toggle steht auf „Alle" und alle zukünftigen Spiele werden angezeigt

#### Scenario: Wechsel zu „Meine"
- **WHEN** Nutzer auf „Meine" klickt
- **THEN** es werden nur noch Spiele angezeigt, bei denen `isOwn === true` auf mindestens einem Eintrag steht

#### Scenario: Keine eigenen Einträge
- **WHEN** Nutzer auf „Meine" klickt und keine eigenen Einträge hat
- **THEN** alle Tab-Listen sind leer; passende Leer-Meldung wird angezeigt

### Requirement: Tab-Counts spiegeln den aktiven Filter
Die Spiel-Counts in den Event-Typ-Tabs (Auswärtsspiele / Heimspiele / Events) SHALL die Anzahl der im aktuell aktiven Filter sichtbaren Spiele zeigen.

#### Scenario: Tab-Count im Meine-Modus
- **WHEN** Filter auf „Meine" steht und Nutzer hat 2 eigene Auswärtsspiele
- **THEN** Tab „Auswärtsspiele" zeigt `(2)` statt der Gesamtzahl

#### Scenario: Tab-Count im Alle-Modus unverändert
- **WHEN** Filter auf „Alle" steht
- **THEN** Tab-Counts zeigen die Gesamtzahl aller zukünftigen Spiele (bestehende Logik)
