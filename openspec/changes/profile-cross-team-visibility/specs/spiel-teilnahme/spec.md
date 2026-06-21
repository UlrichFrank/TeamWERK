## MODIFIED Requirements

### Requirement: Spieldetail-Seite zeigt alle Kader-Mitglieder

Die Spieldetail-Seite (`/termine/spiel/{id}` bzw. `/termine/ereignis/{id}`) SHALL alle Mitglieder aus `kader_members` und `kader_extended_members` der zugehörigen Teams anzeigen, unabhängig davon, ob sie eine RSVP-Antwort abgegeben haben. Die Daten kommen aus `GET /api/games/{id}/participants`.

Bei Spielen mit **mehreren** Teams in `game_teams` SHALL die Response für Caller **ohne** Funktion (kein admin/trainer/sportliche_leitung/vorstand) auf folgende Menge gefiltert sein:

1. Alle Member aus den Teams, in denen der Caller selbst (oder eines seiner Kinder via `family_links`) im Kader oder erweiterten Kader steht ("meine Teams im Event"), PLUS
2. Member aus fremden Teams, deren `members.cross_team_visible = 1` ist.

Caller mit Funktion `admin`, `trainer`, `sportliche_leitung` oder `vorstand` SHALL alle Member ohne Filter sehen. Bei Spielen mit nur einem Team in `game_teams` erfolgt KEIN Filter.

#### Scenario: Mitglied ohne RSVP erscheint in der Teilnahme-Tabelle

- **WHEN** ein reguläres Kader-Mitglied keine RSVP-Antwort für ein Spiel abgegeben hat
- **THEN** erscheint es trotzdem in der Teilnahme-Tabelle mit `rsvp_status: null`

#### Scenario: Erweitertes Kader-Mitglied erscheint ohne RSVP

- **WHEN** ein erweitertes Kader-Mitglied für das Team eingetragen ist
- **THEN** erscheint es in der Teilnahme-Tabelle mit `rsvp_status: null`

#### Scenario: Spieler sieht bei Multi-Team-Event nur eigenes Team

- **WHEN** ein Spieler in Team A ein generisches Event mit Teams A und B öffnet
- **AND** kein Member aus Team B hat `cross_team_visible=1`
- **THEN** enthält die Response nur Member aus Team A

#### Scenario: Opt-In macht fremde Team-Member sichtbar

- **WHEN** ein Spieler in Team A ein generisches Event mit Teams A und B öffnet
- **AND** Member X aus Team B hat `cross_team_visible=1`
- **THEN** enthält die Response alle Member aus Team A plus Member X aus Team B

#### Scenario: Elternteil sieht Teams seiner Kinder

- **WHEN** ein Elternteil (kein eigenes Member) eines Kindes in Team A das Event öffnet
- **THEN** sieht es die Member aus Team A (analog zum Spieler) plus Opt-In-Member fremder Teams

#### Scenario: Member in mehreren Teams sieht alle eigenen Teams

- **WHEN** Member M im Kader von Team A UND Team B steht und das Event Teams A+B+C umfasst
- **THEN** sieht M alle Member aus Team A und Team B vollständig, plus Opt-In-Member aus Team C

#### Scenario: Trainer sieht alle Teams

- **WHEN** ein Caller mit `trainer`-Funktion das Event öffnet
- **THEN** sieht er alle Member aller in `game_teams` eingetragenen Teams, unabhängig von `cross_team_visible`

#### Scenario: Single-Team-Event ungefiltert

- **WHEN** ein Event nur ein Team in `game_teams` hat
- **THEN** erfolgt KEIN Filter; `cross_team_visible` hat keinen Effekt

#### Scenario: Spieldetail nutzt /participants

- **WHEN** die Spieldetail-Seite geladen wird
- **THEN** lädt das Frontend `GET /api/games/{id}/participants` als Datenquelle für die Teilnahme-Tabelle
