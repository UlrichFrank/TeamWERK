## ADDED Requirements

### Requirement: Dienst-CSV über einen wählbaren Zeitraum

`GET /api/duty-slots/export?from=YYYY-MM-DD&to=YYYY-MM-DD` SHALL eine CSV-Datei
(`Content-Type: text/csv; charset=utf-8`, `Content-Disposition: attachment`,
`;` als Trennzeichen, führendes UTF-8-BOM) mit **einer Zeile je Dienst-Slot** im
Zeitraum liefern, chronologisch nach Datum und Startzeit sortiert.

Beide Grenzen SHALL Pflicht sein und in der Form `YYYY-MM-DD` kommen; fehlt eine, ist
sie nicht parsebar oder liegt `from` nach `to`, SHALL der Server mit HTTP 400 und
`{"error":"invalid_range"}` antworten. Ein Export ohne Zeitraum wäre ein Voll-Dump der
Tabelle und ist in keiner Aufrufstelle gewollt.

Der Bereichsvergleich SHALL auf der auf zehn Zeichen gekürzten `event_date` erfolgen
(`substr(event_date,1,10)`), damit ein als ISO-Timestamp gespeichertes Datum
(SQLite-DATE-Gotcha) nicht am Bereichsende verloren geht. Beide Grenzen sind inklusive.

Das BOM ist verpflichtend: ohne es zeigt Excel die Umlaute der Spalten- und
Diensttyp-Namen als Mojibake, weil der Charset aus dem Content-Type den Datei-Import
nicht erreicht. Aus demselben Grund SHALL die Dauer mit Dezimalkomma ausgegeben werden.

#### Scenario: Dienst an einem Heimspiel liefert alle Zeiten
- **WHEN** ein Vorstand den Zeitraum eines Monats exportiert, in dem ein Dienst um 17:15 mit 1,5 h Dauer an einem Heimspiel um 18:00 liegt
- **THEN** enthält die Zeile Beginn `17:15`, Ende `18:45`, Dauer `1,50` und den Anwurf `18:00`

#### Scenario: Bereichsgrenzen sind inklusive, ISO-Timestamp inbegriffen
- **WHEN** der Zeitraum `2099-08-01` bis `2099-08-31` exportiert wird und ein Slot `event_date = '2099-08-31T00:00:00Z'` trägt
- **THEN** erscheint dieser Slot in der Datei
- **AND** Slots vom 31.07. und 01.09. erscheinen nicht

#### Scenario: Zeitraum fehlt oder ist verdreht
- **WHEN** die Route ohne `from`, ohne `to`, mit einem deutschen Datum oder mit `from > to` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Termin-Kontext je Dienst-Zeile

Jede Zeile SHALL den Kontext des Termins tragen, in dem der Dienst steht: den für den
Spieltag geltenden **Ausrichter** samt der Angabe, ob er für diesen Tag explizit gesetzt
oder vom Default geerbt ist, sowie die **Tageskonstellation** — Anzahl der Spiele am Tag,
deren Anwurfzeiten, und ob am Vor- bzw. Folgetag ein Heimspiel liegt.

Diese Größen SHALL exakt dieselben sein, die die Regen-Engine liest
(`loadSameDayContextTx`): die Anwurfzeiten aus **allen** Spielen des Tages derselben
Saison, Vor-/Folgetag nur aus Spielen mit `is_home = 1`. Der Ausrichter SHALL über
`settings.ResolveAusrichterForDayDetailed` aufgelöst werden — die Auflösung ist total,
die Spalte ist also nie leer, solange die Default-Zeile existiert.

Zusätzlich SHALL die am Diensttyp konfigurierte Regel für beide Nachbarschaftsfälle
ausgegeben werden (`same_day_behavior`, `adjacent_day_behavior`), bei `reduced`
einschließlich des Namens der Ziel-Variante.

