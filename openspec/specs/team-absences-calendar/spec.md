# team-absences-calendar Specification

## Purpose

Diese Spezifikation beschreibt die Capability `team-absences-calendar`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Trainer können Team-Abwesenheiten im Kalender einblenden
Nutzer mit System-Rolle `admin` oder einer der Vereinsfunktionen `trainer`, `sportliche_leitung`, `vorstand` SHALL im Kalender einen Toggle sehen, der Team-Abwesenheiten ein- oder ausblendet. Der Toggle-State wird in `sessionStorage` gespeichert und ist nach Seitenneustart standardmäßig AUS.

#### Scenario: Toggle für Trainer sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` den Kalender öffnet
- **THEN** ist ein Toggle „Mannschaftsabwesenheiten" sichtbar

#### Scenario: Toggle für reinen Spieler/Eltern nicht sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `spieler` (ohne Trainer-/Vorstandsfunktion) oder ein reiner Eltern-User (`isParent=true`, keine Funktion) den Kalender öffnet
- **THEN** ist kein Toggle „Mannschaftsabwesenheiten" vorhanden

#### Scenario: Default ist AUS
- **WHEN** ein berechtigter Nutzer den Kalender neu öffnet (kein sessionStorage-Eintrag)
- **THEN** ist der Toggle deaktiviert und keine Team-Abwesenheiten werden geladen

#### Scenario: State wird in sessionStorage gespeichert
- **WHEN** der Nutzer den Toggle aktiviert und die Seite neu lädt (gleiche Browser-Session)
- **THEN** ist der Toggle wieder aktiviert

---

### Requirement: Backend liefert Team-Abwesenheiten nur für berechtigte Rollen
`GET /api/absences/calendar?show_team=true` SHALL für berechtigte Nutzer zusätzlich Abwesenheiten von Teammitgliedern mit `absences_public = 1` zurückgeben, die in mindestens einem der eigenen Teams der aktiven Saison sind.

#### Scenario: Berechtigt — Team-Abwesenheiten werden zurückgegeben
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true` aufruft
- **THEN** enthält die Antwort zusätzlich Abwesenheiten von Mitgliedern seiner Teams mit `absences_public = 1`

#### Scenario: Unberechtigt — Parameter wird ignoriert
- **WHEN** ein Spieler `GET /api/absences/calendar?show_team=true` aufruft
- **THEN** enthält die Antwort nur eigene und Kinder-Abwesenheiten (wie ohne Parameter)

#### Scenario: Team-Filter einschränken
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true&team_id=5` aufruft
- **THEN** enthält die Antwort nur Team-Abwesenheiten von Mitgliedern des Teams 5

---

### Requirement: Abwesenheits-Response enthält `is_own`-Flag
Jede Abwesenheit in der Calendar-Response SHALL ein Feld `is_own: bool` enthalten.

#### Scenario: Eigene Abwesenheit
- **WHEN** die Abwesenheit dem eingeloggten Nutzer oder einem seiner Kinder gehört
- **THEN** ist `is_own = true`

#### Scenario: Team-Abwesenheit
- **WHEN** die Abwesenheit einem anderen Teammitglied gehört
- **THEN** ist `is_own = false`

---

### Requirement: Team-Abwesenheiten werden mit bestehendem Team-Filter koordiniert
Das System SHALL bei aktivem Team-Filter (`filterTeamId`) nur Abwesenheiten dieses Teams laden (`?team_id={filterTeamId}`), wenn Team-Abwesenheiten eingeblendet sind.

#### Scenario: Team-Filter aktiv
- **WHEN** Trainer wählt Team A im Dropdown und aktiviert Team-Abwesenheiten
- **THEN** werden nur Abwesenheiten von Mitgliedern des Teams A angezeigt

#### Scenario: Kein Team-Filter
- **WHEN** Trainer aktiviert Team-Abwesenheiten ohne Team-Filter
- **THEN** werden Abwesenheiten aller eigenen Teams angezeigt
