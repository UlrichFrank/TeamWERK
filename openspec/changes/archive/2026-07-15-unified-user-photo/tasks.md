## 1. Migration & Backfill

- [x] 1.1 Migration `029_photo_user_only.up.sql` + `.down.sql` schreiben:
      User-Foto-Übernahme, `photo_visible`-Spiegel auf `user_visibility`,
      Drop von `members.photo_path` und `members.photo_visible`.
- [x] 1.2 Datei-Cleanup-Backfill in `internal/upload/backfill.go` (Muster
      `internal/videos/backfill.go`): verwaiste Dateien in
      `uploadDir/member-photos/` löschen; als Goroutine in `serve()` starten.
- [x] 1.3 Test `internal/db/migration_029_test.go`: drei Mitglieds-Konstellationen
      (mit User+beide Fotos / mit User+nur member / ohne User+member),
      erwartete Zielzustände prüfen.

## 2. Upload-Handler

- [x] 2.1 `UploadChildPhoto`/`DeleteChildPhoto` in `internal/upload/handler.go`
      auf `users.photo_path` des Kind-Users umbauen (via `members.user_id`);
      HTTP 409 wenn `user_id IS NULL`.
- [x] 2.2 `UploadMemberPhoto`/`DeleteMemberPhoto` analog: `users.photo_path`
      via Lookup; HTTP 409 ohne `user_id`.
- [x] 2.3 `UploadUserPhoto`/`DeleteUserPhoto` unverändert (Regression-Test).
- [x] 2.4 `broadcastMembers` weiterhin nach jedem Write triggern (SSE-Regel).

## 3. Lese-Pfade

- [x] 3.1 `internal/members/handler.go` — `MemberBase.PhotoURL` und
      `UserPhotoURL` zu einem einheitlichen `photo_url` verschmelzen; alle
      `m.photo_path`-Queries auf `u.photo_path` via Join umstellen.
- [x] 3.2 `internal/members/drafts.go` — `m.photo_path`/`m.photo_visible`
      durch Join auf `users`/`user_visibility` ersetzen.
- [x] 3.3 `internal/carpooling/handler.go` — bereits korrekt auf
      `u.photo_path`; Grep bestätigt.
- [x] 3.4 `GetChildProfile`: `member.photo_url` im Response aus
      `users.photo_path` des Kind-Users befüllen (folgt automatisch aus
      der Refaktorierung von `getMember`).
- [x] 3.5 Admin `PUT /api/members/{id}`: `photo_visible` in `user_visibility`
      upserten statt in `members` schreiben.

## 4. Frontend

- [x] 4.1 `web/src/pages/MembersPage.tsx` — `user_photo_url`-Sonderfeld
      entfernen, `PersonChip` nutzt `photo_url`.
- [x] 4.2 `web/src/pages/MemberDetailPage.tsx` — nutzt bereits `photo_url`
      als einzige Quelle (keine Änderung nötig, Backend liefert jetzt aus
      User-Strang).
- [x] 4.3 `web/src/components/admin/MemberStammdatenTab.tsx` — Foto-Sektion
      wird ausgeblendet, wenn kein User verknüpft ist; HTTP 409-Meldung
      „Mitglied hat keinen Account".
- [x] 4.4 `web/src/components/profile/ProfileProfilTab.tsx` (mode='child') —
      HTTP 409 abfängt als „Foto benötigt einen Account".
- [x] 4.5 `web/src/pages/ChildProfilePage.tsx` — Type `Member.photo_url`
      bleibt, Wert kommt jetzt aus User-Strang.

## 5. Tests

- [x] 5.1 `internal/upload/photo_targeting_test.go` — sechs neue Tests
      (Happy-Path pro Route + 409-Fehlerfall pro Kind/Admin-Route).
- [x] 5.2 `GetMember` und `GetChildProfile` liefern `photo_url` aus
      User-Strang (implizit über `getMember`-Refaktor + vollständiger
      `go test ./...`-Lauf, 1423 Tests grün).
- [x] 5.3 Bestehende Tests grün:
      - `internal/upload/photo_broadcast_test.go` (Child-Tests brauchen jetzt
        Child-User)
      - `internal/matchreports/photo_consent_internal_test.go` (photo_visible-
        Seed entfernt, war fachlich ohnehin ohne Belang)
      - vitest 553/553.

## 6. Spec & Verifikation

- [x] 6.1 Delta-Specs in `openspec/changes/unified-user-photo/specs/`
      (kind-profil-user-strang MODIFIED + ADDED, profilbild-crop-upload MODIFIED).
- [x] 6.2 `openspec validate unified-user-photo --strict` grün.
- [x] 6.3 `go build ./...` grün, `go test ./...` 1423 grün, arch/broadcast-Gate
      grün, `golangci-lint` clean, `pnpm -C web build` + `pnpm -C web test`
      553 grün.

## 7. Deploy & Archivierung

- [ ] 7.1 **DB-Backup vor `make migrate-remote-up`** (Down-Migration verliert Foto-Referenzen).
- [ ] 7.2 Nach Deploy: Backfill-Log prüfen (Anzahl verwaister Dateien).
- [ ] 7.3 Nach Merge in `main`: `openspec archive unified-user-photo`.
