## Why

Wer eine Mitteilung an den Verein schickt, erfährt nichts über ihren Verbleib. Chats
zeigen seit `chat-read-receipts` pro eigener Nachricht Häkchen und ein `N/M`-Aggregat —
Mitteilungen, bei denen die Frage „ist das angekommen?" ungleich wichtiger ist, zeigen
gar nichts. Ein Vorstand, der die neue Hallenordnung verschickt hat, hat keine Möglichkeit
zu sehen, ob sie zwölf oder hundertachtzig Leute gelesen haben.

Der Datenbestand liegt dabei vollständig vor:

```
   broadcast_reads
   ┌──────────────┬─────────┬─────────┬───────────┐
   │ broadcast_id │ user_id │ read_at │ hidden_at │
   └──────────────┴─────────┴─────────┴───────────┘
     PRIMARY KEY (broadcast_id, user_id)
                    ▲
        führende Spalte ⇒ Aggregat je Broadcast läuft über den PK-Index
```

`SendBroadcast` schreibt beim Fan-out eine Zeile je Empfänger, `MarkBroadcastRead` setzt
`read_at` — beides seit Migration `001`. Es fehlen ausschließlich Aggregat, Detail-Route
und Anzeige. Insbesondere braucht es **keine Migration**: anders als bei den Chat-Receipts,
für die `035` erst `idx_message_reads_message_id` nachliefern musste, ist der Primärschlüssel
hier bereits der passende Index.

