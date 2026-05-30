## MODIFIED Requirements

### Requirement: Personalisierten Fahrtgemeinschafts-Status anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingHint` einen erweiterten Status zurückgeben, der folgende Informationen enthält:

- Nächstes Auswärtsspiel (Spielgegner, Datum, game_id) — wie bisher
- `bieteCount` / `sucheCount` — aggregierte Zähler — wie bisher
- `myEntry`: ob und mit welchem `typ` der User selbst eingetragen ist (inkl. `id` des Eintrags)
- `paarungen`: Liste der Paarungen, an denen der User beteiligt ist, mit `status`, `updated_at`, Name der Gegenseite
- `recentEvents`: Events aus `carpooling_events` für dieses Spiel, die den User betreffen, aus den letzten 48 h

#### Scenario: User mit Suche-Eintrag und bestätigter Paarung

- **WHEN** ein User eine aktive `suche` eingetragen hat und eine Paarung mit `status='confirmed'` existiert
- **THEN** enthält `carpoolingHint.myEntry` den suche-Eintrag und `paarungen` die bestätigte Paarung mit Name des Bieters

#### Scenario: User mit Suche-Eintrag und abgelehnter Paarung innerhalb 48 h

- **WHEN** eine Paarung des Users auf `status='rejected'` gesetzt wurde und `updated_at >= now - 48h`
- **THEN** ist diese Paarung in `paarungen` enthalten

#### Scenario: User mit Suche-Eintrag und abgelehnter Paarung älter als 48 h

- **WHEN** eine Paarung des Users `status='rejected'` hat und `updated_at < now - 48h`
- **THEN** ist diese Paarung NICHT in `paarungen` enthalten

#### Scenario: Gelöschter Biete-Eintrag — Event im Dashboard

- **WHEN** ein `carpooling_events`-Eintrag mit `type='biete_deleted'` für den User und das nächste Spiel existiert und `created_at >= now - 48h`
- **THEN** erscheint dieser Event in `recentEvents`

#### Scenario: Kein Auswärtsspiel

- **WHEN** kein kommendes Auswärtsspiel für das Team des Users existiert
- **THEN** ist `carpoolingHint` null (unverändert)

#### Scenario: User nicht eingetragen

- **WHEN** der User weder biete noch suche für das nächste Spiel eingetragen hat
- **THEN** ist `myEntry` null; `paarungen` ist ein leeres Array; `recentEvents` kann trotzdem Events enthalten (aus 48-h-Fenster)
