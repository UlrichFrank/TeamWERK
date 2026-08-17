## ADDED Requirements

### Requirement: Zentrale Berechnung der Chat-Unread-Anteile

Das Frontend SHALL die Ungelesen-Zahlen an genau einer Stelle berechnen: einer reinen
Funktion `chatUnreadCounts(conversations, broadcasts)` in `web/src/lib/chatUnread.ts`, die
`{ conversations, broadcasts, total }` liefert. `conversations` ist die Summe aller
`unreadCount`-Felder, `broadcasts` die Anzahl der Mitteilungen mit `isRead === false` UND
`isSent === false`, `total` die Summe beider Anteile.

Alle Anzeigestellen (Navigation, Dashboard, `/chat`) SHALL diese Funktion nutzen und die
Formel NICHT erneut lokal ausschreiben. Die Funktion SHALL ohne React-Render und ohne
Netzwerkzugriff testbar sein.

#### Scenario: Konversationen und Mitteilungen werden getrennt ausgewiesen

- **WHEN** `chatUnreadCounts` mit zwei Konversationen (`unreadCount: 2` und `unreadCount: 1`) und drei ungelesenen, nicht selbst gesendeten Mitteilungen aufgerufen wird
- **THEN** liefert sie `{ conversations: 3, broadcasts: 3, total: 6 }`

#### Scenario: Eigene Mitteilung zählt nicht mit

- **WHEN** `chatUnreadCounts` mit einer Mitteilung aufgerufen wird, die `isRead: false` und `isSent: true` trägt
- **THEN** ist `broadcasts` 0 und die Mitteilung fließt nicht in `total` ein

#### Scenario: Leere Listen

- **WHEN** `chatUnreadCounts([], [])` aufgerufen wird
- **THEN** liefert sie `{ conversations: 0, broadcasts: 0, total: 0 }`

### Requirement: Badge am Navigations-Modul-Header

Die Sidebar-Navigation SHALL an jedem Modul-Header die Summe der Badges seiner **sichtbaren**
Einträge anzeigen, sofern diese Summe größer als 0 ist. Die Zuordnung Route → Badge-Wert
SHALL über eine Map (`navBadges`) erfolgen, nicht über Sonderfall-Vergleiche auf einzelne
Routen; der bestehende Sonderfall für `/chat` am Nav-Item SHALL auf dieselbe Map umgestellt
werden.

Der Wert für `/chat` ist `total` aus `chatUnreadCounts` (Konversationen + Mitteilungen).

Die Summe SHALL unabhängig davon berechnet werden, ob das Modul aufgeklappt ist — das
eingeklappte Modul rendert seine Einträge nicht. Der Badge am Modul-Header SHALL auch dann
sichtbar bleiben, wenn das Modul offen ist und der Eintrag seinen eigenen Badge zeigt.

Einträge, die für den Nutzer nicht sichtbar sind (nicht in `navRoutes` enthalten), SHALL
NICHT in die Summe eingehen.

#### Scenario: Eingeklapptes Modul zeigt die Zahl

- **WHEN** ein Nutzer 3 ungelesene Nachrichten hat, das Modul „Verein" eingeklappt ist und der Eintrag „Nachrichten" folglich nicht gerendert wird
- **THEN** zeigt der Modul-Header „Verein" den Badge `3`

#### Scenario: Offenes Modul zeigt beide Badges

- **WHEN** ein Nutzer 3 ungelesene Nachrichten hat und das Modul „Verein" aufgeklappt ist
- **THEN** zeigt der Modul-Header „Verein" den Badge `3`
- **THEN** zeigt der Eintrag „Nachrichten" zusätzlich den Badge `3`

#### Scenario: Nutzer ohne Chat-Zugriff

- **WHEN** `/chat` nicht in den vom Server gelieferten `navRoutes` des Nutzers enthalten ist
- **THEN** zeigt der Modul-Header „Verein" keinen Badge, unabhängig vom zuletzt geladenen Unread-Wert

#### Scenario: Nichts ungelesen

- **WHEN** die Summe der Badges eines Moduls 0 ist
- **THEN** zeigt der Modul-Header keinen Badge (kein `0`)

