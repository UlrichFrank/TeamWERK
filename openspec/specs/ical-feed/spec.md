# ical-feed Specification

## Purpose

Diese Spezifikation beschreibt die Capability `ical-feed`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Token-Verwaltung

Das System SHALL pro User genau ein Calendar-Token verwalten. Ein Token ist ein UUID-v4-String, der in `calendar_tokens` zusammen mit 6 Boolean-Toggles gespeichert wird (`include_heim`, `include_auswaerts`, `include_training`, `include_generisch`, `include_duty`, `include_practice_groups`). Jeder authentifizierte Nutzer kann sein Token anlegen, seine Einstellungen ändern oder das Token löschen.

`POST /api/calendar/token` ist idempotent: existiert bereits ein Token für den User, werden nur die Einstellungen aktualisiert; der Token-Wert bleibt unverändert. Bei Neuanlage wird ein UUID v4 via `crypto/rand` generiert.

#### Scenario: Erstes Token anlegen

- **WHEN** ein authentifizierter Nutzer `POST /api/calendar/token` mit `{"include_heim": true, "include_auswaerts": true, "include_training": true, "include_generisch": true, "include_duty": true, "include_practice_groups": true}` aufruft
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": true, ...}`
- **AND** ein Eintrag in `calendar_tokens` mit dem generierten UUID und den Toggles existiert

#### Scenario: Einstellungen ändern (Token bleibt gleich)

- **WHEN** ein Nutzer mit bestehendem Token `POST /api/calendar/token` mit geänderten Toggles aufruft
- **THEN** antwortet das System mit HTTP 200 und den aktualisierten Einstellungen
- **AND** der `token`-Wert in der Response ist identisch zum bisherigen Token

#### Scenario: Token abrufen (existiert)

- **WHEN** ein Nutzer `GET /api/calendar/token` aufruft und ein Token besitzt
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": ..., "include_practice_groups": ...}`

#### Scenario: Token abrufen (existiert nicht)

- **WHEN** ein Nutzer `GET /api/calendar/token` aufruft und kein Token besitzt
- **THEN** antwortet das System mit HTTP 404

#### Scenario: Token löschen

- **WHEN** ein Nutzer `DELETE /api/calendar/token` aufruft
- **THEN** antwortet das System mit HTTP 204
- **AND** der Feed-Endpunkt unter dem bisherigen Token-Pfad liefert danach HTTP 404

#### Scenario: Nicht-authentifizierter Zugriff auf Token-Management

- **WHEN** ein nicht eingeloggter Client `GET /api/calendar/token` aufruft
- **THEN** antwortet das System mit HTTP 401

#### Scenario: Bestandstoken nach der Migration

- **WHEN** ein vor der Migration angelegtes Token abgerufen wird
- **THEN** trägt es `include_practice_groups: true`

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

### Requirement: Konfigurierbare Feed-Inhalte

Das System SHALL die im Token gespeicherten Toggles beim Feed-Abruf auswerten. Ein
deaktivierter Toggle bewirkt, dass die entsprechenden Events nicht im iCal ausgegeben werden.

Spiele und Trainings werden dem User über seine Kader-Zugehörigkeit zugeordnet. Als
Zugehörigkeit zählen `kader_members` (regulär), `kader_extended_members` (erweiterter Kader)
und `kader_trainers` (Trainer) — dieselbe Menge, die `auth.GameVisibilityClause` für den
Spielplan auflöst. Für Spiele muss das Team via `game_teams` am Spiel hängen und die Saison
des Kaders mit der des Spiels übereinstimmen; für Trainings muss der Kader des Nutzers der
`kader_id` einer `training_sessions`-Row mit `status='active'` entsprechen. Die Auflösung
läuft für Trainings über `training_sessions.kader_id` und **nicht** über `team_id` — sonst
fielen Übungsgruppen strukturell heraus, deren `team_id` NULL ist.

Den Funktionsträger-Bypass aus `auth` (admin/trainer/sportliche_leitung/vorstand sehen alle
Events der Saison) übernimmt der Feed bewusst NICHT — ein Vorstand hätte sonst den gesamten
Vereinsspielplan im privaten Kalender. Die Auflösung über `family_links` entfällt ebenfalls,
weil Eltern für jedes Kind einen eigenen Kind-Token bekommen.

Hängt ein User über mehrere Kader an demselben Termin, erscheint er trotzdem nur einmal im
Feed (die UID ist pro Spiel bzw. Training eindeutig); für die Beschriftung schlägt dabei eine
reguläre Zugehörigkeit die erweiterte. Damit entfällt für diesen Nutzer auch der
Aufstellungsstatus — er hängt regulär am Termin.

Dienste werden dem User zugeordnet wenn ein Eintrag in `duty_assignments` mit
`user_id = user_id_des_tokens` und `status IN ('assigned', 'fulfilled')` existiert.

