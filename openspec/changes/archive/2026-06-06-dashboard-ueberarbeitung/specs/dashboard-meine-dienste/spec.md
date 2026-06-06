## ADDED Requirements

### Requirement: Meine Dienste im Dashboard

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `meineDienste` Informationen zum nächsten Spiel mit Dienst-Slots sowie den Saison-Saldo zurückgeben.

**Bezugspunkt:** Das nächste `game` (ab heute), für das in `duty_slots` mindestens ein Slot mit `season_id` der aktiven Saison existiert. Trainings werden ignoriert.

**Dienst-Slots:** Es werden nur Slots zurückgegeben, deren `duty_type.target_role` der Rolle des Users entspricht.

**Anzeigelogik:**
- Hat der User ≥1 Slot mit `status IN ('assigned','fulfilled','cash_substitute')` für dieses Spiel → `mySlots` enthält diese Slots (Typ-Name, Event-Zeit)
- Hat der User 0 eigene Slots → `mySlots` ist leer, `openSlotsCount` enthält die Anzahl noch offener Slots (`slots_total - slots_filled`) für dieses Spiel
- `dutyAccount` (Saison-Saldo) wird immer mitgeliefert

#### Scenario: User hat eigene Dienste für das nächste Spiel

- **WHEN** der User mindestens eine Dienst-Zusage für das nächste Spiel mit Slots hat
- **THEN** enthält `meineDienste.mySlots` diese Slots mit `duty_type_name` und `event_time`, und `openSlotsCount` ist 0

#### Scenario: User hat keine eigenen Dienste

- **WHEN** der User keine Dienst-Zusage für das nächste Spiel mit Slots hat
- **THEN** ist `meineDienste.mySlots` ein leeres Array und `openSlotsCount` enthält die Anzahl offener Slots dieses Spiels

#### Scenario: Kein kommendes Spiel mit Slots

- **WHEN** kein kommendes Spiel mit Dienst-Slots für die aktive Saison existiert
- **THEN** ist `meineDienste.nextGame` null; `dutyAccount` wird trotzdem zurückgegeben

#### Scenario: Saison-Saldo immer vorhanden

- **WHEN** eine aktive Saison existiert
- **THEN** enthält `meineDienste.dutyAccount` immer `ist`, `soll` (nullable) und `season`

#### Scenario: Trainer sieht offene Slots seines Teams

- **WHEN** der User die Rolle `trainer` hat und keine eigenen Dienste für das nächste Spiel hat
- **THEN** zeigt `openSlotsCount` die Summe aller offenen Slots des Spiels (teamübergreifend, falls Trainer mehrerer Teams)
