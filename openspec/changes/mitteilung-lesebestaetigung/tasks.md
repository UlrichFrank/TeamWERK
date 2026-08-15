> **Voraussetzung:** `mitteilung-zielgruppen` muss abgeschlossen sein. Der Nenner dieser
> Anzeige ist die Empfängermenge des Fan-outs — solange eine Zielgruppe still auf null
> auflösen kann, wäre `47 / 183` eine Zahl, der man nicht trauen darf (design.md, Kopf).
>
> **Keine Migration.** `broadcast_reads.read_at` existiert seit `001`, der Primärschlüssel
> `(broadcast_id, user_id)` trägt das Aggregat. Anders als bei `chat-read-receipts`, wo
> Migration `035` erst einen Index nachliefern musste.

## 1. Backend: Aggregat in der Broadcast-Liste

- [ ] 1.1 `internal/chat/handler.go`, `ListBroadcasts`: `ReadCount`/`ReadTotal` als `*int` mit `json:"readCount,omitempty"` / `json:"readTotal,omitempty"` ins DTO. Zeiger statt `int`, damit fremde Mitteilungen die Felder gar nicht erst im JSON tragen.
- [ ] 1.2 Aggregat als abgeleitete Tabelle per `LEFT JOIN` anhängen (`GROUP BY broadcast_id` über `broadcast_reads`, gefiltert auf `user_id != b.sender_id`), nicht als zwei korrelierte Subqueries je Zeile — der PK ist mit `broadcast_id` als führender Spalte der passende Index (design.md §6).
- [ ] 1.3 Die beiden Felder **nur** befüllen, wenn `b.sender_id = claims.UserID`. Der Lese-Zustand Dritter darf nicht über einen unbeachteten Response-Rand abfließen.
- [ ] 1.4 Tests `internal/chat/broadcast_reads_test.go`: Absender sieht `3/10`; Empfänger sieht beide Felder **nicht**; niemand hat gelesen → `0/10`; Absender ist in keiner der beiden Zahlen, obwohl er eine Zeile mit `read_at` hat.
- [ ] 1.5 Test „Nenner ist eingefroren": nach dem Senden an 10 Empfänger fünf weitere User anlegen → `readTotal` bleibt 10. Grenzt die Semantik ausdrücklich gegen den live zählenden Chat-Nenner ab (design.md §2).
- [ ] 1.6 Test „weggewischt bleibt gezählt": Empfänger blendet ohne Öffnen aus (`hidden_at` gesetzt, `read_at` NULL) → bleibt in `readTotal`, erhöht `readCount` nicht (design.md §3).

## 2. Backend: Leserliste

- [ ] 2.1 `internal/chat/read_receipts.go`: `BroadcastReads` als Spiegel von `MessageReads` — Absender-Prüfung über `broadcasts.sender_id`, 403 für alle anderen, 404 bei unbekannter ID. Der bestehende `messageReader`-Typ wird wiederverwendet.
- [ ] 2.2 Query: `broadcast_reads` ⋈ `users`, `WHERE broadcast_id = ? AND user_id != ? AND read_at IS NOT NULL`, `ORDER BY read_at ASC`.
- [ ] 2.3 `internal/app/router.go`: `r.Get("/api/chat/broadcasts/{id}/reads", h.Chat.BroadcastReads)` in denselben Authenticated-Block wie die übrigen Broadcast-Routen (neben Z. 211-215).
- [ ] 2.4 Route-Tests: 200 für den Absender (Sortierung nach `readAt`, ohne ihn selbst), 403 für einen Empfänger derselben Mitteilung, 403 für einen Unbeteiligten, 404 für eine unbekannte ID, 401 unauthentifiziert.

## 3. Backend: SSE beim ersten Lesevorgang