Der Export SHALL die Wirkung dieser Regeln **nicht** selbst berechnen. Ob eine Regel für
einen konkreten Slot greift, hängt zusätzlich von seiner Lage zwischen den Anwurfzeiten
ab (`classifySlotPosition`); diese Entscheidung bleibt allein in
`internal/games/regen.go`. Der Export gibt die Eingangsgrößen und die Regel aus, damit
der Leser sie nachvollziehen kann — ein zweiter Nachbau der Entscheidung würde
unbemerkt driften.

#### Scenario: Mehrere Spiele am Tag und Nachbartage werden ausgewiesen
- **WHEN** an einem Tag zwei Spiele mit Anwurf 18:00 und 20:00 liegen und sowohl am Vor- als auch am Folgetag ein Heimspiel stattfindet
- **THEN** trägt die Zeile `Spiele am Tag = 2`, `Anwurfzeiten am Tag = "18:00, 20:00"`, `Heimspiel Vortag = ja`, `Heimspiel Folgetag = ja`

#### Scenario: Explizit gesetzter Tages-Ausrichter
- **WHEN** für den Spieltag eine Zeile in `spieltag_ausrichter` auf „TSV Nachbarort" zeigt
- **THEN** nennt die Zeile diesen Ausrichter und `Ausrichter für Tag gesetzt = ja`

#### Scenario: Tag ohne eigenen Eintrag erbt den Default
- **WHEN** für den Spieltag keine Zeile in `spieltag_ausrichter` existiert
- **THEN** nennt die Zeile den Default-Ausrichter und `Ausrichter für Tag gesetzt = nein`

#### Scenario: Reduktions-Variante wird benannt
- **WHEN** der Diensttyp `same_day_behavior='reduced'` mit der Variante „Zeitnahme kurz" und `adjacent_day_behavior='skip'` trägt
- **THEN** stehen in der Zeile `"reduziert → Zeitnahme kurz"` und `"entfällt"`

---

### Requirement: Der Export trägt keine Belegung und keine Namen

Die CSV SHALL **keine** Angaben zur Besetzung enthalten: keine Namen von Zugewiesenen,
keine Belegungs-, Offen- oder Erledigt-Spalten. Ausgegeben wird ausschließlich die
geplante Platzzahl (`slots_total`).

Damit ist das Blatt frei von personenbezogenen Daten und darf an Ausrichter ohne
TeamWERK-Zugang weitergegeben werden — genau der Anwendungsfall, für den der Export
existiert. Die Belegung sieht man in der Dienstbörse, wo sie an der Zugriffsprüfung des
jeweiligen Slots hängt.

#### Scenario: Zugewiesene Person erscheint nicht in der Datei
- **WHEN** ein Slot eine Zuweisung mit `slots_filled = 1` hat und der Zeitraum exportiert wird
- **THEN** enthält die Datei weder den Namen der Person noch eine Belegungsspalte

---

### Requirement: Zugriff im Tier der Dienst-Slot-Pflege

Die Route SHALL unter `RequireClubFunction("vorstand", "trainer", "sportliche_leitung")`
liegen — dasselbe Tier wie `POST/PUT/DELETE /api/duty-slots` und die Capability
`manage_duties`. Der Menüpunkt im Frontend SHALL an `manage_duties` hängen.

Bewusst weiter als `bulk_regen_duties` und `import_games` (Vorstand/Admin): diese Routen
sind enger, weil sie hunderte Slots schreiben. Der Export liest nur und trägt keine
personenbezogenen Daten.

#### Scenario: Trainer darf exportieren
- **WHEN** Persona `trainer` die Route mit gültigem Zeitraum aufruft
- **THEN** antwortet der Server nicht mit 403
- **AND** sieht der Trainer den Menüpunkt „Dienste als CSV" auf `/kalender`

#### Scenario: Spieler wird abgewiesen
- **WHEN** Persona `spieler` die Route aufruft
- **THEN** antwortet der Server mit HTTP 403
