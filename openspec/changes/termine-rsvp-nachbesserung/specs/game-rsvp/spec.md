## ADDED Requirements

### Requirement: Trainer-Default `confirmed` in `GET /api/games/my`

`GET /api/games/my` SHALL für einen aufrufenden User, der über `kader_trainers` Trainer eines am Spiel beteiligten Teams ist und **keine** eigene `game_responses`-Row hat, `my_rsvp='confirmed'` als virtuellen Default liefern (Priorität: explizite Response > Stammkader-Default > Erweitert-Default > Trainer-`confirmed` > `null`). Ohne Beziehung zum Spiel bleibt `my_rsvp=null`.

#### Scenario: Trainer ohne Response sieht confirmed
- **WHEN** ein User Trainer eines beteiligten Teams eines Spiels ist und keine `game_responses`-Row hat
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp='confirmed'`

#### Scenario: Fremder Funktionsträger sieht keinen Default
- **WHEN** ein Vorstand (kein Trainer/Spieler/Erweiterter eines beteiligten Teams) das Spiel sieht und keine Response hat
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp=null`

## REMOVED Requirements

### Requirement: Konflikt-Sperre „standardmäßig abgesagt" plus „Grund erforderlich" (Spiele)

**Reason**: Die Sperre verhinderte eine Kombination, die tatsächlich widerspruchsfrei ist: `rsvp_default_*='declined'` wirkt nur auf Mitglieder ohne Reaktion (virtuelle Absage, nie eine `game_responses`-Row, nie ein Grund erfragt), während `rsvp_require_reason=1` ausschließlich beim **aktiven** Absagen greift. Zudem erzwingt der Server `rsvp_require_reason` an keiner Stelle.

**Migration**: Keine Datenmigration nötig. `POST /api/games` und `PUT /api/games/{id}` akzeptieren `declined` zusammen mit `rsvp_require_reason=1` ab sofort ohne Fehler; der frühere HTTP-400 `invalid_rsvp_settings` entfällt ersatzlos.
