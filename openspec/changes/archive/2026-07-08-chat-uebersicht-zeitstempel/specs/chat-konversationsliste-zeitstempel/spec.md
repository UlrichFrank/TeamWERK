## ADDED Requirements

### Requirement: Abstands-abhängiges Aktivitäts-Label pro Konversation

Die Chat-Übersichtsliste SHALL für jede Konversation mit mindestens einer Nachricht ein Aktivitäts-Label anzeigen, dessen Format sich nach dem Abstand des Zeitpunkts der letzten Nachricht (`lastMessage.sentAt`) zum aktuellen Zeitpunkt richtet. Der Abstand MUST in ganzen Kalendertagen (lokale Zeitzone) gemessen werden.

Die Buckets sind:
- Abstand = 0 Kalendertage (heute): Uhrzeit im Format `HH:MM` (24h, `de-DE`), z. B. `14:30`.
- Abstand = 1 Kalendertag (gestern): das Wort `Gestern`.
- Abstand 2–6 Kalendertage: ausgeschriebener Wochentag (`de-DE`), z. B. `Montag`.
- Abstand ≥ 7 Kalendertage: numerisches Datum `TT.MM.JJ`, z. B. `08.07.26`.

Die Grenze zwischen Wochentag und Datum MUST strikt bei < 7 Kalendertagen liegen, damit ein Wochentag nie mit dem heutigen Wochentag der Vorwoche kollidiert.

#### Scenario: Letzte Nachricht von heute

- **WHEN** die letzte Nachricht einer Konversation heute um 14:30 gesendet wurde
- **THEN** zeigt das Label `14:30`

#### Scenario: Letzte Nachricht von gestern

- **WHEN** die letzte Nachricht am vorherigen Kalendertag gesendet wurde
- **THEN** zeigt das Label `Gestern`

#### Scenario: Letzte Nachricht vor 3 Kalendertagen

- **WHEN** die letzte Nachricht vor 3 Kalendertagen (an einem Montag) gesendet wurde
- **THEN** zeigt das Label den Wochentag `Montag`

#### Scenario: Letzte Nachricht vor genau 7 Kalendertagen

- **WHEN** die letzte Nachricht vor genau 7 Kalendertagen gesendet wurde
- **THEN** zeigt das Label das numerische Datum im Format `TT.MM.JJ` (nicht den Wochentag)

#### Scenario: Konversation ohne Nachricht

- **WHEN** eine Konversation noch keine Nachricht enthält (`lastMessage` ist `null`)
- **THEN** wird kein Aktivitäts-Label angezeigt

### Requirement: Listeneintrags-Layout nach Messenger-Konvention

Der Eintrag einer Konversation in der Übersichtsliste SHALL das Aktivitäts-Label oben rechts auf Höhe des Konversationsnamens anzeigen und das Unread-Badge unten rechts auf Höhe der Nachrichtenvorschau. Das Label MUST als dezenter Sekundärtext dargestellt werden (`brand-*`-Token, kein Raw-Tailwind) und darf den Konversationsnamen nicht verdrängen (Name bleibt `truncate`).

#### Scenario: Ungelesene Konversation mit letzter Aktivität

- **WHEN** eine Konversation ungelesene Nachrichten hat und eine letzte Nachricht existiert
- **THEN** erscheint das Aktivitäts-Label oben rechts (Namenshöhe) und das Unread-Badge unten rechts (Vorschauhöhe)

#### Scenario: Gelesene Konversation

- **WHEN** eine Konversation keine ungelesenen Nachrichten hat
- **THEN** erscheint das Aktivitäts-Label oben rechts und kein Unread-Badge
