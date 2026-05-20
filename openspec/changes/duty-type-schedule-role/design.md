## Context

Heute generiert `RegenerateSlots` für jedes Heimspiel alle Template-Items ohne Kontext: kein Wissen darüber, ob es weitere Spiele am selben Tag gibt oder ob Vor-/Folgetag ebenfalls Spieltage sind. Aufbau und Abbau entstehen damit redundant bei Mehrfachspieltagen.

Die Slot-Generierung erfolgt in zwei Schritten: Frontend ruft `GET /api/admin/game-template/preview` ab, zeigt dem Nutzer die geplanten Slots und sendet sie dann via `POST /api/admin/games/{id}/regenerate`. Die Intelligenz muss im Preview-Schritt eingebaut werden — dann stimmt die Darstellung und der gespeicherte Plan automatisch.

## Goals / Non-Goals

**Goals:**
- `duty_types` trägt `applies_when`, `consecutive_behavior`, `consecutive_variant_id`
- Preview und Regenerate berücksichtigen same-day-Position und adjacent-day-Kontext
- Admin-UI erlaubt Pflege der neuen Felder

**Non-Goals:**
- Keine Venue-Modellierung — alle Heimspiele teilen dieselbe Halle
- Kein automatisches Neuberechnen bereits gespeicherter Slots
- Kein Einfluss auf manuell erstellte Slots (ohne `game_id`)

## Decisions

### 1. Neue Felder auf `duty_types`, nicht auf `game_template_items`

Die Semantik gehört zum Diensttyp selbst. „Aufbau ist immer day_open" ist eine intrinsische Eigenschaft des Diensttyps, keine Eigenheit eines bestimmten Templates. Jedes Template, das Aufbau enthält, soll automatisch dieselbe Regel erhalten.

Alternative: Felder auf `game_template_items`. Nachteil: Bei mehreren Templates müsste man die Semantik duplizieren und könnte sie inkonsistent pflegen.

### 2. `applies_when`: drei Werte

```
'always'     → jedes Heimspiel (Default — heutiges Verhalten)
'day_open'   → nur wenn kein Heimspiel mit früherem Anpfiff am selben Tag
'day_close'  → nur wenn kein Heimspiel mit späterem Anpfiff am selben Tag
```

Ein Spiel das als einziges am Tag stattfindet ist gleichzeitig `day_open` und `day_close`.

### 3. `consecutive_behavior`: drei Werte + optionale FK

```
'normal'   → adjacent-day hat keinen Einfluss (Default)
'skip'     → Slot weglassen wenn adjacent day condition zutrifft
'reduced'  → anderen Diensttyp verwenden (consecutive_variant_id NOT NULL erforderlich)
```

Adjacent day condition:
- Für `day_open`-Dienste: gibt es Heimspiele am Vortag?
- Für `day_close`-Dienste: gibt es Heimspiele am Folgetag?

### 4. "Same day" und "adjacent day" = is_home=1 + season_id

Da alle Heimspiele in derselben Halle stattfinden, reicht: `WHERE date=? AND is_home=1 AND season_id=?`. Keine Venue-Tabelle nötig.

### 5. Preview muss Spielkontext kennen

`GET /api/admin/game-template/preview` bekommt neue optionale Query-Parameter `game_id` oder `date+season_id`, damit die same-day/adjacent-day-Logik auch im Preview greift. Ohne diese Parameter verhält sich Preview wie heute (kein Kontext = `always`-Modus).

## Risks / Trade-offs

- [Bestehende Diensttypen] Alle vorhandenen `duty_types` haben nach der Migration `applies_when='always'` und `consecutive_behavior='normal'` — heutiges Verhalten bleibt erhalten. Kein Datenverlust.
- [consecutive_behavior='reduced' ohne variant_id] Muss im Backend validiert werden: wenn `reduced` gesetzt, muss `consecutive_variant_id` vorhanden sein. Sonst 400-Fehler.
- [Preview ohne Kontext] Wenn Preview ohne `game_id` aufgerufen wird (z.B. Template-Konfigurationsseite), zeigt er alle Slots ohne Filterung. Das ist korrekt als "was wäre bei isoliertem Spiel".
