# termine-unified-view Specification

## Purpose

Diese Spezifikation beschreibt die Capability `termine-unified-view`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Filterzustand in URL-Query-Parametern

Die `/termine`-Seite SHALL ihren Filterzustand vollständig über URL-Query-Parameter abbilden. Beim Mount liest sie die Parameter aus `useSearchParams()` und initialisiert daraus den State. Jede Änderung an Filtern (Team-Auswahl, Termin-Typen, Vergangene anzeigen) MUSS die URL via `setSearchParams()` (Replace, nicht Push) aktualisieren, sodass die Seite per Browser-Back/Forward navigierbar und per Link teilbar ist.

Unterstützte Parameter:
- `team` (eine numerische Team-ID; fehlt → kein Team-Filter)
- `types` (kommaseparierte Werte aus `training`, `heim`, `auswaerts`; fehlt → alle Typen aktiv, identisch zum bisherigen Default)
- `past` (`1` zeigt vergangene Termine, default `0`)

Ungültige oder unbekannte Werte SHALL ignoriert und der jeweilige Filter auf seinen Default zurückgesetzt werden — ohne Fehlermeldung.

#### Scenario: Page lädt mit Team-Filter aus URL
- **WHEN** ein User `/termine?team=3` aufruft
- **THEN** ist der Team-Filter beim ersten Render auf Team-ID 3 vorbelegt
- **THEN** zeigt die Liste ausschließlich Termine dieses Teams

#### Scenario: Page lädt mit Typ- und Past-Filter aus URL
- **WHEN** ein User `/termine?types=heim,auswaerts&past=1` aufruft
- **THEN** sind nur die Termin-Typen „Heimspiel" und „Auswärtsspiel" aktiv
- **THEN** ist „Vergangene anzeigen" aktiviert

#### Scenario: Filteränderung schreibt URL zurück
- **WHEN** ein User auf der `/termine`-Seite den Team-Filter auf Team 5 ändert
- **THEN** wird die URL via `replaceState` auf `/termine?team=5` aktualisiert
- **THEN** verändert sich der Browser-History-Stack nicht (kein neuer Eintrag pro Filteränderung)

#### Scenario: Ungültiger Query-Parameter
- **WHEN** ein User `/termine?team=abc&types=foo,bar` aufruft
- **THEN** verhält sich die Seite wie ohne Filter (Default-State, kein Fehler)

#### Scenario: Kein Query-Parameter (Rückwärtskompatibilität)
- **WHEN** ein User `/termine` ohne Query-Parameter aufruft
- **THEN** ist das Verhalten identisch zu vorher (Default-Filter, keine sichtbare Änderung)

---

### Requirement: Deep-Link-Fokus auf einzelnen Termin

Die `/termine`-Seite SHALL einen zusätzlichen Query-Parameter `focus` akzeptieren. Format: `focus=<type>-<id>` mit `type ∈ {training, game}` und `id` als numerische ID (z.B. `focus=training-42`, `focus=game-17`). Bei gültigem `focus` MUSS die Seite:

1. nach Datenladen den betreffenden Termin in den Viewport scrollen (`scrollIntoView({ behavior: 'smooth', block: 'center' })`),
2. die Karte für ca. 2 Sekunden visuell hervorheben (Yellow-Ring, fade-out via Tailwind-Animation),
3. automatisch „Vergangene anzeigen" aktivieren, falls der Termin in der Vergangenheit liegt (damit Push-Links zu Spielen, die kurz danach beginnen, nicht ins Leere zeigen).

Wenn die ID nicht in der geladenen Liste existiert (z.B. fremdes Team, gelöschter Termin), SHALL die Seite eine dezente Hinweismeldung „Dieser Termin ist nicht verfügbar" anzeigen — als nicht-blockierende Info oberhalb der Liste — und ansonsten normal funktionieren.

#### Scenario: Push-Notification öffnet konkreten Spieltermin
- **WHEN** ein User über einen Push-Link `/termine?focus=game-17` öffnet
- **THEN** wird zum Spiel mit ID 17 in der Liste gescrollt
- **THEN** ist die Karte kurzzeitig visuell hervorgehoben

#### Scenario: Focus auf vergangenen Termin
- **WHEN** ein User `/termine?focus=training-5` öffnet und Training 5 liegt in der Vergangenheit
- **THEN** aktiviert die Seite automatisch „Vergangene anzeigen"
- **THEN** scrollt sie zum Training 5

#### Scenario: Focus auf nicht existierende ID
- **WHEN** ein User `/termine?focus=game-99999` öffnet und das Spiel existiert nicht (oder gehört zu einem fremden Team)
- **THEN** zeigt die Seite eine Info „Dieser Termin ist nicht verfügbar"
- **THEN** rendert die Seite ansonsten normal die Termin-Liste

#### Scenario: Ungültiges Focus-Format
- **WHEN** ein User `/termine?focus=foobar` öffnet
- **THEN** wird der `focus`-Parameter ignoriert, die Seite rendert normal

