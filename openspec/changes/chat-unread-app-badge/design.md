## Context

Die Chat-Unread-Logik existiert in zwei Welten:

- **Frontend (`AppShell.tsx`, Z. 97–110):** `chatUnread = Σ conversation.unreadCount + count(broadcasts where !isRead && !isSent)`. Live aktualisiert via SSE-Events (`chat:new-message`, `chat:new-broadcast`, `chat:conversation-read`).
- **Backend (`internal/chat/handler.go`):** `unreadCount` pro Konversation wird in `GET /api/chat/conversations` aus `messages` ./. `chat_read_state` berechnet. Eine reine "Summe pro User"-Hilfsfunktion gibt es noch nicht.

Push-Infrastruktur:

- `push.SendToUsers(db, cfg, userIDs, title, body, url)` — JSON-Payload `{title, body, url}`, fire-and-forget.
- Service Worker `web/src/sw.ts` Z. 50–61 — `push`-Handler ruft `showNotification(...)`.

Web Badging API:

- `navigator.setAppBadge(n)` / `navigator.clearAppBadge()` — sowohl im Page- als auch im Service-Worker-Kontext verfügbar.
- Plattformen: Chrome/Edge (Desktop, Android), Safari macOS 16.4+, iOS Safari 16.4+ (PWA installiert + Notification-Permission). Firefox: nicht implementiert → graceful degrade.

## Goals / Non-Goals

**Goals:**

- App-Icon-Badge zeigt aktuelle Summe ungelesener Chat-Nachrichten + ungelesener Broadcasts.
- Live-Update bei offener App (per Page-Context).
- Update bei eingehendem Push, wenn App geschlossen (per Service-Worker-Context).
- Graceful Degrade auf Browsern ohne Badging-Support — Feature-Detection, keine Errors.

**Non-Goals:**

- Kein Multi-Device-Sync beim Lesen (Eventual Consistency akzeptiert). Wer auf Gerät A liest, sieht auf Gerät B den alten Wert, bis dort ein neuer Push eintrifft oder die App geöffnet wird.
- Keine Broadcast-Alterung (alter ungelesener Broadcast zählt weiter).
- Keine Aufnahme anderer Domänen (Carpooling, Anfragen, Dienste).
- Kein separater "Silent Push" zur Badge-Korrektur (iOS lässt das ohnehin nicht zu — jeder Push muss `showNotification` triggern).

## Decisions

### 1. Push-Payload-Erweiterung

Bestehende Payload `{title, body, url}` wird um `badge` ergänzt. `badge` ist optional — andere Push-Caller (z.B. Spielzusagen, Dienste) bleiben unverändert und schicken kein `badge`. Der Service Worker setzt den Badge nur, wenn das Feld vorhanden ist.

```json
{"title": "...", "body": "...", "url": "...", "badge": 7}
```

`badge` ist eine absolute Zahl (Setzwert), nicht ein Delta — das matched die Browser-API `setAppBadge(n)` 1:1 und ist robust gegen verlorene Pushes.

### 2. Neue Funktion `SendToUserWithBadge` statt `SendToUsers`-Refactor

Drei Optionen wurden erwogen:

- **(a)** Signatur-Refactor: `SendToUsers([]UserPayload)` mit `UserID` + `Badge` pro Eintrag.
- **(b)** Variadic-Optionen: `SendToUsers(..., WithBadge(map[int]int))`.
- **(c)** Neue Funktion `SendToUserWithBadge(userID, ...)` parallel zum Bestehenden.

**Entscheidung: (c).** Bestehende Caller (Games, Trainings, Duties) bleiben unangetastet. Chat ist heute der einzige Caller, der pro Empfänger einen eigenen Badge-Wert hat. Pro-User-Funktion macht das auch aufrufseitig klar lesbar: jede Iteration berechnet den User-spezifischen Count und versendet einzeln.

```go
func SendToUserWithBadge(db *sql.DB, cfg *appconfig.Config,
    userID int, title, body, url string, badge int) {
    // identisch zu SendToUsers, aber mit Single-User-Query und Badge in Payload
}
```

### 3. Helper: `chat.ComputeUnreadForUser`

Eine in `internal/chat/` exportierte Funktion replizert die Frontend-Logik aus `AppShell.loadChatUnread`:

```go
// ComputeUnreadForUser liefert die Summe aller ungelesenen 1:1- und Gruppen-
// Nachrichten plus die Anzahl ungelesener Broadcasts (nicht selbst gesendet).
func ComputeUnreadForUser(db *sql.DB, userID int) (int, error)
```

SQL ist eine Vereinigung zweier Counts — die exakten Queries leitet die Implementierung aus den bestehenden List-Endpoints ab (Z. 163 und Z. 396 im Handler). Eine einzelne `SELECT … UNION ALL …`-Abfrage liefert beide Werte und summiert sie in Go.

**Warum nicht in `notifications/` ablegen?** Das Wissen über `messages`, `chat_read_state`, `chat_broadcasts`, `chat_broadcast_reads` ist Chat-Domäne. Das Helper-Modul gehört zum Eigentümer der Tabellen.

### 4. Push-Caller-Stellen

Aus `grep "push.SendToUsers" internal/chat/` ergeben sich die Stellen, an denen heute Chat-Pushes ausgehen (neue Nachricht in Konversation, neuer Broadcast). Beide Stellen werden umgestellt: statt `SendToUsers(recipients, title, body, url)` wird pro Empfänger berechnet und einzeln gesendet:

