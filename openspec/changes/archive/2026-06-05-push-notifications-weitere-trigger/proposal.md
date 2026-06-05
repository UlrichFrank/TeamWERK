## Why

Die Push-Infrastruktur (VAPID, `push_subscriptions`, `SendToUsers`) ist vollständig gebaut und wird für Mitfahrgelegenheiten bereits eingesetzt. Weitere wichtige Ereignisse — Trainingsabsagen, Mitgliedsanfragen, Datenänderungen, Dienst-Reminder — lösen bisher keine Benachrichtigungen aus. Nutzer erfahren von Änderungen erst beim nächsten App-Aufruf.

## What Changes

**Sofort ausgelöste Trigger (im jeweiligen Handler):**
- Training abgesagt oder verschoben (Ort/Zeit) → alle Teammitglieder des betroffenen Teams
- Neue Trainingssession oder -serie angelegt → alle Teammitglieder
- Neue Mitgliedsanfrage eingegangen → alle Admins
- Datenänderung durch Mitglied beantragt → zuständiger Trainer + alle Admins

**Geplante Trigger (Scheduler-Job, täglich ausgeführt):**
- Dienst-Reminder: User mit Dienst am Folgetag erhält am Abend (18:00 Uhr) eine Push Notification
- Dienst-Lücken: Trainer/Admin wird 2 Tage vor einem Event benachrichtigt, wenn noch offene Slots bestehen

## Capabilities

### New Capabilities

- `training-notifications`: Push-Trigger bei Trainingsänderungen, -absagen und neuen Terminen
- `admin-notifications`: Push-Trigger bei Mitgliedsanfragen und Datenänderungs-Requests
- `duty-reminder-push`: Geplante Push-Reminder für bevorstehende Dienste (Tag davor)
- `duty-gap-alert`: Geplanter Push-Alert an Trainer/Admin bei unbesetzten Slots (2 Tage vorher)

### Modified Capabilities

- `training-sessions`: `PUT /api/training-sessions/{id}` und Cancel-Endpoint lösen `notifications.SendToUsers` aus
- `training-series`: `POST /api/training-series` löst Notification an Teammitglieder aus
- `duty-scheduler`: bestehender Scheduler-Job erhält zwei neue Job-Typen

## Impact

- **Backend:** Erweiterungen in `internal/trainings/handler.go`, `internal/auth/handler.go` (membership), `internal/members/handler.go` (change-request), `internal/scheduler/` (neue Job-Typen)
- **Keine neuen API-Routen**
- **Keine neuen DB-Tabellen** (nutzt bestehende `push_subscriptions`, `duty_assignments`, `duty_slots`)
- **Keine Frontend-Änderungen**
- **Rollen:** Trainer/Admin erhalten Mitgliedsanfrage- und Lücken-Alerts; alle User erhalten Team-Trainings-Notifications und eigene Dienst-Reminder
