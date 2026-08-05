## Context

`VideoUploadPage.tsx` startet den tus-Upload (Zeilen ~230–275) und einen Resume-Fall (~330–370). Beide registrieren `onProgress(bytesSent, bytesTotal)` bei tus-js-client 4.3.1. tus-js-client leitet `XHR.upload.onprogress` direkt weiter — d.h. bei jedem TCP-ACK-basierten Progress-Event läuft unser Callback.

React 19 behandelt zwei zusammenhängende `setState`-Aufrufe im selben Event-Handler als eine Batch (automatic batching), aber der komplette Batch-Zyklus (schedule → prepare → commit → paint) läuft trotzdem auf dem Main-Thread. Bei 30–100 Batches/s teilt sich der Main-Thread zwischen React und XHR — das ist der Storm.

## Goals / Non-Goals

**Goals:**
- `setProgress` und `setRemaining` maximal 1× pro Sekunde
- Restzeit-Anzeige reagiert innerhalb ~10 s auf Bandbreiten-Änderungen (statt asymptotisch seit Start)
- Keine externen Deps (keine `lodash.throttle` o.ä.) — kleines, lesbares Ref-basiertes Muster
- Testbar in Vitest ohne echten Upload

**Non-Goals:**
- `useThrottledCallback`-Custom-Hook — nur 2 Call-Sites, Extraktion wäre Overengineering
- Web-Worker für Upload — würde den Main-Thread komplett entlasten, aber tus-js-client nutzt XHR/fetch und Blob-Reads gehen sowieso am Worker vorbei (kein Zugewinn)
- Progress-Bar-Sub-Second-Animation — das UI-Update auf 1 Hz ist für Nutzer:innen völlig ausreichend

## Decisions

### Entscheidung: Throttle via Ref-Timestamp, nicht `useCallback`/State

**Gewählt:**
```typescript
const lastProgressRef = useRef(0)      // ms since epoch
const samplesRef = useRef<Array<{ t: number; bytes: number }>>([])
const lastPctRef = useRef(-1)

onProgress: (bytesSent, bytesTotal) => {
  const now = Date.now()
  samplesRef.current.push({ t: now, bytes: bytesSent })
  // Fenster auf letzte 10 s beschränken
  while (samplesRef.current.length > 1 && now - samplesRef.current[0].t > 10_000) {
    samplesRef.current.shift()
  }
  if (now - lastProgressRef.current < 1000) return
  lastProgressRef.current = now

  const pct = bytesTotal > 0 ? Math.round((bytesSent / bytesTotal) * 100) : 0
  if (pct !== lastPctRef.current) {
    lastPctRef.current = pct
    setProgress(pct)
  }

  const first = samplesRef.current[0]
  const dt = (now - first.t) / 1000
  const dbytes = bytesSent - first.bytes
  if (dt > 0.5 && dbytes > 0) {
    const rate = dbytes / dt
    setRemaining(fmtRemaining((bytesTotal - bytesSent) / rate))
  }
}
```

**Warum:**
- **Ref statt State für Timestamps**: keine Re-Render-Trigger durch die Throttle-Bookkeeping-Werte selbst
- **Sliding-Window (~10s)**: reagiert schnell auf Bandbreiten-Änderungen; Startup-Spike ist nach 10 s gedämpft
- **`pct !== lastPctRef.current`-Guard**: 99 % der onProgress-Events zwischen zwei Prozent-Sprüngen lösen keinen `setProgress` aus, auch wenn der Throttle-Timer bereits abgelaufen ist (spart zusätzlich Renders auf langsameren Uploads)
- **Refs vor Upload-Start zurücksetzen** (in `startTus` / `handleResume` vor `new tus.Upload(...)`), damit ein zweiter Upload in derselben Session sauber startet

**Alternative verworfen:** `useCallback` + `useMemo` mit throttled-Wrapper. Read-only-State (Ref) ist idiomatischer für „Timestamps merken", und tus-js-client speichert den Callback ohnehin in `this.options.onProgress` — ein re-created Callback pro Render würde das Config-Objekt zwar irritieren, tus-js-client liest es aber nur einmal beim `upload.start()`.

### Entscheidung: 1000 ms Throttle-Intervall

**Gewählt:** 1000 ms als fixer Wert.

**Warum:** Für einen Progress-Bar mit „X %" und Restzeit „Y min Z s" ist 1 Hz UI-Update völlig ausreichend — der Nutzer merkt keinen Unterschied zu 30 Hz. Kleinere Werte (250 ms, 500 ms) bringen keinen UX-Gewinn und lassen den Main-Thread wieder mehr arbeiten. Größere Werte (2000 ms) wirken schwerfällig.

