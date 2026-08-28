## MODIFIED Requirements

### Requirement: Rollenbasierte Empfängerbestimmung

Das System SHALL Empfänger anhand von `duty_type.target_role` und dem **Team-Scope** des
Slots bestimmen. Der Team-Scope SHALL wie folgt aufgelöst werden:

1. `duty_slot.game_id` gesetzt → **alle** Teams des Spiels (`game_teams`). `team_id`
   SHALL dabei nicht ausgewertet werden, auch wenn dort noch ein Bestandswert steht.
2. `duty_slot.game_id IS NULL` und `duty_slot.team_id` gesetzt → genau dieses Team.
3. Weder `game_id` noch `team_id` → kein Team-Filter (vereinsweit, unverändert).

#### Scenario: Spieler-Empfänger via team_memberships
- **WHEN** `target_role = 'spieler'` und der Slot ohne `game_id` auf `team_id = X` zeigt
- **THEN** erhalten alle User mit `role = 'spieler'` die über `members → team_memberships` (aktive Saison) dem Team X angehören eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Elternteil-Empfänger via family_links (indirekt)
- **WHEN** `target_role = 'elternteil'` und der Slot ohne `game_id` auf `team_id = X` zeigt
- **THEN** erhalten alle User mit `role = 'elternteil'` deren Kind über `family_links → members → team_memberships` (aktive Saison) dem Team X angehört eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Trainer-Empfänger via team_trainers
- **WHEN** `target_role = 'trainer'` und der Slot ohne `game_id` auf `team_id = X` zeigt
- **THEN** erhalten alle User mit `role = 'trainer'` die über `team_trainers` dem Team X zugeordnet sind eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Team-loser Slot an einem Spiel adressiert die Teams des Spiels
- **WHEN** `duty_slot.game_id` auf ein Spiel mit den Teams A und B verweist
- **THEN** erhalten die passenden Rollen aus Team A und Team B eine Erinnerung
- **AND** erhalten Mitglieder eines unbeteiligten Teams C keine Erinnerung

#### Scenario: Vereinsweite Empfänger bei fehlendem team_id
- **WHEN** weder `duty_slot.game_id` noch `duty_slot.team_id` gesetzt ist
- **THEN** werden alle User mit der passenden Rolle (unabhängig von Team-Zugehörigkeit) als Empfänger betrachtet

#### Scenario: Bestands-Slot mit alter team_id folgt trotzdem dem Spiel
- **WHEN** ein noch nicht migrierter Slot `game_id` eines Spiels mit den Teams A und B trägt und zusätzlich `team_id = A`
- **THEN** erhalten die passenden Rollen aus **beiden** Teams eine Erinnerung
