## Context

Siehe `proposal.md` — Why. Für die Umsetzung sind drei Eigenschaften des Bestands
maßgeblich:

**1. Zwei Bauarten von Kopfzeilen.** Die Seiten teilen sich in zwei Muster, die
unterschiedlich auf Mobile reagieren:

- *Gestapelt* (`flex-col sm:flex-row`): `MembersPage`, `AdminUsersPage`,
  `VideosPage`, `DocumentsPage`, die Admin-Seiten. Auf Mobile stehen Überschrift,
  Suchfeld, Filter und Button untereinander, jeweils volle Breite.
- *Einzeilig* (`flex items-center gap-1.5 flex-nowrap min-w-0`): `TerminePage`,
  `DutyPage`, `KalenderPage`, `MitfahrgelegenheitenPage`. Diese Leiste bleibt auf
  jeder Breite eine Zeile und weicht dem Platzmangel über
  `useCompactHeader(950)` aus: die Typ-Chips zeigen ab < 950 px nur noch ihr Icon,
  einzelne Selects verschwinden per `hidden sm:block`.

Beide Muster bleiben, wie sie sind. Der Change vereinheitlicht nur die Höhe der
Bedienelemente darin.

**2. Die Höhe der einzeiligen Leiste ist bereits fixiert und begründet.**
`components/EventSearchInput.tsx` setzt `h-[30px]` (Compact: `h-7`) statt Padding
und erklärt im Datei-Kommentar warum: Padding läuft auf Mobile auseinander, weil
`index.css` nur Eingabefelder auf 16 px zwingt. Derselbe Kommentar hält fest, dass
die 44-px-Touch-Regel dort bewusst unterschritten wird — mit der Begründung, die
Nachbar-Buttons täten das ohnehin und „eine Leiste mit zwei Höhen sah kaputt aus".
Diese Ausnahme ist genau so lange gültig, wie die Leiste die einzige 30-px-Insel
ist. Sobald alle Header-Controls auf 44 px gehen, kehrt sie sich um.

**3. Es gibt keine Button-Komponente, aber einen Präzedenzfall für Konstanten.**
`pages/admin/BeitragslaufPage.tsx`, `TresorPage.tsx` und `WartungsmodusPage.tsx`
haben je eine modul-lokale `BTN_PRIMARY`-Konstante mit identischem Inhalt. Das ist
die Richtung — nur dreimal kopiert.

## Goals / Non-Goals

**Goals:**

- Eine Fundstelle für die vier Klassen-Strings, importierbar aus Seiten und
  Komponenten.
- Gleiche Höhe für alle Bedienelemente einer Kopfzeile, auf jeder Breite,
  unabhängig davon, ob das Element ein `button`, ein `input` oder ein `select` ist.
- Mechanische Absicherung gegen erneutes Kopieren.

**Non-Goals:**

- **Keine `<Button>`-React-Komponente.** Konstanten, keine Komponente — siehe
  Entscheidung 1.
- **Keine Vereinheitlichung über die Kopfzeile hinaus.** Pills, Tabs,
  Dropdown-Einträge, Tabellen-Buttons und Formular-Buttons behalten ihre Größen.
- **Keine Änderung am Compact-Verhalten.** Welche Elemente ab welcher Breite auf
  Icon-only umschalten oder verschwinden, bleibt exakt wie heute.
- **Keine neuen `brand-*`-Tokens.** `h-11` und `h-[30px]` sind Layout, keine Marke.

## Decisions

### 1. Konstanten-Strings statt `<Button>`-Komponente

Eine `<Button>`-Komponente wäre der Lehrbuchweg, passt hier aber schlecht:

- Die Call-Sites sind heterogen — `button`, `Link`, `a`, `input`, `select`. Eine
  Button-Komponente deckt die Eingabefelder in derselben Zeile nicht ab, und genau
  deren Höhe ist der Kern des Problems.
- Fast jede Fundstelle hängt Layout-Klassen an (`w-full`, `sm:shrink-0`,
  `whitespace-nowrap`, `flex-1`). Eine Komponente bräuchte dafür ein
  `className`-Passthrough plus `tailwind-merge` — neue Dependency auf einem
  1-GB-VPS-Projekt ohne State-Manager und ohne UI-Library.
