## Context

Die Update-Detection (`useVersionCheck` + SSE `__version:<git-hash>`, Spec `deploy-version-detection`) funktioniert: Nach einem Deploy weicht der vom Server gesendete Hash vom zuerst gesehenen ab → Banner. Auch der SW-eigene `useRegisterSW.onNeedRefresh` triggert. Was nicht funktioniert, ist die *Auslieferung* der neuen Version: beim Klick auf "Jetzt laden" — oder beim Cold-Start der iOS-PWA — bekommt der User trotzdem oft die alte Shell.

Aktueller Zustand:

```
Browser                     Service Worker            Go-Server (embed.FS)
───────                     ──────────────            ────────────────────
HTTP-Cache (heuristisch)    workbox-precache-vX       spaHandler
  - sw.js (alt)              - index.html (alt)        - keine Cache-Header
  - index.html (alt)         - assets/* (alt-Hashes)   - kein ETag (embed.FS
  - manifest.json (alt)      - api-cache                 hat keine ModTime)
                             - app-shell (existiert      - FileServer einzig
                               heute NICHT)               für Validators
```

`http.FileServer` über `http.FS(embed.FS)` ruft `http.ServeContent` auf. Ohne ModTime gibt's kein `Last-Modified`; ohne expliziten ETag gibt's auch keinen. Damit fallen Browser-HTTP-Cache und SW-Update-Check auf Heuristiken zurück, die auf iOS-Safari besonders aggressiv sind.

`web/src/sw.ts` precached aktuell ALLE Assets aus `__WB_MANIFEST`, inklusive `index.html`. Beim PWA-Coldstart liefert der SW die Navigation aus dem Precache — ohne jemals das Netz zu fragen, solange der alte SW noch controlling ist.

`web/src/lib/reload.ts` löscht im Fallback-Pfad nur `api-cache`. Der Workbox-Precache-Cache (`workbox-precache-v2-https://intern.team-stuttgart.org/`) bleibt intakt; `location.reload()` triggert dann wieder denselben Precache-Treffer.

Stakeholder: alle aktiven Nutzer (~150–200), besonders iOS-PWA-Anteil (>60 %).

## Goals / Non-Goals

**Goals:**
- Nach einem Deploy erhält jeder User innerhalb einer normalen Session die neue Version, ohne logout/PWA-Beenden zu müssen.
- "Jetzt laden" im Update-Banner ist eine harte Garantie: spätestens nach diesem Klick läuft die neue Version.
- Cold-Start einer installierten iOS-PWA liefert mit Netzverbindung IMMER die aktuelle `index.html`.
- Offline-Start bleibt funktionsfähig (cached App-Shell + Offline-Hinweis).
- Push-Notifications-Subscription bleibt unverändert.
- VPS-RAM/Disk-Footprint unverändert (kein neuer Service, keine neue Abhängigkeit).

**Non-Goals:**
- Kein `clientsClaim()`/`skipWaiting()` im SW-`install`-Handler — der User soll weiterhin entscheiden, wann das Update aktiviert wird; SSE-Streams sollen nicht mitten in Aktion gekappt werden.
- Kein Wechsel `registerType: 'autoUpdate'` → `'prompt'`. Wir promten bereits selbst.
- Kein Server-seitiges "Force-Logout" bei Deploy — zu invasiv.
- Keine Änderung am SSE-`__version:`-Protokoll oder am `useVersionCheck`-Hook.
- Keine Veränderung der bestehenden Workbox-Routen für `/api/*`, Google Fonts oder Push-Handling.

## Decisions

### Decision 1: Backend-Cache-Header nach Pfad-Muster, ETag aus buildHash

`spaHandler` setzt vor `http.FileServer.ServeHTTP` die Header. Pfad-Klassifikation:

```go
isHashedAsset := strings.HasPrefix(path, "assets/") && hasContentHashRegex.MatchString(path)
if isHashedAsset {
    w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
} else {
    w.Header().Set("Cache-Control", "no-cache, must-revalidate")
    etag := fmt.Sprintf(`"%s-%x"`, buildHash, sha256.Sum256([]byte(path))[:4])
    w.Header().Set("ETag", etag)
    if match := r.Header.Get("If-None-Match"); match == etag {
        w.WriteHeader(http.StatusNotModified)
        return
    }
}
```