`chat-read-receipts` nennt den Ausschluss ausdrücklich („und schließt Broadcasts bewusst
aus"); dieser Change löst das ein.

Als Beifang schließt er eine kleine Asymmetrie in der Tab-Leiste: „Mitteilungen" trägt
einen Ungelesen-Badge (`ChatPage.tsx:1213`), „Chats" nicht — obwohl die Zahl im selben
Component-State bereits liegt. Wer den Nav-Badge „5" sieht, kann heute nicht erkennen,
aus welchem der beiden Tabs die fünf stammen.

## What Changes

- **Aggregat in `GET /api/chat/broadcasts`.** Jede Mitteilung, die der Aufrufer selbst
  gesendet hat, trägt künftig `readCount` und `readTotal` (beide **ohne** den Absender).
  Für fremde Mitteilungen fehlen beide Felder — der Lese-Zustand Dritter bleibt für
  Empfänger unsichtbar, wie bei den Chat-Receipts.

- **Die Zahl ist die Anzeige, nicht das Häkchen.** Die Detailansicht einer eigenen
  Mitteilung zeigt `47 / 183 gelesen`. Ein Häkchen-Zustand wie im Direktchat (`readCount>0`
  → „gelesen") wäre bei 183 Empfängern irreführend und entfällt.

- **Neue Route `GET /api/chat/broadcasts/{id}/reads`** — Leserliste `[{userId, name, readAt}]`
  nach `readAt` aufsteigend, **nur für den Absender** (403 sonst, 404 bei unbekannter
  Mitteilung). Spiegelt `GET /api/chat/messages/{id}/reads` bis auf die Tabelle.

- **Live-Aktualisierung ohne Sturm.** `MarkBroadcastRead` sendet dem Absender
  `chat:broadcast-read:<broadcastId>`, **genau dann**, wenn das `UPDATE` tatsächlich eine
  Zeile getroffen hat (`read_at IS NULL`). Das Frontend erhöht `readCount` lokal um eins.
  Kein Coalescing nötig — ein Lesevorgang ist eine Mitteilung ist ein Event.

- **Badge am „Chats"-Tab** (Beifang): Summe der Konversations-Unreads, gleiche
  Darstellung wie der vorhandene Mitteilungs-Badge. Die Zahl steht in `ChatPage.tsx:1174`
  bereits im State; es kommt keine Query dazu.

- **`MessageReadsModal` wird generalisiert** statt kopiert: statt `messageId` nimmt es die
  zu ladende URL entgegen und dient beiden Fällen.

- **Keine Migration, kein Schemawechsel.** Bestehende Mitteilungen bekommen ihre Zähler
  rückwirkend, weil `broadcast_reads` seit jeher gefüllt wird.

## Nicht Teil dieses Changes

- **Keine Lesebestätigung für Bestands-Broadcasts mit `target_type='legacy'` gesondert
  behandeln.** Sie bekommen Zähler wie alle anderen; ihr Nenner ist die damals
  eingefrorene Empfängermenge und damit korrekt.
- **Kein Opt-out.** Weder Absender noch Empfänger können Lesebestätigungen abschalten —
  wie bei den Chat-Receipts. Die Abwägung steht in `design.md` §5.
- **Keine Erinnerung an Nichtleser.** Der Zähler informiert, er löst keine Nachfass-Push
  aus.
- **Der App-Icon-Badge bleibt eine einzige Summe.** Die Web Badging API kennt nur eine
  Zahl; die Aufteilung Chats/Mitteilungen findet ausschließlich in der Tab-Leiste statt.

## Capabilities

### Added Capabilities

- **`mitteilung-lesebestaetigung`** — Absender-Sicht auf den Lese-Zustand einer
  Mitteilung: Aggregat `readCount`/`readTotal` mit eingefrorenem Nenner, Detail-Liste
  absender-only, Live-Erhöhung per SSE bei genau dem ersten Lesevorgang je Empfänger.

### Modified Capabilities

- **`chat-broadcasts`** — `GET /api/chat/broadcasts` liefert zwei zusätzliche Felder für
  eigene Mitteilungen; `POST /api/chat/broadcasts/{id}/read` sendet zusätzlich ein Event
  an den Absender.
- **`chat-konversationen`** — Ungelesen-Badge am „Chats"-Tab (Beifang).

## Test-Anforderungen

| Route | Fall | Erwartung |
|---|---|---|
| `GET /api/chat/broadcasts` | Absender einer Mitteilung mit 3 von 10 Lesern | `readCount == 3`, `readTotal == 10` (Absender in keinem der beiden) |
| | Empfänger derselben Mitteilung | `readCount`/`readTotal` fehlen im JSON |
| | Absender, niemand hat gelesen | `readCount == 0`, `readTotal == 10` |
| `GET /api/chat/broadcasts/{id}/reads` | Absender | 200, Liste nach `readAt` aufsteigend, ohne den Absender |
| | Empfänger (nicht Absender) | 403 |
| | unbeteiligter User | 403 |
| | unbekannte ID | 404 |
| | unauthentifiziert | 401 |
| `POST /api/chat/broadcasts/{id}/read` | erster Aufruf eines Empfängers | 204, Absender erhält `chat:broadcast-read:<id>` |
| | zweiter Aufruf desselben Empfängers | 204, **kein** weiteres Event |
| | Absender markiert seine eigene Mitteilung | 204, **kein** Event an sich selbst |

**Garantierte Invarianten** (je ein eigener Test):

1. **Der Nenner ist eingefroren.** Nach dem Senden an 10 Empfänger und dem anschließenden
   Anlegen von 5 weiteren Nutzern bleibt `readTotal == 10`. Ausdrücklich anders als der
   Chat-Nenner, der „aktive Mitglieder" live zählt.
2. **Ein Leser zählt genau einmal.** Zweimaliges `POST …/read` desselben Users lässt
   `readCount` bei 1 und erzeugt genau ein SSE-Event. Der Test greift die Bedingung ab,
   an der ein Zähler sonst über die Zeit driftet.
3. **Der Absender ist in keiner Zahl.** Weder in `readCount` noch in `readTotal` noch in
   der Leserliste — obwohl `SendBroadcast` ihm eine Zeile mit gesetztem `read_at` schreibt.
4. **Lese-Zustand Dritter ist nicht abgreifbar.** Ein Empfänger, der `GET …/reads` für
   eine fremde Mitteilung aufruft, bekommt 403 — auch dann, wenn er selbst Empfänger ist
   und in der Liste stünde.
5. **Weggewischt bleibt gezählt.** Ein Empfänger, der die Mitteilung ohne Öffnen ausblendet
   (`hidden_at` gesetzt, `read_at` NULL), bleibt Teil von `readTotal` und erhöht
   `readCount` nicht — der Zähler erreicht in diesem Fall bewusst nie 100 %
   (`design.md` §3).
6. **Der Tab-Badge trennt sauber.** Vitest: bei 3 ungelesenen Konversations-Nachrichten
   und 2 ungelesenen Mitteilungen trägt der „Chats"-Tab `3`, der „Mitteilungen"-Tab `2`,
   und der App-Icon-Badge bleibt `5`.
