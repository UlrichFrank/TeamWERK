# Proposal: Design System — Vereinheitlichung UI

**Status:** Proposed
**Erstellt:** 2026-05-23
**Owner:** —

## Entscheidungen (bereits getroffen)

| Thema | Entscheidung |
|---|---|
| Danger-Farbe | `#C0253A` Karmesin — HSL 350° 68% 45% |
| Icon-System | Lucide React (`pnpm add lucide-react`) |

---

## Problem

Das Frontend ist über ~50 Dateien gewachsen, ohne zentrales Design-Token-System. Daraus entstehen Inkonsistenzen, die sowohl die visuelle Qualität als auch die Wartbarkeit beeinträchtigen:

- Gleiche Komponenten (Buttons, Inputs, Alerts) haben 5–15 leicht unterschiedliche Ausprägungen
- Raw-Tailwind-Farben (`bg-gray-50`, `text-gray-700`, `red-600`) werden neben Brand-Tokens verwendet
- Fehlende semantische Tokens für häufige Fälle (Kartenhintergrund, Fließtext, Hinweisboxen)
- Button-Position auf Seiten inkonsistent (oben rechts vs. unten)

---

## Ziel

1. **Token-Layer erweitern:** Alle verwendeten Farben in `brand.*` semantisch benennen
2. **Komponentenstandards definieren:** Je 1 verbindliche Klassen-Kombination pro Variante
3. **Bestehenden Code vereinheitlichen:** Alle Abweichungen in Seiten und Komponenten korrigieren

---

## Bestandsaufnahme (Analyse)

### A — Farbverwendung: Ist-Zustand

**Brand-Tokens (definiert in `tailwind.config.js`):**

| Token | Hex | bg× | text× | border× | Befund |
|---|---|---|---|---|---|
| `brand-yellow` | `#FAE806` | 84 | 66 | 70 | ✓ Hauptfarbe, gut genutzt |
| `brand-black` | `#000000` | 39 | 81 | 13 | △ Konkurrent: `text-black` (kein brand) |
| `brand-table-select` | `#E5E7EB` | 16 | — | — | ✗ Tabellen-Hover / Auswahlmarker, kaum als Token genutzt |
| `brand-white` | `#FFFFFF` | 15 | — | — | ✓ ausreichend |
| `brand-blue` | `#3E4A98` | 2 | 10 | — | △ raw `blue-*` parallel genutzt |
| `brand-green` | `#6EB42E` | 2 | — | — | △ kaum genutzt |
| `brand-error` | `#EF4444` | 6 | 13 | — | △ raw `red-*` parallel genutzt |
| `brand-success` | `#10B981` | 2 | 3 | — | △ kaum genutzt |
| `brand-warning` | `#F59E0B` | 2 | 1 | — | △ kaum genutzt |
| `*-light`-Varianten | — | ≤1 | ≤1 | — | △ kaum genutzt |

**Nicht-brand Raw-Tailwind-Farben (häufigste Vorkommen):**

| Klasse | Vorkommen | Bedeutung |
|---|---|---|
| `bg-gray-50` | 86 | Karten-/Formular-Hintergrund |
| `text-gray-700` | 103 | Fließtext dunkel |
| `text-gray-500` | 91 | Sekundärtext |
| `text-gray-400` | 70 | deaktiviert / schwach |
| `text-gray-600` | 46 | Labels |
| `text-red-600` | 35 | Fehlermeldungen, Löschen |
| `text-green-600` | 14 | Erfolgszustand |
| `bg-gray-200` | 12 | Inaktive Elemente |
| `bg-red-50/100` | 13 | Alert-Hintergrund Fehler |
| `bg-blue-50` | 9 | Alert-Hintergrund Info |

**Kritische Verwechslung — `bg-gray-50` vs `bg-brand-sidebar`:**

```
bg-gray-50          = #F9FAFB   ← fast weiß (wird für Karten-/Formular-Hintergrund genutzt)
bg-brand-table-select = #E5E7EB   ← = gray-200 (Tabellen-Hover / Auswahlmarker)
```

Diese beiden werden nicht klar getrennt. `bg-gray-50` taucht 86× auf, hat aber keinen Brand-Token.

---

### B — Buttons: Ist-Zustand

**Primär-Button (gelb→schwarz) — 15 verschiedene className-Strings:**

