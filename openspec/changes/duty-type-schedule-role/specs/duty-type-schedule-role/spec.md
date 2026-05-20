## ADDED Requirements

### Requirement: Diensttyp trägt Spieltagsrolle

Ein Diensttyp SHALL die Felder `applies_when` (TEXT, DEFAULT `'always'`), `consecutive_behavior` (TEXT, DEFAULT `'normal'`) und `consecutive_variant_id` (INTEGER NULL, FK auf `duty_types.id`) besitzen.

Gültige Werte für `applies_when`: `always`, `day_open`, `day_close`.
Gültige Werte für `consecutive_behavior`: `normal`, `skip`, `reduced`.

Wenn `consecutive_behavior = 'reduced'`, MUSS `consecutive_variant_id` gesetzt sein; andernfalls gibt die API einen 400-Fehler zurück.

#### Scenario: Diensttyp mit applies_when day_open speichern
- **WHEN** Admin speichert Diensttyp „Aufbau" mit `applies_when=day_open`, `consecutive_behavior=reduced`, `consecutive_variant_id=<id von Kleiner Aufbau>`
- **THEN** wird der Diensttyp gespeichert und bei der nächsten Slot-Generierung nur für das erste Heimspiel des Tages verwendet

#### Scenario: reduced ohne variant_id wird abgelehnt
- **WHEN** Admin speichert Diensttyp mit `consecutive_behavior=reduced` aber ohne `consecutive_variant_id`
- **THEN** gibt die API HTTP 400 zurück

---

### Requirement: Slot-Generierung berücksichtigt Spieltagsposition

Beim Generieren von Slots für ein Spiel G (Preview und Regenerate) SHALL das Backend für jeden Template-Item den zugehörigen Diensttyp prüfen:

- `applies_when = 'always'`: Slot immer generieren
- `applies_when = 'day_open'`: Slot nur generieren wenn G das erste Heimspiel des Tages ist (kein Heimspiel mit früherem Anpfiff an G.date in derselben Saison)
- `applies_when = 'day_close'`: Slot nur generieren wenn G das letzte Heimspiel des Tages ist

#### Scenario: Zweites Spiel am Spieltag bekommt keinen Aufbau
- **WHEN** an einem Tag existieren zwei Heimspiele (11:00 und 14:00) und Template-Item „Aufbau" hat `applies_when=day_open`
- **THEN** wird für das 14:00-Spiel kein Aufbau-Slot generiert, für das 11:00-Spiel schon

#### Scenario: Einziges Spiel des Tages bekommt Aufbau und Abbau
- **WHEN** an einem Tag gibt es genau ein Heimspiel und Template hat sowohl „Aufbau" (day_open) als auch „Abbau" (day_close)
- **THEN** werden beide Slots generiert

---

### Requirement: Slot-Generierung berücksichtigt aufeinanderfolgende Spieltage

Wenn ein Diensttyp `consecutive_behavior != 'normal'` hat, SHALL das Backend prüfen ob der relevante Nachbartag ebenfalls Heimspiele hat:

- Für `day_open`-Dienste: prüfe ob am Vortag (G.date − 1) Heimspiele in der Saison existieren
- Für `day_close`-Dienste: prüfe ob am Folgetag (G.date + 1) Heimspiele in der Saison existieren

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

### Requirement: Admin-UI zeigt und pflegt Spieltagsrolle

Die Admin-Oberfläche für Diensttypen SHALL die drei neuen Felder anzeigen und bearbeitbar machen.

`applies_when` wird als Dropdown mit drei Optionen dargestellt. `consecutive_behavior` wird als Dropdown dargestellt und ist nur editierbar wenn `applies_when != 'always'`. `consecutive_variant_id` wird als Dropdown aller Diensttypen angezeigt und ist nur sichtbar wenn `consecutive_behavior = 'reduced'`.

#### Scenario: consecutive-Felder nur für day_open/day_close editierbar
- **WHEN** Admin wählt `applies_when = 'always'`
- **THEN** sind `consecutive_behavior` und `consecutive_variant_id` ausgegraut / nicht sichtbar

#### Scenario: variant-Dropdown erscheint bei reduced
- **WHEN** Admin wählt `consecutive_behavior = 'reduced'`
- **THEN** erscheint ein Dropdown zur Auswahl des Ersatz-Diensttyps
