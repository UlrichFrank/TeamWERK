## ADDED Requirements

### Requirement: Nav-Eintrag Mitfahrgelegenheiten
Im AppShell-Navigationsbereich „Dienste" erscheint ein neuer Eintrag „Mitfahrgelegenheiten" unterhalb von „Dienste" (nach dem bestehenden Dienste-Link, nach Kalender).

#### Scenario: Alle Rollen sehen den Eintrag
- **WHEN** ein authentifizierter Nutzer beliebiger Rolle eingeloggt ist
- **THEN** ist „Mitfahrgelegenheiten" in der Sidebar sichtbar (roles: [])

#### Scenario: Navigation zur Seite
- **WHEN** der Nutzer auf „Mitfahrgelegenheiten" klickt
- **THEN** navigiert er zu `/mitfahrgelegenheiten`

### Requirement: Route
`/mitfahrgelegenheiten` ist eine Route in App.tsx, geschützt hinter dem Authenticated-Wrapper, rendern `MitfahrgelegenheitenPage`.
