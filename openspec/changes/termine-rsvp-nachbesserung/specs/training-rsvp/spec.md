## ADDED Requirements

### Requirement: Trainer-Default `confirmed` in der Trainings-Session-Liste

`GET /api/training-sessions` SHALL für einen aufrufenden User, der über `kader_trainers` Trainer des Team-Kaders der Session ist und **keine** eigene `training_responses`-Row hat, `my_rsvp='confirmed'` als virtuellen Default liefern (Priorität: explizite Response > Stammkader-Default > Erweitert-Default > Trainer-`confirmed` > `null`). Für User, die keine Beziehung zur Session haben (weder Spieler, Erweiterter Kader noch Trainer dieses Teams), bleibt `my_rsvp=null`.

#### Scenario: Trainer ohne Response sieht confirmed
- **WHEN** ein User Trainer des Team-Kaders einer Session ist und keine `training_responses`-Row hat
- **THEN** liefert `GET /api/training-sessions` für diese Session `my_rsvp='confirmed'`

#### Scenario: Fremder Funktionsträger sieht keinen Default
- **WHEN** ein Vorstand (kein Trainer/Spieler/Erweiterter dieses Teams) die Session sieht und keine Response hat
- **THEN** liefert `GET /api/training-sessions` für diese Session `my_rsvp=null`

## REMOVED Requirements

### Requirement: Konflikt-Sperre „standardmäßig abgesagt" plus „Grund erforderlich" (Trainings)

**Reason**: Die Sperre verhinderte eine Kombination, die tatsächlich widerspruchsfrei ist: `rsvp_default_*='declined'` wirkt nur auf Mitglieder ohne Reaktion (virtuelle Absage, nie eine `training_responses`-Row, nie ein Grund erfragt), während `rsvp_require_reason=1` ausschließlich beim **aktiven** Absagen greift. Zudem erzwingt der Server `rsvp_require_reason` an keiner Stelle — die 400-Sperre bewachte eine sonst nirgends angewandte Regel.

**Migration**: Keine Datenmigration nötig. `PUT /api/training-sessions/{id}` und `PUT /api/training-series/{id}` akzeptieren `declined` zusammen mit `rsvp_require_reason=1` ab sofort ohne Fehler; der frühere HTTP-400 `invalid_rsvp_settings` entfällt ersatzlos.
