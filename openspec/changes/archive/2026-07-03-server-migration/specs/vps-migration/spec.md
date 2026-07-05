## ADDED Requirements

### Requirement: Migration targets are invoked with an explicit destination host
Alle drei Migrations-Makefile-Targets (`server-bootstrap`, `server-sync-data`, `server-cutover`) MUST/MÜSSEN den Ziel-Host als `NEW_REMOTE=<ssh-alias>` erwarten, mit optionalem Fallback auf `REMOTE_NEW` aus `.env`. Fehlt beides, MUST/MUSS das Target mit klarer Fehlermeldung inkl. Aufrufbeispiel abbrechen, ohne Änderungen vorzunehmen.

#### Scenario: Zielhost fehlt komplett
- **WHEN** ein Migrations-Target ohne `NEW_REMOTE=…`-Argument und ohne `REMOTE_NEW=…` in `.env` aufgerufen wird
- **THEN** bricht das Target vor jeder Netzwerk- oder Dateisystemaktion ab und gibt eine Fehlermeldung mit dem korrekten Aufrufbeispiel aus (`make server-bootstrap NEW_REMOTE=vServerNeu`)

#### Scenario: Zielhost aus `.env`
- **WHEN** `REMOTE_NEW=vServerNeu` in `.env` steht und ein Migrations-Target ohne `NEW_REMOTE=`-Argument läuft
- **THEN** verwendet das Target `vServerNeu` als Zielhost und protokolliert die Herkunft (`REMOTE_NEW aus .env`) in der ersten Ausgabezeile

#### Scenario: CLI-Argument überschreibt `.env`
- **WHEN** `REMOTE_NEW=vServerAlt` in `.env` steht und das Target mit `make server-bootstrap NEW_REMOTE=vServerNeu` aufgerufen wird
- **THEN** verwendet das Target `vServerNeu`, nicht `vServerAlt`

---

### Requirement: `server-bootstrap` provisioniert und initialisiert den Zielhost idempotent
`make server-bootstrap NEW_REMOTE=<alias>` MUST/MUSS auf einem frischen oder bereits einmal bootstrap-ten Zielhost erfolgreich durchlaufen. Es MUST/MUSS: (1) `deploy/setup-vps.sh` auf dem Zielhost ausführen, (2) die Env-Datei vom aktuellen Produktionshost (`REMOTE`) klonen und dabei ausschließlich die Zeile `BASE_URL=…` auf `https://<neue-domain>` umschreiben, (3) einen konsistenten `sqlite3 .backup`-Snapshot der Quell-DB nach `/var/lib/teamwerk/teamwerk.db` auf dem Ziel schreiben, (4) die Storage-Ordner (`uploads`, `files`, `videos`, `beitragslauf-protokolle`) per `rsync -az` vom Quell- zum Zielhost übertragen, (5) das TeamWERK-Binary via `make deploy` (mit umgebogenem `REMOTE`) deployen inkl. `migrate up`, (6) am Ende `curl -k -H "Host: <neue-domain>" https://<ziel-ip>/api/healthz` aufrufen und den Erfolg (`"status":"ok"`) verifizieren.

#### Scenario: Frischer Zielhost
- **WHEN** `server-bootstrap` gegen einen leeren VPS läuft
- **THEN** ist am Ende `/api/healthz` auf dem Ziel-Host über IP + Host-Header erreichbar und liefert `"status":"ok"` mit `"db":"ok"`

#### Scenario: Zweiter Bootstrap-Lauf auf demselben Zielhost
- **WHEN** `server-bootstrap` erneut gegen einen bereits bootstrap-ten Zielhost läuft
- **THEN** läuft das Target ohne Fehler durch (setup-vps ist idempotent; Env, DB und Storage werden mit frischem Snapshot überschrieben; `migrate up` ist no-op wenn Schema aktuell)

#### Scenario: BASE_URL wird korrekt umgeschrieben
- **WHEN** die Quell-Env `BASE_URL=https://intern.team-stuttgart.org` enthält und `server-bootstrap` mit Ziel-Domain `teamwerk.team-stuttgart.org` läuft
- **THEN** enthält `/etc/teamwerk/env` auf dem Zielhost `BASE_URL=https://teamwerk.team-stuttgart.org`, und ALLE anderen Zeilen (`JWT_SECRET`, `VAPID_*`, `SMTP_*`, `VIDEO_STREAM_SECRET`, `METRICS_TOKEN`) sind byte-identisch zur Quelle

#### Scenario: Env-Datei berührt nie die Entwickler-Disk
- **WHEN** `server-bootstrap` das Env-Klonen durchführt
- **THEN** wird die Env-Datei per SSH-Pipe direkt vom Quell- zum Ziel-Host übertragen und nicht in eine lokale Datei (auch nicht temporär in `/tmp` des Entwicklerrechners) geschrieben

