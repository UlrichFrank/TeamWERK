## 1. Datenbank-Migration

- [x] 1.1 Migration `026_chat_hidden.up.sql` erstellen: `ALTER TABLE broadcast_reads ADD COLUMN hidden_at DATETIME`
- [x] 1.2 Migration `026_chat_hidden.down.sql` erstellen (SQLite unterstützt kein DROP COLUMN — Workaround via Tabellen-Rebuild oder No-Op dokumentieren)

## 2. Backend: Gespräch löschen

- [x] 2.1 `DELETE /api/chat/conversations/{id}` Endpoint in `handler.go` implementieren: setzt `left_at = CURRENT_TIMESTAMP` für den aktuellen User in `conversation_members`
- [x] 2.2 Cleanup-Logik nach `left_at`-Update: wenn `COUNT(*) WHERE left_at IS NULL = 0` → `DELETE FROM conversations WHERE id = ?` (CASCADE via Foreign Keys)
- [x] 2.3 Endpoint in `main.go`/Router registrieren

## 3. Backend: Broadcast löschen

- [x] 3.1 `DELETE /api/chat/broadcasts/{id}` Endpoint implementieren: setzt `broadcast_reads.hidden_at = CURRENT_TIMESTAMP` für den aktuellen User; 403 wenn kein `broadcast_reads`-Eintrag existiert
- [x] 3.2 Cleanup-Logik: wenn `COUNT(*) WHERE hidden_at IS NULL = 0` → `DELETE FROM broadcasts WHERE id = ?`
- [x] 3.3 `ListBroadcasts`-Query um `AND br.hidden_at IS NULL` ergänzen
- [x] 3.4 Endpoint in Router registrieren

## 4. Backend: Mitglied hinzufügen

- [x] 4.1 `POST /api/chat/conversations/{id}/members` Endpoint implementieren: prüft Ersteller-Berechtigung, `canContactUser`, Typ = group; führt UPSERT (UPDATE left_at=NULL oder INSERT) aus
- [x] 4.2 SSE-Event `chat:new-message:{convId}` an neu hinzugefügten User senden
- [x] 4.3 Endpoint in Router registrieren

## 5. Frontend: Bug-Fix Mobile Broadcast-Detail

- [x] 5.1 In `openBroadcast()` (`ChatPage.tsx`) `setMobileShowChat(true)` ergänzen

## 6. Frontend: Löschen-UI

- [x] 6.1 Löschen-Aktion im Chat-Listen-Item: `ActionMenu` (⋮) mit "Gespräch löschen"-Eintrag + `window.confirm()` + `api.delete()`-Call + `loadConversations()` danach
- [x] 6.2 Löschen-Aktion im Broadcast-Listen-Item: analog mit `api.delete('/chat/broadcasts/{id}')` + `loadBroadcasts()`
- [x] 6.3 Wenn das aktive Gespräch/der aktive Broadcast gelöscht wird: `setActiveConv(null)` / `setActiveBroadcast(null)` + `setMobileShowChat(false)`

## 7. Frontend: Mitglieder hinzufügen

- [x] 7.1 `UserPlus`-Icon im Gruppen-Chat-Header ergänzen (nur sichtbar wenn `activeConv.createdBy === user?.id`)
- [x] 7.2 `AddMemberModal`-Komponente erstellen: Suchbox (`GET /api/chat/users?q=`), User-Liste, Einzel-Auswahl, "Hinzufügen"-Button (`POST /api/chat/conversations/{id}/members`)
- [x] 7.3 Nach erfolgreichem Hinzufügen: `loadConversations()` + `activeConv` neu laden (Mitgliederanzahl aktualisieren)
