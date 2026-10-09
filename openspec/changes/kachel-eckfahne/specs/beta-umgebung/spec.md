## ADDED Requirements

### Requirement: Getrennte Beta-Instanz
Es SHALL eine Vorschau-Instanz unter `beta.teamwerk.team-stuttgart.org` geben, die auf demselben VPS läuft, aber keinen Zustand mit Prod teilt: eigener systemd-Dienst `teamwerk-beta`, eigener Port (8081), eigene Env-Datei `/etc/teamwerk-beta/env`, eigene SQLite-DB und eigene Storage-Verzeichnisse unter `/var/lib/teamwerk-beta/`, eigene Video-Ablage. `make deploy-beta` MUSS ausschließlich diese Pfade schreiben und ausschließlich `teamwerk-beta` neu starten.

#### Scenario: Beta-Deploy lässt Prod unberührt
- **WHEN** `make deploy-beta` läuft
- **THEN** bleiben `/usr/local/bin/teamwerk`, `/etc/teamwerk/env`, `/var/lib/teamwerk/` und der Dienst `teamwerk` unverändert

#### Scenario: Eigene Sitzung
- **WHEN** sich jemand auf der Beta anmeldet
- **THEN** gilt das Refresh-Cookie nur für den Beta-Host (host-only), eine Prod-Sitzung im selben Browser bleibt bestehen

### Requirement: Beta ohne Außenwirkung
Die Beta-Instanz SHALL keine Wirkung außerhalb ihrer selbst erzeugen: `MAILER_DISABLED=true`, keine VAPID-Schlüssel (kein Push), `BWHV_ORG_ID=0` (kein Verbandsabruf), kein Scheduler-Cron, keine Matomo-Erfassung (das Frontend erfasst nur auf dem Prod-Host).

#### Scenario: Keine Mail aus der Beta
- **WHEN** auf der Beta eine Aktion eine Benachrichtigung auslöst
- **THEN** wird keine E-Mail und keine Push-Nachricht versendet

#### Scenario: Keine Analytics aus der Beta
- **WHEN** eine Seite auf `beta.teamwerk.team-stuttgart.org` geladen wird
- **THEN** wird Matomo nicht initialisiert
