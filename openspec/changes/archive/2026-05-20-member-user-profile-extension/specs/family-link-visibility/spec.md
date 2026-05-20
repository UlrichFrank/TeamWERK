## ADDED Requirements

### Requirement: Elternteile sehen nur eigene Family-Links
Das System SHALL bei `GET /api/members/{id}/parents` für Nutzer mit Rolle `elternteil` ausschließlich die eigenen Links zurückgeben (WHERE parent_user_id = claims.UserID). Andere Elternteile sind nicht sichtbar.

#### Scenario: Elternteil ruft Elternliste eines Kindes ab (eigener Link)
- **WHEN** Nutzer mit Rolle `elternteil` ruft `GET /api/members/{id}/parents` auf und ist selbst verknüpft
- **THEN** Response enthält nur den eigenen User-Eintrag (Liste mit einem Element)

#### Scenario: Elternteil ruft Elternliste eines fremden Kindes ab
- **WHEN** Nutzer mit Rolle `elternteil` ruft `GET /api/members/{id}/parents` auf und ist NICHT verknüpft
- **THEN** Response enthält leere Liste (HTTP 200, items: [])

#### Scenario: Elternteil ruft Elternliste auf mit zwei Elternteilen
- **WHEN** Kind hat zwei Elternteile A und B; Elternteil A ruft `GET /api/members/{id}/parents` auf
- **THEN** Response enthält nur Elternteil A, nicht B

### Requirement: Andere Rollen sehen alle Family-Links
Nutzer mit Rollen `admin`, `trainer`, `spieler`, `vorstand` DÜRFEN alle `family_links` eines Mitglieds sehen.

#### Scenario: Admin sieht alle Elternteile
- **WHEN** Admin `GET /api/members/{id}/parents`
- **THEN** Response enthält alle verknüpften Elternteile

#### Scenario: Trainer sieht alle Elternteile
- **WHEN** Trainer `GET /api/members/{id}/parents`
- **THEN** Response enthält alle verknüpften Elternteile
