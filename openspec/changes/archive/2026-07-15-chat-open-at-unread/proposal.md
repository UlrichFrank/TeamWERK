## Why

Der aktuelle Zustand (nach `chat-image-dimensions`) scrollt beim Öffnen einer
Konversation **immer** ans Ende. Wer aus dem Urlaub kommt oder eine aktive
Gruppe nach Stunden wieder öffnet, wird an den 50 ungelesenen Nachrichten
vorbeigeschleudert und muss von Hand hochscrollen, um den Anfang des neuen
Chatverlaufs zu finden. Serverseitig ist alles Nötige längst da — jede
Konversation kennt ihren `unreadCount`, die `message_reads`-Tabelle trackt
per Nachricht, was gelesen wurde. Wir positionieren nur an der falschen
Stelle.

Ziel: **wie WhatsApp/Slack** — beim Öffnen landet man am ersten ungelesenen
Eintrag, mit einer sichtbaren „N ungelesene Nachrichten"-Grenze. Nur wenn
nichts ungelesen ist, geht es ans Ende (heutiges Verhalten).

## What Changes

- `openConversation` scrollt nicht mehr unbedingt ans Ende, sondern berechnet
  aus `conv.unreadCount` und `messages.length` eine Ziel-Position:
  - `unreadCount === 0` → ans Ende (wie heute)
  - `0 < unreadCount ≤ messages.length` → an den `UnreadDivider` vor der
    ersten ungelesenen Nachricht (`scrollIntoView({block: 'start'})`)
  - `unreadCount > messages.length` (alles Geladene ungelesen, mehr davor)
    → an den obersten geladenen Eintrag, plus sichtbarer „N weitere
    ungelesene älter"-Chip
- Neuer `UnreadDivider`-Komponente (Muster wie `DaySeparator` in
  `ChatPage.tsx:967`): „N ungelesene Nachrichten" als visuelle Trennlinie
  zwischen dem zuletzt gelesenen und dem ersten ungelesenen Eintrag.
- Neuer „N weitere ungelesene älter — 'Ältere laden' klicken"-Chip oberhalb
  der `WindowedRows`, sichtbar nur wenn der Divider vor der geladenen Seite
  läge.
- `?openUser=<id>`-Deep-Link erbt das Verhalten automatisch (läuft schon
  über `openConversation` seit `chat-image-dimensions`).
- Sticky-Scroll-Verhalten während man in der Konversation ist bleibt
  unverändert — `forceScrollToEndRef` wird weiterhin bei Senden gesetzt
  und beim Öffnen einer bereits vollständig gelesenen Konversation.

Kein BREAKING — Konversationen ohne Ungelesenes verhalten sich exakt wie
heute (landen am Ende).

## Capabilities

### New Capabilities

- `chat-open-at-unread`: Verhalten beim Öffnen einer Konversation und die
  visuellen Marker (Divider, „ältere-ungelesene"-Chip).

### Modified Capabilities

Keine. `chat-konversationen` (Message-Endpoint) und `chat-day-separators`
(visuelles Muster) bleiben unangetastet — dies ist rein additiv.

## Impact

- **Backend**: keine Änderungen. `unreadCount` in der Conversation reicht;
  `message_reads` wird bereits geführt.
- **Frontend**: `ChatPage.tsx` — `openConversation`-Logik, neuer
  `UnreadDivider`-Komponente (~30 Zeilen JSX), Chip-Komponent (~15 Zeilen),
  einer Ref auf den Divider für präzises `scrollIntoView`. Der bestehende
  `forceScrollToEndRef` behält seine Rolle für den `unreadCount === 0`-Pfad.
- **Tests**: 3 neue in `web/src/pages/__tests__/ChatPage.openAtUnread.test.tsx`
  (unread-Fall, kein-unread-Fall, unread-älter-als-Seite-Fall).
- **Kein Migrations-, API- oder Konfigurations-Change.**
