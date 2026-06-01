## ADDED Requirements

### Requirement: E-Mail im Kontaktdaten-Endpoint
Das System SHALL die E-Mail-Adresse eines Nutzers im `GET /api/users/:id/contact`-Response zurückgeben, sofern `email_visible = true` gesetzt ist.

#### Scenario: E-Mail freigegeben
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat `email_visible=true`
- **THEN** enthält die Response `{ ..., "email": "max@example.com" }`

#### Scenario: E-Mail nicht freigegeben
- **WHEN** Nutzer 42 hat `email_visible=false` (Default)
- **THEN** fehlt das `email`-Feld in der Response vollständig

### Requirement: E-Mail-Checkbox in den Profileinstellungen
Das System SHALL eine Checkbox „E-Mail-Adresse sichtbar" in den Sichtbarkeitseinstellungen des Profils anbieten.

#### Scenario: Nutzer aktiviert E-Mail-Sichtbarkeit
- **WHEN** ein Nutzer die Checkbox „E-Mail-Adresse sichtbar" aktiviert und speichert
- **THEN** wird `email_visible=1` in `user_visibility` gesetzt; der PersonChip anderer Nutzer zeigt ab dem nächsten Hover die E-Mail

#### Scenario: Default-Zustand
- **WHEN** ein Nutzer die Sichtbarkeitseinstellungen noch nie geändert hat
- **THEN** ist `email_visible=false` (opt-out)

### Requirement: E-Mail im PersonChip-Tooltip
Das System SHALL die E-Mail im Tooltip als klickbaren mailto:-Link darstellen.

#### Scenario: Tooltip mit E-Mail
- **WHEN** ein Nutzer den Tooltip einer Person öffnet, die `email_visible=true` gesetzt hat
- **THEN** erscheint die E-Mail-Adresse als anklickbarer Link (`<a href="mailto:...">`) im Tooltip

#### Scenario: Tooltip ohne E-Mail
- **WHEN** eine Person `email_visible=false` hat
- **THEN** erscheint kein E-Mail-Abschnitt im Tooltip
