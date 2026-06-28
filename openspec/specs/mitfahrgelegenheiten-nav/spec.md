# mitfahrgelegenheiten-nav Specification

## Purpose

Diese Spezifikation beschreibt die Capability `mitfahrgelegenheiten-nav`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Nav-Eintrag Mitfahrgelegenheiten
Im AppShell-Navigationsbereich „Dienste" SHALL ein neuer Eintrag „Mitfahrgelegenheiten" unterhalb von „Dienste" erscheinen (nach dem bestehenden Dienste-Link, nach Kalender).

#### Scenario: Alle Rollen sehen den Eintrag
- **WHEN** ein authentifizierter Nutzer beliebiger Rolle eingeloggt ist
- **THEN** ist „Mitfahrgelegenheiten" in der Sidebar sichtbar (roles: [])

#### Scenario: Navigation zur Seite
- **WHEN** der Nutzer auf „Mitfahrgelegenheiten" klickt
- **THEN** navigiert er zu `/mitfahrgelegenheiten`

### Requirement: Route
`/mitfahrgelegenheiten` SHALL eine Route in App.tsx sein, geschützt hinter dem Authenticated-Wrapper, die `MitfahrgelegenheitenPage` rendert.

#### Scenario: Route ist zugänglich für authentifizierte Nutzer

- **WHEN** ein authentifizierter Nutzer `/mitfahrgelegenheiten` aufruft
- **THEN** wird `MitfahrgelegenheitenPage` gerendert
