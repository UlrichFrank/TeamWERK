## ADDED Requirements

### Requirement: Nav-Eintrag Mitfahrten
Im AppShell-Navigationsbereich „Dienste" SHALL ein Eintrag „Mitfahrten" unterhalb des Dienste-Links erscheinen (nach Kalender), sichtbar für alle authentifizierten Rollen.

#### Scenario: Alle Rollen sehen den Eintrag
- **WHEN** ein authentifizierter Nutzer beliebiger Rolle eingeloggt ist
- **THEN** ist „Mitfahrten" in der Sidebar sichtbar (roles: [])

#### Scenario: Navigation zur Seite
- **WHEN** der Nutzer auf „Mitfahrten" klickt
- **THEN** navigiert er zu `/mitfahrten`

### Requirement: Route
`/mitfahrten` SHALL als Route in `App.tsx` registriert sein, geschützt hinter dem Authenticated-Wrapper, und die `MitfahrtenPage`-Komponente rendern.

#### Scenario: Route ist registriert
- **WHEN** ein eingeloggter Nutzer `/mitfahrten` direkt aufruft
- **THEN** wird `MitfahrtenPage` gerendert

#### Scenario: Unbekannte Route /mitfahrgelegenheiten
- **WHEN** ein Nutzer den alten Pfad `/mitfahrgelegenheiten` aufruft
- **THEN** zeigt die App den Standard-Nicht-gefunden-Zustand (kein Redirect — alte Bookmarks sind nicht supportet)
