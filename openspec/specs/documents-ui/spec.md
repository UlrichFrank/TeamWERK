# documents-ui Specification

## Purpose
TBD - created by archiving change dateiablage. Update Purpose after archive.
## Requirements
### Requirement: Navigation
Der Nav-Eintrag „Dokumente" MUSS im NavModule „Mitglieder" direkt nach dem Eintrag „Mein Profil" erscheinen. Der Eintrag MUSS für alle authentifizierten Nutzer sichtbar sein (`roles: []`).

#### Scenario: Nav-Eintrag sichtbar
- **WHEN** ein authentifizierter Nutzer die App öffnet
- **THEN** erscheint „Dokumente" im Abschnitt „Mitglieder" der Sidebar

#### Scenario: Nav-Eintrag für Gäste unsichtbar
- **WHEN** ein nicht eingeloggter Nutzer die App öffnet
- **THEN** ist der Eintrag „Dokumente" nicht in der Navigation sichtbar

### Requirement: Desktop-Layout (≥ 640px)
Die Seite `/dokumente` SOLL ein Zwei-Panel-Layout zeigen: linke Spalte mit aufklappbarem Ordnerbaum, rechte Spalte mit Dateiliste als `<table>`. Oberhalb der Dateiliste SOLLEN ein Breadcrumb-Pfad und (bei `can_write`) die Buttons „↑ Hochladen" und „+ Neuer Ordner" erscheinen. Alle Klassen folgen den Brand-Tokens aus CLAUDE.md.

#### Scenario: Ordnerbaum-Navigation
- **WHEN** ein Nutzer auf einen Ordner im linken Panel klickt
- **THEN** zeigt das rechte Panel den Inhalt dieses Ordners und der Breadcrumb aktualisiert sich

#### Scenario: Buttons nur bei Schreibrecht
- **WHEN** ein Nutzer ohne `can_write` die Seite aufruft
- **THEN** sind „↑ Hochladen" und „+ Neuer Ordner" nicht sichtbar

#### Scenario: Tabellen-Zeilenaktionen
- **WHEN** ein Nutzer eine Datei-Zeile betrachtet
- **THEN** erscheint ein Download-Icon und (bei `can_write`) ein ⋮-Dropdown mit „Löschen" und „Berechtigungen"

### Requirement: Mobile-Layout (< 640px)
Auf Mobile SOLL kein Ordnerbaum-Sidebar angezeigt werden. Stattdessen SOLL der Nutzer durch Antippen von Ordner-Cards navigieren. Ein Breadcrumb MUSS `sticky top-0 z-10` oben angezeigt werden. Dateien und Ordner MÜSSEN als Cards dargestellt werden. Aktionen MÜSSEN hinter einem ⋮-Dropdown liegen. Alle interaktiven Elemente MÜSSEN mindestens 44px Höhe haben (`py-2.5`).

#### Scenario: Ordner-Navigation per Card
- **WHEN** ein Nutzer auf eine Ordner-Card tippt
- **THEN** wechselt die Ansicht zum Inhalt dieses Ordners und der Breadcrumb zeigt den neuen Pfad

#### Scenario: Zurück-Navigation via Breadcrumb
- **WHEN** ein Nutzer auf einen Eintrag im Breadcrumb tippt
- **THEN** wechselt die Ansicht zurück zu diesem Ordner

#### Scenario: Datei-Card Inhalt
- **WHEN** eine Datei in der mobilen Ansicht angezeigt wird
- **THEN** zeigt die Card: Dateiname, Typ, Größe, Datum, Uploader-Name und ein ⋮-Dropdown-Icon

### Requirement: Upload-Dialog
Ein Modal SOLL für den Datei-Upload verwendet werden. Es MUSS einen Datei-Picker, eine Fortschrittsanzeige während des Uploads und eine Erfolgsmeldung nach Abschluss enthalten. Das Modal folgt der Klasse `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`.

#### Scenario: Upload-Fortschritt
- **WHEN** ein Nutzer eine Datei hochlädt
- **THEN** zeigt der Dialog einen Fortschrittsbalken bis der Upload abgeschlossen ist

#### Scenario: Upload-Fehler
- **WHEN** ein Upload fehlschlägt (Datei zu groß oder kein Recht)
- **THEN** zeigt der Dialog eine Fehlermeldung mit `bg-brand-danger-light border border-brand-danger/30`

### Requirement: Berechtigungs-Modal
Ein Modal SOLL für die Berechtigungsverwaltung verwendet werden. Es SOLL nur für Nutzer mit `can_write` erreichbar sein. Das Modal zeigt bestehende ACL-Einträge des Ordners und erlaubt das Hinzufügen neuer Einträge (Principal-Typ, Referenz, Lesen/Schreiben). Vergabe ist auf eigene Rechte begrenzt (Anti-Eskalation).

#### Scenario: Berechtigungs-Modal öffnen
- **WHEN** ein Nutzer mit `can_write` „Berechtigungen" im ⋮-Dropdown wählt
- **THEN** öffnet sich ein Modal mit den aktuellen ACL-Einträgen des Ordners

#### Scenario: Neuen Eintrag anlegen
- **WHEN** ein Nutzer einen neuen Eintrag mit Principal-Typ „Rolle" und Wert „trainer" anlegt
- **THEN** erscheint der Eintrag in der Liste und ist ab sofort aktiv

#### Scenario: Eintrag entfernen
- **WHEN** ein Nutzer auf „Entfernen" bei einem ACL-Eintrag klickt
- **THEN** wird der Eintrag gelöscht; geerbte Rechte bleiben sichtbar aber nicht entfernbar

