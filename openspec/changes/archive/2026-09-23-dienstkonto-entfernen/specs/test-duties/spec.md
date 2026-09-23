# Spec Delta

## MODIFIED Requirements

### Requirement: Slot-Kapazität bei Claim/Unclaim
Das System SHALL `slots_filled` beim Claim erhöhen und beim Unclaim verringern. Ein voller Slot (`slots_filled >= slots_total`) kann nicht geclaimt werden. Ein bereits erfüllter Slot kann nicht unclaimed werden.

#### Scenario: Freien Slot claimen
- **WHEN** User POST /api/duty-board/{slotId}/claim auf Slot mit slots_total=2, slots_filled=0
- **THEN** HTTP 204, `duty_assignments` mit status=pending angelegt, `duty_slots.slots_filled=1`

#### Scenario: Vollen Slot claimen
- **WHEN** User POST auf Slot mit slots_total=1, slots_filled=1
- **THEN** HTTP 409

#### Scenario: Doppeltes Claimen desselben Slots
- **WHEN** User POST erneut auf bereits geclaimten Slot
- **THEN** HTTP 409

#### Scenario: Pending Slot freigeben
- **WHEN** User DELETE /api/duty-board/{slotId}/claim auf eigenem pending-Assignment
- **THEN** HTTP 204, Assignment gelöscht, slots_filled dekrementiert

#### Scenario: Fulfilled Slot kann nicht unclaimed werden
- **WHEN** User DELETE auf eigenem Assignment mit status=fulfilled
- **THEN** HTTP 409

#### Scenario: Unclaim ohne vorherige Zuweisung
- **WHEN** User DELETE ohne Assignment für diesen Slot
- **THEN** HTTP 404


## REMOVED Requirements

### Requirement: Dienstkonten-Sichtbarkeit
**Reason**: `GET /api/duty-accounts` wird entfernt. Die Oberfläche liest die Route nicht mehr; der ausgelieferte `ist`-Wert war falsch, weil `Fulfill`/`CashSubstitute` ihn nie gebucht haben.
**Migration**: Dienst-Bilanz über `GET /api/dashboard` (`meineDienste.dutyAccount`) bzw. `GET /api/duty-fairness/rangliste`.
