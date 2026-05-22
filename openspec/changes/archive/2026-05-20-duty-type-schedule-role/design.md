## Context

`duty_slots` hat ein optionales `game_id`-Feld. Slots mit `game_id` gehören zu einem Heimspiel; Slots ohne gehören zu keinem konkreten Spiel. Die Verbindung User → Team läuft über `members.user_id` (Spieler) und `family_links` (Elternteil → Kind → `team_memberships`). Heute ignoriert der Board-Handler beides.

## Goals / Non-Goals

**Goals:**
- `duty_types` trägt vier neue Verhaltensfelder: `same_day_behavior`, `same_day_variant_id`, `adjacent_day_behavior`, `adjacent_day_variant_id` (nicht gespeichert: `applies_when` wird berechnet)
- Preview und Regenerate **berechnen** `applies_when` und wenden beide Verhaltensweisen orthogonal an
- Admin-UI erlaubt Pflege der vier neuen Felder mit abhängiger Sichtbarkeit (variant-Felder nur bei `*_behavior='reduced'`)

**Non-Goals:**
- Keine Pagination (Saison hat überschaubar viele Heimspiele)
- Kein Eintragen für andere User (Claim bleibt immer für den eingeloggten User)
- Kein Ändern der Claim-Logik selbst

## Decisions

### 1. Vier Verhaltensfelder gehören auf `duty_types`, nicht auf `game_template_items`

Die Semantik gehört zum Diensttyp selbst. „Aufbau skip bei mehreren Spielen am Tag" oder „skip bei Folgetag" sind intrinsische Eigenschaften des Diensttyps, keine Eigenheit eines bestimmten Templates. Jedes Template, das Aufbau enthält, soll automatisch dieselben Regeln erhalten.

Alternative: Felder auf `game_template_items`. Nachteil: Bei mehreren Templates müsste man die Semantik duplizieren und könnte sie inkonsistent pflegen.

`applies_when` ist hingegen **nicht** eine Eigenschaft des Diensttyps, sondern eine Berechnung basierend auf Spielposition. Sie wird zur Laufzeit berechnet, nicht gespeichert.

### 2. `applies_when`: **berechnet**, nicht gespeichert

Beim Generieren von Slots wird `applies_when` für jeden Slot berechnet:

```
'day_open'   ← Spiel ist erstes am Tag UND kein Heimspiel am Vortag
'day_close'  ← Spiel ist letztes am Tag UND kein Heimspiel am Folgetag
'always'     ← alles andere (oder Spiel ist einziges am Tag)
```

**Grund:** Dadurch entfällt ein gespeichertes Feld, das mit der Spielplanlogik konsistent bleiben müsste. `applies_when` ergibt sich immer automatisch aus der Spielposition.

### 3. Zwei orthogonale Verhaltensweisen

**`same_day_behavior` + `same_day_variant_id`:** Wird angewendet, wenn mehrere Heimspiele am **gleichen Tag** existieren

```
'normal'   → kein Einfluss (Default)
'skip'     → Slot weglassen wenn mehrere Spiele am Tag
'reduced'  → anderen Diensttyp verwenden wenn mehrere Spiele am Tag
```

**`adjacent_day_behavior` + `adjacent_day_variant_id`:** Wird angewendet, wenn Heimspiele am **Vortag/Folgetag** existieren

```
'normal'   → kein Einfluss (Default)
'skip'     → Slot weglassen wenn adjacent day condition erfüllt
'reduced'  → anderen Diensttyp verwenden wenn adjacent day condition erfüllt
```

Adjacent day condition:
- Für `day_open`-Dienste: gibt es Heimspiele am Vortag?
- Für `day_close`-Dienste: gibt es Heimspiele am Folgetag?

**Beispiele:**
- Aufbau mit `same_day_behavior='skip'` → wenn 2+ Spiele am Tag, keinen Aufbau (weil beim 2. Spiel kein Aufbau nötig)
- Aufbau mit `same_day_behavior='normal'` + `adjacent_day_behavior='reduced'` → beim 1. Spiel normaler Aufbau, aber wenn Vortag Spiele, kleinen Aufbau statt normalem

### 4. Anwendungsreihenfolge der Verhaltensweisen

1. Prüfe `same_day_behavior`: Existieren mehrere Heimspiele am gleichen Tag?
2. Prüfe `adjacent_day_behavior`: Existieren Heimspiele am Vortag/Folgetag?
3. Kombiniere: Wenn einer `skip` sagt, wird der Slot übersprungen. Wenn beide `reduced` sind, wird `same_day_variant_id` bevorzugt (Primary variant).

## Risks / Trade-offs

- [Leere Dienstbörse] Wenn User keiner Mannschaft zugeordnet ist, gibt das Backend eine leere Liste zurück. Frontend zeigt Hinweistext.
- [Bestehende Diensttypen] Alle vorhandenen `duty_types` haben nach der Migration `same_day_behavior='normal'` und `adjacent_day_behavior='normal'` — heutiges Verhalten bleibt erhalten.
- [Validierung bei 'reduced'] Muss im Backend validiert werden: wenn `*_behavior='reduced'`, muss entsprechende `*_variant_id` vorhanden sein. Sonst 400-Fehler.
- [Preview ohne Kontext] Wenn Preview ohne `game_id` aufgerufen wird (z.B. Template-Konfigurationsseite), zeigt er alle Slots mit `applies_when='always'`. Das ist korrekt als "was wäre bei isoliertem Spiel".
