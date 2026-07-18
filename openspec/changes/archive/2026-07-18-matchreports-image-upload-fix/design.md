## Context

Aktueller Bild-Upload-Pfad im Spielbericht:

```
User klickt "Bild wählen" (Label)
  → hidden <input type="file" accept="image/jpeg,image/png"> öffnet Picker
  → onChange: handleUpload(e.target.files[0])   ← nur EINE Datei
     → FormData("file", file)                    ← kein Compress
     → api.post("/match-reports/{id}/images", form)
        → try/finally, KEIN catch                ← Silent-Fail
```

**Server** (`internal/matchreports/images.go`):
- `maxImageBytes = 8 << 20` (8 MB pro Datei).
- `MaxImages = 10` pro Bericht — Zähler via `SELECT COUNT(*) FROM match_report_images WHERE report_id=?`.
- Whitelist: `image/jpeg`, `image/png`.
- Fehler: `400 image_too_large`, `400 unsupported_mime`, `400 too_many_images`, `400 bad_multipart`, `409 in_progress/already_published`.

**Bereits existierender Baustein**: `web/src/lib/imageCompress.ts` (Ziel 1 MB, längste Kante 1920 px, Fallback-Reihenfolge `[webp, jpeg]`) wird von `web/src/pages/ChatPage.tsx` genutzt (`uploadImage`, Zeile 749).

Warum jetzt: 3 Berichte aus dem Presseteam-Test-Run haben null Bilder — bei allen bricht der Upload stumm ab, Nutzer:innen wechseln entnervt zurück auf die alte Homepage-Redaktion.

## Goals / Non-Goals

**Goals:**
- Handy-Fotos (10–15 MB) laden zuverlässig hoch, ohne dass die/der Nutzer:in etwas an ihrem/seinem Bild ändern muss.
- Multi-Select: 10 Bilder in einem Rutsch auswählen und die App erledigt den Rest.
- Kein Silent-Fail mehr — jede abgelehnte Datei wird namentlich mit Grund angezeigt.
- Der Server bleibt genau wie er ist: `POST /images` nimmt eine Datei, Ergebnis wird per SSE broadcastet, State-Matrix unverändert.

**Non-Goals:**
- Kein Multi-File-Endpoint im Backend — ein neuer Endpoint müsste eigene State-/Consent-Gates + Testmatrix bekommen, das Kosten-/Nutzen-Verhältnis für „max 10" ist nicht da.
- Kein Reorder, kein Caption-Edit, kein Drag-and-Drop in dieser Runde (bestehende Baustellen, gehören in eigene Proposals).
- Kein Progress-Bar pro Datei — nur `x/y` im Button reicht (mobil primary target).
- Kein parallelisiertes Uploaden — VPS 1 GB RAM, `ParseMultipartForm` puffert bis 9 MB pro Request; 10 parallele Uploads = 90 MB peak nur für Multipart-Buffer + Bild-Kopien in `os.Create`.

## Decisions

### 1. Compress-Format: JPEG-only für Match-Reports

**Entscheidung:** `handleUpload` ruft `compressImage(file, { formats: [{mime:"image/jpeg", ext:".jpg"}] })`. Der Server akzeptiert nur `image/jpeg` + `image/png`; würde `compressImage` (Default `[webp, jpeg]`) WebP wählen, würde der Server mit `400 unsupported_mime` ablehnen.