### Requirement: Hinweis-Punkt am Hamburger-Icon auf Mobil

Der mobile Header SHALL am Menü-Button (`<Menu>`) einen Hinweis-Punkt anzeigen, sobald
mindestens ein **Modul-Header-Badge** größer als 0 ist — also über dieselbe, bereits gegen
`navRoutes` gefilterte Summe, die die Modul-Header verwenden. Eine Auswertung roh über
`navBadges` SHALL NICHT erfolgen, da sie die Sichtbarkeitsprüfung umginge.

Der Punkt SHALL **keine Zahl** tragen: der Button steht für das gesamte Menü, nicht für eine
einzelne Route, und muss über weitere `navBadges`-Einträge hinweg gültig bleiben.

Der Punkt SHALL in `brand-danger` gerendert werden, nicht in `brand-yellow` — der mobile Header
ist `bg-brand-white`, auf dem der Gelbton der übrigen Badges nicht wahrnehmbar ist.

Das `aria-label` des Buttons SHALL bei vorhandenem Unread die konkrete Zahl nennen, damit die
Information für Screenreader nicht ausschließlich visuell existiert. Ohne Unread SHALL es auf
den bisherigen Text zurückfallen.

Der Punkt SHALL nur im mobilen Header erscheinen; auf Desktop ist die Sidebar dauerhaft
sichtbar und der Modul-Header trägt die Information.

#### Scenario: Ungelesene Nachricht bei geschlossener Sidebar

- **WHEN** ein Nutzer auf Mobil die Seite `/kalender` geöffnet hat, die Sidebar geschlossen ist und 3 ungelesene Nachrichten vorliegen
- **THEN** trägt der Menü-Button einen Hinweis-Punkt
- **THEN** lautet sein `aria-label` „Menü öffnen, 3 ungelesene Nachrichten"

#### Scenario: Punkt trägt keine Zahl

- **WHEN** ein Nutzer 12 ungelesene Nachrichten hat
- **THEN** zeigt der Menü-Button weiterhin nur einen Punkt ohne Zahl

#### Scenario: Nichts ungelesen

- **WHEN** alle Modul-Header-Badges 0 sind
- **THEN** trägt der Menü-Button keinen Punkt
- **THEN** lautet sein `aria-label` „Menü öffnen"

#### Scenario: Nutzer ohne Chat-Zugriff

- **WHEN** `/chat` nicht in den `navRoutes` des Nutzers enthalten ist
- **THEN** trägt der Menü-Button keinen Punkt, unabhängig vom zuletzt geladenen Unread-Wert

### Requirement: Badge am Tab „Chats"

Die Tab-Leiste auf `/chat` SHALL am Tab „Chats" den Anteil `conversations` aus
`chatUnreadCounts` anzeigen, sofern er größer als 0 ist — analog zum bestehenden Badge am Tab
„Mitteilungen", der dessen Anteil zeigt.

Der Tab „Chats" SHALL NICHT die Gesamtsumme anzeigen: die beiden Tab-Badges SHALL die Zahl an
der Seitenüberschrift `<h1>` partitionieren, sich also zu ihr aufsummieren.

#### Scenario: Beide Tabs tragen einen Anteil

- **WHEN** der Nutzer 3 ungelesene Nachrichten in Konversationen und 2 ungelesene Mitteilungen hat
- **THEN** zeigt die Überschrift `5`, der Tab „Chats" `3` und der Tab „Mitteilungen" `2`

#### Scenario: Nur Mitteilungen ungelesen

- **WHEN** der Nutzer keine ungelesenen Konversationen, aber 2 ungelesene Mitteilungen hat
- **THEN** zeigt der Tab „Chats" keinen Badge
- **THEN** zeigt der Tab „Mitteilungen" den Badge `2`

#### Scenario: Nutzer liest eine Konversation

- **WHEN** der Nutzer eine Konversation mit 3 ungelesenen Nachrichten öffnet und diese dadurch als gelesen markiert werden
- **THEN** sinkt der Badge am Tab „Chats" entsprechend und verschwindet bei 0
