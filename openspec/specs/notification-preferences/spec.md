## ADDED Requirements

### Requirement: Nutzer-konfigurierbare Notification-Präferenzen
Das System SHALL es jedem Nutzer ermöglichen, pro Kategorie Push-Benachrichtigungen und (wo verfügbar) E-Mail-Benachrichtigungen ein- oder auszuschalten. Die Einstellungen sind im Profil unter „Sonstiges" zugänglich.

#### Scenario: Präferenzen abrufen
- **WHEN** ein eingeloggter Nutzer `GET /api/profile/notification-preferences` aufruft
- **THEN** erhält er ein JSON-Objekt mit allen Kategorien und ihren aktuellen `push_enabled`/`email_enabled`-Werten (Default: push=true, email=false für alle)

#### Scenario: Präferenzen speichern
- **WHEN** ein Nutzer `PUT /api/profile/notification-preferences` mit einem Objekt der Kategorien und Werte aufruft
- **THEN** werden die Präferenzen in `notification_preferences` gespeichert und 204 zurückgegeben

#### Scenario: Default ohne gespeicherte Präferenz
- **WHEN** ein Nutzer noch keine Zeile in `notification_preferences` für eine Kategorie hat
- **THEN** gilt: `push_enabled=true`, `email_enabled=false` (in API und Notification-Logik)

### Requirement: Profil-UI für Notification-Präferenzen
Das System SHALL im Profil-Tab „Sonstiges" einen Abschnitt „Benachrichtigungen" anzeigen mit Toggle-Rows pro Kategorie.

#### Scenario: Profil zeigt alle Kategorien
- **WHEN** ein Nutzer den Tab „Sonstiges" im Profil öffnet
- **THEN** sieht er Toggle-Rows für: Spiele, Trainings, Dienste, Dienst-Erinnerung (mit Push + E-Mail-Toggle), Fahrgemeinschaften

#### Scenario: E-Mail-Toggle nur bei Dienst-Erinnerung
- **WHEN** der Nutzer den Abschnitt „Benachrichtigungen" betrachtet
- **THEN** ist der E-Mail-Toggle nur bei „Dienst-Erinnerung" sichtbar, alle anderen Kategorien zeigen nur Push

#### Scenario: Hinweis auf vorherige Einstellung
- **WHEN** der Nutzer die Seite öffnet und hatte zuvor `duty_reminder_days` aktiviert
- **THEN** ist der E-Mail-Toggle für „Dienst-Erinnerung" entsprechend vorausgefüllt
