## MODIFIED Requirements

### Requirement: Eigene Nachrichten können für alle gelöscht werden
Das System SHALL es dem Absender ermöglichen, eigene Nachrichten für alle Konversationsmitglieder zu löschen. Das Löschen ist ein Soft-Delete: Die Nachricht bleibt in der DB erhalten, wird aber als gelöscht markiert und nur als Placeholder angezeigt. Admins können alle Nachrichten löschen. Diese Aktion erfordert eine explizite Bestätigung (siehe „Bestätigungsdialog vor dem Löschen") und ist nur eine der beiden im Dialog angebotenen Optionen — die andere ist das private Ausblenden (siehe „Nachricht nur für sich selbst ausblenden").

#### Scenario: Löschen via Rechtsklick auf Desktop
- **WHEN** der Nutzer auf einer eigenen Nachrichten-Bubble einen Rechtsklick ausführt
- **THEN** enthält das Kontext-Menü den Eintrag „Löschen" (Trash2-Icon)

#### Scenario: Soft-Delete nach bestätigter „Für alle löschen"-Auswahl
- **WHEN** der Nutzer im Bestätigungsdialog „Für alle löschen" wählt
- **THEN** sendet das Frontend DELETE `/api/chat/messages/{id}`, das Backend setzt `deleted_at = CURRENT_TIMESTAMP`, die Nachrichtenliste wird neu geladen

#### Scenario: Gelöschte Nachricht als Placeholder anzeigen
- **WHEN** eine Nachricht mit gesetztem `deletedAt` in der Nachrichtenliste enthalten ist
- **THEN** wird anstelle des Nachrichtentexts ein Placeholder mit Trash2-Icon und Text „Nachricht gelöscht" in gedämpfter kursiver Formatierung angezeigt; kein Sender-Name, kein Kontext-Menü

#### Scenario: Kein Löschen fremder Nachrichten für alle durch reguläre Nutzer
- **WHEN** ein Nutzer ohne Admin-Rolle DELETE `/api/chat/messages/{id}` für eine fremde Nachricht aufruft
- **THEN** antwortet das Backend mit HTTP 403

#### Scenario: Admin kann alle Nachrichten für alle löschen
- **WHEN** ein Nutzer mit Rolle `admin` DELETE `/api/chat/messages/{id}` für eine beliebige Nachricht aufruft
- **THEN** setzt das Backend `deleted_at` und antwortet mit HTTP 204

#### Scenario: Bereits gelöschte Nachricht
- **WHEN** DELETE `/api/chat/messages/{id}` für eine bereits gelöschte Nachricht aufgerufen wird
- **THEN** antwortet das Backend idempotent mit HTTP 204

## ADDED Requirements

### Requirement: Bestätigungsdialog vor dem Löschen
Das System SHALL vor jedem Löschen einer Nachricht (Desktop-Kontextmenü und Mobile-Long-Press-Overlay) einen Bestätigungsdialog im Projekt-Modal-Stil anzeigen statt sofort zu löschen. Der Dialog bietet immer „Für mich löschen" und „Abbrechen"; „Für alle löschen" erscheint zusätzlich nur, wenn der Nutzer Absender der Nachricht oder Admin ist. Der „Löschen"-Eintrag im Kontextmenü/Overlay ist für jede nicht bereits gelöschte, sichtbare Nachricht verfügbar (nicht mehr nur für eigene Nachrichten/Admin), weil „Für mich löschen" jedem aktiven Mitglied offensteht.

#### Scenario: Dialog bei eigener Nachricht (Desktop und Mobile)
- **WHEN** der Nutzer „Löschen" für eine eigene Nachricht wählt (Rechtsklick-Menü auf Desktop oder Long-Press-Overlay auf Mobile)
- **THEN** öffnet sich der Bestätigungsdialog mit den Optionen „Für mich löschen", „Für alle löschen" und „Abbrechen"

#### Scenario: Dialog bei fremder Nachricht ohne Admin-Rolle
- **WHEN** ein Nutzer ohne Admin-Rolle „Löschen" für eine fremde Nachricht wählt
- **THEN** öffnet sich der Bestätigungsdialog nur mit „Für mich löschen" und „Abbrechen", ohne „Für alle löschen"

