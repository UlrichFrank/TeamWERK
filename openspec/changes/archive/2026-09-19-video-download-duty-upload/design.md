## Context

Siehe proposal.md für Motivation. Relevanter Bestand:

- `internal/videos/access.go`: `CanUploadToTeam(claims, teamID) (bool, error)` — reine Rollen-/Trainer-Prüfung, kennt kein `game_id`. `CanViewVideo(claims, video) (bool, error)` bleibt unverändert (dieser Change rührt nicht daran).
- `internal/videos/upload.go`: `CreateUpload` validiert Referenzen (Team/Saison/Spiel existieren), ruft dann `CanUploadToTeam(claims, req.TeamID)`.
- `internal/videos/paths.go`: nach Transcode existiert nur noch `processed/{id}/master.m3u8` + `{rendition}/index.m3u8` + `seg_NNN.ts`-Segmente (`raw/{id}.mp4` ist gelöscht). Segment-Dateinamen `seg_%03d.ts` (`worker.go`) — ab Segment 1000 vierstellig, **lexikographische Verzeichnis-Sortierung ist deshalb keine verlässliche Abspielreihenfolge**. Die verlässliche Reihenfolge steht in `index.m3u8` (`#EXTINF`-Einträge in Abspielreihenfolge).
- `internal/videos/disk.go`: `RequireFreeBytes(dir, needed, reserved)` — bestehendes Muster für Disk-Guards (Upload nutzt `size_bytes × 2.5 + 2 GiB`).
- ffmpeg/ffprobe sind auf dem VPS installiert und werden bereits per `exec.CommandContext` aus `internal/videos` aufgerufen (`codecs.go`, `worker.go`) — gleiches Muster wiederverwenden.
- Duty-Schema: `duty_types(id, name, …)`, `duty_slots(id, event_date, duty_type_id, team_id NULLABLE, season_id, game_id NULLABLE, …)`, `duty_assignments(id, duty_slot_id, user_id, status[assigned|fulfilled|cash_substitute], …)`.
- Router-Tiers (bestätigt in `internal/app/router.go`): `GET /api/teams` und `GET /api/games` sind bereits Authenticated-Tier (jeder eingeloggte Nutzer). `GET /api/seasons/active` existiert bereits als Authenticated-Tier-Alternative zum Vorstand/Trainer/sL/Kassierer-only `GET /api/seasons`.
- Objektrechte-Matrix (`internal/permissions/object_matrix_test.go`) verlangt für jede neue `{id}`-Route einen Fixture-Erzeuger, sonst ist der Test rot.

## Goals / Non-Goals

**Goals:**
- Upload-Berechtigung um einen dienst-basierten, spielgenauen Pfad erweitern, ohne die bestehende Rollen-Berechtigung zu verändern.
- Download fertiger Videos ermöglichen, ohne dauerhaften zusätzlichen Speicherbedarf und ohne die Streaming-Sichtbarkeit anzufassen.
- Desktop-Tool so anpassen, dass ein Nutzer mit reiner Dienst-Berechtigung das Tool tatsächlich benutzen kann (nicht nur der Server-Endpoint erlaubt es, sondern auch der Client kommt bis dahin).

**Non-Goals:**
- Keine Änderung der Video-Sichtbarkeit (`CanViewVideo`/`userBelongsToTeam`/`visibilityFilter`/`pushRecipients`) — bestätigt vom Anforderer.
- Kein Browser-Upload (bleibt Desktop-Tool-exklusiv, siehe `video-offline-encoding-tool`).
- Kein Wiederaufnehmbarer/Range-Download (HTTP Range-Requests) — einfacher, kompletter Download reicht für den Anwendungsfall.
- Keine rückwirkende Migration bestehender Diensttypen — das neue Flag ist per Default `0`, Vorstand markiert existierende oder neue Diensttypen manuell.

## Decisions

### 1. Neues Flag `duty_types.grants_video_upload` statt Namensabgleich

Migration `064_duty_type_video_upload.up.sql`: `ALTER TABLE duty_types ADD COLUMN grants_video_upload INTEGER NOT NULL DEFAULT 0`. Analog zu bestehenden boolean-artigen Flags auf `duty_types` (`end_at_next_duty`, Migration 054). Admin-UI (`web/src/pages/AdminDutyTypesPage.tsx`, bestehendes Anlage-/Bearbeitungs-Modal) bekommt eine Checkbox „Berechtigt zum Video-Upload"; `internal/duties/handler.go` (Create/Update Duty-Type) übernimmt das Feld wie die übrigen Flags.

Alternative (Namensabgleich auf `duty_types.name = 'Video'`) verworfen: fragil gegen Umbenennung/Tippfehler, nicht auditierbar im Admin-UI, widerspricht dem bestehenden Muster expliziter Flags.

### 2. Upload-Berechtigung: neue Funktion statt Erweiterung der Signatur an allen Call-Sites

