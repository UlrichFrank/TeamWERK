## Why

Beim Video-Upload lässt sich pro Termin faktisch nur **ein** Video ablegen: Startet man
einen zweiten Upload derselben Datei, adoptiert er die noch offene tus-Session des ersten
(der tus-Fingerprint matcht dateibasiert, nicht pro `video_id`) — nur eine Video-Zeile
bekommt Daten, die andere bleibt als Waise auf `status='uploading'` hängen und erscheint
dauerhaft als „Wird hochgeladen". DB und Backend erlauben mehrere Videos pro Spiel bereits
(kein `UNIQUE` auf `videos.game_id`, `INSERT` ohne UPSERT); der Fehler liegt allein im
Frontend-Resume-Pfad. Zusätzlich ist die Spiel-Zuordnung eines vorhandenen Videos im UI
nicht änderbar, obwohl der Server sie schon unterstützt.

## What Changes

- **Frischer Upload resumt nie eine fremde Session.** `handleSubmit` (Button „Hochladen")
  startet für die neu angelegte `video_id` **immer** eine neue tus-Session; der Resume-Pfad
  bleibt ausschließlich dem expliziten Button „Upload fortsetzen" (`handleResume`)
  vorbehalten, der korrekt die alte `video_id` aus den Session-Metadaten behält. Damit
  koexistieren beliebig viele Videos pro Termin/Name zuverlässig.
- **Spiel-Zuordnung im Bearbeiten-Modal änderbar.** Das Edit-Modal auf der
  Video-Detailseite bekommt einen Spiel-Selector (inkl. „Kein Spiel zuordnen"); die
  Zuordnung wird per `PATCH /api/videos/{id}` als Tri-State (`game_id`: Zahl | `null` |
  weggelassen) gespeichert. Die Backend-Logik existiert bereits, ist aber untested — der
  Test wird ergänzt.
- **Hängende `uploading`-Zeilen werden aufgeräumt.** Ein neuer Scheduler-Job setzt
  Videos, die länger als 24 h auf `status='uploading'` stehen, auf `status='failed'`
  (`failure_reason` = „Upload abgebrochen"), sodass keine Geister-Einträge „Wird
  hochgeladen" bestehen bleiben.

## Capabilities

### New Capabilities
- `video-multi-upload`: Mehrere Videos pro Termin/Spiel zuverlässig hochladbar
  (kein Session-Hijack), Spiel-Zuordnung eines Videos nachträglich änderbar, und
  Bereinigung hängengebliebener Upload-Zeilen.

### Modified Capabilities
<!-- keine bestehende Spec ändert ihre Anforderungen -->

## Impact

- **Frontend:** `web/src/pages/VideoUploadPage.tsx` (kein Resume bei frischem Submit),
  `web/src/pages/VideoDetailPage.tsx` (Spiel-Selector im Edit-Modal, Games laden,
  `game_id` im PATCH).
- **Backend:** `internal/videos/crud.go` (bereits vorhandene `game_id`-Tri-State-Logik —
  nur Testabdeckung ergänzen), `internal/scheduler/scheduler.go` (neuer Inline-Job
  `failStaleVideoUploads`, aufgerufen im Scheduler-Tick).
- **API:** keine neue Route; `PATCH /api/videos/{id}` akzeptiert `game_id` bereits.
- **DB/Migrationen:** keine — Schema unterstützt n:1 bereits (`013_videos.up.sql`).
- **Tests:** `internal/videos/crud_test.go` (PATCH `game_id`), Scheduler-Test für den
  neuen Cleanup-Job, Frontend-Tests für Edit-Modal-Zuordnung und frischen Upload.