```
Gemeinsamkeit:    bg-brand-yellow rounded-md text-sm font-medium
                  hover:bg-brand-black hover:text-brand-yellow

Abweichungen:
  Textfarbe:      text-black (36×) vs text-brand-black (48×)      ← selbe Farbe, inkonsistent
  Padding Y:      py-2 vs py-2.5 vs py-1.5 vs py-2.5 sm:py-2
  Disabled:       disabled:opacity-40 vs disabled:opacity-50
  transition:     transition-colors manchmal fehlend
  font:           font-medium vs font-semibold (full-width Forms)
  Rahmen:         manchmal border border-brand-yellow (selten)
```

**Klein-Button (xs-Variante):**
```
text-xs bg-brand-yellow text-brand-black px-3 py-1 rounded font-medium
hover:bg-brand-black hover:text-brand-yellow transition-colors
```
→ `rounded` statt `rounded-md`, `text-xs` statt `text-sm`

**Sekundär-Button — 2 verschiedene Konzepte:**
```
Konzept A (Outline schwarz):
  border border-black text-black rounded-md px-3 py-1.5

Konzept B (Outline grau):
  border border-gray-300 rounded-md px-4 py-2 text-sm hover:border-gray-500

→ unklar wann welches
```

**Destruktiv-Button — 4 verschiedene Konzepte:**
```
a) Nur Text:           text-red-600 (kein bg, kein border)
b) Outline rot:        border border-red-300 text-red-600 px-3 py-1 rounded
c) Gefüllt rot:        bg-red-100 text-red-700 px-2 py-1 rounded
d) Gefüllt schwarz:    bg-black text-white rounded-md px-3 py-1.5  ← "Ablehnen"-Button
```

**Button-Position auf Seiten:**

| Seite | Position des Primär-Buttons |
|---|---|
| AdminClubPage | unten im Formular |
| AdminTeamsPage | unten im Formular |
| AdminSeasonsPage | unten im Formular |
| AdminDutyTypesPage | unten im Formular |
| AdminDutyTemplatesPage | **oben rechts** neben h1 |
| DutyAccountsPage | **oben rechts** neben h1 |
| MembersPage | **oben rechts** neben h1 |
| AdminUsersPage | **oben rechts** neben h1 |
| MembershipRequestsPage | inline pro Karte |

---

### C — Inputs: Ist-Zustand

**3 konkurrierende Standards:**

```
Variante A (33×, Standard):
  w-full border border-gray-300 rounded-md px-3 py-2 text-sm
  → kein Focus-Ring!

Variante B (20×, mit Focus):
  w-full border border-gray-300 rounded-md px-3 py-2 text-sm
  focus:outline-none focus:ring-2 focus:ring-brand-yellow

Variante C (4×, blauer Focus):
  w-full border rounded-md px-3 py-2 text-sm
  focus:outline-none focus:ring-2 focus:ring-brand-blue

Klein (7×):
  w-full border rounded-md px-2 py-1.5 text-sm
  → kein border-gray-300!
```

---

### D — Karten / Panels: Ist-Zustand

**Dominantes Muster (gut):**
```
bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow
```

**Padding-Varianten:** `p-4`, `p-5`, `p-6`, `p-8` — alle vorkommen, keine Regel

**Modals — kein border-t-4:**
```
bg-white rounded-xl shadow-xl p-6        (klein)
bg-white rounded-xl shadow-2xl           (groß, kein padding auf root)
```

**Shadow-Varianten:**
```
shadow     → Standard-Karten (46×)
shadow-xl  → Modals (7×)
shadow-2xl → Haupt-Modals (10×)
shadow-lg  → Dropdowns (6×)
```

---

### E — Alerts / Hinweisboxen: Ist-Zustand

Komplett ungeklärt — 4 semantische Typen, aber inkonsistent umgesetzt:

```
Info (Beispiel A):  p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-gray-700
Info (Beispiel B):  p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-700
                    → gleicher Typ, verschiedene Textfarbe

Fehler:             p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700

Erfolg:             p-3 bg-green-50 border border-green-200 rounded-lg text-sm

Warnung A:          p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-xs text-yellow-700
Warnung B:          p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-700
Warnung C (brand):  p-3 bg-brand-warning-light border border-brand-warning rounded-lg text-sm
                    → 3 Varianten für denselben Typ!
```

---

### F — Radien: Ist-Zustand

| Klasse | Vorkommen | Aktuell genutzt für |
|---|---|---|
| `rounded-md` | 174 | Buttons, Inputs, Tags |
| `rounded-xl` | 57 | Karten, Panels, Modals |
| `rounded` | 57 | Kleine Badges, Aktionen |
| `rounded-lg` | 35 | Alerts, Dropdowns |
| `rounded-full` | 19 | Pill-Badges, Avatare |

