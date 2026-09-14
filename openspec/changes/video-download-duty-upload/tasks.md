## 1. Datenbank

- [x] 1.1 Migration `internal/db/migrations/064_duty_type_video_upload.up.sql`: `ALTER TABLE duty_types ADD COLUMN grants_video_upload INTEGER NOT NULL DEFAULT 0`
- [x] 1.2 `internal/db/migrations/064_duty_type_video_upload.down.sql` (SQLite: Spalte per Tabellen-Rebuild entfernen, analog vorhandener `.down.sql`-Migrationen mit `ALTER TABLE … DROP COLUMN` bzw. dem im Projekt üblichen Muster)

## 2. Backend: Upload-Berechtigung (dienst-basiert)

- [x] 2.1 `internal/videos/access.go`: neue Funktion `CanUploadForGameViaDuty(claims *auth.Claims, gameID int) (bool, error)` gemäß design.md Entscheidung 2 (Join über `duty_assignments`/`duty_slots`/`duty_types`, kein Status-Filter, kein Team-Abgleich)
- [x] 2.2 `internal/videos/upload.go` (`CreateUpload`): fällt auf `CanUploadForGameViaDuty` zurück, wenn `CanUploadToTeam` false liefert UND `req.GameID != nil`; bei fehlender Berechtigung weiterhin HTTP 403. Zusätzlich (Korrektur beim Implementieren, nicht in design.md vorhergesehen): `POST /api/videos` UND der tus-Upload-Mount lagen bisher im Router-Tier `RequireClubFunction("vorstand","trainer","sportliche_leitung")` — das hätte jeden dienst-berechtigten Nutzer ohne diese Vereinsfunktionen schon vor dem Handler mit 403 blockiert. Beide Routen in `internal/app/router.go` ins Authenticated-Tier verschoben (Autorisierung sitzt jetzt vollständig im Handler bzw. in `preUploadCreate`, das die tus-Session ohnehin unabhängig an `created_by` bindet).
- [x] 2.3 Go-Test: dienst-berechtigter Upload für das zugewiesene Spiel gelingt (201) — `TestCreateUpload_DutyBasedForAssignedGame`
- [x] 2.4 Go-Test: Dienst-Zuweisung für ein anderes Spiel → 403 — `TestCreateUpload_DutyForDifferentGameForbidden`
- [x] 2.5 Go-Test: dienst-basierte Anfrage ohne `game_id` → 403 — `TestCreateUpload_DutyBasedWithoutGameIDForbidden`
- [x] 2.6 Go-Test: Zuweisung auf Diensttyp mit `grants_video_upload=0` → 403 — `TestCreateUpload_DutyOnNonGrantingTypeForbidden`
- [x] 2.7 Go-Test: bestehende Rollen-Pfade (Trainer/sportl. Leitung/Vorstand/Admin) bleiben unverändert grün — komplette bestehende `upload_test.go`-Suite grün (inkl. angepasstem `newUploadServer`, das jetzt wie die Produktion ohne `RequireClubFunction` mountet); zusätzlich Unit-Tests `TestCanUploadForGameViaDuty` in `access_test.go`

## 3. Backend: Eigene upload-berechtigte Spiele

- [x] 3.1 Neuer Handler `internal/videos/eligible_games.go`: `GET /api/videos/upload-eligible-games` — vereinigt Rollen-Pfad (Spiele der Teams mit `CanUploadToTeam`) und Dienst-Pfad (siehe design.md Entscheidung 3). Response-Form: `{ game_ids: number[] }`.
- [x] 3.2 Route in `internal/app/router.go` im Authenticated-Tier registriert (direkt neben den übrigen Video-Routen)
- [x] 3.3 Go-Test: Nutzer mit reiner Dienst-Berechtigung sieht nur sein zugewiesenes Spiel — `TestEligibleGames_DutyOnlyUserSeesOnlyAssignedGame`
- [x] 3.4 Go-Test: Trainer sieht alle Spiele seines Teams unabhängig von Dienst-Zuweisungen — `TestEligibleGames_TrainerSeesAllTeamGames`
- [x] 3.5 Go-Test: Nutzer ohne jede Berechtigung erhält leere Liste mit HTTP 200 (kein 403) — `TestEligibleGames_NoPermissionYieldsEmptyList`

## 4. Backend: Download-Endpoint

