## Why

Dokumentierter Folgeschritt aus Roadmap 8.3 (`test-coverage-roadmap`). `games.regenSingleDay`
(`internal/games/regen.go:113`, ~290 Zeilen) ist mit **gocognit 124** das komplexeste Konstrukt
des Codebase — mehr als doppelt so hoch wie `members.Import` vor dessen Zerlegung (60). Fünf
verschachtelte Verantwortlichkeiten in einer `for _, g := range dayGames`-Schleife
(Games laden → gelöschte Slots snapshotten → Custom-Slots snapshotten → Template-Items einfügen
mit Conflict-Branch → Notification-Intents ableiten), Verschachtelungstiefe 4–5.

Anders als bei `members.Import` liegt das Sicherheitsnetz bereits vor: die Auto-Duty-Regen ist
über die HTTP-Handler (Create/Update/DeleteGame) voll charakterisiert (adjacent-skip,
is_custom-Überleben, Zeit-Verschiebung, template_id=NULL, Konflikt, removed-Notification,
skipped/reduced-`regen_summary`). Der Refactor ist damit sicher durchführbar.

## What Changes

Verhaltenserhaltender Extract-Method-Refactor von `regenSingleDay` — **kein** beobachtbares
Verhalten, keine API-/Schema-/SSE-/`regen_summary`-Änderung. `regenSingleDay` bleibt als schlanker
Orchestrator (~40 Zeilen); die fünf Blöcke wandern in benannte Helfer:
- `loadDayGames` (Query + Scan der Tages-Spiele)
- `snapshotDeletedSlots` (zu löschende `is_custom=0`-Slots + Assignments)
- `snapshotCustomSlots` (`is_custom=1`-Slots für Konfliktdetektion)
- `regenGameItems` (innere Item-Schleife inkl. Conflict-Branch + Team-Iteration)
- `buildNotificationIntents` (Ableitung der Notification-Intents)

Ziel: `regenSingleDay` und die neuen Helfer unter die Rohschwellen (gocognit 20 / gocyclo 15);
`make metrics-gate` grün ohne Re-baseline (die Zerlegung erzeugt keine über-Schwelle-Helfer,
sofern jeder unter 20 bleibt — sonst wird der Ratchet mit Begründung angepasst).

## Capabilities

### New Capabilities

- `games-regen-refactor`: dokumentiert, dass die Auto-Duty-Regeneration durch Tests festgenagelt
  ist und `regenSingleDay` in benannte Einheiten unter der Komplexitätsschwelle zerlegt ist,
  ohne beobachtbares Verhalten zu ändern.

### Modified Capabilities

_(keine — die funktionale Regen-Logik behält ihre Requirements; reiner Struktur-Refactor.)_

## Impact

- **Code:** `internal/games/regen.go` — `regenSingleDay` in 5 Helfer zerlegt. Wörtlich erhalten:
  `regen_summary`-Struktur/-Feldnamen, Conflict-Detektion (`is_custom=1`-Schutz), `template_id
  IS NULL`-Semantik, same/adjacent-day-Behavior, Notification-Intents.
- **Tests:** keine neuen nötig (Netz auf main); Suite grün nach jedem Extract-Schritt.
- **Metriken:** `make metrics-gate` — `regenSingleDay` von gocognit 124 unter Schwelle.
- **Nachgelagert-Verweis:** schließt den in Roadmap 8.3 vertagten Refactor ab.
