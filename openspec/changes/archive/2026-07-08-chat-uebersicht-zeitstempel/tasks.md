## 1. Zeitformat-Helfer (Frontend)

- [x] 1.1 In `web/src/lib/chatDateFormat.ts` die Funktion `conversationTimeLabel(date: Date, now: Date): string` ergänzen — Buckets: heute→`HH:MM` (`de-DE`), gestern→`Gestern`, 2–6 Kalendertage→Wochentag (`weekday:'long'`), ≥7→`TT.MM.JJ`. Kalendertag-Differenz über die bestehende `dayKey`-Logik, strikte Grenze `< 7`.
- [x] 1.2 Vitest für `conversationTimeLabel` (`web/src/lib/chatDateFormat.test.ts` oder bestehende Testdatei erweitern): je ein Fall für heute, gestern, 3 Tage (Wochentag), genau 7 Tage (→ Datum, nicht Wochentag). `now` fix injizieren.

## 2. Listeneintrags-Layout (Frontend)

- [x] 2.1 In `web/src/pages/ChatPage.tsx` (Konversations-Listeneintrag ~Z. 662–695) das Aktivitäts-Label rendern: obere Zeile `flex justify-between` mit Name links (`truncate`, `flex-1`) und `conversationTimeLabel(new Date(conv.lastMessage.sentAt), new Date())` rechts (`shrink-0 text-xs text-brand-text-muted`). Label nur bei vorhandener `lastMessage`.
- [x] 2.2 Unread-Badge aus der Namenszeile in die Vorschauzeile verschieben: untere Zeile `flex justify-between`, Vorschautext links (`truncate`, `flex-1`), Badge rechts (`shrink-0`). Bestehende `brand-*`-Token/Icon-Konventionen beibehalten.

## 3. Sortier-Invariante absichern (Backend)

- [x] 3.1 Regressionstest in `internal/chat/handler_test.go` für `ListConversations`: zwei Konversationen mit Nachrichten unterschiedlicher `sent_at` anlegen → prüfen, dass die zuletzt aktive an Index 0 steht; Konversation ohne Nachricht wird per `created_at` einsortiert.

## 4. Verifikation

- [x] 4.1 `pnpm -C web test` (Vitest grün), `pnpm -C web build`, `pnpm -C web lint`.
- [x] 4.2 `go test ./internal/chat/...` grün.
- [x] 4.3 `/verify-change` durchlaufen (brand-Tokens, lucide-Icons, Build/Test/Lint, `openspec validate`).

## Test-Anforderungen

- `conversationTimeLabel` heute → Ausgabe `HH:MM`; **Invariante:** heutige Aktivität zeigt Uhrzeit.
- `conversationTimeLabel` gestern → Ausgabe `Gestern`.
- `conversationTimeLabel` Abstand 2–6 Tage → ausgeschriebener Wochentag.
- `conversationTimeLabel` Abstand = 7 Tage → numerisches Datum `TT.MM.JJ` (nicht Wochentag); **Invariante:** keine Wochentag-Kollision mit der Vorwoche.
- `ListConversations` (Backend) → zuletzt aktive Konversation an Index 0; **Invariante:** Liste absteigend nach letzter Aktivität, leere Konversation nach `created_at`.