- [x] 4.1 `internal/videos/paths.go`: `DownloadTempDir(root)`-Helper für Temp-Verzeichnis (`{root}/tmp/`); Temp-Dateiname-Generierung über `os.CreateTemp` (Pattern `download-{id}-*.mp4` / `concat-*.txt`) direkt im Handler
- [x] 4.2 `readManifestSegmentOrder` (`download.go`) parst `index.m3u8` einer Rendition in Segment-Reihenfolge (Manifest-Reihenfolge, nicht Verzeichnis-Sortierung — siehe design.md Context)
- [x] 4.3 Neuer Handler `internal/videos/download.go`: `GET /api/videos/{id}/download` — `CanViewVideo`-Prüfung (403/404 wie `Get`), `status != 'ready'` → 409, Disk-Guard via `RequireFreeBytes` (507 bei Unterschreitung), ffmpeg-Concat-Remux (`-f concat -safe 0 -c copy -movflags +faststart`) in Temp-Datei (injizierbare `remux`-Naht auf `Handler`, analog `Worker.transcode`), bei Erfolg Stream mit `Content-Type: video/mp4`, `Content-Disposition: attachment`, `Content-Length`; bei ffmpeg-Fehler HTTP 500 ohne Teilantwort; Temp-Dateien in jedem Fall per `defer os.Remove` gelöscht
- [x] 4.4 Dateinamen-Sanitisierung (`sanitizeDownloadFilename`: Unicode-Buchstaben/Ziffern/Leerzeichen/Punkt/Bindestrich/Unterstrich erlaubt, Rest ersetzt; leer → `video-{id}.mp4`)
- [x] 4.5 Route in `internal/app/router.go` registriert (Authenticated-Tier wie `GET /api/videos/{id}`, außerhalb der `StreamTokenMiddleware`)
- [x] 4.6 `internal/permissions/object_matrix_test.go`: Fixture-Eintrag für `GET /api/videos/{id}/download` ergänzt (analog `GET /api/videos/{id}/play`, `noOwnerProbe: true`); `internal/permissions/matrix_test.go` (Tier-Matrix) ebenfalls ergänzt (`POST /api/videos`-Erwartung von `exVorstandTrainer` auf `exAuth` korrigiert, neue Zeilen für Download + eligible-games)
- [x] 4.7 Go-Test: berechtigter Nutzer, Video `status='ready'` → 200, korrekte Header, vollständiger Inhalt — `TestDownload_HappyPath` (nutzt injizierte Fake-Remux-Naht `h.remux`, echte ffmpeg-Invocation ist Aufgabe 4.12/manueller Smoke-Test, s. u.)
- [x] 4.8 Go-Test: nicht berechtigter Nutzer → derselbe Statuscode wie `GET /api/videos/{id}` für diesen Nutzer — `TestDownload_ForbiddenUserGetsSameStatusAsGet` (vergleicht beide Responses direkt, beide 404)
- [x] 4.9 Go-Test: Video nicht `status='ready'` → 409, kein ffmpeg-Aufruf — `TestDownload_NotReadyIsConflict` (zählt Remux-Aufrufe über `countingRemux`)
- [x] 4.10 Go-Test: ffmpeg-Fehler → 500, keine Teilantwort (Content-Type bleibt nicht video/mp4), Temp-Verzeichnis danach leer — `TestDownload_RemuxFailureIsInternalError`
- [x] 4.11 Go-Test: Disk-Guard unterschreitet → 507, kein ffmpeg-Aufruf — `TestDownload_InsufficientStorage`
- [x] 4.12 Scheduler-Job `internal/scheduler.cleanStaleVideoDownloadTemp` (inline, kein Import von `internal/videos` — Architektur-Test) löscht Dateien in `{root}/tmp/` älter als 1 h (Cutoff final: 1 h, kurzlebig wie in design.md skizziert, nicht 24 h wie bei tus-Sessions), in `Run()` registriert; Go-Tests `TestCleanStaleVideoDownloadTemp`/`_MissingDirIsNoError`

## 5. Backend: Admin-Flag für Diensttypen

- [x] 5.1 `internal/duties/handler.go`: Create/Update-Duty-Type-Requests um `grants_video_upload bool` erweitert, in INSERT/UPDATE übernommen (analog `end_at_next_duty`)
- [x] 5.2 `ListTypes`-Response um `grants_video_upload` erweitert
- [x] 5.3 Go-Tests: `TestCreateType_GrantsVideoUploadIsPersisted`, `TestCreateType_GrantsVideoUploadDefaultsFalse`, `TestUpdateType_GrantsVideoUploadRoundtrip` (`internal/duties/duty_type_video_upload_test.go`)

