## Why

Kinderaccounts ohne eigene E-Mail (`users.email IS NULL`, Login über `login_name` = „Vorname.Nachname") werden heute mit einer **Eltern-E-Mail** registriert. Diese `parent_email` lebt aber **nur** auf der `membership_requests`-Zeile und wird beim Approval **ein einziges Mal** verwendet (Passwort-Setup-Link an die Eltern, `auth/handler.go:437`) — danach weggeworfen. Der entstandene `users`-Datensatz hat dauerhaft `email = NULL`.

Folge: Sobald der Beitrittsantrag archiviert ist, hat das Kind **keine hinterlegte Wiederherstellungsadresse mehr**. `POST /api/auth/forgot-password` matcht strikt auf `email = ? AND can_login = 1` — ein Kind kommt dort gar nicht erst rein. Eine spätere Passwort-Zurücksetzung ist unmöglich.

Diese Änderung persistiert die Eltern-/Wiederherstellungsadresse dauerhaft auf dem Konto, macht sie les- und (kontrolliert) änderbar und schließt damit den Passwort-Reset-Pfad für Kinder.

## What Changes

- **Neue Spalte `users.recovery_email TEXT`** (nullable, **kein** Unique-Index, **nie** ein Login-/Forgot-Password-Lookup-Key). Trägt die Eltern-E-Mail als reine Korrespondenz-/Wiederherstellungsadresse — getrennt von der Login-Identität.
- **Zwei „Qualitäten" von E-Mail** werden explizit getrennt:
  - *AccountName / Nutzeremail* = Login-Identität & Lookup-Key. Erwachsene: `users.email`. Kinder: `users.login_name`.
  - *Wiederherstellungs-/Eltern-E-Mail* = Ziel für Passwort-Mails. Erwachsene: `users.email` (gleicher Wert). Kinder: `users.recovery_email`.
- **Passwort vergessen läuft gleich ab**, nur mit getrennten Qualitäten:
  - Frontend `/passwort-vergessen`: Label `„E-Mail"` → `„E-Mail oder Nutzername"`.
  - `ForgotPassword`: Lookup `WHERE (LOWER(email)=LOWER(?) OR LOWER(login_name)=LOWER(?)) AND can_login=1`; Reset-Mail an `COALESCE(NULLIF(email,''), recovery_email)`. `recovery_email` ist **niemals** Lookup-Key (Eltern-E-Mail trifft den Eltern-Account, nicht das Kind).
- **Änderung der `recovery_email` per doppelter Bestätigung (ALT → NEU)**:
  1. Eltern stoßen die Änderung an → Bestätigungslink an die **aktuelle (alte)** `recovery_email` (autorisiert: nur der Mailbox-Inhaber kann klicken ⇒ Kind kann nicht unkontrolliert umstellen).
  2. Klick auf den ALT-Link löst eine zweite Bestätigung an die **neue** Adresse aus (Erreichbarkeitsnachweis ⇒ kein stiller Lockout durch Tippfehler).
  3. Klick auf den NEU-Link schreibt `users.recovery_email`.
  Reuse von `email_change_tokens` mit Diskriminator-Spalten `field` + `stage`.
- **Admin/Vorstand-Direkt-Override** ohne Workflow: `PUT /api/users/{id}/recovery-email` schreibt direkt. Escape-Hatch für den Fall, dass die alte Registrierungs-Adresse nicht mehr existiert und der Bestätigungs-Loop deshalb tot ist.
- **Kind selbst kann die `recovery_email` NICHT ändern** — das Feld wird im Self-Edit (`PUT /api/profile/account`) nicht exponiert; es ist nur **lesbar** im eigenen Profil und im eingeblendeten Kindprofil der Eltern.
- **Approval-Wiring**: `approveChildRequest` schreibt `parent_email` → `recovery_email` des neuen Kontos.
- **Backfill-Migration**: bestehende Kinderkonten erhalten ihre `recovery_email` aus `membership_requests.parent_email`, soweit ableitbar.

## Capabilities

### Added Capabilities

- `kinderaccount-recovery-email`: Persistente Eltern-/Wiederherstellungs-E-Mail auf Kinderkonten — Speicherung, Anzeige, doppelt bestätigter Änderungs-Workflow, Admin/Vorstand-Override, Forgot-Password-Routing über `login_name`.

### Modified Capabilities

_(keine bestehende Capability ändert ihr Verhalten — der Erwachsenen-Flow `email-aenderung` bleibt unverändert; `email_change_tokens` erhält additive Spalten mit `DEFAULT`, die Bestandszeilen nicht berühren.)_

## Test-Anforderungen

| Route / Capability | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `POST /api/auth/forgot-password` | `TestForgotPassword_KindPerLoginName_MailAnRecoveryEmail` | Kind (`email=NULL`, `login_name` gesetzt, `recovery_email` gesetzt, `can_login=1`): Aufruf mit `login_name` legt Reset-Token an, Mail-Ziel = `recovery_email`. |
| `POST /api/auth/forgot-password` | `TestForgotPassword_RecoveryEmailIstKeinLookupKey` | Aufruf mit der Eltern-E-Mail trifft den **Eltern**-Account (dessen `email`), nie das Kind; kein Token fürs Kind. |
| `POST /api/auth/forgot-password` | `TestForgotPassword_ErwachsenerUnverändert` | Erwachsener mit `email` bekommt Reset-Mail wie bisher an `email`. |
| `POST /api/auth/forgot-password` | `TestForgotPassword_UnbekannterIdentifier_204OhneToken` | Unbekannte E-Mail/Nutzername → HTTP 204, kein Token (keine Enumeration). |
| `POST /api/profile/kind/{memberId}/recovery-email` | `TestRequestRecoveryEmailChange_MailAnAlteAdresse` | Elternteil (via `family_links`) startet Änderung → Token `field=recovery_email,stage=auth`, Bestätigungsmail an die **alte** `recovery_email`. |
| `POST /api/profile/kind/{memberId}/recovery-email` | `TestRequestRecoveryEmailChange_FremdesKind_403` | Nutzer ohne Eltern-Beziehung → HTTP 403, kein Token. |
| `GET /api/profile/recovery-email/confirm` | `TestConfirmRecovery_StufeAlt_LöstStufeNeuAus` | Gültiger ALT-Token (`stage=auth`) → `stage` wird `verify`, zweite Mail an die neue Adresse; `recovery_email` noch **unverändert**. |
| `GET /api/profile/recovery-email/confirm` | `TestConfirmRecovery_StufeNeu_SchreibtRecoveryEmail` | Gültiger NEU-Token (`stage=verify`) → `users.recovery_email` = neue Adresse, Token `used_at` gesetzt. |
| `GET /api/profile/recovery-email/confirm` | `TestConfirmRecovery_AbgelaufenerToken_RedirectInvalid` | Abgelaufener/unbekannter Token → 302 `…?error=invalid_token`, keine Schreibung. |
| `PUT /api/users/{id}/recovery-email` | `TestAdminSetRecoveryEmail_DirektOhneWorkflow` | Admin/Vorstand → `users.recovery_email` sofort gesetzt, **kein** Token, HTTP 204. |
| `PUT /api/users/{id}/recovery-email` | `TestSetRecoveryEmail_OhneFunktion_403` | Caller ohne `admin`/`vorstand` → HTTP 403. |
| `PUT /api/profile/account` | `TestUpdateAccount_KindKannRecoveryEmailNichtSetzen` | `recovery_email` im Body wird ignoriert / nicht geschrieben (Self-Edit exponiert das Feld nicht). |
| `POST /api/auth/approve-membership-request/{id}` | `TestApproveChild_PersistiertRecoveryEmail` | Nach Approval eines Kind-Antrags hat das neue `users`-Konto `recovery_email = parent_email` des Antrags. |
| `GET /api/profile/kind/{memberId}` | `TestGetChildProfile_ZeigtRecoveryEmail` | Antwort enthält die `recovery_email` des Kindkontos (lesbar für verknüpfte Eltern). |

**Garantierte Invarianten:**
1. `recovery_email` ist **niemals** Lookup-Key für Login oder Forgot-Password — ausschließlich Ziel-Adresse.
2. Eine `recovery_email`-Änderung wird genau dann wirksam, wenn **beide** Bestätigungen vorliegen (ALT autorisiert, NEU erreichbar) **oder** ein Admin/Vorstand sie direkt setzt.
3. Das Kind selbst kann seine `recovery_email` nicht schreiben.

## Impact

- **Migration:** `internal/db/migrations/004_recovery_email.up.sql` (+ `.down.sql`):
  - `ALTER TABLE users ADD COLUMN recovery_email TEXT;`
  - `ALTER TABLE email_change_tokens ADD COLUMN field TEXT NOT NULL DEFAULT 'email';`
  - `ALTER TABLE email_change_tokens ADD COLUMN stage TEXT;` (NULL = klassischer einstufiger Erwachsenen-Flow)
  - Backfill: `UPDATE users SET recovery_email = (SELECT mr.parent_email FROM membership_requests mr …)` für Kinderkonten, soweit eindeutig ableitbar.
- **Backend:**
  - `internal/auth/handler.go` — `ForgotPassword` (Lookup `email OR login_name`, Ziel `COALESCE`); `approveChildRequest` (persistiert `recovery_email`); neue Handler `RequestRecoveryEmailChange`, `ConfirmRecoveryEmailChange`; `UpdateAccount` exponiert `recovery_email` **nicht**.
  - `internal/users/handler.go` (oder members) — `SetRecoveryEmail` (Admin/Vorstand-Direkt-Override).
  - `internal/members/handler.go` — `GetChildProfile` / `GetProfile` geben `recovery_email` mit aus.
  - `internal/app/router.go` — neue Routen (Authenticated: `POST /api/profile/kind/{memberId}/recovery-email`; Public: `GET /api/profile/recovery-email/confirm`; Vorstand: `PUT /api/users/{id}/recovery-email`).
- **Frontend:**
  - `web/src/pages/ForgotPasswordPage.tsx` — Label/Placeholder `„E-Mail oder Nutzername"`, Feld nicht mehr `type=email`-erzwungen.
  - `web/src/pages/ChildProfilePage.tsx` — `recovery_email` anzeigen + Änderungs-Formular (Eltern), Hinweis auf Doppelbestätigung.
  - `web/src/pages/ProfilePage.tsx` (Kind-Sicht) — `recovery_email` **read-only** anzeigen.
  - ggf. Admin-Nutzerdetail — Direkt-Setzen-Feld für Admin/Vorstand.
- **Bewusste Trade-offs:**
  - Doppelbestätigung kostet zwei Klicks; akzeptiert, weil seltene Operation.
  - Tote alte Adresse blockiert den Loop → nur Admin/Vorstand-Override behebt das (per Hand, identität außerhalb des Systems geprüft).
- **Tests:** Fixtures `CreateUser`, `CreateMember`, `CreateRefreshToken`, `CreatePasswordResetToken` vorhanden; ggf. Helper `CreateChildAccount` / `CreateFamilyLink` ergänzen.
