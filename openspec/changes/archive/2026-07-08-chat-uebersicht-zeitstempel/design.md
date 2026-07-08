## Context

Die Chat-Übersicht (`web/src/pages/ChatPage.tsx`, linke Liste) rendert pro Konversation nur `convName` + Unread-Badge (obere Zeile) und den Vorschautext der letzten Nachricht (untere Zeile). Ein Zeitstempel fehlt. Das Backend (`internal/chat/handler.go`, `ListConversations`) liefert `lastMessage.sentAt` als ISO-8601-String bereits mit und sortiert die Liste absteigend nach letzter Aktivität (`ORDER BY COALESCE(last sent_at, created_at) DESC`); das Frontend hält diese Reihenfolge, weil es bei jedem relevanten SSE-Event und beim eigenen Senden `loadConversations()` aufruft.

Für den Thread existiert bereits `web/src/lib/chatDateFormat.ts` mit `daySeparatorLabel` (`Heute`/`Gestern`/vollständiges Datum) und `shouldRenderSeparator`. Dessen Semantik passt aber nicht auf die Übersicht — dort ist das Messenger-Schema Uhrzeit/`Gestern`/Wochentag/Datum gewünscht.

## Goals / Non-Goals

**Goals:**
- Abstands-abhängiges Aktivitäts-Label pro Konversation in der Übersicht.
- Listeneintrags-Layout nach Messenger-Konvention (Label oben rechts, Unread-Badge unten rechts).
- Bestehende Sortier-Invariante per Test gegen Regression sichern.

**Non-Goals:**
- Keine Änderung an Sortierlogik, Backend-Query oder API-Shape.
- Kein Live-Update des Labels über Mitternacht ohne Reload (bewusst; siehe Risks).
- Keine Änderung an `daySeparatorLabel`/Thread-Formatierung.

## Decisions

**Neuer Helfer statt Erweiterung von `daySeparatorLabel`.** `conversationTimeLabel(date: Date, now: Date): string` als eigene, exportierte Funktion in `chatDateFormat.ts`. Grund: unterschiedliche Ausgabesemantik (Uhrzeit/Wochentag vs. „Heute"/Vollformat); ein gemeinsamer Helfer mit Modus-Flag würde beide Aufrufstellen verkomplizieren. Abstands-Berechnung wiederverwendet die vorhandene `dayKey`-Logik (Kalendertag-Differenz statt Millisekunden), damit `Gestern`/Wochentag an Kalendergrenzen korrekt kippt, nicht an 24-h-Fenstern.
- Formate über `toLocaleTimeString`/`toLocaleDateString` mit Locale `de-DE`: Uhrzeit `{hour:'2-digit',minute:'2-digit'}`, Wochentag `{weekday:'long'}`, Datum `{day:'2-digit',month:'2-digit',year:'2-digit'}`.
- `now` wird als Parameter übergeben (nicht intern `new Date()`), damit der Helfer deterministisch unit-testbar ist.

**Layout: WhatsApp-Muster.** Obere Zeile `flex justify-between`: Name (`truncate`, `flex-1`) links, Label (`shrink-0`, `text-xs text-brand-text-muted`) rechts. Untere Zeile ebenfalls `flex justify-between`: Vorschautext (`truncate`, `flex-1`) links, Unread-Badge (`shrink-0`) rechts. Das Badge wandert damit aus der Namenszeile in die Vorschauzeile.

**Sortierung bleibt unangetastet.** Wunsch „neueste oben" ist erfüllt; wir fügen nur einen Backend-Regressionstest (`ListConversations`, zwei Konversationen mit unterschiedlichem `sent_at` → Reihenfolge) und formalisieren die Invariante in der Spec.

## Risks / Trade-offs

- **Label kippt nicht live über Mitternacht** (ein „14:30" von gestern bleibt „14:30", bis die Liste neu lädt) → Mitigation: In der Praxis lädt jedes SSE-Event und jeder Seitenaufruf die Liste neu; entspricht dem Verhalten gängiger Messenger. Kein Timer nötig.
- **Wochentag-Kollision Vorwoche** (heute Mo, letzte Aktivität Mo vor 7 Tagen) → Mitigation: strikte Grenze `< 7` Kalendertage, danach Datum.
- **Zeitzone**: Kalendertag-Differenz über lokale Zeit des Browsers; `sentAt` ist UTC-ISO. `new Date(iso)` interpretiert korrekt in lokale Zeit → keine Sonderbehandlung nötig, konsistent mit bestehender Thread-Formatierung.