`CanUploadToTeam(claims, teamID)` bleibt für die reine Rollen-Prüfung bestehen (wird an anderer Stelle evtl. noch gebraucht, z. B. für „darf grundsätzlich zu diesem Team hochladen" ohne Spielbezug). `CreateUpload` prüft stattdessen neu:

```
ok, err := h.CanUploadToTeam(claims, req.TeamID)
if !ok && req.GameID != nil {
    ok, err = h.CanUploadForGameViaDuty(claims, *req.GameID)
}
```

`CanUploadForGameViaDuty(claims, gameID)` prüft:

```sql
SELECT COUNT(*) FROM duty_assignments da
JOIN duty_slots ds ON ds.id = da.duty_slot_id
JOIN duty_types dt ON dt.id = ds.duty_type_id
WHERE da.user_id = ? AND ds.game_id = ? AND dt.grants_video_upload = 1
```

`status` der Zuweisung wird **nicht** gefiltert (jeder Status zählt — „eingetragen bin/war" deckt sowohl aktuell zugewiesen als auch bereits erledigt ab; `cash_substitute` zählt bewusst mit, weil die Berechtigung an der Zuweisung hängt, nicht an der tatsächlichen Ausführung — wer sich freigekauft hat, hat keinen Anspruch, aber das explizit auszuschließen wäre eine zusätzliche, vom Anforderer nicht verlangte Verschärfung; falls das Vereinsverhalten anders sein soll, ist das eine spätere, separate Entscheidung).

Kein `team_id`-Abgleich in der Query: die Berechtigung hängt an `game_id`, nicht am Team des Dienst-Slots (ein generischer Slot mit `team_id IS NULL` soll genauso zählen wie ein team-gebundener). Referenzielle Validierung (`team_id` gehört zu `game_id`) bleibt Sache der bereits bestehenden Pre-Checks in `CreateUpload`.

### 3. Neuer Endpoint `GET /api/videos/upload-eligible-games`

Authenticated-Tier, liefert `{ game_ids: number[] }` (oder eine kleine Objekt-Liste mit `id`/`team_id`, damit der Client nicht zusätzlich `/api/games` filtern muss — Response-Form final in tasks.md). Vereinigt:
- alle Spiele der Teams, für die `CanUploadToTeam` true liefert (Trainer/sportl. Leitung/Vorstand/Admin),
- alle `game_id`, für die eine Zeile in obiger Dienst-Query existiert.

Dies ist ein reiner Komfort-/Vorauswahl-Endpoint für Client-UIs (Desktop-Tool, potenziell Web). Die eigentliche Autorisierung bleibt serverseitig in `CreateUpload` — dieser Endpoint MUSS nicht 1:1 dieselbe Query sein, darf aber nicht großzügiger filtern als sie (sonst zeigt das Tool ein Spiel an, für das der Upload dann 403 zurückgibt).

### 4. Download: manifest-geordneter Remux in eine temporäre Datei, dann Stream

`GET /api/videos/{id}/download`:
1. Video laden, `CanViewVideo` prüfen (403/404 wie `Get`), `status == 'ready'` prüfen (sonst 409).
2. `index.m3u8` der 720p-Rendition parsen, Segment-Dateinamen in Manifest-Reihenfolge extrahieren (nicht Verzeichnis-Sortierung — s. Context).
3. Disk-Guard: `RequireFreeBytes(root, DirSize(720p-Verzeichnis), 2 GiB)` (Remux-Output ist bei Stream-Copy ungefähr so groß wie die Summe der Segmente) — bei Unterschreitung HTTP 507, kein ffmpeg-Start.
4. ffmpeg-Concat-Demuxer (`-f concat -safe 0 -i <generierte Listendatei>`) mit `-c copy` in eine temporäre Datei unter `{root}/tmp/download-{id}-{random}.mp4` remuxen (`exec.CommandContext` mit Timeout, analog `worker.go`).
5. Bei Erfolg: Datei öffnen, `Content-Type: video/mp4`, `Content-Disposition: attachment; filename="<sanitierter Titel>.mp4"`, `Content-Length` (bekannt, da Datei fertig geschrieben) setzen, per `io.Copy` an die Response streamen.
6. Temp-Datei in jedem Fall (Erfolg wie Fehler) per `defer os.Remove(...)` löschen — „kein dauerhaftes Caching" (Spec-Requirement).
7. Bei ffmpeg-Fehler (Exit-Code ≠ 0): HTTP 500, **kein** Byte der (kaputten) Datei geht an den Client — das ist der Grund für den Umweg über eine Temp-Datei statt direktem Pipe von ffmpeg-stdout in die Response: bei direktem Pipe wären die HTTP-Header (200) schon gesendet, bevor ein Fehler mitten im Stream erkennbar ist, der Client bekäme eine stillschweigend abgeschnittene Datei statt eines Fehlerstatus.

Alternative (direktes Piping von ffmpeg-stdout in die Response, fragmentiertes MP4 via `-movflags frag_keyframe+empty_moov`) verworfen: spart die Temp-Datei, aber verschenkt sauberes Fehlerverhalten (Scenario „Remux schlägt fehl" im Spec verlangt HTTP 500, nicht eine abgeschnittene 200-Antwort) und einen korrekten `Content-Length`-Header. Die Temp-Datei existiert nur für die Dauer eines Requests (Sekunden bis niedrige Minuten bei Stream-Copy) und wird danach entfernt — kein Dauerspeicherbedarf, das Spec-Requirement „kein dauerhaftes Caching" bezieht sich auf genau diesen Unterschied zu „einmal erzeugen und für spätere Downloads aufheben".

Route liegt **außerhalb** der `StreamTokenMiddleware`/`?st=`-Token-Welt (die existiert nur, weil HLS-Player keine Authorization-Header an Segment-Requests anhängen können) — der Download ist ein einzelner Request mit normalem Bearer-Token wie `GET /api/videos/{id}`.

### 5. Desktop-Tool (`tools/video-encoder`)

- `ActiveSeasonID()` ruft künftig `GET /api/seasons/active` (liefert direkt die aktive Saison, kein Filtern über eine Liste nötig) statt `GET /api/seasons` — damit scheitert ein Nutzer ohne Trainer/sL/Vorstand-Funktion nicht mehr an dieser Stelle mit einem irreführenden „kein Upload-Recht".
- Neue Client-Methode `EligibleGameIDs(ctx, seasonID)` ruft `GET /api/videos/upload-eligible-games`.
- Die UI-Schicht filtert die Spielauswahl (aktuell alle vergangenen Spiele des gewählten Teams) auf diese Menge, wenn der Nutzer keine Trainer-/Vorstand-/sL-Funktion hat (kann clientseitig am `role`/`club_functions`-Claim erkannt werden, wie andernorts im Tool bereits gehandhabt) — Details in tasks.md.
- `ErrNoUploadPermission` bleibt als Fallback für den Fall, dass der Nutzer trotz allem kein einziges berechtigtes Spiel hat (leere Auswahl → verständliche Meldung statt eines rohen 403 beim Absenden).

## Risks / Trade-offs

- **[Risk]** ffmpeg-Remux ist CPU/IO-Last auf der 1-GB-RAM-VPS pro Download-Request → **Mitigation**: Stream-Copy (`-c copy`, kein Decode/Encode) ist sehr günstig, vergleichbar mit dem bestehenden Rendition-Repackaging beim Upload (`nice -n 19 ffmpeg`); Prozess läuft ebenfalls mit `nice`.
- **[Risk]** Mehrere gleichzeitige Downloads desselben oder verschiedener großer Videos könnten Disk-I/O und Temp-Speicher kurzzeitig belasten → **Mitigation**: Disk-Guard vor Start (507 statt Absturz), Temp-Dateien werden unmittelbar nach dem Request gelöscht; kein Rate-Limiting in diesem Change (kleiner Verein, seltene Downloads) — bei Bedarf später nachschärfbar.
- **[Risk]** `duty_assignments` ohne Status-Filter zählt auch `cash_substitute` (Freikauf) als Upload-Berechtigung → **Mitigation**: bewusst so gewählt (s. Entscheidung 2), einfach nachträglich einschränkbar, falls das in der Praxis zu Verwirrung führt.
- **[Risk]** Temp-Verzeichnis (`{root}/tmp/`) könnte bei einem Server-Absturz mitten im Remux verwaiste Dateien hinterlassen → **Mitigation**: analog zu verwaisten tus-Sessions (`video-upload`-Spec, tägliches Cleanup) einen kurzen Retention-Cleanup (z. B. Scheduler löscht Dateien in `{root}/tmp/` älter als 1 h) ergänzen — Detail in tasks.md.

## Migration Plan

1. Migration 064 (`duty_types.grants_video_upload`) — additiv, kein Downtime-Risiko, folgt dem Standard-Deploy-Pfad (`make deploy` führt Migrationen automatisch aus).
2. Kein Rollback-Sonderfall nötig (additive Spalte, Default `0` — bestehende Diensttypen bleiben unverändert).
3. Kein Datenbackfill nötig — Vorstand markiert das Flag manuell auf dem existierenden „Video"-Diensttyp nach dem Deploy.

## Open Questions

- Exakte Response-Form von `GET /api/videos/upload-eligible-games` (nur `game_ids` oder angereicherte Objekte) — wird in tasks.md/Implementierung entschieden, ändert nichts an Spec oder Berechtigungslogik.
- Genaues Cleanup-Intervall für `{root}/tmp/` (1 h vs. 24 h wie bei tus-Sessions) — unkritisch, kann beim Implementieren festgelegt werden.
