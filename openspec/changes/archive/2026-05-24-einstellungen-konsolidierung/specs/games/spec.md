## MODIFIED Requirements

### Requirement: Altersklassen-Regeln-Verwaltung (Route-Änderung)

Funktionalität unverändert. Nur die Route ändert sich.

**Vorher:** `/admin/altersklassen`  
**Nachher:** `/admin/einstellungen?tab=altersklassen` (Redirect von alter Route vorhanden)

#### Scenario: Altersklassen-Regeln bearbeiten (updated path)
- **WHEN** ein Admin die Altersklassen-Regeln bearbeiten möchte
- **THEN** findet er sie unter `/admin/einstellungen` im Tab „Altersklassen"
- **AND** die Inline-Edit-Tabelle (Halbzeit + Pause pro Klasse) funktioniert wie bisher
