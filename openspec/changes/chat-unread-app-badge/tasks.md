## 1. Vorbereitung & Indizes

- [ ] 1.1 Verifizieren, dass Indizes auf `chat_read_state(user_id, conversation_id)` und `chat_broadcast_reads(user_id, broadcast_id)` existieren; falls nicht: Migration `01N_chat_unread_indexes.up.sql` mit den fehlenden Indizes anlegen
- [ ] 1.2 Die genauen Tabellen/Spalten für ungelesene 1:1-/Gruppen-Nachrichten und ungelesene Broadcasts aus den bestehenden List-Endpoints (`handler.go` Z. 163 + Z. 396) extrahieren und als SQL-Snippet im PR-Beschreibung dokumentieren

## 2. Backend: Chat-Unread-Helper

- [ ] 2.1 Funktion `chat.ComputeUnreadForUser(db *sql.DB, userID int) (int, error)` in `internal/chat/` neu anlegen (eigene Datei `unread.go`)
- [ ] 2.2 Implementierung: Summe `conversations.unreadCount` (replizierte Logik aus dem `GET /api/chat/conversations`-Endpoint) + Anzahl ungelesener, nicht selbst gesendeter Broadcasts
- [ ] 2.3 Unit-Test `TestComputeUnreadForUser`: User mit 0 Konversationen → 0; User mit 2 ungelesenen in Konv. A und 1 in Konv. B → 3; User mit 2 ungelesenen Broadcasts → +2; selbst gesendeter Broadcast zählt NICHT mit
- [ ] 2.4 Unit-Test: User der eine Konv. als gelesen markiert hat → 0 für diese Konv.

## 3. Backend: Push-Funktion mit Badge

- [ ] 3.1 In `internal/push/push.go`: neue Funktion `SendToUserWithBadge(db, cfg, userID int, title, body, url string, badge int)` parallel zur bestehenden `SendToUsers`
- [ ] 3.2 Payload um `badge` ergänzen: `json.Marshal(map[string]any{"title": title, "body": body, "url": url, "badge": badge})`
- [ ] 3.3 Bestehende `SendToUsers` unangetastet lassen (Games/Trainings/Duties-Pushes ändern sich nicht)
- [ ] 3.4 Test (Tabelle-Test) für die JSON-Payload: enthält die Felder `title`, `body`, `url`, `badge`; `badge` ist Integer (nicht String)

## 4. Backend: Chat-Push-Caller umstellen

- [ ] 4.1 Stellen finden: `grep -n "push.SendToUsers" internal/chat/handler.go`
- [ ] 4.2 Jeden Aufruf von `go push.SendToUsers(...)` in eine `for`-Schleife über die Empfänger umwandeln; pro Empfänger `ComputeUnreadForUser` + `SendToUserWithBadge`
- [ ] 4.3 Fehler von `ComputeUnreadForUser` loggen, aber Push trotzdem senden (mit `badge = 0` als Fallback — Badge wird beim nächsten App-Start im Page-Effect korrigiert)
- [ ] 4.4 Integration-Test `TestPostChatMessage_TriggersPushWithBadge`: zweite Konversation existiert mit 2 ungelesenen für Empfänger; nach POST der neuen Nachricht ist die erwartete Push-Payload für den Empfänger `badge = 3` (2 alte + 1 neue)
  - Mock/Spy auf `push.SendToUserWithBadge` (oder Refactor: dependency-injizierte Push-Funktion via Interface) damit testbar

## 5. Frontend: AppShell-Badge-Effekt

- [ ] 5.1 In `web/src/components/AppShell.tsx`: `useEffect(() => { ... }, [chatUnread])` ergänzen, der `navigator.setAppBadge(chatUnread)` ruft (bzw. `clearAppBadge()` bei 0)
- [ ] 5.2 Feature-Detection via `'setAppBadge' in navigator` vor jedem Call
- [ ] 5.3 Zweiter `useEffect` auf `user`: bei Logout `clearAppBadge` rufen
- [ ] 5.4 TypeScript: bei Build-Fehlern für `navigator.setAppBadge` ein lokales Typ-Alias `(navigator as Navigator & { setAppBadge?: ...; clearAppBadge?: ... })` verwenden

## 6. Service Worker: Push-Handler erweitern

- [ ] 6.1 In `web/src/sw.ts`: Payload-Type um `badge?: number` ergänzen
- [ ] 6.2 Push-Handler so erweitern, dass parallel zu `showNotification` (im selben `event.waitUntil`) `self.navigator.setAppBadge(badge)` bzw. `clearAppBadge()` gerufen wird — nur wenn `typeof data.badge === 'number'` UND `'setAppBadge' in self.navigator`
- [ ] 6.3 Achtung: Der bestehende Parameter `badge: '/icons/icon-192.png'` in `showNotification` bleibt — das ist das monochrome Notification-Icon, nicht der App-Badge

## 7. Manuelle Verifikation

- [ ] 7.1 Lokal: PWA in Chrome installieren (Desktop), Chat-Nachricht von zweitem Account senden, Badge auf Taskbar/Dock prüfen
- [ ] 7.2 Konversation lesen → Badge geht runter / verschwindet
- [ ] 7.3 Logout → Badge verschwindet sofort
- [ ] 7.4 Firefox-Test: keine Errors in der Konsole; App funktioniert ohne Badge
- [ ] 7.5 iOS-PWA (Safari 16.4+, zur Homescreen hinzugefügt, Push-Permission erteilt): Push kommt an, Badge erscheint auf Icon — verifiziert mit User-Gerät vor Deploy

## 8. Dokumentation

- [ ] 8.1 In `CLAUDE.md` unter "Push Notifications" Abschnitt um Hinweis erweitern: Chat-Pushes setzen App-Icon-Badge via Payload-Feld `badge`; andere Push-Caller können das Feld optional mitsenden
- [ ] 8.2 In OpenSpec archivieren (siehe Commit-Konventionen)
