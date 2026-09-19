## MODIFIED Requirements

### Requirement: Mitglieder Sichtbarkeit
„Mitglieder" SHALL für `admin` sowie für Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` sichtbar sein. Der Kassierer erreicht die Seite lesend (siehe `kassierer-member-zugriff`); Anlegen, Import und die Verwaltungs-Tabs bleiben an `manage_members` gebunden.

#### Scenario: Trainer sieht Mitglieder nicht
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` (ohne `vorstand`/`kassierer`) die Navigation lädt
- **THEN** enthält sie den Eintrag „Mitglieder" nicht

#### Scenario: Admin sieht Mitglieder
- **WHEN** ein Nutzer mit System-Rolle `admin` die Navigation lädt
- **THEN** enthält sie den Eintrag „Mitglieder"

#### Scenario: Vorstand sieht Mitglieder
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` die Navigation lädt
- **THEN** enthält sie den Eintrag „Mitglieder"

#### Scenario: Kassierer sieht Mitglieder
- **WHEN** ein Nutzer mit Vereinsfunktion `kassierer` die Navigation lädt
- **THEN** enthält sie den Eintrag „Mitglieder"
