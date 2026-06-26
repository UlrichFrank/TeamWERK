# web-push-subscriptions Specification

## Purpose

Diese Spezifikation beschreibt die Capability `web-push-subscriptions`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

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
Das Backend MUST Subscriptions automatisch entfernen, deren Endpoint beim Versuch einer Push-Nachricht mit einer der folgenden HTTP-Statuscodes antwortet: 410 Gone, 404 Not Found, 401 Unauthorized, oder 400 Bad Request. Diese Statuscodes zeigen an, dass die Subscription ungültig oder nicht mehr erreichbar ist und nicht erneut kontaktiert werden soll.

#### Scenario: Push-Dienst meldet abgelaufenen Endpoint (410/404)
- **WHEN** der Push-Dienst beim Versand eine 410- oder 404-Antwort zurückgibt
- **THEN** wird die betreffende Subscription aus der Datenbank gelöscht, sodass sie nicht erneut kontaktiert wird

#### Scenario: Push-Dienst meldet ungültige Auth (401)
- **WHEN** der Push-Dienst beim Versand HTTP 401 zurückgibt (z.B. abgelaufene VAPID-Keys oder ungültige Subscription-Auth)
- **THEN** wird die betreffende Subscription aus der Datenbank gelöscht

#### Scenario: Push-Dienst meldet ungültige Anfrage (400)
- **WHEN** der Push-Dienst beim Versand HTTP 400 zurückgibt (z.B. fehlerhaftes Subscription-Objekt)
- **THEN** wird die betreffende Subscription aus der Datenbank gelöscht

#### Scenario: Transienter Fehler — Subscription bleibt erhalten
- **WHEN** der Push-Dienst beim Versand einen 5xx-Fehler oder Netzwerkfehler zurückgibt
- **THEN** wird die Subscription NICHT gelöscht (transienter Fehler, Retry beim nächsten Scheduler-Run)

### Requirement: Frontend abonniert Push silent beim App-Start
Das Frontend MUST beim App-Start (nach Login) prüfen, ob eine Push-Subscription vorliegt, und diese ggf. anlegen — ohne den Nutzer um Erlaubnis zu fragen, wenn die Berechtigung bereits erteilt wurde.

#### Scenario: Berechtigung bereits erteilt
- **WHEN** der Nutzer die App lädt und `Notification.permission === 'granted'`
- **THEN** holt das Frontend den VAPID Public Key, erstellt oder aktualisiert die Subscription im Service Worker und registriert sie via `POST /api/push/subscribe` — ohne Nutzerinteraktion

#### Scenario: Berechtigung noch nicht erteilt
- **WHEN** `Notification.permission` nicht `'granted'` ist
- **THEN** unternimmt das Frontend beim App-Start keinen automatischen Abonnierversuch; die Erlaubnisabfrage erfolgt zu einem späteren, nutzerinitiierten Zeitpunkt
