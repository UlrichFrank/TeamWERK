## Why

Ein 9-GB-Video-Upload lief am 2026-07-22 nach 4,5 h nicht durch. Diagnose des Netzwerkpfads:

- Speedtest-Baseline: ~50 Mbit/s Upload
- **Curl-Single-Stream gegen VPS**: 4,37 MB/s ≈ **35 Mbit/s** (WAN-TCP-Overhead ~30%, unvermeidlich)
- **tus-js-client im Browser**: 2–5 MB/s beobachtet (aus abgebrochenen Session-Größen), teils dramatisch schlechter

Der Server ist entlastet (AES-NI, HTTP/2, `proxy_buffering off`, kein per-chunk `fsync`, 82 GB frei). Der Verlust von **35 → 5 Mbit/s** entsteht im Browser.

`onProgress` in `VideoUploadPage.tsx` feuert bei jedem `XHR.upload.progress`-Event (typisch 30–100 Hz) und ruft pro Event **zwei React-`setState`** auf (`setProgress`, `setRemaining`). Bei 144 Chunks × 64 MB pro 9-GB-Upload landet der React-Reconciler auf dem Main-Thread, während derselbe Main-Thread den XHR-Upload speist — klassischer Re-Render-Storm.

Zusatz-Symptom: `startTimeRef` wird nie aktualisiert; die Restzeit-Schätzung ist ein Gesamt-Durchschnitt seit Upload-Start und reagiert nach Aussetzern nur träge.

## What Changes

- `onProgress` in `web/src/pages/VideoUploadPage.tsx` **throttlet** auf max. 1× pro Sekunde (Start- und Resume-Pfad)
- `setProgress` wird nur bei geändertem gerundeten Prozentwert aufgerufen (verhindert Duplicate-Renders bei gleichem `pct`)
- Restzeit-Schätzung basiert auf **Sliding-Window** (letzte ~10 Sekunden) statt „seit Start" — reagiert schneller nach Aussetzern
- Refs (Throttle-Timestamp, Sample-Ring, letzter Prozentwert) werden vor Upload-Start zurückgesetzt
- Ein kleiner Vitest-Contract sichert die Throttle-Invariante („100 rasche onProgress → max 1 setState/s")

## Non-Goals

- **Keine** `parallelUploads` (separater Change; braucht `preUploadCreate`-Anpassung für `Upload-Concat: final` und Disk-Guard-Skalierung)
- **Keine** Chunk-Size-Änderung (separater Change; Trade-off Recovery-Granularität vs. Overhead-Amortisation)
- **Kein** Web-Worker (Overkill für diesen Fix)
- Serverseitige Optimierungen: keine (Server ist entlastet)

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `video-upload`: neue Anforderung „Client-seitige Progress-Throttle" ergänzt die bestehenden Upload-Anforderungen

## Impact

- `web/src/pages/VideoUploadPage.tsx` — `startTus` + `handleResume`: `onProgress` umschreiben (~30 LoC), zusätzliche Refs für Throttle-Timestamp + Sample-Ring
- `web/src/pages/VideoUploadPage.test.tsx` — neuer Test für Throttle-Verhalten (Vitest + `vi.useFakeTimers`)
- Doku: Gotcha in `docs/agent/06-gotchas.md` unter „Video-Upload"-Abschnitt ergänzen (Warum onProgress throttlen)

## Erwarteter Effekt

Wenn die Diagnose stimmt (React-Storm ist Hauptursache des 35 → 5 Mbit/s-Verlusts), sollten Uploads nach dem Fix im Bereich **20–30 Mbit/s** landen — Faktor 4–6× schneller. Bei einem 9-GB-Upload: ~40–70 min statt Stunden.

Falls der Fix keinen sichtbaren Effekt hat, ist der Bottleneck woanders (WebSocket-SSE, Chrome Tab-Throttling, Blob-Read von langsamer Quelle). Dann wäre der nächste Schritt ein Chrome-Performance-Profile eines Real-Uploads. Dieser Change ist trotzdem sinnvoll — die Throttle-Logik ist unabhängig ein Best-Practice.
