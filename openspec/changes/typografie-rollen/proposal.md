# Proposal: typografie-rollen

## Why

Für Buttons und Header-Controls gibt es in `web/src/lib/buttonStyles.ts` verbindliche Klassen-Strings, und ein Gate hält sie ein. Für Überschriften fehlt so etwas: jede Seite und jedes Modal setzt Schriftgröße und Schriftschnitt von Hand. Die Bestandsaufnahme (09/2026) zeigt, wohin das führt:

- **Modal-Titel** gibt es in drei etwa gleich häufigen Varianten: `text-lg font-bold` (26×), `text-lg font-semibold` (22×) und `text-base font-semibold` (22×). `EditModal` und `docs/agent/05-frontend.md` sehen `text-lg font-bold` vor.
- **Abschnitts-Überschriften auf Seiten** (`<h2>` in Karten) kommen in mindestens fünf Varianten vor. Im Profil stehen sie gedämpft in `base` (`font-semibold text-brand-text-muted`, 20×), im Datenschutz dunkel in `lg`, im Admin-Bereich mal `sm medium`, mal `lg bold`.
- **Dropdown-Menüeinträge** sind teils `text-xs` (5×), teils `text-sm` (12×). Nebeneinander geöffnet wirken die Menüs verschiedener Seiten deshalb wie aus verschiedenen Anwendungen.
- **Einzelne gefüllte Buttons** (`py-1.5 text-sm`, `py-2 text-sm`, Split-Button in den Dienstvorlagen) bauen die `BTN_*`-/`HEADER_*`-Konstanten nach und weichen dabei in Höhe und Schriftgröße ab. Das Gate erkennt sie nicht, weil die Kopie nicht wörtlich ist.

„Konsistent“ heißt dabei nicht, dass alles gleich aussieht. Jede **Rolle** soll genau eine Ausprägung haben, und die Rollen sollen eine erkennbare Hierarchie bilden.

## What Changes

- Neue Datei `web/src/lib/typography.ts` mit einem String je Rolle:
  - `PAGE_TITLE` (`<h1>` einer App-Seite)
  - `ENTRY_TITLE` (Kartentitel auf Einstiegsseiten wie Anmelden und Registrieren)
  - `MODAL_TITLE`
  - `SECTION_TITLE` (Abschnitt einer Seite bzw. Kartenkopf)
  - `SUBSECTION_TITLE`
  - `OVERLINE` (kleine Versal-Zwischenzeile)
  - `MENU_ITEM` und `MENU_ITEM_DANGER` (Einträge in Dropdown- und Aktionsmenüs)
- Alle `<h1>`/`<h2>`/`<h3>` in `pages/` und `components/` beziehen ihre Typografie aus diesen Konstanten. Layout-Klassen wie `mb-*`, `flex` oder `truncate` hängt der Aufrufer weiterhin an.
- **Sichtbare Änderung:** Die gedämpften Abschnittsköpfe im Profil und in der Mitgliederverwaltung werden zu `SECTION_TITLE`, also dunkel und eine Stufe größer. Kleine Modal-Titel (`text-base`) werden zu `text-lg font-bold`, Menüeinträge durchgängig `text-sm`.
- Frei nachgebaute gefüllte Buttons werden auf `BTN_PRIMARY`/`BTN_SMALL`/`BTN_DANGER`/`HEADER_*` umgestellt. Die Seiten-Tabs in `AdminTrainingsPage` werden auf `TAB`/`TAB_ACTIVE`/`TAB_INACTIVE` umgestellt.
- Neuer Gate-Test `web/src/lib/__tests__/typography.gate.test.ts`: Eine Überschrift, deren `className` Schriftgröße oder Schriftschnitt als Literal setzt, statt eine Rollen-Konstante zu nutzen, lässt den Test fehlschlagen. Menüeinträge werden über ihre Metrik erkannt. Dazu kommen eine Allowlist mit Begründung und Poison-Tests.
- `docs/agent/05-frontend.md` beschreibt die Rollen, verweist auf die Fundstelle und schreibt die Strings nicht ab.

Nicht Teil des Changes: Fließtext-Größen, Tabellenzellen, Badges und Chips. Deren Größen sind bereits weitgehend über die bestehenden Tabellen- und Alert-Strings geregelt, und eine vollständige Typo-Erfassung aller `<p>`/`<span>` stünde in keinem Verhältnis zum Ertrag.

## Capabilities

### New Capabilities

<!-- keine -->

### Modified Capabilities

- `component-standards`: Neue Requirements zu verbindlichen Typografie-Rollen (Überschriften, Menüeinträge), zu ihrer Herkunft aus `lib/typography.ts` und zum Gate.

## Impact

- Frontend: `web/src/lib/typography.ts` (neu), rund 60 Seiten und Komponenten mit Überschriften, `ActionMenu`, `EditModal` sowie die Dropdown-Menüs in Kalender, Nutzerverwaltung, Veranstaltungsorten und Videos.
- Tests: neuer Gate-Test; bestehende Tests, die Überschriften per Klasse abfragen, werden bei Bedarf angepasst.
- Kein Backend, keine API, keine Migration.
- Sichtbar für alle Nutzer, aber nur optisch: Größe und Schnitt einzelner Überschriften und Menüeinträge ändern sich, am Verhalten ändert sich nichts.
