## MODIFIED Requirements

### Requirement: Mein Profil Sichtbarkeit
„Mein Profil" SHALL für alle eingeloggten Nutzer außer `admin` sichtbar sein.
Die Sichtbarkeit wird serverseitig in `policy.NavFor(claims)` berechnet und via
`GET /api/me` → `nav`-Array an das Frontend geliefert. Das Frontend DARF NICHT
`navModules[i].items[j].roles` oder lokale `const isXxx`-Prüfungen verwenden.

#### Scenario: Trainer sieht Mein Profil
- **WHEN** ein User mit Vereinsfunktion `trainer` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` einen Eintrag mit `route: "/profil"`

#### Scenario: Vorstand sieht Mein Profil
- **WHEN** ein User mit Vereinsfunktion `vorstand` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` einen Eintrag mit `route: "/profil"`

#### Scenario: Admin sieht Mein Profil nicht
- **WHEN** ein User mit `role == "admin"` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` KEINEN Eintrag mit `route: "/profil"`

---

### Requirement: Mitglieder Sichtbarkeit
„Mitglieder" SHALL nur für `admin` und Nutzer mit Vereinsfunktion `vorstand` sichtbar sein.
Berechnung serverseitig in `policy.NavFor(claims)`.

#### Scenario: Trainer sieht Mitglieder nicht
- **WHEN** ein User mit Vereinsfunktion `trainer` (aber nicht `vorstand`) eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` KEINEN Eintrag mit `route: "/mitglieder"`

#### Scenario: Admin sieht Mitglieder
- **WHEN** ein User mit `role == "admin"` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` einen Eintrag mit `route: "/mitglieder"`

#### Scenario: Vorstand sieht Mitglieder
- **WHEN** ein User mit Vereinsfunktion `vorstand` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` einen Eintrag mit `route: "/mitglieder"`

---

### Requirement: Kader Sichtbarkeit und Zugriff
„Kader" SHALL für `admin` und Nutzer mit Vereinsfunktion `vorstand`, `trainer` oder
`sportliche_leitung` sichtbar und zugänglich sein. Berechnung serverseitig in `policy.NavFor(claims)`.

#### Scenario: Trainer sieht Kader in Navigation
- **WHEN** ein User mit Vereinsfunktion `trainer` eingeloggt ist
- **THEN** enthält `GET /api/me` → `nav` einen Eintrag mit `route: "/kader"`

#### Scenario: Trainer kann Kader-API aufrufen
- **WHEN** ein User mit Vereinsfunktion `trainer` die Kader-API-Endpunkte aufruft
- **THEN** antwortet der Server mit 200 (nicht 403)

#### Scenario: Trainer hat vollen Kader-Schreibzugriff
- **WHEN** ein User mit Vereinsfunktion `trainer` PUT/POST auf Kader-Endpunkte sendet
- **THEN** werden die Änderungen wie bei `vorstand` verarbeitet
