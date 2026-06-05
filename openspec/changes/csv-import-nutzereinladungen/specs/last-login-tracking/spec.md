## ADDED Requirements

### Requirement: Letzter Login wird gespeichert
Das System SHALL bei jedem erfolgreichen Login den Zeitpunkt in `users.last_login_at` (DATETIME, nullable) speichern.

#### Scenario: Erfolgreicher Login setzt last_login_at
- **WHEN** ein Nutzer sich erfolgreich mit E-Mail und Passwort anmeldet
- **THEN** setzt das System `users.last_login_at = CURRENT_TIMESTAMP` für diesen Nutzer

#### Scenario: Erstmaliger Login
- **WHEN** ein Nutzer sich zum ersten Mal anmeldet (bisher `last_login_at IS NULL`)
- **THEN** setzt das System `last_login_at` auf den aktuellen Zeitpunkt

### Requirement: Letzter Login wird in der Nutzerverwaltung angezeigt
Das System SHALL `last_login_at` im `GET /api/admin/users`-Response mitliefern, und das Frontend SHALL diesen Wert in der Nutzertabelle anzeigen.

#### Scenario: Nutzer hat sich bereits eingeloggt
- **WHEN** ein Admin die Nutzerverwaltung aufruft
- **THEN** zeigt die Tabelle für jeden Nutzer den letzten Login-Zeitpunkt in lesbarer Form an (z.B. „vor 3 Tagen")

#### Scenario: Nutzer hat sich noch nie eingeloggt
- **WHEN** ein Nutzer `last_login_at IS NULL` hat (noch nie eingeloggt oder Einladung noch nicht angenommen)
- **THEN** zeigt das Frontend „Noch nie" oder „–" an
