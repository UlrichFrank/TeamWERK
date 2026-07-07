# Design — efficient-data-loading-quickwins

## Warum nur ETag/304 (kein `max-age`) für nutzerspezifische Listen

`GET /api/teams` (`Games.ListTeamsForUser`) und `GET /api/seasons` liefern potenziell **nutzer-/rollenabhängig gefilterte** Daten. Ein geteilter `Cache-Control: public, max-age=…` würde in Zwischencaches (Nginx, Browser-Shared-Cache) unter Umständen die Antwort eines Nutzers an einen anderen ausliefern. Deshalb:

| Klasse | Beispiele | Strategie |
|---|---|---|
| **Immutable, nicht geheim** | `/api/encryption-pubkey`, `/api/push/vapid-public-key` | `Cache-Control: public, max-age=86400`(`immutable` für VAPID) + `ETag` |
| **Referenz, evtl. nutzergefiltert** | `/api/teams`, `/api/seasons`, `/api/venues`, `/api/age-class-rules`, `/api/duty-types` | `Cache-Control: private, no-cache` + `ETag` → Revalidierung per `304` |

`private, no-cache` erzwingt Revalidierung bei jedem Zugriff, aber ein `304` ist ein leerer Body — die Ersparnis ist die Payload, nicht der Round-Trip.

## ETag-Berechnung (günstig, ohne Full-Serialize)

Ein schwacher ETag aus einem billigen Fingerprint statt Hash über die volle Antwort:

```
ETag: W/"<COUNT>-<MAX(updated_at) oder MAX(id)>"
```

- `/api/seasons`, `/api/venues`, `/api/duty-types`: `SELECT COUNT(*), COALESCE(MAX(updated_at), '')` der jeweiligen Tabelle.
- Immutable-Keys: ETag = Hash des Key-Materials bzw. `buildHash` (analog zum bestehenden Asset-ETag in `router.go`).

Gemeinsamer Helfer:

```go
// internal/httpcache/etag.go (neu, Foundation-Package)
func Serve(w http.ResponseWriter, r *http.Request, etag string, cc string, body func() any) {
    w.Header().Set("ETag", etag)
    if cc != "" { w.Header().Set("Cache-Control", cc) }
    if match := r.Header.Get("If-None-Match"); match == etag {
        w.WriteHeader(http.StatusNotModified); return
    }
    writeJSON(w, body())
}
```

Der Helfer ist bewusst klein und domainfrei — er wird über den Architektur-Test als Foundation klassifiziert (importiert keine Domain-Packages).

## Client-TTL-Cache in `lib/api.ts`

Kein react-query/SWR (RAM-/Bundle-Budget, VPS 1 GB, „kein State-Manager"-Konvention). Stattdessen eine Map:

```
cache: Map<url, { data, expires }>
inflight: Map<url, Promise>   // Single-Flight gegen Doppel-Requests paralleler Komponenten
```

- Nur für eine **Allowlist** von Referenzrouten aktiv; alle anderen Calls unverändert.
- TTL pro Route konfigurierbar; Invalidierung durch SSE-Events (die App hört ohnehin auf `seasons`/`settings`/`venues`/`duties`).
- Der Browser-`ETag`/`304` und dieser TTL-Cache sind komplementär: TTL spart den Request ganz, `304` spart die Payload wenn der Request doch rausgeht.

## Coalescing in `useLiveUpdates`

```
onmessage → pendingEvents.add(type) → debounce(300ms) → flush: rufe Callback je eindeutigem Typ einmal
```

- Kein Event-Verlust: gepufferte, deduplizierte Typen werden nach dem Fenster gemeinsam gefeuert.
- 300 ms ist kürzer als eine menschliche Wahrnehmungsschwelle für „live" und lang genug, um Server-Broadcast-Bursts (mehrere `Broadcast`-Aufrufe in einem Handler) zusammenzufassen.
- Der bestehende `__version:`-Sonderfall (Deploy-Erkennung, `useVersionCheck` an separater EventSource) bleibt unberührt.

## Risiken

- **Stale-Referenzdaten:** Ein Nutzer sieht nach einer Team-/Saison-Änderung bis zu TTL-Länge alte Daten, wenn das SSE-Invalidierungs-Event ausfällt (Buffer-1-Drop). Mitigation: kurze TTLs; kritische Flows (`AdminSettingsPage`) können den Cache gezielt umgehen.
- **ETag-Fingerprint zu grob:** `MAX(updated_at)` erkennt Löschungen nur zusammen mit `COUNT`. Beide zusammen decken Insert/Update/Delete ab.