```go
for _, uid := range recipients {
    badge, err := chat.ComputeUnreadForUser(h.db, uid)
    if err != nil {
        log.Printf("chat: compute unread for user %d: %v", uid, err)
        badge = 0
    }
    go push.SendToUserWithBadge(h.db, h.cfg, uid, title, body, url, badge)
}
```

Die Goroutine bleibt — Push darf den HTTP-Response nicht blockieren.

### 5. Frontend: AppShell

```tsx
useEffect(() => {
    if (!('setAppBadge' in navigator)) return
    if (chatUnread > 0) {
        navigator.setAppBadge?.(chatUnread)
    } else {
        navigator.clearAppBadge?.()
    }
}, [chatUnread])

// Beim Logout:
useEffect(() => {
    if (!user && 'clearAppBadge' in navigator) {
        navigator.clearAppBadge?.()
    }
}, [user])
```

TypeScript: `navigator.setAppBadge` ist in den DOM-Lib-Typen ab TS 5.x deklariert. Falls Build-Probleme — `(navigator as Navigator & { setAppBadge?: (n: number) => Promise<void>; clearAppBadge?: () => Promise<void> })`.

### 6. Service Worker

```ts
self.addEventListener('push', (event) => {
    if (!event.data) return
    const data = event.data.json() as {
        title: string; body: string; url: string; badge?: number
    }
    const tasks: Promise<unknown>[] = [
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: '/icons/icon-192.png',
            badge: '/icons/icon-192.png',
            data: { url: data.url },
        }),
    ]
    if (typeof data.badge === 'number' && 'setAppBadge' in self.navigator) {
        tasks.push(
            data.badge > 0
                ? (self.navigator as any).setAppBadge(data.badge)
                : (self.navigator as any).clearAppBadge()
        )
    }
    event.waitUntil(Promise.all(tasks))
})
```

Achtung: `badge` ist im Web-Manifest-Sinn ein Icon (das `badge: '/icons/icon-192.png'`-Feld in `showNotification` ist der monochrome Push-Indikator auf Android, NICHT die App-Badge-Zahl). Beide Konzepte heißen "badge" — bleiben getrennte Variablen.

### 7. Kein Push bei Read

Wenn ein User eine Konversation als gelesen markiert, geht KEIN Push an die anderen Geräte des Users. Begründung: iOS lässt keinen Push ohne Notification zu — der User würde auf seinem Zweitgerät eine Notification "[…] hat eine Nachricht gelesen" bekommen, was nervig wäre. Die im Frontend offenen Geräte aktualisieren sich live über SSE (`chat:conversation-read`). Die anderen warten auf den nächsten regulären Push.

### 8. Werte-Range

`setAppBadge(n)` akzeptiert beliebige nicht-negative Integer; viele Plattformen visualisieren >99 als "99+". Wir senden den exakten Wert und überlassen der Plattform die Darstellung. Maximum aus DB ist effektiv durch die Tabellengrößen begrenzt; kein Cap nötig.

## Risks / Trade-offs

**[Mehr Push-Requests pro Chat-Event]** → Heute: ein einziger `SendToUsers`-Call mit `len(recipients)` Subscriptions. Künftig: `len(recipients)` einzelne Aufrufe, jeder lädt seine Subscriptions. Bei N Empfängern N×M Subscriptions → unverändert in der Anzahl der webpush-Calls; nur die DB-Query-Last steigt linear mit Empfängern (ein `SELECT … FROM push_subscriptions WHERE user_id = ?` pro Empfänger statt einem `WHERE user_id IN (...)`). Für TeamWERK-Größen (Gruppen typischerweise < 30 Personen) irrelevant.

**[ComputeUnreadForUser-Query pro Empfänger]** → Eine zusätzliche Aggregation pro Push-Empfänger. SQLite mit Index auf `chat_read_state(user_id, conversation_id)` und `chat_broadcast_reads(user_id, broadcast_id)` (sollten existieren — Verifikation in Task 1). Bei Gruppen-Chats N Counts × N Empfänger = N² für die seltene "großer Gruppen-Chat"-Auslastung. Akzeptabel; falls eines Tages zu langsam, kann der Count im selben Statement wie die Pushliste gejoined werden.

**[Badge driftet bei verlorenem Push]** → Ein verpasster Push lässt den Badge zu niedrig; ein bereits gelesener Broadcast auf anderem Gerät lässt ihn zu hoch (Eventual Consistency). Selbstheilung beim nächsten Push (absoluter Setzwert) oder beim App-Öffnen (Page-Effect setzt aktuellen Wert).

**[iOS-PWA-Sonderfall]** → Auf iOS funktioniert `setAppBadge` nur, wenn der Push auch eine Notification anzeigt. Wir tun beides im selben `waitUntil` — passt zu iOS. Wenn der User Push-Permission verweigert, gibt es weder Notifications noch Badge — selbe Limitation wie heute schon.

**[Feature-Detection-False-Positive]** → `'setAppBadge' in navigator` ist `true` auch in einem normalen Tab — die Methode existiert, hat aber im nicht-installierten Browser oft keinen sichtbaren Effekt. Das ist kein Bug: setAppBadge gibt dann einfach lautlos zurück. Kein Sonderfall im Code nötig.
