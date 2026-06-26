# erweiterter-kader-dashboard-badge Specification

## Purpose

Diese Spezifikation beschreibt die Capability `erweiterter-kader-dashboard-badge`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: GET /teams/my liefert isExtended-Flag pro Team

`GET /api/teams/my` SHALL pro Team-Objekt ein Feld `isExtended: bool` zurückgeben. Das Feld ist `true`, wenn der anfragende User das Team ausschließlich über `kader_extended_members` sieht — also kein Eintrag in `kader_members` für denselben User, dasselbe Team und dieselbe aktive Saison existiert. Für Spieler im Stammkader, Trainer und Eltern ist `isExtended` immer `false`.

#### Scenario: Spieler im Erweiterten Kader bekommt isExtended=true

- **WHEN** ein User ausschließlich über `kader_extended_members` Zugang zu Team A hat
- **THEN** enthält `GET /api/teams/my` für Team A das Feld `isExtended: true`

#### Scenario: Stammkader-Spieler bekommt isExtended=false

- **WHEN** ein User über `kader_members` Zugang zu Team A hat
- **THEN** enthält `GET /api/teams/my` für Team A das Feld `isExtended: false`

#### Scenario: Trainer bekommt isExtended=false

- **WHEN** ein Trainer über `kader_trainers` Zugang zu Team A hat
- **THEN** enthält `GET /api/teams/my` für Team A das Feld `isExtended: false`

---

### Requirement: GET /dashboard liefert isExtended-Flag pro Termin

`GET /api/dashboard` → `meineTermine` SHALL pro `NextEvent`-Objekt ein Feld `isExtended: bool` zurückgeben. Das Feld ist `true`, wenn der anfragende User das zugehörige Team ausschließlich über `kader_extended_members` sieht.

#### Scenario: Training eines Extended-Teams ist als Erweiterter Kader markiert

- **WHEN** ein Spieler nur im Erweiterten Kader von Team B ist
- **WHEN** das nächste Event ein Training von Team B ist
- **THEN** enthält das Event-Objekt in `meineTermine` das Feld `isExtended: true`

#### Scenario: Spiel des Stammteams ist nicht markiert

- **WHEN** ein Spieler im Stammkader von Team A ist
- **WHEN** das nächste Event ein Spiel von Team A ist
- **THEN** enthält das Event-Objekt in `meineTermine` das Feld `isExtended: false`

---

### Requirement: Dashboard „Mein Team" kennzeichnet Extended-Teams

Die `MeinTeamSection` im Dashboard SHALL für Teams mit `isExtended: true` ein sichtbares Badge „Erw. Kader" neben dem Teamnamen darstellen.

#### Scenario: Extended-Team zeigt Badge

- **WHEN** `GET /api/teams/my` ein Team mit `isExtended: true` zurückgibt
- **THEN** zeigt die „Mein Team"-Sektion im Dashboard neben dem Teamnamen das Badge „Erw. Kader"

#### Scenario: Stammkader-Team zeigt kein Badge

- **WHEN** `GET /api/teams/my` ein Team mit `isExtended: false` zurückgibt
- **THEN** erscheint kein Badge neben dem Teamnamen

---

### Requirement: Dashboard „Meine Termine" kennzeichnet Events erweiterter Teams

Die `MeineTermineSection` im Dashboard SHALL für Events mit `isExtended: true` in der Teamzeile den Zusatz „(Erw. Kader)" darstellen.

#### Scenario: Termin eines Extended-Teams zeigt Zusatz

- **WHEN** ein `NextEvent` mit `isExtended: true` gerendert wird
- **THEN** erscheint in der Teamzeile des Events der Text „(Erw. Kader)" hinter dem Teamnamen

#### Scenario: Termin eines Stammteams zeigt keinen Zusatz

- **WHEN** ein `NextEvent` mit `isExtended: false` gerendert wird
- **THEN** erscheint kein Zusatz in der Teamzeile
