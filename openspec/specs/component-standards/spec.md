# component-standards Specification

## Purpose

Diese Spezifikation beschreibt die Capability `component-standards`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Verbindlicher Button-Klassen-String
Jeder Button in `web/src/` SHALL exakt einen der vier definierten Klassen-Strings verwenden. Abweichungen sind nicht erlaubt.

**Header-Control (Aktion in der Seiten-Kopfzeile, Filter-Chip, Header-Select):**
`inline-flex items-center justify-center gap-1 rounded-md border h-8 sm:h-[30px] px-3 text-xs font-medium transition-colors shrink-0 disabled:opacity-40 disabled:cursor-not-allowed`
kombiniert mit einem Farbsatz:
- primär: `border-brand-yellow bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black`
- neutral / inaktiver Toggle: `bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text`
- destruktiv: `border-brand-danger bg-brand-danger text-white hover:bg-brand-danger/90`
- rahmenlos (Schließen, Abbrechen): `border-transparent bg-transparent text-brand-text-muted hover:bg-brand-table-select hover:text-brand-text`

Ein Farbsatz SHALL ausschließlich Farbe tragen und kein Maß (keine Höhe, Breite, Padding, Schriftgröße oder Rundung) — sonst wäre ein Control je nach Zustand verschieden groß. Ein zustandsabhängiger Farbsatz aus einer anderen Quelle (etwa `getEventColors(type).filter` für die Termin-Typ-Filter) ist zulässig, solange er dieselbe Bedingung erfüllt.

Der rahmenlose Farbsatz setzt `border-transparent` statt gar keines Rahmens, damit die Höhe die der Nachbarn bleibt.

Icon-only-Varianten (Compact-Modus, Split-Button-Caret) verwenden `px-2` statt `px-3`.
Split-Buttons ersetzen `rounded-md` durch `rounded-l-md` bzw. `rounded-r-md`.

**Primary (Formular- und Modal-Aktion):**
`bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

**Small (in Tabellen, eingebettet):**
`bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

**Danger (destruktive Aktionen):**
`bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed`

Header-Control und Primary sind zwei verschiedene Rollen und SHALL NOT gegeneinander ausgetauscht werden: Header-Control gilt für Bedienelemente in der Zeile der Seitenüberschrift, Primary für die Bestätigungs-Aktion eines Formulars oder Modals.

Header-Controls SHALL innerhalb einer Kopfzeile dieselbe Höhe haben — 30 px ab 640 px, 32 px darunter — und diese über eine feste Höhen-Utility festlegen, nicht über vertikales Padding. Grund ist die Regel aus `ios-input-zoom-prevention`: unterhalb von 640 px erzwingt `index.css` `font-size: 16px` für `input`, `select` und `textarea`, aber nicht für `button`. Aus derselben `py-*`-Klasse entstehen dort deshalb unterschiedliche Höhen, und eine Kopfzeile aus Suchfeld, Auswahl und Button läuft auseinander.

#### Scenario: Header-Controls einer Zeile sind gleich hoch
- **WHEN** eine Seiten-Kopfzeile mit Überschrift, Suchfeld, Auswahlfeld und Aktions-Button gerendert wird
- **THEN** haben Suchfeld, Auswahlfeld und Button dieselbe Höhe — 30 px ab 640 px, 32 px darunter
- **AND** das gilt auch unterhalb von 640 px, wo `index.css` die Schriftgröße der Eingabefelder auf 16 px zwingt

#### Scenario: Header-Controls sind vom 44-px-Touch-Target ausgenommen
- **WHEN** ein Header-Control auf einem Viewport < 640 px gerendert wird
- **THEN** ist es 32 px hoch und unterschreitet damit bewusst die 44-px-Regel aus `docs/agent/05-frontend.md`
- **AND** für Buttons außerhalb der Kopfzeile gilt die Regel unverändert

#### Scenario: Compact-Modus ändert die Höhe nicht
- **WHEN** die Filterleiste in den Compact-Modus wechselt und die Chips nur noch ihr Icon ohne Beschriftung zeigen
- **THEN** bleibt ihre Höhe unverändert; nur ihre Breite verringert sich

#### Scenario: Kopfzeilen-Aktion nutzt nicht die Formular-Größe
- **WHEN** der „Neu anlegen"-Button einer Listenseite gerendert wird
- **THEN** verwendet er den Header-Control-String (`text-xs`, feste Höhe), nicht den Primary-String (`text-sm`, `py-2.5 sm:py-2`)

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
Alle Panel-Container SHALL folgende Varianten verwenden:

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
Das System SHALL die Button-Position auf Seiten nach folgenden Regeln festlegen:
- Listen-Seiten (mit Tabelle): Primär-Button MUSS oben rechts neben `<h1>` erscheinen und den Header-Control-Klassen-String verwenden
- Formular-Seiten (ganzseitiges Formular): Primär-Button MUSS unten im Formular erscheinen und den Primary-Klassen-String verwenden
- Karten mit Inline-Form: Button MUSS unten in der Karte erscheinen und den Primary-Klassen-String verwenden

#### Scenario: Listen-Seite hat Button oben rechts
- **WHEN** eine Listenseite (MembersPage, AdminUsersPage, AdminTeamsPage, AdminDutyTypesPage) gerendert wird
- **THEN** erscheint der „Neu anlegen"-Button in der gleichen Zeile wie die Überschrift, rechtsbündig

#### Scenario: Position bestimmt die Größe
- **WHEN** derselbe fachliche Vorgang sowohl über einen Kopfzeilen-Button als auch über einen Modal-Button ausgelöst werden kann
- **THEN** trägt der Kopfzeilen-Button den Header-Control-String und der Modal-Button den Primary-String

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

### Requirement: Button-Klassen-Strings stammen aus einer geteilten Konstante
Die vier verbindlichen Klassen-Strings SHALL an genau einer Stelle im Frontend definiert sein (`web/src/lib/buttonStyles.ts`) und von den Aufrufern importiert werden. Ein Klassen-String SHALL NOT als Literal in einer Seite oder Komponente wiederholt werden.

Ein automatischer Test SHALL diese Regel prüfen und fehlschlagen, wenn eine der Metriken erneut inline auftaucht. Begründete Ausnahmen SHALL in einer Allowlist im Test stehen; ein Allowlist-Eintrag, dessen Fundstelle nicht mehr existiert, SHALL den Test ebenfalls fehlschlagen lassen.

#### Scenario: Kopierter Klassen-String fällt auf
- **WHEN** eine Seite den Primary- oder Header-Control-String als Literal enthält, statt ihn zu importieren
- **THEN** schlägt der Gate-Test mit Datei und Zeile der Fundstelle fehl

#### Scenario: Verwaiste Allowlist-Einträge fallen auf
- **WHEN** eine Ausnahme in der Allowlist steht, die zugehörige Fundstelle aber entfernt wurde
- **THEN** schlägt der Gate-Test fehl

