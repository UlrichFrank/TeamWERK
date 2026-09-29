# Design: typografie-rollen

## Context

Die Motivation steht in proposal.md unter „Why“. Maßgeblich für den Ansatz ist das bestehende Muster in `web/src/lib/buttonStyles.ts` samt `lib/__tests__/buttonStyles.gate.test.ts`: Es gibt eine Fundstelle je Klassen-String, die Aufrufer hängen nur Layout-Klassen an, und ein textuelles Gate mit Allowlist und Poison-Tests hält die Regel ein. Das hat bei den Header-Controls funktioniert; die Kopfzeilen sind seitdem einheitlich hoch. Die Überschriften brauchen dieselbe Figur, aber ein anderes Erkennungsmerkmal, weil eine handgesetzte Überschrift keine wörtliche Kopie eines Strings ist, sondern eine freie Kombination.

Die Schrift selbst ist bereits einheitlich: `tailwind.config.js` setzt `fontFamily.sans` auf Hanken Grotesk, und `font-mono` kommt nur in Code-/Token-Anzeigen vor. Offen ist also die **Größe und der Schnitt**, nicht die Familie.

## Goals / Non-Goals

**Goals:**
- Eine Typo-Hierarchie aus sechs Überschriften-Rollen plus Menüeintrag, jede an genau einer Stelle definiert.
- Ein Gate, das eine neu handgesetzte Überschrift auf Anhieb meldet.

**Non-Goals:**
- Keine Rollen für Fließtext, Hilfetexte, Badges oder Tabellenzellen (siehe proposal.md).
- Keine Umbenennung von Überschriften-Ebenen (`h2` → `h3`) aus Barrierefreiheitsgründen. Die Rolle bestimmt das Aussehen, die Tag-Ebene bleibt, wie sie ist. Eine Korrektur der Dokumentgliederung wäre ein eigener a11y-Change.
- Keine Tailwind-Theme-Erweiterung (`fontSize`-Tokens wie `text-title`). Die Rollen bündeln Größe, Schnitt **und** Farbe; ein Theme-Token deckte nur die Größe ab und ließe die Schnitt-Abweichungen (`bold` gegen `semibold`) bestehen, die heute den größten Teil der Uneinheitlichkeit ausmachen.

## Decisions

### 1. Eigene Datei `lib/typography.ts` statt Erweiterung von `buttonStyles.ts`

`buttonStyles.ts` beschreibt Bedienelemente. Überschriften sind keine, und die Datei wäre mit sechs weiteren Konstanten kein „Buttons und Controls“-Modul mehr. `MENU_ITEM` gehört dem Wesen nach zu den Bedienelementen, steht aber trotzdem in `typography.ts`: Es ist dieselbe Frage (welche Schriftgröße hat dieses Element?), und das Gate für beide liegt in einer Datei. Die Alternative, `MENU_ITEM` in `buttonStyles.ts` zu legen, hätte zwei Gates für eine Regel bedeutet.

### 2. Die Werte folgen der Mehrheit bzw. der bestehenden Doku, nicht einem neuen Entwurf

- `PAGE_TITLE` = `text-2xl font-bold text-brand-text`: 38 von 44 `<h1>` tragen das heute schon, `05-frontend.md` schreibt es vor.
- `MODAL_TITLE` = `text-lg font-bold text-brand-text`: das nutzen `EditModal` und die Doku.
- `SECTION_TITLE` = `text-lg font-semibold text-brand-text`: Gegenüber dem Modal-Titel ist er eine Stufe leichter im Schnitt. Seitentitel (24 px bold), Abschnitt (18 px semibold) und Unterabschnitt (16 px semibold) ergeben eine sichtbare Treppe.
- `ENTRY_TITLE` = `text-xl font-bold text-brand-text`: Auf den Einstiegsseiten (Login, Registrieren, Passwort) steht darüber bereits die 24-px-Marke „TeamWERK“. Ein zweiter 24-px-Titel direkt darunter würde mit ihr konkurrieren, deshalb bleibt es bei der heutigen Größe.
- `OVERLINE` = `text-xs font-semibold uppercase text-brand-text-muted`: bewusst identisch mit dem Tabellenkopf.

**Abgewogene Alternative für die Profil-Abschnitte:** Die gedämpften 16-px-Köpfe (`font-semibold text-brand-text-muted`, 20×) als eigene Rolle „Kartenkopf“ behalten. Dagegen spricht, dass dieselbe semantische Rolle (Überschrift einer Karte auf einer Seite) dann zwei Erscheinungsbilder hätte, abhängig davon, ob sie im Profil oder im Admin-Bereich steht. Genau diese Uneinheitlichkeit soll der Change beseitigen. Das Profil ändert sich dadurch sichtbar; das ist gewollt und in proposal.md benannt.

