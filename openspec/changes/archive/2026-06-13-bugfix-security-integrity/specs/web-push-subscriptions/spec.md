## MODIFIED Requirements

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
