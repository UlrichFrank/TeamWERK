## ADDED Requirements

### Requirement: Token-Verwaltung

Das System SHALL pro User genau ein Calendar-Token verwalten. Ein Token ist ein UUID-v4-String, der in `calendar_tokens` zusammen mit 5 Boolean-Toggles gespeichert wird. Jeder authentifizierte Nutzer kann sein Token anlegen, seine Einstellungen ändern oder das Token löschen.

`POST /api/calendar/token` ist idempotent: existiert bereits ein Token für den User, werden nur die Einstellungen aktualisiert; der Token-Wert bleibt unverändert. Bei Neuanlage wird ein UUID v4 via `crypto/rand` generiert.

#### Scenario: Erstes Token anlegen

- **WHEN** ein authentifizierter Nutzer `POST /api/calendar/token` mit `{"include_heim": true, "include_auswaerts": true, "include_training": true, "include_generisch": true, "include_duty": true}` aufruft
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": true, ...}`
- **AND** ein Eintrag in `calendar_tokens` mit dem generierten UUID und den Toggles existiert

#### Scenario: Einstellungen ändern (Token bleibt gleich)

- **WHEN** ein Nutzer mit bestehendem Token `POST /api/calendar/token` mit geänderten Toggles aufruft
- **THEN** antwortet das System mit HTTP 200 und den aktualisierten Einstellungen
- **AND** der `token`-Wert in der Response ist identisch zum bisherigen Token

#### Scenario: Token abrufen (existiert)

- **WHEN** ein Nutzer `GET /api/calendar/token` aufruft und ein Token besitzt
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": ..., ...}`

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

### Requirement: Feed-Generierung

Das System SHALL unter `GET /api/calendar/feed/{token}.ics` ohne Authentifizierung eine valide iCal-Datei (RFC 5545) zurückgeben. Der Token identifiziert den User und die Einstellungen. Die Datei enthält alle aktivierten Events des Users als VEVENT-Einträge.

Content-Type SHALL `text/calendar; charset=utf-8` sein. Zeilenenden SHALL CRLF sein. Lange Zeilen SHALL bei 75 Oktetten gefaltet werden. Text-Felder SHALL `\`, `,`, `;` und Zeilenumbrüche escapen.

#### Scenario: Feed mit gültigem Token

- **WHEN** ein Calendar-Client `GET /api/calendar/feed/{token}.ics` aufruft
- **THEN** antwortet das System mit HTTP 200 und Content-Type `text/calendar; charset=utf-8`
- **AND** der Body beginnt mit `BEGIN:VCALENDAR` und endet mit `END:VCALENDAR`
- **AND** die Datei enthält für jedes aktivierte Event einen `BEGIN:VEVENT … END:VEVENT`-Block

#### Scenario: Feed mit ungültigem oder gelöschtem Token

- **WHEN** ein Client `GET /api/calendar/feed/{unbekannter-token}.ics` aufruft
- **THEN** antwortet das System mit HTTP 404

#### Scenario: VEVENT-Struktur für ein Spiel

- **WHEN** der Feed ein Heimspiel mit Venue enthält
- **THEN** hat das VEVENT:
  - `SUMMARY:Heim: SG Stuttgart – <Gegner>` (Heimspiel) oder `SUMMARY:Auswärts: <Gegner> – SG Stuttgart` (Auswärtsspiel)
  - `DTSTART;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>`
  - `DTEND;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>` (aus `end_time`/`end_date`; fehlen diese, DURATION:PT2H)
  - `LOCATION:<Venue-Name>, <Street>, <PostalCode> <City>` (wenn Venue vorhanden)
  - `UID:game-<id>@teamwerk`

#### Scenario: VEVENT-Struktur für einen Dienst

- **WHEN** der Feed einen Dienst des Users enthält
- **THEN** hat das VEVENT:
  - `SUMMARY:Dienst: <duty_type_name> – <event_name>`
  - `DTSTART;TZID=Europe/Berlin:<YYYYMMDDTHHmmss>` (aus event_date + event_time; fehlt event_time: T000000)
  - `UID:duty-<duty_slot_id>@teamwerk`

#### Scenario: Training-Event im Feed (include_training=true)

- **WHEN** der Feed aktiviert ist und für ein Team des Users eine aktive `training_sessions`-Row existiert
- **THEN** erscheint sie im Feed mit `SUMMARY:Training: <team_name>`, `UID:training-<id>@teamwerk` und `LOCATION:<location>`
- **AND** `DTSTART`/`DTEND` werden aus `date`, `start_time`, `end_time` gebildet

### Requirement: Konfigurierbare Feed-Inhalte

Das System SHALL die im Token gespeicherten Toggles beim Feed-Abruf auswerten. Ein deaktivierter Toggle bewirkt, dass die entsprechenden Events nicht im iCal ausgegeben werden.

Spiele werden dem User zugeordnet wenn er über `kader_members` Mitglied eines Teams ist, dem das Spiel via `game_teams` zugeordnet ist, und die aktive Saison übereinstimmt.

Trainings werden dem User zugeordnet wenn er über `kader_members` Mitglied eines Teams ist, das `team_id` in einer `training_sessions`-Row mit `status='active'` referenziert.

Dienste werden dem User zugeordnet wenn ein Eintrag in `duty_assignments` mit `user_id = user_id_des_tokens` und `status IN ('assigned', 'fulfilled')` existiert.

#### Scenario: include_training=false filtert training_sessions heraus

- **WHEN** ein Token mit `include_training=false` existiert und der User Mitglied eines Teams ist, für das eine `training_sessions`-Row existiert
- **THEN** enthält der Feed kein VEVENT mit `UID:training-*`
- **AND** alle anderen aktivierten Event-Typen sind weiterhin enthalten

#### Scenario: include_duty=false filtert Dienste heraus

- **WHEN** ein Token mit `include_duty=false` existiert und der User einen zugewiesenen Dienst hat
- **THEN** enthält der Feed kein VEVENT mit `UID:duty-*`

#### Scenario: Alle Toggles deaktiviert — leerer Feed

- **WHEN** alle 5 Toggles auf false gesetzt sind
- **THEN** enthält der Feed einen validen VCALENDAR-Rahmen aber keine VEVENTs
