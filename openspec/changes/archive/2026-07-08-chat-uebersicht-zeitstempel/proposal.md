## Why

In der Chat-Übersicht (`/chat`, linke Konversationsliste) wird pro Konversation nur der Text der letzten Nachricht gezeigt — **kein Zeitstempel**. Damit fehlt der zeitliche Kontext, den man aus jedem Messenger kennt: Auf einen Blick erkennen, wie frisch eine Unterhaltung ist. Die dafür nötigen Daten (`lastMessage.sentAt`) liefert das Backend bereits mit, sie werden nur nicht gerendert.

## What Changes

- Jeder Eintrag der Konversationsliste zeigt oben rechts ein **Aktivitäts-Label** auf Basis von `lastMessage.sentAt`, dessen Format sich nach dem zeitlichen Abstand richtet:
  - heute → Uhrzeit (`14:30`)
  - gestern → `Gestern`
  - 2–6 Tage her → Wochentag (`Montag`)
  - ≥ 7 Tage her → Datum (`08.07.26`)
- **Layout-Anpassung** des Listeneintrags nach Messenger-Konvention: Aktivitäts-Label oben rechts (auf Höhe des Namens), Unread-Badge wandert nach unten rechts (auf Höhe der Nachrichtenvorschau).
- Konversationen ohne Nachricht (`lastMessage == null`) zeigen **kein** Label.
- Neuer, testbarer Frontend-Helfer `conversationTimeLabel(date, now)` in `web/src/lib/chatDateFormat.ts` (Geschwister zu `daySeparatorLabel`).

**Nicht Teil dieses Changes (bereits vorhanden):** Die Sortierung der Liste nach letzter Aktivität (neueste oben) ist im Bestand vollständig umgesetzt — Backend `ORDER BY COALESCE(letzte sent_at, created_at) DESC` und Frontend-Reload (`loadConversations()`) bei jedem relevanten SSE-Event sowie beim eigenen Senden. Dieser Change fügt dafür lediglich einen **Regressionstest** hinzu, um das Verhalten gegen Regression abzusichern.

## Capabilities

### New Capabilities

- `chat-konversationsliste-zeitstempel`: Abstands-abhängige Anzeige der letzten Aktivität pro Konversation in der Übersichtsliste (Uhrzeit / „Gestern" / Wochentag / Datum) inkl. Listeneintrags-Layout (Label oben rechts, Unread-Badge unten rechts).

### Modified Capabilities

- `chat-konversationen`: Sichert die bestehende Sortier-Invariante der Liste (neueste Aktivität oben) explizit als getestete Anforderung. Kein Verhaltenswechsel — nur Formalisierung + Regressionstest.

## Impact

- **Frontend:**
  - `web/src/lib/chatDateFormat.ts` — neuer Helfer `conversationTimeLabel(date, now)`.
  - `web/src/pages/ChatPage.tsx` (Listeneintrag ~Z. 662–695) — Label rendern, Unread-Badge in die untere Zeile verschieben.
  - Neuer Vitest für den Helfer (`chatDateFormat`-Tests).
- **Backend:** Keine Code-Änderung. Optionaler Regressionstest für die Sortierung in `internal/chat/handler_test.go` (`ListConversations`).
- **Keine** Migration, **keine** neue Route, **kein** neuer Broadcast, **keine** neuen Dependencies.