#### Scenario: Focus kombiniert mit Filter
- **WHEN** ein User `/termine?team=2&types=heim&focus=game-17` öffnet
- **THEN** werden Filter angewendet UND auf den fokussierten Termin gescrollt (sofern er nicht durch den Filter ausgeblendet würde; in dem Fall werden die einschränkenden Filter, die genau diesen Termin verbergen würden, ignoriert)

---

### Requirement: Scroll-Position beim Vergangene-Toggle beibehalten

Beim Umschalten von „Vergangene anzeigen" SHALL die `/termine`-Seite die aktuell betrachtete Position beibehalten: Der zuvor oberste sichtbare Termin MUSS nach dem Neuladen an derselben Viewport-Position bleiben. Neu eingeblendete vergangene Termine erscheinen oberhalb; der Nutzer scrollt selbst nach oben, um sie zu sehen.

Da der Scrollcontainer der Seite das `<main>`-Element in `AppShell` (`overflow-auto`) ist und nicht das `window`, MUSS die Wiederherstellung auf diesem Container erfolgen (ermittelt via `el.closest('main')`), nicht über `window.scrollBy`. Die Messung erfolgt viewport-relativ (`getBoundingClientRect().top`), die Wiederherstellung relativ (`scrollBy`) auf dem Container.

Existiert der zuvor als Anker gemerkte Termin nach dem Umschalten nicht mehr (z.B. der Anker war ein vergangener Termin und „Vergangene" wird ausgeschaltet), SHALL keine Wiederherstellung erfolgen und die Liste am Anfang stehen bleiben.

Der Deep-Link-Fokus (`?focus=…`) behält Vorrang: Liegt ein aktiver Fokus vor, unterbleibt die Scroll-Wiederherstellung zugunsten des Fokus-Scrolls.

#### Scenario: Position bleibt beim Einblenden vergangener Termine erhalten
- **WHEN** ein Nutzer auf `/termine` einen Termin betrachtet und „Vergangene anzeigen" aktiviert
- **THEN** bleibt der zuvor oberste sichtbare Termin an derselben Viewport-Position
- **THEN** erscheinen die eingeblendeten vergangenen Termine oberhalb dieser Position

#### Scenario: Position bleibt beim Ausblenden vergangener Termine erhalten
- **WHEN** ein Nutzer mit eingeblendeten Vergangenen einen zukünftigen Termin betrachtet und „Vergangene anzeigen" deaktiviert
- **THEN** bleibt dieser Termin an derselben Viewport-Position (sofern er weiterhin sichtbar ist)

#### Scenario: Anker nicht mehr vorhanden nach Ausblenden
- **WHEN** der oberste sichtbare Termin ein vergangener Termin war und der Nutzer „Vergangene anzeigen" deaktiviert
- **THEN** erfolgt keine Scroll-Wiederherstellung und die Liste steht am Anfang

#### Scenario: Fokus hat Vorrang vor Wiederherstellung
- **WHEN** die Seite mit aktivem `?focus=…` neu lädt und gleichzeitig eine Scroll-Wiederherstellung anstünde
- **THEN** wird zum fokussierten Termin gescrollt und die Wiederherstellung unterbleibt

---

### Requirement: Trennlinie „heute" vor dem ersten nicht-vergangenen Termin

Die `/termine`-Liste SHALL eine Trennlinie mit der Beschriftung „heute" unmittelbar vor dem ersten Termin rendern, dessen Datum nicht in der Vergangenheit liegt (`date >= today`, verglichen auf Tagesebene via `date.slice(0,10)`). Die Trennlinie SHALL nur erscheinen, wenn davor mindestens ein Termin steht (Index des ersten nicht-vergangenen Termins > 0) — andernfalls (alle sichtbaren Termine liegen in Gegenwart/Zukunft) wird sie nicht gerendert.

Die Trennlinie ist ein rein visuelles, nicht anklickbares Element und trägt keine `termin-…`-ID, sodass sie die Anker-Ermittlung der Scroll-Wiederherstellung nicht beeinflusst.

#### Scenario: Trennlinie bei eingeblendeten vergangenen Terminen
- **WHEN** vergangene und zukünftige Termine in der Liste sichtbar sind
- **THEN** erscheint die Trennlinie „heute" unmittelbar vor dem ersten Termin mit `date >= today`

#### Scenario: Keine Trennlinie ohne vergangene Termine
- **WHEN** alle sichtbaren Termine in Gegenwart oder Zukunft liegen (z.B. „Vergangene" ist ausgeschaltet)
- **THEN** wird keine „heute"-Trennlinie gerendert

#### Scenario: Trennlinie beeinflusst Scroll-Anker nicht
- **WHEN** die Scroll-Wiederherstellung den obersten sichtbaren Termin als Anker ermittelt
- **THEN** wird die „heute"-Trennlinie dabei nicht als Anker herangezogen
