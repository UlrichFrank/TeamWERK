## ADDED Requirements

### Requirement: Lösch-Ereignisse persistieren

Das System SHALL Lösch-Ereignisse in `carpooling_events` speichern, bevor ein Mitfahrgelegenheiten-Eintrag gelöscht wird, wenn betroffene User mit einer aktiven Paarung existieren.

Betroffene User:
- `biete_deleted`: alle User, deren `suche`-Eintrag eine `pending` oder `confirmed` Paarung gegen diesen `biete`-Eintrag hat
- `suche_deleted`: der User des `biete`-Eintrags, falls eine `pending` oder `confirmed` Paarung existiert

#### Scenario: Biete-Eintrag mit aktiver Paarung gelöscht

- **WHEN** ein User seinen `biete`-Eintrag löscht und mindestens eine `pending` oder `confirmed` Paarung dagegen existiert
- **THEN** wird für jeden betroffenen `suche`-User ein `carpooling_events`-Eintrag mit `type='biete_deleted'` und `actor_name` des löschenden Users angelegt, bevor das DELETE ausgeführt wird

#### Scenario: Biete-Eintrag ohne aktive Paarung gelöscht

- **WHEN** ein User seinen `biete`-Eintrag löscht und keine `pending`/`confirmed` Paarung existiert
- **THEN** wird kein Event angelegt; das DELETE wird normal ausgeführt

#### Scenario: Suche-Eintrag mit aktiver Paarung gelöscht

- **WHEN** ein User seinen `suche`-Eintrag löscht und eine `pending` oder `confirmed` Paarung dagegen existiert
- **THEN** wird für den Biete-User ein `carpooling_events`-Eintrag mit `type='suche_deleted'` angelegt

#### Scenario: Atomarität sichergestellt

- **WHEN** das Schreiben des Events oder das DELETE fehlschlägt
- **THEN** werden beide Operationen zurückgerollt (Transaktion)

### Requirement: Events nur für zukünftige Spiele anzeigen

Das System SHALL beim Laden des Dashboards nur Events zurückgeben, deren verknüpftes Spiel (`game_id`) ein Datum >= heute hat.

#### Scenario: Event zu vergangenem Spiel

- **WHEN** ein `carpooling_events`-Eintrag existiert und `DATE(g.date) < DATE('now')`
- **THEN** wird er im Dashboard-Response nicht zurückgegeben