**Rationale:**
- Hashed Assets können beliebig lange cached werden — der Dateiname ändert sich bei jedem Build.
- `no-cache, must-revalidate` zwingt den Browser zur Revalidierung bei JEDER Navigation. Kombiniert mit dem ETag wird der Bandbreiten-Hit minimal (304-Antwort), aber der Browser bekommt zuverlässig die neue Version.
- ETag aus `buildHash`+Pfad: beim nächsten Deploy ändert sich `buildHash` → ETag ändert sich → 200 statt 304 → Browser holt neu.
- Manuelles 304-Handling im Handler ist nötig, weil `http.ServeContent` ohne ModTime keine Validators selbst evaluiert.

**Alternative verworfen:** ETag aus SHA256 des Datei-Inhalts. Teurer (jeder Request hashed embed-Bytes); für `assets/*` ohnehin überflüssig; für entry-points reicht `buildHash` als Generations-ID.

**Alternative verworfen:** `Cache-Control: no-store`. Zu hart — verhindert 304 und blockiert Back/Forward-Cache.

**Pfad-Klassifikation für hashed Assets:** Vite emittiert standardmäßig `assets/<name>-<8+ hex>.<ext>`. Regex: `^assets/[^/]+-[a-zA-Z0-9_-]{8,}\.[a-z0-9]+$`. Match → immutable.

### Decision 2: NetworkFirst für Navigationen, kein Precache von index.html

`vite.config.ts` schränkt das Precache-Manifest ein:

```ts
VitePWA({
  strategies: 'injectManifest',
  srcDir: 'src',
  filename: 'sw.ts',
  registerType: 'autoUpdate',
  injectManifest: {
    globPatterns: ['**/*.{js,css,ico,png,svg,woff2}'], // KEIN .html
  },
  manifest: { ... },
})
```

`web/src/sw.ts` registriert VOR allen anderen API-Routen:

```ts
registerRoute(
  ({ request }) => request.mode === 'navigate',
  new NetworkFirst({
    cacheName: 'app-shell',
    networkTimeoutSeconds: 3,
    plugins: [new ExpirationPlugin({ maxEntries: 1, maxAgeSeconds: 60 * 60 * 24 * 30 })],
  })
)
```

**Rationale:**
- Bei Netzverbindung gewinnt das Netz — der User bekommt immer die aktuellste `index.html`, die auf die neuen Asset-Hashes verweist. Diese Assets sind im alten Precache nicht enthalten, werden per Workbox-Default an `fetch` durchgereicht und vom Server (mit `immutable`-Header) geliefert.
- Offline / sehr langsames Netz (>3 s): Fallback aus `app-shell`-Cache → User sieht die zuletzt erfolgreich geladene Shell, kein 4xx.
- `networkTimeoutSeconds: 3` ist UX-Kompromiss: lang genug für realistische Mobilfunk-Latenz, kurz genug um Cold-Start nicht zu blockieren.
- `ExpirationPlugin maxEntries: 1`: wir wollen genau die letzte erfolgreich gefetchte Shell behalten.

**Alternative verworfen:** `StaleWhileRevalidate` für Navigation. Würde dem User initial die alte Shell zeigen und im Hintergrund neu fetchen — UX wäre schlechter als heute, nicht besser. Genau das Problem, das wir lösen wollen.

**Alternative verworfen:** Precache komplett deaktivieren. Wir verlieren Offline-Support für Assets — Hashed Assets sind aber ohnehin am Server immutable, das ist OK; aber Push-Handling braucht ggf. den SW im Lifetime.

### Decision 3: Reload-Fallback erweitert Cache-Cleanup

In `web/src/lib/reload.ts`, im Fallback-Zweig (kein waiting SW nach Polling):

```ts
const keys = await caches.keys()
await Promise.all(
  keys
    .filter(n =>
      n === API_CACHE_NAME ||
      n === APP_SHELL_CACHE_NAME ||
      n.startsWith('workbox-precache')
    )
    .map(n => caches.delete(n))
)
location.reload()
```