#### Scenario: Termine aus dem erweiterten Kader sind gekennzeichnet

- **WHEN** ein User über `kader_extended_members` am Kader eines Teams hängt und für dieses
  Team ein Spiel und ein Training existieren
- **THEN** enthält der Feed beide Events
- **AND** die Mannschaft trägt im `SUMMARY` den Zusatz `<team_name> · erw. Kader` (z. B.
  `SUMMARY:Training: mB1 · erw. Kader`)
- **AND** das Spiel-Event trägt zusätzlich den Aufstellungsstatus (z. B.
  `SUMMARY:Heim: Team (mB1 · erw. Kader · aufgestellt) – <Gegner>`)

#### Scenario: Vereinsfunktion allein zieht keine fremden Termine in den Feed

- **WHEN** ein User die Vereinsfunktion `trainer` hat, aber an keinem Kader des Teams hängt,
  dem ein Spiel zugeordnet ist
- **THEN** enthält der Feed dieses Spiel nicht

#### Scenario: include_training=false filtert training_sessions heraus

- **WHEN** ein Token mit `include_training=false` existiert und der User Mitglied eines Teams
  ist, für das eine `training_sessions`-Row existiert
- **THEN** enthält der Feed kein VEVENT mit `UID:training-*` für diesen Mannschaftstermin
- **AND** alle anderen aktivierten Event-Typen sind weiterhin enthalten

#### Scenario: include_duty=false filtert Dienste heraus

- **WHEN** ein Token mit `include_duty=false` existiert und der User einen zugewiesenen Dienst
  hat
- **THEN** enthält der Feed kein VEVENT mit `UID:duty-*`

#### Scenario: Alle Toggles deaktiviert — leerer Feed

- **WHEN** alle 6 Toggles auf false gesetzt sind
- **THEN** enthält der Feed einen validen VCALENDAR-Rahmen aber keine VEVENTs

### Requirement: Aufstellungsstatus im Spiel-Event des erweiterten Kaders

Für ein Spiel-Event vom Typ `heim` oder `auswärts`, an dem der Feed-Nutzer **ausschließlich
über den erweiterten Kader** (`kader_extended_members`) hängt, SHALL der Feed den
Aufstellungsstatus dieses Nutzers ausweisen — in der Mannschafts-Klammer des `SUMMARY` als
kurzes Kennwort und im `DESCRIPTION` als vollständiger Satz.

Der Status SHALL drei Zustände kennen, abgeleitet aus `game_lineup` für das jeweilige Spiel:

| Zustand | Bedingung | Kennwort im `SUMMARY` | Satz im `DESCRIPTION` |
|---|---|---|---|
| aufgestellt | eine Zeile für dieses Spiel **und** dieses Mitglied | `aufgestellt` | „Du bist für das Spiel aufgestellt." |
| nicht aufgestellt | mindestens eine Zeile für dieses Spiel, aber keine für dieses Mitglied | `nicht aufgestellt` | „Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist." |
| offen | **keine** Zeile für dieses Spiel | `Aufstellung offen` | „Die Aufstellung für dieses Spiel steht noch nicht fest." |

Der Zustand „offen" SHALL eigenständig bleiben: aus einer leeren Aufstellung SHALL das
System **nicht** „nicht aufgestellt" ableiten. Eine nicht gepflegte Aufstellung ist keine
Nichtberücksichtigung, und der Feed SHALL dem Empfänger keine Absage melden, die niemand
ausgesprochen hat.

Der Satz SHALL an eine vorhandene Notiz des Termins angehängt werden, getrennt durch eine
Leerzeile; die Notiz SHALL dabei erhalten bleiben. Hat der Termin keine Notiz, SHALL das
`DESCRIPTION` allein den Satz tragen.

Nutzer, die über den regulären Kader (`kader_members`) oder als Trainer
(`kader_trainers`) am Spiel hängen, SHALL der Feed **ohne** Status ausweisen — für sie ist
die Teilnahme der Regelfall. Events vom Typ `generisch` SHALL keinen Status tragen; sie
haben keine Aufstellung.

#### Scenario: Aufgestellter Spieler des erweiterten Kaders

- **WHEN** ein Nutzer über `kader_extended_members` am Kader von `mB1` hängt, für das Heimspiel gegen `SG Weinstadt` eine Aufstellung gespeichert ist und sein Mitglied darin steht
- **THEN** lautet das `SUMMARY` `Heim: Team (mB1 · erw. Kader · aufgestellt) – SG Weinstadt`
- **AND** enthält das `DESCRIPTION` den Satz `Du bist für das Spiel aufgestellt.`

#### Scenario: Nicht aufgestellter Spieler des erweiterten Kaders

