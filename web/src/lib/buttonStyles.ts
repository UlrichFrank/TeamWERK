/**
 * Die verbindlichen Klassen-Strings für Buttons und Bedienelemente.
 * Einzige Fundstelle — siehe Capability `component-standards`.
 *
 * Zwei Rollen, die nicht gegeneinander austauschbar sind:
 *
 * - **Header-Control** (`HEADER_*`): alles, was in der Zeile der Seitenüberschrift
 *   steht — Aktions-Button, Filter-Chip, Suchfeld, Auswahlfeld. 32px, auf jeder
 *   Breite gleich.
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
 * Der Wert selbst steht bei `HEADER_H`.
 *
 * Verwendung: Basis + Farbsatz kombinieren, Layout-Klassen am Aufrufer anhängen.
 *
 *   className={`${HEADER_CTRL} ${HEADER_PRIMARY}`}
 *   className={`${HEADER_CTRL} ${active ? HEADER_PRIMARY : HEADER_NEUTRAL}`}
 */

/**
 * Die Höhe jedes Bedienelements in einer Kopfzeile. Auch für Elemente gedacht,
 * die sonst keinen der Strings hier verwenden können, etwa ein Suchfeld mit
 * Icon-Padding.
 *
 * Desktop 30px — die gewachsene Höhe der Filterleiste, unverändert. Mobile 32px,
 * die Höhe des „Bild hochladen"-Buttons im Profil (`px-3 py-1.5 text-sm`).
 *
 * Die 32px unterschreiten bewusst das 44px-Touch-Target aus
 * `docs/agent/05-frontend.md`: die Kopfzeile ist eine dichte Leiste aus
 * Suchfeld, Filtern und einer Aktion, und ein Sprung auf 44px macht sie auf
 * kleinen Geräten unverhältnismäßig hoch. Für Buttons außerhalb der Kopfzeile
 * gilt die Regel unverändert.
 */
export const HEADER_H = 'h-8 sm:h-[30px]'

/**
 * Platz für den Auswahlpfeil eines `<select>`. `@tailwindcss/forms` zeichnet ihn
 * als Hintergrundbild 0,5rem vom rechten Rand, 1,5em breit, und reserviert dafür
 * `padding-right: 2.5rem` — ein `px-3` am Feld überschreibt das, und der Text
 * läuft unter den Pfeil (Staffel-Auswahl: „mA1" mit Chevron mitten im Wort).
 * Bei 16px (Mobile, iOS-Zoom-Schutz) braucht der Pfeil 32px, daher `pr-9`.
 * Nur für `select`: Text- und Suchfelder behalten ihr symmetrisches Padding.
 */
export const SELECT_CHEVRON_ROOM = '[&:is(select)]:pr-9'

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
  `border border-brand-border rounded-md ${HEADER_H} px-3 ${SELECT_CHEVRON_ROOM} bg-white ` +
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

/**
 * Eingabefeld in Formularen und Modals (Text, Datum, Zahl, Select, Textarea).
 * Layout-Klassen (`mb-*`, `resize-none`, Icon-Padding) hängt der Aufrufer an.
 */
export const INPUT =
  `w-full border border-brand-border rounded-md px-3 ${SELECT_CHEVRON_ROOM} py-2 text-sm text-brand-text ` +
  'placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 ' +
  'focus:ring-brand-yellow focus:border-brand-yellow'

/** Beschriftung über einem Formularfeld. */
export const LABEL = 'block text-sm font-medium text-brand-text-muted mb-1'

/**
 * Filter-/Suchgruppe in einer umbrechenden Kopfzeile (`flex flex-wrap`).
 *
 * Bewusst `min-w-[12rem]` und nicht `min-w-0`: ein `flex-1`-Item hat Basis 0,
 * und mit Mindestbreite 0 passt es beim Zeilenumbruch rechnerisch IMMER noch in
 * die erste Zeile. Auf Mobile wurde die Gruppe dadurch auf wenige Pixel
 * gequetscht, und Filter-Button und Suchfeld lagen über den Aktions-Buttons
 * rechts daneben (Kalender: Abwesenheits-Button komplett verdeckt). Mit
 * Mindestbreite bricht die Gruppe stattdessen in eine eigene Zeile um.
 */
export const HEADER_GROUP = 'flex items-center gap-1.5 flex-1 flex-nowrap min-w-[12rem]'

/**
 * Tab-Leiste unter der Seitenüberschrift. Scrollt auf schmalen Bildschirmen
 * horizontal statt umzubrechen: eine umbrochene Leiste mit Unterstrich sieht aus
 * wie zwei Leisten, eine nicht scrollende (so war es in den Einstellungen) schiebt
 * die ganze Seite seitlich aus dem Bild. Abstand nach unten setzt der Aufrufer.
 */
export const TAB_BAR =
  'flex gap-1 border-b border-brand-border-subtle overflow-x-auto ' +
  // Scrollbalken ausblenden: er lag als zweite, graue Linie unter dem Unterstrich.
  // Wischen bleibt möglich, der angeschnittene letzte Tab zeigt, dass es weitergeht.
  '[scrollbar-width:none] [&::-webkit-scrollbar]:hidden'

/** Einzelner Tab; mit `TAB_ACTIVE` bzw. `TAB_INACTIVE` kombinieren. */
export const TAB =
  'inline-flex items-center gap-1 px-4 py-2 text-sm font-medium border-b-2 ' +
  'transition-colors whitespace-nowrap shrink-0'
export const TAB_ACTIVE = 'border-brand-yellow text-brand-text'
export const TAB_INACTIVE = 'border-transparent text-brand-text-muted hover:text-brand-text'
