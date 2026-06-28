# documents-ui Specification

## Purpose
TBD - created by archiving change dateiablage. Update Purpose after archive.
## Requirements
### Requirement: Navigation
Der Nav-Eintrag „Dokumente" MUST im NavModule „Mitglieder" direkt nach dem Eintrag „Mein Profil" erscheinen. Der Eintrag MUST für alle authentifizierten Nutzer sichtbar sein (`roles: []`).

#### Scenario: Nav-Eintrag sichtbar
- **WHEN** ein authentifizierter Nutzer die App öffnet
- **THEN** erscheint „Dokumente" im Abschnitt „Mitglieder" der Sidebar

#### Scenario: Nav-Eintrag für Gäste unsichtbar
- **WHEN** ein nicht eingeloggter Nutzer die App öffnet
- **THEN** ist der Eintrag „Dokumente" nicht in der Navigation sichtbar

### Requirement: Desktop-Layout (≥ 640px)
Die Seite `/dokumente` SHALL ein Zwei-Panel-Layout zeigen: linke Spalte mit aufklappbarem Ordnerbaum, rechte Spalte mit Dateiliste als `<table>`. Oberhalb der Dateiliste SHALL ein Breadcrumb-Pfad und (bei `can_write`) die Buttons „↑ Hochladen" und „+ Neuer Ordner" erscheinen. Alle Klassen folgen den Brand-Tokens aus CLAUDE.md.

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
Auf Mobile SHALL kein Ordnerbaum-Sidebar angezeigt werden. Stattdessen SHALL der Nutzer durch Antippen von Ordner-Cards navigieren. Ein Breadcrumb MUST `sticky top-0 z-10` oben angezeigt werden. Dateien und Ordner MUST als Cards dargestellt werden. Aktionen MUST hinter einem ⋮-Dropdown liegen. Alle interaktiven Elemente MUST mindestens 44px Höhe haben (`py-2.5`).

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
Ein Modal SHALL für den Datei-Upload verwendet werden. Es MUST einen Datei-Picker, eine Fortschrittsanzeige während des Uploads und eine Erfolgsmeldung nach Abschluss enthalten. Das Modal folgt der Klasse `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`.

#### Scenario: Upload-Fortschritt
- **WHEN** ein Nutzer eine Datei hochlädt
- **THEN** zeigt der Dialog einen Fortschrittsbalken bis der Upload abgeschlossen ist

#### Scenario: Upload-Fehler
- **WHEN** ein Upload fehlschlägt (Datei zu groß oder kein Recht)
- **THEN** zeigt der Dialog eine Fehlermeldung mit `bg-brand-danger-light border border-brand-danger/30`

### Requirement: Datei öffnen (alle Plattformen)
Das System MUST beim Klick auf eine Datei zuerst ein kurzlebiges Download-Token vom Backend anfordern und anschließend die Datei-URL mit Token via `window.open(url, '_blank')` öffnen. Es DARF kein Blob heruntergeladen oder eine Blob-URL erzeugt werden. Der Browser entscheidet anhand des `Content-Type` selbst, ob er die Datei anzeigt (PDF, Bild) oder herunterlädt (DOCX, ZIP). Dies MUST sowohl im iOS-PWA-Standalone-Modus als auch im Desktop-Browser funktionieren.

#### Scenario: PDF-Klick in iOS PWA
- **WHEN** ein Nutzer in der installierten iOS PWA auf eine PDF-Datei klickt
- **THEN** öffnet Safari die Datei in der nativen PDF-Ansicht

#### Scenario: Bild-Klick im Desktop-Browser
- **WHEN** ein Nutzer im Desktop-Browser auf eine Bilddatei klickt
- **THEN** öffnet ein neuer Tab die Datei direkt im Browser

#### Scenario: DOCX-Klick (kein nativer Viewer)
- **WHEN** ein Nutzer auf eine DOCX-Datei klickt
- **THEN** triggert der Browser einen Download-Dialog

#### Scenario: Token-Fehler beim Öffnen
- **WHEN** die Token-Anfrage fehlschlägt (z.B. Netzwerkfehler)
- **THEN** zeigt die UI einen Fehlerhinweis (kein stiller Fehler)

### Requirement: Berechtigungs-Modal
Ein Modal SHALL für die Berechtigungsverwaltung verwendet werden. Es SHALL nur für Nutzer mit `can_write` erreichbar sein. Das Modal zeigt bestehende ACL-Einträge des Ordners und erlaubt das Hinzufügen neuer Einträge (Principal-Typ, Referenz, Lesen/Schreiben). Vergabe ist auf eigene Rechte begrenzt (Anti-Eskalation).

#### Scenario: Berechtigungs-Modal öffnen
- **WHEN** ein Nutzer mit `can_write` „Berechtigungen" im ⋮-Dropdown wählt
- **THEN** öffnet sich ein Modal mit den aktuellen ACL-Einträgen des Ordners

#### Scenario: Neuen Eintrag anlegen
- **WHEN** ein Nutzer einen neuen Eintrag mit Principal-Typ „Rolle" und Wert „trainer" anlegt
- **THEN** erscheint der Eintrag in der Liste und ist ab sofort aktiv

#### Scenario: Eintrag entfernen
- **WHEN** ein Nutzer auf „Entfernen" bei einem ACL-Eintrag klickt
- **THEN** wird der Eintrag gelöscht; geerbte Rechte bleiben sichtbar aber nicht entfernbar

