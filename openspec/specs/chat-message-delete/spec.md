# chat-message-delete Specification

## Purpose
Eigene Chat-Nachrichten können soft-gelöscht werden. Admins können alle Nachrichten löschen.

## Requirements

### Requirement: Eigene Nachrichten können gelöscht werden
Das System SHALL es dem Absender ermöglichen, eigene Nachrichten zu löschen. Das Löschen ist ein Soft-Delete: Die Nachricht bleibt in der DB erhalten, wird aber als gelöscht markiert und nur als Placeholder angezeigt. Admins können alle Nachrichten löschen.

#### Scenario: Löschen via Rechtsklick auf Desktop
- **WHEN** der Nutzer auf einer eigenen Nachrichten-Bubble einen Rechtsklick ausführt
- **THEN** enthält das Kontext-Menü den Eintrag „Löschen" (Trash2-Icon)

#### Scenario: Soft-Delete durchführen
- **WHEN** der Nutzer „Löschen" im Kontext-Menü wählt
- **THEN** sendet das Frontend DELETE `/api/chat/messages/{id}`, das Backend setzt `deleted_at = CURRENT_TIMESTAMP`, die Nachrichtenliste wird neu geladen

#### Scenario: Gelöschte Nachricht als Placeholder anzeigen
- **WHEN** eine Nachricht mit gesetztem `deletedAt` in der Nachrichtenliste enthalten ist
- **THEN** wird anstelle des Nachrichtentexts ein Placeholder mit Trash2-Icon und Text „Nachricht gelöscht" in gedämpfter kursiver Formatierung angezeigt; kein Sender-Name, kein Kontext-Menü

#### Scenario: Kein Löschen fremder Nachrichten durch reguläre Nutzer
- **WHEN** ein Nutzer ohne Admin-Rolle DELETE `/api/chat/messages/{id}` für eine fremde Nachricht aufruft
- **THEN** antwortet das Backend mit HTTP 403

#### Scenario: Admin kann alle Nachrichten löschen
- **WHEN** ein Nutzer mit Rolle `admin` DELETE `/api/chat/messages/{id}` für eine beliebige Nachricht aufruft
- **THEN** setzt das Backend `deleted_at` und antwortet mit HTTP 204

#### Scenario: Bereits gelöschte Nachricht
- **WHEN** DELETE `/api/chat/messages/{id}` für eine bereits gelöschte Nachricht aufgerufen wird
- **THEN** antwortet das Backend idempotent mit HTTP 204
