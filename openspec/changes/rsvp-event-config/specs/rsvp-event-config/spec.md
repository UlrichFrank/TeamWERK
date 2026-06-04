## ADDED Requirements

### Requirement: rsvp-opt-out-flag
Jeder Termin (training_session, game) MUSS ein `rsvp_opt_out`-Flag besitzen (INTEGER 0/1).
Bei `rsvp_opt_out = 1` gilt ein Spieler ohne Response-Eintrag als "confirmed".
Das Flag MUSS beim Anlegen einer Session von der zugehörigen training_series kopiert werden.
Nach dem Anlegen darf das Flag einer Session nicht mehr geändert werden.

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

#### Scenario: Flag beim Bearbeiten eingefroren
- **WHEN** eine bestehende training_session bearbeitet wird
- **THEN** DÜRFEN `rsvp_opt_out` und `rsvp_require_reason` nicht geändert werden; das Frontend blendet die Felder aus

### Requirement: rsvp-require-reason-flag
Jeder Termin MUSS ein `rsvp_require_reason`-Flag besitzen (INTEGER 0/1, DEFAULT 1).
Bei `rsvp_require_reason = 0` wird beim Klick auf Absagen/Vielleicht kein Modal geöffnet.
Bei `rsvp_require_reason = 1` MUSS vor dem Senden einer Absage/Vielleicht-RSVP ein
Modal mit Pflichtbegründung erscheinen.

#### Scenario: Direkt-RSVP ohne Begründung
- **WHEN** ein Termin hat `rsvp_require_reason = 0` und ein Spieler klickt Absagen oder Vielleicht
- **THEN** wird die RSVP sofort gesendet ohne Modal

#### Scenario: Pflichtbegründung via Modal
- **WHEN** ein Termin hat `rsvp_require_reason = 1` und ein Spieler klickt Absagen oder Vielleicht
- **THEN** öffnet sich ein Modal; der OK-Button ist disabled solange das Textfeld leer ist; erst nach Eingabe und OK-Klick wird die RSVP gesendet

#### Scenario: Abbrechen schliesst Modal ohne Aktion
- **WHEN** das Begründungs-Modal geöffnet ist und der Nutzer auf Abbrechen klickt
- **THEN** schließt das Modal ohne API-Call und ohne Statusänderung

### Requirement: rsvp-config-creation-ui
Beim Anlegen einer training_series oder eines Spiels MÜSSEN zwei Checkboxen sichtbar sein:
„Alle Spieler standardmäßig zugesagt (Opt-Out)" und „Begründung bei Absage erforderlich".
Bei `event_type = 'generisch'` MUSS `rsvp_require_reason` im Formular mit 0 vorbelegt sein.

#### Scenario: Konfiguration beim Anlegen
- **WHEN** ein Trainer eine neue training_series anlegt
- **THEN** kann er `rsvp_opt_out` und `rsvp_require_reason` über Checkboxen setzen

#### Scenario: Default für generische Events
- **WHEN** ein Trainer ein Spiel mit `event_type = 'generisch'` anlegt
- **THEN** ist `rsvp_require_reason` im Formular mit 0 vorbelegt
