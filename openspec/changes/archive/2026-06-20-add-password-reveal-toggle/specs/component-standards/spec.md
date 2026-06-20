## ADDED Requirements

### Requirement: Passwort-Felder nutzen `<PasswordInput>`

Jedes Passwort-Eingabefeld in `web/src/` SHALL durch die Shared-Komponente `<PasswordInput>` aus `web/src/components/forms/PasswordInput.tsx` gerendert werden. Direkte Verwendung von `<input type="password">` ist nicht erlaubt — Ausnahme: die Komponente selbst.

Die Komponente kapselt:

- Den Standard-Input-Klassen-String (siehe Requirement „Verbindlicher Input-Klassen-String") plus `pr-10` Padding-right
- Den Reveal-Toggle gemäß Capability `password-reveal-toggle`
- Den Wrapper-`<div class="relative">` für die absolute Toggle-Positionierung

Der Aufrufer SHALL den `autoComplete`-Prop explizit setzen (Pflicht-Prop):

- `"current-password"` für Login-/Aktuelles-Passwort-Felder
- `"new-password"` für Felder bei Registrierung, Passwort-Reset und neuem Passwort
- `"off"` nur für besondere Ausnahmen

#### Scenario: LoginPage nutzt die Komponente

- **WHEN** die LoginPage gerendert wird
- **THEN** kommt das Passwort-Feld aus `<PasswordInput>`, nicht direkt aus `<input type="password">`

#### Scenario: Keine direkte `<input type="password">`-Verwendung

- **WHEN** der Code in `web/src/` außerhalb von `components/forms/PasswordInput.tsx` auf `type="password"` geprüft wird
- **THEN** gibt es keine Treffer

#### Scenario: `autoComplete` ist Pflicht

- **WHEN** `<PasswordInput>` ohne `autoComplete`-Prop verwendet wird
- **THEN** schlägt der TypeScript-Build mit einem Type-Error fehl
