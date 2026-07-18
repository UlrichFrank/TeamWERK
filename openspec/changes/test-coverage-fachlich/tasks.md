## 1. Testutil-Erweiterungen

- [x] 1.1 `NoopMailer` in `internal/testutil/` anlegen (implementiert `mailer.Mailer`, verwirft alle Mails)
- [x] 1.2 `CreateDutyType(t, db, name, hoursValue)` in `internal/testutil/fixtures.go` ergänzen
- [x] 1.3 `CreateDutySlot(t, db, dutyTypeID, seasonID, teamID, gameID, date)` in `internal/testutil/fixtures.go` ergänzen
- [x] 1.4 `CreateInvitationToken(t, db, email, role, expiresAt)` in `internal/testutil/fixtures.go` ergänzen (gibt plain-Token zurück)
- [x] 1.5 `CreatePasswordResetToken(t, db, userID, expiresAt)` in `internal/testutil/fixtures.go` ergänzen (gibt plain-Token zurück)

## 2. auth-Tests (internal/auth/handler_test.go)

- [x] 2.1 Testserver-Setup: `newAuthServer(t, db)` mit allen auth-Routen und NoopMailer
- [x] 2.2 TC-A01: `TestLogin_ValidCredentials` — HTTP 200, access_token, Cookie gesetzt, refresh_tokens-Eintrag
- [x] 2.3 TC-A02: `TestLogin_WrongPassword` — HTTP 401
- [x] 2.4 TC-A03: `TestLogin_UnknownEmail` — HTTP 401
- [x] 2.5 TC-A04: `TestLogin_ProxyAccountBlocked` — User mit can_login=0 → HTTP 401
- [x] 2.6 TC-A05: `TestRefresh_ValidCookie` — Token rotiert (alter gelöscht, neuer in DB), HTTP 200
- [x] 2.7 TC-A06: `TestRefresh_InvalidCookie` — HTTP 401
- [x] 2.8 TC-A07: `TestLogout_ClearsToken` — DB-Eintrag gelöscht, Cookie MaxAge=-1
- [x] 2.9 TC-A08: `TestRegister_ValidToken` — User angelegt, used_at gesetzt
- [x] 2.10 TC-A09: `TestRegister_ExpiredToken` — HTTP 400
- [x] 2.11 TC-A10: `TestRegister_UsedToken` — HTTP 400
- [x] 2.12 TC-A11: `TestForgotPassword_AlwaysNoContent` — 204 für bekannte UND unbekannte Mail
- [x] 2.13 TC-A12: `TestResetPassword_Valid` — Passwort geändert, Token used, alle RefreshTokens gelöscht
- [x] 2.14 TC-A13: `TestResetPassword_ExpiredToken` — HTTP 400
- [x] 2.15 TC-A14: `TestUpdateUserRole_AdminOnly` — Admin darf "admin" vergeben; Nicht-Admin → 403; "trainer" → 400
- [x] 2.16 TC-A15: `TestDeleteUser_SelfForbidden` und `TestDeleteUser_Cascade` — 400 bzw. Cascade-Prüfung
- [x] 2.17 Commit: `test(auth): Login, Token-Rotation, Passwort-Reset, Nutzerverwaltung`

## 3. duties-Tests (internal/duties/handler_test.go)

- [x] 3.1 Testserver-Setup: `testServer(t, h)` mit Claim/Unclaim/Board/Slot-Routen (umgesetzt als `testServer`, nicht `newDutiesServer`)
- [x] 3.2 TC-D01: `TestClaim_FreeSlot` — slots_filled++, duty_accounts-Eintrag angelegt
- [x] 3.3 TC-D02: `TestClaim_FullSlot` — HTTP 409
- [x] 3.4 TC-D03: `TestClaim_Duplicate` — HTTP 409 (UNIQUE-Verletzung)
- [x] 3.5 TC-D04: `TestUnclaim_Pending` — Assignment gelöscht, slots_filled--
- [x] 3.6 TC-D05: `TestUnclaim_Fulfilled` — HTTP 409
- [x] 3.7 TC-D06: `TestUnclaim_NotFound` — HTTP 404
- [x] 3.8 TC-D07: `TestClaim_ForProxyChild` — Elternteil claimt für Kind mit can_login=0
- [x] 3.9 TC-D08: `TestClaim_ForeignUserForbidden` — HTTP 403
- [x] 3.10 TC-D09: `TestBoard_AdminSeesAll` — alle Slots in aktiver Saison
- [x] 3.11 TC-D10: `TestBoard_UserSeesOwnTeam` — nur eigene Team-Slots
- [x] 3.12 TC-D11: `TestBoard_AudienceElternVisible` — eltern-Slot für Elternteil sichtbar
- [x] 3.13 TC-D12: `TestBoard_AudienceElternHidden` — eltern-Slot für User ohne Kinder unsichtbar
- [x] 3.14 TC-D13: Trainer-Audience-Verhalten — bei Umsetzung präzisiert: Trainer umgeht Audience *nicht* per se, sondern via `?audience=all`/Team-Quelle (`TestDutyBoard_TrainerAudienceFilterDefault`, `TestBoard_AudienceElternTeamScoped`) statt eines `TestBoard_TrainerBypassesAudience`
- [x] 3.15 TC-D14: `TestBoard_ViewMine` — nur eigene geclaimten Slots
- [x] 3.16 TC-D15: `TestAccounts_AdminSeesAll` und `TestAccounts_UserSeesOwn` — Sichtbarkeit + Balance
- [x] 3.17 TC-D16: `TestCreateSlot_IsCustom` — is_custom=1
- [x] 3.18 TC-D17: `TestUpdateSlot_IsCustom` — is_custom=1 nach Update
- [x] 3.19 TC-D18: `TestDeleteSlot_WithAssignments` — Slot gelöscht
- [x] 3.20 Commit: `test(duties): Claim/Unclaim, Board-Audience, Dienstkonten`

