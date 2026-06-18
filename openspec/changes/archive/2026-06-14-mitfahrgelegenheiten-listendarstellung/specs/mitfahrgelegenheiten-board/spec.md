## ADDED Requirements

### Requirement: Mitfahrgelegenheiten als chronologische Liste

Die Mitfahrgelegenheiten-Seite SHALL alle zukünftigen Spiele und Events in einer einzigen, fortlaufenden Liste anzeigen. Eine Aufteilung in Tabs nach Event-Typ SHALL NICHT mehr existieren.

#### Scenario: Liste zeigt alle Event-Typen zusammen

- **WHEN** ein Nutzer die Mitfahrgelegenheiten-Seite öffnet
- **THEN** sieht er alle zukünftigen Spiele und Events seines Teams in einer durchgehenden Liste — unabhängig vom Event-Typ (heim, auswärts, generisch)

#### Scenario: Keine Tab-Navigation vorhanden

- **WHEN** ein Nutzer die Seite öffnet
- **THEN** existieren keine Tab-Schaltflächen "Auswärtsspiele", "Heimspiele" oder "Events" — alle Filterung erfolgt über Pill-Buttons

### Requirement: Chronologische Sortierung mit Team-Kürzel-Sekundärschlüssel

Die Liste SHALL primär aufsteigend nach Datum + Uhrzeit des Spiels sortiert sein. Bei gleichem Datum und gleicher Uhrzeit SHALL die Sortierung sekundär alphabetisch nach Team-Kürzel erfolgen.

#### Scenario: Nächstes Spiel zuerst

- **WHEN** zwei Spiele mit unterschiedlichem Datum vorliegen
- **THEN** erscheint das früher stattfindende Spiel oberhalb des späteren

#### Scenario: Gleiche Anstoßzeit, unterschiedliche Teams

- **WHEN** zwei Spiele am selben Tag zur selben Uhrzeit stattfinden, eines für Team `mA` und eines für Team `wB1`
- **THEN** erscheint `mA` oberhalb von `wB1` (alphabetische Sortierung der Kürzel)

#### Scenario: Generisches Multi-Team-Event

- **WHEN** ein generisches Event mehreren Teams zugeordnet ist
- **THEN** wird der alphabetisch kleinste der zugeordneten Team-Kürzel als Sortierschlüssel verwendet

### Requirement: Farbcodierung der Game-Cards nach Event-Typ

Jede Game-Card SHALL eine farbliche Markierung erhalten, die dem Event-Typ entspricht und konsistent mit der Terminübersicht (`TerminePage`) ist. Die Farbzuordnung folgt der zentralen Funktion `getEventColors()`:

| Event-Typ   | Border / Tint       |
|-------------|---------------------|
| `heim`      | brand-yellow        |
| `auswärts`  | brand-text-muted / grau |
| `generisch` | brand-blue          |

#### Scenario: Heimspiel-Card gelb markiert

- **WHEN** ein Spiel mit `eventType=heim` angezeigt wird
- **THEN** zeigt die Card oben einen gelben Border-Streifen und einen leichten gelben Background-Tint

#### Scenario: Auswärtsspiel-Card grau markiert

- **WHEN** ein Spiel mit `eventType=auswärts` angezeigt wird
- **THEN** zeigt die Card oben einen grauen Border-Streifen und einen leichten grauen Background-Tint

#### Scenario: Generisches Event blau markiert

- **WHEN** ein Spiel mit `eventType=generisch` angezeigt wird
- **THEN** zeigt die Card oben einen blauen Border-Streifen und einen leichten blauen Background-Tint

### Requirement: Event-Typ-Filter als Pill-Buttons

Die Seite SHALL drei Pill-Buttons im Header anzeigen: "Heim", "Auswärts" und "Sonstiges". Mehrere Pills SHALL gleichzeitig aktiv sein können. Eine Card wird angezeigt, wenn ihr Event-Typ in der Menge der aktiven Pills enthalten ist. Wenn keine Pill aktiv ist, ist die Liste leer.

#### Scenario: Standardansicht — alle Pills aktiv

- **WHEN** ein Nutzer die Seite ohne URL-Filter öffnet
- **THEN** sind alle drei Event-Typ-Pills (Heim, Auswärts, Sonstiges) aktiv und alle Spiel-Typen werden angezeigt

#### Scenario: Nur Auswärtsspiele

- **WHEN** ein Nutzer alle Pills außer "Auswärts" deaktiviert
- **THEN** zeigt die Liste nur Spiele mit `eventType=auswärts`

#### Scenario: Keine Pill aktiv

- **WHEN** ein Nutzer alle Event-Typ-Pills deaktiviert
- **THEN** ist die Liste leer und zeigt eine entsprechende Hinweismeldung

### Requirement: Filter-State persistiert in URL-Search-Params

Die Auswahl des Team-Filters, der aktiven Event-Typ-Pills und der "Meine"-Pill SHALL als URL-Search-Params gespeichert werden, damit Reload und Deep-Linking den Filter-State erhalten. Default-State (alle Pills aktiv, alle Teams, Team-Modus) SHALL keine Params in der URL erzeugen.

#### Scenario: Reload erhält Filter

- **WHEN** ein Nutzer die Pills "Heim" und "Sonstiges" deaktiviert, sodass nur "Auswärts" aktiv bleibt, und die Seite neu lädt
- **THEN** zeigt die URL `?types=auswärts` und nur Auswärtsspiele werden angezeigt

#### Scenario: Default-Zustand zeigt saubere URL

- **WHEN** ein Nutzer alle Pills aktiviert lässt, keinen Team-Filter setzt und Team-Modus aktiv hat
- **THEN** enthält die URL keine `team`-, `types`- oder `mine`-Params

#### Scenario: Deep-Link mit Filter

- **WHEN** ein Nutzer eine URL `?team=3&types=heim&mine=1` öffnet
- **THEN** sind Team 3, nur die Heim-Pill und die "Meine"-Pill aktiv

### Requirement: Compact-Header bei schmalen Viewports

Bei Viewport-Breiten unter 950 px SHALL die Pill-Leiste nur die Icons anzeigen (Labels ausgeblendet), um Platz zu sparen. Die Schwelle entspricht der `TerminePage`-Konvention.

#### Scenario: Compact-Modus aktiviert

- **WHEN** die Viewport-Breite < 950 px ist
- **THEN** zeigen die Filter-Pills nur ihr Icon (z. B. `<Home>`, `<Plane>`, `<Calendar>`, `<UserCheck>`) und keine Text-Labels

#### Scenario: Vollformat-Modus

- **WHEN** die Viewport-Breite ≥ 950 px ist
- **THEN** zeigen die Filter-Pills sowohl Icon als auch Label
