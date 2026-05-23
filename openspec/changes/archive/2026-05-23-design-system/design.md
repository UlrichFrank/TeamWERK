## Context

Das Frontend von TeamWERK ist organisch über ~50 TSX-Dateien gewachsen ohne zentrales Design-Token-System. Die Analyse im Proposal zeigt: gleiche Komponenten haben 5–15 leicht unterschiedliche Ausprägungen, Raw-Tailwind-Farben (`bg-gray-50`, `text-gray-700`, `text-red-600`) stehen neben Brand-Tokens, und es gibt keine verbindliche Definition für Buttons, Inputs, Karten, Alerts oder Tabellen.

Lucide React ist im Dashboard bereits eingeführt und funktioniert — das Icon-System läuft also, muss aber in alle anderen Dateien nachgezogen werden.

Das Dashboard nutzt aktuell bereits eine solide Accordion-Struktur und Lucide Icons, weicht aber bei Statusbadges (raw `bg-green-100`, `bg-blue-100`, `bg-yellow-100`) und Team-Stats-Farben (`text-red-500`, `text-yellow-500`) von den geplanten Brand-Tokens ab.

---

## Goals / Non-Goals

**Goals:**
- Neue semantische Tokens in `tailwind.config.js` definieren (`brand-surface-card`, `brand-text`, `brand-text-muted`, `brand-text-subtle`, `brand-border`, `brand-border-subtle`, `brand-danger`, `brand-danger-light`, `brand-info`)
- Verbindliche Klassen-Strings für alle Komponentenarten festlegen (Button, Input, Card, Modal, Alert, Table)
- Alle ~50 TSX-Dateien auf die neuen Standards vereinheitlichen
- Dashboard auf Brand-Tokens migrieren (Statusbadges, TeamStats-Farben, raw-Farben ersetzen)
- Unicode-Zeichen und Emojis durch Lucide Icons ersetzen

**Non-Goals:**
- Neue Features oder Funktionalitätsänderungen
- Backend-Code, Migrationen, API-Änderungen
- Neue Seiten oder Routen
- Responsiveness-Grundstruktur ändern (Mobile-Breakpoint `sm:` bleibt)

---

## Decisions

### 1 — Token-Layer: Semantische Namensgebung

**Entscheidung:** Neue Tokens werden als semantische Alias-Namen über den bestehenden `brand-*`-Namespace gelegt — nicht als separate Design-Token-Datei oder CSS-Custom-Properties.

**Warum:** Tailwind-Config ist bereits vorhanden und funktioniert. CSS-Custom-Properties würden eine Vite-Plugin-Änderung erfordern und sind für diesen Scope überdimensioniert. Semantische Names in `tailwind.config.js` sind direkt einsetzbar ohne Build-Änderung.

**Neue Tokens:**

| Token | Wert | Ersetzt |
|---|---|---|
| `brand-surface-card` | `#F9FAFB` (= gray-50) | `bg-gray-50` (86×) |
| `brand-text` | `#111827` (= gray-900) | `text-gray-900`, `text-black` |
| `brand-text-muted` | `#6B7280` (= gray-500) | `text-gray-500` (91×), `text-black/50` |
| `brand-text-subtle` | `#9CA3AF` (= gray-400) | `text-gray-400` (70×) |
| `brand-border` | `#D1D5DB` (= gray-300) | `border-gray-300` |
| `brand-border-subtle` | `#E5E7EB` (= gray-200) | `border-gray-200`, Divider-Linien |
| `brand-danger` | `#C0253A` (Karmesin) | `text-red-600`, `bg-red-*` (destruktiv) |
| `brand-danger-light` | `#FCEEF1` | `bg-red-50`, `bg-red-100` |
| `brand-info` | `#3B82F6` (= blue-500) | `bg-blue-50/border-blue-200` in Alerts |

`brand-table-select` (`#E5E7EB`) bleibt bestehen als dedizierter Token für Row-Hover in Tabellen.