### 3. Gate-Erkennung: Tag-basiert statt metrik-basiert

Das Button-Gate sucht wörtliche Kopien bekannter Strings. Für Überschriften taugt das nicht, weil die Abweichungen gerade *keine* Kopien sind. Stattdessen:

- Jede `<h1>`, `<h2>` und `<h3>` in `pages/` und `components/` (ohne Tests) wird gefunden, ihr `className`-Ausdruck wird bis zum Ende des Attributs gelesen (String-Literal oder Template-Literal, auch mehrzeilig).
- **Verstoß**, wenn der Ausdruck eine Größenklasse (`text-xs` bis `text-4xl`, `text-[…]`) oder eine Schnittklasse (`font-medium|semibold|bold|extrabold`) enthält. Das gilt auch in den Literal-Teilen eines Template-Literals.
- **Verstoß**, wenn der Ausdruck keine der Rollen-Konstanten referenziert. Das fängt auch eine Überschrift ohne jede Klasse, die sonst über den Tailwind-Preflight als ungestylter Fließtext erschiene.
- Farbklassen werden **nicht** geprüft. Eine Überschrift darf situativ gedämpft oder durchgestrichen erscheinen (abgesagter Termin: `line-through opacity-60`). Das ist Zustand, nicht Rolle.

Menüeinträge werden über `role="menuitem"` erkannt: Ein Element mit diesem Attribut muss `MENU_ITEM` oder `MENU_ITEM_DANGER` referenzieren. Wo Menüeinträge das Attribut heute nicht tragen, bekommen sie es im Zuge der Migration. Das verbessert zugleich die Barrierefreiheit, und das Gate erfasst die Einträge damit.

Wie beim Button-Gate ist die Prüfung textuell und keine AST-Analyse. Der Preis ist bekannt und in Risiken benannt.

### 4. Freie gefüllte Buttons: Zuordnung nach Position, nicht nach heutiger Größe

Die Button-Position bestimmt die Rolle (`05-frontend.md`): neben der `<h1>` gilt `HEADER_*`, am Ende eines Formulars oder einer Karte `BTN_PRIMARY`/`BTN_DANGER`, in einer Tabellen- oder Listenzeile `BTN_SMALL`. Voll breite Formular-Buttons (Login, Chat-Modals) behalten `w-full` als Layout-Klasse am Aufrufer. Zwei Muster hatten keine Konstante und bekommen eine in `buttonStyles.ts`: `BTN_SMALL_DANGER` (destruktive Zeilen-Aktion wie „Austragen“) und `BTN_PRIMARY_SPLIT_MAIN`/`_CARET` (Karten-Aktion mit Zusatzmenü, „Bild hochladen“). Beide sind als Metrik im bestehenden Button-Gate erfasst. Aktionen neben einer Unterüberschrift (Kopf einer Karte oder eines Modal-Abschnitts) nutzen die Header-Controls, weil die Zeile dieselbe Figur hat wie die Kopfzeile einer Seite. Für gefüllte Buttons wird **kein** neues Gate gebaut: Die Erkennung „gefüllt + gerundet + gepolstert“ trifft Chips, Badges und Checkbox-Labels mit, und die Allowlist würde länger als die Regel. Das bestehende Button-Gate zusammen mit dem Review reicht hier aus.

## Risks / Trade-offs

- [Eine Überschrift, die als `<div>`/`<p>` statt als `<h*>` gebaut ist, sieht das Gate nicht] → Bei der Migration werden Modal-Köpfe per Suche nach `text-lg font-bold`/`font-semibold` außerhalb von `<h*>` mit erfasst. Dauerhaft bleibt das eine Lücke, dieselbe Grenze wie beim textuellen Button-Gate.
- [Die sichtbare Änderung im Profil und in Modals fällt Nutzern auf] → Die Änderung ist rein optisch. Sie wird im Changelog-Eintrag der Version als „einheitlichere Überschriften“ benannt.
- [Bestehende Vitest-Tests fragen Überschriften über Klassen ab] → Sie werden in der jeweiligen Migrationsgruppe angepasst. Die Tests fragen überwiegend über `getByRole('heading')` ab und sind davon nicht betroffen.
- [Das Gate meldet legitime Sonderfälle (Markdown-Renderer, Viewer-Toolbar)] → Die Allowlist ist begründet und hat einen Verwaist-Check wie `buttonStyles.gate.test.ts`.

## Migration Plan

Reines Frontend-Refactoring, ausgeliefert mit dem nächsten `make deploy`. Ein Rollback erfolgt über `make deploy-rollback` und hat keine Datenwirkung.
