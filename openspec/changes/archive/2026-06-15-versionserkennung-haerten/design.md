## Design

### Ausgangslage

```
       PROD-DEPLOY                       BROWSER-TAB (heute)
       ───────────                       ───────────────────

  Makefile -ldflags
  -X main.buildHash=<sha>                ┌─────────────────────────────┐
            │                            │ AppUpdateBanner             │
            ▼                            │  useVersionCheck()          │ Hook #1
  cmd/teamwerk/main.go                   │   → EventSource('/api/events│  (oft ohne Token,
            │                            │     ?token=…')              │   feuert ins Leere)
            ▼                            └─────────────────────────────┘
  hub.NewHandler(hub, buildHash)         ┌─────────────────────────────┐
            │                            │ AppShell (post-Login)       │
            ▼                            │  useVersionCheck()          │ Hook #2
  GET /api/events                        │   → 2. EventSource          │  (parallele Verbindung)
   data: __version:<sha>                 └─────────────────────────────┘
                                         ┌─────────────────────────────┐
                                         │ Service Worker (workbox)    │
                                         │   /api/auth/*  → NetworkOnly│
                                         │   /api/*       → NetworkFirst, 10s timeout
                                         │   (matcht auch /api/events!)│ ← Problem
                                         └─────────────────────────────┘
```

### Soll

```
  ┌──────────────────────────────────────────────────────────────────┐
  │                  VersionProvider (App-Root)                      │
  │                                                                  │
  │   useVersionCheck() ←─ einzige SSE-Verbindung; reagiert auf user │
  │                                                                  │
  │   liefert per Context: { version, updateAvailable }              │
  └──────────────────────────────────────────────────────────────────┘
              │                                  │
              ▼                                  ▼
      AppUpdateBanner                  AppShell.Sidebar
        useVersion()                     useVersion()  →  Versionslink

                                         Service Worker (workbox)
                                          /api/auth/*    → NetworkOnly
                                          /api/events    → NetworkOnly  ← neu
                                          /api/chat/events → NetworkOnly ← neu
                                          /api/*         → NetworkFirst
```

### Warum SSE und NetworkFirst sich beißen

Workbox `NetworkFirst` ist für request/response-Endpoints gedacht:

1. Browser fragt Resource.
2. SW startet `fetch()`, wartet bis `networkTimeoutSeconds`.
3. Bei Erfolg: Response klonen, ein Klon an den Caller, einer in den Cache.
4. Bei Timeout/Fehler: Cache-Fallback.

Für `text/event-stream` ist das defekt:

- **Response-Klon:** Ein Klon eines Streams ist legal, aber Workbox versucht den Klon zu `put()`-en — das materialisiert den Stream. Bei einer long-lived Verbindung, die nie endet, bleibt der `put()` hängen oder cached ein nutzloses Fragment.
- **Timeout-Fallback:** SSE liefert oft Sekunden nichts. 10 s ohne erste Bytes → SW liefert Cache → `EventSource` sieht alte/sinnlose Frames oder bekommt einen 0-Byte-Response und reconnectet sofort wieder. Ergebnis: Reconnect-Storm ohne je das aktuelle `__version:` zu sehen.
- **Server-Restart nach Deploy:** Reconnect-Versuch landet typischerweise in einem 1–5 s Restart-Fenster. Bei 10 s Timeout greift der Cache-Fallback dann nicht, aber die Konstellation ist trotzdem zerbrechlich, weil Workbox die Stream-Lebensdauer nicht erwartet.

**Lösung:** SSE-Routes explizit auf `NetworkOnly`. Routes-Match in Workbox läuft in Registrierungsreihenfolge — wir setzen die NetworkOnly-Regeln **vor** die `/api/*`-NetworkFirst-Regel.

### Warum zentraler Provider

Der heutige Doppelaufruf von `useVersionCheck()` produziert zwei parallele SSE-Verbindungen pro Tab (+ je eine pro `useLiveUpdates`-Hook, der davon unabhängig ist). Die obere (in `AppUpdateBanner`, vor Login) feuert mit leerem Token und wird vom Hook sofort verlassen (`if (!token) return`); die untere (in `AppShell`, nach Login) liefert die Daten. Beide Aufrufe haben dieselbe Hook-Identität und Lifecycle.

Das ist verschwenderisch und macht den Code schwer zu reasonen. Ein zentraler Provider:

- öffnet **eine** SSE-Verbindung pro Tab,
- exposed `{ version, updateAvailable }` per Context,
- reagiert auf `user`-Wechsel (Login/Logout/Impersonation) mit Reconnect,
- macht Test/Debug trivial (eine Quelle der Wahrheit).

### Warum kein `?token=…` mehr in der URL

Der Server (`internal/auth/middleware.go:CookieMiddleware`) liest ausschließlich das `refresh_token`-Cookie aus dem Request. `?token=…` ist effektlos. EventSource sendet Cookies bei same-origin automatisch mit. Der URL-Param ist Noise und verschleiert, dass die Auth über Cookie läuft.

### Robuster Reload-Flow

Heute:

```
reloadWithSwActivation()
  reg.waiting?
    ├─ ja → postMessage SKIP_WAITING + reload bei controllerchange   ← funktioniert
    └─ nein → location.reload()                                       ← serviert alten Precache
```

Das Problem ist `nein`: SSE meldet die neue Version Sekunden nach Deploy; der SW hat aber häufig noch nicht den neuen `sw.js` geladen (Browser pollt selbst-getaktet). Dann ist `reg.waiting` null und `location.reload()` greift den alten Precache.

Neu:

```
reloadWithSwActivation()
  reg = navigator.serviceWorker?.getRegistration()
  if (!reg) → location.reload()  // kein SW, einfacher Pfad

  if (reg.waiting) → skip-waiting-Pfad wie bisher

  // sonst: forcieren und kurz warten
  await reg.update()                  // löst SW-Update-Check sofort aus
  await waitForWaiting(reg, 5_000)    // pollt bis 5 s auf reg.waiting

  if (reg.waiting) → skip-waiting-Pfad
  else:
    // SW hat nichts Neues gefunden — wahrscheinlich Caches sind ok,
    // aber zur Sicherheit den API-Cache leeren
    await caches.delete('api-cache')
    location.reload()
```

Die 5 s sind ausreichend, weil der Browser bei `reg.update()` proaktiv pollt; der `sw.js` ist klein. Wenn der Server nach 5 s immer noch keinen neuen SW liefert, dann ist die SSE-Versions-Diff vermutlich entweder ein falscher Alarm (gleicher Hash, anderer Server-Pod) oder ein partieller Deploy — in beiden Fällen ist `caches.delete + reload` der sichere Fallback.

### Versionsbezogenes Dismiss

```ts
// statt: const [dismissed, setDismissed] = useState(false)
const [dismissedVersion, setDismissedVersion] = useState<string | null>(null)

// Banner sichtbar wenn:
const visible =
  (sseUpdateAvailable || swUpdateAvailable) &&
  dismissedVersion !== currentBanneredVersion
```

`currentBanneredVersion` ist der Versions-String, den der SSE-Pfad als „neue Version" erkennt. Wenn der Banner für Version X dismissed wurde und später Version Y kommt, ist `dismissedVersion (X) !== currentBanneredVersion (Y)` → Banner wieder sichtbar.

### Dev-Modus-Anzeige

Heute: `if (import.meta.env.DEV) return` in `useVersionCheck` → `version=null` → kein Link in der Sidebar.

Neu: in DEV setzen wir `version = 'dev'` direkt, ohne SSE-Verbindung. Der Sidebar-Link zeigt `v dev` und ist klickbar (öffnet das `ChangelogModal` wie gewohnt). `updateAvailable` bleibt `false` in DEV.

### Auswirkungen auf `useLiveUpdates`

`useLiveUpdates` öffnet eine eigene `/api/events`-EventSource für Domain-Events. Mit dem SW-Fix (NetworkOnly für `/api/events`) profitiert auch dieser Hook von zuverlässigem Reconnect — bisher waren wahrscheinlich vereinzelte Live-Update-Aussetzer (kalender-event, etc.) ebenfalls auf das gleiche SW-Problem zurückzuführen. Kein Code-Change am Hook nötig.

Mehrere parallele EventSource-Verbindungen auf `/api/events` sind serverseitig kein Problem — der Hub broadcastet an alle Subscriber.

### Was wir NICHT ändern

- **Backend (`internal/hub/handler.go`):** funktioniert korrekt. Schickt initial `__version:`, dann Domain-Events. Kein Change.
- **`useLiveUpdates`-Hook:** funktioniert. Profitiert nur indirekt vom SW-Fix.
- **CHANGELOG-Generierung:** unverändert.
- **`ChangelogModal`:** unverändert.

### Risiken & Tradeoffs

- **Risiko: Anwender:innen mit altem SW hängen weiter fest.** Ihr Browser hat den alten `sw.js` aktiv und fängt `/api/events` weiter `NetworkFirst` ab, bis der neue SW kommt. Workbox' `registerType: 'autoUpdate'` pollt selbst-getaktet — kann Stunden dauern. Bewusstes Inkaufnehmen: Wir können den Bug nicht aus der Vergangenheit fixen, nur die Zukunft. Ein einmaliger Hinweis im Release-Changelog („Browser-Tab schließen und neu öffnen für die zuverlässige Update-Erkennung") reicht.
- **Tradeoff: VersionContext führt eine neue Provider-Ebene ein.** Akzeptabel, weil bisher zwei parallele Hooks denselben Job machen — der Provider ist die saubere Konsolidierung.
- **Tradeoff: `caches.delete('api-cache')` im Reload-Fallback ist eine harte Aktion.** Wir leeren damit Offline-Daten. Vertretbar, weil dieser Pfad nur greift, wenn (a) die SSE eine neue Version meldet und (b) der SW nach 5 s noch kein Update gefunden hat — eine ungewöhnliche Konstellation, in der Cache-Konsistenz wichtiger ist als Offline-Verfügbarkeit.

### Verifikation

Manuell, weil PWA/SW automatisiert schwer zu testen sind:

1. **Smoke nach Deploy:** Browser-Tab vor Deploy offen halten. Nach Deploy: Banner muss innerhalb von ~10 s erscheinen.
2. **Klick auf „Jetzt laden":** Seite zeigt neuen Stand (am Versions-Link in der Sidebar überprüfbar).
3. **Sidebar-Versionslink im DEV:** `pnpm dev` → Sidebar zeigt `v dev`.
4. **Dismiss + zweiter Deploy:** Banner wegklicken, zweiten Deploy ausführen → Banner erscheint wieder.

Automatisch:

- Backend-Tests (`internal/hub/handler_test.go`) bleiben grün.
- Optional: Vitest-Test für `useVersionCheck`, der `EventSource` mockt und das Reconnect-Verhalten bei `user`-Change prüft. Kein zwingender Bestandteil dieses Changes (kein Test-Standard-Trigger, da keine neuen HTTP-Routen), aber sinnvoll für Regression-Schutz.
