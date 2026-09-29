/**
 * Die verbindlichen Typografie-Rollen — Gegenstück zu `buttonStyles.ts` für
 * Überschriften und Menüeinträge. Einzige Fundstelle, siehe Capability
 * `component-standards` (Change `typografie-rollen`).
 *
 * Eine Rolle bündelt Größe, Schnitt und Farbe. Einheitlich heißt nicht gleich:
 * die Rollen bilden eine absteigende Treppe (24 fett → 18 fett → 18 halbfett →
 * 16 halbfett), und keine tiefere Ebene ist größer oder kräftiger als ihre
 * übergeordnete. Der Tag (`h1`/`h2`/`h3`) folgt der Dokumentgliederung, die
 * Rolle dem Aussehen — beides ist unabhängig.
 *
 * Verwendung: Konstante plus Layout-Klassen des Aufrufers (`mb-4`, `truncate`,
 * `flex items-center gap-2`). Größen- und Schnittklassen daneben meldet
 * `lib/__tests__/typography.gate.test.ts`; Zustandsfarben (gedämpft,
 * durchgestrichen) bleiben erlaubt.
 */

/** `<h1>` einer App-Seite, auch auf Detailseiten. */
export const PAGE_TITLE = 'text-2xl font-bold text-brand-text'

/**
 * Kartentitel auf Einstiegsseiten (Anmelden, Registrieren, Passwort). Eine
 * Stufe unter dem Seitentitel, weil darüber schon die „TeamWERK“-Marke in
 * Seitentitelgröße steht — zwei 24-px-Zeilen übereinander konkurrieren.
 */
export const ENTRY_TITLE = 'text-xl font-bold text-brand-text'

/** Kopf eines Modals oder Dialogs. */
export const MODAL_TITLE = 'text-lg font-bold text-brand-text'

/** Abschnitt einer Seite bzw. Kopf einer Karte. */
export const SECTION_TITLE = 'text-lg font-semibold text-brand-text'

/** Gliederung innerhalb eines Abschnitts oder Modals. */
export const SUBSECTION_TITLE = 'text-base font-semibold text-brand-text'

/** Kleine Versal-Überschrift über einer Gruppe — bewusst wie der Tabellenkopf. */
export const OVERLINE = 'text-xs font-semibold uppercase text-brand-text-muted'

const MENU_ITEM_BASE =
  'w-full flex items-center gap-2 text-left px-4 py-2.5 text-sm transition-colors'

/** Eintrag in einem Dropdown- oder Aktionsmenü (`role="menuitem"`). */
export const MENU_ITEM = `${MENU_ITEM_BASE} text-brand-text hover:bg-brand-surface-card`

/** Destruktiver Menüeintrag — unterscheidet sich nur in der Farbe. */
export const MENU_ITEM_DANGER = `${MENU_ITEM_BASE} text-brand-danger hover:bg-brand-danger-light`
