# Frontend-Konventionen

**Auth:** `const { user, login, logout, loading } = useAuth()` — `user` hat `email`, `role` (aus JWT).
**API:** `import { api } from '../lib/api'` → `api.get('/members')` (Bearer + Auto-Refresh bei 401).
**Neue Seite:** Datei in `web/src/pages/`, Route in `App.tsx` unter dem `AppShell`-Outlet, ggf. Nav-Eintrag in `AppShell.tsx` (`roles`-Array).

## Styling (Tailwind v3)

Keine eigene CSS-Datei außer `index.css` (nur `@tailwind`). Schrift: Hanken Grotesk.
Marke: Schwarz `#181310`, Gelb `#FDE400`, Weiß `#FFFFFF`; sekundär Blau `#3E4A98`, Grün `#6EB42E`.
**Keine raw Tailwind-Farben** — immer `brand-*`-Tokens (`tailwind.config.js`):

| Token | Wert | Ersetzt |
|---|---|---|
| `brand-surface-card` | `#F9FAFB` | `bg-gray-50` |
| `brand-text` | `#111827` | `text-gray-900`, `text-black` |
| `brand-text-muted` | `#6B7280` | `text-gray-500` |
| `brand-text-subtle` | `#9CA3AF` | `text-gray-400`, Placeholder |
| `brand-border` | `#D1D5DB` | `border-gray-300` |
| `brand-border-subtle` | `#E5E7EB` | `border-gray-200`, Divider |
| `brand-danger` | `#C0253A` | `text-red-600`, destruktiv |
| `brand-danger-light` | `#FCEEF1` | `bg-red-50/100` in Alerts |
| `brand-info` | `#3B82F6` | Info-Alert |
| `brand-table-select` | `#E5E7EB` | Row-Hover |

**Verbindliche Klassen-Strings** — Fundstelle ist `web/src/lib/buttonStyles.ts`, nicht diese Tabelle: die Strings werden importiert, nicht abgetippt (`buttonStyles.gate.test.ts` prüft das).

**Zwei Button-Rollen, nicht austauschbar:** *Header-Control* steht in der Zeile der `<h1>` (Aktion, Filter-Chip, Suchfeld, Auswahlfeld) und ist 30 px hoch (Mobile 44 px); *Primary* ist die Bestätigungs-Aktion eines Formulars oder Modals und bleibt größer (`text-sm`). Header-Controls setzen ihre Höhe **fix** (`h-11 sm:h-[30px]`), nie über `py-*` — `index.css` zwingt unter 640 px nur `input`/`select`/`textarea` auf 16 px, nicht `button`, sodass dieselbe Padding-Klasse dort ein 41-px-Feld neben einem 38-px-Button ergibt.

- **Button Header** (`HEADER_CTRL` + `HEADER_PRIMARY`/`HEADER_NEUTRAL`/`HEADER_DANGER`): `inline-flex items-center justify-center gap-1 border h-11 sm:h-[30px] text-xs font-medium transition-colors shrink-0 disabled:opacity-40 disabled:cursor-not-allowed` + `rounded-md px-3` + Farbsatz. Icon-only `px-2` (`HEADER_CTRL_ICON`), Split-Button `rounded-l-md` / `rounded-r-md border-l-brand-black/20`.
- **Header-Feld** (`HEADER_FIELD`, Input/Select in der Kopfzeile): `border border-brand-border rounded-md h-11 sm:h-[30px] px-3 bg-white text-xs text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`
- **Button Primary:** `bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Button Small (Tabellen):** `bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Button Danger:** `bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Input:** `w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`
- **Card:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6` (Tabellen-Container: `… overflow-hidden`)
- **Modal:** `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`
- **Alert Info:** `p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text`
- **Alert Fehler:** `p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger`
- **Tabellen-Header (th):** `bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left`
- **Tabellen-Row:** `hover:bg-brand-table-select transition-colors` / Zelle `px-4 py-3 text-sm text-brand-text`

**Icons (lucide-react):** Keine Unicode/Emojis in JSX. `☰`→`<Menu>`, `✕`→`<X>`, `⋮`→`<MoreVertical>`, `▸/▾`→`<ChevronRight>/<ChevronDown>`, `✓`→`<Check>`, `⚠`→`<AlertTriangle>`, `🗑`→`<Trash2>`, `«/»`→`<ChevronsLeft>/<ChevronsRight>`, Heim→`<Home>`, Auswärts→`<MapPin>`. Größen `w-4 h-4` (inline) · `w-5 h-5` (Buttons/Nav) · `w-6 h-6` (standalone). Icon-only-Buttons brauchen `aria-label`.

**Button-Position:** Listen → Primär oben rechts neben `<h1>`; Formulare → unten; Inline-Form in Karte → unten in der Karte. Die Position bestimmt die Größe: oben neben `<h1>` = Header-Control, unten im Formular/in der Karte = Primary.

## Mobile & PWA

- **Breakpoint:** `sm:` (640px) ist die einzige Mobile/Desktop-Grenze. Keine `md:`-Logik für Mobile.
- **Navigation:** Hamburger (`<Menu>`) öffnet die Sidebar als Fixed-Overlay (`z-50`) mit Backdrop. Desktop-Sidebar immer sichtbar. Main-Padding Mobile `px-4 py-4` statt `p-8`; Deko-Klassen (`rounded-tl-3xl …`) nur `sm:`.
- **Tabellen auf Mobile:** Card-Layout statt `<table>`; Actions hinter `<MoreVertical>`-Dropdown; Multi-Feld-Inline-Edit als Modal. Shared: `MobileCard`, `ActionMenu`, `EditModal` in `web/src/components/`.
- **Touch-Targets:** min. 44px Höhe → `py-2.5` auf Mobile (`sm:py-1.5`); Header-Controls stattdessen `h-11 sm:h-[30px]` (fixe Höhe, s. o.).
- **PWA** (`vite-plugin-pwa`): Service Worker network-first für `/api/*`, cache-first für Assets. Manifest `web/public/manifest.json`, Icons `web/public/icons/`. Offline-Shell mit Hinweis.
