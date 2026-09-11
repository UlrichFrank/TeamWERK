## ADDED Requirements

### Requirement: JWT-Secret hat eine Mindeststärke

Der Server MUST den Start verweigern, wenn `JWT_SECRET` kürzer als 32 Byte ist oder dem Beispielwert der Konfigurationsvorlage entspricht. Die Fehlermeldung MUST den Grund und einen Weg zur Erzeugung eines geeigneten Werts nennen.

#### Scenario: Kurzes Secret
- **WHEN** der Server mit einem `JWT_SECRET` von 16 Byte startet
- **THEN** bricht er mit einer Fehlermeldung ab

#### Scenario: Beispielwert
- **WHEN** der Server mit dem Wert aus der Vorlage startet
- **THEN** bricht er mit einer Fehlermeldung ab
