## Why

Die Dienstoptimierung (same-day / adjacent-day Reduktion) ist im Backend zwar implementiert, wird aber weder beim Anlegen von Spielen noch beim Regenerieren tatsächlich angewendet. Gleichzeitig ist der Regenerierungs-Workflow in SpieltagDetailPage durch eine falsche API-URL komplett defekt, und das gewählte Template wird nicht gespeichert – was ein nachträgliches Ändern und Neu-Anwenden unmöglich macht.

## What Changes

- **Migration 004**: `games.template_id INTEGER REFERENCES game_templates(id)` – speichert das beim Anlegen gewählte Template dauerhaft.
- **PreviewSlots-Endpunkt** erhält neuen Query-Parameter `date` (ISO-Date), damit die Optimierungslogik das neue Spiel selbst im Same-Day-Kontext berücksichtigt.
- **CreateGame**: speichert `template_id`, ruft `applyBehavior` mit korrektem Same-Day-Kontext auf (nur Heimspiele zählen).
- **GetGame**: gibt `template_id` im Response-Objekt zurück.
- **RegenerateGame**: akzeptiert `template_id` im Body (überschreibt gespeichertes); gibt HTTP 400 zurück wenn weder Body noch DB ein Template haben; ruft `applyBehavior` korrekt auf.
- **SpielplanPage**: übergibt `date` an Preview-Endpunkt und `template_id` an CreateGame.
- **SpieltagDetailPage**: ersetzt den defekten Regenerierungs-Aufruf durch einen Dialog mit Template-Dropdown, Live-Preview und "Anwenden"-Button.

## Capabilities

### New Capabilities

- `dienst-optimierung`: Dienstoptimierung (same_day_behavior, adjacent_day_behavior) wird beim Anlegen und Regenerieren von Events tatsächlich auf Duty-Slots angewendet. Nur Heimspiele zählen als Kontext.

### Modified Capabilities

- `games`: Template-Speicherung (`template_id`) in `games`-Tabelle; `GetGame` gibt `template_id` zurück; `RegenerateGame` erfordert Template-Angabe und wendet Optimierung an; `PreviewSlots` berücksichtigt `date`-Kontext für Optimierung.

## Impact

- **DB**: neue Spalte `games.template_id` (Migration 004, nullable, non-breaking)
- **Backend**: `internal/games/handler.go` – `PreviewSlots`, `CreateGame`, `GetGame`, `RegenerateGame`
- **Frontend**: `web/src/pages/SpielplanPage.tsx`, `web/src/pages/SpieltagDetailPage.tsx`
- **Keine neuen Abhängigkeiten** – `applyBehavior` und `loadSameDayContext` sind bereits vorhanden
