## ADDED Requirements

### Requirement: Konversationsliste nach letzter Aktivität sortiert

`GET /api/chat/conversations` SHALL die Konversationen des anfragenden Nutzers absteigend nach dem Zeitpunkt der letzten Aktivität zurückgeben — die zuletzt aktive Konversation zuerst. Die letzte Aktivität MUST der `sent_at`-Zeitpunkt der jüngsten Nachricht der Konversation sein; für Konversationen ohne Nachricht MUST als Sortierschlüssel `conversations.created_at` verwendet werden.

Diese Anforderung formalisiert bestehendes Verhalten und sichert es gegen Regression; sie ändert das Verhalten nicht.

#### Scenario: Neue Nachricht hebt Konversation an die Spitze

- **WHEN** in einer weiter unten stehenden Konversation eine neue Nachricht eintrifft und die Liste erneut geladen wird
- **THEN** steht diese Konversation an erster Stelle der zurückgegebenen Liste

#### Scenario: Konversation ohne Nachrichten wird nach Erstellzeit einsortiert

- **WHEN** eine Konversation noch keine Nachricht enthält
- **THEN** wird sie anhand von `created_at` in die nach letzter Aktivität absteigend sortierte Liste einsortiert
