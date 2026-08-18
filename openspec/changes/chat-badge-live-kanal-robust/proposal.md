## Why

Der In-App-Zähler für ungelesene Nachrichten (Nav-Modul-Header, Hinweis-Punkt am Hamburger,
Tabs auf `/chat`) friert dauerhaft ein, sobald der Chat-SSE-Kanal einmal abreißt. Gemeldet
als „die Badges wurden plötzlich gar nicht mehr angezeigt, obwohl es anfangs funktioniert
hat" — reproduzierbar in der iOS-Homescreen-PWA, die beim Wechsel in den Hintergrund
eingefroren und später ohne Neu-Mount fortgesetzt wird.

Der Grund ist strukturell, nicht datenabhängig: `chatUnread` hat genau zwei Schreiber — den
Mount-Effekt und den SSE-Callback. Fällt der Kanal aus, gibt es **keinen** dritten Weg, über
den die Zahl je wieder aktuell wird; sie bleibt auf dem zuletzt geladenen Wert stehen, und
der ist typischerweise `0`, weil beim letzten Blick in den Chat alles gelesen war. Der
Nutzer sieht dann dauerhaft „nichts Ungelesenes", während Nachrichten eintreffen.

## What Changes

- **Chat-SSE-Kanal übernimmt das Muster von `useLiveUpdates`**: `useChatEvents` baut die
  Verbindung an der Nutzer-Identität auf (`[user]` statt `[]`), nicht mehr einmalig beim
  Mount. Nach Login-Wechsel und Impersonation hört der Kanal damit nicht länger auf den
  vorherigen Nutzer.
- **Access-Token verschwindet aus der EventSource-URL.** `/api/chat/events` hängt
  serverseitig an `auth.CookieMiddleware` und wertet den Query-Parameter überhaupt nicht
  aus — der Token wird heute wirkungslos mitgeschickt und landet dabei in nginx-Access-Logs
  und Proxy-Logs. Das ist derselbe Punkt, den die Spec `sse-live-updates` für `/api/events`
  bereits fordert; der später gebaute Chat-Kanal hat das Muster nie übernommen.
- **Reconnect statt endgültigem Aufgeben.** Heute schließt `es.onerror` die Verbindung bei
  `readyState === CLOSED` und gibt damit für die Lebensdauer der Seite auf. Künftig wird mit
  begrenztem Backoff neu verbunden.
- **Zweiter Aktualisierungspfad für den Zähler**: Refetch, wenn das Dokument wieder sichtbar
  wird (`visibilitychange`) und wenn der Browser wieder online geht (`online`). Genau der
  Fall der resumten PWA, die keinen Neu-Mount durchläuft.
- **Fehlgeschlagener Start-Load bleibt nicht folgenlos.** `loadChatUnread` schluckt Fehler
  heute still (`catch {}`) und versucht es nie erneut; ein Kaltstart ohne Netz lässt den
  Zähler dauerhaft auf `0`. Künftig wird ein fehlgeschlagener Ladeversuch beim nächsten
  Sichtbarwerden nachgeholt.
- Kein fachliches Backend-Verhalten ändert sich: keine neue Route, keine Migration, keine
  Änderung an der Unread-Berechnung selbst.

## Capabilities

### New Capabilities

_Keine._ Der Change härtet zwei bestehende Fähigkeiten.

### Modified Capabilities

- `sse-live-updates`: Die Anforderung „Auth via Cookie, kein `?token=`-Query-Parameter" gilt
  ausdrücklich für **alle** SSE-Endpunkte des Systems, nicht nur `/api/events` — heute ist
  sie für `/api/chat/events` nicht erfüllt. Ergänzt wird die Anforderung, dass ein
  SSE-Hook seine Verbindung an die Nutzer-Identität bindet und einen Verbindungsabbruch mit
  begrenztem Backoff behandelt, statt endgültig aufzugeben.
- `chat-unread-badge-pfad`: Ergänzt eine Anforderung zur **Aktualität** der Zahl. Die
  bestehende Spec beschreibt vollständig, wo die Zahl erscheint und wie sie berechnet wird,
  aber nicht, unter welchen Bedingungen sie aktuell sein muss — genau die Lücke, in die
  dieser Fehler fällt.

## Impact

**Frontend**
- `web/src/hooks/useChatEvents.ts` — Verbindungsaufbau, Auth, Reconnect
- `web/src/components/AppShell.tsx` — `loadChatUnread`, zusätzliche Refetch-Auslöser
- Tests: `web/src/components/__tests__/AppShell.navBadges.test.tsx` (bestehend), neue Tests
  für Hook-Verhalten und Refetch-Pfade

**Backend**
- Keine Änderung am Verhalten. `internal/app/router.go` und `internal/auth/middleware.go`
  bleiben unverändert — der Query-Parameter wird schon heute ignoriert; er verschwindet
  lediglich auf der Client-Seite.

**Nicht betroffen**
- App-Icon-Badge (`navigator.setAppBadge`) und der Push-Pfad über
  `push.SendToUserWithBadge` — eigener Kanal, eigener Zähler, siehe `chat-unread-app-badge`.
- Die Berechnungsfunktion `chatUnreadCounts` in `web/src/lib/chatUnread.ts`.

## Test-Anforderungen

Keine neue oder geänderte HTTP-Route — die Absicherung liegt vollständig im Frontend
(Vitest). Garantierte Invarianten und die Tests, die sie halten:

| Invariante | Test |
|---|---|
| Ein Verbindungsabbruch (`readyState === CLOSED`) führt zu einem erneuten Verbindungsversuch, kein endgültiges Aufgeben | `useEventStream.test.ts` — „reconnect nach CLOSED" |
| Die Wartezeit wächst und ist nach oben begrenzt | `useEventStream.test.ts` — `nextBackoffMs`-Folge inkl. Obergrenze |
| Nach Unmount wird nicht mehr verbunden (kein Timer-Leak) | `useEventStream.test.ts` — „kein Reconnect nach Unmount" |
| Der Chat-Kanal schickt keinen Access Token in der URL | `useChatEvents.test.ts` — geöffnete URL ist exakt `/api/chat/events` |
| Ein Identitätswechsel baut den Kanal neu auf | `useChatEvents.test.ts` — Neuaufbau bei Wechsel von `user` |
| Der Zähler wird beim Sichtbarwerden und bei `online` neu geladen | `AppShell.unreadRefresh.test.tsx` |
| Ein gescheiterter Ladeversuch zeigt keinen Badge und wird nachgeholt | `AppShell.unreadRefresh.test.tsx` |
| Eine erfolgreich geladene `0` zeigt keinen Badge und gilt nicht als „unbekannt" | `AppShell.unreadRefresh.test.tsx` |
| Die bestehenden Anzeigeregeln bleiben unverändert | `AppShell.navBadges.test.tsx` (bestehend, muss grün bleiben) |
