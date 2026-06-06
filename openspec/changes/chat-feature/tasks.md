## 1. Datenbank-Migration

- [x] 1.1 Migration `025_chat.up.sql` anlegen mit Tabellen: `conversations`, `conversation_members`, `messages`, `message_reads`, `broadcasts`, `broadcast_reads`
- [x] 1.2 Migration `025_chat.down.sql` anlegen (DROP TABLE in reverse order)
- [x] 1.3 `make migrate-up` lokal ausführen und Schema prüfen

## 2. Hub-Erweiterung

- [x] 2.1 `internal/hub/hub.go`: `userClients map[int]map[chan string]struct{}` hinzufügen
- [x] 2.2 `SubscribeUser(userID int) chan string` implementieren
- [x] 2.3 `UnsubscribeUser(userID int, ch chan string)` implementieren
- [x] 2.4 `BroadcastToUser(userID int, event string)` implementieren

## 3. Chat SSE-Endpoint

- [x] 3.1 `internal/chat/handler.go`: `ChatEvents(w, r)` Handler hinzufügen (userID aus JWT-Claims, `SubscribeUser`/`UnsubscribeUser`, 30s-Keepalive)
- [x] 3.2 In `cmd/teamwerk/main.go` Route `GET /api/chat/events` im authenticated-Block registrieren

## 4. Chat Backend (internal/chat/)

- [x] 4.1 `internal/chat/handler.go`: `Handler`-Struct mit `db *sql.DB`, `hub *hub.EventHub`, `cfg *config.Config` anlegen, `NewHandler` implementieren
- [x] 4.2 `GET /api/chat/users`: rollenbasierter User-Picker (spieler/elternteil=eigenes Team, trainer=ihre Teams, vorstand/admin=alle)
- [x] 4.3 `GET /api/chat/conversations`: Konversationsliste mit `unreadCount` und `lastMessage` pro User
- [x] 4.4 `POST /api/chat/conversations`: Direct (Deduplizierung) und Group erstellen, Sichtbarkeitsvalidierung
- [x] 4.5 `GET /api/chat/conversations/{id}/messages`: Letzte 100 Nachrichten, Mitgliedschaft prüfen
- [x] 4.6 `POST /api/chat/conversations/{id}/messages`: Nachricht speichern, SSE `BroadcastToUser` an alle aktiven Mitglieder, Push Notification async
- [x] 4.7 `POST /api/chat/conversations/{id}/read`: `message_reads` für alle ungelesenen Nachrichten der Konversation setzen
- [x] 4.8 `DELETE /api/chat/conversations/{id}/members/me`: `left_at` setzen, nur für `type=group`
- [x] 4.9 Alle Chat-Routen in `cmd/teamwerk/main.go` im authenticated-Block registrieren

## 5. Broadcast Backend

- [x] 5.1 `GET /api/chat/broadcasts`: Empfangene + gesendete Broadcasts für den User, `isRead`-Flag
- [x] 5.2 `POST /api/chat/broadcasts`: Broadcast speichern, Berechtigungsprüfung (trainer nur eigenes Team), Zielgruppe auflösen, SSE `BroadcastToUser` + Push async
- [x] 5.3 `POST /api/chat/broadcasts/{id}/read`: `broadcast_reads.read_at` setzen
- [x] 5.4 Broadcast-Routen in `cmd/teamwerk/main.go` registrieren (admin+vorstand+trainer per Middleware)

## 6. Frontend: Chat-Seite

- [x] 6.1 `web/src/pages/ChatPage.tsx` anlegen: Zwei-Spalten-Layout (Konversationsliste links, aktiver Chat rechts), mobile: ein Panel sichtbar
- [x] 6.2 Konversationsliste: Direct- und Gruppen-Konversationen auflisten, unread Badge pro Konversation, Neue Konversation Button
- [x] 6.3 Chat-View: Nachrichten anzeigen (älteste unten), Eingabefeld, Senden-Button, Gruppe-Verlassen-Button (nur bei Gruppen)
- [x] 6.4 Neues-Gespräch-Modal: Typ wählen (Direct/Gruppe), User-Picker mit Suche (`GET /api/chat/users`), Gruppen-Name-Feld
- [x] 6.5 `useChatEvents()` Hook: abonniert `GET /api/chat/events` via EventSource, triggert Reload bei `chat:new-message` und `chat:new-broadcast`
- [x] 6.6 Route `/chat` in `App.tsx` registrieren, Nav-Eintrag in `AppShell.tsx` mit Nav-Badge

## 7. Frontend: Broadcasts

- [x] 7.1 Broadcasts-Tab in `ChatPage.tsx`: empfangene Broadcasts auflisten, Sender sichtbar, keine Reply-Option
- [x] 7.2 Broadcast-Composer: Nur für admin/vorstand/trainer sichtbar, Zielgruppen-Auswahl (all/team/role), body-Feld
- [x] 7.3 Broadcast-Read beim Öffnen automatisch markieren (`POST /api/chat/broadcasts/{id}/read`)

## 8. Nav-Badge

- [x] 8.1 `AppShell.tsx`: Badge-Count aus `unreadCount`-Summe aller Konversationen + ungelesener Broadcasts
- [x] 8.2 Badge-Count bei SSE-Event `chat:new-message` oder `chat:new-broadcast` neu laden
- [x] 8.3 Badge verschwindet wenn Count = 0
