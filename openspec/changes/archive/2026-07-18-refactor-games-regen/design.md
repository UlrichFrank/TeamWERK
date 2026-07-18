## Context

Folgeschritt zu Roadmap 8.3. `games.regenSingleDay` (`internal/games/regen.go:113`, gocognit 124)
wird verhaltenserhaltend zerlegt. Netz: die HTTP-Charakterisierungstests der Auto-Duty-Regen
(handler_test.go, scoped_events_test.go) liegen auf main.

## Goals / Non-Goals

**Goals:** `regenSingleDay` in benannte Helfer unter gocognit 20 / gocyclo 15 zerlegen, ohne
beobachtbares Verhalten (inkl. `regen_summary`) zu ändern; Suite nach jedem Schritt grün.

**Non-Goals:** keine Verhaltens-/API-/SSE-Änderung; keine neuen Regen-Features; keine
Änderung der `regen_summary`-Struktur.

## Decisions

**D1 — Extract-Reihenfolge (aus dem games-Scope-Bauplan), Suite grün nach jedem Schritt:**
1. `loadDayGames(tx, date, seasonID) ([]dayGame, error)` — Query + Scan (~119-146).
2. `snapshotDeletedSlots(tx, gameID) (map…, error)` — Loop ~192-222.
3. `snapshotCustomSlots(tx, gameID) (map…, error)` — Loop ~225-255.
4. `regenGameItems(tx, g, items, snapshots…) (outcomes, summaryDelta, error)` — innere
   Item-Schleife ~270-362 inkl. `insertOne`-Closure + Conflict-Branch + Team-Iteration.
5. `buildNotificationIntents(slotsByID, outcomes, eventName, date) []NotificationIntent` — ~364-399.
`regenSingleDay` bleibt schlanker Orchestrator.

**D2 — Wörtlich erhaltene Contracts:** `regen_summary`-Feldnamen/-Struktur (created/reduced/
skipped/conflicts/notified_users), Conflict-Key (duty_type_id/event_time/team), `template_id IS
NULL` → keine Auto-Dienste aber Löschung, same/adjacent-day-Behavior, Notification-Kinds
(removed/variant_changed).

**D3 — Metrics-Gate:** Ziel ist, jeden Helfer < gocognit 20 zu halten, sodass `make metrics-gate`
OHNE Re-baseline grün bleibt (Lehre aus Welle 3: Extract-Method kann die Anzahl über-Schwelle-
Funktionen erhöhen). Falls ein Helfer (voraussichtlich `regenGameItems`) trotzdem knapp über 20
landet, wird der Ratchet mit dokumentierter Begründung minimal angepasst — nicht still.

## Risks / Trade-offs

- **`regenGameItems` ist der Risiko-/Komplexitäts-Kern** (Conflict-Branch + Team-Iteration +
  Summary-Delta). Sorgfältige Signatur (Snapshots rein, outcomes+summaryDelta raus, kein
  verstecktes State-Sharing).
- **Transaktions-Semantik:** alle Helfer arbeiten auf demselben `*sql.Tx` — Reihenfolge der
  DELETE/INSERT-Operationen exakt erhalten.
- Netz ist HTTP-black-box; ein Extract, der die Query-/Insert-Reihenfolge subtil ändert, würde
  von den `regen_summary`-Inhaltstests gefangen.
