## Context

`DashboardPage.tsx` rendert vier `Accordion`-Sections (`termine`, `dienste`, `fahrt`, `team`) in einem `max-w-2xl`-Container. Jede Section ist eine Card (`bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`) mit Titel + lucide-Icon und `DashboardRow`-Einträgen (jeweils `<Link to>` mit Icon/Titel/Subtitle). Auf Mobile ist nur eine Section gleichzeitig offen (State in der Page). Der Chat-Unread-Count wird bereits in `AppShell.tsx` über `GET /api/chat/conversations` (`unreadCount` je Konversation) und `GET /api/chat/broadcasts` (`isRead`/`isSent`) ermittelt und via `useChatEvents` live gehalten.

## Goals / Non-Goals

**Goals:**
- Ungelesene Konversationen und Mitteilungen auf dem Dashboard sichtbar machen.
- Ein-Klick-Sprung in den richtigen Chat-Tab / die richtige Konversation.
- Optik und Verhalten deckungsgleich mit den bestehenden vier Sections.

**Non-Goals:**
- Keine Nachrichtenvorschau-Volltexte über das hinaus, was die Endpunkte liefern.
- Kein Antworten/Senden direkt vom Dashboard.
- Kein neuer Backend-Endpunkt, keine Aggregations-Route.

## Decisions

### D1: Wiederverwendung der bestehenden Chat-Endpunkte

Die Section lädt `GET /api/chat/conversations` und `GET /api/chat/broadcasts` (dieselben Calls wie `AppShell.loadChatUnread`) und filtert clientseitig auf ungelesen (`unreadCount > 0` bzw. `!isRead && !isSent`). Kein neuer Endpunkt — hält den Change frontend-only und vermeidet Backend-Test-/Broadcast-Gate-Aufwand.

**Alternative:** Ein aggregierter `GET /api/dashboard/nachrichten`-Endpunkt — verworfen, weil die vorhandenen Endpunkte genau die nötigen Felder liefern und ein weiterer Endpunkt Pflege-/Testkosten ohne Mehrwert erzeugt.

### D2: Begrenzung und Sortierung clientseitig

Anzeige der neuesten ungelesenen Einträge (Konversationen nach letzter Aktivität, Mitteilungen nach `sentAt`), gedeckelt auf max. 5 Einträge; darunter „Zum Chat →". Bei null Ungelesenem zeigt die Section einen dezenten Leerzustand („Keine ungelesenen Nachrichten") — konsistent mit den Leerzuständen der anderen Sections.

### D3: Ziel-Links

- Konversation → `/chat` (Tab „chats", ggf. mit Query zur Vorauswahl der Konversation, falls `ChatPage` das unterstützt; sonst nur `/chat`).
- Mitteilung → `/chat?tab=broadcasts` (derselbe Deeplink, den die Push-Notification nutzt).

### D4: Live-Update

Die Section abonniert `useChatEvents` (`chat:new-message`, `chat:new-broadcast`, `chat:conversation-read`) und lädt bei diesen Events neu — analog zu `AppShell`. So bleibt die Liste konsistent mit dem Sidebar-Badge.

## Risks / Trade-offs

- **Doppel-Fetch:** Dashboard und AppShell laden dieselben Endpunkte. Vernachlässigbar (kleine Payloads, gecacht durch kurze Nutzung); kein gemeinsamer Store nötig.
- **Deeplink-Vorauswahl:** Ob `ChatPage` eine Konversation per Query öffnet, ist zu prüfen; falls nicht vorhanden, ist der Link auf `/chat` ausreichend und kein Blocker.
