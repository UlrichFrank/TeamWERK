## MODIFIED Requirements

### Requirement: Personalisierten Fahrtgemeinschafts-Status anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingHint` einen erweiterten Status zurückgeben, der folgende Informationen enthält:

- Nächstes Auswärtsspiel (Spielgegner, Datum, game_id) — wie bisher
- `bieteCount` / `sucheCount` — aggregierte Zähler — wie bisher
- `myEntry`: ob und mit welchem `typ` der User selbst eingetragen ist (inkl. `id` des Eintrags), oder `null`
- `paarungen`: Liste der aktuell `confirmed` Paarungen des Users (unabhängig vom Alter), mit Name der Gegenseite
- `recentEvents`: Ereignisse aus `carpooling_events` für dieses Spiel und diesen User aus den letzten 48 h, absteigend nach `created_at`

#### Scenario: User mit bestätigter Paarung

- **WHEN** der User eine Paarung mit `status='confirmed'` für das nächste Spiel hat
- **THEN** enthält `paarungen` diese Paarung mit Name der Gegenseite, unabhängig davon wann sie bestätigt wurde

#### Scenario: User mit abgelehnter Paarung

- **WHEN** eine Paarung des Users `status='rejected'` hat
- **THEN** erscheint sie NICHT in `paarungen`; stattdessen ist ein `pairing_rejected`- oder `pairing_cancelled`-Event in `recentEvents` (sofern innerhalb 48 h)

#### Scenario: Gelöschter Biete-Eintrag im Event-Feed

- **WHEN** ein `carpooling_events`-Eintrag `type='biete_deleted'` für den User und das Spiel existiert und `created_at >= now - 48h`
- **THEN** erscheint dieser Event in `recentEvents`

#### Scenario: Neue Biete/Suche-Einträge anderer im Event-Feed

- **WHEN** ein `carpooling_events`-Eintrag `type='biete_created'` oder `type='suche_created'` für den User existiert und `created_at >= now - 48h`
- **THEN** erscheint dieser Event in `recentEvents`

#### Scenario: Kein Auswärtsspiel

- **WHEN** kein kommendes Auswärtsspiel für das Team des Users existiert
- **THEN** ist `carpoolingHint` null (unverändert)

#### Scenario: User nicht eingetragen, keine Events

- **WHEN** der User weder biete noch suche für das nächste Spiel eingetragen hat und keine Events in den letzten 48 h existieren
- **THEN** ist `myEntry` null, `paarungen` ist `[]`, `recentEvents` ist `[]`