## 4. members-Tests (internal/members/handler_test.go)

- [ ] 4.1 Testserver-Setup: `newMembersServer(t, db)` mit List/FamilyLink/ProxyAccount-Routen
- [x] 4.2 TC-M01: `TestList_Pagination` — limit/offset + total
- [x] 4.3 TC-M02: `TestList_SearchByName` — serverseitige Namenssuche
- [x] 4.4 TC-M03: `TestList_AusgetretenHidden` — ausgetretene nicht in Liste
- [x] 4.5 TC-M04: `TestList_TrainerScope` — Trainer sieht nur eigenes Team
- [x] 4.6 TC-M05: `TestFamilyLink_Create` — Eintrag angelegt
- [x] 4.7 TC-M06: `TestFamilyLink_MaxTwo` — HTTP 409 bei drittem Elternteil
- [x] 4.8 TC-M07: `TestFamilyLink_DuplicateIdempotent` — kein Fehler, ein Eintrag
- [x] 4.9 TC-M08: `TestFamilyLink_DeleteNotFound` — HTTP 404
- [x] 4.10 TC-M09: `TestProxyAccount_Create` — can_login=0, members.user_id gesetzt
- [x] 4.11 TC-M10: `TestProxyAccount_AlreadyHasAccount` — HTTP 409
- [x] 4.12 Commit: `test(members): Mitgliederliste, Familien-Links, Proxy-Accounts`

## 5. kader-Handler-Tests (internal/kader/handler_test.go)

- [x] 5.1 Testserver-Setup: `newKaderServer(t, db)` mit AutoAssign/MemberSuggestions-Routen
- [x] 5.2 TC-K01: `TestAutoAssign_BracketFilter` — Jg. 2007 drin, Jg. 2005 draußen (A-Jugend 2025/26)
- [x] 5.3 TC-K02: `TestAutoAssign_ExcludesAusgetreten` — ausgetretenes Mitglied nicht zugewiesen
- [x] 5.4 TC-K03: `TestAutoAssign_DedicatedBirthYear` — exakter Jahrgang statt Bracket
- [x] 5.5 TC-K04: `TestMemberSuggestions_BracketActive` — nur Mitglied im Bracket
- [x] 5.6 TC-K05: `TestMemberSuggestions_BracketDisabled` — alle Mitglieder sichtbar
- [x] 5.7 Commit: `test(kader): AutoAssign Bracket-Logik, Member-Suggestions`

## 6. Lückenfüller in bestehenden Paketen

- [x] 6.1 `games/handler_test.go`: `TestListTeamsForUser_Trainer` — nur eigene Teams
- [x] 6.2 `games/handler_test.go`: `TestListTeamsForUser_Admin` — alle Teams
- [x] 6.3 `games/handler_test.go`: `TestListTeamsForUser_Spieler` — nur Team-Mitgliedschaft
- [x] 6.4 `trainings/handler_test.go`: `TestGetAttendances_ReadsBack` — gespeicherte Anwesenheiten lesen
- [x] 6.5 `trainings/handler_test.go`: `TestRespond_ParentForChild` — Elternteil antwortet für Kind
- [x] 6.6 `absences/handler_test.go`: `TestCreateAbsence_UnauthorizedMember` — fremdes Mitglied → 403/gefiltert
- [x] 6.7 `absences/handler_test.go`: `TestPreview_Empty` — kein Event im Zeitraum → []
- [x] 6.8 `chat/handler_test.go`: `TestLeave_MemberLeavesGroup` — left_at gesetzt, System-Nachricht
- [x] 6.9 `chat/handler_test.go`: `TestLeave_DirectConversationRejected` — HTTP 400
- [x] 6.10 `chat/handler_test.go`: `TestCreateDirect_DuplicateReturnsExisting` — kein Duplikat
- [x] 6.11 Commit: `test(games/trainings/absences/chat): Lückenfüller alle Pakete`
