## Context

Siehe `proposal.md` — Why. Für den Entwurf relevant ist der Ist-Zustand der beiden
SSE-Hooks, die dasselbe tun und dabei unterschiedlich viel richtig machen:

| | `useLiveUpdates` (`/api/events`) | `useChatEvents` (`/api/chat/events`) |
|---|---|---|
| Auth | Cookie, kein Token in der URL | Cookie — aber Token trotzdem in der URL |
| Dependency | `[user]` | `[]` |
| Bei `readyState === CLOSED` | `es.close()`, kein Reconnect | `es.close()`, kein Reconnect |
| Coalescing | 300 ms je Event-Typ | keins |

Beide geben also bei einem fatalen Verbindungsfehler endgültig auf. Beim globalen Kanal
fällt das kaum auf, weil dort jede betroffene Seite ihre Daten ohnehin beim Betreten lädt
und Nutzer zwischen Seiten navigieren. Beim Chat-Zähler fällt es maximal auf: `AppShell`
bleibt über die gesamte Sitzung gemountet, der Zähler wird beim Navigieren **nicht** neu
geladen, und die PWA wird nicht neu geladen, sondern fortgesetzt.

Ein wichtiges Detail zum Reconnect-Verhalten von `EventSource`: Bei einem reinen
Transportabbruch reconnected der Browser selbst (`readyState === CONNECTING`) — hier ist
nichts zu tun. Endgültig tot ist die Verbindung nur bei `readyState === CLOSED`, was der
Browser bei einer HTTP-Fehlerantwort setzt (401 nach abgelaufenem Refresh-Cookie, 502 beim
Neustart des Servers hinter nginx, 503 im Wartungsmodus). Genau dieser Zustand wird heute
mit `es.close()` quittiert und nie verlassen.

## Goals / Non-Goals

**Goals:**
- Ein Verbindungsabbruch darf die Aktualität einer Anzeige nicht dauerhaft beschädigen.
- Der Chat-Zähler bekommt einen vom Live-Kanal unabhängigen Weg zurück zur Wahrheit.
- Die Reconnect-Logik entsteht **einmal**, nicht als dritte Kopie neben den zwei bestehenden.

**Non-Goals:**
- Kein Polling im Sekunden-/Minutentakt. Der VPS hat 1 GB RAM, und jeder offene SSE-Kanal
  hält bereits eine Goroutine; ein zusätzlicher periodischer Request pro Client wäre der
  falsche Preis für einen Fall, der bei intaktem Kanal nie eintritt.
- Kein Nachreichen verpasster Ereignisse (`Last-Event-ID`, Event-Journal). Der Zähler wird
  nach dem Reconnect frisch geladen — das ist die Wahrheit, nicht die Summe der verpassten
  Deltas.
- Der App-Icon-Badge (`navigator.setAppBadge`) bleibt unangetastet; er hängt am Push-Pfad
  und hat mit `chatUnread` nur die Quelle gemein.

## Decisions

### 1. Gemeinsame Basis `useEventStream` statt Reparatur an zwei Stellen

Die Verbindungsführung (aufbauen, an die Identität binden, bei `CLOSED` mit Backoff neu
verbinden, beim Unmount aufräumen) zieht in einen Hook `web/src/hooks/useEventStream.ts`.
`useLiveUpdates` und `useChatEvents` behalten ihre Namen, ihre Signatur und ihre
Besonderheiten (Coalescing bzw. keins) und bauen darauf auf.

*Warum nicht nur `useChatEvents` reparieren?* Weil der Reconnect-Defekt in beiden steckt und
die nächste Kopie sonst denselben Fehler erbt — die Tabelle oben ist die Dokumentation eines
Musters, das schon einmal unvollständig übernommen wurde. Der gemeinsame Hook ist die
Naht, an der ein dritter Kanal automatisch alles richtig macht.

*Warum `useLiveUpdates` mitverändern, wenn der Fehler beim Chat gemeldet wurde?* Weil sein
Verhalten sich dadurch nur an einer Stelle ändert (Reconnect statt Aufgeben) und diese
Änderung dort ebenso erwünscht ist. Der Alternative — Reconnect nur im Chat-Kanal — stünde
eine dauerhafte Asymmetrie gegenüber, die niemand mehr auflöst.

### 2. Backoff: exponentiell ab 1 s bis 30 s, Reset bei `onopen`