## 6. Web-Frontend: Admin-UI

- [x] 6.1 `web/src/pages/AdminDutyTypesPage.tsx`: Checkbox „Berechtigt zum Video-Upload" im Anlage-/Bearbeitungs-Modal (unabhängig vom Zeit-Modus sichtbar, anders als „Endet spätestens bei Ablösung"), `brand-*`-Tokens, keine neuen Raw-Farben
- [x] 6.2 Vitest: `AdminDutyTypesPage.videoUpload.test.tsx` — Sichtbarkeit, Anlegen sendet Feld, Bearbeiten lädt/aktualisiert Bestandswert

## 7. Web-Frontend: Download-Button

- [x] 7.1 `web/src/pages/VideoDetailPage.tsx`: Download-Button neben `CastButton` (an dieselbe `masterURL`-Bedingung gekoppelt wie CastButton — beide erscheinen erst, wenn der Player tatsächlich initialisiert ist, was `status==='ready'` bereits voraussetzt), `api.get('/videos/{id}/download', {responseType:'blob'})` + `URL.createObjectURL` + programmatischer `<a download>`-Klick (Muster aus `DutyExportModal.tsx` übernommen), `lucide-react`-Icon (`Download`)
- [x] 7.2 Ladezustand (`downloading`) + Fehlerbehandlung über dasselbe `error`-State/-Anzeige wie Wiedergabefehler im `VideoPlayer`
- [x] 7.3 Vitest (`VideoDetailPage.test.tsx`, neues Describe „Download-Button"): Button löst den erwarteten Blob-Request aus, zeigt Fehlermeldung bei fehlgeschlagenem Download

## 8. Desktop-Tool (`tools/video-encoder`)

- [x] 8.1 `internal/client/client.go`: `ActiveSeasonID` auf `GET /api/seasons/active` umgestellt (404 → `ErrNoActiveSeason`; `ErrNoUploadPermission` komplett entfernt, da serverseitig kein 403 mehr an dieser Stelle produziert wird)
- [x] 8.2 Neue Methode `EligibleGameIDs(ctx) (map[int]bool, error)` für `GET /api/videos/upload-eligible-games`
- [x] 8.3 UI-Schicht (`ui.go`): Spielauswahl filtert **immer** auf `eligibleGames` — kein gesonderter Rollen-Check nötig, weil der Server-Endpoint die Rollen-Berechtigung (alle Spiele der eigenen Teams) bereits mit den Dienst-Zuweisungen vereinigt; eine leere gefilterte Liste zeigt eine erklärende Statuszeile statt eines rohen 403 beim Absenden
- [x] 8.4 Client-Tests (`client_test.go`): `TestAPI_CreateVideoSeasonsGames` auf `/api/seasons/active` umgestellt (404-Fall statt 403-Fall), neuer Test `TestEligibleGameIDs`
- [x] 8.5 UI-Tests: bewusst ausgelassen — das Projekt hat für `ui.go` (Fyne) keine bestehende Test-Infrastruktur (nur `internal/client`/`internal/encode`/… werden unit-getestet, siehe `docs/agent/02-workflow.md`); die testbare Logik (Filterung der Spiel-IDs) steckt vollständig in der bereits getesteten `EligibleGameIDs`

## 9. Dokumentation

- [x] 9.1 Zwei neue Gotcha-Einträge in `docs/agent/06-gotchas.md`: „Video-Upload-Berechtigung über den Dienst 'Video'" (1:1 Dienst↔Spiel, unabhängig von `CanViewVideo`, Router-Tier-Falle) und „Video-Download" (Manifest- vs. Verzeichnis-Reihenfolge, Temp-Datei-Umweg statt Direkt-Pipe)
- [x] 9.2 Grün: `go build ./...`, `go vet ./...`, `golangci-lint run ./...` (0 issues, nach Fix des `errcheck`-Fundes bei `io.Copy`), `go test ./...` (Backend, alle Pakete), `pnpm -C web run build`, `pnpm -C web exec eslint .` (0 Fehler, nur bestehende Warnings), `pnpm -C web exec vitest run` (1211 Tests, 1 vorbestehender/unabhängiger Flake in `VaultContext.test.tsx` — isoliert grün, Datei von diesem Change nicht berührt), `openspec validate video-download-duty-upload --strict` (valid). `tools/video-encoder`: `go build ./...`, `go test ./internal/...` grün.
