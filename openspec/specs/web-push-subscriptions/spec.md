## ADDED Requirements

### Requirement: VAPID Public Key verfügbar
Das Backend SHALL den VAPID Public Key über einen öffentlichen Endpunkt bereitstellen, damit das Frontend eine Push-Subscription anlegen kann.

#### Scenario: Frontend ruft Public Key ab
- **WHEN** `GET /api/push/vapid-public-key` aufgerufen wird (ohne Authentifizierung)
- **THEN** antwortet die API mit dem Base64url-kodierten VAPID Public Key

### Requirement: Subscription registrieren
Ein authentifizierter Nutzer SHALL seine Browser-Push-Subscription im Backend speichern können.

#### Scenario: Subscription anlegen
- **WHEN** `POST /api/push/subscribe` mit dem PushSubscription-Objekt (endpoint, keys) aufgerufen wird
- **THEN** wird die Subscription für den aktuellen Nutzer gespeichert (upsert anhand des Endpoints) und die API antwortet mit 201

#### Scenario: Doppelte Subscription
- **WHEN** derselbe Endpoint bereits für denselben Nutzer registriert ist
- **THEN** wird die bestehende Subscription aktualisiert (keys bleiben gleich oder werden erneuert) ohne Fehler

### Requirement: Subscription löschen
Ein authentifizierter Nutzer SHALL seine Push-Subscription widerrufen können.

#### Scenario: Subscription entfernen
- **WHEN** `DELETE /api/push/subscribe` mit dem Endpoint im Body aufgerufen wird
- **THEN** wird die Subscription aus der Datenbank entfernt und die API antwortet mit 204

### Requirement: Tote Endpoints bereinigen
Das Backend MUST Subscriptions automatisch entfernen, deren Endpoint beim Versuch einer Push-Nachricht mit 410 Gone oder 404 Not Found antwortet.

#### Scenario: Push-Dienst meldet abgelaufenen Endpoint
- **WHEN** der Push-Dienst beim Versand eine 410- oder 404-Antwort zurückgibt
- **THEN** wird die betreffende Subscription aus der Datenbank gelöscht, sodass sie nicht erneut kontaktiert wird

### Requirement: Frontend abonniert Push silent beim App-Start
Das Frontend MUST beim App-Start (nach Login) prüfen, ob eine Push-Subscription vorliegt, und diese ggf. anlegen — ohne den Nutzer um Erlaubnis zu fragen, wenn die Berechtigung bereits erteilt wurde.

#### Scenario: Berechtigung bereits erteilt
- **WHEN** der Nutzer die App lädt und `Notification.permission === 'granted'`
- **THEN** holt das Frontend den VAPID Public Key, erstellt oder aktualisiert die Subscription im Service Worker und registriert sie via `POST /api/push/subscribe` — ohne Nutzerinteraktion

#### Scenario: Berechtigung noch nicht erteilt
- **WHEN** `Notification.permission` nicht `'granted'` ist
- **THEN** unternimmt das Frontend beim App-Start keinen automatischen Abonnierversuch; die Erlaubnisabfrage erfolgt zu einem späteren, nutzerinitiierten Zeitpunkt