Wartezeiten 1 s → 2 s → 4 s → 8 s → 16 s → 30 s (danach konstant), Zähler zurück auf 0,
sobald eine Verbindung offen ist. Kein Jitter: die Nutzerzahl liegt im niedrigen dreistelligen
Bereich, ein Thundering Herd nach Server-Neustart ist bei 30 s Obergrenze kein reales
Problem, und Jitter würde die Tests unnötig verwackeln.

Kein Abbruch nach N Versuchen. Ein dauerhaft scheiternder Kanal kostet alle 30 s einen
Request — billiger als ein Client, der bis zum nächsten manuellen Reload blind ist.

### 3. Sichtbarwerden als zweiter Auslöser — für Verbindung *und* Zähler

`visibilitychange` (sichtbar) und `online` lösen beides aus: einen sofortigen
Reconnect-Versuch, falls der Kanal tot ist (mit zurückgesetztem Backoff — die Rückkehr des
Nutzers ist ein starkes Signal, dass sich die Netzlage geändert hat), und ein Neuladen des
Zählers.

Der Zähler-Refetch hängt bewusst **nicht** am Reconnect-Erfolg: er ist der Sicherheitsgurt
für genau den Fall, dass mit dem Kanal etwas nicht stimmt. Beide Auslöser sitzen in
`AppShell`, wo `loadChatUnread` bereits lebt.

### 4. „Unbekannt" wird von „0" getrennt

`chatUnread` bekommt einen Begleitzustand „schon einmal erfolgreich geladen". Vor dem ersten
Erfolg zeigt keine Anzeigestelle einen Badge — dieselbe Trennung, die der zurückgerollte
Commit `8c8c649e` für den App-Icon-Badge eingeführt hatte, hier aber für die In-App-Anzeige
und ohne dessen Nebenwirkung: das Icon-Badge wird von diesem Zustand nicht angefasst.

Der Unterschied zu heute ist im Normalfall unsichtbar (der erste Ladeversuch gelingt in
Millisekunden) und zählt nur im Fehlerfall — dort verhindert er, dass ein gescheiterter
Ladeversuch als „nichts ungelesen" gelesen wird.

### 5. Der Query-Token verschwindet ersatzlos

Kein Fallback, keine Übergangsfrist: Der Server wertet den Parameter nicht aus (die Route
hängt an `auth.CookieMiddleware`, siehe `internal/app/router.go`), es gibt also nichts, was
brechen könnte. Backend-seitig ist nichts zu tun.

## Risks / Trade-offs

**Reconnect-Sturm bei ausgefallenem Backend** → Obergrenze 30 s plus die Tatsache, dass ein
totes Backend ohnehin keine Verbindung annimmt. Im Wartungsmodus (HTTP 503 auf Mutationen)
ist der SSE-GET nicht betroffen.

**`useLiveUpdates` ist an ~15 Seiten im Einsatz** → Die Signatur bleibt identisch, die
Änderung ist additiv (Reconnect statt Aufgeben). Die bestehenden Tests der Seiten decken
den Normalfall ab; der Hook selbst bekommt eigene Tests für Abbruch und Wiederaufbau.

**`visibilitychange` feuert in der PWA häufiger als erwartet** (Task-Switcher, Splitscreen)
→ Der Refetch sind zwei GETs auf schlanke Listen; er ist idempotent und ohne Spinner. Falls
sich das in der Praxis als zu gesprächig erweist, ist die Stelle für ein Mindestintervall
eine einzelne Funktion.

**Fake-Timer-Tests** für den Backoff neigen zu Flakiness → Der Backoff wird als reine,
exportierte Funktion (`nextBackoffMs(attempt)`) getestet, die Verbindungslogik separat mit
einem EventSource-Mock. Dasselbe Muster wie `createThrottledProgress` in
`VideoUploadPage.tsx`.

## Migration Plan

Reines Frontend-Deployment, keine Migration, keine Änderung an Routen oder Daten. Rollback
ist ein Revert des Commits und ein erneuter Deploy — der vorherige Stand bleibt lauffähig,
weil serverseitig nichts angepasst wurde.

Nach dem Deploy in der iOS-PWA gegenprüfen: App in den Hintergrund, Nachricht von einem
zweiten Account senden, App wieder in den Vordergrund holen — der Zähler muss ohne Reload
erscheinen.
