## MODIFIED Requirements

### Requirement: Push bei Trainings-Ereignissen
Das System SHALL allen Mitgliedern des betroffenen Teams und deren Elternteilen eine Push Notification senden, wenn eine Trainingseinheit abgesagt oder verschoben wird — sofern Push für Kategorie `trainings` nicht deaktiviert. Die Notification-`url` MUSS auf den konkreten Trainingstermin in der Termine-Seite zeigen (`/termine?focus=training-<id>`), damit der Empfänger direkt zu- oder absagen kann. Für gelöschte Einheiten oder gelöschte Serien (kein navigierbarer Termin mehr) zeigt die `url` auf `/termine`.

#### Scenario: Einzelne Session abgesagt
- **WHEN** ein Trainer eine einzelne Trainingseinheit über `DELETE /api/training-sessions/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training abgesagt"
- **THEN** zeigt der Klick-Link auf `/termine` (kein `focus`, die Session ist gelöscht)

#### Scenario: Session verschoben
- **WHEN** ein Trainer eine Einheit über `PUT /api/training-sessions/{id}` aktualisiert (Zeit oder Ort geändert)
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training geändert"
- **THEN** zeigt der Klick-Link auf `/termine?focus=training-<id>` der geänderten Einheit

#### Scenario: Ganze Serie gelöscht
- **WHEN** ein Trainer eine gesamte Trainingsserie über `DELETE /api/training-series/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Trainingsserie gelöscht"
- **THEN** zeigt der Klick-Link auf `/termine`

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Trainings-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `trainings`
- **THEN** erhält dieser Nutzer keine Push Notification
