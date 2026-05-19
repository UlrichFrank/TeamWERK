## ADDED Requirements

### Requirement: Tabellen-Seiten zeigen Card-Layout auf Mobile
Alle 6 Tabellen-Seiten (AdminUsersPage, AdminTeamsPage, MembersPage, AdminDutyTypesPage, DutyAccountsPage, DutySlotsPage) SHALL auf Viewports unter 640px anstelle der `<table>`-Struktur ein Card-basiertes Layout anzeigen. Jede Tabellenzeile MUSS als eigenständige Card erscheinen.

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
- **DutyAccountsPage**: Name (groß) + Soll/Ist-Werte · Differenz-Badge
- **DutySlotsPage**: Event-Name (groß) + Datum · Diensttyp · Belegungs-Anzeige

#### Scenario: Primärfeld immer sichtbar
- **WHEN** eine Card angezeigt wird
- **THEN** ist der Name / Primärwert in fetter Schrift dargestellt

#### Scenario: Sekundärfelder als kompakte Zeile
- **WHEN** eine Card angezeigt wird
- **THEN** erscheinen Sekundärfelder in kleinerer, gedimmter Schrift in einer zweiten Zeile

### Requirement: ⋮-Aktionen-Dropdown auf Mobile
Cards MIT Aktionen (Bearbeiten, Löschen, Genehmigen, etc.) SHALL einen ⋮-Button in der rechten oberen Ecke anzeigen. Ein Klick MUSS ein Dropdown mit den verfügbaren Aktionen öffnen.

#### Scenario: ⋮-Button zeigt Dropdown
- **WHEN** der Nutzer auf den ⋮-Button einer Card tippt
- **THEN** öffnet sich ein Dropdown mit den Aktionen dieser Zeile

#### Scenario: Dropdown schließt nach Aktion
- **WHEN** der Nutzer eine Aktion im Dropdown wählt
- **THEN** wird die Aktion ausgeführt
- **THEN** schließt sich das Dropdown

#### Scenario: Dropdown schließt bei Klick außerhalb
- **WHEN** ein Dropdown geöffnet ist
- **WHEN** der Nutzer außerhalb des Dropdowns tippt
- **THEN** schließt sich das Dropdown

#### Scenario: Desktop behält Inline-Buttons
- **WHEN** der Viewport 640px oder breiter ist
- **THEN** werden Aktionen weiterhin als Inline-Buttons in der Tabellenzeile angezeigt

### Requirement: Edit-Modal für Inline-Edit-Formulare auf Mobile
Seiten mit Inline-Edit-Formularen (AdminDutyTypesPage mit 5 Feldern) SHALL auf Mobile die „Bearbeiten"-Aktion in einem Modal öffnen. Das Modal MUSS alle bearbeitbaren Felder enthalten und Speichern/Abbrechen-Buttons bereitstellen.

#### Scenario: Bearbeiten öffnet Modal auf Mobile
- **WHEN** der Nutzer auf Mobile im ⋮-Dropdown auf „Bearbeiten" tippt
- **THEN** öffnet sich ein Modal mit dem Bearbeitungsformular
- **THEN** sind alle Felder (Name, Stunden, Geldersatz, Anker, Versatz) ausfüllbar

#### Scenario: Speichern schließt Modal
- **WHEN** der Nutzer auf „Speichern" im Modal tippt
- **THEN** werden die Änderungen gespeichert
- **THEN** schließt sich das Modal
- **THEN** zeigt die Card die aktualisierten Werte

#### Scenario: Abbrechen schließt Modal ohne Änderung
- **WHEN** der Nutzer auf „Abbrechen" im Modal tippt
- **THEN** schließt sich das Modal ohne Änderungen

### Requirement: DutySlotsPage bleibt expandierbar auf Mobile
Die expandierbaren Zuteilungs-Details in DutySlotsPage SHALL auch auf Mobile als Accordion funktionieren. Die Zuteilungen werden als flache Liste (kein verschachteltes Card-Layout) unterhalb der Slot-Card eingeblendet.

#### Scenario: Zuteilungen expandieren auf Mobile
- **WHEN** der Nutzer auf „Zuteilungen anzeigen" tippt
- **THEN** erscheinen die Zuteilungen als flache Liste unterhalb der Card
- **THEN** zeigt jeder Eintrag: Name · Status-Badge · Aktions-Button (wenn pending)

#### Scenario: Schließen der Zuteilungen
- **WHEN** die Zuteilungen ausgeklappt sind
- **WHEN** der Nutzer auf „schließen" tippt
- **THEN** werden die Zuteilungen wieder ausgeblendet

### Requirement: Touch-freundliche Tap-Targets in Cards
Alle Buttons und interaktiven Elemente in Cards und Dropdowns SHALL eine Mindesthöhe von 44px aufweisen.

#### Scenario: Ausreichende Button-Größe
- **WHEN** ein Nutzer mit dem Finger interagiert
- **THEN** haben alle Buttons und Dropdown-Einträge mindestens 44px Höhe

### Requirement: Serverseitige Paginierung für große Listen
`GET /api/members` und `GET /api/admin/users` SHALL Query-Parameter `search`, `limit` und `offset` akzeptieren. Die Response MUSS das Format `{ items: T[], total: int }` haben. Der Default-Wert für `limit` ist 50.

#### Scenario: Erste Seite laden
- **WHEN** MembersPage geladen wird
- **THEN** wird `GET /api/members?limit=50&offset=0` aufgerufen
- **THEN** zeigt die Seite die ersten 50 Einträge

#### Scenario: „Mehr laden" lädt nächste Einträge
- **WHEN** der Nutzer auf „Mehr laden" klickt
- **WHEN** noch weitere Einträge vorhanden sind (`offset + limit < total`)
- **THEN** wird `GET /api/members?limit=50&offset=50` aufgerufen
- **THEN** werden die neuen Einträge an die bestehende Liste angehängt

#### Scenario: „Mehr laden" ausgeblendet wenn alle geladen
- **WHEN** alle Einträge geladen sind (`items.length >= total`)
- **THEN** ist der „Mehr laden"-Button nicht sichtbar

#### Scenario: Serverseitige Suche
- **WHEN** der Nutzer in die Suchleiste tippt (300ms Debounce)
- **THEN** wird `GET /api/members?search=<term>&limit=50&offset=0` aufgerufen
- **THEN** zeigt die Liste nur die passenden Einträge
- **THEN** wird der `offset` auf 0 zurückgesetzt

### Requirement: Responsive Grid/Form-Seiten
Alle Seiten mit Grid-Layouts oder Formularen SHALL auf Mobile als einspaltige, vertikal gestapelte Layouts erscheinen. Button-Gruppen MÜSSEN auf Mobile umbrechen oder vertikal gestapelt werden.

#### Scenario: Grid wird einspaltig auf Mobile
- **WHEN** eine Seite mit `grid-cols-2` auf einem Gerät unter 640px aufgerufen wird
- **THEN** werden die Spalten vertikal gestapelt

#### Scenario: Button-Gruppen umbrechen auf Mobile
- **WHEN** eine Seite mit mehreren nebeneinanderliegenden Buttons auf Mobile aufgerufen wird
- **THEN** sind die Buttons untereinander oder umbrechen in die nächste Zeile
