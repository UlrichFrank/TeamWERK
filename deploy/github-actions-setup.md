# GitHub-Actions Release + Deploy — Setup

Zwei Workflows, eine Kette:

1. `.github/workflows/release.yml` läuft bei jedem `push` auf `main`, liest die
   Conventional Commits seit dem letzten `vX.Y.Z`-Tag, bestimmt den Bump
   (`feat`→minor, `fix`/`perf`→patch, `!:`/`BREAKING CHANGE:`→major), setzt das
   Tag und legt ein GitHub-Release an.
2. `.github/workflows/deploy.yml` läuft bei jedem Tag-Push `v[0-9]+.[0-9]+.[0-9]+`:
   `resolve` → `gate` → `backup` → `deploy`. `concurrency: deploy-prod`
   verhindert parallele Prod-Deploys.

Manueller Override:
- `release.yml` → `workflow_dispatch` mit `force_version=v1.2.3`.
- `deploy.yml`  → `workflow_dispatch` mit `tag=v1.2.3` (z. B. Rollback).

## Erforderliche Secrets (Repo → Settings → Secrets and variables → Actions → Secrets)

| Name | Inhalt |
|---|---|
| `DEPLOY_SSH_PRIVATE_KEY` | Privater SSH-Key (ed25519), dessen Public-Key in `~/.ssh/authorized_keys` des Deploy-Users auf dem VPS liegt. **Eigener Key nur für CI**, nicht den persönlichen Dev-Key. |
| `DEPLOY_SSH_KNOWN_HOSTS` | Output von `ssh-keyscan -t ed25519,rsa <vps-host>` (mehrere Zeilen ok). |
| `DEPLOY_REMOTE` | SSH-Ziel wie für `make deploy`, z. B. `deploy@217.160.118.39`. Wird in `.env` als `REMOTE=` geschrieben. |

## Optionale Variables (Repo → Settings → Secrets and variables → Actions → Variables)

| Name | Default | Zweck |
|---|---|---|
| `DEPLOY_REMOTE_DIR` | `/usr/local/bin` | Zielverzeichnis des Binaries (entspricht `REMOTE_DIR` in `make deploy`). |
| `DEPLOY_DB_PATH` | `/var/lib/teamwerk/teamwerk.db` | Quelle für `sqlite3 .backup`. |
| `DEPLOY_BACKUP_DIR` | `/var/lib/teamwerk/backups` | Zielordner für `pre-deploy-<tag>-<stamp>.db`. |
| `HEALTHZ_URL` | (leer) | Wenn gesetzt, läuft nach `make deploy` ein Smoke-Check mit Retry. Gleiche Variable wie in `uptime.yml`. |

## Voraussetzungen auf dem VPS (einmalig)

Der CI-Deploy-User muss **passwortlos** sudo dürfen für die Befehle, die
`make deploy` und das Backup-Step nutzen:

```sudoers
deploy ALL=(ALL) NOPASSWD: /usr/bin/sqlite3, /bin/mkdir, /bin/ls, /bin/mv, /bin/chown, /bin/systemctl, /usr/bin/tee
```

Backup-Verzeichnis muss vom Deploy-User beschreibbar oder per sudo anlegbar
sein — Workflow legt es bei Bedarf via `sudo mkdir -p` an.

## GitHub-Environment "production"

Beide privilegierten Jobs (`backup`, `deploy`) sind an das Environment
`production` gebunden. Empfehlung: Repo → Settings → Environments → `production`
mit **Required reviewers** versehen, damit jeder Deploy manuell abgenickt werden
muss. Die Secrets oben dann **am Environment** statt am Repo hinterlegen, dann
sind sie nur für Jobs mit `environment: production` zugreifbar.

## Erst-Release

Ohne existierendes Tag erzeugt `scripts/next-version.sh` `v0.1.0` (sofern
Commits vorliegen). Wer mit höherer Startversion einsteigen will, einmal manuell:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Danach übernimmt der `release`-Workflow automatisch.

## Test ohne Prod-Risiko

`scripts/next-version.sh` lokal ausführen:

```bash
scripts/next-version.sh          # gibt nächste Version aus
scripts/next-version.sh --check  # Exit 0 wenn Bump anfällt, sonst 1
```
