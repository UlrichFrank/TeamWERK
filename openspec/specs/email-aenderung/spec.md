# email-aenderung Specification

## Purpose

Diese Spezifikation beschreibt die Capability `email-aenderung`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Nutzer kann E-Mail-Änderung mit Passwort-Verifikation anfordern
Das System SHALL jedem authentifizierten Nutzer erlauben, eine E-Mail-Adressänderung anzufordern. Das aktuelle Passwort MUSS korrekt sein. Das System speichert die ausstehende Änderung in `email_change_tokens` (TTL 24h) und sendet einen Bestätigungslink an die neue Adresse.

#### Scenario: E-Mail-Änderung erfolgreich angefordert
- **WHEN** ein eingeloggter Nutzer `POST /api/profile/email` mit `{ "new_email": "neu@example.com", "password": "..." }` aufruft und das Passwort korrekt ist
- **THEN** wird ein Token in `email_change_tokens` gespeichert, eine Bestätigungs-Mail an `neu@example.com` gesendet und HTTP 204 zurückgegeben

#### Scenario: Falsches Passwort wird abgelehnt
- **WHEN** `POST /api/profile/email` mit falschem `password` aufgerufen wird
- **THEN** antwortet das System mit HTTP 403; kein Token wird erzeugt

#### Scenario: Neue E-Mail bereits vergeben
- **WHEN** `POST /api/profile/email` mit einer `new_email` aufgerufen wird, die bereits in `users.email` existiert
- **THEN** antwortet das System mit HTTP 409

#### Scenario: Mehrfache Anfragen überschreiben vorherigen Token
- **WHEN** ein Nutzer eine zweite E-Mail-Änderungs-Anfrage stellt bevor der erste Token bestätigt wurde
- **THEN** wird der alte Token ungültig (gelöscht oder überschrieben) und ein neuer Token für die neue Adresse ausgestellt

### Requirement: Bestätigungslink aktiviert die neue E-Mail-Adresse
Das System SHALL beim Aufrufen des Bestätigungslinks (`GET /api/profile/email/confirm?token=xyz`) die neue E-Mail-Adresse in `users.email` eintragen, alle Refresh-Tokens des Nutzers löschen und zu `/login` weiterleiten.

#### Scenario: Gültiger Token — E-Mail wird getauscht
- **WHEN** `GET /api/profile/email/confirm?token=xyz` mit einem gültigen, unbenutzten Token aufgerufen wird
- **THEN** wird `users.email` auf `new_email` aktualisiert, der Token als `used_at` markiert, alle `refresh_tokens` des Nutzers gelöscht und HTTP 302 zu `/login` zurückgegeben

#### Scenario: Abgelaufener oder ungültiger Token
- **WHEN** `GET /api/profile/email/confirm?token=xyz` mit einem abgelaufenen oder nicht existierenden Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 302 zu `/login?error=invalid_token` (kein E-Mail-Tausch)

#### Scenario: Bereits verwendeter Token
- **WHEN** derselbe Bestätigungslink ein zweites Mal aufgerufen wird
- **THEN** antwortet das System mit HTTP 302 zu `/login?error=invalid_token`

### Requirement: Profilseite zeigt E-Mail-Änderungsformular
Das Frontend SHALL ein Formular mit zwei Feldern anzeigen: neue E-Mail-Adresse und aktuelles Passwort. Nach dem Absenden wird ein Hinweis angezeigt, dass eine Bestätigungs-Mail versendet wurde.

#### Scenario: Bestätigungs-Hinweis nach Absenden
- **WHEN** ein Nutzer das E-Mail-Formular erfolgreich absendet
- **THEN** verschwindet das Formular und der Hinweis „Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach." wird angezeigt

#### Scenario: Fehler bei falschem Passwort
- **WHEN** der API-Call HTTP 403 zurückgibt
- **THEN** zeigt das Formular die Meldung „Passwort nicht korrekt"
