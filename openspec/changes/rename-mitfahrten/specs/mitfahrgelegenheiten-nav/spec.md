## MODIFIED Requirements

### Requirement: Nav-Eintrag Mitfahrgelegenheiten
Im AppShell-Navigationsbereich „Dienste" SHALL ein Eintrag „Mitfahrten" unterhalb von „Dienste" (nach dem bestehenden Dienste-Link, nach Kalender) erscheinen, sichtbar für alle authentifizierten Rollen.

#### Scenario: Alle Rollen sehen den Eintrag
- **WHEN** ein authentifizierter Nutzer beliebiger Rolle eingeloggt ist
- **THEN** ist „Mitfahrten" in der Sidebar sichtbar (roles: [])

#### Scenario: Navigation zur Seite
- **WHEN** der Nutzer auf „Mitfahrten" klickt
- **THEN** navigiert er zu `/mitfahrten`

### Requirement: Route
`/mitfahrten` SHALL eine Route in App.tsx sein, geschützt hinter dem Authenticated-Wrapper, und die `MitfahrtenPage`-Komponente rendern.

#### Scenario: Route rendert MitfahrtenPage
- **WHEN** ein eingeloggter Nutzer `/mitfahrten` direkt aufruft
- **THEN** wird `MitfahrtenPage` gerendert
