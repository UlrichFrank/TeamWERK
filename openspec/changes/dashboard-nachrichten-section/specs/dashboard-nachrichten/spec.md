## ADDED Requirements

### Requirement: Dashboard-Section „Nachrichten"

Das Dashboard SHALL eine Section „Nachrichten" anzeigen, die als kollabierbares `Accordion` mit derselben Card-Optik wie die bestehenden Sections (`bg-brand-surface-card`, `border-t-4 border-brand-yellow`) gerendert wird. Die Section listet die ungelesenen Chat-Konversationen und Mitteilungen des Nutzers (neueste zuerst, gedeckelt auf max. 5 Einträge) und enthält einen Fußzeilen-Link „Zum Chat". Die Daten stammen aus `GET /api/chat/conversations` und `GET /api/chat/broadcasts`; es wird kein neuer Endpunkt eingeführt.

#### Scenario: Ungelesene Nachrichten vorhanden

- **WHEN** ein eingeloggter Nutzer das Dashboard öffnet und mindestens eine Konversation `unreadCount > 0` oder eine Mitteilung `isRead=false && isSent=false` hat
- **THEN** zeigt die Section „Nachrichten" die entsprechenden Einträge (max. 5, neueste zuerst) mit Titel/Absender als `DashboardRow`
- **THEN** führt ein Klick auf einen Konversations-Eintrag nach `/chat` und auf einen Mitteilungs-Eintrag nach `/chat?tab=broadcasts`

#### Scenario: Keine ungelesenen Nachrichten

- **WHEN** ein Nutzer das Dashboard öffnet und weder ungelesene Konversationen noch Mitteilungen hat
- **THEN** zeigt die Section einen dezenten Leerzustand („Keine ungelesenen Nachrichten")
- **THEN** bleibt der Fußzeilen-Link „Zum Chat" erreichbar

#### Scenario: Live-Aktualisierung

- **WHEN** während geöffnetem Dashboard ein `chat:new-message`- oder `chat:new-broadcast`-Event eintrifft
- **THEN** aktualisiert die Section ihre Liste ohne manuelles Neuladen der Seite
