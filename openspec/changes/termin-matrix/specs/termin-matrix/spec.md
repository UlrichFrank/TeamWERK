## ADDED Requirements

### Requirement: Rückmelde-Matrix einer Mannschaft abrufen

Das System SHALL unter `GET /api/teams/{id}/rsvp-matrix?from=YYYY-MM-DD&to=YYYY-MM-DD`
die Termine einer Mannschaft im Zeitraum als Spalten (`events`) und die Spieler ihres
Kaders der aktiven Saison als Zeilen (`members`) liefern. Jede Zeile SHALL ein Array
`cells` tragen, dessen i-ter Eintrag zum i-ten Termin in `events` gehört.

Spalten SHALL alle Trainings der Mannschaft (auch abgesagte, mit `cancelled=true`) und
alle Spiele/Events, an denen die Mannschaft über `game_teams` beteiligt ist, mit Datum
im Zeitraum umfassen, sortiert nach Datum und Uhrzeit. Jede Spalte SHALL `kind`
(`training`|`game`), `id`, `date` (`YYYY-MM-DD`), `time`, `event_type`
(`training`|`heim`|`auswärts`|`generisch`), `title` und `cancelled` tragen.

Zeilen SHALL den Stammkader und — ohne Doppelte — den erweiterten Kader umfassen, mit
`member_id`, `name` und `extended`. Trainer SHALL keine Zeile bekommen.

Eine Zelle SHALL `status` (`confirmed`|`declined`|`maybe`|`null`) und `is_default`
tragen. Ohne Antwortzeile SHALL die Rollen-Voreinstellung des Termins
(`rsvp_default_players` für Stammkader, `rsvp_default_extended` für den erweiterten
Kader) angewandt werden, sofern sie `confirmed` oder `declined` ist, mit
`is_default=true`. Bei einer Serien-Abmeldung, die das Training abdeckt, SHALL die
Zelle `unavailable=true` tragen.

#### Scenario: Status aus Antwort und Voreinstellung
- **WHEN** ein Kaderspieler auf ein Training mit `declined` geantwortet hat und für ein Spiel mit `rsvp_default_players='confirmed'` keine Antwort vorliegt
- **THEN** trägt seine Training-Zelle `status='declined'`, `is_default=false`
- **AND** seine Spiel-Zelle `status='confirmed'`, `is_default=true`

#### Scenario: Fremdes Spiel erscheint nicht
- **WHEN** im Zeitraum ein Spiel liegt, an dem die Mannschaft nicht beteiligt ist
- **THEN** erscheint es nicht in `events`

#### Scenario: Ohne aktive Saison
- **WHEN** keine Saison aktiv ist
- **THEN** antwortet die Route mit 200 und leerer `members`-Liste

#### Scenario: Ungültiger Zeitraum
- **WHEN** `from` fehlt, kein Datum ist, nach `to` liegt oder der Zeitraum 400 Tage überschreitet
- **THEN** antwortet die Route mit 400

### Requirement: Sichtbarkeit der Matrix

Die Matrix SHALL für `admin`, `vorstand`, `sportliche_leitung` und Nutzer mit der
Vereinsfunktion `trainer` abrufbar sein, sonst nur für Nutzer, denen
`user_accessible_teams` die Mannschaft in der aktiven Saison zuordnet. Andernfalls SHALL
die Route 403 liefern, bei unbekannter Mannschaft 404.

Die Matrix SHALL **keine** Absagegründe enthalten. Die erfasste Anwesenheit (`present`)
SHALL nur für Admin, sportliche Leitung und Trainer der Mannschaft in der aktiven
Saison enthalten sein; für alle anderen SHALL das Feld fehlen.

#### Scenario: Unbeteiligter Nutzer
- **WHEN** ein Nutzer ohne Vereinsfunktion und ohne Bezug zur Mannschaft die Matrix abruft
- **THEN** antwortet die Route mit 403

#### Scenario: Anwesenheit nur für Trainer
- **WHEN** für ein vergangenes Training Anwesenheit erfasst ist
- **THEN** enthält die Zelle für den Trainer der Mannschaft `present`
- **AND** für einen Spieler derselben Mannschaft fehlt `present`

### Requirement: Tabellenansicht auf /termine

Die Seite `/termine` SHALL über einen Umschalter in der Kopfzeile zwischen Liste
(Default) und Tabelle wechseln; der Zustand SHALL im URL-Parameter `view=tabelle`
stehen. In der Tabellenansicht SHALL genau eine Mannschaft gewählt sein (`team=<id>`,
Einfachauswahl über die Mannschaften des Nutzers); Übungsgruppen SHALL nicht zur
Auswahl stehen. Typ-Filter und „Vergangene" SHALL auf die Spalten wirken.

Die Tabelle SHALL je Spieler eine Spalte „Teilnahme" zeigen: Anzahl der Zusagen (bzw.
der erfassten Anwesenheiten, wo vorhanden) und deren Anteil an den sichtbaren, nicht
abgesagten und nicht per Serie abgemeldeten Terminen. Die Namensspalte SHALL beim
horizontalen Scrollen stehen bleiben; ein Spaltenkopf SHALL zur Termin-Detailseite
führen. Die Ansicht SHALL sich bei `trainings`/`games`-Events live aktualisieren.

#### Scenario: Umschalten auf die Tabelle
- **WHEN** ein Nutzer auf `/termine` „Tabelle" wählt
- **THEN** enthält die URL `view=tabelle` und `team=<id>`
- **AND** die Seite zeigt eine Zeile je Kaderspieler und eine Spalte je Termin

#### Scenario: Typ-Filter wirkt auf Spalten und Quote
- **WHEN** in der Tabellenansicht nur „Training" aktiv ist
- **THEN** zeigt die Tabelle nur Trainingsspalten
- **AND** die Teilnahme-Quote rechnet nur über diese Spalten
