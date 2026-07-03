## ADDED Requirements

### Requirement: RSVP-Voreinstellung pro Rolle (Trainings)

Jede Trainings-Session und Trainings-Serie SHALL für Stammkader-Spieler und den Erweiterten Kader **unabhängig** eine der drei Voreinstellungen tragen: `confirmed` („standardmäßig zugesagt"), `declined` („standardmäßig abgesagt"), `none` („keine automatische Rückmeldung"). Die Spalten heißen `rsvp_default_players` und `rsvp_default_extended` (TEXT NOT NULL DEFAULT `'none'` mit `CHECK` auf die drei Werte). Trainer haben KEINE Voreinstellungs-Spalte und werden weiterhin hart als `confirmed` behandelt.

Die Voreinstellung wird **virtuell** angewendet: fehlt zu einem Mitglied eine `training_responses`-Row, liefert die API den passenden Default-Status. Es werden dabei KEINE Rows in `training_responses` erzeugt.

#### Scenario: Stammkader-Spieler ohne Response bei `players='confirmed'`
- **WHEN** eine Session `rsvp_default_players='confirmed'` hat
- **AND** ein Mitglied ist über `kader_members` im Stammkader und hat keine `training_responses`-Row
- **THEN** liefert `GET /api/training-sessions/{id}/attendances` für dieses Mitglied `rsvp_status='confirmed'` und `rsvp_is_default=true`

#### Scenario: Erweiterter Kader unabhängig von Stammkader
- **WHEN** eine Session `rsvp_default_players='confirmed'` und `rsvp_default_extended='none'` hat
- **AND** ein Mitglied ist nur über `kader_extended_members` beteiligt und hat keine Response
- **THEN** liefert die API für dieses Mitglied `rsvp_status=null` (kein Default) und `rsvp_is_default=false`

#### Scenario: Default „standardmäßig abgesagt" wird angezeigt
- **WHEN** eine Session `rsvp_default_extended='declined'` hat
- **AND** ein Erweitertes-Kader-Mitglied hat keine Response
- **THEN** liefert die API `rsvp_status='declined'` und `rsvp_is_default=true`

#### Scenario: Aktive Response überschreibt Default
- **WHEN** dieselbe Session `rsvp_default_players='confirmed'` hat und ein Stammkader-Spieler hat `training_responses.status='declined'`
- **THEN** liefert die API `rsvp_status='declined'` und `rsvp_is_default=false`

---

### Requirement: Konflikt-Sperre „standardmäßig abgesagt" plus „Grund erforderlich" (Trainings)

Das System SHALL `PUT /api/training-sessions/{id}` und `PUT /api/training-series/{id}` mit HTTP 400 (`{"error":"invalid_rsvp_settings"}`) ablehnen, wenn der Payload gleichzeitig `rsvp_require_reason=1` UND (`rsvp_default_players='declined'` ODER `rsvp_default_extended='declined'`) enthält. Grund: eine Default-Absage entsteht ohne Nutzerhandlung, ein erzwungener Grund ist dann nicht erhebbar.

#### Scenario: Session-Update mit widersprüchlicher Kombination
- **WHEN** `PUT /api/training-sessions/{id}` mit `{"rsvp_default_players":"declined","rsvp_require_reason":1}` gerufen wird
- **THEN** antwortet der Server mit HTTP 400 und dem Body enthält `"invalid_rsvp_settings"`
- **THEN** wird KEINE Änderung an der Session gespeichert

#### Scenario: Serie-Update mit widersprüchlicher Kombination auf Erweitertem Kader
- **WHEN** `PUT /api/training-series/{id}` mit `{"rsvp_default_extended":"declined","rsvp_require_reason":1}` gerufen wird
- **THEN** antwortet der Server mit HTTP 400 und die Serie bleibt unverändert

---

### Requirement: Header-Zähler bezieht Voreinstellungen ein (Trainings)

`GET /api/training-sessions/{id}` sowie die aggregierte Session-Liste SHALL in `confirmed_count`, `declined_count` und `pending_count` Mitglieder mit virtuellem Default-Status ihrer Rolle mitzählen — nach der Formel `COALESCE(training_responses.status, session.rsvp_default_<role>)`, wobei `'none'` nirgends mitzählt. Trainer bleiben (unverändert) aus allen drei Zählern ausgeschlossen.

#### Scenario: Zähler bei `players='confirmed'` ohne Responses
- **WHEN** eine Session `rsvp_default_players='confirmed'` hat und 3 Stammkader-Spieler ohne Response existieren
- **THEN** enthält der Session-Response `confirmed_count=3` und `declined_count=0`

#### Scenario: Zähler bei `extended='declined'` ohne Responses
- **WHEN** eine Session `rsvp_default_extended='declined'` hat und 2 Erweiterte-Kader-Mitglieder ohne Response existieren
- **THEN** enthält der Session-Response `declined_count=2`

#### Scenario: Zähler ignoriert Default `'none'`
- **WHEN** beide Voreinstellungen `'none'` sind und keine Responses existieren
- **THEN** sind `confirmed_count=0`, `declined_count=0`, `pending_count` = Anzahl der spieler-orientierten Zeilen
