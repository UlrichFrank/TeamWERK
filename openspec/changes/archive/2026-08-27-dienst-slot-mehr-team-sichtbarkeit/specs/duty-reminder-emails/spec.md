## MODIFIED Requirements

### Requirement: Rollenbasierte Empfängerbestimmung

Das System SHALL Empfänger anhand von `duty_type.target_role` und dem **Team-Scope** des
Slots bestimmen. Der Team-Scope SHALL wie folgt aufgelöst werden:

1. `duty_slot.team_id` gesetzt → genau dieses Team.
2. `duty_slot.team_id IS NULL` und `duty_slot.game_id` gesetzt → **alle** Teams des
   Spiels (`game_teams`). Damit erreicht die Erinnerung dieselbe Menge, die den Slot in
   der Dienstbörse sieht, statt den ganzen Verein.
3. `duty_slot.team_id IS NULL` und `duty_slot.game_id IS NULL` → kein Team-Filter
   (vereinsweit, unverändert).

#### Scenario: Spieler-Empfänger via team_memberships
- **WHEN** `target_role = 'spieler'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'spieler'` die über `members → team_memberships` (aktive Saison) dem Team X angehören eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Elternteil-Empfänger via family_links (indirekt)
- **WHEN** `target_role = 'elternteil'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'elternteil'` deren Kind über `family_links → members → team_memberships` (aktive Saison) dem Team X angehört eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Trainer-Empfänger via team_trainers
- **WHEN** `target_role = 'trainer'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'trainer'` die über `team_trainers` dem Team X zugeordnet sind eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Team-loser Slot an einem Spiel adressiert die Teams des Spiels
- **WHEN** `duty_slot.team_id IS NULL` und `duty_slot.game_id` verweist auf ein Spiel mit den Teams A und B
- **THEN** erhalten die passenden Rollen aus Team A und Team B eine Erinnerung
- **AND** erhalten Mitglieder eines unbeteiligten Teams C keine Erinnerung

#### Scenario: Vereinsweite Empfänger bei fehlendem team_id
- **WHEN** `duty_slot.team_id IS NULL` und `duty_slot.game_id IS NULL`
- **THEN** werden alle User mit der passenden Rolle (unabhängig von Team-Zugehörigkeit) als Empfänger betrachtet