- Die bestehende Spec (`component-standards`) ist bereits als *Klassen-String*
  formuliert, ebenso `docs/agent/05-frontend.md`. Konstanten passen dazu ohne
  Bruch; der Gate-Test kann weiter rein textuell prüfen.

Verworfen: `<Button>`-Komponente, `class-variance-authority`, Tailwind
`@apply`-Komponentenklassen in `index.css` (letzteres verstößt gegen die
Projektregel „keine eigene CSS-Datei außer `index.css` mit nur `@tailwind`").

### 2. Basis-String + Farbsatz, nicht ein String pro Zustand

```ts
export const HEADER_CTRL = 'inline-flex items-center justify-center gap-1 rounded-md border h-11 sm:h-[30px] px-3 text-xs font-medium transition-colors shrink-0 disabled:opacity-40 disabled:cursor-not-allowed'
export const HEADER_CTRL_ICON = /* wie HEADER_CTRL, px-2 */
export const HEADER_PRIMARY = 'border-brand-yellow bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black'
export const HEADER_NEUTRAL = 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
```

Verwendung: `` className={`${HEADER_CTRL} ${HEADER_PRIMARY}`} ``.

Grund für die Zweiteilung: die Filter-Chips sind Toggles mit zwei Farbzuständen,
und `EventTypeFilter` bezieht den aktiven Zustand aus `getEventColors(type).filter`
(Capability `event-type-colors`) — dort ist die Farbe pro Termintyp verschieden.
Ein monolithischer String pro Zustand müsste diese Farbquelle duplizieren. Mit
Basis + Farbsatz bleibt `getEventColors` unangetastet und trägt weiterhin nur die
Farbe.

Das `border` ohne Farbe steht bewusst im Basis-String: es ist der Grund, warum die
heutigen Buttons ohne Rahmen 28 px statt 30 px hoch sind. Bei fixer Höhe spielt das
für die Höhe keine Rolle mehr, für die optische Kantengleichheit in der Zeile aber
sehr wohl.

### 3. Feste Höhe (`h-11 sm:h-[30px]`) statt Padding

Der eigentliche Beschluss, und der einzige, der die Zusage der Spec trägt. Die
Alternative — Padding so wählen, dass es zufällig passt — scheitert daran, dass
`index.css` Buttons und Eingabefelder verschieden behandelt: dieselbe `py-2.5`-Klasse
ergibt unter 640 px ein 41 px hohes `select` und einen 38 px hohen `button`. Kein
Padding-Wert löst das für beide gleichzeitig, weil die Differenz aus der
Schriftgröße kommt, nicht aus dem Padding.

`h-11` (44 px) ist der Touch-Target-Wert aus `docs/agent/05-frontend.md`.
`h-[30px]` ist die gemessene Ist-Höhe der heutigen Leiste — der Change verschiebt
das Desktop-Bild also nicht, er zieht die Ausreißer darauf.

Nebenwirkung, die man mitnehmen muss: bei fixer Höhe braucht jedes Element
`inline-flex items-center` (bzw. `flex`), sonst sitzt der Text oben. Für die
`select`-Elemente reicht die fixe Höhe allein, weil der Browser den Text darin selbst
zentriert — genau das Argument, das schon im `EventSearchInput`-Kommentar steht.

### 4. Der Compact-Sonderfall `h-7` entfällt

Heute ist die Leiste im Compact-Modus 28 px hoch statt 30 px, weil die Chips dort
nur ein 14-px-Icon enthalten und paddinggetrieben schrumpfen — `EventSearchInput`
bildet das mit `h-7` nach. Sobald alle Chips eine feste Höhe tragen, entsteht dieser
Unterschied nicht mehr, und `h-7` würde ihn künstlich wiederherstellen. Compact
ändert danach nur noch Inhalt (Icon statt Icon+Label) und Breite (`px-2` statt
`px-3`) — was es eigentlich immer sollte, denn der Modus existiert gegen
horizontalen, nicht vertikalen Platzmangel.

Das vereinfacht `EventSearchInput` von zwei Höhen auf eine.

### 5. Die 44-px-Ausnahme wird aufgehoben, nicht umgangen

Der Kommentarblock in `EventSearchInput.tsx` begründet die Unterschreitung mit der
Einheitlichkeit der Leiste. Das Argument bleibt gültig — es zeigt nach diesem Change
nur in die andere Richtung. Der Kommentar wird deshalb ersetzt statt gelöscht: die
Fixhöhen-Begründung (Punkt 3) bleibt stehen, der Absatz zur bewussten
Regelunterschreitung verschwindet.

Verworfen: die Ausnahme beibehalten und nur die gestapelten Kopfzeilen auf 44 px
heben. Das war der erste Entwurf und hätte zwei Header-Familien mit zwei
Mobile-Höhen zementiert — also genau den Zustand, den dieser Change beseitigt.

### 6. Gate als Vitest-Textscan mit Allowlist

Der Test scannt `web/src/pages/` und `web/src/components/` nach den vier Metriken
als Literal und meldet Datei + Zeile. Vorbild ist das Muster aus
`internal/arch/broadcast_test.go`: Ausnahmen stehen mit Begründung in einer
Allowlist, und ein verwaister Eintrag lässt den Test ebenfalls fallen.

Bewusst textuell und nicht über einen AST: die Klassen-Strings stehen teils in
Template-Literalen mit eingebetteten Ternaries, teils über mehrere Zeilen
umgebrochen. Ein AST-Ansatz müsste den JSX-Ausdruck auswerten, um an den
tatsächlichen String zu kommen — deutlich mehr Maschinerie für dieselbe Aussage.
Der Preis ist, dass ein anders umbrochener Kopiervorgang durchrutschen kann; das
Gate ist eine Bremse gegen Copy-Paste, keine Beweisführung.

## Risks / Trade-offs

**Kopfzeilen brechen um, weil die Buttons schmaler werden** (`px-3`/`text-xs` statt
`px-4`/`text-sm`) → Betrifft die Seiten, die heute die Formular-Größe verwenden. In
der Breite werden sie kleiner, nicht größer, ein Umbruch ist also unwahrscheinlich —
zu prüfen sind trotzdem `AdminVenuesPage` (Split-Button mit Caret) und
`AdminKaderPage` (drei Buttons nebeneinander mit `whitespace-nowrap`). Nach den
Umbau-Tasks je ein Blick in Chrome DevTools auf 375 px und 1280 px.

**Die einzeilige Filterleiste wird auf Mobile ~14 px höher** → Nur vertikal;
horizontal ändert sich nichts, weil Compact unangetastet bleibt. Die Leiste ist auf
`TerminePage`/`DutyPage` `sticky top-0`, kostet also dauerhaft etwas Sichtfläche.
Bewusst akzeptiert: der Gegenwert sind Touch-Targets, die die eigene Projektregel
erfüllen.

**Fixe Höhe verträgt keinen Zeilenumbruch im Label** → Ein zu langes Button-Label
würde bei `h-[30px]` überlaufen statt umzubrechen. Heute tritt das nicht auf (alle
Labels sind ein bis zwei Wörter), aber es ist eine neue Einschränkung. `shrink-0`
im Basis-String hält den Button in der Flex-Zeile stabil; überlange Labels sind
künftig eine Frage der Beschriftung, nicht des Layouts.

**Textueller Gate-Test meldet Fehlalarme** → Wenn eine Metrik künftig legitim
anders gebraucht wird, wächst die Allowlist. Mitigation: die Allowlist verlangt eine
Begründung pro Eintrag und schlägt bei verwaisten Einträgen fehl, wird also nicht
still zur Müllhalde.

**Rein präsentational, aber breit gestreut** → 13 Seiten, 2 geteilte Komponenten.
Kein Backend, keine DB, keine Route, kein Berechtigungs-Gate ist berührt; die
Sichtbarkeitsbedingungen an den Buttons (`canUpload`, `isAdmin`, Capabilities)
bleiben unverändert. Rollback ist ein `git revert` ohne Migration.

## Migration Plan

Kein Datenbank- oder Deploy-Schritt. Die Umsetzung geht in der Reihenfolge
Konstante → gestapelte Kopfzeilen → einzeilige Filterleiste → Aufräumen → Gate;
jeder Schritt ist für sich lauffähig und commit-fähig (ein Commit pro Task, siehe
`docs/agent/09-openspec.md`). Der Gate-Test kommt zuletzt, weil er vorher
zwangsläufig rot wäre.