#### Scenario: Dialog bei fremder Nachricht als Admin
- **WHEN** ein Nutzer mit Rolle `admin` „Löschen" für eine fremde Nachricht wählt
- **THEN** öffnet sich der Bestätigungsdialog mit allen drei Optionen

#### Scenario: Abbrechen verändert nichts
- **WHEN** der Nutzer im Bestätigungsdialog „Abbrechen" wählt
- **THEN** schließt sich der Dialog ohne einen Request an das Backend, die Nachricht bleibt unverändert sichtbar

### Requirement: Nachricht nur für sich selbst ausblenden
Das System SHALL es jedem aktiven Mitglied einer Konversation ermöglichen, eine beliebige, nicht bereits (für alle) gelöschte Nachricht privat aus der eigenen Ansicht auszublenden, ohne dass sie für andere Mitglieder verändert wird. Ausgeblendete Nachrichten SHALL für den ausblendenden Nutzer aus Nachrichtenliste, Suchergebnissen, Ungelesen-Zähler/App-Badge und der Konversations-Vorschau (letzte Nachricht) verschwinden; als Zitat in einer späteren Antwort erscheinen sie für ihn als Placeholder „Nachricht gelöscht", identisch zur Anzeige einer für alle gelöschten Nachricht.

#### Scenario: Für mich löschen entfernt die Nachricht nur für den handelnden Nutzer
- **WHEN** Nutzer A im Bestätigungsdialog „Für mich löschen" wählt
- **THEN** sendet das Frontend POST `/api/chat/messages/{id}/hide`, das Backend legt eine `message_hides`-Zeile für Nutzer A an und antwortet mit HTTP 204; die Nachricht bleibt für alle anderen Konversationsmitglieder unverändert sichtbar

#### Scenario: Ausgeblendete Nachricht fehlt in Liste, Suche, Badge und Vorschau
- **WHEN** Nutzer A eine Nachricht für sich ausgeblendet hat
- **THEN** erscheint sie für Nutzer A weder in `GET /api/chat/conversations/{id}/messages`, noch in `GET /api/chat/search`-Treffern, noch zählt sie in seinem Ungelesen-Zähler/App-Badge, noch als `LastMessage` in seiner Konversationsliste

#### Scenario: Ausgeblendete Nachricht als Zitat
- **WHEN** eine spätere Nachricht auf eine von Nutzer A ausgeblendete Nachricht antwortet und Nutzer A diese spätere Nachricht lädt
- **THEN** zeigt das Zitat für Nutzer A den Placeholder „Nachricht gelöscht" statt des Originaltexts

#### Scenario: Jedes aktive Mitglied darf ausblenden, ohne Absender oder Admin zu sein
- **WHEN** ein Nutzer, der weder Absender noch Admin ist, aber aktives Mitglied der Konversation, POST `/api/chat/messages/{id}/hide` aufruft
- **THEN** antwortet das Backend mit HTTP 204

#### Scenario: Kein Ausblenden ohne aktive Mitgliedschaft
- **WHEN** ein Nutzer, der kein aktives Mitglied der Konversation ist, POST `/api/chat/messages/{id}/hide` aufruft
- **THEN** antwortet das Backend mit HTTP 403

#### Scenario: Kein Ausblenden einer bereits für alle gelöschten Nachricht
- **WHEN** POST `/api/chat/messages/{id}/hide` für eine bereits über `deleted_at` gelöschte Nachricht aufgerufen wird
- **THEN** antwortet das Backend mit HTTP 404

#### Scenario: Wiederholtes Ausblenden ist idempotent
- **WHEN** ein Nutzer POST `/api/chat/messages/{id}/hide` für eine von ihm bereits ausgeblendete Nachricht erneut aufruft
- **THEN** antwortet das Backend mit HTTP 204, ohne Fehler

#### Scenario: Multi-Geräte-Sync des Ausblendens
- **WHEN** ein Nutzer eine Nachricht auf einem Gerät ausblendet und auf einem zweiten Gerät dieselbe Konversation offen hat
- **THEN** verschwindet die Nachricht auch auf dem zweiten Gerät, ohne dass andere Konversationsmitglieder ein Live-Update erhalten