**Alternative verworfen:** Separate `design-tokens.ts`-Datei mit JS-Konstanten — zu viel Indirektion für ein Tailwind-Projekt.

---

### 2 — Einheitliche Klassen-Strings (Copy-paste-Standards)

**Entscheidung:** Pro Komponentenart wird exakt ein verbindlicher Klassen-String definiert. Abweichungen werden als Bug behandelt.

**Button — Primary:**
```
bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium
hover:bg-brand-black hover:text-brand-yellow transition-colors
disabled:opacity-40 disabled:cursor-not-allowed
```

**Button — Small (in Tabellen, eingebettet):**
```
bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium
hover:bg-brand-black hover:text-brand-yellow transition-colors
disabled:opacity-40 disabled:cursor-not-allowed
```

**Button — Danger:**
```
bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium
hover:bg-brand-danger/90 transition-colors
disabled:opacity-40 disabled:cursor-not-allowed
```

**Input — Standard:**
```
w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text
placeholder:text-brand-text-subtle
focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow
```

**Input — Klein:**
```
w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text
focus:outline-none focus:ring-1 focus:ring-brand-yellow
```

**Card — Standard:**
```
bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6
```

**Card — Kompakt:**
```
bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4
```

**Card — Tabellen-Container:**
```
bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden
```

**Modal — Klein (≤ max-w-sm):**
```
bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6
```

**Modal — Groß:**
```
bg-white rounded-xl shadow-2xl border-t-4 border-brand-yellow
Header: px-6 py-4 border-b border-brand-border-subtle
Body: flex-1 overflow-y-auto p-6
Footer: px-6 py-4 border-t border-brand-border-subtle flex justify-end gap-3
```

**Alert — Info:**
```
p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text
```

**Alert — Fehler:**
```
p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger
```

**Alert — Erfolg:**
```
p-3 bg-brand-success-light border border-brand-success/40 rounded-lg text-sm text-brand-text
```

**Alert — Warnung:**
```
p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-sm text-brand-text
```

**Tabellen-Header:**
```
th: bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left
```

**Tabellen-Row:**
```
tr: hover:bg-brand-table-select transition-colors divide-y divide-brand-border-subtle
td: px-4 py-3 text-sm text-brand-text
```

**Warum Copy-paste-Standard statt Komponente:** Komponenten würden eine weitere Abstraktionsschicht einführen, die bei einem Django-ähnlich gewachsenen Codebase mit direkten Tailwind-Klassen schnell zur Last wird. Der Standard wird als verbindliche Inline-Konvention gehalten — Tailwind-Plugins oder Komponentenextraktion sind ein separater Change.

---

### 3 — Lucide React: vollständige Migration

**Entscheidung:** Lucide React ist bereits installiert und im Dashboard in Verwendung. Alle verbleibenden Unicode-Zeichen (`☰`, `✕`, `⋮`, `▸`, `▾`, `✓`, `✗`, `⚠`, `«`, `»`) und Emojis (`🗑`, `📋`, `⚽`, `✈`) werden durch Lucide Icons ersetzt.

**Icon-Größen-Standard:**
- `w-4 h-4` — Inline in Text, Tabellen-Aktionen (xs-Button)
- `w-5 h-5` — Standard in Buttons, Nav-Chevrons
- `w-6 h-6` — Standalone, leere Zustände

**Farbe:** `currentColor` (erbt Textfarbe, kein separates Styling)

**Mapping:**

| Alt | Lucide |
|---|---|
| `☰` | `Menu` |
| `✕` / `✗` | `X` |
| `⋮` | `MoreVertical` |
| `▸` | `ChevronRight` |
| `▾` | `ChevronDown` |
| `✓` | `Check` |
| `⚠` | `AlertTriangle` |
| `🗑` | `Trash2` |
| `«` / `»` | `ChevronsLeft` / `ChevronsRight` |
| `📋` | `Calendar` |
| `⚽` (Heimspiel) | `Home` |
| `🚌` (Auswärtsspiel) | `MapPin` |

