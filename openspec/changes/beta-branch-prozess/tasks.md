## 1. Makefile

- [x] 1.1 ENV-Block (prod|beta) mit Pfaden, Dienst, Port, Branch, Deploy-Hash, Backup-Ziel
- [x] 1.2 `deploy`/`deploy-rollback`/`_smoke` generisch; Prod-only-Teile (Erst-Env, Unit, Backup-Cron, HSTS) unter `ifeq prod`
- [x] 1.3 `_check-branch`, `_check-env`, `_check-prod-only`
- [x] 1.4 Übrige Server-Targets mit `ENV` (migrate/create-admin/push-test/backup*/setup-vps/dev-remote), `restore-local*` über `BACKUP_ROOT`

## 2. GitHub Actions

- [x] 2.1 `deploy-beta.yml` (Push auf beta, Gate, Deploy, Smoke)
- [x] 2.2 `ci.yml` auch für beta
- [ ] 2.3 Environment `beta` + Secrets in GitHub anlegen (manuell)

## 3. Branch und Verifikation

- [x] 3.1 Doku (02-workflow, 10-deployment, github-actions-setup)
- [x] 3.2 Branch `beta` anlegen, `make deploy ENV=beta` vom Branch beta, Prod-PID unverändert