---

### G — Tabellenstruktur: Ist-Zustand

```
Header:   bg-gray-50 text-gray-500 text-xs uppercase   (4×)
          bg-gray-50 text-gray-500 uppercase text-xs   (1×, andere Reihenfolge)

Row Hover: hover:bg-brand-gray (11×) vs hover:bg-gray-50 (24×)
           → brand-gray = #E5E7EB, gray-50 = #F9FAFB — visuell unterschiedlich!
```

---

### H — Icons / Symbole / Grafiken: Ist-Zustand

**Kein Icon-System vorhanden.** Es gibt keine eingebundene Icon-Library. Stattdessen werden drei verschiedene Ansätze gemischt:

**1. Unicode-Zeichen als funktionale Icons**

| Zeichen | Bedeutung | Verwendet in | Vorkommen |
|---|---|---|---|
| `☰` | Hamburger-Menü öffnen | AppShell (Mobile) | 1× |
| `✕` | Schließen (Modal/Sidebar) | AppShell, DutyTemplateDetail | 2× |
| `⋮` | Aktionsmenü öffnen | ActionMenu | 1× |
| `▸` | Nav-Gruppe eingeklappt | AppShell | 1× |
| `▾` | Nav-Gruppe ausgeklappt | AppShell | 1× |
| `✓` | Annehmen / Bestätigen | MemberKontaktTab, Stammdaten, Datenschutz, AdminClub | 8× |
| `✗` | Ablehnen | MemberKontaktTab, Stammdaten, Datenschutz, MembersPage | 7× |
| `⚠` | Warnung | SpieltagDetailPage, AdminDutyTemplatesPage | 3× |
| `«` `»` | Pagination Anfang/Ende | Pagination | je 1× |

**2. Emojis als dekorative / funktionale Icons**

| Emoji | Bedeutung | Verwendet in | Problem |
|---|---|---|---|
| `🗑` | Löschen | SpieltagDetailPage, DutyPage | 2× — kein konsistentes Styling |
| `📋` | Sonstiges Event | SpielplanPage | 1× — dekorativ |
| `⚽` | Heimspiel | SpielplanPage | 1× — dekorativ |
| `✈` | Auswärtsspiel | SpielplanPage | 1× — dekorativ |

**3. Inline SVG (einmalig)**

| Komponente | SVG | Beschreibung |
|---|---|---|
| `BrandCheckbox` | 3 horizontale Linien (abgestuft) | Filter-/Sortier-Symbol, custom gezeichnet |

**Befund:** Es gibt keine Icon für Navigation (nur Text-Labels), keine konsistente Größe für Symbole, kein `aria-label` auf Icon-only-Buttons, und keinen gemeinsamen Stil zwischen Unicode-Zeichen und Emojis.

---

## Vorschlag: Design System

> **Hinweis:** Dieser Abschnitt ist Vorschlag — bitte überarbeite und ergänze nach deinen Vorstellungen.

### Token-Erweiterungen (tailwind.config.js)

**Neue semantische Tokens vorgeschlagen:**

```js
// Oberflächen
'brand-surface-card': '#F9FAFB'  // = gray-50 → Karten-/Formular-Hintergrund

// Fließtext
'brand-text':        '#111827'  // = gray-900 → Primärtext
'brand-text-muted':  '#6B7280'  // = gray-500 → Sekundärtext
'brand-text-subtle': '#9CA3AF'  // = gray-400 → deaktiviert, Placeholder

// Rahmen
'brand-border':      '#D1D5DB'  // = gray-300 → Standard-Input-Rahmen
'brand-border-subtle':'#E5E7EB' // = gray-200 → Trennlinien

// Danger — ENTSCHIEDEN: Karmesin #C0253A (ersetzt brand-error #EF4444)
'brand-danger':        '#C0253A'  // Karmesin, HSL 350° 68% 45%
'brand-danger-light':  '#FCEEF1'  // Hintergrund für Danger-Alerts
'brand-info':          '#3B82F6'  // = blue-500 → neu für Alert-Typ Info
```

**Hinweis:** `brand-table-select` (`#E5E7EB`) ist als dedizierter Token für Tabellenmarkierungen/Row-Hover geplant. Sekundäre Flächen wie Tabellenheader und inaktive Input-Hintergründe werden stattdessen aus dem bestehenden Token-System abgeleitet oder behalten einen weißen/`brand-surface-card`-ähnlichen Hintergrund.
---

