## ADDED Requirements

### Requirement: Videoliste abrufen
Alle authentifizierten Nutzer SHALL via `GET /api/videos` die Videoliste abrufen können. Die Antwort MUST `id`, `title`, `youtube_id`, `game_date`, `description`, `team_id`, `visibility`, `created_by_name` enthalten. Videos ohne Zugriffsberechtigung DÜRFEN NICHT in der Liste erscheinen.

#### Scenario: Spieler ruft vereinsweit-Videos ab
- **WHEN** ein Nutzer mit Rolle `spieler` `GET /api/videos` aufruft
- **THEN** erhält er alle Videos mit `visibility = vereinsweit`

#### Scenario: Gefilterte Liste nach Team (Stufe 2)
- **WHEN** ein Nutzer Mitglied von Team 5 ist und `GET /api/videos?team_id=5` aufruft
- **THEN** erhält er Videos mit `team_id = 5` und `visibility = team`, sofern er dem Team angehört

### Requirement: YouTube-Embed-URL
Der Server MUST den Embed-URL als `embed_url` in der Videolisten-Antwort mitliefern. Format: `https://www.youtube.com/embed/<youtube_id>`. Der `youtube_id` DARF NICHT direkt exponiert werden.

#### Scenario: Embed-URL in Antwort
- **WHEN** ein berechtigter Nutzer die Videoliste abruft
- **THEN** enthält jedes Video-Objekt `embed_url: "https://www.youtube.com/embed/<id>"`

#### Scenario: Kein direkter Zugriff auf YouTube-ID
- **WHEN** ein nicht eingeloggter Nutzer `GET /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 401

### Requirement: YouTube-Vorschaubild
Das Frontend SHALL für jedes Video ein Vorschaubild anzeigen. Die Thumbnail-URL MUST vom Server mitgeliefert werden (`https://img.youtube.com/vi/<id>/hqdefault.jpg`). Kein YouTube-API-Key erforderlich.

#### Scenario: Thumbnail-URL in Antwort
- **WHEN** ein berechtigter Nutzer die Videoliste abruft
- **THEN** enthält jedes Video-Objekt `thumbnail_url: "https://img.youtube.com/vi/<id>/hqdefault.jpg"`