---

### Requirement: `server-sync-data` überträgt einen frischen Snapshot mit Bestätigung
`make server-sync-data NEW_REMOTE=<alias>` MUST/MUSS beliebig oft wiederholbar sein und einen frischen `sqlite3 .backup`-Snapshot plus rsync der Storage-Ordner vom Quell- zum Zielhost übertragen. Vor jeder Aktion MUST/MUSS es eine `[y/N]`-Bestätigung einholen, dass bestehende Testdaten auf dem Ziel überschrieben werden. Die Sequenz MUST/MUSS sein: (1) Ziel-`teamwerk`-Service stoppen, (2) DB-Snapshot schreiben, (3) Storage-Ordner rsyncen, (4) `migrate up` auf Ziel, (5) Ziel-Service starten, (6) Smoke-Test `/api/healthz`.

#### Scenario: Standard-Sync
- **WHEN** `server-sync-data` läuft und der Betreuer die Bestätigung mit `y` bestätigt
- **THEN** enthält die Ziel-DB nach Abschluss den Quell-Stand vom Zeitpunkt des `.backup`-Aufrufs, und der Ziel-Service läuft mit `/api/healthz` = ok

#### Scenario: Betreuer bricht ab
- **WHEN** `server-sync-data` die Bestätigung anzeigt und der Betreuer nichts oder `n` eingibt
- **THEN** beendet sich das Target mit Exit-Code 1, ohne Ziel-Service zu stoppen oder Ziel-DB zu berühren

#### Scenario: `migrate up` läuft nach Snapshot, nicht davor
- **WHEN** `server-sync-data` läuft und der Zielhost bereits eine höhere Schema-Version als der Quell-Snapshot hatte
- **THEN** wird zuerst der Snapshot geschrieben und danach `migrate up` ausgeführt, sodass das Ziel am Ende auf der Schema-Version des Quell-Codes (nicht des Ziel-Vorzustands) läuft

#### Scenario: Ziel-Service läuft am Ende
- **WHEN** `server-sync-data` erfolgreich durchläuft
- **THEN** ist `systemctl is-active teamwerk` auf dem Ziel `active`, und `/api/healthz` liefert `"status":"ok"`

---

