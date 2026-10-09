# Proposal: beta-branch-prozess

## Why

Mit der Beta-Instanz (`kachel-eckfahne`, `beta-prod-spiegel`) gibt es eine zweite Umgebung, aber keinen Prozess dafür: Beta-Deploys laufen nur über Sondertargets (`deploy-beta`) vom beliebigen Arbeitsstand, Prod-Pflege (Migrationen, Admin anlegen, Backup, Rollback) gibt es nur für Prod. Gewünscht ist ein zweiter langlebiger Branch `beta` neben `main`: eine Änderung geht entweder direkt nach `main` oder erst über `beta` nach `main`, und alles, was es für Prod gibt, soll es auch für die Beta geben — ohne dass die Beta Prod je beeinflusst.

## What Changes

- **`ENV=prod|beta` an allen Server-Targets** im Makefile statt Doppel-Targets: `deploy`, `deploy-rollback`, `migrate-remote-up`, `create-admin-remote`, `push-test-remote`, `backup`, `backup-files`, `backup-videos`, `pull-*`/`restore-local*`, `setup-vps`, `dev-remote`. Ohne `ENV` bleibt alles Prod wie bisher (Prod-Deploy als Trockenlauf gegen den alten Stand verglichen: dieselben Befehle).
- **Branch-Bindung** vor dem Build: Prod nur von `main` oder einem `vX.Y.Z`-Tag, Beta nur vom Branch `beta`, jeweils ohne uncommittete Änderungen; Notausgang `ALLOW_BRANCH=1`. Rollback bleibt ungebunden.
- **Schutz gegen Fehlgriffe**: `_check-env` bricht bei `ENV=beta` ab, wenn ein Pfad/Dienst nicht `teamwerk-beta` enthält; Prod-only-Targets (Server-Umzug) lehnen `ENV=beta` ab; Beta-Backups liegen unter `backup/beta/`, damit `restore-local` nie einen Beta-Stand für Prod hält; schreibende Remote-Schritte der Beta laufen als `www-data`.
- **GitHub Actions**: neuer Workflow `deploy-beta.yml` (Push auf `beta` → Gate wie Prod → `make deploy ENV=beta` → Smoke über nginx, Environment `beta`, eigene Concurrency-Gruppe); `ci.yml` läuft zusätzlich für `beta`. `release.yml`/`deploy.yml` bleiben unverändert (Prod nur über Tags von `main`).
- `deploy-beta` bleibt als Kurzform, `setup-beta` = `setup-vps ENV=beta`; `seed-beta`, `mirror-beta` bleiben Beta-only.
- Doku: `docs/agent/02-workflow.md`, `docs/agent/10-deployment.md`, `deploy/github-actions-setup.md`.

## Capabilities

- `beta-umgebung` (ADDED): Branch-Prozess und Umgebungs-Parameter.

## Impact

- Prod-Deploy über `deploy.yml` (Tag-Checkout) besteht die Branch-Bindung unverändert. Ein manueller `make deploy` von einem Feature-Branch bricht künftig ab (gewollt).
- Einmaliger manueller Schritt in GitHub: Environment `beta` mit den drei Deploy-Secrets.

## Test-Anforderungen

Keine Routen. Geprüft: Trockenlauf `make -n deploy` Prod alt vs. neu (gleiche Remote-Befehle), Abbruch von `make deploy ENV=beta` auf einem anderen Branch bzw. mit uncommitteten Änderungen, Abbruch von `ENV=staging` und `server-cutover ENV=beta`, echter `make deploy ENV=beta` vom Branch `beta` mit unveränderter Prod-PID.
