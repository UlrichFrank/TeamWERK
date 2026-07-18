## 1. `compressImage`-Signatur erweitern

- [x] 1.1 `web/src/lib/imageCompress.ts`: zweiten Parameter zu `opts?: { targetBytes?: number; maxEdge?: number; formats?: {mime, ext}[] }` migrieren, Defaults beibehalten (`TARGET_BYTES`, `MAX_EDGE`, `[{webp}, {jpeg}]`). Loop respektiert `opts.formats`.
- [x] 1.2 Rückwärtskompatibilität sicherstellen: Aufruf ohne zweites Argument muss identisch zum bisherigen Verhalten laufen. Alte Aufrufer in `web/src/pages/ChatPage.tsx` (Zeilen 751, 2363) bleiben unverändert und funktionieren weiter.
- [x] 1.3 Vitest `web/src/lib/imageCompress.test.ts` ergänzen:
   - „JPEG-only output": `compressImage(file, {formats:[{mime:"image/jpeg", ext:".jpg"}]})` liefert `fileName` mit `.jpg`-Endung.
   - „Default unchanged": `compressImage(file)` liefert `.webp` oder `.jpg` (Reihenfolge des ersten passenden).
   - „Kleine Datei wird durchgereicht": File < 1 MB bleibt Blob-identisch.

## 2. `MatchReportFormPage.tsx` — Handler & UI umbauen

- [x] 2.1 `ImagesSection`: `<input type="file" ... multiple onChange={handleUpload} />` — `multiple`-Attribut ergänzen. Kein `capture`.
- [x] 2.2 `handleUpload`: aus `e.target.files` ein Array bilden. Wenn leer → return. Sofort `e.target.value = ''` (damit dieselbe Auswahl neu getriggert werden kann).
- [x] 2.3 Vorab-Trim: `remaining = 10 - props.images.length`. Wenn `files.length > remaining`, Array auf `remaining` kürzen und Info-Meldung im lokalen State setzen („Nur die ersten N Bilder werden hochgeladen — Limit 10 erreicht").
- [x] 2.4 Fehler-State: `useState<{name, reason}[]>([])`. Beim Start des Uploads leeren.
- [x] 2.5 Fortschritts-State: `useState<{done, total}>({done:0, total:0})`. Button-Label zeigt `Lade ${done+1}/${total}…` während Loop.
- [x] 2.6 Sequenzieller Loop: `for (const [i, file] of files.entries()) { setProgress({done:i, total:files.length}); await uploadOne(file) }`.
- [x] 2.7 `uploadOne(file)`: 
   1. `compressImage(file, {formats:[{mime:"image/jpeg", ext:".jpg"}]})` — bei Exception die Original-Datei nutzen (Fallback bleibt).
   2. `FormData` mit `file`+`caption:""` bauen (`form.append("file", blob, fileName)`).
   3. `api.post` mit `try/catch`; im `catch` Fehler übersetzen und in `errors`-State pushen.
   4. Bei Erfolg **kein** `onChange()` sofort — sondern erst nach dem Loop einmalig `onChange()` (spart 10× Reload bei Multi-Upload).
- [x] 2.8 Fehler-Übersetzung: Helper `translateUploadError(status:number, error:string, filename:string) → string` nach der in `design.md` §6 spezifizierten Tabelle. Netzfehler (kein `response`) → generischer Text.
- [x] 2.9 Nach dem Loop: `props.onChange()` einmal aufrufen (löst `load()` im Parent), `setUploading(false)`, `setProgress({done:0,total:0})`.
- [x] 2.10 UI: unter dem Bilder-Grid `errors.length > 0` als `brand-danger-light`-Card rendern (Liste `<ul>` mit `filename — reason`). Info-Meldung (Trim) als eigene `brand-info/10`-Card darüber.

## 3. Vitest — `MatchReportFormPage` Multi-Upload-Verhalten

- [x] 3.1 Test-Setup: `web/src/pages/__tests__/MatchReportFormPage.upload.test.tsx` mit `msw` (bereits im Projekt genutzt? falls nicht: `vi.spyOn(api, 'post')`).
- [x] 3.2 „3 Files erfolgreich": Mock `api.post` liefert 201, `handleUpload` mit 3 Files → `api.post` genau 3× aufgerufen, `onChange` genau 1× am Ende.
- [x] 3.3 „Trim auf Cap": Bericht mit 8 Bildern, 5 Files ausgewählt → `api.post` nur 2× aufgerufen (10−8), Info-Meldung sichtbar.
- [x] 3.4 „Cap erreicht": Bericht mit 10 Bildern → „Bild wählen"-Label wird nicht gerendert.
- [x] 3.5 „Fehler-Sammlung": 2. File liefert 400 `unsupported_mime`, 1. und 3. je 201 → nach Loop steht 1 Fehlereintrag mit dem Dateinamen und Text „Format nicht unterstützt (nur JPG/PNG)" im DOM; `api.post` wurde 3× aufgerufen.
- [x] 3.6 „Netzfehler": Mock `api.post` throwt `new Error('Network Error')` ohne `response` → Fehler-Text „Upload fehlgeschlagen — bitte erneut versuchen".

## 4. Regressions-Check am Bestehenden

- [x] 4.1 `pnpm -C web build` läuft grün (TypeScript).
- [x] 4.2 `pnpm -C web test` läuft grün (bestehende `ChatPage`-Snapshots/Uploads unverändert). 71/71 Files, 592/592 Tests.
- [x] 4.3 `pnpm -C web lint` läuft grün (0 errors; nur 2 pre-existing Warnungen in `ImageTile` — nicht Teil dieser Änderung).
- [x] 4.4 `go test ./...` läuft grün (1445 Tests in 47 Packages).
- [x] 4.5 ~~Manueller Smoke-Test lokal~~ — übersprungen: die aufgeführten Verhaltensweisen (Multi-Select, Trim auf Cap, Fehler-Card mit Dateiname, Netzfehler) sind durch die Vitest-Suites 3.2–3.6 automatisiert abgedeckt; kein separater visueller Check durchgeführt.
   - Draft anlegen (`/spielberichte` → „Bericht schreiben").
   - 3 Bilder auf einmal auswählen → alle 3 erscheinen als Kacheln.
   - Weitere 8 Bilder auswählen → Trim-Info sichtbar, nur 7 werden geladen (bis Cap 10).
   - HEIC-Datei (oder eine `.txt` umbenennen zu `.jpg`) → Fehlermeldung mit Dateiname sichtbar.

## 5. Proposal-Validierung & Archivierung (nach Merge)

- [x] 5.1 `openspec validate matchreports-image-upload-fix --strict` läuft grün.
- [x] 5.2 `/verify-change` durchlaufen (Build/Test/Lint + Projekt-Invarianten).
- [x] 5.3 Change archiviert — bewusst **ohne** Prod-Deploy (Deploy separat einzuplanen; die Spec-Deltas gelten ab Merge auf `main`).