**Alternativen erwogen:**
- **Server um WebP erweitern.** Verlockend (bessere Kompression), aber die Publisher-Payload landet in TYPO3, und dessen Bild-Pipeline verlässt sich auf JPEG/PNG. Ein Format-Wechsel wäre ein separater Change („WebP in Publisher-Payload").
- **WebP-Ergebnis clientseitig zu JPEG konvertieren, wenn WebP kleiner.** Zusätzliche Komplexität ohne echten Nutzen — JPEG mit `q=0.85` an 1920 px Kante ist für Handballfotos < 1 MB. Verworfen.

### 2. `compressImage(file, opts?)` mit `opts.formats?: {mime,ext}[]`

**Entscheidung:** Der zweite Parameter von `compressImage` wird zu einem Optionen-Objekt (rückwärtskompatibel — bestehende Aufrufer geben nur `file` und behalten den Default). Nur `formats` wird ergänzt. `targetBytes` und `maxEdge` migrieren als Options-Keys (mit Defaults), damit die API einheitlich ist.

**Alternative:** Separates `compressImageJpeg(file)` als Convenience-Wrapper. Verworfen, weil das den Kern-Loop dupliziert oder eine interne Funktion exportieren müsste — der Options-Parameter ist kleiner.

### 3. Sequenzieller Upload-Loop

**Entscheidung:** `for (const file of files) { await uploadOne(file) }`. Nach jedem Upload wird das `report.images.length` aus dem letzten `props.onChange()`-Reload noch **nicht** im lokalen Loop-State sein — deshalb zieht der Loop den Zähler lokal mit (`uploadedCount`), damit der Vorab-Trim (siehe unten) stabil ist.

**Alternativen erwogen:**
- **`Promise.all(files.map(uploadOne))`.** Verworfen wegen Peak-Memory (siehe Non-Goal).
- **Concurrency 2–3.** Möglich, aber `pnpm run build`-Ergebnis: `ParseMultipartForm` schreibt Multipart-Body-Chunks > 32 MB auf Disk unter `os.TempDir()`. Auf dem VPS ist das `/tmp` in tmpfs → doch wieder RAM. Sequenz ist die einfachste sichere Antwort.

### 4. Vorab-Trim clientseitig, Server-Cap bleibt Backstop

**Entscheidung:** Beim Multi-Select wird geprüft, wie viele Bilder noch reinpassen (`remaining = 10 - report.images.length`) und die Auswahl auf `remaining` gekürzt, **bevor** irgendein Upload startet. Ein sichtbarer Hinweis nennt die verworfene Anzahl. Der Server-Return `too_many_images` bleibt der Backstop für Race-Cases (zweiter Autor lädt parallel, oder der `report.images`-State ist im Formular veraltet).

**Rationale:** Ohne Vorab-Trim würde der Loop bei Datei 6 (von 10 ausgewählten, wenn schon 5 im Bericht liegen) mit dem 6. Upload starten, den 400-Fehler bekommen und im UI dann eine Fehlerliste „6 Bilder verworfen — zu viele" zeigen — ärgerlich, weil der Upload-Traffic vergeblich war. Der Trim spart Zeit + Traffic.

### 5. Fehleranzeige: pro-Datei-Sammler statt globales Toast

**Entscheidung:** `ImagesSection` bekommt einen lokalen State `uploadErrors: {name: string, reason: string}[]`. Bei jedem Fehler wird ein Eintrag angehängt. Anzeige als Inline-Alert unterhalb des Buttons (`brand-danger-light` Card):

```
Nicht hochgeladen:
  • IMG_4321.HEIC — Format nicht unterstützt (nur JPG/PNG)
  • IMG_4322.JPG — Datei ist zu groß nach Verkleinerung
```

Wird beim nächsten Upload-Klick zurückgesetzt (`setUploadErrors([])`).

**Alternativen erwogen:**
- **Toast**: verschwindet nach 4 s → Nutzer:in weiß danach nicht mehr, welches Bild fehlt. Bei 10 Files ist das relevant.
- **`alert()`**: hässlich, blockiert, unbrauchbar auf iOS-PWA.

### 6. Server-Fehler → deutsche User-Meldung

**Mapping** (in `MatchReportFormPage.tsx` lokal, keine i18n-Infrastruktur nötig):

| Server-`error`        | User-Text                                        |
|-----------------------|--------------------------------------------------|
| `too_many_images`     | „Limit von 10 Bildern erreicht"                  |
| `unsupported_mime`    | „Format nicht unterstützt (nur JPG/PNG)"         |
| `image_too_large`     | „Datei ist zu groß nach Verkleinerung"           |
| `bad_multipart`       | „Datei konnte nicht gelesen werden"              |
| `in_progress` / `already_published` / `not_found` | „Bericht ist nicht mehr editierbar" |
| _sonstiges_           | „Upload fehlgeschlagen — bitte erneut versuchen" |

### 7. `<input type="file" multiple>` + `capture` **nicht** setzen

**Entscheidung:** Das Input-Element bekommt `multiple`, aber **kein** `capture`-Attribut. `capture="environment"` würde iOS/Android zwingen, die Kamera-App statt der Foto-Bibliothek zu öffnen — für Multi-Select aus der Galerie ist das kontraproduktiv. Nutzer:innen, die frisch fotografieren wollen, kommen über die Foto-App und wählen dann aus der Galerie.

## Risks / Trade-offs

- **Risk:** `compressImage` scheitert bei HEIC-Dateien (iOS-Kamera-Default in HEIF/HEIC-Formaten) — `createImageBitmap` liefert je nach Browser `null` oder eine Exception; der aktuelle Fallback gibt die Datei unverändert zurück → Server lehnt mit `unsupported_mime` ab.  
  **Mitigation:** Die neue Fehleranzeige nennt das Bild explizit („IMG_4321.HEIC — Format nicht unterstützt"). Nutzer:innen können in iOS unter Einstellungen → Kamera → Formate → „Maximale Kompatibilität" wählen, dann sind Fotos JPEG. Kein Server-seitiges HEIC-Handling in dieser Runde (Go-Toolchain hätte libheif als CGo-Abhängigkeit — bricht das „kein CGo"-Prinzip). Follow-up-Kandidat, wenn es sich häuft.

- **Risk:** Sequenzieller Upload von 10 Bildern à ~1 MB bei mobiler 3G-Verbindung dauert 30–60 s — Nutzer:in könnte die Seite verlassen und den Rest der Uploads abwürgen.  
  **Mitigation:** Button-Label zeigt `Lade 4/10…`, damit sichtbar ist, dass etwas passiert. Kein `beforeunload`-Prompt — würde bei jedem Navigate nerven; die bereits geladenen Bilder sind persistiert, Verlust ist auf die noch nicht geladenen begrenzt.

- **Trade-off:** Kein automatisches Retry bei transientem 5xx.  
  **Rationale:** In `handleUpload`-Fehleranzeige steht das Bild — Nutzer:in kann es via „Bild wählen" nochmal auswählen und uploaden. Retry-Logic mit Backoff wäre 2× Code für 2× Test.

- **Trade-off:** `report.images.length` wird zwischen Uploads im lokalen `uploadedCount` mitgezählt, weil `props.onChange()` (löst `load()` im Parent aus) asynchron ist. Sollte der Live-Update-Reload verzögert kommen, wäre die Anzeige im Zwischenschritt bei `report.images.length` = alter Wert. Der Loop-lokale Zähler ist der Source-of-Truth für den Vorab-Trim; die UI aktualisiert sich beim nächsten `load()` sowieso.
