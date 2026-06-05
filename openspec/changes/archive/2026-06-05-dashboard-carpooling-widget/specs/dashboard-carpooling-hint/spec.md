## MODIFIED Requirements

### Requirement: Personalisierten Fahrtgemeinschafts-Status anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingHint` einen Status zurückgeben, der folgende Informationen enthält:

- Nächstes Auswärtsspiel (Spielgegner, Datum, game_id) — wie bisher
- `bieteCount` / `sucheCount` — aggregierte Zähler aller Einträge für dieses Spiel — wie bisher
- `myEntry`: ob und mit welchem `typ` der User selbst eingetragen ist (inkl. `id`), oder `null`
- `paarungen`: Liste der aktuell `confirmed` Paarungen des Users, mit Name der Gegenseite
- `openEntries`: Liste offener Einträge anderer Nutzer (ohne confirmed-Pairing), max. 5, mit `typ` und Name

`recentEvents` WIRD NICHT MEHR zurückgegeben.

#### Scenario: User mit bestätigter Paarung

- **WHEN** der User eine Paarung mit `status='confirmed'` für das nächste Spiel hat
- **THEN** enthält `paarungen` diese Paarung mit Name der Gegenseite

#### Scenario: Offene Einträge anderer sichtbar

- **WHEN** andere Nutzer für das nächste Spiel Einträge (biete oder suche) ohne `confirmed`-Pairing haben
- **THEN** enthält `openEntries` bis zu 5 dieser Einträge mit `typ` und Name des Nutzers

#### Scenario: Eintrag mit pending-Pairing gilt als offen

- **WHEN** ein Eintrag eines anderen Nutzers eine Paarung mit `status='pending'` hat (aber keine `confirmed`)
- **THEN** erscheint dieser Eintrag in `openEntries`

#### Scenario: Mehr als 5 offene Einträge

- **WHEN** mehr als 5 offene Einträge anderer Nutzer existieren
- **THEN** liefert `openEntries` genau 5 Einträge; `bieteCount` und `sucheCount` spiegeln die tatsächliche Gesamtzahl aller Einträge wider

#### Scenario: Eigener Eintrag nicht in openEntries

- **WHEN** der User selbst einen offenen Eintrag hat
- **THEN** erscheint dieser Eintrag NICHT in `openEntries` (nur in `myEntry`)

#### Scenario: Kein Auswärtsspiel

- **WHEN** kein kommendes Auswärtsspiel für das Team des Users existiert
- **THEN** ist `carpoolingHint` null

#### Scenario: User nicht eingetragen, keine offenen Einträge anderer

- **WHEN** der User weder biete noch suche hat und keine anderen offenen Einträge existieren
- **THEN** ist `myEntry` null, `paarungen` ist `[]`, `openEntries` ist `[]`

## REMOVED Requirements

### Requirement: recentEvents im CarpoolingHint

**Reason**: Der Event-Verlauf erzeugt Redundanz und Hintergrundrauschen im Dashboard-Widget. Bestätigte Paarungen werden über `paarungen` angezeigt; offene Mitfahrmöglichkeiten über `openEntries`.
**Migration**: Kein externer Konsum dieses Feldes. Frontend und Backend werden im selben Deployment aktualisiert.
