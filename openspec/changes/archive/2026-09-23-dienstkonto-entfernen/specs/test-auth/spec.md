# Spec Delta

## MODIFIED Requirements

### Requirement: Nutzer-Löschung mit Cascade
Das System SHALL beim Löschen eines Nutzers alle abhängigen Daten in einer Transaktion entfernen. Ein Admin darf sich nicht selbst löschen.

#### Scenario: Selbstlöschung verboten
- **WHEN** Admin DELETE /api/admin/users/{eigene_id}
- **THEN** HTTP 400

#### Scenario: Cascade-Löschung
- **WHEN** Admin DELETE /api/admin/users/{andere_id}
- **THEN** HTTP 204; `refresh_tokens`, `duty_assignments`, `family_links` des Nutzers sind entfernt