- **WHEN** für dasselbe Spiel eine Aufstellung gespeichert ist, das Mitglied des Nutzers aber nicht darin steht
- **THEN** trägt das `SUMMARY` in der Mannschafts-Klammer den Zusatz `· nicht aufgestellt`
- **AND** enthält das `DESCRIPTION` den Satz `Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist.`

#### Scenario: Aufstellung noch nicht gespeichert

- **WHEN** für das Spiel **keine** Zeile in `game_lineup` existiert
- **THEN** trägt das `SUMMARY` in der Mannschafts-Klammer den Zusatz `· Aufstellung offen`
- **AND** enthält das `DESCRIPTION` den Satz `Die Aufstellung für dieses Spiel steht noch nicht fest.`
- **AND** enthält der Feed an keiner Stelle die Aussage `NICHT aufgestellt`

#### Scenario: Notiz und Aufstellungssatz stehen beide im DESCRIPTION

- **WHEN** das Spiel eine Notiz trägt und der Nutzer über den erweiterten Kader daran hängt
- **THEN** enthält das `DESCRIPTION` zuerst die Notiz, dann eine Leerzeile, dann den Aufstellungssatz

#### Scenario: Regulärer Kader bekommt keinen Status

- **WHEN** ein Nutzer über `kader_members` am Kader des Teams hängt
- **THEN** lautet das `SUMMARY` `Heim: Team (mB1) – SG Weinstadt` ohne Kader- und ohne Status-Zusatz
- **AND** enthält das `DESCRIPTION` keinen Aufstellungssatz

#### Scenario: Doppelte Zugehörigkeit — regulär schlägt erweitert

- **WHEN** ein Nutzer an demselben Spiel sowohl über `kader_members` als auch über `kader_extended_members` hängt
- **THEN** enthält der Feed genau ein VEVENT für dieses Spiel
- **AND** trägt es weder den Kader- noch den Status-Zusatz

#### Scenario: Generisches Event trägt keinen Status

- **WHEN** ein Event vom Typ `generisch` im Feed erscheint und der Nutzer über den erweiterten Kader daran hängt
- **THEN** bleibt das `SUMMARY` der Terminname ohne Status-Zusatz
- **AND** enthält das `DESCRIPTION` keinen Aufstellungssatz

### Requirement: Übungsgruppen-Termine im Feed

Das System SHALL Trainingstermine von Übungsgruppen (`kader.kind='practice'`, erkennbar an
`training_sessions.team_id IS NULL`) im iCal-Feed ausgeben, wenn der Token
`include_practice_groups=true` trägt.

Die Zuordnung zum Feed-Nutzer erfolgt über dieselbe Kader-Zugehörigkeit wie bei
Mannschaftsterminen (`kader_members`, `kader_extended_members`, `kader_trainers`), aufgelöst
über `training_sessions.kader_id`. Der Funktionsträger-Bypass bleibt auch hier ausgeschlossen.

`include_practice_groups` und `include_training` sind voneinander unabhängig:
`include_training` steuert ausschließlich Termine mit `team_id IS NOT NULL`,
`include_practice_groups` ausschließlich Termine mit `team_id IS NULL`.

Beim Anlegen eines neuen Tokens ist `include_practice_groups` wie die übrigen Toggles
standardmäßig aktiv; bestehende Tokens erhalten den Wert `true` per Migration.

#### Scenario: Übungsgruppen-Termin erscheint im Feed

- **WHEN** ein User über `kader_members` an einer Übungsgruppe hängt, für die ein
  `training_sessions`-Eintrag mit `status='active'` existiert, und sein Token
  `include_practice_groups=true` trägt
- **THEN** enthält der Feed ein VEVENT mit `UID:training-<id>@teamwerk`

#### Scenario: Beschriftung eines Übungsgruppen-Termins

- **WHEN** ein Übungsgruppen-Termin im Feed ausgegeben wird und die Gruppe
  `kader.name = "Förderkinder 2016"` heißt
- **THEN** lautet das `SUMMARY` des VEVENT `Training: Förderkinder 2016`
- **AND** der Name stammt aus `kader.name`, nicht aus `training_sessions.title`

#### Scenario: include_practice_groups=false filtert Übungsgruppen heraus

- **WHEN** ein Token `include_practice_groups=false` und `include_training=true` trägt und
  der User sowohl an einer Mannschaft als auch an einer Übungsgruppe hängt, für die je ein
  Trainingstermin existiert
- **THEN** enthält der Feed den Mannschaftstermin
- **AND** enthält der Feed den Übungsgruppen-Termin nicht

#### Scenario: include_training=false lässt Übungsgruppen unberührt

- **WHEN** ein Token `include_training=false` und `include_practice_groups=true` trägt und
  der User sowohl an einer Mannschaft als auch an einer Übungsgruppe hängt, für die je ein
  Trainingstermin existiert
- **THEN** enthält der Feed den Übungsgruppen-Termin
- **AND** enthält der Feed den Mannschaftstermin nicht
