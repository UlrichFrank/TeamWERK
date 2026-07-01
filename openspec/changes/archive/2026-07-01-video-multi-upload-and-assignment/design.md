## Context

Videos werden per tus (resumable uploads) hochgeladen. `POST /api/videos` legt vorab eine
`videos`-Zeile mit `status='uploading'` an und liefert die `video_id`; der Client startet
dann eine tus-Session mit Metadata `video_id=<id>`. Der Finish-Hook korreliert die fertige
Datei über diese Metadata (`internal/videos/upload.go`).

Der tus-Client speichert Sessions im localStorage unter einem **Fingerprint**, der per
Default aus Endpoint + Dateiname + Dateityp + Dateigröße gebildet wird — **nicht** aus der
`video_id`. `VideoUploadPage.tsx` bietet über `findPreviousUploads()` einen Resume an; der
Fehler: `handleSubmit` legt eine neue `video_id` an, ruft dann aber
`upload.resumeFromPreviousUpload(resumable)` auf. Bei zwei Uploads derselben Datei matcht
der Fingerprint die alte Session → der frische Upload bespielt die **alte** `video_id`, die
neue Zeile bleibt als Waise auf `uploading`.

DB/Backend erlauben n:1 bereits: `videos.game_id` hat kein `UNIQUE`, der Upload macht ein
reines `INSERT`, und `PATCH /api/videos/{id}` unterstützt `game_id` als Tri-State
(`internal/videos/crud.go`). Nur der Frontend-Resume-Pfad und die fehlende Edit-UI sowie
das fehlende Cleanup hängender Zeilen sind die Lücken.

## Goals / Non-Goals

**Goals:**
- Mehrere Videos pro Termin/Name zuverlässig — kein Session-Hijack durch frische Uploads.
- Spiel-Zuordnung eines Videos nachträglich im UI änder-/entfernbar.
- Keine dauerhaften „Wird hochgeladen"-Geister durch abgebrochene Uploads.

**Non-Goals:**
- Keine Änderung am tus-/Streaming-/Transcoding-Pfad oder am DB-Schema.
- Kein „Video ersetzen"-Feature (Ersetzen bleibt = löschen + neu hochladen).
- Keine Gruppierung/Anzeige mehrerer Videos je Spiel auf der Termin-Detailseite (separat).

## Decisions

**1. Frischer Submit resumt nie.** `handleSubmit`/`startTus` rufen kein
`resumeFromPreviousUpload` mehr auf — der frische Upload startet immer eine neue Session für
die neu angelegte `video_id`. Der Resume bleibt allein dem Button „Upload fortsetzen"
(`handleResume`) vorbehalten, der die alte `video_id` aus `resumable.metadata` behält und
keine neue Zeile anlegt.
- *Alternative:* Fingerprint pro `video_id` schlüsseln. Verworfen: die Resume-Sonde in
  `useEffect([file])` kennt die `video_id` noch nicht (läuft vor dem `POST`), das würde
  „Upload fortsetzen" brechen. Die minimale Änderung (kein Auto-Resume) behebt den Hijack
  vollständig, ohne den Resume-Komfort zu verlieren.
- *Nebeneffekt:* tus überschreibt beim frischen Start den Fingerprint→URL-Eintrag der
  alten Session in localStorage; die alte Partial-Session verwaist und wird serverseitig
  vom bestehenden Stale-Cleanup (>24 h) entfernt.

**2. Zuordnung über bestehenden PATCH.** Keine neue Route. Das Edit-Modal lädt die Spiele
des Video-Teams (`GET /api/games`, clientseitig nach `team_id` gefiltert — wie in
`VideoUploadPage`), zeigt einen Selector inkl. „Kein Spiel zuordnen" und sendet `game_id`
im `PATCH` (Zahl oder `null`). Die vorhandene Tri-State-Logik in `crud.go` bleibt
unverändert; nur die Testabdeckung wird ergänzt.

**3. Cleanup als Inline-Scheduler-Job.** Analog zu `cleanFailedVideoRaw` ein Inline-Job
`failStaleVideoUploads` (Foundation-Package darf `videos` nicht importieren → direktes SQL):
`UPDATE videos SET status='failed', failure_reason='Upload abgebrochen' WHERE
status='uploading' AND created_at < datetime('now','-24 hours')`. Aufruf im Scheduler-Tick.
- *Warum 24 h:* deckt sich mit dem bestehenden Stale-tus-Session-Cutoff; ein legitimer
  2,5-GB-Upload dauert nie annähernd so lange.

## Risks / Trade-offs

- **[Frischer Upload verwirft unterbrochenen Teil-Upload derselben Datei]** → Gewollt: „Neu
  hochladen" bedeutet neues Video; für Fortsetzen gibt es den expliziten Button. Klare
  UX-Trennung.
- **[Zwei Videos mit identischem Auto-Titel je Spiel]** → Akzeptiert; über die editierbaren
  Titel/Beschreibungen unterscheidbar. Kein Blocker für die Kernfunktion.
- **[Cleanup markiert einen extrem langsamen, echten Upload als failed]** → 24-h-Cutoff
  macht das praktisch unmöglich (Hard-Limit 2,5 GB).
