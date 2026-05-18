# Design: Umbenennung VereinsWERK → TeamWERK

**Datum:** 2026-05-18  
**Status:** Genehmigt

## Ziel

Vollständige Umbenennung der Plattform von „VereinsWERK" zu „TeamWERK" — sowohl Anzeigenamen als auch alle technischen Bezeichner (Binary, Go-Modul, Service, VPS-Pfade).

## Umfang

### Phase 1 – Codebase (automatisiert)

| Kategorie | Vorher | Nachher |
|-----------|--------|---------|
| Anzeigename (UI, E-Mails, Titel) | VereinsWerk / VereinsWERK | TeamWERK |
| Go-Modulpfad | `github.com/teamstuttgart/vereinswerk` | `github.com/teamstuttgart/teamwerk` |
| `cmd/`-Verzeichnis | `cmd/vereinswerk/` | `cmd/teamwerk/` |
| Binary-Name | `vereinswerk` | `teamwerk` |
| systemd-Service-Datei | `deploy/vereinswerk.service` | `deploy/teamwerk.service` |
| VPS-Pfade (in Scripts) | `/etc/vereinswerk/`, `/var/lib/vereinswerk/` | `/etc/teamwerk/`, `/var/lib/teamwerk/` |
| Scheduler-Log | `/var/log/vereinswerk-scheduler.log` | `/var/log/teamwerk-scheduler.log` |
| DB-Dateiname (Default) | `vereinswerk.db` | `teamwerk.db` |
| npm-Packagename | `vereinswerk-web` | `teamwerk-web` |
| Lokales Verzeichnis | `/Users/ulrich/Dev/vereinswerk` | `/Users/ulrich/Dev/teamwerk` |
| GitHub-Repo | `teamstuttgart/vereinswerk` | `teamstuttgart/teamwerk` (via `gh repo rename`) |

**Betroffene Dateien:**
- `go.mod` — Modulpfad
- `cmd/vereinswerk/main.go` → `cmd/teamwerk/main.go` (Verzeichnis umbenennen)
- Alle `internal/**/*.go` — Import-Pfade
- `web/package.json` — `name`-Feld
- `web/index.html` — `<title>`
- `web/src/components/AppShell.tsx` — Sidebar-Name
- `web/src/pages/LoginPage.tsx` — Heading
- `web/src/pages/RequestMembershipPage.tsx` — Untertitel
- `internal/auth/handler.go` — E-Mail-Betreffzeilen und Nachrichtentexte
- `internal/config/config.go` — Default-Werte (DB_PATH, SMTP_FROM)
- `deploy/vereinswerk.service` → `deploy/teamwerk.service`
- `deploy/setup-vps.sh` — alle Pfade und Service-Referenzen
- `Makefile` — Binary-Name, Build-Pfade, Service-Name
- `.env` und `.env.example` — DB_PATH, SMTP_FROM
- `CLAUDE.md` — alle Referenzen
- `openspec/config.yaml` — Projektname
- Lokale DB-Datei `./vereinswerk.db` → `./teamwerk.db` (falls vorhanden)

### Phase 2 – VPS-Migration (manuell per SSH)

Nach dem Deploy muss auf dem VPS einmalig ausgeführt werden:

```bash
systemctl stop vereinswerk
mv /etc/vereinswerk /etc/teamwerk
mv /var/lib/vereinswerk /var/lib/teamwerk
mv /var/lib/teamwerk/vereinswerk.db /var/lib/teamwerk/teamwerk.db
# /etc/teamwerk/env: DB_PATH anpassen auf /var/lib/teamwerk/teamwerk.db
systemctl disable vereinswerk
make deploy
systemctl enable teamwerk && systemctl start teamwerk
# Crontab aktualisieren: vereinswerk → teamwerk
crontab -e
```

### GitHub-Repo umbenennen

```bash
gh repo rename teamwerk
git remote set-url origin https://github.com/teamstuttgart/teamwerk.git
```

## Tagline

„TeamWERK — Where Engagement Really Klicks" — WERK-Akronym bleibt gültig, T-E-A-M ist das natürliche Präfix.

## Nicht im Scope

- Änderungen an der TYPO3-Hauptsite (`team-stuttgart.org`)
- DNS-Änderungen (Domain bleibt `intern.team-stuttgart.org`)
- Datenbankschema (keine Spalten oder Tabellen heißen „vereinswerk")
