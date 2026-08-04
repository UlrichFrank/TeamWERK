## 1. Datenbank

- [x] 1.1 Migration `internal/db/migrations/040_training_diary.up.sql` + `.down.sql` anlegen. Tabelle `training_diary_entries`: `id`, `member_id` (FK `members` ON DELETE CASCADE), `season_id` (FK `seasons` ON DELETE SET NULL, nullbar), `trained_on DATE NOT NULL`, `kind TEXT NOT NULL CHECK (kind IN ('kraft','ausdauer','athletik','technik','beweglichkeit','reha','sonstiges'))`, `kind_custom TEXT`, `duration_min INTEGER NOT NULL CHECK (duration_min > 0 AND duration_min <= 600)`, `rpe INTEGER NOT NULL CHECK (rpe BETWEEN 1 AND 10)`, `note TEXT`, `proof_disk_name TEXT`, `proof_mime TEXT`, `proof_size INTEGER`, `proof_uploaded_at DATETIME`, `proof_purged_at DATETIME`, `created_at`, `updated_at`. Indizes auf `(member_id, trained_on DESC)` und `(season_id)`. **Nummer 040 vor dem Schreiben gegen `ls internal/db/migrations/ | tail` gegenprüfen** — eine Nummer ≤ DB-Version wird lautlos übersprungen.
- [x] 1.2 `make migrate-up` lokal ausführen und die Down-Migration nachweislich durchspielen. **Befund:** `make migrate-down` ist wirkungslos — `runMigrate()` → `db.Migrate()` ruft immer `m.Up()`, das `up`/`down`-Argument aus `cmd/teamwerk/main.go` wird nie ausgewertet. Betrifft alle Migrationen, nicht nur diese; eigener Change nötig. Down-Migration deshalb ersatzweise direkt per `sqlite3` gegen eine Kopie der migrierten DB geprüft (down → alle Objekte weg, up → sauber neu angelegt).

## 2. Backend — Package-Gerüst

- [x] 2.1 `internal/trainingdiary/handler.go`: `Handler` mit `db *sql.DB`, `hub *hub.EventHub`, `dir string`; `NewHandler(db, hub, dir)` legt das Verzeichnis an (Muster `media.NewHandler`). In `internal/app/router.go` als `TrainingDiary *trainingdiary.Handler` aufnehmen und in `cmd/teamwerk/main.go` verdrahten.
- [x] 2.2 `internal/config/`: `TRAINING_DIARY_DIR` mit Default `./storage/training-diary` ergänzen, analog zu den bestehenden Storage-Pfaden.
- [x] 2.3 `internal/arch/arch_test.go`: `internal/trainingdiary` als **Domain**-Package klassifizieren. `go test ./internal/arch/...` muss grün sein (das Package importiert kein anderes Domain-Package).

## 3. Backend — CRUD auf Einträgen

- [x] 3.1 `resolveOwnMember(ctx, claims) (memberID int, err error)`: Mitglied des aufrufenden Nutzers auflösen; kein Mitglied → 403.
- [x] 3.2 `validateEntry(req)`: `kind` gegen die feste Liste, `kind_custom` genau dann gesetzt wenn `kind='sonstiges'` (nicht leer, max. 60 Zeichen), `duration_min` in `1..600`, `rpe` in `1..10`, `trained_on` nicht in der Zukunft. Verletzung → 400 mit sprechender Meldung.
- [x] 3.3 `CreateEntry` (`POST /api/training-diary`): `member_id` aus dem Token, `season_id` aus `seasons.is_active = 1` (kein Treffer → `NULL`), Insert, `Broadcast("training-diary-changed")`, 201 mit dem angelegten Eintrag.
- [x] 3.4 `ListOwn` (`GET /api/training-diary?season=`): eigene Einträge absteigend nach `trained_on`, mit Nachweis-Status (`none` / `present` / `purged`).
- [x] 3.5 `UpdateEntry` (`PUT /api/training-diary/{id}`): nur Eigentümer (sonst 403, 404 vor 403 prüfen, damit fremde IDs nicht per Statuscode enumerierbar sind), dieselbe Validierung, `season_id` bleibt unverändert, Broadcast.
- [x] 3.6 `DeleteEntry` (`DELETE /api/training-diary/{id}`): nur Eigentümer, Nachweisdatei mit entfernen (fehlende Datei ist kein Fehler), Broadcast, 204.
- [x] 3.7 Routen in `internal/app/router.go` im Tier „Authenticated" eintragen (ACL im Handler, Muster `attendance-stats`).

## 4. Backend — Sichtbarkeit