### Button-Hierarchie (Vorschlag)

**1 Button-Stil, 1 kompakte Variante, plus Danger:**

```
BUTTON (alle Aktionen)
  bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium
  hover:bg-brand-black hover:text-brand-yellow transition-colors
  disabled:bg-brand-yellow/50 disabled:text-brand-black/50 disabled:cursor-not-allowed
  Mobile (Touch-Target): py-2.5 sm:py-2

SMALL BUTTON (eingebettet, z.B. in Tabellen)
  bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium
  hover:bg-brand-black hover:text-brand-yellow transition-colors
  disabled:bg-brand-yellow/50 disabled:text-brand-black/50 disabled:cursor-not-allowed

DANGER BUTTON (Löschen, Ablehnen)
  bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium
  hover:bg-brand-danger/90 transition-colors
  disabled:bg-brand-danger/30 disabled:text-white/60 disabled:cursor-not-allowed
```

Es gibt weiterhin nur einen visuellen Button-„Familienstil“. Danger-Buttons sind die einzige semantisch abweichende Variante neben dem normalen Look.
Deaktivierte Buttons behalten das Grundlayout bei, wirken aber mit reduzierter Farbintensität und `cursor-not-allowed` deutlich inaktiv.

Anlehnung an das interne Login-Layout: Die Textflächen sollen dabei eine weiße bzw. `brand-surface-card`-Hintergrundfarbe haben, aber nicht blau eingefärbt werden.

**Button-Position — Vorschlag:**
```
Listen-Seiten (Tabellen):  Primär-Button OBEN RECHTS neben h1 → "Neu anlegen"
Formular-Seiten:           Primär-Button UNTEN im Formular → "Speichern"
Karten mit Inline-Form:    Button UNTEN in der Karte
```

---

### Input-Standard (Vorschlag)

**1 Standard, 1 Klein-Variante:**

```
Standard:
  w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text
  placeholder:text-brand-text-subtle
  focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow

Klein (in Tabellen, Filterzeilen):
  w-full border border-brand-border rounded px-2 py-1.5 text-sm
  focus:outline-none focus:ring-1 focus:ring-brand-yellow

FRAGE: focus:ring-brand-yellow überall, oder nur für Standard? Blau weg?
```

---

### Karten-Standard (Vorschlag)

```
Standard-Card (Formular, Listen):
  bg-brand-surface-card rounded-xl shadow-card border-t-4 border-brand-yellow
  Padding: p-6 (Standard), p-4 (kompakt)

Tabellen-Card:
  bg-brand-surface-card rounded-xl shadow-card border-t-4 border-brand-yellow overflow-hidden
  (kein padding auf root, Padding in thead/tbody)

Modal (klein ≤ max-w-sm):
  bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6

Modal (groß > max-w-sm):
  bg-white rounded-xl shadow-2xl border-t-4 border-brand-yellow
  Header: px-6 py-4 border-b border-brand-border
  Body: flex-1 overflow-y-auto p-6
  Footer: px-6 py-4 border-t border-brand-border flex justify-end gap-3

FRAGE: Sollen Modals auch border-t-4 haben? Aktuell haben sie es nicht.
```

Erläuterung: `shadow-card` ist ein gut definierter Drop-Shadow nach rechts und unten, z.B. `shadow-[0_10px_20px_rgba(0,0,0,0.08)]` mit feiner Unschärfe und leichter Transparenz.
---

### Alert-Standard (Vorschlag)

**4 Typen, je 1 Definition:**

```
INFO:
  p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text

WARNUNG:
  p-3 bg-brand-warning-light border border-brand-warning rounded-lg text-sm text-brand-warning

FEHLER:
  p-3 bg-brand-error-light border border-brand-error rounded-lg text-sm text-brand-error

ERFOLG:
  p-3 bg-brand-success-light border border-brand-success rounded-lg text-sm text-brand-success
```

Alle Alert-Farben laufen über `brand-*` Tokens, damit das System zentral steuerbar bleibt.
---

### Tabellen-Standard (Vorschlag)

