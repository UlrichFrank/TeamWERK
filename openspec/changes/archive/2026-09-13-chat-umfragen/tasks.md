## 1. Datenmodell

- [x] 1.1 Nächste freie Migrationsnummer prüfen (`ls internal/db/migrations/`, erwartet `059`) und `internal/db/migrations/059_chat_polls.up.sql` anlegen: `chat_polls`, `chat_poll_options`, `chat_poll_votes` + `idx_chat_poll_votes_user` gemäß design.md §1 (alle FKs `ON DELETE CASCADE`).
- [x] 1.2 `059_chat_polls.down.sql`: Tabellen in umgekehrter Reihenfolge droppen.
- [x] 1.3 `make migrate-up` lokal, `make migrate-down` + erneut `up` → beide Richtungen laufen sauber.

## 2. Backend — Lesepfad

- [x] 2.1 Neue Datei `internal/chat/polls.go`: Typen `pollView`/`pollOption`/`pollVoter` (JSON wie design.md §4) und `loadPolls(ctx, userID, msgIDs) (map[int]*pollView, error)` — zwei Queries (Umfragen+Optionen nach `position`, Stimmen mit Namen), `voterCount` = verschiedene Nutzer.
- [x] 2.2 `ListMessages`: Feld `Poll *pollView \`json:"poll"\`` am `Message`-Struct, nach dem Reaktions-Anhang `loadPolls` für alle **nicht gelöschten** Nachrichten der Seite aufrufen.
- [x] 2.3 `GET /api/chat/messages/{id}/poll` (`GetPoll`): 404 bei unbekannter/gelöschter Nachricht oder ohne `chat_polls`-Zeile, 403 wenn nicht `isMember`, sonst 200 mit `loadPolls`-Ergebnis.
- [x] 2.4 `LastMessage` um `IsPoll bool \`json:"isPoll"\`` erweitern; in `ListConversations` und `getConversation` per `EXISTS (SELECT 1 FROM chat_polls …)` auf die jüngste Nachricht befüllen.

## 3. Backend — Schreibpfad

- [x] 3.1 Fan-out-Teil von `SendMessage` (SSE `chat:new-message` + Push mit Unread-Badge, Direct-Rejoin bleibt in `SendMessage`) in den Helfer `broadcastNewMessage(r, convID, senderID, convType, convName, preview)` extrahieren; `SendMessage` nutzt ihn unverändert weiter (bestehende Chat-Tests grün).
- [x] 3.2 `POST /api/chat/conversations/{id}/polls` (`CreatePoll`): `isActiveMember` (403), `type='group'` (400), Validierung Frage 1–200 / Optionen 2–10 à 1–100 / keine Duplikate case-insensitive nach Trim (400); Insert `messages` + `chat_polls` + Optionen in **einer** Transaktion; danach `broadcastNewMessage` mit Preview `Umfrage: <Frage>`; 201 `{id}`.
- [x] 3.3 `PUT /api/chat/messages/{id}/poll/vote` (`VotePoll`): Existenz/gelöscht/Umfrage (404), `isActiveMember` (403), Option-IDs gehören zur Umfrage + keine Duplikate (400), Einfachauswahl ≤ 1 (400); in einer Transaktion `closed_at IS NULL` prüfen (409), eigene Stimmen der Umfrage löschen, neue einfügen; `chat:poll-updated:<convId>:<msgId>` per `BroadcastToUser` an alle aktiven Mitglieder; kein Push; 204.
- [x] 3.4 `POST /api/chat/messages/{id}/poll/close` (`ClosePoll`): 404 wie oben, 403 wenn `sender_id != claims.UserID` (kein Admin-Bypass), `UPDATE chat_polls SET closed_at = CURRENT_TIMESTAMP WHERE message_id = ? AND closed_at IS NULL`; Event nur bei tatsächlicher Änderung; 204.
- [x] 3.5 `EditMessage`: vor dem `UPDATE` auf `chat_polls`-Zeile prüfen → 409.
- [x] 3.6 Routen in `internal/app/router.go` im Authenticated-Block neben den übrigen `/api/chat/*`-Routen registrieren.

## 4. Backend — Tests

- [x] 4.1 `internal/chat/polls_test.go` mit allen Tests aus proposal.md „Test-Anforderungen" für `CreatePoll`, `VotePoll`, `ClosePoll`, `GetPoll` (Fixtures `newChatServer`, `createGroupConv`; Push über `SetPushFn` abfangen, Hub-Events über einen abonnierten User-Kanal prüfen).
- [x] 4.2 `TestEditMessage_Poll_409`, `TestListMessages_IncludesPoll`, `TestListMessages_DeletedPollHasNoPoll`, `TestListConversations_LastMessageIsPoll`.
- [x] 4.3 Invarianten-Test: parallele `VotePoll`-Requests desselben Nutzers auf eine Einfachauswahl-Umfrage hinterlassen genau eine Stimme.
- [x] 4.4 `go test ./internal/chat/... ./internal/arch/...` grün (Broadcast-Gate ohne neuen Allowlist-Eintrag, Push-Fan-out-Gate unverändert).

