# Tasks — kinderaccount-recovery-email

> Ein Commit pro Task. Konventionelle Commits, Scope = führendes Domänen-Package.

## 1. Migration

- [x] 1.1 `internal/db/migrations/004_recovery_email.up.sql` + `.down.sql`:
  - `ALTER TABLE users ADD COLUMN recovery_email TEXT;`
  - `ALTER TABLE email_change_tokens ADD COLUMN field TEXT NOT NULL DEFAULT 'email';`
  - `ALTER TABLE email_change_tokens ADD COLUMN stage TEXT;`
  - Backfill: `UPDATE users SET recovery_email = (SELECT mr.parent_email FROM membership_requests mr WHERE …)` für Kinderkonten, soweit eindeutig zuordenbar.

  _Commit:_ `feat(db): recovery_email auf users + field/stage auf email_change_tokens`

## 2. Backend: Approval persistiert recovery_email

- [x] 2.1 `internal/auth/handler.go` (`approveChildRequest`): beim `INSERT INTO users` `recovery_email = parentEmail` setzen (statt NULL).
- [x] 2.2 `go vet` + `gofmt`.

  _Commit:_ `feat(auth): Kind-Approval persistiert Eltern-E-Mail als recovery_email`

## 3. Backend: Forgot-Password über login_name

- [x] 3.1 `internal/auth/handler.go` (`ForgotPassword`): Lookup auf `(LOWER(email)=LOWER(?) OR LOWER(login_name)=LOWER(?)) AND can_login=1`; Ziel-Adresse `COALESCE(NULLIF(email,''), recovery_email)`; weiterhin immer HTTP 204. Mail nur senden, wenn eine Ziel-Adresse vorhanden ist.
- [x] 3.2 `go vet` + `gofmt`.

  _Commit:_ `feat(auth): forgot-password akzeptiert login_name, sendet an recovery_email`

## 4. Backend: Eltern-Änderungs-Workflow (doppelte Bestätigung)

- [x] 4.1 `internal/auth/handler.go` (`RequestRecoveryEmailChange`): Route `POST /api/profile/kind/{memberId}/recovery-email`. `isParentOf`-Check; alten Token zum Konto löschen; Token `field='recovery_email', stage='auth', new_email=...` anlegen; Bestätigungsmail an die **aktuelle** `recovery_email`. 403 ohne Eltern-Beziehung.
- [x] 4.2 `internal/auth/handler.go` (`ConfirmRecoveryEmailChange`): Route `GET /api/profile/recovery-email/confirm?token=...` (public). `stage='auth'` → Token rotieren auf `stage='verify'`, Mail an `new_email`; `stage='verify'` → `users.recovery_email = new_email`, `used_at` setzen. Abgelaufen/unbekannt → 302 `…?error=invalid_token`.
- [x] 4.3 `internal/app/router.go`: beide Routen mounten (POST in Authenticated-Tier, GET im Public-Tier).
- [x] 4.4 `h.hub.Broadcast(...)` bei erfolgreicher Schreibung (Profil-Update sichtbar). `go vet` + `gofmt`.

  _Commit:_ `feat(auth): doppelt bestätigte recovery_email-Änderung durch Eltern`

## 5. Backend: Admin/Vorstand-Direkt-Override

- [x] 5.1 `internal/users/handler.go` (`SetRecoveryEmail`): Route `PUT /api/users/{id}/recovery-email`, direkter Write ohne Token/Mail; `h.hub.Broadcast(...)`. Gating `RequireRole("admin")`/`RequireClubFunction("vorstand")` im Router (Vorstand-Tier).
- [x] 5.2 `internal/app/router.go`: Route im Vorstand-Tier mounten.
- [x] 5.3 `go vet` + `gofmt`.

  _Commit:_ `feat(users): Admin/Vorstand setzen recovery_email direkt (Override)`

## 6. Backend: Self-Edit härten + Read-Surfaces

- [x] 6.1 `internal/auth/handler.go` (`UpdateAccount`): sicherstellen, dass `recovery_email` **nicht** ins Update-DTO/SQL aufgenommen wird (kein Schreibpfad fürs Kind).
- [x] 6.2 `internal/members/handler.go` (`GetChildProfile`, `GetProfile`): `recovery_email` des verknüpften Kontos in die Antwort aufnehmen (read-only).
- [x] 6.3 `go vet` + `gofmt`.

  _Commit:_ `feat(members): recovery_email lesbar im Kindprofil, nicht im Self-Edit`

## 7. Backend-Tests

- [x] 7.1 `TestForgotPassword_KindPerLoginName_MailAnRecoveryEmail`
- [x] 7.2 `TestForgotPassword_RecoveryEmailIstKeinLookupKey`
- [x] 7.3 `TestForgotPassword_ErwachsenerUnverändert`
- [x] 7.4 `TestForgotPassword_UnbekannterIdentifier_204OhneToken`
- [x] 7.5 `TestRequestRecoveryEmailChange_MailAnAlteAdresse`
- [x] 7.6 `TestRequestRecoveryEmailChange_FremdesKind_403`
- [x] 7.7 `TestConfirmRecovery_StufeAlt_LöstStufeNeuAus`
- [x] 7.8 `TestConfirmRecovery_StufeNeu_SchreibtRecoveryEmail`
- [x] 7.9 `TestConfirmRecovery_AbgelaufenerToken_RedirectInvalid`
- [x] 7.10 `TestAdminSetRecoveryEmail_DirektOhneWorkflow`
- [x] 7.11 `TestSetRecoveryEmail_OhneFunktion_403`
- [x] 7.12 `TestUpdateAccount_KindKannRecoveryEmailNichtSetzen`
- [x] 7.13 `TestApproveChild_PersistiertRecoveryEmail`
- [x] 7.14 `TestGetChildProfile_ZeigtRecoveryEmail`
- [x] 7.15 Ggf. Fixtures `CreateChildAccount` / `CreateFamilyLink` in `internal/testutil/` ergänzen.

  _Commit:_ `test(auth): recovery_email — Forgot-Password, Doppelbestätigung, Override`

## 8. Frontend: Forgot-Password-Label

- [x] 8.1 `web/src/pages/ForgotPasswordPage.tsx`: Label/Placeholder `„E-Mail oder Nutzername"`, Feld nicht `type=email` erzwingen.

  _Commit:_ `feat(auth): /passwort-vergessen akzeptiert Nutzername`

## 9. Frontend: Anzeige & Änderung im Kindprofil

- [x] 9.1 `web/src/pages/ChildProfilePage.tsx`: `recovery_email` anzeigen + Änderungs-Formular (Eltern) gegen `POST /api/profile/kind/{memberId}/recovery-email`; Hinweis „Bestätigung an alte und neue Adresse nötig". `brand-*`-Tokens, lucide-Icons.
- [x] 9.2 `web/src/pages/ProfilePage.tsx` (Kind-Sicht): `recovery_email` **read-only** anzeigen.
- [x] 9.3 Admin-Nutzerdetail: Direkt-Setzen-Feld für Admin/Vorstand (`PUT /api/users/{id}/recovery-email`).
- [x] 9.4 `useLiveUpdates` abonniert das passende Broadcast-Event; `pnpm -C web build/lint`.

  _Commit:_ `feat(members): recovery_email im Kindprofil anzeigen und ändern`

## 10. Abschluss

- [x] 10.1 `/verify-change` (Build/Test/Lint + Invarianten: Route→Tests, Mutation→Broadcast/useLiveUpdates, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- [ ] 10.2 Proposal archivieren.

  _Commit:_ `docs(openspec): Change kinderaccount-recovery-email archivieren`
