## MODIFIED Requirements

### Requirement: Filterzustand in URL-Query-Parametern

Die `/termine`-Seite SHALL ihren Filterzustand vollständig über URL-Query-Parameter abbilden. Beim Mount liest sie die Parameter aus `useSearchParams()` und initialisiert daraus den State. Jede Änderung an Filtern (Team-Auswahl, Termin-Typen, Vergangene anzeigen, Textfilter) MUSS die URL via `setSearchParams()` (Replace, nicht Push) aktualisieren, sodass die Seite per Browser-Back/Forward navigierbar und per Link teilbar ist.

Unterstützte Parameter:
- `team` (eine numerische Team-ID; fehlt → kein Team-Filter)
- `types` (kommaseparierte Werte aus `training`, `heim`, `auswaerts`; fehlt → alle Typen aktiv, identisch zum bisherigen Default)
- `past` (`1` zeigt vergangene Termine, default `0`)
- `q` (Freitext-Filterausdruck; fehlt oder leer → kein Textfilter. Auswertung siehe Capability `termin-textfilter`)

Ungültige oder unbekannte Werte SHALL ignoriert und der jeweilige Filter auf seinen Default zurückgesetzt werden — ohne Fehlermeldung. Für `q` gibt es keine ungültigen Werte: jeder Zeichenketten-Inhalt ist ein zulässiger Filterausdruck, ein Ausdruck ohne Treffer führt zu einer leeren Liste, nicht zu einem Fehler.

Anders als die übrigen Filter SHALL `q` die URL **verzögert** aktualisieren (~250 ms nach dem letzten Tastenanschlag), während die Liste unmittelbar gefiltert wird. Damit entsteht kein URL-Schreibvorgang pro Zeichen.

#### Scenario: Page lädt mit Team-Filter aus URL
- **WHEN** ein User `/termine?team=3` aufruft
- **THEN** ist der Team-Filter beim ersten Render auf Team-ID 3 vorbelegt
- **THEN** zeigt die Liste ausschließlich Termine dieses Teams

#### Scenario: Page lädt mit Typ- und Past-Filter aus URL
- **WHEN** ein User `/termine?types=heim,auswaerts&past=1` aufruft
- **THEN** sind nur die Termin-Typen „Heimspiel" und „Auswärtsspiel" aktiv
- **THEN** ist „Vergangene anzeigen" aktiviert

#### Scenario: Page lädt mit Textfilter aus URL
- **WHEN** ein User `/termine?q=ludwigsburg` aufruft
- **THEN** ist das Textfeld beim ersten Render mit `ludwigsburg` vorbelegt
- **THEN** zeigt die Liste ausschließlich passende Termine

#### Scenario: Textfilter kombiniert mit Team-Filter
- **WHEN** ein User `/termine?team=3&q=ludwigsburg` aufruft
- **THEN** bleiben nur Termine sichtbar, die Team 3 zugeordnet sind **und** auf den Textfilter passen

#### Scenario: Filteränderung schreibt URL zurück
- **WHEN** ein User auf der `/termine`-Seite den Team-Filter auf Team 5 ändert
- **THEN** wird die URL via `replaceState` auf `/termine?team=5` aktualisiert
- **THEN** verändert sich der Browser-History-Stack nicht (kein neuer Eintrag pro Filteränderung)

#### Scenario: Tippen erzeugt keinen History-Eintrag pro Zeichen
- **WHEN** ein User acht Zeichen in das Textfeld tippt
- **THEN** enthält der History-Stack keinen zusätzlichen Eintrag
- **THEN** steht nach kurzer Verzögerung der vollständige Ausdruck als `q` in der URL

#### Scenario: Ungültiger Query-Parameter
- **WHEN** ein User `/termine?team=abc&types=foo,bar` aufruft
- **THEN** verhält sich die Seite wie ohne Filter (Default-State, kein Fehler)

#### Scenario: Leerer Textfilter ist wirkungslos
- **WHEN** ein User `/termine?q=` oder `/termine?q=%20%20` aufruft
- **THEN** ist die sichtbare Menge identisch zu `/termine` ohne Parameter

#### Scenario: Kein Query-Parameter (Rückwärtskompatibilität)
- **WHEN** ein User `/termine` ohne Query-Parameter aufruft
- **THEN** ist das Verhalten identisch zu vorher (Default-Filter, keine sichtbare Änderung)
