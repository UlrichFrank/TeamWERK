## ADDED Requirements

### Requirement: Push bei Beitrittsanfrage
Das System SHALL alle Admin-Nutzer per Push benachrichtigen, wenn eine neue Beitrittsanfrage eingeht — sofern Push für Kategorie `membership` nicht deaktiviert.

#### Scenario: Neue Beitrittsanfrage
- **WHEN** ein Interessent eine Beitrittsanfrage über `POST /api/auth/request-membership` stellt
- **THEN** erhalten alle Nutzer mit Rolle `admin` eine Push Notification „Neue Beitrittsanfrage"

#### Scenario: Admin mit deaktiviertem Push
- **WHEN** eine Beitrittsanfrage eingeht und ein Admin hat `push_enabled=0` für `membership`
- **THEN** erhält dieser Admin keine Push Notification
