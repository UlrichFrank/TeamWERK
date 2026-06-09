## ADDED Requirements

### Requirement: Admin kann SEPA-Mandat-Dokument hochladen
Der Admin SHALL ein SEPA-Mandat-Dokument (PDF oder Bild) in der Mitglieder-Detailseite hochladen können. Nach erfolgreichem Upload SHALL das Dokument sofort als Öffnen-Link angezeigt werden.

#### Scenario: Upload erfolgreich
- **WHEN** der Admin eine Datei auswählt und hochlädt
- **THEN** wird `POST /api/upload/sepa-mandat/{memberId}` aufgerufen und nach Erfolg ein Öffnen-Link angezeigt

#### Scenario: Upload-Fehler
- **WHEN** der Server einen Fehler zurückgibt
- **THEN** wird eine Fehlermeldung im Tab angezeigt, der Button ist wieder verfügbar

### Requirement: Berechtigter Zugriff auf SEPA-Mandat-Dokument
Das System SHALL das SEPA-Mandat-Dokument nur für berechtigte Nutzer zugänglich machen. Der Zugriff MUSS über einen kurzlebigen HMAC-Token gesichert sein.

#### Scenario: Berechtigter Nutzer öffnet Dokument
- **WHEN** ein Mitglied (isOwn), ein Elternteil (family_link), ein Vorstandsmitglied oder ein Admin auf den Öffnen-Link klickt
- **THEN** wird zuerst ein Token via `GET /api/members/{id}/sepa-mandat/download-token` geholt und das Dokument via `window.open` geöffnet

#### Scenario: Unbefugter Zugriff auf Download-Token
- **WHEN** ein Nutzer ohne Berechtigung `GET /api/members/{id}/sepa-mandat/download-token` aufruft
- **THEN** antwortet der Server mit 403 Forbidden

#### Scenario: Direktzugriff auf Download ohne gültigen Token
- **WHEN** `GET /api/members/{id}/sepa-mandat/download` ohne oder mit abgelaufenem Token aufgerufen wird
- **THEN** antwortet der Server mit 401 Unauthorized

#### Scenario: Kein Dokument hinterlegt
- **WHEN** `sepa_mandat_path` NULL ist und der Download-Endpoint aufgerufen wird
- **THEN** antwortet der Server mit 404 Not Found

### Requirement: Sichtbarkeit des Öffnen-Links
Das System SHALL den Öffnen-Link in der Mitglieder-Detailansicht allen berechtigten Nutzern anzeigen.

#### Scenario: `sepa_mandat_url` in API-Response für berechtigte Rollen
- **WHEN** `GET /api/members/{id}` von einem Mitglied (isOwn), Elternteil, Vorstand oder Admin aufgerufen wird und ein Dokument vorhanden ist
- **THEN** enthält die Response `sepa_mandat_url`

#### Scenario: `sepa_mandat_url` nicht sichtbar für Trainer/Spieler ohne Bezug
- **WHEN** ein Trainer oder ein Spieler ohne Familienbezug `GET /api/members/{id}` aufruft
- **THEN** ist `sepa_mandat_url` nicht in der Response enthalten

### Requirement: Dokument öffnen auf iOS und Desktop
Das Öffnen des Dokuments MUSS sowohl im Desktop-Browser als auch auf iOS (PWA Standalone-Modus) funktionieren.

#### Scenario: Öffnen auf Desktop
- **WHEN** der Nutzer auf den Öffnen-Link klickt
- **THEN** öffnet sich ein neuer Tab mit dem Dokument

#### Scenario: Öffnen auf iOS PWA
- **WHEN** der Nutzer in der installierten iOS PWA auf den Öffnen-Link tippt
- **THEN** öffnet Safari die Datei in der nativen Ansicht (kein stilles Versagen)

#### Scenario: Token-Fehler beim Öffnen
- **WHEN** der Token-Fetch fehlschlägt
- **THEN** wird der leere Tab geschlossen und eine Fehlermeldung angezeigt

### Requirement: Mitglied kann Mandat zurückziehen
Ein Mitglied SHALL sein eigenes SEPA-Mandat-Dokument löschen können. Elternteile (family_link), Vorstand und Admin KÖNNEN ebenfalls löschen.

#### Scenario: Mitglied zieht Mandat zurück
- **WHEN** ein Mitglied auf „Zurückziehen" klickt und bestätigt
- **THEN** wird `DELETE /api/members/{id}/sepa-mandat` aufgerufen, die Datei vom Server gelöscht und `sepa_mandat_path` auf NULL gesetzt

#### Scenario: Elternteil löscht Mandat
- **WHEN** ein Elternteil (family_link) `DELETE /api/members/{id}/sepa-mandat` aufruft
- **THEN** wird die Datei gelöscht und `sepa_mandat_path` auf NULL gesetzt

#### Scenario: Kein Dokument vorhanden beim Löschen
- **WHEN** `DELETE /api/members/{id}/sepa-mandat` aufgerufen wird, aber kein Dokument hinterlegt ist
- **THEN** antwortet der Server mit 404 Not Found

### Requirement: DocumentsPage öffnet Dateien auf iOS korrekt
`DocumentsPage.openFile` MUSS auf iOS (PWA Standalone-Modus) funktionieren. Das neue Fenster MUSS synchron vor dem async Token-Fetch geöffnet werden.

#### Scenario: PDF-Klick in iOS PWA
- **WHEN** ein Nutzer in der installierten iOS PWA auf eine Datei klickt
- **THEN** öffnet Safari die Datei (kein stilles Versagen)

#### Scenario: Token-Fehler beim Öffnen in DocumentsPage
- **WHEN** die Token-Anfrage fehlschlägt
- **THEN** wird der leere Tab geschlossen und eine Fehlermeldung angezeigt
