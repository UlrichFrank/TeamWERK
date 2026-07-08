## MODIFIED Requirements

### Requirement: Tote Endpoints bereinigen
Das Backend MUST Subscriptions automatisch entfernen, deren Endpoint beim Versuch einer Push-Nachricht mit **HTTP 410 Gone** oder **HTTP 404 Not Found** antwortet — diese Codes zeigen laut Web-Push-Spezifikation an, dass die Subscription endgültig ungültig ist. Bei **transienten** Fehlerantworten (insbesondere HTTP 400 Bad Request, 401 Unauthorized sowie 5xx/Netzwerkfehler) MUST das Backend die Subscription **erhalten** und den Vorfall protokollieren (`slog.Warn`), da diese Codes auf vorübergehende VAPID-Signatur-/Payload-Probleme hindeuten und ein gültiges Abo sonst dauerhaft verloren ginge.

#### Scenario: Push-Dienst meldet abgelaufenen Endpoint (410/404)
- **WHEN** der Push-Dienst beim Versand eine 410- oder 404-Antwort zurückgibt
- **THEN** wird die betreffende Subscription aus der Datenbank gelöscht, sodass sie nicht erneut kontaktiert wird

#### Scenario: Push-Dienst meldet ungültige Auth (401) — Subscription bleibt erhalten
- **WHEN** der Push-Dienst beim Versand HTTP 401 zurückgibt (z.B. transientes VAPID-Signatur-Problem)
- **THEN** wird die Subscription NICHT gelöscht, sondern der Vorfall mit `slog.Warn` protokolliert

#### Scenario: Push-Dienst meldet ungültige Anfrage (400) — Subscription bleibt erhalten
- **WHEN** der Push-Dienst beim Versand HTTP 400 zurückgibt
- **THEN** wird die Subscription NICHT gelöscht, sondern der Vorfall mit `slog.Warn` protokolliert

#### Scenario: Transienter Fehler — Subscription bleibt erhalten
- **WHEN** der Push-Dienst beim Versand einen 5xx-Fehler oder Netzwerkfehler zurückgibt
- **THEN** wird die Subscription NICHT gelöscht (transienter Fehler, Retry beim nächsten Versand)

### Requirement: Frontend abonniert Push silent beim App-Start
Das Frontend MUST beim App-Start (nach Login) prüfen, ob eine Push-Subscription vorliegt, und diese ggf. anlegen — ohne den Nutzer um Erlaubnis zu fragen, wenn die Berechtigung bereits erteilt wurde. Fehler beim Abonnieren MUST **beobachtbar** protokolliert werden (Konsole/`console.warn`), damit ein fehlgeschlagenes Re-Subscribe diagnostizierbar bleibt; sie dürfen nicht stillschweigend verschluckt werden.

#### Scenario: Berechtigung bereits erteilt
- **WHEN** der Nutzer die App lädt und `Notification.permission === 'granted'`
- **THEN** holt das Frontend den VAPID Public Key, erstellt oder aktualisiert die Subscription im Service Worker und registriert sie via `POST /api/push/subscribe` — ohne Nutzerinteraktion

#### Scenario: Abonnieren schlägt fehl — Fehler wird protokolliert
- **WHEN** beim Abonnieren ein Fehler auftritt (z.B. `applicationServerKey`-Mismatch/`InvalidStateError`, Netzwerkfehler beim `POST /api/push/subscribe`)
- **THEN** wird der Fehler beobachtbar in der Konsole protokolliert (nicht still verworfen), sodass der Abo-Verlust nachvollziehbar ist