**Accessibility:** Alle Icon-only-Buttons erhalten `aria-label`.

---

### 4 — Dashboard: Brand-Token-Migration

**Entscheidung:** Das Dashboard bekommt keine neue Layout- oder Funktionsstruktur, sondern wird nur auf Brand-Tokens migriert.

**Konkrete Stellen:**

| Aktuell | Neu |
|---|---|
| `bg-green-100 text-green-800` (Status „Erfüllt") | `bg-brand-success-light text-brand-text` |
| `bg-yellow-100 text-yellow-800` (Status „Ablöse") | `bg-brand-warning-light text-brand-text` |
| `bg-blue-100 text-blue-800` (Status „Zugesagt") | `bg-brand-info/10 text-brand-text` |
| `text-red-500` (Verletzt-Zahl in TeamStats) | `text-brand-danger` |
| `text-yellow-500` (Pausiert-Zahl in TeamStats) | `text-brand-warning` |
| `text-black/50` (Muted-Text) | `text-brand-text-muted` |
| `🏠` / `🚌` (Spielplan-Emojis) | `Home` / `MapPin` aus Lucide |

---

### 5 — Button-Position auf Listen-Seiten

**Entscheidung:** Verbindliche Konvention für den Primär-Button:
- **Listen-Seiten** (Tabellen): Primär-Button oben rechts neben `<h1>` → „Neu anlegen"
- **Formular-Seiten**: Primär-Button unten im Formular → „Speichern"
- **Karten mit Inline-Form**: Button unten in der Karte

Seiten, die aktuell abweichen (AdminClubPage, AdminTeamsPage, AdminSeasonsPage, AdminDutyTypesPage) werden korrigiert.

---

## Risks / Trade-offs

**Viele Dateien, kleiner Änderungsradius** → Jede Datei erhält nur Klassen-Ersetzungen, keine Logik-Änderungen. Risiko ist niedrig, Merge-Konflikte bei paralleler Feature-Arbeit möglich. Mitigation: Change in einem Zug ausrollen, nicht über mehrere Branches verteilen.

**`brand-danger` vs. `brand-error`** → Zwei Tokens für rote Farben. `brand-error` bleibt für Kompatibilität, `brand-danger` ist die neue primäre Danger-Farbe (Karmesin `#C0253A`). Mitigation: `brand-error` nach dem Rollout deprecaten und in einem Follow-up-Change entfernen.

**Tailwind-Purge bei neuen Tokens** → Neue Token-Namen müssen in Klassen vorkommen, sonst werden sie vom Purge entfernt. Mitigation: Tokens in `tailwind.config.js` als `safelist` oder direkt in Dateien verwenden — kein dynamisches Klassen-Konstruieren.

**Dashboard-Emojis** → `🏠` und `🚌` sind im DashboardPage-JSX. Lucide `Home` und `MapPin` sind visuelle Änderungen, die getestet werden müssen (insbesondere ob `MapPin` für Auswärtsspiel verständlich ist).

---

## Migration Plan

**Reihenfolge (bottom-up):**

1. `tailwind.config.js` — neue Tokens eintragen
2. Globale Komponenten (`AppShell`, `ActionMenu`, `MobileCard`, `EditModal`, `Pagination`, `Accordion`) — Unicode/Emojis → Lucide, Klassen vereinheitlichen
3. Shared-Komponenten (`BrandCheckbox`, `ConfirmModal`) — Klassen vereinheitlichen
4. Pages alphabetisch — jede Datei einzeln migrieren
5. Dashboard zuletzt — Token-Migration nach fertigem Token-Layer

**Rollback:** Rein visuelle Änderungen. Bei Problemen reicht `git revert` auf einzelne Dateien. Keine DB-Migrationen, keine API-Änderungen.

**Kein Feature-Flag nötig:** Das ist ein Refactoring-Change ohne Verhaltensänderung.

---

## Open Questions

Keine offenen Fragen — alle Entscheidungen sind im Proposal abgeschlossen.