## 5. Frontend — Komponenten

- [x] 5.1 Typen in `ChatPage.tsx`: `Poll`/`PollOption`/`PollVoter`, `Message.poll: Poll | null`, `LastMessage.isPoll`.
- [x] 5.2 `web/src/components/ChatPollCreateModal.tsx`: Frage, 2 Start-Felder, „Option hinzufügen" bis 10, Entfernen (`X`) ab dem 3. Feld, Schalter „Mehrfachantworten erlauben", Hinweis „Stimmen sind für alle Mitglieder sichtbar"; Senden aktiv nur bei Frage + ≥ 2 verschiedenen nicht-leeren Optionen; `POST /chat/conversations/{id}/polls`; Klassen aus `buttonStyles.ts`, nur `brand-*`.
- [x] 5.3 `web/src/components/ChatPollCard.tsx`: Frage, Modus-/„Beendet"-Hinweis, Optionen mit Radio-/Checkbox-Icon (lucide), Zahl, Anteilsbalken (`count / voterCount`), Fußzeile „N Stimmen · Stimmen anzeigen"; Tippen berechnet die neue vollständige Auswahl (Einfach: ersetzen bzw. bei eigener Option leeren; Mehrfach: umschalten), optimistisches Update, `PUT …/poll/vote`, Rollback + Fehlermeldung bei Fehlschlag; beendet ⇒ nicht klickbar.
- [x] 5.4 `web/src/components/ChatPollVotesModal.tsx`: je Option Label + Namen der Abstimmenden.

## 6. Frontend — Einbau in ChatPage

- [x] 6.1 `MessageBubble`: bei `msg.poll` statt Text-Body die `ChatPollCard` rendern (Antwort-Zitat, Reaktionen, Zeitstempel, Read-Ticks bleiben).
- [x] 6.2 Composer: `BarChart3`-Button neben `Paperclip` mit `aria-label="Umfrage erstellen"`, nur bei `activeConv.type === "group"` und ohne Edit-Modus; öffnet `ChatPollCreateModal`.
- [x] 6.3 Kontextmenü (Desktop) und `MobileMessageActionOverlay`: „Bearbeiten" für Umfragen ausblenden; „Umfrage beenden" für den Ersteller bei offener Umfrage (`POST …/poll/close`).
- [x] 6.4 `useChatEvents`: `chat:poll-updated:<convId>:<msgId>` → bei geöffneter Konversation und Nachricht im State `GET /chat/messages/{id}/poll` laden und nur `poll` dieser Nachricht ersetzen (kein `loadMessages`, keine Scroll-Änderung).
- [x] 6.5 Konversationsliste: bei `lastMessage.isPoll` `BarChart3`-Icon vor der Frage.

## 7. Frontend — Tests

- [x] 7.1 `ChatPollCreateModal.test.tsx`: Senden deaktiviert bei < 2 bzw. doppelten Optionen, max. 10 Felder, leere Felder werden nicht gesendet, Request-Payload korrekt.
- [x] 7.2 `ChatPollCard.test.tsx`: Zahlen/Balken/eigene Wahl korrekt; Einfachauswahl-Wechsel sendet `{optionIds:[neu]}`, Antippen der eigenen Option sendet `[]`, Mehrfachauswahl schaltet um; beendete Umfrage nicht klickbar; Rollback bei Fehler.
- [x] 7.3 ChatPage: `chat:poll-updated` aktualisiert nur die betroffene Nachricht und löst keinen Listen-Reload aus; Umfrage-Button fehlt in Direktchats.

## 8. Dokumentation

- [x] 8.1 `docs/agent/06-gotchas.md`: Absatz „Chat-Umfragen" — Umfrage = `messages`-Zeile + `chat_polls` (kein `kind`), eigenes Event `chat:poll-updated` statt `chat:new-message` (Voll-Reload-Falle), `PUT` setzt vollständige Auswahl, Beenden ohne Admin-Bypass, Stimmen Ausgetretener bleiben.

## 9. Verifikation

- [x] 9.1 `/verify-change` — Build/Test/Lint + Projekt-Invarianten (Route→Tests, Broadcast, brand-Tokens, lucide, Migrationsnummer, `openspec validate`).
- [ ] 9.2 Manuell mit zwei Sessions in einer Gruppe: Umfrage anlegen (Push/Unread beim Gegenüber), abstimmen/ändern/zurückziehen live beobachten, Scroll-Position bleibt stehen, beenden, Bearbeiten nicht angeboten, Löschen zeigt Placeholder; Mobile-Breite (< 640 px) prüfen.
