## ADDED Requirements

### Requirement: Rotations-Slots ohne Team-Zuordnung bleiben sichtbar und übernehmbar

Ein Duty-Slot aus einem rotations-aktivierten Vorlagen-Item, der wegen erschöpfter Team-Warteschlange ohne Team-Zuordnung entsteht (`duty_slots.team_id = NULL`, `game_id` gesetzt), SHALL über den bestehenden `team_id IS NULL`-Fallback (Sichtbarkeit anhand der Teams des referenzierten Spiels, siehe `GET /api/duty-board`) für Mitglieder, Trainer und Eltern der am Spiel beteiligten Teams sichtbar und übernehmbar bleiben — ohne dass dafür eine gesonderte Sichtbarkeitsregel nötig ist.

#### Scenario: Unzugeordneter Rotations-Slot ist für das Spielteam sichtbar

- **WHEN** ein Regen-Lauf für ein Heimspiel von Team X einen Rotations-Slot ohne Team-Zuordnung erzeugt (Cap-Überlauf)
- **AND** ein Elternteil eines Spielers von Team X ruft `GET /api/duty-board` auf
- **THEN** erscheint der Slot in der Gruppe dieses Spiels, für den Elternteil sichtbar und übernehmbar (vorbehaltlich der bestehenden Audience-Filterung des Items)
