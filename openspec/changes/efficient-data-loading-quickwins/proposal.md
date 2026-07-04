## Why

Eine Analyse des Datenverkehrs zum Client hat drei ineinandergreifende Ineffizienzen aufgedeckt, deren **billigste, risikoärmste Teilmenge** dieser Change adressiert (additiv, kein API-Vertragsbruch):

1. **Referenzdaten ohne Cache.** `/api/teams` (in 10+ Seiten), `/api/seasons` (7 Seiten), `/api/venues`, `/api/duty-types`, `/api/encryption-pubkey`, `/api/push/vapid-public-key` sind quasi statisch, werden aber bei **jedem** Seiten-Mount neu geladen. Es gibt weder serverseitige HTTP-Cache-Header (kein `Cache-Control`/`ETag` auf diesen Routen) noch einen clientseitigen Cache (`web/src/lib/api.ts` ist ein dünner Axios-Wrapper ohne Cache, kein react-query/SWR).
2. **SSE-Events lösen Voll-Refetch ohne Coalescing aus.** `web/src/hooks/useLiveUpdates.ts` ruft die Callback pro Event **sofort** auf; jede Seite lädt daraufhin ihre komplette Liste neu. Event-Bursts (z. B. mehrere Slot-Zuweisungen in Folge) erzeugen so N volle Refetches statt einem. Der globale SSE-Channel hat Buffer 1 mit Drop (`internal/hub/hub.go:36`) — die 1:1-Zustellung ist ohnehin nicht garantiert, ein Client-Debounce ist daher **semantisch unbedenklich**.
3. **`/api/duty-types` liefert `instruction_md` in der Liste.** Die Board-Antwort macht es bereits richtig (`has_instruction`-Flag statt Volltext), die Typen-Liste (`internal/duties/handler.go:90`) transportiert dagegen den vollständigen Markdown-Text (potenziell 2–5 KB je Typ) an alle Aufrufer, obwohl die Liste den Volltext nicht braucht.

Der größte strukturelle Hebel (ungescopter globaler SSE-Fan-out) und die Vertrags-ändernde Listen-Paginierung sind bewusst **ausgelagert** in die Changes `scoped-live-updates` bzw. `list-endpoint-pagination`.

## What Changes

- **Server-HTTP-Caching auf Referenz-/Immutable-Routen:**
  - `GET /api/encryption-pubkey`, `GET /api/push/vapid-public-key` → `Cache-Control: public, max-age=86400` (bzw. `immutable` für VAPID) + `ETag`/`304`. Diese ändern sich nur bei Key-Rotation/Deploy.
  - `GET /api/seasons`, `GET /api/teams`, `GET /api/venues`, `GET /api/age-class-rules`, `GET /api/duty-types` → schwacher `ETag` aus einem günstigen Content-Fingerprint (z. B. `MAX(updated_at)`/`COUNT`), `If-None-Match` → `304`. Kein `max-age` mit Klartext-Caching für nutzerspezifisch gefilterte Antworten (`/api/teams` filtert per Nutzer) — nur `ETag`/`304` + `Cache-Control: private, no-cache`.
