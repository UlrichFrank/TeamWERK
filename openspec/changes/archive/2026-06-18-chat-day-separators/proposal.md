## Why

Im Chat steht unter jeder Nachrichten-Bubble nur `HH:MM`. Wer im Verlauf nach oben scrollt, verliert komplett den Bezug zum Tag — eine Bubble um „09:10" könnte heute, gestern oder vor zwei Wochen sein. Das ist im Alltag bei Vereinskommunikation (Spielverlegungen, Trainingsabsagen) störend, weil der Zeitbezug oft entscheidend ist.

## What Changes

- **DaySeparator-Komponente:** Zwischen Nachrichten an unterschiedlichen Tagen erscheint ein horizontal zentrierter Trenner. Format: dünner Hairline-Divider links/rechts vom Datums-Label.
- **Label-Abstufung nach Distanz:**
  - 0 Tage zurück → `Heute`
  - 1 Tag zurück → `Gestern`
  - ≥ 2 Tage zurück → `Mittwoch, 15. April 2026` (Wochentag + volles Datum)
- **Einfüge-Logik:** Genau ein Separator vor jeder Nachricht, deren Datums-Schlüssel sich vom Vorgänger unterscheidet. Vor der ersten Nachricht der Liste ebenfalls. Der Separator zeigt das Datum der _neueren_ Nachricht (die darunter folgt). Gilt unabhängig vom Typ — auch über System-Messages (`isSystem`).
- **Reine Funktion `daySeparatorLabel(date, now): string`** in neuer Datei `web/src/lib/chatDateFormat.ts`. Wird sowohl von der Komponente als auch von Vitest-Tests konsumiert.
- **Bubble-Timestamp unverändert:** Bleibt `HH:MM` (`ChatPage.tsx:908, 1011`). Der Separator trägt das Datum, die Bubble nur die Uhrzeit.

### Bewusste Nicht-Änderungen

- Keine Backend-Änderung. Kein API-Touch. Keine Migration.
- Zeitzonen-Logik bleibt browser-lokal (`new Date(msg.sentAt)`); explizite Europe/Berlin-Behandlung ist nicht Teil dieses Changes.
- `MessageBubble.tsx:908` und `:1011` behalten ihr `toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })`. Kein zusätzliches Datum an der Bubble.
- Keine Datumssprünge in Konversations-Listen-Vorschau (Sidebar) — der Separator ist rein für den geöffneten Chat-Verlauf.

## Test-Anforderungen

- **`daySeparatorLabel`-Funktion** (Unit, Vitest):
  - `now = 2026-06-18`, `date = 2026-06-18 14:23` → `"Heute"`
  - `now = 2026-06-18`, `date = 2026-06-17 09:00` → `"Gestern"`
  - `now = 2026-06-18`, `date = 2026-06-16 14:00` → `"Dienstag, 16. Juni 2026"`
  - `now = 2026-06-18`, `date = 2025-12-24 14:00` → `"Mittwoch, 24. Dezember 2025"`
  - Tagesgrenze respektiert lokale Mitternacht — nicht 24h-Distanz: `now = 2026-06-18 00:30`, `date = 2026-06-17 23:30` → `"Gestern"` (nicht `"Heute"`)
- **Insertion-Invariante:** Niemals zwei Separatoren hintereinander; vor der ersten Nachricht immer einer; an gleichem Tag wie Vorgänger keiner. Wird über eine reine Hilfsfunktion `shouldRenderSeparator(prevMsg, currentMsg, now)` getestet, ohne DOM.

## Capabilities

### New Capabilities

- `chat-day-separators`: Chat-Verlauf zeigt Tageswechsel als zentrierten Trenner mit nach Distanz abgestuftem Label.

## Impact

- `web/src/lib/chatDateFormat.ts` — **NEU**: Pure Funktionen `daySeparatorLabel(date, now)` und `shouldRenderSeparator(prev, current, now)` plus interner Helper `dayKey(date)`.
- `web/src/pages/ChatPage.tsx` — Render-Loop (`:545`) ergänzt um Separator-Insertion. Tracking-Variable `lastDayKey` außerhalb des `.map(...)` (z.B. `messages.reduce` oder ein `for`-Loop, das Knoten in ein Array pusht).
- `web/src/components/DaySeparator.tsx` — **NEU**: Kleine Präsentationskomponente, Hairline + Label.
- `web/package.json` — Devdependency `vitest` + `@vitest/ui` falls noch nicht vorhanden. Falls Frontend bisher kein Test-Setup hat (zu verifizieren), Mini-Setup mit `vitest.config.ts`.
- Kein Datenbankschema-Change, keine neue Migration.
- Kein API-Endpoint-Change.
