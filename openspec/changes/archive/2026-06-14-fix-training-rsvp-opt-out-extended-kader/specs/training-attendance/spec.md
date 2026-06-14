## MODIFIED Requirements

### Requirement: Trainer kann Anwesenheitsliste einer Session abrufen
Ein Trainer oder Admin SHALL die Anwesenheitsliste einer Session abrufen können, die beide Dimensionen zeigt: RSVP-Status (was angesagt wurde) und tatsächliche Anwesenheit. Jedes Element der Liste SHALL ein `is_extended`-Feld enthalten. Für primäre Kader-Mitglieder ohne explizite Rückmeldung gilt `rsvp_opt_out` der Session: ist es aktiv, wird ihr Status als `confirmed` ausgewiesen. Für erweiterte Kader-Mitglieder gilt `rsvp_opt_out` NICHT — ihr Status ist `null` wenn keine explizite Rückmeldung vorliegt, unabhängig von der Session-Konfiguration.

#### Scenario: Anwesenheitsliste abrufen
- **WHEN** ein Trainer GET `/api/training-sessions/{id}/attendances` aufruft
- **THEN** erhält er eine Liste aller Teammitglieder mit jeweils `member_id`, `member_name`, `rsvp_status`, `reason`, `present` und `is_extended`

#### Scenario: Primärer Kader mit rsvp_opt_out auto-confirmed
- **WHEN** eine Session `rsvp_opt_out = true` hat und ein primäres Kader-Mitglied keine Rückmeldung abgegeben hat
- **THEN** hat dieses Mitglied `rsvp_status: "confirmed"` in der Response

#### Scenario: Erweiterter Kader NICHT auto-confirmed
- **WHEN** eine Session `rsvp_opt_out = true` hat und ein erweitertes Kader-Mitglied keine Rückmeldung abgegeben hat
- **THEN** hat dieses Mitglied `rsvp_status: null` in der Response

#### Scenario: Erweiterter Kader mit expliziter Rückmeldung
- **WHEN** ein erweitertes Kader-Mitglied explizit `confirmed` oder `declined` geantwortet hat
- **THEN** wird dieser Status korrekt zurückgegeben, unabhängig von `rsvp_opt_out`

#### Scenario: Diskrepanz sichtbar
- **WHEN** ein Mitglied `rsvp_status='confirmed'` hat, aber `present=false`
- **THEN** sind beide Werte in der Liste sichtbar, sodass Trainer Zusagen ohne Erscheinen erkennen kann