- [x] 4.1 `canReadMemberDiary(ctx, claims, memberID) (bool, error)`: Muster von `attendance.canSeeMemberStats` übernehmen — Mitglied selbst, Elternteil via `family_links`, Trainer via `trainer_memberships` × `kader` in der aktiven Saison (Stamm- **und** erweiterter Kader), `sportliche_leitung`, `admin`. `vorstand` bewusst **nicht**.
- [x] 4.2 `GetMemberDiary` (`GET /api/members/{id}/training-diary?season=`): ACL aus 4.1, Einzeleinträge des Mitglieds.
- [x] 4.3 `canSeeTeamDiary(ctx, claims, teamID)`: Muster `attendance.canSeeTeamStats` — Trainer des Teams in der aktiven Saison, `sportliche_leitung`, `admin`.
- [x] 4.4 `GetTeamStats` (`GET /api/teams/{id}/training-diary-stats?season=`): je Kadermitglied `entries`, `minutes`, `avg_rpe` (eine Nachkommastelle); Mitglieder ohne Einträge mit Nullwerten. Ohne aktive Saison und ohne `?season=` → 200 mit leerer Liste.

## 5. Backend — Nachweis

- [x] 5.1 `UploadProof` (`POST /api/training-diary/{id}/proof`, multipart-Feld `proof`): nur Eigentümer; `MaxBytesReader` auf 1 MB + Multipart-Headroom; Typ per `http.DetectContentType` gegen die Whitelist `image/jpeg`, `image/png`, `image/webp`, `application/pdf` (alles andere 400, ausdrücklich auch `image/heic`/`image/heif`); Ablage als `<uuid>.<ext>` in `TRAINING_DIARY_DIR`; **alte Datei entfernen**, wenn schon eine hing; `proof_purged_at` zurücksetzen; Broadcast; 201.
- [x] 5.2 `DeleteProof` (`DELETE /api/training-diary/{id}/proof`): nur Eigentümer, Datei entfernen, Nachweis-Spalten leeren, Broadcast, 204.
- [x] 5.3 `ServeProof` (`GET /api/training-diary/{id}/proof`): ACL aus 4.1; `proof_purged_at` gesetzt → **410**; nie ein Nachweis vorhanden → 404; sonst 200 mit gespeichertem Content-Type und `X-Content-Type-Options: nosniff` (Muster `media.Serve`).
- [x] 5.4 Aufräum-Pfad prüfen: schlägt der DB-Schreibvorgang nach dem `os.WriteFile` fehl, muss die Datei wieder entfernt werden (kein Waisen-Blob) — Muster `media.Upload`.

## 6. Backend — Retention im Scheduler

- [x] 6.1 `internal/scheduler/scheduler.go`: `runTrainingDiaryRetention()` im Daily-Block aufrufen (neben `runVideoRetention`). Inline-SQL, kein Import von `internal/trainingdiary` (Foundation-Regel, Architektur-Test).
- [x] 6.2 Query: Einträge mit `proof_disk_name IS NOT NULL` und `proof_purged_at IS NULL`, deren Saison über `JOIN seasons` ein `end_date < date('now','-90 days')` hat. `season_id IS NULL` fällt durch den Join heraus und wird damit nie bereinigt.
- [x] 6.3 Je Treffer: Datei löschen (`os.ErrNotExist` ignorieren), dann `UPDATE … SET proof_purged_at = CURRENT_TIMESTAMP, proof_disk_name = NULL`. Reihenfolge so, dass ein Abbruch dazwischen beim nächsten Lauf sauber nachzieht.
- [x] 6.4 Scheduler-Config um `TrainingDiaryDir` erweitern (Muster `VideoStorageDir`).

## 7. Backend — Tests

- [x] 7.1 Testutil-Fixture `CreateTrainingDiaryEntry` in `internal/testutil/fixtures.go` ergänzen (Muster der bestehenden Fixtures).
- [x] 7.2 CRUD-Tests: `TestCreateEntry_Success`, `_CustomKindRequiresText`, `_InvalidRPE`, `_FutureDate`, `_NoMemberForUser`, `TestUpdateEntry_ForeignEntry`, `TestDeleteEntry_Success`, `_ForeignEntry`, `TestListOwn_OnlyOwnEntries`.
- [x] 7.3 Sichtbarkeits-Tests: `TestMemberDiary_ParentAccess`, `_OtherPlayer`, `_SportlicheLeitung`, `TestTeamStats_TrainerOwnTeam`, `_PlayerForbidden`, `_TrainerForeignTeam`. **Kern-Invariante:** Spieler sehen einander nicht.
- [x] 7.4 Nachweis-Tests: `TestUploadProof_Success`, `_UnsupportedType`, `_TooLarge`, `_ForeignEntry`, `_ReplacesOld`, `TestServeProof_Owner`, `_TrainerOfKader`, `_OtherPlayer`, `_TrainerOtherTeam`, `_Purged` (410).
- [x] 7.5 Retention-Tests in `internal/scheduler/`: `TestTrainingDiaryRetention_PurgesAfterSeasonEnd`, `_KeepsWithinWindow`, `_NullSeasonNeverPurged`, `_Idempotent`, plus „Datei fehlt bereits" (Muster `video_retention_test.go`).
- [x] 7.6 `go test ./...` grün, inklusive `internal/arch` (Architektur- **und** Broadcast-Gate — alle sechs Mutations-Routen müssen ohne Allowlist-Eintrag durchgehen).

