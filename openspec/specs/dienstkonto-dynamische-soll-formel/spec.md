# dienstkonto-dynamische-soll-formel Specification

## Purpose

Diese Spezifikation beschreibt die Capability `dienstkonto-dynamische-soll-formel`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Dynamische soll-Berechnung für Elternteil
Der `/api/dashboard`-Endpoint MUST `dutyAccount.soll` für Elternteile dynamisch aus Kader-Daten berechnen statt pauschal.

Formel pro verknüpftem Kind:
```
child_soll = (kader.games_per_season × avg_slots_per_game) / player_count / parent_count
```
wobei:
- `avg_slots_per_game = (heim_template_slots + auswärts_template_slots) / 2`
- `player_count` = Anzahl Mitglieder in diesem Kader
- `parent_count` = Anzahl Elternteile verknüpft mit diesem Kind (1 oder 2)

`soll = round(Summe aller child_soll)`

#### Scenario: Kind mit einem Elternteil, 20 Spiele, 6 Slots/Spiel, 20 Spieler
- **WHEN** 1 Elternteil verknüpft, kader.games_per_season=20, avg_slots=6, player_count=20
- **THEN** soll = round(20 × 6 / 20 / 1) = 6

#### Scenario: Kind mit zwei Elternteilen
- **WHEN** 2 Elternteile verknüpft, gleiche Rahmenwerte
- **THEN** soll = round(20 × 6 / 20 / 2) = 3 (jedes Elternteil sieht 3)

#### Scenario: Zwei Kinder im selben Kader, ein Elternteil
- **WHEN** 2 Kinder im gleichen Kader, 1 Elternteil
- **THEN** soll = 2 × round(pro Kind) — addiert

#### Scenario: games_per_season = 0
- **WHEN** kader.games_per_season ist 0
- **THEN** soll = 0 (kein Fehler)

#### Scenario: Kind in keinem aktiven Kader
- **WHEN** Kind hat keinen kader_members-Eintrag für die aktive Saison
- **THEN** Kind wird übersprungen, kein Beitrag zu soll

### Requirement: Datenschutz
Das System MUST sicherstellen, dass kein Elternteil das Dienstkonto oder den soll-Wert des anderen Elternteils sieht. Jedes Konto wird individuell berechnet und ausgegeben.

#### Scenario: Zwei Elternteile sehen unterschiedliche Dienstkonten

- **WHEN** zwei Elternteile mit demselben Kind jeweils das Dashboard laden
- **THEN** sieht jedes Elternteil nur sein eigenes Dienstkonto, nicht das des anderen

### Requirement: Erklärtext im Frontend
DutyAccountTile SHALL für Elternteile den Text „Ziel: {soll} Dienste (Saison {season.name})" anzeigen — keine Formel-Details sichtbar.

#### Scenario: soll = 0 (nicht konfiguriert)
- **WHEN** soll = 0
- **THEN** zeigt das UI keinen Fortschrittsbalken, nur den Zähler „0/0 Dienste"
