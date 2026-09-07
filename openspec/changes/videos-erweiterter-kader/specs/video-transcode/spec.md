## MODIFIED Requirements

### Requirement: Push-Notification bei Fertigstellung

Bei `status='ready'` SHALL der Worker — nicht-blockierend in einer Goroutine — Push-Notifications an folgende Empfänger senden: Hochladenden (`created_by`), alle aktiven Spieler des Teams (`team_memberships`), alle Eltern dieser Spieler (`family_links`), alle Trainer des Teams (`team_trainers`) sowie die **Mitglieder des erweiterten Kaders des Teams und deren Elternteile**. Inhalt: Titel `"Neues Video: {team_name}"`, Body `"{title}"`, Ziel-URL `/videos/{id}`.

Der Empfängerkreis SHALL deckungsgleich mit der Sicht-Berechtigung aus `video-management` sein: es SHALL niemand über ein Video benachrichtigt werden, das er anschließend nicht öffnen kann.

#### Scenario: Empfängerkreis
- **WHEN** ein Video für Team `U17` fertig wird
- **THEN** erhalten Hochladender, alle aktiven U17-Spieler, deren Eltern und alle U17-Trainer eine Push-Notification

#### Scenario: Erweiterter Kader im Empfängerkreis
- **WHEN** ein Video für Team `U17` fertig wird und ein Mitglied steht nur im erweiterten Kader dieser Mannschaft
- **THEN** erhalten dieses Mitglied und seine über `family_links` verknüpften Elternteile dieselbe Push-Notification

#### Scenario: Push schlägt einzeln fehl
- **WHEN** Push-Versand an einen Empfänger fehlschlägt
- **THEN** wird der Transcode-Erfolg nicht rückgängig gemacht, der Fehler wird geloggt
