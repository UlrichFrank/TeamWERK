## 1. files — Route-Ebenen-Authz (`internal/files/handler_test.go`)

- [x] 1.1 `TestCreateFolder_NoWriteForbidden` — `POST /api/folders` bzw. Subfolder ohne `can_write` → 403
- [x] 1.2 `TestCreateFolder_HappyPath` — Nutzer mit `can_write` legt Ordner an → 201, Zeile in `file_folders`
- [x] 1.3 `TestDeleteFolder_NoWriteForbidden` — `DELETE /api/folders/{id}` ohne `can_write` → 403; Ordner bleibt
- [x] 1.4 `TestUploadFile_NoWriteForbidden` — `POST /api/folders/{id}/files` (via `testutil.PostMultipart`) ohne `can_write` → 403, keine Datei gespeichert
- [x] 1.5 `TestUploadFile_HappyPath` — mit `can_write` → 201/200, Datei in `files`
- [x] 1.6 `TestAddPermission_EscalationForbidden` — HTTP-403, wenn ein Nutzer ein Recht vergibt, das er nicht hält (ergänzt die vorhandenen `checkAntiEscalation`-Units auf Route-Ebene)
- [x] 1.7 `TestDeletePermission_NoWriteForbidden` — `DELETE /api/folders/{id}/permissions/{permId}` ohne `can_write` → 403
- [x] 1.8 `TestDownloadToken_NoReadForbidden` — `GET /api/files/{id}/download-token` ohne Leserecht → 403 (fail-closed, kein Token). **Vor dem Test Verhalten am Code verifizieren (D3);** liefert die Route fälschlich einen Token, erst `fix(files)`, dann Test.
- [x] 1.9 `TestDownloadToken_HappyPath` — mit Leserecht → 200 + Token
- [x] 1.10 Commit: `test(files): Route-Authz für Ordner-/Datei-CRUD + Download-Token`

## 2. matchreports — ServeImage-Authz (`internal/matchreports/`)

- [x] 2.1 Testserver mit `GET /api/match-reports/{id}/images/{imgId}/blob` verdrahten (Handler `ServeImage`)
- [x] 2.2 `TestServeImage_Unauthenticated` — kein Claim → 401
- [x] 2.3 `TestServeImage_ForeignForbidden` — eingeloggt, weder Autor noch Reviewer → 403
- [x] 2.4 `TestServeImage_NotFound` — unbekannte Report-/Image-ID → 404
- [x] 2.5 `TestServeImage_AuthorOK` — Autor ruft eigenes Bild ab → 200
- [x] 2.6 `TestServeImage_ReviewerOK` — Reviewer (medien/vorstand/admin) → 200
- [x] 2.7 Commit: `test(matchreports): ServeImage nur Autor/Reviewer`

## 3. duties — Spielbericht-Slot-Guard (`internal/duties/`)

- [x] 3.1 `TestClaim_MatchReportSlot_NonPressForbidden` — Nicht-Presseteam claimt Spielbericht-Slot → 403 (`role_required`)
- [x] 3.2 `TestClaim_MatchReportSlot_PressTeamOK` — `presseteam` claimt → 204
- [x] 3.3 `TestClaim_MatchReportSlot_AdminOK` — `admin` claimt → 204
- [x] 3.4 `TestClaim_MatchReportSlot_ProxyParentForbidden` — Elternteil ohne `presseteam` claimt für Kind → 403 (Rolle des handelnden Users wird gewertet)
- [x] 3.5 `TestClaim_NonMatchReportSlot_Unaffected` — Slot anderen Typs → Guard greift nicht, regulärer Claim
- [x] 3.6 Commit: `test(duties): Spielbericht-Slot-Guard inkl. Proxy-Rollenverschiebung`

## 4. attendance-Recording (`internal/training/` + `internal/games/`)

- [ ] 4.1 `internal/training`: Testserver für `POST /api/training-sessions/{id}/attendances` (Package hat bisher **keine** Testdatei — neu anlegen)
- [ ] 4.2 `TestSaveAttendances_ForeignTeamTrainerForbidden` (training) — Trainer eines fremden Teams → 403
- [ ] 4.3 `TestSaveAttendances_OwnTeamTrainerOK` (training) — zuständiger Trainer → 2xx, Recording persistiert
- [ ] 4.4 `TestSaveAttendances_NonStaffForbidden` (training) — Nicht-Staff → 403
- [ ] 4.5 `TestGameSaveAttendances_ForeignTeamForbidden` (games) — analog für `POST /api/games/{id}/attendances`
- [ ] 4.6 `TestGameSaveAttendances_OwnTeamOK` (games) — zuständiger Trainer → 2xx
- [ ] 4.7 Commit: `test(training,games): Recording-Authz für Anwesenheiten`

