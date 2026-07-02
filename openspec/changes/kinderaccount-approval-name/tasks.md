## 1. Code-Fix (Namen ins Kinder-Konto)

- [x] 1.1 In `approveChildRequest` (`internal/auth/handler.go`) das `INSERT INTO users` um `first_name`, `last_name` erweitern und `firstName`, `lastName` als Parameter übergeben (Reihenfolge: `loginName, firstName, lastName, recoveryEmail`)

## 2. Backfill-Migration

- [x] 2.1 `internal/db/migrations/016_kinderaccount_name_backfill.up.sql`: namenlose Kinder-Konten (`can_login=0 AND email IS NULL AND login_name IS NOT NULL AND COALESCE(first_name,'')=''`) aus `membership_requests` nachfüllen — Match über `LOWER(recovery_email)=LOWER(parent_email) AND is_child=1 AND status='approved'`, disambiguiert über `LOWER(login_name)=LOWER(first_name||'.'||last_name)`
- [x] 2.2 `016_kinderaccount_name_backfill.down.sql`: dokumentierter No-op (Backfill nicht sinnvoll reversibel)

## 3. Tests

- [x] 3.1 Happy-Path: Approve eines `is_child=1`-Antrags → `users.first_name`/`last_name` entsprechen dem Antragsnamen, `login_name` gesetzt, `can_login=0`
- [x] 3.2 Regress: Bestandsverhalten bleibt (Eltern-Mail versandt, kein `family_link`, Antrag-Status `approved`)
- [x] 3.3 Backfill: Migration füllt ein vorab ohne Namen angelegtes Kinder-Konto mit passendem `membership_requests`-Eintrag korrekt; mehrdeutige/nicht matchbare Konten bleiben unverändert

## 4. Verifikation

- [x] 4.1 `/verify-change` + `openspec validate kinderaccount-approval-name --strict`
