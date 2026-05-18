## ADDED Requirements

### Requirement: Admin kann aktive Einladungen auflisten
Das System SHALL einen `GET /api/admin/invitations` Endpunkt bereitstellen, der alle aktiven Einladungstoken zurückgibt. Aktiv bedeutet: `used_at IS NULL` AND `expires_at > jetzt`. Jeder Eintrag SHALL id, email, role, team_name (oder leer) und expires_at enthalten.

#### Scenario: Aktive Einladungen vorhanden
- **WHEN** `GET /api/admin/invitations` aufgerufen wird und ungenutzte, nicht abgelaufene Tokens existieren
- **THEN** antwortet das Backend mit HTTP 200 und einem JSON-Array der aktiven Einladungen

#### Scenario: Keine aktiven Einladungen
- **WHEN** `GET /api/admin/invitations` aufgerufen wird und keine aktiven Tokens vorhanden sind
- **THEN** antwortet das Backend mit HTTP 200 und einem leeren Array

#### Scenario: Nur Admins dürfen Einladungen abrufen
- **WHEN** ein Nicht-Admin `GET /api/admin/invitations` aufruft
- **THEN** antwortet das Backend mit HTTP 403

### Requirement: Nutzertabelle zeigt Einladungen und Anfragen
Die `AdminUsersPage` SHALL registrierte Nutzer, aktive Einladungen und offene Beitrittsanfragen in einer einzigen Tabelle darstellen. Typ-Unterscheidung erfolgt über Status-Badges.

#### Scenario: Unified Table mit gemischten Eintragstypen
- **WHEN** ein Admin die Nutzerverwaltungsseite aufruft
- **THEN** lädt die Seite alle drei Datenquellen parallel und zeigt sie in einer Tabelle: offene Anfragen und Einladungen zuerst (oben), registrierte Nutzer darunter

#### Scenario: Einladung in Tabelle erkennbar
- **WHEN** eine aktive Einladung in der Tabelle angezeigt wird
- **THEN** erscheint der Status-Badge mit der Beschriftung „Einladung" und die E-Mail-Adresse des Empfängers ist sichtbar
