## 1. Backend — DeleteConversation (Direkt-Chat)

- [x] 1.1 In `DeleteConversation` den Conversation-Typ vor dem Setzen von `left_at` abfragen (bereits vorhanden, aber Logik anpassen)
- [x] 1.2 Nach dem Setzen von `left_at`: Systemnachricht `"hat diesen Chat verlassen"` mit `is_system = 1` in `messages` einfügen
- [x] 1.3 SSE `chat:member-left:{convId}` an alle noch aktiven Mitglieder broadcasten (nach dem Einfügen der Systemnachricht)
- [x] 1.4 Dauerhafte Löschung nur wenn nach dem Setzen von `left_at` keine aktiven Mitglieder mehr übrig sind (`remaining == 0`)

## 2. Backend — createDirect (Re-join statt Duplikat)

- [x] 2.1 Die Such-Query in `createDirect` anpassen: B muss `left_at IS NULL` haben, für A gibt es keinen `left_at`-Filter
- [x] 2.2 Wenn bestehende Conversation gefunden: A's `left_at = NULL` setzen via UPDATE auf `conversation_members`
- [x] 2.3 SSE `chat:new-message:{convId}` an B broadcasten wenn B noch aktiv war (kein Re-join nötig) oder neu hinzugefügt wurde
- [x] 2.4 Wenn keine Conversation gefunden (beide hatten gelöscht): neuen Thread anlegen wie bisher, SSE an B

## 3. Backend — SendMessage (Auto-Re-join bei Direkt-Chat)

- [x] 3.1 In `SendMessage` nach dem Einfügen der Nachricht prüfen ob es ein Direkt-Chat ist (`SELECT type FROM conversations WHERE id = ?`)
- [x] 3.2 Wenn Direkt-Chat: alle Mitglieder mit `left_at IS NOT NULL` per UPDATE wiederherstellen (`left_at = NULL`)
- [x] 3.3 Sicherstellen dass `activeMembers` nach dem Re-join aufgerufen wird, damit das SSE auch A erreicht

## 4. Frontend — Generisches System-Nachrichten-Rendering

- [x] 4.1 In `ChatPage.tsx` den hardcodierten Text `hat die Gruppe verlassen` durch `{msg.body}` ersetzen (Zeile ~472)
