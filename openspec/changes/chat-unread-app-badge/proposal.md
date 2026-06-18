## Why

Die installierte PWA zeigt heute keine ungelesenen Chat-Nachrichten auf dem App-Icon an. Nutzer müssen die App öffnen, um zu erkennen, dass etwas auf sie wartet — der zentrale Nutzen einer installierten App (Glanceability vom Homescreen aus) bleibt ungenutzt. Die nötigen Daten (`chatUnread` aus Konversations- und Broadcast-Unreads) sind im Frontend bereits live verfügbar, die Push-Infrastruktur ist vollständig implementiert. Es fehlt nur das Mapping auf die Web-Badging-API.

## What Changes

- **Frontend (Live, App offen):** `AppShell` setzt bei jeder Änderung von `chatUnread` per `navigator.setAppBadge(n)` den App-Icon-Badge; bei 0 oder Logout wird `clearAppBadge()` gerufen. Feature-Detection via `'setAppBadge' in navigator` — Firefox/alte Browser bleiben no-op.
- **Backend (Push):** `internal/push/push.go` erhält eine zweite Funktion `SendToUserWithBadge(db, cfg, userID, title, body, url, badge)`, die den Wert in die Payload aufnimmt (Feld `badge: number | null`).
- **Backend (Chat-Push-Caller):** Im `chat`-Handler wird beim Versand neuer Nachrichten/Broadcasts pro Empfänger der aktuelle Chat-Unread (Konversations-Summe + ungelesene Broadcasts) berechnet und im Push mitgegeben. Caller wechseln von `push.SendToUsers` auf den Per-User-Variant.
- **Service Worker (web/src/sw.ts):** Push-Handler liest `payload.badge`. Wenn gesetzt UND `'setAppBadge' in self.navigator`, parallel zu `showNotification` `self.navigator.setAppBadge(badge)` (bzw. `clearAppBadge()` bei 0) im `event.waitUntil` ausführen.
- **Hilfsfunktion:** Neue, exportierte Funktion `chat.ComputeUnreadForUser(db, userID) (int, error)` — identische Semantik wie der bestehende `loadChatUnread` im Frontend (Summe `unreadCount` über alle Conversations + Anzahl ungelesener, nicht selbst gesendeter Broadcasts). Wird vom Push-Caller und ggf. Tests genutzt.

### Bewusste Nicht-Änderungen

- Kein zusätzlicher Push beim Lesen einer Konversation (Multi-Device-Sync = Eventual Consistency, Bedingung 2 vom Nutzer).
- Broadcasts altern nicht — ein ungelesener Broadcast aus letztem Monat zählt weiter mit (Bedingung 3 vom Nutzer).
- Andere "wartende" Dinge (Mitgliedschaftsanfragen, Carpooling-Anfragen, offene Dienst-Slots) gehen NICHT in den Badge ein (Bedingung 1 vom Nutzer).

## Capabilities

### New Capabilities

- `chat-unread-app-badge`: App-Icon-Badge zeigt die Summe ungelesener Chat-Nachrichten und Broadcasts auf der installierten PWA.

## Impact

- `internal/push/push.go` — neue Funktion `SendToUserWithBadge`, Payload-Feld `badge`
- `internal/chat/handler.go` — Neue Helper `ComputeUnreadForUser`; Push-Versand-Stellen für neue Nachrichten/Broadcasts nutzen Per-User-Variant
- `web/src/components/AppShell.tsx` — `useEffect` auf `chatUnread`, Badge-Set/Clear
- `web/src/sw.ts` — Push-Handler liest `badge` aus Payload
- Kein Datenbankschema-Change, keine neue Migration
- Keine neuen Frontend-Dependencies
