## ADDED Requirements

### Requirement: Slot-Kapazität bei Claim/Unclaim
Das System SHALL `slots_filled` beim Claim erhöhen und beim Unclaim verringern. Ein voller Slot (`slots_filled >= slots_total`) kann nicht geclaimt werden. Ein bereits erfüllter Slot kann nicht unclaimed werden.

#### Scenario: Freien Slot claimen
- **WHEN** User POST /api/duty-board/{slotId}/claim auf Slot mit slots_total=2, slots_filled=0
- **THEN** HTTP 204, `duty_assignments` mit status=pending angelegt, `duty_slots.slots_filled=1`, `duty_accounts`-Eintrag für aktive Saison existiert

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

### Requirement: Proxy-Kind Claim durch Elternteil
Das System SHALL einem Elternteil erlauben, einen Slot für ein Proxy-Kind zu claimen. Ein Proxy-Kind hat `can_login=0` und ist via `family_links` verknüpft.

#### Scenario: Elternteil claimt für Proxy-Kind
- **WHEN** Elternteil POST mit Body `{ user_id: <kind_user_id> }`, Kind hat can_login=0
- **THEN** HTTP 204, Assignment für Kind angelegt

#### Scenario: Claim für fremden User ohne Proxy-Verknüpfung
- **WHEN** User POST mit Body `{ user_id: <fremder_user_id> }`
- **THEN** HTTP 403

### Requirement: Board-Sichtbarkeit nach Rolle und Team
Das System SHALL im Board nur Slots anzeigen, die zur Teamzugehörigkeit des Users passen. Admins sehen alle Slots der aktiven Saison.

#### Scenario: Admin sieht alle Slots
- **WHEN** Admin GET /api/duty-board
- **THEN** Alle Slots der aktiven Saison im Response

#### Scenario: User sieht nur eigene Team-Slots
- **WHEN** User GET /api/duty-board, User ist via player_memberships in Team A, nicht in Team B
- **THEN** Nur Team-A-Slots

### Requirement: Audience-Filterung im Board
Das System SHALL Slots mit definiertem `audiences`-Array nur für passende Nutzergruppen anzeigen. `audiences = NULL` bedeutet: für alle sichtbar. Trainer, Vorstand und Admins bypassen den Filter.

#### Scenario: eltern-Slot für Elternteil sichtbar
- **WHEN** User mit family_links-Eintrag GET /api/duty-board, Slot hat audiences=["eltern"]
- **THEN** Slot enthalten

#### Scenario: eltern-Slot für User ohne Kinder unsichtbar
- **WHEN** User ohne family_links GET /api/duty-board, Slot hat audiences=["eltern"]
- **THEN** Slot nicht enthalten

#### Scenario: Trainer bypasses Audience-Filter
- **WHEN** User mit member_club_functions.function="trainer" GET /api/duty-board, Slot hat audiences=["eltern"]
- **THEN** Slot enthalten

#### Scenario: view=mine filtert auf eigene Slots
- **WHEN** User GET /api/duty-board?view=mine, User hat 2 von 5 Slots geclaimt
- **THEN** Genau 2 Slots

### Requirement: Dienstkonten-Sichtbarkeit
Das System SHALL Admins alle Dienstkonten zeigen, anderen Nutzern nur das eigene. Balance wird als soll−ist berechnet.

#### Scenario: Admin sieht alle Konten
- **WHEN** Admin GET /api/duty-accounts mit 3 Einträgen für verschiedene User
- **THEN** Alle 3 Einträge, jeder mit balance=soll-ist

#### Scenario: Standard-User sieht nur eigenes Konto
- **WHEN** Standard-User GET /api/duty-accounts
- **THEN** Nur eigener Eintrag

### Requirement: is_custom-Flag bei Slot-Mutations
Das System SHALL jeden manuell angelegten oder bearbeiteten Slot mit `is_custom=1` markieren. Beim Löschen eines Slots werden eingetragene Nutzer benachrichtigt.

#### Scenario: CreateSlot setzt is_custom=1
- **WHEN** Trainer POST /api/duty-slots
- **THEN** HTTP 201, `duty_slots.is_custom=1`

#### Scenario: UpdateSlot setzt is_custom=1
- **WHEN** Trainer PUT /api/duty-slots/{id} auf Slot mit is_custom=0
- **THEN** HTTP 204, `duty_slots.is_custom=1`

#### Scenario: DeleteSlot mit Assignments
- **WHEN** Trainer DELETE /api/duty-slots/{id} mit 1 eingetragenem User
- **THEN** HTTP 204, Slot gelöscht
