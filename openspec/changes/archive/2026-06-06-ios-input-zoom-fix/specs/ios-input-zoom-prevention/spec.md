## ADDED Requirements

### Requirement: Eingabefelder verhindern iOS-Auto-Zoom
Alle `input`-, `textarea`- und `select`-Elemente der Anwendung SHALL eine effektive `font-size` von mindestens 16px haben, damit iOS Safari keinen automatischen Zoom beim Fokussieren auslöst.

#### Scenario: Login-Formular fokussieren
- **WHEN** ein iOS-Nutzer ein Eingabefeld im Login-Formular antippt
- **THEN** bleibt der Viewport-Zoom unverändert (kein automatischer Zoom)

#### Scenario: Suchfeld fokussieren
- **WHEN** ein iOS-Nutzer das Suchfeld auf einer Listen-Seite antippt
- **THEN** bleibt der Viewport-Zoom unverändert

#### Scenario: Formularfeld in einem Modal fokussieren
- **WHEN** ein iOS-Nutzer ein Eingabefeld in einem geöffneten Modal antippt
- **THEN** bleibt der Viewport-Zoom unverändert

#### Scenario: Kein Zoom-Reset nötig
- **WHEN** ein iOS-Nutzer ein Eingabefeld verlässt (blur)
- **THEN** ändert sich der Viewport-Zoom nicht, da er nie gesetzt wurde

### Requirement: Visuelles Design bleibt unverändert
Das Setzen der Basis-`font-size` auf 16px SHALL keine sichtbaren Layout- oder Größenänderungen in der Anwendung verursachen.

#### Scenario: Inputs sehen wie zuvor aus
- **WHEN** die CSS-Regel aktiv ist
- **THEN** wirken alle Eingabefelder optisch identisch wie vor dem Fix, weil Tailwind-Klassen (`text-sm` etc.) die Darstellung weiterhin kontrollieren können
