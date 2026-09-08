## MODIFIED Requirements

### Requirement: Feed-Generierung

Das System SHALL unter `GET /api/calendar/feed/{token}.ics` ohne Authentifizierung eine valide iCal-Datei (RFC 5545) zurückgeben. Der Token identifiziert den User und die Einstellungen. Die Datei enthält alle aktivierten Events des Users als VEVENT-Einträge.

Content-Type SHALL `text/calendar; charset=utf-8` sein. Zeilenenden SHALL CRLF sein. Lange Zeilen SHALL bei 75 Oktetten gefaltet werden, dabei SHALL der Schnitt **nie innerhalb einer UTF-8-Mehrbyte-Sequenz** liegen. Text-Felder SHALL `\`, `,`, `;` und Zeilenumbrüche escapen.

Jedes `VEVENT` SHALL ein `DTSTAMP` tragen. Der Kalender SHALL eine `VTIMEZONE`-Komponente für `Europe/Berlin` enthalten, solange `DTSTART`/`DTEND` den Parameter `TZID=Europe/Berlin` verwenden.

#### Scenario: Feed mit gültigem Token

- **WHEN** ein Calendar-Client `GET /api/calendar/feed/{token}.ics` aufruft
- **THEN** antwortet das System mit HTTP 200 und Content-Type `text/calendar; charset=utf-8`
- **AND** der Body beginnt mit `BEGIN:VCALENDAR` und endet mit `END:VCALENDAR`
- **AND** die Datei enthält für jedes aktivierte Event einen `BEGIN:VEVENT … END:VEVENT`-Block

#### Scenario: Feed mit ungültigem oder gelöschtem Token

- **WHEN** ein Client `GET /api/calendar/feed/{unbekannter-token}.ics` aufruft
- **THEN** antwortet das System mit HTTP 404

#### Scenario: Kalender enthält die referenzierte Zeitzone

- **WHEN** der Feed mindestens ein Event enthält
- **THEN** enthält der Kalender genau eine `BEGIN:VTIMEZONE … END:VTIMEZONE`-Komponente mit `TZID:Europe/Berlin`
- **AND** sie steht vor dem ersten `BEGIN:VEVENT`

#### Scenario: Jedes VEVENT trägt DTSTAMP

- **WHEN** der Feed ein Event beliebigen Typs enthält
- **THEN** enthält der zugehörige `BEGIN:VEVENT … END:VEVENT`-Block eine `DTSTAMP`-Zeile im UTC-Format `YYYYMMDDTHHmmssZ`

#### Scenario: Unveränderte Daten liefern denselben DTSTAMP

- **WHEN** derselbe Feed zweimal abgerufen wird, ohne dass sich die zugrunde liegenden Termine geändert haben
- **THEN** tragen die Events in beiden Antworten denselben `DTSTAMP`-Wert

#### Scenario: Faltung zerteilt kein Mehrbyte-Zeichen

- **WHEN** ein Feld so lang ist, dass es gefaltet werden muss, und an der 75-Oktett-Grenze ein Mehrbyte-Zeichen steht (z. B. `·`, `ä`, `–`)
- **THEN** liegt der Schnitt vor diesem Zeichen, nicht in ihm
- **AND** jede einzelne Zeile der Ausgabe ist für sich gültiges UTF-8
- **AND** keine Zeile überschreitet 75 Oktette

#### Scenario: VEVENT-Struktur für ein Spiel

- **WHEN** der Feed ein Heimspiel mit Venue enthält
- **THEN** hat das VEVENT:
  - `SUMMARY:Heim: Team (<Mannschaft>) – <Gegner>` (Heimspiel) oder `SUMMARY:Auswärts: <Gegner> – Team (<Mannschaft>)` (Auswärtsspiel). `<Mannschaft>` ist der Name des Teams, über dessen Kader der Feed-Nutzer am Spiel hängt (z. B. `mA1`, `gD`); ohne auflösbaren Teamnamen bleibt es bei `Team`. Hängt der Nutzer nur über den erweiterten Kader am Spiel, folgen in derselben Klammer der Kader-Zusatz und der Aufstellungsstatus (`<Mannschaft> · erw. Kader · <Status>`)
  - `DTSTART;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>`
  - `DTEND;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>` (aus `end_time`/`end_date`; fehlen diese, DURATION:PT2H)
  - `LOCATION:<Venue-Name>, <Street>, <PostalCode> <City>` (wenn Venue vorhanden)
  - `UID:game-<id>@teamwerk`

#### Scenario: VEVENT-Struktur für einen Dienst

- **WHEN** der Feed einen Dienst des Users enthält
- **THEN** hat das VEVENT:
  - `SUMMARY:Dienst: <duty_type_name> – <event_name>`
  - `DTSTART;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>` (aus event_date + event_time)
  - `DTEND;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>` — Start zuzüglich der Dauer aus `duty_slots.hours_value`
  - `LOCATION:<Venue-Name>, <Street>, <PostalCode> <City>` — im selben Format wie beim Spiel, aufgelöst über den Spielbezug des Slots (wenn vorhanden)
  - `DESCRIPTION:<role_desc>` (wenn gepflegt)
  - `UID:duty-<duty_slot_id>@teamwerk`

#### Scenario: Dienstdauer folgt hours_value

- **WHEN** ein Dienst mit Startzeit `13:30` und `hours_value = 3.0` im Feed erscheint
- **THEN** lautet das `DTEND` `16:30` desselben Tages
- **AND** ein Dienst mit `hours_value = 1.5` endet 90 Minuten nach seinem Start

#### Scenario: Dienst ohne Uhrzeit ist ein Ganztags-Event

- **WHEN** ein Dienst ohne `event_time` im Feed erscheint
- **THEN** trägt sein VEVENT `DTSTART;VALUE=DATE:<YYYYMMDD>` und `DTEND;VALUE=DATE:<YYYYMMDD des Folgetags>`
- **AND** es entsteht kein Termin, der um Mitternacht beginnt und die Dauer aus `hours_value` belegt

#### Scenario: Dienst mit nicht-positiver Dauer

- **WHEN** ein Dienst mit `hours_value <= 0` im Feed erscheint
- **THEN** endet sein VEVENT eine Stunde nach dem Start
- **AND** es entsteht kein Event mit `DTEND <= DTSTART`

#### Scenario: Dienst am Spiel erbt den Spielort

- **WHEN** ein Dienst zu einem Spiel gehört, dem ein Venue zugeordnet ist
- **THEN** trägt das Dienst-VEVENT dieselbe `LOCATION` wie das Spiel-VEVENT desselben Spiels

#### Scenario: Dienst ohne Spielbezug bleibt ohne Ort

- **WHEN** ein Dienst keinen Spielbezug hat oder das zugehörige Spiel kein Venue trägt
- **THEN** enthält sein VEVENT keine `LOCATION`-Zeile

#### Scenario: Dienst ohne Rollenbeschreibung bleibt ohne DESCRIPTION

- **WHEN** ein Dienst keine gepflegte Rollenbeschreibung hat
- **THEN** enthält sein VEVENT keine `DESCRIPTION`-Zeile

#### Scenario: Training-Event im Feed (include_training=true)

- **WHEN** der Feed aktiviert ist und für ein Team des Users eine aktive `training_sessions`-Row existiert
- **THEN** erscheint sie im Feed mit `SUMMARY:Training: <team_name>` (erweiterter Kader: `<team_name> · erw. Kader`), `UID:training-<id>@teamwerk` und `LOCATION:<location>`
- **AND** `DTSTART`/`DTEND` werden aus `date`, `start_time`, `end_time` gebildet
