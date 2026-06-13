## 1. Tooling & Konvention

- [x] 1.1 `make coverage` in `Makefile` ergänzen: `go test -coverprofile=/tmp/teamwerk-coverage.out ./internal/...`, dann `go tool cover -func` + `-html`
- [x] 1.2 CLAUDE.md: Abschnitt „## Test-Standard" hinzufügen mit Regel „neue Route = ≥1 Happy-Path + ≥1 Fehlerfall" und Pflichtabschnitt „Test-Anforderungen" in OpenSpec-Proposals
- [x] 1.3 Commit: `chore(make): coverage-Target ergänzen`
- [x] 1.4 Commit: `docs(claude): Test-Standard als Projektregeln verankern`

## 2. auth-Lücken (internal/auth/handler_test.go)

- [x] 2.1 `TestChangePassword_Valid` — korrektes altes PW → 204, Passwort geändert, alle refresh_tokens gelöscht
- [x] 2.2 `TestChangePassword_WrongCurrentPassword` — falsches altes PW → 403
- [x] 2.3 `TestApproveMembershipRequest_CreatesInvitationToken` — pending-Antrag → 204, invitation_tokens-Eintrag angelegt
- [x] 2.4 `TestApproveMembershipRequest_NotPending` — bereits approved/rejected → 404
- [x] 2.5 `TestRejectMembershipRequest_SetsStatus` — pending → 204, status='rejected'
- [x] 2.6 `TestListUsers_Pagination` — 12 User, limit=5 offset=5 → 5 items, total=12
- [x] 2.7 `TestListUsers_SearchByName` — Suche nach Nachnamen → Treffer gefiltert
- [x] 2.8 Commit: `test(auth): ChangePassword, ApproveMembership, RejectMembership, ListUsers`

## 3. members-Lücken (internal/members/handler_test.go)

- [x] 3.1 `TestGetProfile_ReturnsOwnData` — GET /api/profile/me → Response mit email, first_name des eingeloggten Users
- [x] 3.2 `TestUpdateProfile_PersistsChange` — PUT /api/profile/me mit geändertem first_name → Änderung in DB
- [x] 3.3 Commit: `test(members): GetProfile, UpdateProfile`

## 4. duties-Lücken (internal/duties/handler_test.go)

- [x] 4.1 `TestFulfill_SetsStatusFulfilled` — POST /api/duty-assignments/{id}/fulfill → status='fulfilled', duty_accounts.ist unverändert
- [x] 4.2 `TestCashSubstitute_SetsStatusAndAmount` — POST /api/duty-assignments/{id}/cash-substitute {amount: 15.0} → status='cash_substitute', cash_amount=15.0
- [x] 4.3 `TestListAssignments_ReturnsAll` — GET /api/duty-slots/{id}/assignments mit 2 Assignments → Liste mit user_name, status
- [x] 4.4 Commit: `test(duties): Fulfill, CashSubstitute, ListAssignments`

## 5. trainings-Lücken (internal/trainings/handler_test.go)

- [x] 5.1 `TestCreateSession_AdminOK` — POST /api/training-sessions → 201, Session in DB
- [x] 5.2 `TestUpdateSession_ChangesTime` — PUT /api/training-sessions/{id} → 204, Änderung persistiert
- [x] 5.3 `TestDeleteSeries_CascadesSessionsAndResponses` — DELETE /api/training-series/{id} → 204, Series + Sessions + Responses alle weg
- [x] 5.4 Commit: `test(trainings): CreateSession, UpdateSession, DeleteSeries`

## 6. kader-Lücken (internal/kader/handler_test.go)

- [x] 6.1 `TestCopyFromSeason_SameAgePrevious` — POST /api/admin/kader/copy-from-season mit member_source=same-age-previous → Kader angelegt, Mitglieder übernommen
- [x] 6.2 `TestCopyFromSeason_EmptyMemberSource` — member_source="" → Kader angelegt, keine Mitglieder
- [x] 6.3 Commit: `test(kader): CopyFromSeason`
