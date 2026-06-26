# passwort-aenderung Specification

## Purpose

Diese Spezifikation beschreibt die Capability `passwort-aenderung`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Nutzer kann sein Passwort mit Verifikation des alten Passworts ändern
Das System SHALL jedem authentifizierten Nutzer erlauben, sein Passwort zu ändern. Das aktuelle Passwort MUSS korrekt sein. Nach erfolgreicher Änderung werden alle Refresh-Tokens des Nutzers invalidiert.

#### Scenario: Passwort erfolgreich ändern
- **WHEN** ein eingeloggter Nutzer `POST /api/profile/password` mit `{ "current_password": "...", "new_password": "..." }` aufruft und `current_password` korrekt ist
- **THEN** wird das Passwort in `users.password` (bcrypt) aktualisiert, alle Einträge in `refresh_tokens` für diesen Nutzer gelöscht und HTTP 204 zurückgegeben

#### Scenario: Falsches aktuelles Passwort wird abgelehnt
- **WHEN** `POST /api/profile/password` mit einem falschen `current_password` aufgerufen wird
- **THEN** antwortet das System mit HTTP 403 und ändert das Passwort nicht

#### Scenario: Neues Passwort fehlt oder ist leer
- **WHEN** `POST /api/profile/password` ohne `new_password` oder mit leerem String aufgerufen wird
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Session wird nach Passwortänderung invalidiert
- **WHEN** ein Nutzer sein Passwort erfolgreich ändert
- **THEN** sind alle bestehenden Refresh-Tokens des Nutzers ungültig; ein Refresh-Versuch mit altem Token liefert HTTP 401

### Requirement: Profilseite zeigt Passwort-Änderungsformular
Das Frontend SHALL ein Formular mit drei Feldern anzeigen: aktuelles Passwort, neues Passwort, neues Passwort wiederholen. Das Formular MUSS client-seitig prüfen, ob die beiden neuen Passwörter übereinstimmen.

#### Scenario: Passwörter stimmen nicht überein
- **WHEN** ein Nutzer zwei unterschiedliche Werte in „Neues Passwort" und „Wiederholen" eingibt und absendet
- **THEN** wird eine Fehlermeldung angezeigt und kein API-Call ausgeführt

#### Scenario: Erfolgreich geändert — Hinweis auf Re-Login
- **WHEN** das Passwort erfolgreich geändert wurde
- **THEN** zeigt das Frontend den Hinweis „Du wirst in Kürze ausgeloggt" und leitet nach kurzem Delay zum Login um
