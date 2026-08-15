# Design — Lesebestätigung für Mitteilungen

> **Voraussetzung:** `mitteilung-zielgruppen` muss abgeschlossen sein. Der Nenner dieser
> Anzeige ist die Empfängermenge des Fan-outs; solange eine Zielgruppe still auf null
> auflösen kann (der Bug, den jener Change behebt), wäre `47 / 183` eine Zahl, der man
> nicht trauen darf.

## 1. Warum die Zahl und nicht das Häkchen

Die Chat-Receipts kollabieren im Direktchat bewusst auf zwei Zustände: gesendet / gelesen.
Das trägt bei einem Gegenüber. Bei einer Mitteilung an 183 Leute wäre „gelesen, weil einer
gelesen hat" schlicht falsch verstanden.

```
   Chat (direkt)          Chat (Gruppe)          Mitteilung
   ┌──────────┐           ┌──────────┐           ┌────────────────────┐
   │  ✓✓      │           │  ✓✓ 4/7  │           │  47 / 183 gelesen  │
   └──────────┘           └──────────┘           └────────────────────┘
   binär                  Zahl + Haken            nur Zahl
```

**Entscheidung:** kein `read`-Boolean im Broadcast-DTO, kein Häkchen-Icon. Nur
`readCount`/`readTotal`, gerendert als `N / M gelesen`, klickbar für die Detailliste.

**Verworfen:** ein Häkchen bei 100 %. Reizvoll, aber der Zustand wird praktisch nie
erreicht (§3) — ein Indikator, der nie anspringt, erzieht dazu, ihn zu ignorieren.

## 2. Der eingefrorene Nenner — der Punkt, an dem Broadcasts es besser haben

Im Chat ist `readTotal` „aktive Mitglieder außer Sender", also ein **Live**-Wert: tritt
jemand der Gruppe bei, wächst der Nenner rückwirkend für alte Nachrichten. Bei
Mitteilungen ist der Nenner die Zeilenzahl in `broadcast_reads`, festgeschrieben beim
Fan-out.

```
   Chat                                  Mitteilung
   ────                                  ──────────
   readTotal = COUNT(aktive Mitglieder)  readTotal = COUNT(broadcast_reads)
               ▲ ändert sich später                  ▲ Snapshot vom Sendezeitpunkt

   "4/7" kann morgen "4/9" sein          "47/183" bleibt für immer 47/183
```

Das ist konzeptuell sauberer, und es ergibt sich hier von selbst — der Fan-out **ist** die
Empfängerliste. Kein Zusatzaufwand, aber ein bewusst festgehaltener Unterschied: wer die
beiden Anzeigen nebeneinander sieht, soll wissen, dass sie unterschiedlich altern.

## 3. `hidden_at`: der Zähler erreicht nie 100 %

`DeleteBroadcast` blendet eine Mitteilung pro Nutzer aus (`hidden_at`), statt sie zu
löschen; erst wenn alle sie ausgeblendet haben, verschwindet die `broadcasts`-Zeile. Ein
Empfänger kann also wegwischen, **ohne** je `read_at` gesetzt zu haben.

Drei Möglichkeiten:

| Option | `readTotal` | Effekt |
|---|---|---|
| Ausgeblendete aus dem Nenner nehmen | schrumpft | Nenner sinkt nachträglich — der Snapshot aus §2 wäre keiner mehr |
| Ausblenden als „gelesen" werten | konstant | Zähler lügt: wegwischen ist nicht lesen |
| **Ausgeblendete bleiben im Nenner, ungelesen** | konstant | Zähler stagniert bei < 100 % |

**Entscheidung:** die dritte. Sie ist die einzige, bei der beide Zahlen weiter bedeuten,
was sie sagen — „von 183 Adressierten haben 47 geöffnet". Der Preis ist, dass 183/183
praktisch nie eintritt.

Das ist auch der Grund, warum §1 kein 100-%-Häkchen vorsieht: die beiden Entscheidungen
hängen zusammen.

## 4. Das SSE-Event und die Bedingung, an der Zähler sonst driften

Die Chat-Receipts brauchen den `upToMessageId`-Trick, weil ein einziges `MarkRead` bis zu
hunderte Nachrichten auf einmal markiert und sonst einen Event-Sturm auslösen würde. Hier
gibt es das Problem nicht: eine Mitteilung wird als Ganzes gelesen.

```
   MarkBroadcastRead(broadcastID, reader)
        │
        ├─ UPDATE broadcast_reads SET read_at = CURRENT_TIMESTAMP
        │  WHERE broadcast_id = ? AND user_id = ? AND read_at IS NULL
        │                                          ▲
        │                              schon vorhanden ⇒ 0 Zeilen
        ├─ RowsAffected() == 1 ?
        │        ├─ ja   → BroadcastToUser(sender, "chat:broadcast-read:<id>")
        │        └─ nein → nichts (idempotent)
        │
        └─ BroadcastToUser(reader, "chat:conversation-read")   [wie bisher]
```

