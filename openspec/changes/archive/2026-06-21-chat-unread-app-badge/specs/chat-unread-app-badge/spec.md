## ADDED Requirements

### Requirement: App-Icon-Badge spiegelt Chat-Unread im offenen Frontend

Das System SHALL bei jeder Änderung der Chat-Unread-Summe (Konversations-Unreads + ungelesene Broadcasts) den App-Icon-Badge auf den aktuellen Wert setzen, wenn der Browser die Web Badging API unterstützt (`'setAppBadge' in navigator`). Bei 0 SHALL der Badge entfernt werden (`clearAppBadge`).

#### Scenario: Neue Nachricht trifft ein, App ist offen
- **WHEN** ein anderer Nutzer eine Chat-Nachricht an den eingeloggten User sendet und das SSE-Event `chat:new-message` `chatUnread` auf 3 erhöht
- **THEN** ruft das Frontend `navigator.setAppBadge(3)` auf

#### Scenario: User liest eine Konversation
- **WHEN** der eingeloggte User eine Konversation öffnet, die zuvor 2 ungelesene Nachrichten hatte, und `chatUnread` von 5 auf 3 fällt
- **THEN** ruft das Frontend `navigator.setAppBadge(3)` auf

#### Scenario: Alle Nachrichten gelesen
- **WHEN** der eingeloggte User alle ungelesenen Konversationen und Broadcasts gelesen hat und `chatUnread` 0 wird
- **THEN** ruft das Frontend `navigator.clearAppBadge()` auf

#### Scenario: User loggt sich aus
- **WHEN** der eingeloggte User Logout auslöst
- **THEN** ruft das Frontend `navigator.clearAppBadge()` auf, unabhängig vom zuletzt gesetzten Wert

#### Scenario: Browser unterstützt keine Badging-API
- **WHEN** `'setAppBadge' in navigator` `false` ist (z.B. Firefox)
- **THEN** wird kein Badge-Call ausgeführt; es entstehen keine Console-Errors

### Requirement: Push-Payload trägt aktuellen Chat-Unread des Empfängers

Beim Versand eines Chat-bezogenen Push (neue Nachricht in Konversation oder neuer Broadcast) SHALL die Payload das Feld `badge` enthalten — gesetzt auf den per `chat.ComputeUnreadForUser` ermittelten aktuellen Stand des Empfängers (inklusive der gerade ausgelösten Nachricht). Andere Push-Caller (Games, Trainings, Duties) bleiben unverändert und schicken kein `badge`-Feld.

#### Scenario: Empfänger hat bereits 2 ungelesene Nachrichten
- **WHEN** in Konversation A bereits 2 ungelesene Nachrichten für den Empfänger existieren und ein anderer Nutzer eine dritte Nachricht in Konversation B sendet
- **THEN** wird die Push-Payload an den Empfänger mit `badge: 3` versendet

#### Scenario: Empfänger hat keinen weiteren Unread
- **WHEN** alle bisherigen Konversationen des Empfängers gelesen sind und eine neue Nachricht in einer Konversation eintrifft
- **THEN** wird die Push-Payload mit `badge: 1` versendet

#### Scenario: Neuer Broadcast
- **WHEN** ein Vorstand einen Broadcast versendet und der Empfänger keine weiteren ungelesenen Chat-Nachrichten hat
- **THEN** wird die Push-Payload an den Empfänger mit `badge: 1` versendet

#### Scenario: Empfänger hat alten ungelesenen Broadcast
- **WHEN** der Empfänger einen Broadcast vor einer Woche nicht gelesen hat und eine neue Direktnachricht eintrifft
- **THEN** wird die Push-Payload mit `badge: 2` versendet (alter Broadcast zählt mit)

### Requirement: Service Worker setzt App-Badge aus Push-Payload

Bei Empfang eines Push mit `badge`-Feld SHALL der Service Worker zusätzlich zur Notification-Anzeige `navigator.setAppBadge(payload.badge)` aufrufen (bzw. `clearAppBadge()` bei 0), wenn die Badging-API verfügbar ist. Beide Operationen werden im selben `event.waitUntil` gewartet.

#### Scenario: Push mit badge=5 trifft ein, App ist geschlossen
- **WHEN** ein Push mit Payload `{"title":"Chat","body":"Neue Nachricht","url":"/chat","badge":5}` eintrifft und der Browser die Badging-API unterstützt
- **THEN** ruft der Service Worker `self.registration.showNotification("Chat", ...)` UND `navigator.setAppBadge(5)` auf

#### Scenario: Push ohne badge-Feld
- **WHEN** ein Push aus einem anderen Modul (z.B. Spielzusage) ohne `badge`-Feld eintrifft
- **THEN** wird `showNotification` gerufen, aber kein `setAppBadge`/`clearAppBadge` (Badge bleibt unverändert)

#### Scenario: Push mit badge=0
- **WHEN** ein Push mit `badge: 0` eintrifft
- **THEN** ruft der Service Worker `navigator.clearAppBadge()` auf

### Requirement: Helper-Funktion ComputeUnreadForUser

Das System SHALL eine in `internal/chat` exportierte Funktion `ComputeUnreadForUser(db *sql.DB, userID int) (int, error)` bereitstellen. Sie liefert die Summe aus (a) ungelesenen Nachrichten in allen Konversationen des Users — exakt die Semantik des bestehenden `unreadCount`-Felds aus `GET /api/chat/conversations` — und (b) der Anzahl ungelesener Broadcasts, die der User NICHT selbst gesendet hat.

#### Scenario: Nutzer ohne ungelesene Inhalte
- **WHEN** `ComputeUnreadForUser(db, userID)` für einen User aufgerufen wird, der alle seine Konversationen gelesen hat und keine ungelesenen Broadcasts hat
- **THEN** liefert die Funktion `(0, nil)`

#### Scenario: Nutzer mit Konversations- und Broadcast-Unreads
- **WHEN** der User 2 ungelesene Nachrichten in Konv. A, 1 in Konv. B und 3 ungelesene Broadcasts (alle nicht selbst gesendet) hat
- **THEN** liefert die Funktion `(6, nil)`

#### Scenario: Selbst gesendeter Broadcast wird nicht gezählt
- **WHEN** der User 1 ungelesene Nachricht in einer Konv. hat und einen eigenen, nicht-gelesenen Broadcast versendet hat
- **THEN** liefert die Funktion `(1, nil)` — der eigene Broadcast zählt nicht mit