**Rationale:**
- "Jetzt laden" muss als Notfall-Hammer funktionieren: wenn die SW-Maschinerie aus irgendeinem Grund nicht reagiert, soll nach dem Klick keine alte Daten-Quelle übrig sein.
- `workbox-precache*` (Wildcard via `startsWith`) deckt alle Varianten von Workbox-Cache-Namen ab (`workbox-precache-v2-<origin>/` etc.).
- Andere Caches (`google-fonts-cache`, `google-fonts-static-cache`) bleiben — sie sind unkritisch für die App-Version.

**Alternative verworfen:** `caches.keys().then(ks => Promise.all(ks.map(caches.delete)))` (alles löschen). Killt unbeteiligte Caches; auch Fonts müssen neu geladen werden. Unnötiger Bandbreiten-Hit.

### Decision 4: Banner-UX und Cleanup unverändert

`AppShell.tsx` zeigt das Banner weiter über `showUpdateBanner = sseUpdateAvailable || swUpdateAvailable`. Mit den drei Bausteinen sollte der SSE-Pfad in 95 % der Fälle ausreichen, der SW-Pfad als zweite Quelle bleibt. Kein UI-Change.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| NetworkFirst mit 3 s Timeout verzögert Cold-Start bei sehr langsamen Netzen | 3 s ist konservativer Kompromiss; bei Offline-Fallback sieht der User die letzte Shell. Bei realer 4G-Latenz < 1 s. |
| Beim ersten Rollout sehen User, die noch den ALTEN SW haben, einmalig die alte Shell weiter (alter SW kennt die NetworkFirst-Regel nicht) | Self-healing nach SKIP_WAITING + Reload. Einmalig-Effekt. Optional: kurze Hinweis-E-Mail an Vorstand. |
| ETag-Berechnung pro Request — Performance-Hit | Minimal: SHA256 über 4-Byte-Pfad-Prefix, einmal pro Request, Sub-Mikrosekunden. Bei 1 GB-RAM-VPS irrelevant. |
| Workbox-Cache-Name `workbox-precache*` ändert sich in zukünftiger Workbox-Version | Risiko gering (stabile Konvention seit Workbox 5). Bei Major-Bump anpassen — wird in CHANGELOG sichtbar. |
| Browser ohne SW (alte Safari-Versionen) profitieren nur von Baustein 1 | Akzeptiert — Baustein 1 allein behebt schon ~80 % laut Hypothese. |
| Push-Notifications-Handler in `sw.ts` muss bei Code-Bewegungen vollständig erhalten bleiben | Tasks-Checkliste enthält expliziten Schritt zur Verifikation. |
| `manifest.webmanifest` vs `manifest.json` — Vite-PWA generiert `manifest.webmanifest`, wir referenzieren ggf. beides | Cache-Header-Regel matcht beide Dateinamen über die "alles außer hashed-assets"-Klausel. |

## Migration Plan

1. **Phase 1 (in einem PR)**: Alle drei Bausteine zusammen mergen. Sie sind unabhängig wirksam, aber gegenseitig verstärkend.
2. **Vor Deploy auf VPS**: lokaler Smoke-Test mit `make build` + `bin/teamwerk`, Curl-Tests gegen `localhost:8080/` und `localhost:8080/assets/...`.
3. **Deploy**: `make deploy`. Direkt nach Deploy `curl -I https://intern.team-stuttgart.org/` und auf `Cache-Control` + `ETag` prüfen.
4. **Validation auf iOS-PWA**: Vorstand-Tester schließt PWA, öffnet sie wieder; verifiziert neuen Hash im Sidebar-Footer (`useVersionCheck` zeigt ihn an).
5. **Rollback**: `git revert` der entsprechenden Commits + `make deploy`. Die Cache-Header-Änderung im Backend ist sofort wirksam; die SW-Änderung braucht einen weiteren Reload-Zyklus, weil der "neue" alte SW erst wieder aktivieren muss.

## Open Questions

- Soll `manifest.webmanifest` mit explizitem `version`-Feld versehen werden? iOS verwendet das Feld kaum, könnte aber bei Add-to-Homescreen-Update-Erkennung helfen. Heute nicht vorgesehen — separat evaluierbar.
- Vite-Plugin-PWA: `injectManifest` ignoriert `globPatterns` für HTML standardmäßig nicht — Validation, ob `index.html` wirklich aus `__WB_MANIFEST` rausfällt, muss in den Tasks (Build-Output prüfen). Fallback wäre `manifestTransforms`-Hook im Vite-Config.
