## MODIFIED Requirements

### Requirement: Nutzer-konfigurierbare Notification-Präferenzen
Das System SHALL es jedem Nutzer ermöglichen, pro Kategorie Push-Benachrichtigungen und (wo verfügbar) E-Mail-Benachrichtigungen ein- oder auszuschalten. Die Einstellungen sind im Profil unter „Sonstiges" zugänglich. Die Kategorie `chat` MUST als vollwertige, persistierbare Kategorie behandelt werden (der DB-CHECK von `notification_preferences.category` schließt `chat` ein). Das Speichern MUST transaktional erfolgen (alles-oder-nichts) und unbekannte Kategorien mit HTTP 400 ablehnen.

#### Scenario: Präferenzen abrufen
- **WHEN** ein eingeloggter Nutzer `GET /api/profile/notification-preferences` aufruft
- **THEN** erhält er ein JSON-Objekt mit allen Kategorien (inkl. `chat`) und ihren aktuellen `push_enabled`/`email_enabled`-Werten (Default: push=true, email=false für alle)

#### Scenario: Präferenzen speichern
- **WHEN** ein Nutzer `PUT /api/profile/notification-preferences` mit einem Objekt der Kategorien und Werte aufruft
- **THEN** werden die Präferenzen in `notification_preferences` gespeichert und 204 zurückgegeben

#### Scenario: Chat-Präferenz speichern und wirksam abschalten
- **WHEN** ein Nutzer `PUT /api/profile/notification-preferences` mit `chat: { push: false }` aufruft
- **THEN** wird die Zeile `(user_id, 'chat', push_enabled=0)` gespeichert (kein CHECK-Fehler, kein 500), 204 zurückgegeben, und `FilterByPushPref` schließt diesen Nutzer bei Chat-Pushes künftig aus

#### Scenario: Unbekannte Kategorie wird abgelehnt
- **WHEN** ein Nutzer `PUT /api/profile/notification-preferences` mit einer Kategorie aufruft, die nicht in der erlaubten Menge liegt
- **THEN** antwortet die API mit HTTP 400 und schreibt **keine** der übermittelten Kategorien (transaktional zurückgerollt)

#### Scenario: Default ohne gespeicherte Präferenz
- **WHEN** ein Nutzer noch keine Zeile in `notification_preferences` für eine Kategorie hat
- **THEN** gilt: `push_enabled=true`, `email_enabled=false` (in API und Notification-Logik)

### Requirement: Profil-UI für Notification-Präferenzen
Das System SHALL im Profil-Tab „Sonstiges" einen Abschnitt „Benachrichtigungen" anzeigen mit Toggle-Rows pro Kategorie, einschließlich der Kategorie „Nachrichten" (`chat`).

#### Scenario: Profil zeigt alle Kategorien
- **WHEN** ein Nutzer den Tab „Sonstiges" im Profil öffnet
- **THEN** sieht er Toggle-Rows für: Spiele, Trainings, Dienste, Dienst-Erinnerung (mit Push + E-Mail-Toggle), Fahrgemeinschaften, Nachrichten

#### Scenario: Chat-Toggle wirkt persistent
- **WHEN** der Nutzer den „Nachrichten"-Toggle ausschaltet und speichert, dann die Seite neu lädt
- **THEN** ist der „Nachrichten"-Toggle weiterhin ausgeschaltet (die Präferenz wurde erfolgreich persistiert)

#### Scenario: E-Mail-Toggle nur bei Dienst-Erinnerung
- **WHEN** der Nutzer den Abschnitt „Benachrichtigungen" betrachtet
- **THEN** ist der E-Mail-Toggle nur bei „Dienst-Erinnerung" sichtbar, alle anderen Kategorien zeigen nur Push
