## REMOVED Requirements

### Requirement: Manuell angelegter Dienst-Slot an einem Mehr-Team-Termin trägt kein Team

**Reason**: Die Regel galt nur für Termine mit mehr als einem Team und ließ das Feld bei
Ein-Team-Terminen weiter schreiben — damit blieb `duty_slots.team_id` eine zweite,
einfrierende Quelle für eine Tatsache, die `game_teams` bereits führt, und die Fehlerklasse
"jemand schreibt hier ein Team hinein" blieb offen. Sie wird durch die weiter gefasste
Anforderung unten ersetzt, die für **jeden** spielgebundenen Slot gilt.

## ADDED Requirements

### Requirement: Ein spielgebundener Dienst-Slot trägt kein Team

Ein Dienst-Slot mit gesetztem `game_id` SHALL `team_id = NULL` tragen — unabhängig davon,
wie viele Teams dem Termin zugeordnet sind und über welchen Weg er entsteht (Vorlagen-Regen,
Bewirtungsrotation, `POST /api/duty-slots`, Massenlauf, H4A-Import). `POST /api/duty-slots`
SHALL ein mitgeschicktes `team_id` bei gesetztem `game_id` ignorieren statt es abzulehnen;
das Frontend SHALL es nicht mehr senden.

Die Sichtbarkeit eines spielgebundenen Slots SHALL ausschließlich über `game_id` →
`game_teams` aufgelöst werden. Die Leseabfragen SHALL `ds.team_id` bei gesetztem `game_id`
nicht auswerten — auch nicht, wenn dort noch ein Bestandswert steht. Damit sind migrierte
und nicht migrierte Zeilen ununterscheidbar sichtbar.

`duty_slots.team_id` SHALL für Slots **ohne** `game_id` unverändert der Geltungsbereich
bleiben (Vereinsfest o. ä.): dort bestimmt es allein, wer den Slot sieht, und ein Slot ohne
Team und ohne Spiel bleibt nur für Vorstand/Admin sichtbar.

#### Scenario: Slot an Termin mit drei Teams ist für alle drei sichtbar
- **WHEN** ein Vorstand an einem Termin mit den Teams A, B und C einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` und gesetztem `game_id` gespeichert
- **AND** erscheint er in `GET /api/duty-board` für Spieler, Trainer und Eltern aller drei Teams

#### Scenario: Slot an Termin mit genau einem Team trägt ebenfalls kein Team
- **WHEN** ein Vorstand an einem Termin mit nur Team A einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` gespeichert
- **AND** sehen Mitglieder von Team A ihn, Mitglieder anderer Teams nicht

#### Scenario: Mitgeschicktes team_id wird ignoriert
- **WHEN** ein Client `POST /api/duty-slots` mit `game_id` UND `team_id` aufruft
- **THEN** antwortet die API mit 201 und der gespeicherte Slot trägt `team_id = NULL`

#### Scenario: Bestands-Slot mit alter team_id bleibt für alle Teams des Termins sichtbar
- **WHEN** ein vor der Migration angelegter Slot noch `team_id` eines der Teams trägt
- **THEN** sehen ihn Mitglieder **aller** Teams seines Termins, nicht nur die dieses einen Teams

#### Scenario: Slot ohne Spiel behält seinen Team-Geltungsbereich
- **WHEN** ein Dienst-Slot ohne `game_id`, aber mit `team_id = A` angelegt wird
- **THEN** sehen ihn nur Mitglieder von Team A und deren Eltern
