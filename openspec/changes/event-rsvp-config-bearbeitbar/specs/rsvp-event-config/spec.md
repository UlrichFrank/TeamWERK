## MODIFIED Requirements

### Requirement: rsvp-opt-out-flag
Jeder Termin (training_session, game) MUSS ein `rsvp_opt_out`-Flag besitzen (INTEGER 0/1).
Bei `rsvp_opt_out = 1` gilt ein Spieler ohne Response-Eintrag als "confirmed".
Das Flag MUSS beim Anlegen einer Session von der zugehörigen training_series kopiert werden.
Das Flag MUSS nach dem Anlegen für berechtigte Nutzer änderbar sein (siehe Requirement
`rsvp-config-edit-ui`).

#### Scenario: Spieler ohne Eintrag bei Opt-Out-Termin
- **WHEN** ein training_session oder game hat `rsvp_opt_out = 1` und ein Spieler hat keinen Eintrag in der Response-Tabelle
- **THEN** gibt `my_rsvp` den Wert `"confirmed"` zurück

#### Scenario: confirmed_count bei Opt-Out
- **WHEN** ein Termin hat `rsvp_opt_out = 1`
- **THEN** ist `confirmed_count` gleich der Anzahl explizit bestätigter Einträge plus der Anzahl Team-Mitglieder ohne Response-Eintrag

#### Scenario: Zusagen-Button vorausgewählt
- **WHEN** `my_rsvp = "confirmed"` (implizit oder explizit)
- **THEN** zeigt die TerminePage den Zusagen-Button als aktiv/ausgewählt

#### Scenario: Session erbt Flag von Serie
- **WHEN** eine neue training_session aus einer training_series erstellt wird
- **THEN** werden `rsvp_opt_out` und `rsvp_require_reason` von der Serie kopiert

#### Scenario: Flag nach Anlegen änderbar
- **WHEN** ein berechtigter Nutzer (admin, trainer, sportliche_leitung, vorstand) ein bestehendes game oder eine bestehende training_session bearbeitet
- **THEN** KÖNNEN `rsvp_opt_out` und `rsvp_require_reason` geändert werden; der neue Wert wird persistiert und beeinflusst alle künftigen Response-Auswertungen (z.B. `confirmed_count`, `my_rsvp`-Default)

## ADDED Requirements

### Requirement: rsvp-config-edit-ui
Berechtigte Nutzer (admin, trainer, sportliche_leitung, vorstand) MÜSSEN den aktuell konfigurierten
Wert von `rsvp_opt_out` und `rsvp_require_reason` eines bestehenden Spiels bzw. einer bestehenden
training_session und training_series im Edit-Modal sehen und ändern können.
Das UI MUSS dieselben zwei Checkboxen wie beim Anlegen zeigen (siehe Requirement
`rsvp-config-creation-ui`), vorbelegt mit den aktuellen DB-Werten. Beim Speichern werden die
Felder im PUT-Payload mitgesendet.

#### Scenario: Bearbeiten eines Spiels
- **WHEN** ein Trainer ein bestehendes Spiel im `GameEditModal` öffnet
- **THEN** zeigt das Modal Checkboxen für `rsvp_opt_out` und `rsvp_require_reason`, vorbelegt mit dem aktuellen DB-Wert
- **THEN** sendet „Speichern" die Werte an `PUT /api/games/{id}`

#### Scenario: Bearbeiten einer Trainings-Session
- **WHEN** ein Trainer eine bestehende training_session bearbeitet
- **THEN** zeigt das Edit-Formular Checkboxen für `rsvp_opt_out` und `rsvp_require_reason`, vorbelegt mit dem aktuellen DB-Wert
- **THEN** sendet „Speichern" die Werte an `PUT /api/training-sessions/{id}`

#### Scenario: Bearbeiten einer Trainings-Serie
- **WHEN** ein Trainer eine bestehende training_series bearbeitet
- **THEN** zeigt das Edit-Formular Checkboxen für `rsvp_opt_out` und `rsvp_require_reason`, vorbelegt mit dem aktuellen DB-Wert; die Sperre „nur bei neuer Serie editierbar" entfällt
- **THEN** sendet „Speichern" die Werte an `PUT /api/training-series/{id}`

#### Scenario: Partial-Update lässt Wert unverändert
- **WHEN** ein PUT-Request für ein game oder eine training_session/training_series die RSVP-Felder NICHT enthält
- **THEN** bleiben `rsvp_opt_out` und `rsvp_require_reason` in der DB unverändert (kein impliziter Reset auf 0)

#### Scenario: Spieler darf nicht ändern
- **WHEN** ein Nutzer mit Rolle `spieler` einen PUT-Request mit `rsvp_opt_out` oder `rsvp_require_reason` an `/api/games/{id}`, `/api/training-sessions/{id}` oder `/api/training-series/{id}` schickt
- **THEN** antwortet der Server mit Status 403 ohne die DB zu ändern

### Requirement: rsvp-config-status-badge
Die Detailansicht eines Termins (Spiel, training_session) MUSS den aktuellen Wert von
`rsvp_opt_out` und `rsvp_require_reason` visuell sichtbar machen, sodass der konfigurierte
RSVP-Modus auch ohne Edit-Modal erkennbar ist.

#### Scenario: Badge "Opt-Out aktiv"
- **WHEN** ein Termin hat `rsvp_opt_out = 1`
- **THEN** zeigt die Detailansicht ein Badge „Opt-Out aktiv" (oder semantisch gleichwertig) im Termin-Header

#### Scenario: Badge "Begründung Pflicht"
- **WHEN** ein Termin hat `rsvp_require_reason = 1`
- **THEN** zeigt die Detailansicht ein Badge „Begründung bei Absage Pflicht" (oder semantisch gleichwertig)

#### Scenario: Keine Badges bei Default-Konfig
- **WHEN** ein Termin hat `rsvp_opt_out = 0` UND `rsvp_require_reason = 0`
- **THEN** zeigt die Detailansicht keines der beiden Badges

## REMOVED Requirements

### Requirement: rsvp-opt-out-flag-Scenario "Flag beim Bearbeiten eingefroren"
**Reason:** Die Anforderung, dass `rsvp_opt_out` und `rsvp_require_reason` nach dem Anlegen
eingefroren sind, war eine vereinfachende Annahme und macht Konfigurationsfehler im Betrieb
unbehebbar. Insbesondere bei Spielen ist sie nie konsequent umgesetzt worden (das Backend kennt
diese Felder im UpdateGame gar nicht). Ersetzt durch das neue Requirement `rsvp-config-edit-ui`.

**Migration:** Das Scenario „Flag beim Bearbeiten eingefroren" entfällt ersatzlos. Bestehende
Daten bleiben unverändert; Frontend-Sperren (`disabled={!isNewSeries}` in `AdminTrainingsPage`)
werden entfernt. Das Scenario `Session erbt Flag von Serie` bleibt unverändert bestehen — nur
das anschließende Einfrieren entfällt.
