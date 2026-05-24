## MODIFIED Requirements

### Requirement: Mein Profil Sichtbarkeit
„Mein Profil" SHALL für alle eingeloggten Rollen außer `admin` sichtbar sein.

#### Scenario: Trainer sieht Mein Profil
- **WHEN** ein User mit Rolle `trainer` eingeloggt ist
- **THEN** erscheint „Mein Profil" in der Navigation

#### Scenario: Vorstand sieht Mein Profil
- **WHEN** ein User mit Rolle `vorstand` eingeloggt ist
- **THEN** erscheint „Mein Profil" in der Navigation

#### Scenario: Admin sieht Mein Profil nicht
- **WHEN** ein User mit Rolle `admin` eingeloggt ist
- **THEN** erscheint „Mein Profil" nicht in der Navigation

### Requirement: Mitglieder Sichtbarkeit
„Mitglieder" SHALL nur für `admin` und `vorstand` sichtbar sein.

#### Scenario: Trainer sieht Mitglieder nicht
- **WHEN** ein User mit Rolle `trainer` eingeloggt ist
- **THEN** erscheint „Mitglieder" nicht in der Navigation

#### Scenario: Admin sieht Mitglieder
- **WHEN** ein User mit Rolle `admin` eingeloggt ist
- **THEN** erscheint „Mitglieder" in der Navigation

### Requirement: Kader Sichtbarkeit und Zugriff
„Kader" SHALL für `admin`, `vorstand` und `trainer` sichtbar und zugänglich sein.

#### Scenario: Trainer sieht Kader in Navigation
- **WHEN** ein User mit Rolle `trainer` eingeloggt ist
- **THEN** erscheint „Kader" in der Navigation

#### Scenario: Trainer kann Kader-API aufrufen
- **WHEN** ein User mit Rolle `trainer` die Kader-API-Endpunkte aufruft
- **THEN** antwortet der Server mit 200 (nicht 403)

#### Scenario: Trainer hat vollen Kader-Schreibzugriff
- **WHEN** ein User mit Rolle `trainer` PUT/POST auf Kader-Endpunkte sendet
- **THEN** werden die Änderungen wie bei `vorstand` verarbeitet