```
Container:    bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden

Header-Row:   bg-brand-surface-card/80 text-brand-text-muted text-xs uppercase
              th: px-4 py-3 text-left (oder text-right für Zahlen)

Data-Row:     hover:bg-brand-table-select transition-colors
              divide-y divide-brand-border-subtle
              td: px-4 py-3

FRAGE: Row-Hover: brand-table-select (#E5E7EB, deutlich) oder eine andere, flachere Marke für Tabellen-Hover?```

---

### Icon-System (Vorschlag)

**Entscheidung:** Wir führen Lucide React ein. Das Icon-System basiert auf konsistenten SVG-Icons, `currentColor` und klaren Größen.

```
Option A — Lucide React (empfohlen)
  Vorteile:  ~1000 Icons, konsistente 24px-SVG-Sprache, Tailwind-freundlich,
             tree-shakeable (nur genutzte Icons im Bundle), aktiv gepflegt
  Nachteil:  Neue Dependency (~150 KB gzip: minimal)
  Install:   pnpm add lucide-react
  Nutzung:   import { Trash2, X, Menu } from 'lucide-react'
             <Trash2 className="w-4 h-4" />
```

**Semantisches Icon-Set:**

| Funktion | Unicode jetzt | Lucide-Vorschlag | Verwendung |
|---|---|---|---|
| Menü öffnen (Mobile) | `☰` | `Menu` | AppShell Header |
| Schließen / Abbrechen | `✕` | `X` | Modals, Sidebar |
| Aktionsmenü | `⋮` | `MoreVertical` | ActionMenu |
| Nav eingeklappt | `▸` | `ChevronRight` | AppShell Nav |
| Nav ausgeklappt | `▾` | `ChevronDown` | AppShell Nav |
| Bestätigen / Annehmen | `✓` | `Check` | Buttons, Status |
| Ablehnen | `✗` | `X` | Buttons |
| Warnung | `⚠` | `AlertTriangle` | Alert-Boxen |
| Löschen | `🗑` | `Trash2` | Tabellen-Aktionen |
| Heimspiel | `⚽` | `Home` oder eigenes SVG | Spielplan |
| Auswärtsspiel | `✈` | `MapPin` oder eigenes SVG | Spielplan |
| Sonstiges Event | `📋` | `Calendar` | Spielplan |
| Pagination zurück | `«` | `ChevronsLeft` | Pagination |
| Pagination vor | `»` | `ChevronsRight` | Pagination |

**Größen-Standard:**

```
w-4 h-4  (16px) — Inline in Text, Tabellen-Aktionen (xs-Button)
w-5 h-5  (20px) — Standard in Buttons, Nav-Chevrons
w-6 h-6  (24px) — Standalone-Icons, leere Zustände
```

**Farbe:** `currentColor` (erbt Textfarbe des Elternelements — kein separates Styling nötig)

**Accessibility:** Icon-only-Buttons brauchen zwingend `aria-label`:
```tsx
<button aria-label="Eintrag löschen"><Trash2 className="w-4 h-4" /></button>
```

**Entscheidung:** Wir setzen auf Lucide React als Icon-Library. Es gibt keine zusätzlichen Icons in den Sidebar-Navigationseinträgen.

---

### Beschlossene Richtlinien

1. **Danger-Button:** Gefüllte Danger-Variante mit `brand-danger` (Karmesin). Das gilt für „Ablehnen“, Löschen und andere destruktive Aktionen.
2. **Focus-Ring:** Überall Gelb. Die UI orientiert sich am Login-Layout von https://internal.team-stuttgart.org/login, aber die Textflächen bleiben nicht blau sondern sind weiß.
3. **Modal border-t-4:** Ja. Modals erhalten den gleichen oberen Farbstreifen wie Karten.
4. **Alert-Farben:** Vollständig über `brand-*` Tokens.
5. **`brand-table-select`:** Passt als Name für die Tabellen-Markierungsfarbe `#E5E7EB`.
6. **xs-Button Radius:** `rounded-md` für Konsistenz mit dem Standard-Brand-Button.
7. **Button-Typ "Ablehnen":** Wird als Danger-Button eingesetzt, nicht als eigenes neutral-spezifisches Konzept.
8. **Icon-System:** Lucide React wird eingeführt.
9. **Nav-Icons:** Nein, die Sidebar bleibt textbasiert.

---

## Scope

Nur **Frontend** (`web/src/`). Kein Backend-Code, keine Migrationen, keine neuen API-Endpunkte.

**Betroffene Dateien:** ~50 TSX-Dateien (alle Pages, Komponenten)
**Kein Breaking Change:** Nur visuelle Vereinheitlichung, keine Funktionsänderung
