## ADDED Requirements

### Requirement: VAPID Public Key verfügbar
Der Server MUSS einen Endpunkt bereitstellen, über den das Frontend den VAPID Public Key abrufen kann, um eine Push Subscription zu erstellen.

#### Scenario: Public Key abrufen
- **WHEN** ein Client `GET /api/push/vapid-public-key` aufruft
- **THEN** antwortet der Server mit `{ "publicKey": "<base64url-encoded-key>" }` und HTTP 200

### Requirement: Subscription registrieren
Das System MUSS es authentifizierten Usern ermöglichen, ihre Push Subscription (Endpoint + Schlüssel) serverseitig zu speichern. Ein User kann mehrere Subscriptions haben (verschiedene Geräte).

#### Scenario: Neue Subscription speichern
- **WHEN** ein authentifizierter User `POST /api/push/subscribe` mit `{ endpoint, p256dh, auth }` aufruft
- **THEN** wird die Subscription in `push_subscriptions` gespeichert (INSERT OR REPLACE) und der Server antwortet mit HTTP 204

#### Scenario: Doppelte Subscription (selber Endpoint)
- **WHEN** ein User denselben Endpoint erneut registriert
- **THEN** wird die bestehende Subscription aktualisiert (kein Duplikat) und der Server antwortet mit HTTP 204

#### Scenario: Fehlende Felder
- **WHEN** `endpoint`, `p256dh` oder `auth` fehlen im Request
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unauthentifizierter Zugriff
- **WHEN** ein nicht eingeloggter Client die Route aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Subscription löschen
Das System MUSS es authentifizierten Usern ermöglichen, ihre Push Subscription zu entfernen.

#### Scenario: Subscription entfernen
- **WHEN** ein authentifizierter User `DELETE /api/push/subscribe` mit `{ endpoint }` aufruft
- **THEN** wird die Subscription aus `push_subscriptions` gelöscht und der Server antwortet mit HTTP 204

### Requirement: Tote Endpoints bereinigen
Das System MUSS Push Subscriptions automatisch entfernen, wenn der Push-Dienst HTTP 410 Gone zurückgibt.

#### Scenario: Push-Endpoint ungültig (410)
- **WHEN** beim Senden einer Push Notification der Push-Service mit HTTP 410 antwortet
- **THEN** wird die Subscription aus `push_subscriptions` gelöscht und kein weiterer Push-Versuch unternommen

### Requirement: Frontend abonniert Push silent beim App-Start
Das Frontend SOLL beim App-Start versuchen, Push Notifications zu abonnieren, ohne den User zu unterbrechen.

#### Scenario: Push unterstützt — Android Chrome
- **WHEN** die App in Android Chrome startet (installiert oder nicht), `PushManager` verfügbar ist und die Permission nicht `denied` ist
- **THEN** abonniert das Frontend Push Notifications und sendet die Subscription an `POST /api/push/subscribe`

#### Scenario: Push unterstützt — iOS installierte PWA
- **WHEN** iOS erkannt wird (`/iphone|ipad|ipod/i`), die App als installierte PWA läuft (`display-mode: standalone`) und `PushManager` verfügbar ist und die Permission nicht `denied` ist
- **THEN** abonniert das Frontend Push Notifications und sendet die Subscription an `POST /api/push/subscribe`

#### Scenario: Kein Push — iOS Safari ohne PWA-Install
- **WHEN** iOS erkannt wird (`/iphone|ipad|ipod/i`) und `display-mode: standalone` nicht aktiv ist
- **THEN** bricht der Subscribe-Versuch ohne Fehlermeldung ab

#### Scenario: Kein Push — PushManager nicht verfügbar
- **WHEN** `'PushManager' in window` false ergibt (älterer Browser, Desktop-Safari ohne PWA)
- **THEN** bricht der Subscribe-Versuch ohne Fehlermeldung ab
