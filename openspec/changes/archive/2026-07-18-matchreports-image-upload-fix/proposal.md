## Why

Der Bild-Upload im Spielbericht-Formular (`/spielberichte/{id}`) scheitert für Nutzer:innen mit typischen Handy-Fotos **stumm**: Kamera-JPGs sind heute regelmäßig 8–15 MB, der Server-Cap liegt bei 8 MB (`maxImageBytes` in `internal/matchreports/images.go`) und `handleUpload` in `web/src/pages/MatchReportFormPage.tsx` hat weder Client-Downscale (obwohl `web/src/lib/imageCompress.ts` existiert und in `ChatPage` bereits benutzt wird) noch einen `catch`-Zweig — der 400-Fehler landet in einer Unhandled Promise Rejection und die UI stellt „Lade…" aus, ohne dass der User erfährt, dass der Upload abgelehnt wurde. Zusätzlich ist der Auswahl-Flow für Presseteam-User umständlich: pro Foto einzeln „Bild wählen" klicken, warten, wiederholen — bei 5–10 Bildern pro Bericht (der Regelfall) ist das eine spürbare UX-Baustelle.

## What Changes

- **Client-Downscale vor dem Upload:** `handleUpload` nutzt `compressImage()` mit JPEG-only Output (Server-Whitelist ist `image/jpeg`+`image/png`) — Zielgröße 1 MB, längste Kante 1920 px, analog zum bereits produktiven Chat-Bild-Flow.
- **`compressImage()` bekommt optionalen `formats`-Parameter,** damit Aufrufer die MIME-Reihenfolge steuern können (Default bleibt `[webp, jpeg]` für Chat; Spielbericht übergibt `[jpeg]`). Kein Breaking Change für bestehende Aufrufer.
- **Sichtbare Fehleranzeige:** gescheiterte Uploads erzeugen eine Inline-Fehlermeldung im Bilder-Bereich mit dem Server-`error`-Code (übersetzt: `too_many_images`, `unsupported_mime`, `image_too_large`, `bad_multipart`, sonstige). Kein Silent-Fail mehr.
- **Multi-Select im File-Picker:** Das `<input type="file">` bekommt `multiple`; ausgewählte Dateien werden **sequenziell** hochgeladen (nicht parallel — VPS mit 1 GB RAM soll nicht in `ParseMultipartForm` mit 10× 8 MB parallel gepuffert werden). Fortschritt (`x/y`) wird im Button angezeigt.
- **Client-seitiges Vorab-Cap:** Übersteigt die aktuelle Bilder-Anzahl + Auswahl das Gesamt-Limit von 10, wird die Auswahl **vorab** getrimmt und eine klare Meldung angezeigt („Nur die ersten N Bilder werden hochgeladen — Limit 10 erreicht"). Der Server-Cap von 10 (`MaxImages`) bleibt der Backstop und wird per bestehender 400-Response weiter durchgesetzt — es gibt kein neues Serververhalten, nur eine bessere UX auf dem Client.
- **Kein Backend-Change:** Der bestehende `POST /api/match-reports/{id}/images`-Endpoint bleibt Single-File pro Request; der Client löst Multi-Select durch mehrere sequentielle Requests auf. Damit bleiben Auth-/State-/Consent-Gates und die bestehende Test-Matrix intakt.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `match-reports`: neue Requirements zum Multi-Select-Upload-Verhalten des Clients und zur sichtbaren Fehleranzeige. Kern-Anforderung „Bilder anhängen mit Limit 10" (Server-Route) bleibt unverändert; die Delta ergänzt Client-Verhalten, Fehler-Sichtbarkeit und das Vorab-Trimmen.

## Impact

- **Frontend:**
  - `web/src/pages/MatchReportFormPage.tsx` — `ImagesSection`/`handleUpload` (Multi-Select, sequenzieller Loop, Fortschritt, Fehleranzeige, Vorab-Trim).
  - `web/src/lib/imageCompress.ts` — optionaler `formats`-Parameter (rückwärtskompatibel; Default unverändert).
- **Backend:** keine API-Änderungen, keine Migration, keine neuen Env-Vars. `internal/matchreports/images.go` bleibt unverändert — Client-Downscale drückt die typische Payload deutlich unter 1 MB, der bestehende 8 MB-Backstop und `MaxImages=10` bleiben als Server-Guard.
- **Tests:** neue Vitest-Suites für `compressImage(formats=[jpeg])` (kein WebP im Output) und `handleUpload` (Multi-Select, Trim-Verhalten, Fehleranzeige). Bestehende Go-Handler-Tests werden nicht verändert.
- **RAM/Perf (VPS 1 GB):** sequenzieller Upload statt parallel + 1 MB-Ziel-Payload → geringere Peak-Memory im Backend als heute (bei einem 8 MB-Foto).
- **SSE:** unverändert — `POST /images` broadcastet weiterhin `match-report-event`, jeder Einzel-Upload triggert ein Reload im Formular.
