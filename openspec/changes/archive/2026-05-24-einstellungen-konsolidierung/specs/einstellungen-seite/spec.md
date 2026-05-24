## ADDED Requirements

### Requirement: Einstellungen-Seite mit Tab-Navigation
Die Seite `/admin/einstellungen` enthält drei Tabs: „Verein", „Saisons", „Altersklassen". Zugänglich nur für Rollen admin und vorstand.

#### Scenario: Direktaufruf ohne Tab-Parameter
- **WHEN** `/admin/einstellungen` ohne `?tab=` aufgerufen wird
- **THEN** ist der Tab „Verein" standardmäßig aktiv

#### Scenario: Direktaufruf mit Tab-Parameter
- **WHEN** `/admin/einstellungen?tab=saisons` aufgerufen wird
- **THEN** ist der Tab „Saisons" aktiv und dessen Daten werden geladen

#### Scenario: Redirect von alter Route
- **WHEN** `/admin/verein` aufgerufen wird
- **THEN** wird auf `/admin/einstellungen?tab=verein` weitergeleitet

#### Scenario: Redirect Saisons
- **WHEN** `/admin/saisons` aufgerufen wird
- **THEN** wird auf `/admin/einstellungen?tab=saisons` weitergeleitet

#### Scenario: Redirect Altersklassen
- **WHEN** `/admin/altersklassen` aufgerufen wird
- **THEN** wird auf `/admin/einstellungen?tab=altersklassen` weitergeleitet

### Requirement: Saisons-Tab — Modal-Muster
Im Saisons-Tab gibt es einen „Saison anlegen"-Button oben rechts (wie bei Diensttypen). Jede Saison-Zeile hat einen „Bearbeiten"-Button. Beide öffnen modale Dialoge.

#### Scenario: Saison anlegen
- **WHEN** der Admin auf „Saison anlegen" klickt
- **THEN** öffnet sich ein modaler Dialog mit Preset-Dropdown, Name, Startdatum, Enddatum
- **AND** nach Bestätigung wird die Saison angelegt und die Liste aktualisiert

#### Scenario: Saison bearbeiten
- **WHEN** der Admin auf „Bearbeiten" bei einer Saison klickt
- **THEN** öffnet sich ein modaler Dialog mit Name, Startdatum, Enddatum vorbefüllt
- **AND** bei aktiver Saison erscheint ein Hinweistext im Modal

#### Scenario: Nav-Eintrag
- **WHEN** ein Admin/Vorstand die Sidebar sieht
- **THEN** gibt es unter „Kaderplanung" (oder einem anderen Abschnitt) nur einen Eintrag „Einstellungen" statt drei separaten Einträgen
