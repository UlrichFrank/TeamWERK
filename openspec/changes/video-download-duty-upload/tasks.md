## 1. Datenbank

- [ ] 1.1 Migration `internal/db/migrations/064_duty_type_video_upload.up.sql`: `ALTER TABLE duty_types ADD COLUMN grants_video_upload INTEGER NOT NULL DEFAULT 0`
- [ ] 1.2 `internal/db/migrations/064_duty_type_video_upload.down.sql` (SQLite: Spalte per Tabellen-Rebuild entfernen, analog vorhandener `.down.sql`-Migrationen mit `ALTER TABLE … DROP COLUMN` bzw. dem im Projekt üblichen Muster)

## 2. Backend: Upload-Berechtigung (dienst-basiert)

- [ ] 2.1 `internal/videos/access.go`: neue Funktion `CanUploadForGameViaDuty(claims *auth.Claims, gameID int) (bool, error)` gemäß design.md Entscheidung 2 (Join über `duty_assignments`/`duty_slots`/`duty_types`, kein Status-Filter, kein Team-Abgleich)
- [ ] 2.2 `internal/videos/upload.go` (`CreateUpload`): fällt auf `CanUploadForGameViaDuty` zurück, wenn `CanUploadToTeam` false liefert UND `req.GameID != nil`; bei fehlender Berechtigung weiterhin HTTP 403
- [ ] 2.3 Go-Test: dienst-berechtigter Upload für das zugewiesene Spiel gelingt (201)
- [ ] 2.4 Go-Test: Dienst-Zuweisung für ein anderes Spiel → 403
- [ ] 2.5 Go-Test: dienst-basierte Anfrage ohne `game_id` → 403
- [ ] 2.6 Go-Test: Zuweisung auf Diensttyp mit `grants_video_upload=0` → 403
- [ ] 2.7 Go-Test: bestehende Rollen-Pfade (Trainer/sportl. Leitung/Vorstand/Admin) bleiben unverändert grün (Regression bestehender Tests in `upload_test.go`)

## 3. Backend: Eigene upload-berechtigte Spiele

- [ ] 3.1 Neuer Handler `internal/videos/eligible_games.go` (oder Erweiterung von `crud.go`): `GET /api/videos/upload-eligible-games` — vereinigt Rollen-Pfad (Spiele der Teams mit `CanUploadToTeam`) und Dienst-Pfad (siehe design.md Entscheidung 3), Response-Form final beim Implementieren festlegen (z. B. `{ game_ids: number[] }`)
- [ ] 3.2 Route in `internal/app/router.go` im Authenticated-Tier registrieren
- [ ] 3.3 Go-Test: Nutzer mit reiner Dienst-Berechtigung sieht nur sein zugewiesenes Spiel
- [ ] 3.4 Go-Test: Trainer sieht alle Spiele seines Teams unabhängig von Dienst-Zuweisungen
- [ ] 3.5 Go-Test: Nutzer ohne jede Berechtigung erhält leere Liste mit HTTP 200 (kein 403)

## 4. Backend: Download-Endpoint

