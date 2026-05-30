## ADDED Requirements

### Requirement: Rollenabhängige Spielliste im Backend
`GET /api/mitfahrgelegenheiten` SHALL Spiele nach Team-Zugehörigkeit des anfragenden Nutzers filtern. Die Filterlogik richtet sich nach der Rolle:
- `admin`, `vorstand`: alle zukünftigen Spiele (kein Filter)
- `trainer`: Spiele der Teams, bei denen der Trainer als `kader_trainers`-Eintrag hinterlegt ist (aktive Saison)
- `spieler`: Spiele der Teams, in denen der Spieler als `kader_members`-Mitglied geführt ist (aktive Saison)
- `elternteil`: Spiele der Teams aller Kinder, die via `family_links` verknüpft sind (aktive Saison)

#### Scenario: Elternteil sieht nur Team-Spiele
- **WHEN** ein Nutzer mit Rolle `elternteil` `GET /api/mitfahrgelegenheiten` aufruft
- **THEN** werden nur Spiele zurückgegeben, die zum Team mindestens eines verknüpften Kindes gehören

#### Scenario: Spieler sieht nur Team-Spiele
- **WHEN** ein Nutzer mit Rolle `spieler` `GET /api/mitfahrgelegenheiten` aufruft
- **THEN** werden nur Spiele zurückgegeben, die zu seinen Kaderteams der aktiven Saison gehören

#### Scenario: Admin sieht alle Spiele
- **WHEN** ein Nutzer mit Rolle `admin` oder `vorstand` `GET /api/mitfahrgelegenheiten` aufruft
- **THEN** werden alle zukünftigen Spiele zurückgegeben (wie bisher)

#### Scenario: Nutzer ohne Team-Zuordnung
- **WHEN** ein Elternteil ohne `family_links` oder ein Spieler ohne Kader-Eintrag die Liste abruft
- **THEN** wird eine leere Spielliste zurückgegeben (kein Fehler)
