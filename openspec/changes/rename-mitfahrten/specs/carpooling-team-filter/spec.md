## MODIFIED Requirements

### Requirement: Rollenabhängige Spielliste im Backend
`GET /api/mitfahrten` SHALL Spiele nach Team-Zugehörigkeit des anfragenden Nutzers filtern. Filterlogik:
- `admin`, `vorstand`: alle zukünftigen Spiele (kein Filter, View wird nicht genutzt)
- alle anderen Rollen: nur Spiele, deren team_id in `user_accessible_teams` für den Nutzer und die aktive Saison enthalten ist

#### Scenario: Elternteil sieht nur Team-Spiele
- **WHEN** ein Nutzer mit Rolle `elternteil` `GET /api/mitfahrten` aufruft
- **THEN** werden nur Spiele zurückgegeben, die zum Team mindestens eines verknüpften Kindes gehören (aktive Saison)

#### Scenario: Admin sieht alle Spiele
- **WHEN** ein Nutzer mit Rolle `admin` oder `vorstand` den Endpunkt aufruft
- **THEN** werden alle zukünftigen Spiele zurückgegeben

#### Scenario: Nutzer ohne Team-Zuordnung
- **WHEN** ein Elternteil ohne `family_links` oder ein Spieler ohne Kader-Eintrag die Liste abruft
- **THEN** wird eine leere Spielliste zurückgegeben (HTTP 200, kein Fehler)

### Requirement: Optionaler team_id-Filter
`GET /api/mitfahrten?team_id=X` SHALL die Ergebnisse auf das angegebene Team einschränken. Der Filter wird nur angewendet, wenn `team_id` in den zugänglichen Teams des Nutzers liegt.

#### Scenario: Elternteil filtert auf ein Team
- **WHEN** ein Elternteil mit Kindern in zwei Teams `?team_id=15` übergibt
- **THEN** werden nur Spiele von Team 15 zurückgegeben

#### Scenario: Ungültige team_id (kein Zugriff)
- **WHEN** ein Nutzer eine team_id übergibt, die nicht in seinen zugänglichen Teams liegt
- **THEN** wird eine leere Spielliste zurückgegeben (HTTP 200, kein Fehler, keine Fehlermeldung)
