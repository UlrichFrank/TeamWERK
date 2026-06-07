# chat-message-copy Specification

## Purpose
Nachrichtentext via Kontextmenü in die Systemzwischenablage kopieren.

## Requirements

### Requirement: Nachricht kopieren

Das Frontend SHALL im Nachrichtenkontextmenü einen „Kopieren"-Eintrag anzeigen, der den `body` der Nachricht via `navigator.clipboard.writeText` in die Zwischenablage schreibt. Der Eintrag MUSS für alle nicht-gelöschten Nachrichten sichtbar sein, unabhängig davon ob der User Absender ist. Nachrichten ohne Textinhalt (body leer, nur Bild) erhalten keinen „Kopieren"-Eintrag.

#### Scenario: Eigene Textnachricht kopieren

- **WHEN** ein User das Kontextmenü einer eigenen Textnachricht öffnet und auf „Kopieren" tippt
- **THEN** wird `msg.body` via `navigator.clipboard.writeText` in die Zwischenablage geschrieben
- **THEN** schließt sich das Kontextmenü

#### Scenario: Fremde Textnachricht kopieren

- **WHEN** ein User das Kontextmenü einer fremden Textnachricht öffnet und auf „Kopieren" tippt
- **THEN** wird `msg.body` in die Zwischenablage geschrieben
- **THEN** schließt sich das Kontextmenü

#### Scenario: Gelöschte Nachricht hat keinen Kopieren-Eintrag

- **WHEN** ein User das Kontextmenü einer gelöschten Nachricht öffnen würde
- **THEN** wird kein Kontextmenü angezeigt (bestehendes Verhalten: `if (msg.deletedAt) return`)

#### Scenario: Reine Bildnachricht hat keinen Kopieren-Eintrag

- **WHEN** eine Nachricht `body` leer ist und nur `imageUrl` gesetzt hat
- **THEN** wird der „Kopieren"-Eintrag im Kontextmenü nicht angezeigt
