# Spec Delta

## MODIFIED Requirements

### Requirement: Tabellen-Seiten zeigen Card-Layout auf Mobile
Alle 5 Tabellen-Seiten (AdminUsersPage, AdminTeamsPage, MembersPage, AdminDutyTypesPage, DutySlotsPage) SHALL auf Viewports unter 640px anstelle der `<table>`-Struktur ein Card-basiertes Layout anzeigen. Jede Tabellenzeile MUSS als eigenständige Card erscheinen.

#### Scenario: Card-Layout auf Mobile
- **WHEN** der Viewport unter 640px ist
- **THEN** sind die `<table>`-Elemente ausgeblendet
- **THEN** ist eine Liste von Cards sichtbar, eine Card pro Datensatz

#### Scenario: Tabellen-Layout auf Desktop
- **WHEN** der Viewport 640px oder breiter ist
- **THEN** sind die `<table>`-Elemente sichtbar
- **THEN** sind die Card-Listen ausgeblendet

### Requirement: Cards zeigen primäre Informationen
Jede Card SHALL den Namen / Primärwert des Datensatzes als Hauptzeile und die wichtigsten Sekundärfelder als zweite Zeile anzeigen. Pro Seite gelten folgende Prioritäten:

- **AdminUsersPage**: Name (groß) + E-Mail · Rolle-Badge
- **AdminTeamsPage**: Teamname (groß) + Altersklasse · Status-Badge
- **MembersPage**: Nachname, Vorname (groß) + Position · Status-Badge
- **AdminDutyTypesPage**: Name (groß) + Stundenwert · Geldersatz (wenn vorhanden)
- **DutySlotsPage**: Event-Name (groß) + Datum · Diensttyp · Belegungs-Anzeige

#### Scenario: Primärfeld immer sichtbar
- **WHEN** eine Card angezeigt wird
- **THEN** ist der Name / Primärwert in fetter Schrift dargestellt

#### Scenario: Sekundärfelder als kompakte Zeile
- **WHEN** eine Card angezeigt wird
- **THEN** erscheinen Sekundärfelder in kleinerer, gedimmter Schrift in einer zweiten Zeile