## 5. absences — Sichtbarkeit & Mutation (`internal/absences/handler_test.go`)

- [ ] 5.1 `TestCalendar_ShowTeam_MemberSeesNoTeamAbsences` — einfaches Mitglied mit `?show_team=true` → keine fremden Team-Abwesenheiten
- [ ] 5.2 `TestCalendar_ShowTeam_VorstandSeesTeam` — vorstand/trainer-like → Team-Abwesenheiten sichtbar
- [ ] 5.3 `TestUpdate_ForeignForbidden` — `PUT /api/absences/{id}` durch Fremden → 403
- [ ] 5.4 `TestDelete_ForeignForbidden` — `DELETE /api/absences/{id}` durch Fremden → 403; Eintrag bleibt
- [ ] 5.5 `TestList_NoForeignAbsences` — `GET /api/absences` gibt keine fremden Abwesenheiten zurück
- [ ] 5.6 Commit: `test(absences): Calendar-show_team-Scoping + Update/Delete/List-Authz`

## 6. Abschluss

- [ ] 6.1 `go test ./...` grün; `/verify-change` (Build/Test/Lint + Invarianten) grün
- [ ] 6.2 `openspec validate test-pii-route-authz --strict` grün
- [ ] 6.3 Rückblick (Roadmap 9.1): Risiko-/Churn-Bild nach Welle 1 neu bewerten; Roadmap-Section 4 abhaken
- [ ] 6.4 Change archivieren (`openspec archive`) — appliziert Capability `pii-route-authz`

## Test-Anforderungen

Route → Testname → erwarteter Status → garantierte Invariante.

**files** (`internal/files`)
- `POST /api/folders` → `TestCreateFolder_NoWriteForbidden` → 403 → ohne `can_write` kein Ordner-Anlegen
- `DELETE /api/folders/{id}` → `TestDeleteFolder_NoWriteForbidden` → 403 → ohne `can_write` keine Löschung
- `POST /api/folders/{id}/files` → `TestUploadFile_NoWriteForbidden` → 403 → ohne `can_write` kein Upload, keine Datei persistiert
- `POST /api/folders/{id}/permissions` → `TestAddPermission_EscalationForbidden` → 403 → kein Grant über eigene Rechte hinaus (HTTP-Ebene)
- `DELETE /api/folders/{id}/permissions/{permId}` → `TestDeletePermission_NoWriteForbidden` → 403 → ohne `can_write` keine Rechte-Entnahme
- `GET /api/files/{id}/download-token` → `TestDownloadToken_NoReadForbidden` → 403 → fail-closed, kein Token ohne Leserecht

**matchreports** (`internal/matchreports`)
- `GET /api/match-reports/{id}/images/{imgId}/blob` → `TestServeImage_ForeignForbidden` → 403 → Bild nur Autor/Reviewer
- `GET …/blob` → `TestServeImage_NotFound` → 404 → unbekannte ID gibt nichts preis
- `GET …/blob` → `TestServeImage_Unauthenticated` → 401 → ohne Claim kein Zugriff

**duties** (`internal/duties`)
- `POST /api/duty-board/{slotId}/claim` (Spielbericht) → `TestClaim_MatchReportSlot_NonPressForbidden` → 403 → nur presseteam/admin
- `POST …/claim` (Spielbericht, Proxy) → `TestClaim_MatchReportSlot_ProxyParentForbidden` → 403 → Rolle des handelnden Users zählt

**training/games** (`internal/training`, `internal/games`)
- `POST /api/training-sessions/{id}/attendances` → `TestSaveAttendances_ForeignTeamTrainerForbidden` → 403 → nur Staff des zuständigen Teams
- `POST /api/games/{id}/attendances` → `TestGameSaveAttendances_ForeignTeamForbidden` → 403 → nur Staff des zuständigen Teams

**absences** (`internal/absences`)
- `GET /api/absences/calendar?show_team=true` → `TestCalendar_ShowTeam_MemberSeesNoTeamAbsences` → 200 (leer) → kein Team-Leak an Nicht-Berechtigte
- `PUT /api/absences/{id}` → `TestUpdate_ForeignForbidden` → 403 → keine Fremd-Mutation
- `DELETE /api/absences/{id}` → `TestDelete_ForeignForbidden` → 403 → keine Fremd-Löschung
- `GET /api/absences` → `TestList_NoForeignAbsences` → 200 → keine fremden Abwesenheiten im Ergebnis
