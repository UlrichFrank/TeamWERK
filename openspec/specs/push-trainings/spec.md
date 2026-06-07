## ADDED Requirements

### Requirement: Push bei Trainings-Ereignissen
Das System SHALL allen Mitgliedern des betroffenen Teams und deren Elternteilen eine Push Notification senden, wenn eine Trainingseinheit abgesagt oder verschoben wird — sofern Push für Kategorie `trainings` nicht deaktiviert.

#### Scenario: Einzelne Session abgesagt
- **WHEN** ein Trainer eine einzelne Trainingseinheit über `DELETE /api/admin/training-sessions/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training abgesagt"

#### Scenario: Session verschoben
- **WHEN** ein Trainer eine Einheit über `PUT /api/admin/training-sessions/{id}` aktualisiert (Zeit oder Ort geändert)
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training geändert"

#### Scenario: Ganze Serie gelöscht
- **WHEN** ein Trainer eine gesamte Trainingsserie über `DELETE /api/admin/training-series/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Trainingsserie gelöscht"

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Trainings-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `trainings`
- **THEN** erhält dieser Nutzer keine Push Notification
