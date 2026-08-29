/**
 * Die verbindlichen Klassen-Strings für Buttons und Bedienelemente.
 * Einzige Fundstelle — siehe Capability `component-standards`.
 *
 * Zwei Rollen, die nicht gegeneinander austauschbar sind:
 *
 * - **Header-Control** (`HEADER_*`): alles, was in der Zeile der Seitenüberschrift
 *   steht — Aktions-Button, Filter-Chip, Suchfeld, Auswahlfeld. 30px auf Desktop,
 *   44px auf Mobile.
 * - **Formular-Aktion** (`BTN_*`): die Bestätigungs-Aktion eines Formulars oder
 *   Modals. Größer (`text-sm`), weil sie am Ende eines Eingabeflusses steht und
 *   nicht neben einer Überschrift.
 *
 * **Warum die Header-Controls ihre Höhe fix setzen und nicht über `py-*`:**
 * `index.css` zwingt unterhalb von 640px `font-size: 16px` auf `input`, `select`
 * und `textarea` (gegen den iOS-Auto-Zoom, Capability `ios-input-zoom-prevention`),
 * auf `button` aber nicht. Aus derselben `py-2.5`-Klasse entsteht dort deshalb ein
 * 41px hohes Auswahlfeld neben einem 38px hohen Button — eine Kopfzeile aus
 * Suchfeld, Filter und Button läuft auseinander, und kein Padding-Wert repariert
 * das für beide gleichzeitig, weil die Differenz aus der Schriftgröße kommt.
 * Bei fixer Höhe zentriert der Browser den Inhalt unabhängig von der Schriftgröße.
 *
 * `h-[30px]` ist die gewachsene Ist-Höhe der Filterleiste, `h-11` (44px) das
 * Touch-Target-Minimum aus `docs/agent/05-frontend.md`.
 *
 * Verwendung: Basis + Farbsatz kombinieren, Layout-Klassen am Aufrufer anhängen.
 *
 *   className={`${HEADER_CTRL} ${HEADER_PRIMARY}`}
 *   className={`${HEADER_CTRL} ${active ? HEADER_PRIMARY : HEADER_NEUTRAL}`}
 */

/**
 * Die Höhe jedes Bedienelements in einer Kopfzeile. Auch für Elemente gedacht,
 * die sonst keinen der Strings hier verwenden können — etwa ein Suchfeld mit
 * Icon-Padding.
 */
export const HEADER_H = 'h-11 sm:h-[30px]'

/**
 * Gemeinsame Basis aller Header-Controls — ohne Rundung und ohne horizontales
 * Padding, weil Split-Buttons und Icon-only-Varianten beides verändern.
 * Das farblose `border` steht bewusst hier: ohne Rahmen wäre ein Control 2px
 * flacher als seine Nachbarn, und genau diese Abweichung war einer der Ausreißer.
 */
const HEADER_BASE =
  `inline-flex items-center justify-center gap-1 border ${HEADER_H} ` +
  'text-xs font-medium transition-colors shrink-0 ' +
  'disabled:opacity-40 disabled:cursor-not-allowed'

/** Header-Control mit Beschriftung. */
export const HEADER_CTRL = `${HEADER_BASE} rounded-md px-3`

/**
 * Header-Control ohne Beschriftung (Compact-Modus, Icon-Buttons).
 * Schmaler, aber gleich hoch — der Compact-Modus weicht horizontalem
 * Platzmangel aus, nicht vertikalem.
 */
export const HEADER_CTRL_ICON = `${HEADER_BASE} rounded-md px-2`

/** Linke Hälfte eines Split-Buttons (Hauptaktion). */
export const HEADER_SPLIT_MAIN = `${HEADER_BASE} rounded-l-md px-3`

/** Rechte Hälfte eines Split-Buttons (Caret, öffnet das Menü). */
export const HEADER_SPLIT_CARET = `${HEADER_BASE} rounded-r-md px-2 border-l-brand-black/20`

/** Farbsatz: Hauptaktion / aktiver Toggle. */
export const HEADER_PRIMARY =
  'border-brand-yellow bg-brand-yellow text-brand-black ' +
  'hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black'

/** Farbsatz: Nebenaktion / inaktiver Toggle. */
export const HEADER_NEUTRAL =
  'bg-white text-brand-text-muted border-brand-border ' +
  'hover:border-brand-text hover:text-brand-text'

/** Farbsatz: destruktive Aktion in der Kopfzeile. */
export const HEADER_DANGER =
  'border-brand-danger bg-brand-danger text-white hover:bg-brand-danger/90'

/**
 * Farbsatz: rahmenloses Icon in der Kopfzeile (Schließen, Abbrechen).
 * `border-transparent` statt gar keinem Rahmen, damit die Höhe die der
 * Nachbarn bleibt.
 */
export const HEADER_GHOST =
  'border-transparent bg-transparent text-brand-text-muted ' +
  'hover:bg-brand-table-select hover:text-brand-text'

/**
 * Eingabe- und Auswahlfeld in der Kopfzeile. Kein `shrink-0` — anders als die
 * Buttons dürfen Felder in einer engen Zeile schmaler werden, und auf Mobile
 * hängen die Aufrufer `w-full` an.
 */
export const HEADER_FIELD =
  `border border-brand-border rounded-md ${HEADER_H} px-3 bg-white ` +
  'text-xs text-brand-text placeholder:text-brand-text-subtle ' +
  'focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

/** Formular- und Modal-Aktion. */
export const BTN_PRIMARY =
  'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium ' +
  'hover:bg-brand-black hover:text-brand-yellow transition-colors ' +
  'disabled:opacity-40 disabled:cursor-not-allowed'

/** Nebenaktion neben einer Formular-Aktion (Abbrechen, Zurück). */
export const BTN_SECONDARY =
  'border border-brand-border text-brand-text rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium ' +
  'hover:bg-brand-table-select transition-colors ' +
  'disabled:opacity-40 disabled:cursor-not-allowed'

/** Kleiner Button innerhalb einer Tabellenzeile. */
export const BTN_SMALL =
  'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium ' +
  'hover:bg-brand-black hover:text-brand-yellow transition-colors ' +
  'disabled:opacity-40 disabled:cursor-not-allowed'

/** Destruktive Formular- und Modal-Aktion. */
export const BTN_DANGER =
  'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium ' +
  'hover:bg-brand-danger/90 transition-colors ' +
  'disabled:opacity-40 disabled:cursor-not-allowed'
