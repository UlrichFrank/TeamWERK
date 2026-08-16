# GitHub-Actions Release + Deploy — Setup

Zwei Workflows, eine Kette:

1. `.github/workflows/release.yml` läuft bei jedem `push` auf `main`, liest die
   Conventional Commits seit dem letzten `vX.Y.Z`-Tag, bestimmt den Bump
   (`feat`→minor, `fix`/`perf`→patch, `!:`/`BREAKING CHANGE:`→major), setzt das
   Tag und legt ein GitHub-Release an.
2. `.github/workflows/deploy.yml` läuft bei jedem Tag-Push `v[0-9]+.[0-9]+.[0-9]+`:
   `resolve` → `gate` → `backup` → `deploy`. `concurrency: deploy-prod`
   verhindert parallele Prod-Deploys.

Daneben räumt `.github/workflows/cleanup-runs.yml` täglich um 03:17 UTC
abgeschlossene Workflow-Runs weg, die älter als 3 Tage sind (`workflow_dispatch`
mit `days` und `dry_run` zum Nachjustieren). Angefasst wird nur
`status=completed` — ein Deploy, der vor dem `production`-Environment auf
Freigabe wartet, bleibt also stehen. Was dabei verschwindet, sind Run-Logs;
Tags, GitHub-Releases und die VPS-seitigen `pre-deploy-*.db`-Backups bleiben.
Sollen die Prod-Deploy-Logs länger nachvollziehbar bleiben, trägt man
`.github/workflows/deploy.yml` in die `PROTECTED`-Variable des Workflows ein
(Default: leer, also keine Ausnahme).

## Taggen + Deployen in einem Zug (der normale manuelle Weg)

Actions → **release** → *Run workflow* auf `main`, `deploy` auf **true**. Der Lauf
legt das nächste Tag an, erstellt das GitHub-Release und ruft `deploy.yml` als
reusable Workflow mit genau diesem Tag auf — `resolve` → `gate` → `backup` →
`deploy` läuft also vollständig durch, ohne dass irgendwo ein Tag von Hand
eingetippt wird.

| Input | Default | Bedeutung |
|---|---|---|
| `bump` | `patch` | **Untergrenze**, kein Zwang: die Conventional Commits entscheiden weiter über die Höhe. Ein `feat:` seit dem letzten Tag ergibt minor, ein `!:`/`BREAKING CHANGE:` major — auch bei `bump=patch`. Umgekehrt greift die Untergrenze, wenn die Analyse gar nichts hergibt. |
| `force_version` | (leer) | Tag hart setzen (`v1.2.3`), überspringt die Analyse. Ein bereits vergebenes Tag bricht den Lauf mit klarer Meldung ab. |
| `deploy` | `false` | Nach dem Taggen sofort auf Prod deployen. |

Der Unterschied zum Push-Trigger liegt nur im Leerlauf-Fall: findet der
**automatische** Lauf keinen release-relevanten Commit, überspringt er das Tag
(kein Rauschen). Der **manuelle** Lauf erzeugt trotzdem eins — sonst wäre
„manuell taggen + deployen" ausgerechnet dann blockiert, wenn man es braucht,
etwa nach einem Squash-Merge, dessen Titel der Branch-Name ist
(„Feat/event log (#193)") und der damit nicht als `feat:` durchgeht.

Weitere manuelle Wege:
- `deploy.yml` → `workflow_dispatch` mit `tag=v1.2.3` — deployt ein **bestehendes**
  Tag erneut, ohne ein neues anzulegen (Rollback auf eine ältere Version).

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
scripts/next-version.sh              # gibt nächste Version aus
scripts/next-version.sh --check      # Exit 0 wenn Bump anfällt, sonst 1
scripts/next-version.sh --min patch  # wie der manuelle release-Lauf: nie „kein Bump"
```

Gibt `scripts/next-version.sh` ohne Flags das **bestehende** Tag zurück, würde
ein Push-Lauf nichts taggen — dann ist `--min` (bzw. der manuelle Trigger) der
Weg zum Release.
