## ADDED Requirements

### Requirement: Verbindlicher Button-Klassen-String
Jeder Button in `web/src/` SHALL exakt einen der drei definierten Klassen-Strings verwenden. Abweichungen sind nicht erlaubt.

**Primary:**
`bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

**Small (in Tabellen, eingebettet):**
`bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

**Danger (destruktive Aktionen):**
`bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

#### Scenario: Primary Button rendert korrekt
- **WHEN** ein Primary-Button gerendert wird
- **THEN** hat er gelben Hintergrund, schwarzen Text, `rounded-md`, und `py-2.5` auf Mobile (`sm:py-2` auf Desktop)

#### Scenario: Danger Button ersetzt alle destruktiven Varianten
- **WHEN** ein Lösch- oder Ablehnen-Button gerendert wird
- **THEN** verwendet er `bg-brand-danger` (Karmesin), nicht `text-red-600`, `bg-red-100`, oder `bg-black`

#### Scenario: Disabled-Zustand ist einheitlich
- **WHEN** ein Button `disabled` ist
- **THEN** hat er `opacity-40` und `cursor-not-allowed`, das Basis-Layout bleibt erhalten

---

### Requirement: Verbindlicher Input-Klassen-String
Alle `<input>`, `<select>` und `<textarea>` SHALL einen der zwei definierten Klassen-Strings verwenden.

**Standard:**
`w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`

**Klein (in Tabellen, Filterzeilen):**
`w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow`

#### Scenario: Focus-Ring ist einheitlich gelb
- **WHEN** ein Input fokussiert wird
- **THEN** erscheint ein `ring-brand-yellow`-Focus-Ring, kein blauer Ring

#### Scenario: Kein Input ohne Focus-Ring
- **WHEN** der Code auf Inputs ohne `focus:ring-*` geprüft wird
- **THEN** gibt es keine Treffer

---

### Requirement: Verbindlicher Card-Klassen-String
Alle Panel-Container SHA folgende Varianten verwenden:

**Standard:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6`
**Kompakt:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4`
**Tabellen-Container:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`

Modals erhalten ebenfalls `border-t-4 border-brand-yellow`.

#### Scenario: Karte hat oberen Farbstreifen
- **WHEN** eine Standard-Karte gerendert wird
- **THEN** hat sie `border-t-4 border-brand-yellow` als oberen Akzent-Streifen

#### Scenario: Modal hat oberen Farbstreifen
- **WHEN** ein Modal geöffnet wird
- **THEN** hat auch das Modal `border-t-4 border-brand-yellow`

---

### Requirement: Verbindlicher Alert-Klassen-String
Alle Hinweisboxen SHALL einen der vier semantischen Alert-Typen verwenden:

**Info:** `p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text`
**Fehler:** `p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger`
**Erfolg:** `p-3 bg-brand-success-light border border-brand-success/40 rounded-lg text-sm text-brand-text`
**Warnung:** `p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-sm text-brand-text`

#### Scenario: Kein Alert mit raw Tailwind-Farbe
- **WHEN** Alerts auf `bg-blue-50`, `bg-red-50`, `bg-yellow-50`, `bg-amber-50` geprüft werden
- **THEN** gibt es keine Treffer — alle Alerts nutzen `brand-*`-Tokens

#### Scenario: Fehler-Alert ist Karmesin
- **WHEN** ein Fehler-Alert gerendert wird
- **THEN** hat er `bg-brand-danger-light` Hintergrund und `text-brand-danger` Text (`#C0253A`)

---

### Requirement: Verbindlicher Tabellen-Klassen-String
Alle `<table>`-Strukturen SHALL folgende Klassen verwenden:

**Container:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`
**Header-TH:** `bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left`
**Row-TR:** `hover:bg-brand-table-select transition-colors`
**Data-TD:** `px-4 py-3 text-sm text-brand-text`

#### Scenario: Row-Hover ist brand-table-select
- **WHEN** der Cursor über eine Tabellenzeile bewegt wird
- **THEN** färbt sich der Hintergrund auf `#E5E7EB` (brand-table-select), nicht `#F9FAFB` (gray-50)

---

### Requirement: Button-Position auf Seiten
- Listen-Seiten (mit Tabelle): Primär-Button MUSS oben rechts neben `<h1>` erscheinen
- Formular-Seiten (ganzseitiges Formular): Primär-Button MUSS unten im Formular erscheinen
- Karten mit Inline-Form: Button MUSS unten in der Karte erscheinen

#### Scenario: Listen-Seite hat Button oben rechts
- **WHEN** eine Listenseite (MembersPage, AdminUsersPage, AdminTeamsPage, AdminDutyTypesPage) gerendert wird
- **THEN** erscheint der „Neu anlegen"-Button in der gleichen Zeile wie die Überschrift, rechtsbündig

---

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