- **Client-TTL-Cache in `lib/api.ts`:** eine schlanke In-Memory-Map (kein neues NPM-Paket) für die o. g. Referenzrouten mit kurzer TTL (z. B. `/seasons` 1 h, `/teams` 5 min, `/venues` 1 d), invalidiert durch die passenden SSE-Events (`seasons`/`settings`/`venues`/`duties`). Dedupliziert außerdem gleichzeitige In-Flight-Requests derselben URL (Single-Flight).
- **Service-Worker-Feintuning (`web/src/sw.ts`):** Referenzrouten auf `StaleWhileRevalidate`; die generische `api-cache` bekommt `maxEntries`/`maxAgeSeconds` (heute unbegrenzt → wächst monoton).
- **Coalescing in `useLiveUpdates`:** die Event-Callback wird über ein kurzes Fenster (z. B. 300 ms) gebündelt, sodass ein Burst gleicher Events genau **einen** Reload auslöst. Verhalten pro Event-Typ erhalten (kein Event geht „verloren", nur zusammengefasst).
- **`GET /api/duty-types` trimmt `instruction_md`:** die Liste liefert `has_instruction: bool` statt Volltext; der Volltext bleibt über den Einzel-/Detail-Pfad abrufbar (wie schon beim Board).

## Capabilities

### Added Capabilities

- `reference-data-caching`: End-to-end-Caching für quasi-statische Referenz-/Immutable-Daten (Server-`ETag`/`Cache-Control` + `304`, Client-TTL-Cache mit Single-Flight, SW-`StaleWhileRevalidate`).
- `live-update-coalescing`: `useLiveUpdates` bündelt Event-Bursts in einem kurzen Fenster zu einem Reload.

### Modified Capabilities

- `duty-type-instructions`: `GET /api/duty-types` liefert `has_instruction` statt `instruction_md`; Volltext nur noch im Detail-Pfad.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/encryption-pubkey` | `TestEncryptionPubkey_ETag_304` | Zweiter Request mit `If-None-Match` des zuvor gelieferten `ETag` → `304`, leerer Body. |
| `GET /api/push/vapid-public-key` | `TestVapidKey_CacheControlImmutable` | Antwort trägt `Cache-Control` mit `immutable`; Body = konfigurierter Public Key. |
| `GET /api/seasons` | `TestSeasons_ETagChangesOnMutation` | `ETag` nach Saison-Änderung unterscheidet sich vom vorherigen; unveränderter Stand → `304`. |
| `GET /api/duty-types` | `TestDutyTypes_ListOmitsInstructionMd` | Listen-Antwort enthält **kein** `instruction_md`, aber `has_instruction` (true bei gepflegtem Text). |
| `GET /api/duty-types/{id}` bzw. Detail-Pfad | `TestDutyTypes_DetailKeepsInstructionMd` | Volltext `instruction_md` bleibt im Detail-Pfad abrufbar. |

**Garantierte Invariante:** Kein Caching-Mechanismus verändert die **Autorisierung** — nutzerspezifisch gefilterte Antworten (`/api/teams`) bleiben `private` und werden nur per `ETag`/`304`, nie per gemeinsam nutzbarem `max-age`, gecacht.

## Mess-Anforderungen

Verglichen wird gegen `metrics/payload-baseline.md` aus `payload-measurement-harness` (Voraussetzung).

| Kennzahl | Werkzeug | Erwartung nach diesem Change |
|---|---|---|
| Payload `GET /api/duty-types` | `make measure` (Payload-Tabelle) | ~30 KB kleiner (10 Typen × fixer 3 072-Byte-`instruction_md` fallen aus der Liste). |
| Revalidierung Referenzrouten (`/seasons`, `/venues`, `/age-class-rules`, `/duty-types`, pubkey, VAPID) | `make measure` (Revalidierungs-Tabelle) | zweiter Call mit `If-None-Match` → `304` + ~0 Bytes statt 200 + volle Bytes. |
| HTTP-Requests je Session für Referenzrouten | Frontend-Test (Cache-Hit/Single-Flight in `lib/api.ts`) | Mehrfach-Mounts derselben Referenzroute lösen ≤ 1 Request je TTL-Fenster aus. |
| Client-Reloads bei Event-Burst | Frontend-Test (`useLiveUpdates`-Coalescing) | N Events desselben Typs im Fenster → 1 Reload. |

**Baseline-Regel:** Die Vorher-Zahlen für `duty-types`-Payload und die 304-Spalte werden vor der Umsetzung aus der Baseline übernommen; die Nachher-Zahlen ersetzen die betreffenden Zeilen in `metrics/payload-baseline.md`.

## Impact

- **Backend:**
  - `internal/config/vault.go` (pubkey), `internal/notifications/handler.go` (VAPID), `internal/config/handler.go` (seasons/teams-Quelle/age-class-rules), `internal/venues/handler.go`, `internal/duties/handler.go` (duty-types-Liste trimmen) — `Cache-Control`/`ETag`/`304`-Behandlung ergänzen; ein kleiner gemeinsamer `httpcache`-Helfer (ETag berechnen + `If-None-Match` prüfen) vermeidet Duplikation.
  - `internal/duties/handler.go:90` — `instruction_md` aus der Listen-Serialisierung entfernen, `has_instruction` ergänzen.
- **Frontend:**
  - `web/src/lib/api.ts` — TTL-Cache + Single-Flight für Referenzrouten, Invalidierung per SSE-Event.
  - `web/src/hooks/useLiveUpdates.ts` — Coalescing-Fenster.
  - `web/src/sw.ts` — `StaleWhileRevalidate` für Referenzrouten, `api-cache`-Grenzen.
  - `web/src/pages/AdminDutyTypesPage.tsx` bzw. Typ-Detail — Volltext aus Detail-Route laden statt aus Liste.
- **Kein** Schema-/Migrations-Change, **keine** neue Route, **kein** neues NPM-Paket.
- **Modellgrenze:** Bank-/SEPA-Routen und nutzerspezifische Listen bleiben unangetastet bzw. `private`.