**Alternative verworfen:** Adaptives Throttling (z.B. „bei langsamer CPU aggressiver"). Fest 1 Hz ist einfacher und universal richtig.

### Entscheidung: Sliding-Window statt exponentieller Glättung

**Gewählt:** Einfaches Fenster über die letzten 10 s (array-basierter Ring).

**Warum:** Bei einem 10-s-Fenster fällt der Startup-Overhead nach 10 s aus der Berechnung und die Rate reflektiert nur den aktuellen Durchsatz. Exponentielle Glättung (`newRate = α × current + (1-α) × prev`) wäre kürzer im Code, aber intransparenter zu debuggen und das α wäre ein Magic-Number.

**Alternative verworfen:** EWMA mit α=0.1. Verworfen wegen Debuggbarkeit — bei einem Fehler-Report kann ein Sample-Array direkt inspiziert werden.

### Entscheidung: Kein separater Hook

**Gewählt:** Throttle-Logik inline in beiden `onProgress`-Handlern (start + resume).

**Warum:** Zwei Call-Sites, jeweils ~15 LoC. Ein Custom-Hook (`useThrottledUploadProgress`) hätte ähnlich viel Boilerplate (Return-Tuple, Reset-Funktion, useRef-Kollektiv) und ohne echtes drittes Call-Site keine Wiederverwendbarkeit. Wenn später parallelUploads oder ein zweiter Upload-Screen dazukommt, kann die Extraktion nachgezogen werden.

**Alternative verworfen:** `useThrottledUploadProgress`-Hook. Prämature Abstraktion.

## Risks / Trade-offs

- **[Annahme: React-Storm ist Hauptursache]** Wir haben die Diagnose **nicht** mit einem Chrome-Performance-Profile abgesichert. Es ist plausibel (bekanntes tus-in-Browser-Muster), aber nicht bewiesen. Wenn der Fix nicht den erwarteten Effekt bringt: nächster Schritt ist ein Real-Upload-Performance-Profile. Die Änderung ist unabhängig davon Best-Practice und kostet nichts.
- **[Fehlende Progress-Events kurz vor Ende]** Wenn der letzte Chunk in <1 s durchläuft, bleibt die Anzeige evtl. bei 99 % stehen bis `onSuccess`. Mitigation: `onSuccess` setzt bereits `setProgress(100)` (bestehendes Verhalten).
- **[Sliding-Window bei sehr langsamer Verbindung]** Wenn <10 s vergehen zwischen Samples (was nur bei stallendem Upload passiert), enthält das Fenster wenige Punkte und die Rate ist ungenau. Der `dt > 0.5`-Guard verhindert Division-durch-fast-Null; die Restzeit bleibt in dem Fall bei ihrem vorigen Wert (keine Aktualisierung), was eine ehrliche UX ist.
- **[Ref-Cleanup zwischen zwei Uploads]** Wenn ein Nutzer zwei Videos hintereinander hochlädt ohne die Seite neu zu laden, müssen die Refs zurückgesetzt sein. Das wird in `startTus` explizit vor `new tus.Upload(...)` gemacht, gleich für `handleResume`.

## Test-Anforderungen

Frontend-only Change — keine neuen HTTP-Routen. Vitest-Contract für die Throttle-Invariante:

| Testfall | Erwartung |
|---|---|
| 100 rasche `onProgress`-Aufrufe in 100 ms | genau 1× `setProgress`, genau 1× `setRemaining` |
| `onProgress`-Aufrufe mit gleichem gerundeten `pct` | `setProgress` wird **nicht** erneut gerufen |
| Sliding-Window nach 15 s | erste Samples werden verworfen (Länge stabil bei ~10 Einträgen bei 1/s Sampling) |
| Reset zwischen zwei Uploads | Refs sind vor `upload.start()` frisch |

**Kein E2E-Test nötig** — Playwright/Chromium würde die Throttle-Logik zwar korrekt ausführen, aber die Speedmessung im Test wäre Flaky (Netzwerk-Simulation im Headless-Chrome ist nicht deterministisch).

## Verification-Pfad (nach Merge, vor Archivierung)

1. `pnpm -C web test` grün (inkl. neuem Throttle-Contract)
2. Manuell: 500-MB-Testvideo hochladen und Chrome DevTools → Performance → Record. Die `commit`-Gelben Blöcke sollten pro Sekunde auf max. 1 einbrechen.
3. Optional: Realen 2-GB-Upload messen und mit vorherigem Baseline vergleichen. Nicht blockierend für Archivierung (Netzwerk-Variabilität).