### Requirement: `server-cutover` schaltet die Quelle auf 301-Redirect
`make server-cutover NEW_REMOTE=<alias>` MUST/MUSS: (1) einen `[y/N]`-Bestätigungsdialog anzeigen (deutliche Warnung „Alt-Instanz wird auf Redirect umgeschaltet"), (2) intern `server-sync-data` ausführen (frischer Snapshot), (3) auf dem Quell-Host den `teamwerk`-Service stoppen und disablen, (4) die bestehende Nginx-Config unter `/etc/nginx/sites-available/teamwerk.<timestamp>.bak` sichern, (5) die neue Redirect-Konfig aus `deploy/nginx-redirect.conf` mit ersetztem `NEW_DOMAIN`-Platzhalter deployen, (6) `nginx -t` prüfen und `systemctl reload nginx` ausführen, (7) am Ende einen Hinweis-Text mit Nachpflege-Punkten ausgeben (Better-Stack-Monitor umhängen, User informieren, PWA-Neuinstallation kommunizieren).

#### Scenario: Erfolgreicher Cutover
- **WHEN** `server-cutover` mit `y` bestätigt wird und alle Schritte gelingen
- **THEN** liefert `curl -sI https://intern.team-stuttgart.org/beliebiger/pfad` einen `301`-Response mit `Location: https://teamwerk.team-stuttgart.org/beliebiger/pfad` und `Cache-Control: no-store`

#### Scenario: Redirect auch für API-Pfade
- **WHEN** `server-cutover` erfolgreich lief und ein Alt-PWA-Client `POST https://intern.team-stuttgart.org/api/anything` schickt
- **THEN** antwortet der Alt-Host mit HTTP 301 und `Location: https://teamwerk.team-stuttgart.org/api/anything`, damit die Axios-Instanz die neue Origin sieht (statt Netzwerkfehler)

#### Scenario: Nginx-Backup wird geschrieben
- **WHEN** `server-cutover` die Nginx-Config ersetzt
- **THEN** existiert unter `/etc/nginx/sites-available/teamwerk.<ISO-timestamp>.bak` eine byte-identische Kopie der vorherigen Konfiguration

#### Scenario: Rollback nach Cutover ist per Runbook möglich
- **WHEN** nach dem Cutover ein Problem auftritt und der Betreuer dem Rollback-Abschnitt des Runbooks folgt
- **THEN** kann er die gesicherte Nginx-Config einspielen, `systemctl enable teamwerk && systemctl start teamwerk` ausführen und `nginx -t && systemctl reload nginx` starten, sodass der Alt-Host wieder als produktive Instanz reagiert

#### Scenario: Betreuer bricht Bestätigungsdialog ab
- **WHEN** `server-cutover` die Bestätigung anzeigt und der Betreuer nichts oder `n` eingibt
- **THEN** beendet sich das Target mit Exit-Code 1, ohne den Quell-Service zu stoppen, ohne die Nginx-Config zu ändern und ohne einen weiteren `server-sync-data`-Lauf zu starten

---

### Requirement: Zero-Knowledge-Verschlüsselung überlebt den Umzug ohne Umschlüsseln
Der Migrations-Vorgang MUST/MUSS die verschlüsselten Bankdaten-Felder byte-genau vom Quell- zum Zielhost übertragen und MUST NOT/DARF keinen Umschlüsselungs-Schritt einführen. Insbesondere MUST NOT/DARF weder ein Klartext-Schlüssel noch die Tresor-Passphrase im Skript, im Runbook oder in temporären Dateien vorkommen.

#### Scenario: Verschlüsselte Blobs überleben Snapshot
- **WHEN** `server-bootstrap` oder `server-sync-data` einen DB-Snapshot überträgt
- **THEN** sind die Blobs in `clubs.group_public_key`, `clubs.group_private_key_enc`, `clubs.sepa_ciphertext`, `clubs.sepa_dek_enc`, `member_sensitive.ciphertext`, `member_sensitive.dek_enc_vorstand`, `members.sepa_mandat_dek_enc` und `member_change_drafts.new_value` (bei `field_name='bankdaten'`) auf dem Zielhost byte-identisch zur Quelle

#### Scenario: Vorstand kann Bankdaten nach Cutover entschlüsseln
- **WHEN** nach dem Cutover ein Vorstands-User sich auf `https://teamwerk.team-stuttgart.org` einloggt und den Tresor mit der bekannten Passphrase öffnet
- **THEN** entschlüsseln die geladenen Ciphertexts im Browser wie zuvor, ohne dass der Server einen Umschlüsselungs-Schritt durchgeführt hat

---

### Requirement: JWT- und Push-Sender-Identität bleiben stabil
`server-bootstrap` MUST/MUSS `JWT_SECRET`, `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL` und `VIDEO_STREAM_SECRET` aus der Quell-Env unverändert übernehmen, damit bestehende Refresh-Tokens auf dem Zielhost gültig bleiben und die Push-Sender-Identität stabil ist.

#### Scenario: Refresh-Token bleibt gültig
- **WHEN** ein User mit gültigem Refresh-Token nach dem Cutover die neue Domain aufruft
- **THEN** kann der Access-Token mit dem im Cookie liegenden Refresh-Token erneuert werden (JWT_SECRET signiert wie zuvor), ohne dass der User neu einloggen muss

#### Scenario: Video-Stream-Token akzeptiert werden
- **WHEN** ein Stream-Token, das kurz vor dem Cutover ausgestellt wurde, nach dem Cutover an den Zielhost gesendet wird
- **THEN** akzeptiert der Zielhost das Token, weil `VIDEO_STREAM_SECRET` identisch ist (Token-Lebensdauer beachtet)

---

### Requirement: Better-Stack-Konfiguration wandert mit
`server-bootstrap` MUST/MUSS die Konfigurationsdateien `/etc/teamwerk/heartbeat-url`, `/etc/teamwerk/betterstack-logs-token`, `/etc/teamwerk/betterstack-metrics-token`, `/etc/teamwerk/betterstack-metrics-endpoint` vom Quell- zum Zielhost übertragen (Owner/Modus wie im Setup-Skript definiert). `server-cutover` MUST NOT/DARF diese Dateien auf der Quelle nicht verändern.

#### Scenario: Vector auf Ziel schickt Logs an Better Stack
- **WHEN** `server-bootstrap` durchläuft und danach `systemctl restart vector` auf dem Ziel gelaufen ist
- **THEN** erscheinen Journald-Log-Einträge des Zielhosts innerhalb von ~1 min in derselben Better-Stack-Log-Source, die auch der Quellhost verwendet

#### Scenario: Heartbeat-URL zeigt weiter auf gleiches Monitor-Element
- **WHEN** `server-bootstrap` durchläuft und der Cron-Wrapper `teamwerk-scheduler.sh` das erste Mal auf dem Ziel läuft
- **THEN** meldet er sich beim Better-Stack-Heartbeat-Monitor, den auch der Quellhost benutzt (identische Datei `/etc/teamwerk/heartbeat-url`)

---

### Requirement: Runbook dokumentiert manuelle Schritte und Rollback
`deploy/server-migration-runbook.md` MUST/MUSS existieren und mindestens folgende Punkte behandeln: (1) Vorbereitung (`.env`-Erweiterung, SSH-Alias, Ziel-VPS provisioniert), (2) `server-bootstrap`, (3) Test-Phase mit `/etc/hosts`-Ansteuerung, (4) DNS-A-Record-Wechsel und Certbot auf Ziel, (5) `server-cutover`, (6) Nachpflege (Better-Stack, User-Kommunikation, PWA-Neuinstallation), (7) Rollback-Anleitung („Wenn was schiefgeht"-Sektion).

#### Scenario: Runbook enthält Rollback
- **WHEN** ein Betreuer nach fehlgeschlagenem Cutover den Rollback-Abschnitt liest
- **THEN** findet er nummerierte Schritte, die die vom Cutover-Target gesicherte Nginx-Config wiederherstellen, den teamwerk-Service auf der Quelle wieder starten und (falls DNS schon umgestellt war) den DNS-A-Record zurücksetzen

#### Scenario: Runbook nennt manuelle externe Schritte explizit
- **WHEN** ein Betreuer das Runbook liest
- **THEN** findet er einen expliziten Abschnitt „Was nicht automatisiert ist" mit DNS-Wechsel, Certbot-Erstlauf, Better-Stack-Monitor-Umhängen und User-Kommunikation

---

### Requirement: `.env.example` dokumentiert die neuen Migrations-Variablen
`.env.example` MUST/MUSS `REMOTE_NEW`, `REMOTE_NEW_DIR` und `BASE_URL_NEW` als auskommentierte Zeilen enthalten, mit kurzem Hinweis, dass diese nur während eines Server-Umzugs gesetzt werden.

#### Scenario: Frisches Repo-Clone hat den Hinweis
- **WHEN** ein Entwickler `.env` aus `.env.example` erstellt (via `make env`)
- **THEN** enthält `.env` auskommentierte `REMOTE_NEW=`-, `REMOTE_NEW_DIR=`- und `BASE_URL_NEW=`-Zeilen mit erklärendem Kommentar

---

### Requirement: Neue Domain kommt aus `BASE_URL_NEW` in `.env`
Alle drei Migrations-Targets MUST/MÜSSEN die neue Zieldomain aus `BASE_URL_NEW` in `.env` beziehen (Format: vollständige URL inkl. Schema, z. B. `https://teamwerk.team-stuttgart.org`). Es gibt bewusst KEINEN CLI-Fallback für die Domain. Fehlt `BASE_URL_NEW` oder ist leer, MUST/MUSS das Target vor jeder Aktion mit einer klaren Fehlermeldung abbrechen.

#### Scenario: BASE_URL_NEW fehlt in .env
- **WHEN** ein Migrations-Target läuft und `.env` keinen `BASE_URL_NEW=`-Eintrag hat (oder Wert leer)
- **THEN** bricht das Target vor jeder Netzwerk-Aktion mit einer Fehlermeldung ab, die auf `BASE_URL_NEW=https://…` in `.env` hinweist

#### Scenario: Bootstrap nutzt BASE_URL_NEW für Env-Rewrite
- **WHEN** `.env` `BASE_URL_NEW=https://teamwerk.team-stuttgart.org` enthält und `server-bootstrap` läuft
- **THEN** enthält `/etc/teamwerk/env` auf dem Zielhost nach Abschluss `BASE_URL=https://teamwerk.team-stuttgart.org`

#### Scenario: Cutover nutzt BASE_URL_NEW im Redirect
- **WHEN** `.env` `BASE_URL_NEW=https://teamwerk.team-stuttgart.org` enthält und `server-cutover` läuft
- **THEN** enthält die neue Alt-Host-Nginx-Config `return 301 https://teamwerk.team-stuttgart.org$request_uri;` (bzw. äquivalente Weiche für alle Pfade)

---

### Requirement: Alle Migrations-Targets brechen bei fehlendem Quell-`REMOTE` sauber ab
Wenn `REMOTE` in `.env` nicht gesetzt oder leer ist, MUST/MÜSSEN alle drei Migrations-Targets vor jeder Aktion mit einer klaren Fehlermeldung abbrechen, weil ohne Quelle kein Umzug möglich ist.

#### Scenario: Quelle fehlt
- **WHEN** `.env` kein `REMOTE=`-Eintrag hat und `make server-bootstrap NEW_REMOTE=vServerNeu` läuft
- **THEN** bricht das Target mit einer Fehlermeldung ab, die explizit auf das fehlende `REMOTE=` in `.env` hinweist, und macht keine Änderungen am Zielhost
