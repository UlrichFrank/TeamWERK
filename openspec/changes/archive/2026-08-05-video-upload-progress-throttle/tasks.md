## 1. Frontend — Throttle in startTus

- [x] 1.1 Refs anlegen: `lastProgressRef` (Timestamp), `samplesRef` (Ring), `lastPctRef` (letzter Prozentwert) — als `useRef` am Anfang der Komponente
- [x] 1.2 In `startTus` vor `new tus.Upload(...)`: Refs zurücksetzen (`lastProgressRef.current = 0`, `samplesRef.current = []`, `lastPctRef.current = -1`)
- [x] 1.3 `onProgress` in `startTus` durch die Throttle-Variante ersetzen: Sample sammeln, 10-s-Window pflegen, bei `now - lastProgressRef < 1000` früh returnen, `setProgress` nur bei Prozent-Änderung, `setRemaining` aus Sliding-Window-Rate
- [x] 1.4 `startTimeRef` entfernen (wird nicht mehr gebraucht) — falls andere Stellen es lesen, `elapsed`-Berechnung raus

## 2. Frontend — Throttle in handleResume

- [x] 2.1 In `handleResume` vor `new tus.Upload(...)`: gleiches Ref-Reset wie in 1.2
- [x] 2.2 `onProgress` in `handleResume` durch identische Throttle-Variante ersetzen (Duplikation bewusst — kein Hook)

## 3. Frontend — Vitest-Contract

- [x] 3.1 Neue Datei `web/src/pages/VideoUploadPage.test.tsx` (falls nicht existent) mit `vi.useFakeTimers`
- [x] 3.2 Test „100 rasche Progress-Events → 1 setState": Komponente rendern, `upload.onProgress` extrahieren, 100× aufrufen innerhalb 100 ms, prüfen dass `setProgress` genau 1× gerufen wurde
- [x] 3.3 Test „gleicher Prozent-Wert löst kein setProgress aus": zwei `onProgress`-Calls im Abstand von 1500 ms mit demselben bytesSent → nur der erste setzt State
- [x] 3.4 Test „Sliding-Window Länge stabil bei kontinuierlichem Sampling über 15 s"
- [x] 3.5 Test „Refs sind nach zweitem `upload.start()` in derselben Session frisch"

## 4. Doku

- [x] 4.1 In `docs/agent/06-gotchas.md` unter dem Video-Upload-Block einen Kurzhinweis einfügen: „`onProgress` in tus-in-Browser MUSS throttlen — jeder Event triggert sonst React-Renders auf demselben Main-Thread, der auch die XHR-Uploads speist, → 5–10× Throughput-Verlust. Siehe `VideoUploadPage.tsx`."

## 5. Verification

- [x] 5.1 `pnpm -C web test` grün
- [x] 5.2 `pnpm -C web build` grün (TypeScript)
- [ ] 5.3 Manueller Testupload eines mittelgroßen Videos (~500 MB) und Chrome-DevTools-Performance-Recording: „Main"-Bereich zeigt maximal 1 React-Commit pro Sekunde während des Uploads _(nur vom Nutzer verifizierbar, nicht durch den Assistant)_
- [x] 5.4 `openspec validate video-upload-progress-throttle --strict` grün
