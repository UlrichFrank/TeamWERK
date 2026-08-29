# Header-Button-Größen vereinheitlichen

## Why

Die Aktions-Buttons in den Seiten-Kopfzeilen haben vier verschiedene Höhen (28 px,
30 px, 38 px, 40 px), obwohl sie dieselbe Rolle spielen. Der Grund ist strukturell:
`component-standards` kennt nur drei Klassen-Strings — „Primary" (Formular-Größe,
`text-sm px-4 py-2.5 sm:py-2`) und „Small" (Tabellen-Größe, `text-xs px-3 py-1`) —
aber keinen für die Kopfzeile. Jede Seite hat sich daher einen der beiden geliehen
oder einen dritten erfunden. Es gibt keine `Button`-Komponente und keine geteilte
Konstante; 216 Klassen-Strings mit `bg-brand-yellow` sind über `pages/` und
`components/` von Hand kopiert.

Dazu kommt ein zweiter, unsichtbarer Fehler: alle diese Buttons leiten ihre Höhe aus
`py-*` ab. `index.css` zwingt unter 640 px aber nur `input`/`select`/`textarea` auf
`font-size: 16px`, Buttons nicht (Capability `ios-input-zoom-prevention`). Dieselbe
Padding-Klasse ergibt auf Mobile deshalb ein 41 px hohes Suchfeld neben einem 38 px
hohen Button — und keiner der Header-Buttons erreicht das 44-px-Touch-Target aus
`docs/agent/05-frontend.md`. `EventSearchInput` hat dieses Problem bereits gelöst
(fixe `h-[30px]` statt Padding) und dokumentiert die Begründung im Datei-Kommentar;
die Lösung ist nur nie verallgemeinert worden.

## What Changes

- **Neuer, vierter verbindlicher Klassen-String „Header-Control"** in
  `component-standards`: `h-11 sm:h-[30px] px-3 text-xs` mit `border`. Desktop 30 px
  (die heutige Höhe der Filterleiste und von `/mitglieder`), Mobile 44 px
  (Touch-Target). Höhe **fix**, nicht paddinggetrieben.
- **Neue geteilte Konstanten** in `web/src/lib/buttonStyles.ts`
  (`HEADER_CTRL`, `HEADER_CTRL_ICON`, `HEADER_PRIMARY`, `HEADER_NEUTRAL`,
  `HEADER_SPLIT_MAIN`, `HEADER_SPLIT_CARET`) sowie die drei bestehenden Strings
  (`BTN_PRIMARY`, `BTN_SMALL`, `BTN_DANGER`), damit es künftig eine Fundstelle gibt.
- **Alle Kopfzeilen-Aktionen** auf 13 Seiten ziehen auf den neuen String um.
- **Die Filterleiste zieht mit** (`EventSearchInput`, `EventTypeFilter`, die
  Toggle-Chips auf Termine/Dienste/Kalender/Mitfahrten, die Team-Selects). Ohne das
  bliebe die Leiste auf Mobile bei 30 px, während der Rest auf 44 px geht — genau die
  Ungleichheit, die dieser Change beseitigen soll.
- **Der 28-px-Compact-Sonderfall entfällt.** `EventSearchInput` unterscheidet heute
  `h-[30px]` / `h-7`, weil die Chips im Compact-Modus nur ein 14-px-Icon ohne Label
  enthalten und dadurch schrumpfen. Bei fixer Höhe verändert Compact nur noch Inhalt
  und Breite (`px-2` statt `px-3`), nicht die Höhe.
- **Die bewusste 44-px-Ausnahme der Filterleiste wird aufgehoben.** Der begründende
  Kommentar in `EventSearchInput.tsx` wird ersetzt: die Ausnahme war nötig, solange
  die Leiste als einzige Zeile 30 px hoch blieb; mit einheitlichen 44 px auf Mobile
  entfällt der Anlass.
- **Vier Abweichungen vom bestehenden Primary-String** werden nebenbei korrigiert
  (`DashboardPage` Retry-Button mit `rounded` statt `rounded-md`; drei lokale
  `BTN_PRIMARY`-Kopien in `pages/admin/`).
- **Neues Vitest-Gate**, das kopierte Metriken statt der Konstanten erkennt.
- **Nicht Teil dieses Changes:** Formular-/Modal-Buttons behalten ihre Größe
  (`text-sm`, 38/40 px) — Kopfzeilen-Aktion und Formular-Aktion bleiben zwei Rollen.
  Ebenso unangetastet: Pills/Badges (`rounded-full`), Tab-Leisten (`border-b-2`),
  Dropdown-Menüeinträge, Tabellen-Buttons (`BTN_SMALL`).

Keine Breaking Changes — die Änderung ist rein präsentational, es ändert sich kein
API-Vertrag und kein Verhalten.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `component-standards` — die Requirement „Verbindlicher Button-Klassen-String"
  bekommt einen vierten String (Header-Control) und die Regel, dass Header-Controls
  ihre Höhe fix und nicht über `py-*` festlegen.

## Impact

**Neue Datei:** `web/src/lib/buttonStyles.ts`, `web/src/lib/__tests__/buttonStyles.gate.test.ts`

**Kopfzeilen-Aktionen (13 Seiten, ~20 Call-Sites):**
`MembersPage`, `AdminUsersPage`, `VideosPage`, `KalenderPage`,
`AdminDutyTypesPage`, `AdminDutyTemplatesPage`, `DocumentsPage`, `AdminKaderPage`,
`AdminVenuesPage`, `AdminTrainingsPage`, `VideoUploadPage`,
`ProfilTrainingstagebuchPage`, `VideoDetailPage`

**Filterleiste (2 Komponenten + 5 Seiten):**
`components/EventSearchInput.tsx`, `components/EventTypeFilter.tsx`,
`DutyPage`, `TerminePage`, `KalenderPage`, `MitfahrgelegenheitenPage`,
`TeamTrainingstagebuchPage`

**Aufräumen:** `pages/admin/BeitragslaufPage.tsx`, `pages/admin/TresorPage.tsx`,
`pages/admin/WartungsmodusPage.tsx`, `pages/DashboardPage.tsx`

**Doku:** `docs/agent/05-frontend.md` (vierter verbindlicher Klassen-String),
`openspec/specs/component-standards/spec.md`

**Nicht betroffen:** Backend, DB, API-Routen, Tests außerhalb von `web/`.
Kein neuer RAM-Bedarf, keine neue Dependency, keine Berührung mit dem
Berechtigungsmodell — die Sichtbarkeits-Gates an den Buttons (`canUpload`,
`isAdmin`, Capabilities) bleiben unverändert, nur ihre Klassen-Strings ändern sich.

**Risiko:** Layout-Umbruch in Kopfzeilen, die bisher mit breiteren `px-4`/`text-sm`-
Buttons gerechnet haben — insbesondere `AdminVenuesPage` (Split-Button) und
`AdminKaderPage` (drei Buttons nebeneinander). Auf Mobile werden die Filterleisten
von Termine/Dienste/Kalender/Mitfahrten ~14 px höher; horizontal ändert sich nichts,
weil der Compact-Modus (Icons ohne Label ab < 950 px) unangetastet bleibt.
