## MODIFIED Requirements

### Requirement: Club master data management

Funktionalität unverändert. Nur die Route ändert sich.

**Vorher:** zugänglich unter `/admin/verein`  
**Nachher:** zugänglich unter `/admin/einstellungen?tab=verein` (Redirect von alter Route vorhanden)

#### Scenario: Non-admin cannot access club settings (updated)
- **WHEN** ein Nutzer ohne `admin`- oder `vorstand`-Rolle `/admin/einstellungen` aufruft
- **THEN** wird er abgewiesen (RoleRoute-Guard)

### Requirement: Season configuration

Erweitert um Bearbeitungsmöglichkeit.

**Neu:** Saisons können nach dem Anlegen bearbeitet werden (Name, Start-/Enddatum) über den neuen `PUT /api/admin/seasons/{id}` Endpoint.

#### Scenario: Edit existing season
- **WHEN** ein Admin eine bestehende Saison bearbeitet und speichert
- **THEN** werden Name und Datumsfelder aktualisiert; `is_active` bleibt unverändert
