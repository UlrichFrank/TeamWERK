## Why

Der TeamWERK-Produktionsserver soll vom privaten IONOS-VPS (`intern.team-stuttgart.org`, 217.160.118.39) auf einen Team-Stuttgart-eigenen Server (`teamwerk.team-stuttgart.org`, 31.70.110.19) umziehen. Aktuell existiert für so einen VPS-Wechsel nur ein Grüne-Wiese-Runbook (`deploy/vps-setup-runbook.md`) plus manuelle `scp`-Schritte für Backups — das reicht für einen einmaligen Aufbau, aber nicht für einen begleiteten Umzug mit Test-Phase und wiederkehrender Wiederholbarkeit (z. B. bei künftigen Provider-Wechseln oder Disaster-Recovery-Übungen).

## What Changes

- **Neues Makefile-Target `server-bootstrap NEW_REMOTE=<alias>`**: richtet einen frischen Zielserver ein (`setup-vps.sh`), klont Env-Datei + Secrets vom aktuellen Produktionshost (mit `BASE_URL`-Rewrite auf die neue Domain), zieht einen konsistenten Snapshot der DB (`sqlite3 .backup`) und aller Storage-Ordner (`uploads`, `files`, `videos`, `beitragslauf-protokolle`) und deployt das Binary. Idempotent.
- **Neues Makefile-Target `server-sync-data NEW_REMOTE=<alias>`**: wiederholbar. Frischer DB- und Storage-Sync vom Quell-Host zum Ziel-Host mit Bestätigungsdialog („Testdaten auf Ziel werden überschrieben — ok?"). Muss `migrate up` **nach** dem Kopieren ausführen, damit ein Ziel-Schema, das schon voraus ist, nicht auf einen älteren Snapshot trifft.
- **Neues Makefile-Target `server-cutover NEW_REMOTE=<alias>`**: finaler Umschalter. Ruft intern `server-sync-data` auf, stoppt und deaktiviert den `teamwerk`-Service auf dem Quell-Host, ersetzt dessen Nginx-Konfiguration durch einen 301-Redirect (Template `deploy/nginx-redirect.conf`) auf die neue Domain, reloadet Nginx. Ausgabe: Erinnerung an Better-Stack-Monitor-Umhängen und User-Kommunikation.
- **Neue Datei `deploy/nginx-redirect.conf`**: Nginx-Template, das den alten Host für **alle** Pfade (inkl. `/api/*`) auf `https://teamwerk.team-stuttgart.org$request_uri` per 301 weiterleitet, mit `Cache-Control: no-store` gegen dauerhaftes Browser-Caching.
- **Neue Datei `deploy/server-migration-runbook.md`**: Ablauf für den konkreten Umzug (DNS-A-Record setzen, Certbot auf Ziel-Host, Better-Stack-Monitor-URLs umhängen, User-Kommunikation zur PWA-Neuinstallation).
- **`.env`-Erweiterung um `REMOTE_NEW`, `REMOTE_NEW_DIR` und `BASE_URL_NEW`**: die Migrations-Targets lesen `NEW_REMOTE`/`NEW_REMOTE_DIR` erst aus Kommandozeilen-Argumenten, fallen dann auf `REMOTE_NEW`/`REMOTE_NEW_DIR` aus `.env` zurück (analog zum bestehenden `REMOTE`/`REMOTE_DIR`). Die neue Domain kommt ausschließlich aus `BASE_URL_NEW` in `.env` (kein CLI-Fallback), damit der Wert für alle Läufe konsistent ist und der Betreuer beim wiederholten `server-sync-data` nichts vergessen kann.

**Nicht Teil dieser Änderung** (mechanisch nicht automatisierbar):
- DNS-A-Record-Wechsel (Provider-Panel)
- Certbot-Erstlauf auf Ziel-Host (braucht bereits gesetzten DNS)
- Umhängen der Better-Stack-Monitor-URLs (externe API, Handarbeit im Panel)
- Neuinstallation der PWA + neue Push-Erlaubnis durch die Nutzer (Origin-gebunden im Browser)

## Capabilities

### New Capabilities
- `vps-migration`: wiederkehrender, skript-getriebener Umzug einer TeamWERK-Instanz von einem Quell-VPS auf einen Ziel-VPS mit optionaler Test-Phase (Variante A: initialer Bootstrap → beliebig oft wiederholbarer Daten-Sync während der Testphase → finaler Cutover mit Redirect vom Alt-Host).

### Modified Capabilities

(keine — `vps-deployment` beschreibt den Ziel-Host-Zustand und bleibt unverändert; `vps-migration` beschreibt den *Wechsel-Vorgang* zwischen zwei solchen Hosts als eigenständige Capability.)

## Impact

- **`Makefile`**: drei neue Targets, ein `.PHONY`-Eintrag pro Target. Fällt Target ohne `NEW_REMOTE=` auf, Fehlermeldung mit Beispielaufruf.
- **`deploy/`**: `nginx-redirect.conf` (neu), `server-migration-runbook.md` (neu).
- **`.env.example`**: dokumentiert `REMOTE_NEW` und `REMOTE_NEW_DIR` als optionale Variablen (bleiben ungesetzt, solange kein Umzug ansteht).
- **`docs/agent/10-deployment.md`**: Verweis auf das neue Runbook, kurze Erwähnung der drei Targets.
- **Keine Änderung an Go-Code, Migrationen, Frontend, DB-Schema oder Auth-Modell.** Es handelt sich um reine Operationstooling-Änderung.
- **Sicherheit**: `server-bootstrap` überträgt `JWT_SECRET`, `VAPID_PRIVATE_KEY`, `VIDEO_STREAM_SECRET` und SMTP-Passwörter vom Quell- zum Ziel-Host via `scp`/SSH. Kein Zwischenspeicher auf dem Entwickler-Laptop (Env-Datei wird per Pipe direkt weitergereicht). Der Zero-Knowledge-Bankdaten-Schutz bleibt intakt, weil `clubs.group_private_key_enc` als opaker Blob mitkopiert wird und die Tresor-Passphrase den Browser nie verlässt.
- **Ausfallzeit**: Bootstrap = 0 min (Produktion läuft weiter auf Quell-Host). `server-sync-data` = 0 min (Quell-Host bleibt online, `.backup` ist WAL-safe). `server-cutover` = Sekunden auf Quell-Seite (Nginx-Reload); von außen sichtbar ist erst dann, wenn Nutzer den Redirect vom Alt-Host folgen bzw. den DNS-Wechsel bemerken.
