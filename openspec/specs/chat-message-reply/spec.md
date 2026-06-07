# chat-message-reply Specification

## Purpose
Nachrichten im Chat können auf eine spezifische Vorgängernachricht antworten (Reply), mit Quote-Darstellung der Ursprungsnachricht.

## Requirements

### Requirement: Nachrichten können beantwortet werden
Das System SHALL es ermöglichen, auf eine beliebige Nachricht in einem Gespräch zu antworten (Reply). Die Antwort-Nachricht referenziert die Ursprungsnachricht und zeigt deren Inhalt als Quote-Block.

#### Scenario: Reply via Swipe auf Mobile
- **WHEN** der Nutzer eine Nachrichten-Bubble auf einem Touchscreen mindestens 60px nach rechts wischt
- **THEN** öffnet sich die Reply-Leiste über dem Eingabefeld mit dem Namen des ursprünglichen Senders und einem Vorschau-Text der Ursprungsnachricht

#### Scenario: Reply via Rechtsklick auf Desktop
- **WHEN** der Nutzer auf einer Nachrichten-Bubble einen Rechtsklick ausführt
- **THEN** erscheint ein Kontext-Menü mit dem Eintrag „Antworten" (CornerUpLeft-Icon)

#### Scenario: Reply senden
- **WHEN** die Reply-Leiste aktiv ist und der Nutzer eine Nachricht sendet
- **THEN** wird die Nachricht mit `reply_to_id` der Ursprungsnachricht gespeichert und der API-Call enthält `replyToId`

#### Scenario: Reply-Quote in der Anzeige
- **WHEN** eine Nachricht mit `replyToId` im Chat angezeigt wird
- **THEN** erscheint oberhalb des Nachrichtentexts ein Quote-Block mit linker farbiger Border, dem Namen des ursprünglichen Senders und einem gekürzten Vorschautext der Ursprungsnachricht

#### Scenario: Reply auf gelöschte Nachricht
- **WHEN** eine Antwort-Nachricht angezeigt wird, deren Ursprungsnachricht gelöscht wurde
- **THEN** zeigt der Quote-Block den Text „[Nachricht gelöscht]" anstelle des ursprünglichen Inhalts

#### Scenario: Reply-Leiste schließen
- **WHEN** der Nutzer auf das X-Icon in der Reply-Leiste klickt
- **THEN** wird die Reply-Leiste geschlossen und die Nachricht wird ohne Reply-Referenz gesendet
