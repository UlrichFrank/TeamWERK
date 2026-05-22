## ADDED Requirements

### Requirement: Diensttyp trägt consecutive-Verhalten

Ein Diensttyp SHALL die Felder `consecutive_behavior` (TEXT, DEFAULT `'normal'`) und `consecutive_variant_id` (INTEGER NULL, FK auf `duty_types.id`) besitzen.

`applies_when` wird NICHT gespeichert, sondern bei der Slot-Generierung berechnet.

Gültige Werte für `consecutive_behavior`: `normal`, `skip`, `reduced`.

Wenn `consecutive_behavior = 'reduced'`, MUSS `consecutive_variant_id` gesetzt sein; andernfalls gibt die API einen 400-Fehler zurück.

#### Scenario: Diensttyp mit consecutive_behavior speichern
- **WHEN** Admin speichert Diensttyp „Aufbau" mit `consecutive_behavior=reduced`, `consecutive_variant_id=<id von Kleiner Aufbau>`
- **THEN** wird der Diensttyp gespeichert und bei der nächsten Slot-Generierung wird die consecutive-Logik angewendet

#### Scenario: reduced ohne variant_id wird abgelehnt
- **WHEN** Admin speichert Diensttyp mit `consecutive_behavior=reduced` aber ohne `consecutive_variant_id`
- **THEN** gibt die API HTTP 400 zurück

---

### Requirement: Slot-Generierung berechnet applies_when aus Spieltagsposition

Beim Generieren von Slots für ein Spiel G (Preview und Regenerate) SHALL das Backend für jeden Template-Item folgende Berechnung vornehmen:

**applies_when wird berechnet als:**
- `'day_open'`: wenn G das erste Heimspiel des Tages ist (kein Heimspiel mit früherem Anpfiff an G.date in derselben Saison)
- `'day_close'`: wenn G das letzte Heimspiel des Tages ist (kein Heimspiel mit späterem Anpfiff an G.date in derselben Saison)
- `'always'`: alle anderen Fälle

**Dann wird der Slot generiert, wenn:**
- `applies_when = 'always'`: immer
- `applies_when = 'day_open'`: immer (dieses Spiel ist das erste am Tag)
- `applies_when = 'day_close'`: immer (dieses Spiel ist das letzte am Tag)

#### Scenario: Zweites Spiel am Spieltag bekommt keinen Aufbau
- **WHEN** an einem Tag existieren zwei Heimspiele (11:00 und 14:00) und Template-Item „Aufbau" hat `applies_when=day_open`
- **THEN** wird für das 14:00-Spiel kein Aufbau-Slot generiert, für das 11:00-Spiel schon

#### Scenario: Einziges Spiel des Tages bekommt Aufbau und Abbau
- **WHEN** an einem Tag gibt es genau ein Heimspiel und Template hat sowohl „Aufbau" (day_open) als auch „Abbau" (day_close)
- **THEN** werden beide Slots generiert

---

### Requirement: Slot-Generierung wendet consecutive_behavior an

Nach der Berechnung von `applies_when` prüft das Backend, ob `consecutive_behavior != 'normal'` ist. Wenn ja und die folgende Bedingung erfüllt ist, wird das consecutive-Verhalten angewendet:

- Für `applies_when = 'day_open'`: prüfe ob am Vortag (G.date − 1) Heimspiele in der Saison existieren
- Für `applies_when = 'day_close'`: prüfe ob am Folgetag (G.date + 1) Heimspiele in der Saison existieren
- Für `applies_when = 'always'`: keine Nachbartagsprüfung (consecutive_behavior hat keine Wirkung)

Wenn die Bedingung zutrifft:
- `consecutive_behavior = 'skip'`: keinen Slot generieren
- `consecutive_behavior = 'reduced'`: Slot mit `consecutive_variant_id` statt Original-Diensttyp generieren

#### Scenario: Samstag-Abbau wird durch Kleinen Abbau ersetzt wenn Sonntagsspiel existiert
- **WHEN** Samstag hat ein Heimspiel, Sonntag ebenfalls, und „Abbau" hat `consecutive_behavior=reduced`, `consecutive_variant_id=<Kleiner Abbau>`
- **THEN** wird für das Samstag-Spiel ein Slot mit Diensttyp „Kleiner Abbau" generiert, nicht „Abbau"

#### Scenario: Aufbau entfällt wenn Vortag Spieltag war und skip gesetzt
- **WHEN** Sonntag hat ein Heimspiel, Samstag hatte auch Heimspiele, und „Aufbau" hat `consecutive_behavior=skip`
- **THEN** wird für das Sonntag-Spiel kein Aufbau-Slot generiert

#### Scenario: Normaler Abbau wenn kein Folgetag
- **WHEN** Samstag hat ein Heimspiel aber Sonntag nicht, und „Abbau" hat `consecutive_behavior=reduced`
- **THEN** wird der normale Abbau-Slot generiert (kein Folgetag → Bedingung nicht erfüllt)

---

### Requirement: Admin-UI zeigt und pflegt consecutive-Verhalten

Die Admin-Oberfläche für Diensttypen SHALL die zwei neuen Felder `consecutive_behavior` und `consecutive_variant_id` anzeigen und bearbeitbar machen.

`applies_when` wird NICHT in der Admin-UI angezeigt oder bearbeitet (es wird berechnet).

`consecutive_behavior` wird als Dropdown mit Optionen `normal`, `skip`, `reduced` dargestellt. `consecutive_variant_id` wird als Dropdown aller Diensttypen angezeigt und ist nur sichtbar wenn `consecutive_behavior = 'reduced'`.

#### Scenario: variant-Dropdown erscheint bei reduced
- **WHEN** Admin wählt `consecutive_behavior = 'reduced'`
- **THEN** erscheint ein Dropdown zur Auswahl des Ersatz-Diensttyps

#### Scenario: variant-Dropdown verschwindet bei normal/skip
- **WHEN** Admin wechselt von `reduced` zu `normal` oder `skip`
- **THEN** wird `consecutive_variant_id` geleert und das Feld verschwindet
