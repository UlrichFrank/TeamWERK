## 1. Gemeinsamer Validator

- [x] 1.1 `validatePassword(pw string) error` in `internal/auth` ergänzen: Mindestlänge ≥ 12 Zeichen, ≤ 72 Byte; aussagekräftige Fehler für 400-Mapping
- [x] 1.2 Mindestlänge optional über `internal/config` konfigurierbar (Default 12)

## 2. Handler verdrahten

- [x] 2.1 Register-Handler (`handler.go:528`): `validatePassword` vor `bcrypt.GenerateFromPassword`; bei Fehler 400
- [x] 2.2 ResetPassword-Handler (`:644`): `validatePassword` vor dem Setzen; bei Fehler 400, kein `can_login=1`-Aktivieren des Kind-Accounts
- [x] 2.3 ChangePassword-Handler (`:1190`): `validatePassword` vor dem Setzen; bei Fehler 400

## 3. Frontend spiegeln

- [x] 3.1 `RegisterPage.tsx`, `ResetPasswordPage.tsx` und Passwort-Ändern-Dialog: `minLength`/Hinweistext auf ≥ 12 Zeichen; Server-400 als Fehlermeldung anzeigen

## 4. Tests & Verifikation

- [x] 4.1 Register: gültiges Passwort → 2xx; < 12 Zeichen → 400; > 72 Byte → 400
- [x] 4.2 ResetPassword: < 12 Zeichen → 400 und Account bleibt deaktiviert; gültig → 2xx
- [x] 4.3 ChangePassword: < 12 Zeichen → 400; gültig → 2xx
- [x] 4.4 `/verify-change` + `openspec validate server-password-policy --strict`