- [ ] 3.1 `MarkBroadcastRead`: Rückgabewert des `UPDATE` auswerten (`res.RowsAffected()`), Absender-ID des Broadcasts laden.
- [ ] 3.2 Nur bei `RowsAffected() == 1` **und** `sender_id != claims.UserID`: `h.hub.BroadcastToUser(senderID, fmt.Sprintf("chat:broadcast-read:%d", broadcastID))`. Das bestehende `BroadcastToUser(claims.UserID, "chat:conversation-read")` bleibt unverändert.
- [ ] 3.3 **Kein Leser im Event.** Plain colon-string ohne JSON-Payload, konsistent zu den übrigen Chat-Events; das Frontend erhöht lokal um eins (design.md §4).
- [ ] 3.4 Tests: erster Aufruf → genau ein Event an den Absender; zweiter Aufruf desselben Users → 204 ohne weiteres Event; Absender markiert selbst → kein Event an sich. **Der zweite Fall ist der wichtige** — ohne die `RowsAffected`-Prüfung überholt `readCount` mit der Zeit `readTotal`, ohne dass ein Request fehlschlägt.

## 4. Frontend: Anzeige

- [ ] 4.1 `web/src/components/MessageReadsModal.tsx` generalisieren: statt `messageId: number` ein `url: string` (bzw. eine schmale Union) entgegennehmen, damit beide Fälle dieselbe Komponente nutzen. Aufrufer in `ChatPage.tsx:1882` mitziehen.
- [ ] 4.2 `ChatPage.tsx`, `BroadcastLite`-Interface um `readCount?: number` / `readTotal?: number` ergänzen.
- [ ] 4.3 In der Broadcast-Detailansicht (`ChatPage.tsx:1673 ff.`, im Block bei `activeBroadcast.isSent`) `N / M gelesen` als Button rendern, der das Modal öffnet. **Kein** `Check`/`CheckCheck`-Icon — die Zahl ist die Anzeige (design.md §1).
- [ ] 4.4 `useChatEvents`-Handler in `ChatPage.tsx` um `chat:broadcast-read:<id>` erweitern: `readCount` der betroffenen Mitteilung im State um eins erhöhen (Muster wie Z. 734 für Chat-Receipts).
- [ ] 4.5 Vitest: eigene Mitteilung zeigt `3 / 10 gelesen`; fremde Mitteilung zeigt nichts; SSE-Event erhöht die Anzeige auf `4 / 10`; Klick öffnet das Modal und lädt `/chat/broadcasts/{id}/reads`.

## 5. Beifang: Badge am Chats-Tab

- [ ] 5.1 `ChatPage.tsx:1201 ff.`: am Tab „Chats" denselben Badge rendern wie am Tab „Mitteilungen" (Z. 1213), gespeist aus `conversations.reduce((s, c) => s + c.unreadCount, 0)`. Die Reduktion steht in Z. 1174 bereits im Component — kein neuer Fetch, keine neue Ableitung.
- [ ] 5.2 Badge bei 0 nicht rendern (wie beim vorhandenen Mitteilungs-Badge).
- [ ] 5.3 Vitest: 3 ungelesene Konversations-Nachrichten + 2 ungelesene Mitteilungen → „Chats" trägt `3`, „Mitteilungen" trägt `2`; App-Icon-Badge bleibt `5`. Zweiter Fall: 0 Mitteilungen → am Mitteilungs-Tab kein Badge.

## 6. Verifikation

- [ ] 6.1 `openspec validate mitteilung-lesebestaetigung --strict`.
- [ ] 6.2 Beim Archivieren die Purpose-Zeile von `openspec/specs/chat-read-receipts/spec.md` nachziehen — sie sagt heute „und schließt Broadcasts bewusst aus", was mit diesem Change nicht mehr stimmt. Verweis auf `mitteilung-lesebestaetigung` setzen.
- [ ] 6.3 `/verify-change` — Build/Test/Lint plus Projekt-Invarianten (Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons).
- [ ] 6.4 Neue Route in der `broadcastAllowlist` prüfen: `GET` ist vom Broadcast-Gate ohnehin nicht betroffen, `POST …/read` broadcastet bereits — es ist also kein Allowlist-Eintrag nötig. Vor dem Commit gegenprüfen, dass `internal/arch/broadcast_test.go` grün ist.
- [ ] 6.5 Ein Commit pro Task-Gruppe, Conventional Commits mit Scope `chat`.
