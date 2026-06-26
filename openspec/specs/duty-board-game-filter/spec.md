# duty-board-game-filter Specification

## Purpose

Diese Spezifikation beschreibt die Capability `duty-board-game-filter`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: duty-board game_id filter
`GET /duty-board` SHALL akzeptieren einen optionalen Query-Parameter `game_id` (Integer). Wenn angegeben, gibt der Endpunkt nur Slots zurück, deren `ds.game_id` dem angegebenen Wert entspricht. Alle bestehenden Zugriffskontrollen (Team-Sichtbarkeit, aktive Saison) bleiben unverändert aktiv.

#### Scenario: Filter auf bekanntes Spiel
- **WHEN** ein authentifizierter Nutzer `GET /duty-board?game_id=9` aufruft
- **THEN** enthält die Antwort nur Slots, die zu Spiel 9 gehören

#### Scenario: Filter auf Spiel ohne Berechtigung
- **WHEN** ein Nutzer `GET /duty-board?game_id=9` aufruft, dessen Team keinen Zugang zu Spiel 9 hat
- **THEN** gibt der Endpunkt ein leeres Array zurück (kein Fehler, keine Slots)

#### Scenario: Filter ohne game_id (Rückwärtskompatibilität)
- **WHEN** ein Nutzer `GET /duty-board` ohne `game_id`-Parameter aufruft
- **THEN** verhält sich der Endpunkt identisch zum bisherigen Verhalten — alle berechtigten Slots

#### Scenario: Admin mit game_id Filter
- **WHEN** ein Admin-Nutzer `GET /duty-board?game_id=9` aufruft
- **THEN** erhält er alle Slots von Spiel 9, unabhängig von Team-Zugehörigkeit