**Die `RowsAffected`-Prüfung ist nicht optional.** Ohne sie feuert jedes wiederholte
Öffnen derselben Mitteilung ein weiteres Event, das Frontend zählt jedes Mal `+1` — und
`readCount` überholt `readTotal`, ohne dass irgendein Request fehlschlägt. Genau die
Klasse stiller Fehler, die schon die Zielgruppen-Auflösung erwischt hat. Invariante 2 im
Proposal testet sie.

**Kein Payload über den Leser.** Das Event trägt nur die Broadcast-ID; das Frontend erhöht
lokal um eins. Der Absender dürfte die Identität zwar sehen (er darf ja die Detailliste
abrufen), aber sie wird für die Zahl nicht gebraucht, und ein offenes Modal kann
nachladen. Chat-Events sind ohnehin plain colon-strings ohne JSON-Payload — das Format
bleibt gewahrt.

## 5. Warum kein Opt-out — und wo die Grenze liegt

Eine Leseliste über 183 Personen ist etwas anderes als ein Häkchen im Zweiergespräch. Sie
macht aus einer Ansage ein Anwesenheitsprotokoll: „wer hat die Hallenordnung nicht
gelesen" ist beantwortbar, und zwar namentlich.

Drei Dinge begrenzen das, ohne dass es einen Schalter braucht:

1. **Absender-only.** Wie bei den Chat-Receipts (403 für alle anderen). Empfänger
   erfahren nichts über andere Empfänger.
2. **Der Absenderkreis ist bereits der engste.** Nach `mitteilung-zielgruppen` darf nur
   `admin | vorstand | sportliche_leitung` senden. Die früher erwogene Unterscheidung
   „Zahl für alle, Namen nur für Vorstand" trennt damit keine zwei Mengen mehr — jeder
   Absender ist vorstand-like.
3. **Es gibt keine Nachfass-Funktion.** Der Zähler informiert; er bietet keinen Knopf
   „alle Nichtleser anstupsen". Ohne diesen Knopf bleibt die Information beschreibend
   statt drängend.

**Bewusst nicht gelöst:** ein Empfänger kann nicht verhindern, dass sein Öffnen sichtbar
wird — außer indem er nicht öffnet. Das ist dieselbe Modellgrenze, die die Chat-Receipts
schon haben, hier nur mit größerem Publikum. Falls das später stört, ist ein Opt-out
additiv nachrüstbar (`read_at` weiter setzen, Sichtbarkeit filtern).

## 6. Aggregat-Query: ein Join, nicht 100 Subqueries

`ListBroadcasts` liefert bis zu 100 Zeilen. Zwei korrelierte Subqueries pro Zeile wären
bei der Größe unkritisch, aber unnötig — der Primärschlüssel `(broadcast_id, user_id)`
hat `broadcast_id` als führende Spalte, ein `GROUP BY` darüber ist indexgestützt.

**Entscheidung:** eine abgeleitete Tabelle, per `LEFT JOIN` angehängt, gefiltert auf
`user_id != b.sender_id`. Die Felder werden nur befüllt, wenn `b.sender_id = claims.UserID`
— fremde Mitteilungen tragen sie gar nicht erst im JSON (`omitempty` auf beiden Zeigern),
damit der Lese-Zustand Dritter nicht über einen unbeachteten Response-Rand abfließt.

**Keine Migration.** Der Kontrast zu `chat-read-receipts` ist der Erwähnung wert: dort
musste Migration `035` erst `idx_message_reads_message_id` nachliefern, weil
`message_reads` nach `user_id` indiziert war und das Aggregat je Nachricht ins Leere lief.
Hier ist der passende Index seit `001` der Primärschlüssel selbst.

## 7. Der Tab-Badge als Beifang

Er hängt nicht inhaltlich an den Lesebestätigungen, sondern an derselben Stelle im Code
und derselben Frage des Nutzers („was von den 5 ist was?"). Ihn als eigenen Change zu
führen, hieße zwei Mal dieselbe Datei anzufassen.

```
   heute                              nachher
   ┌──────────┬───────────────┐      ┌────────────┬───────────────┐
   │ 💬 Chats │ 📢 Mitteil. ② │      │ 💬 Chats ③ │ 📢 Mitteil. ② │
   └──────────┴───────────────┘      └────────────┴───────────────┘
     ▲ Zahl liegt in ChatPage.tsx:1174 bereits im State
```

Kein Backend, keine Query, keine neue Ableitung — `conversations.reduce(…)` steht dort
schon, nur wird das Ergebnis heute ausschließlich summiert.

**Der App-Icon-Badge bleibt die Summe.** Die Web Badging API nimmt eine Zahl entgegen;
eine Aufteilung ist dort technisch nicht darstellbar und wird nicht versucht.