- [ ] 4.1 `internal/videos/paths.go`: Helper für Temp-Verzeichnis (`{root}/tmp/`) und Temp-Dateiname-Generierung
- [ ] 4.2 Helper zum Parsen von `index.m3u8` einer Rendition in Segment-Reihenfolge (Manifest-Reihenfolge, nicht Verzeichnis-Sortierung — siehe design.md Context)
- [ ] 4.3 Neuer Handler `internal/videos/download.go`: `GET /api/videos/{id}/download` — `CanViewVideo`-Prüfung (403/404 wie `Get`), `status != 'ready'` → 409, Disk-Guard via `RequireFreeBytes` (507 bei Unterschreitung), ffmpeg-Concat-Remux (`-f concat -safe 0 -c copy`) in Temp-Datei, bei Erfolg Stream mit `Content-Type: video/mp4`, `Content-Disposition: attachment; filename="<sanitierter Titel>.mp4"`, `Content-Length`; bei ffmpeg-Fehler HTTP 500 ohne Teilantwort; Temp-Datei in jedem Fall per `defer os.Remove` löschen
- [ ] 4.4 Dateinamen-Sanitisierung (Titel → sicherer Dateiname, keine Pfadtrenner/Sonderzeichen, die `Content-Disposition` brechen)
- [ ] 4.5 Route in `internal/app/router.go` registrieren (gleiches Auth-Tier wie `GET /api/videos/{id}`, außerhalb der `StreamTokenMiddleware`)
- [ ] 4.6 `internal/permissions/object_matrix_test.go`: Fixture-Eintrag für `GET /api/videos/{id}/download` ergänzen (analog `GET /api/videos/{id}/play`)
- [ ] 4.7 Go-Test: berechtigter Nutzer, Video `status='ready'` → 200, korrekte Header, vollständiger, abspielbarer Inhalt
- [ ] 4.8 Go-Test: nicht berechtigter Nutzer → derselbe Statuscode wie `GET /api/videos/{id}` für diesen Nutzer
- [ ] 4.9 Go-Test: Video nicht `status='ready'` → 409, kein ffmpeg-Aufruf (Fake-ffmpeg-Naht wie in `worker_test.go` verwenden, um „kein Aufruf" zu verifizieren)
- [ ] 4.10 Go-Test: ffmpeg-Fehler (Fake-ffmpeg-Naht liefert Fehler) → 500, keine Teilantwort, Temp-Datei wird trotzdem aufgeräumt
- [ ] 4.11 Go-Test: Disk-Guard unterschreitet → 507, kein ffmpeg-Aufruf
- [ ] 4.12 Scheduler-Job (`internal/scheduler`) oder Cleanup-Helfer: verwaiste Dateien in `{root}/tmp/` älter als N Stunden löschen (Detail-Intervall aus design.md Open Questions final festlegen), Go-Test dafür

## 5. Backend: Admin-Flag für Diensttypen

- [ ] 5.1 `internal/duties/handler.go`: Create/Update-Duty-Type-Requests um `grants_video_upload bool` erweitern, in INSERT/UPDATE übernehmen (analog `end_at_next_duty`)
- [ ] 5.2 List/Get-Duty-Type-Response um `grants_video_upload` erweitern
- [ ] 5.3 Go-Test: Anlegen/Bearbeiten eines Diensttyps mit `grants_video_upload=true` persistiert und wird zurückgegeben

## 6. Web-Frontend: Admin-UI

- [ ] 6.1 `web/src/pages/AdminDutyTypesPage.tsx`: Checkbox „Berechtigt zum Video-Upload" im Anlage-/Bearbeitungs-Modal, `brand-*`-Tokens, keine neuen Raw-Farben
- [ ] 6.2 Vitest: Checkbox-Zustand wird korrekt gesendet/vorbefüllt

## 7. Web-Frontend: Download-Button

- [ ] 7.1 `web/src/pages/VideoDetailPage.tsx`: Download-Button neben `CastButton` (nur wenn `video.status === 'ready'`), verlinkt/triggert `GET /api/videos/{id}/download` — Anmerkung: Artifact-Sandbox-Hinweise zu `<a download>` betreffen nur Artifacts, nicht die reguläre Web-App; hier normaler authentifizierter Request über `api`-Client mit Blob-Response und programmatischem Download-Trigger, `lucide-react`-Icon (`Download`), keine Unicode-Icons
- [ ] 7.2 Ladezustand/Fehlerbehandlung analog bestehender Aktionen auf der Seite (z. B. `actionError`-Pattern)
- [ ] 7.3 Vitest: Button erscheint nur bei `status='ready'`, löst den erwarteten Request aus, zeigt Fehler bei fehlgeschlagenem Download

## 8. Desktop-Tool (`tools/video-encoder`)

- [ ] 8.1 `internal/client/client.go`: `ActiveSeasonID` auf `GET /api/seasons/active` umstellen (kein 403-basiertes `ErrNoUploadPermission` mehr an dieser Stelle)
- [ ] 8.2 Neue Methode `EligibleGameIDs(ctx, seasonID) (map[int]bool, error)` für `GET /api/videos/upload-eligible-games`
- [ ] 8.3 UI-Schicht: Spielauswahl filtert auf `EligibleGameIDs`, wenn der Nutzer keine Trainer-/sportliche-Leitung-/Vorstand-Funktion hat; verständliche Meldung, wenn die Menge leer ist (Ersatz/Ergänzung für `ErrNoUploadPermission`)
- [ ] 8.4 Client-Tests (`client_test.go`) für `ActiveSeasonID`-Umstellung und `EligibleGameIDs`
- [ ] 8.5 UI-Tests für die gefilterte Spielauswahl, soweit im Tool testbar

## 9. Dokumentation

- [ ] 9.1 Neuen Gotcha-Eintrag in `docs/agent/06-gotchas.md` ergänzen: dienst-basierte Video-Upload-Berechtigung (1:1 Dienst↔Spiel, kein teamweiter Freifahrtschein, unabhängig von `CanViewVideo`) — kurz, analog bestehender Video-Gotchas
- [ ] 9.2 `make verify-change`/`openspec validate` grün vor Abschluss
