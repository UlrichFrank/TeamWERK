## MODIFIED Requirements

### Requirement: Konversationsliste zeigt nur aktive Gespräche
Die Konversationsliste eines Users SHALL nur Gespräche anzeigen, bei denen der User aktiv ist (`left_at IS NULL`). Dies gilt bereits; die Änderung besteht darin, dass auch Direct Chats über `left_at` ausgeblendet werden können (bisher war `left_at` nur für Gruppen semantisch genutzt).

#### Scenario: Direct Chat nach Löschen nicht mehr sichtbar
- **WHEN** ein User einen Direct Chat gelöscht hat (`left_at` gesetzt)
- **THEN** erscheint dieser Chat nicht mehr in `GET /api/chat/conversations`

#### Scenario: Gruppe nach Verlassen nicht mehr sichtbar
- **WHEN** ein User eine Gruppe verlassen hat (`left_at` gesetzt)
- **THEN** erscheint diese Gruppe nicht mehr in `GET /api/chat/conversations`
