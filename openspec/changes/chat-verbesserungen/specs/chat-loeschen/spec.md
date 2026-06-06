## ADDED Requirements

### Requirement: Gespräch für sich selbst löschen
Ein User SHALL ein Gespräch (Direct oder Gruppe) für sich selbst löschen können. Das Gespräch verschwindet aus seiner Liste. Andere Teilnehmer sind nicht betroffen. Löscht der letzte aktive Teilnehmer das Gespräch, werden alle zugehörigen Daten (Conversation, Messages, Message Reads, Members) bereinigt.

#### Scenario: Direct Chat löschen
- **WHEN** ein User `DELETE /api/chat/conversations/{id}` aufruft
- **THEN** wird `conversation_members.left_at` für diesen User gesetzt
- **THEN** erscheint das Gespräch nicht mehr in seiner Konversationsliste
- **THEN** der andere Teilnehmer sieht das Gespräch weiterhin

#### Scenario: Letzter Teilnehmer löscht Direct Chat
- **WHEN** beide User das Gespräch gelöscht haben (beide `left_at` gesetzt)
- **THEN** wird die Conversation inklusive aller Messages, Message Reads und Members gelöscht

#### Scenario: Gruppe verlassen entspricht Löschen
- **WHEN** ein User eine Gruppe verlässt (existing `DELETE /api/chat/conversations/{id}/members/me`) oder löscht
- **THEN** gilt dieselbe Cleanup-Logik: wenn alle Mitglieder `left_at` gesetzt haben, werden alle Daten gelöscht

### Requirement: Broadcast-Mitteilung für sich selbst löschen
Ein User SHALL eine Broadcast-Mitteilung für sich selbst ausblenden können. Die Mitteilung verschwindet aus seiner Liste. Andere Empfänger sind nicht betroffen. Haben alle Empfänger die Mitteilung ausgeblendet, werden alle zugehörigen Daten (Broadcast, Broadcast Reads) bereinigt.

#### Scenario: Broadcast ausblenden
- **WHEN** ein User `DELETE /api/chat/broadcasts/{id}` aufruft
- **THEN** wird `broadcast_reads.hidden_at` für diesen User gesetzt
- **THEN** erscheint die Mitteilung nicht mehr in seiner Broadcasts-Liste
- **THEN** andere Empfänger sehen die Mitteilung weiterhin

#### Scenario: Letzter Empfänger blendet Broadcast aus
- **WHEN** alle `broadcast_reads`-Einträge für einen Broadcast `hidden_at` gesetzt haben
- **THEN** wird der Broadcast inklusive aller Broadcast Reads gelöscht

#### Scenario: Unberechtigter Zugriff
- **WHEN** ein User `DELETE /api/chat/broadcasts/{id}` aufruft für eine Mitteilung, die nicht in seiner `broadcast_reads`-Tabelle existiert
- **THEN** antwortet der Server mit 403 Forbidden