## 8. Frontend — Erfassung

- [x] 8.1 `web/src/components/RpeScaleInfo.tsx`: eingeklappte Info-Box mit `<ChevronRight>`/`<ChevronDown>`, Stufen 1–2 / 3–4 / 5–6 / 7–8 / 9–10 in Alltagssprache plus dem Hinweis, dass Schätzen genügt. Nur `brand-*`-Tokens, keine Unicode-Icons.
- [x] 8.2 `web/src/components/TrainingDiaryEntryForm.tsx`: Datum (max. heute), Art als Auswahl mit Freitextfeld bei `sonstiges`, Dauer in Minuten, RPE 1–10, Notiz, Nachweis-Dateifeld. Verbindliche Klassen-Strings aus `docs/agent/05-frontend.md`, Touch-Targets `py-2.5` auf Mobile.
- [x] 8.3 Upload-Pfad: `compressImage(file, { targetBytes: 150 * 1024, maxEdge: 1280 })` für `image/*`, andere Dateien unverändert. Danach `POST …/proof` als multipart.
- [x] 8.4 `web/src/pages/ProfilTrainingstagebuchPage.tsx`: eigene Liste, Anlegen/Bearbeiten/Löschen, Nachweis nachträglich anfügen. Nachweis-Anzeige über `AuthImage`; bei `proof_status='purged'` den Hinweis „Nachweis gelöscht" **ohne** Bildabruf. Statischer Hinweis auf die 90-Tage-Regel. `useLiveUpdates` auf `training-diary-changed`.
- [x] 8.5 Route `profil/trainingstagebuch` in `App.tsx` + Nav-Eintrag in `AppShell.tsx`. Mobile: Card-Layout statt Tabelle, Aktionen hinter `<MoreVertical>` (`MobileCard`, `ActionMenu`).

## 9. Frontend — Trainer-Sicht

- [x] 9.1 `web/src/components/TrainingDiaryStatsView.tsx`: Mannschaftsliste je Mitglied mit Einheiten, Minuten, ⌀ RPE und Balken; Klick lädt die Einzeleinträge über `GET /api/members/{id}/training-diary` nach. Struktur von `AttendanceStatsView.tsx` übernehmen.
- [x] 9.2 Hinweistext in der Übersicht: die Zahlen beruhen auf Selbstauskunft und messen Erfassungsdisziplin, nicht Trainingsfleiß.
- [x] 9.3 `web/src/pages/TeamTrainingstagebuchPage.tsx` + Routen `trainingstagebuch` und `team/:id/trainingstagebuch` mit `RoleRoute roles={['admin','trainer','sportliche_leitung']}`; Nav-Eintrag in `AppShell.tsx`.
- [x] 9.4 Tagebuch-Tab in `ProfilePage` und `ChildProfilePage` ergänzen (Muster: bestehender Anwesenheits-Tab), damit Eltern das Kind über den gewohnten Weg erreichen.

## 10. Frontend — Tests

- [x] 10.1 `TrainingDiaryEntryForm.test.tsx`: `sonstiges` blendet das Freitextfeld ein; Speichern ohne Freitext ist blockiert; Zukunftsdatum ist nicht wählbar.
- [x] 10.2 `TrainingDiaryEntryForm.compress.test.tsx`: `compressImage` wird mit `targetBytes: 153600` und `maxEdge: 1280` aufgerufen; eine PDF-Auswahl ruft ihn **nicht** auf.
- [x] 10.3 `RpeScaleInfo.test.tsx`: eingeklappt gerendert, Klick klappt auf.
- [x] 10.4 `TrainingDiaryStatsView.test.tsx`: Kennzahlen werden gerendert; Klick auf eine Zeile löst den Detail-Abruf aus.
- [x] 10.5 `ProfilTrainingstagebuchPage.test.tsx`: `proof_status='purged'` rendert den Hinweis und löst **keinen** Bildabruf aus.
- [x] 10.6 `pnpm -C web test` und `pnpm -C web lint` grün.

## 11. Dokumentation & Abschluss

- [x] 11.1 `docs/agent/06-gotchas.md`: kurzer Abschnitt „Trainingstagebuch" — Saison-Anker über die aktive Saison (nicht über `trained_on`), eigener Store getrennt von `media`, Retention löscht nur das Bild und setzt `proof_purged_at`, Serve antwortet 410 statt 404.
- [x] 11.2 `docs/agent/10-deployment.md`: `TRAINING_DIARY_DIR` in die Config-Liste aufnehmen und als backup-relevant kennzeichnen (wie `BEITRAGSLAUF_DIR`).
- [x] 11.3 `/verify-change` durchlaufen lassen: Build/Test/Lint, Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, `brand-*`-Tokens, lucide-Icons, Migrationsnummer, `openspec validate --strict`.
- [ ] 11.4 Proposal archivieren.
