## MODIFIED Requirements

### Requirement: Dashboard-Section „Nachrichten"

Das Dashboard SHALL eine Section „Nachrichten" anzeigen, die als kollabierbares `Accordion` mit derselben Card-Optik wie die bestehenden Sections (`bg-brand-surface-card`, `border-t-4 border-brand-yellow`) gerendert wird. Die Section listet die ungelesenen Chat-Konversationen und Mitteilungen des Nutzers (neueste zuerst, gedeckelt auf max. 5 Einträge) und enthält einen Fußzeilen-Link „Zum Chat". Die Daten stammen aus `GET /api/chat/conversations` und `GET /api/chat/broadcasts`; es wird kein neuer Endpunkt eingeführt.

Der Abruf der Daten und das Abonnement der Chat-Events SHALL **außerhalb** des kollabierbaren Bereichs liegen (in `DashboardPage`, nicht in der Section-Komponente), damit beides auch bei eingeklappter Section läuft. Die Section-Komponente SHALL ihre Einträge als Prop erhalten und selbst keinen Abruf mehr auslösen.

Eine Konversations-Zeile SHALL auf `/chat?conv=<id>` verlinken, nicht auf das nackte `/chat` —
denselben Deep-Link-Parameter, den der Chat-Push nutzt. Andernfalls landet der Nutzer zwar im
Chat, die angeklickte Konversation bleibt aber ungeöffnet, wird also nicht als gelesen markiert,
und sämtliche Badges (Section-Header, Nav-Modul, Hamburger) stehen unverändert weiter, obwohl
der Nutzer die Nachricht gerade angeklickt hat.

Der Header der Section SHALL die Gesamtzahl der ungelesenen Nachrichten als Badge anzeigen, sofern sie größer als 0 ist — berechnet als `total` aus `chatUnreadCounts` über die **ungefilterten** Listen. Der Badge SHALL NICHT aus der Anzahl der angezeigten Zeilen abgeleitet werden: eine Zeile kann mehrere ungelesene Nachrichten bündeln, und die Liste ist auf 5 Einträge gedeckelt. Der Badge SHALL unabhängig vom Aufklapp-Zustand sichtbar sein.

#### Scenario: Ungelesene Nachrichten vorhanden

- **WHEN** ein eingeloggter Nutzer das Dashboard öffnet und mindestens eine Konversation `unreadCount > 0` oder eine Mitteilung `isRead=false && isSent=false` hat
- **THEN** zeigt die Section „Nachrichten" die entsprechenden Einträge (max. 5, neueste zuerst) mit Titel/Absender als `DashboardRow`
- **THEN** führt ein Klick auf einen Konversations-Eintrag nach `/chat?conv=<id>` und auf einen Mitteilungs-Eintrag nach `/chat?tab=broadcasts`

#### Scenario: Klick auf eine Konversations-Zeile öffnet die Konversation

- **WHEN** ein Nutzer im Dashboard auf die Zeile einer Konversation mit ungelesenen Nachrichten klickt
- **THEN** öffnet `/chat` genau diese Konversation, markiert sie als gelesen und alle Badges (Section-Header, Nav-Modul-Header, Hamburger-Punkt) fallen entsprechend

#### Scenario: Keine ungelesenen Nachrichten

- **WHEN** ein Nutzer das Dashboard öffnet und weder ungelesene Konversationen noch Mitteilungen hat
- **THEN** zeigt die Section einen dezenten Leerzustand („Keine ungelesenen Nachrichten")
- **THEN** bleibt der Fußzeilen-Link „Zum Chat" erreichbar
- **THEN** zeigt der Section-Header keinen Badge

#### Scenario: Live-Aktualisierung

- **WHEN** während geöffnetem Dashboard ein `chat:new-message`- oder `chat:new-broadcast`-Event eintrifft
- **THEN** aktualisiert die Section ihre Liste ohne manuelles Neuladen der Seite

#### Scenario: Badge bei eingeklappter Section

- **WHEN** ein Nutzer das Dashboard auf Mobil öffnet, wo die Section „Nachrichten" per Default eingeklappt ist, und 3 ungelesene Nachrichten hat
- **THEN** zeigt der Header der Section den Badge `3`, obwohl der Section-Inhalt nicht gerendert wird

#### Scenario: Live-Aktualisierung bei eingeklappter Section

- **WHEN** die Section „Nachrichten" eingeklappt ist und ein `chat:new-message`-Event eintrifft
- **THEN** erhöht sich der Badge am Header, ohne dass die Section aufgeklappt werden muss

#### Scenario: Badge zählt Nachrichten, nicht Zeilen

- **WHEN** ein Nutzer 7 ungelesene Konversationen mit je 2 ungelesenen Nachrichten hat und die Liste dadurch auf 5 Einträge gedeckelt wird
- **THEN** zeigt der Header den Badge `14`
- **THEN** zeigt die Section weiterhin höchstens 5 Einträge
