## ADDED Requirements

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
