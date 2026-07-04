## Context

TeamWERK läuft aktuell auf einem privaten IONOS-VPS (`intern.team-stuttgart.org`, 217.160.118.39). Team Stuttgart stellt einen eigenen Server bereit (`teamwerk.team-stuttgart.org`, 31.70.110.19). Der Umzug soll begleitet stattfinden: initialer Aufbau + Test-Phase → beliebig oft wiederholbarer Daten-Sync → finaler Cutover.

Aktueller Betriebsstand:

- `deploy/setup-vps.sh` richtet einen frischen VPS idempotent ein (Nginx, systemd, Certbot-Vorbereitung, Vector, Cron).
- `make deploy` baut Binary + Frontend, kopiert es auf den in `.env` gesetzten `REMOTE`, führt `migrate up` aus, startet Service neu.
- `make backup` / `make backup-files` ziehen einen konsistenten `sqlite3 .backup` und die Storage-Ordner (`uploads`, `files`, `videos`, `beitragslauf-protokolle`) lokal.
- `deploy/vps-setup-runbook.md` beschreibt manuelle Schritte (DNS, Certbot, Better-Stack).

Zero-Knowledge-Modell (Modell B): die Server-DB enthält `clubs.group_public_key` (nicht geheim) und `clubs.group_private_key_enc` (PBKDF2-verschlüsselt mit Tresor-Passphrase). Der Server besitzt **keinen** Entschlüsselungsschlüssel — die Passphrase existiert nur in den Browsern der Vorstand/Kassierer-Nutzer. Für den Umzug bedeutet das: DB muss bit-genau übertragen werden, es gibt aber nichts umzuschlüsseln.

## Goals / Non-Goals

**Goals:**

- Umzug von Quell- auf Ziel-VPS in drei skript-getriebenen Schritten (`bootstrap` → beliebig oft `sync-data` → `cutover`), jedes Target idempotent und wiederholbar.
- Test-Phase mit produktivem Betrieb auf Quell-Host, während der Ziel-Host mit einer Snapshot-DB läuft und nur vom Betreuer benutzt wird.
- Wiederverwendbar für künftige VPS-Wechsel (Provider-Wechsel, Disaster-Recovery-Übung, Staging-Auffrischung).
- Kein Verlust von Bankdaten-Ciphertext oder Push-Endpoints beim Kopieren.
- Klare Ausfallzeit: Bootstrap und Sync sind ohne Downtime; der Cutover schaltet den Quell-Service auf 301-Redirect um — kurz sichtbar für Nutzer, die den Redirect verfolgen.

**Non-Goals:**

- **Kein Live-Replika-Sync** zwischen Quell- und Ziel-DB. SQLite unterstützt keinen brauchbaren logischen Replikationsmechanismus, und für den Vereinsbetrieb (nachts idle) ist eine kurze Snapshot-Kohärenz vollkommen ausreichend.
- **Kein Provider-DNS-API**. DNS-A-Record-Wechsel bleibt manueller Schritt im Runbook.
- **Kein automatisches Umhängen der Better-Stack-Monitore**. Es gibt eine API, aber der Aufwand (Token, Bindings) übersteigt den Nutzen bei einem Handvoll Monitoren.
- **Keine automatische User-Benachrichtigung**. Ob per Push-Broadcast, E-Mail oder mündlicher Vorstandsansage kommuniziert wird, ist Entscheidung des Betreuers — das Runbook nennt die Optionen.
- **Kein Rollback-Target**. Rollback = alte Nginx-Config auf Quell-Host wiederherstellen und teamwerk-Service dort starten. Das Runbook beschreibt es, es wird nicht als Makefile-Target automatisiert (selten gebraucht, riskant zu skripten, weil sich Zustand seit Cutover geändert hat).

## Decisions

### D1 — Drei separate Targets statt „einem Kommando"

`server-bootstrap`, `server-sync-data`, `server-cutover` sind eigenständig aufrufbar. Grund: die drei Phasen haben verschiedene Vorbedingungen (DNS gesetzt? Certbot fertig? Testphase abgeschlossen?) und der Betreuer entscheidet, wann er weitergeht. Ein monolithisches `server-migrate`-Target würde entweder in der Mitte auf User-Input warten (interaktiv, schwer zu skripten) oder Annahmen treffen, die nicht immer stimmen (z. B. „DNS ist umgestellt").

**Alternativen verworfen:**
- **Ein Target mit Phasen-Argument** (`make server-migrate PHASE=bootstrap`): funktional gleichwertig, aber `PHASE=cutover` ist bei versehentlichem Aufruf zerstörerisch. Drei Targets machen die Absicht am Aufrufer-Ende sichtbar.
- **Shell-Skript unter `deploy/` statt Makefile-Targets**: Makefile-Ökosystem ist im Projekt etabliert (`make deploy`, `make backup`, `make migrate-remote-up`). Kein Grund, für Deploy-Adjazenz das Muster zu brechen.

### D2 — Snapshot via `sqlite3 .backup` + rsync, kein `.dump`/`.read`

`sqlite3 .backup` erzeugt einen konsistenten binären Snapshot einer laufenden WAL-DB ohne Service-Stopp. Der Snapshot wird via `scp`/`rsync` übertragen, nicht ein `.dump | .read`.

**Grund:** `.dump/.read` schreibt Rowids neu und kann bei Foreign-Key- oder AUTOINCREMENT-Bezügen zu subtilen Verschiebungen führen. Die Bankdaten-Envelopes (`member_sensitive.member_id`, `dek_enc_vorstand`) hängen an den Rowids — jedes Umschreiben ist Risiko. `.backup` ist byte-genau.

**Trade-off:** `.backup` liefert eine `.db` inklusive vollem Schema, nicht nur Deltas. Bei einer 20 MB-DB egal, bei künftig größerer DB ggf. langsamer als delta-basierte Ansätze — aber es gibt für SQLite keine wartungsarme Delta-Alternative.

### D3 — `.backup`-Snapshot vor `migrate up` auf Ziel

Sequenz auf Ziel-Host bei `server-sync-data`:
1. Ziel-`teamwerk`-Service stoppen (verhindert Schreiben während Overwrite).
2. Snapshot der Quell-DB nach Ziel-`/var/lib/teamwerk/teamwerk.db` schreiben.
3. `migrate up` **danach** ausführen.
4. Ziel-Service wieder starten.

**Grund:** Wenn zwischen Bootstrap und Sync auf dem Ziel-Host schon einmal eine höhere Migrations-Nummer aktiv war (weil in der Zwischenzeit deployt wurde), und der Snapshot der Quelle noch auf niedrigerer Version steht, würde ohne `migrate up` das Schema unstimmig sein. `migrate up` ist idempotent — läuft der Snapshot schon auf der aktuellen Nummer, no-op.

**Alternative verworfen:** `migrate up` weglassen und darauf verlassen, dass Quelle und Ziel dieselbe Version haben. Zu fragil — sobald jemand auf dem Ziel-Host einen Test-Deploy macht, kippt es.

### D4 — Env-Klonen: BASE_URL rewriten, Secrets unverändert

`server-bootstrap` liest `/etc/teamwerk/env` vom Quell-Host via SSH, filtert `BASE_URL` heraus und ersetzt sie durch `https://<neue-Domain>`, alle anderen Zeilen bleiben unangetastet. Ergebnis wird via SSH-Pipe direkt in `/etc/teamwerk/env` auf dem Ziel-Host geschrieben (nie auf Entwickler-Disk zwischenspeichern).

**Konkret mitgenommen:**
- `JWT_SECRET` (sonst würden alle Refresh-Tokens ungültig)
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL` (Push-Sender-Identität bleibt gleich)
- `VIDEO_STREAM_SECRET` (Stream-Token-Signaturen)
- `SMTP_*`
- `METRICS_TOKEN` (Better-Stack Vector auf Ziel scrapet mit demselben Token)

**Bewusst nicht ausgetauscht:** Auch die Better-Stack-Tokens (`heartbeat-url`, `betterstack-logs-token`, `betterstack-metrics-token`, `betterstack-metrics-endpoint`) unter `/etc/teamwerk/*` werden 1:1 kopiert, damit Ziel-Host sofort in denselben Monitor-/Log-Stream schreibt. Sauberer wären zwar getrennte Sources, aber der Aufwand rechtfertigt sich nicht — nach Cutover ist der Alt-Host offline, doppelte Streams sind kein Problem.

**Alternative verworfen:** Env-Datei komplett vom Entwickler frisch aufbauen. Zu fehleranfällig (JWT_SECRET vergessen = alle User ausgeloggt; VAPID-Keys frisch generiert = alle bestehenden `push_subscriptions` sofort ungültig).

### D5 — Nginx-Redirect als vollständiger Wildcard-301 inkl. `/api/*`

`deploy/nginx-redirect.conf` leitet **alle** Pfade per `return 301 https://teamwerk.team-stuttgart.org$request_uri;` weiter, inklusive `/api/*`. `Cache-Control: no-store` in derselben Server-Block-Antwort.

**Grund:** PWA-Instanzen alter Nutzer würden ohne API-Redirect Netzwerkfehler produzieren und den ServiceWorker-Cache in einem kaputten Zustand hinterlassen. Mit 301 auch für API-Routen bekommt die Axios-Instanz einen Response, der Browser wechselt die Origin, User landet auf der neuen Domain und muss sich dort erneut einloggen (JWT gleich → funktioniert dann sofort).

**Trade-off:** Better-Stack-HTTP-Monitor auf `intern.team-stuttgart.org/api/healthz` würde bei aktivem Redirect grün bleiben („folgt 301"), was ein falsches Sicherheitsgefühl gibt. Das Runbook weist explizit an, den Monitor vor dem Cutover auf die neue Domain umzuhängen.

### D6 — `.env`-Erweiterung um `REMOTE_NEW` / `REMOTE_NEW_DIR` / `BASE_URL_NEW`

`Makefile` liest `NEW_REMOTE` / `NEW_REMOTE_DIR` aus dem Aufruf, fällt bei fehlendem Argument auf `REMOTE_NEW` / `REMOTE_NEW_DIR` aus `.env` zurück (analog zum bestehenden `REMOTE`/`REMOTE_DIR`-Muster). Die neue Domain kommt **ausschließlich** aus `BASE_URL_NEW` in `.env` (kein CLI-Fallback).

**Grund:** Wiederkehrend heißt: der Betreuer will während der Testphase mehrfach `make server-sync-data` tippen, ohne jedes Mal Argumente zu wiederholen. `.env`-Backing macht das komfortabel und ist konsistent mit dem existierenden Muster. Für die Domain gibt es bewusst keinen CLI-Fallback, damit `BASE_URL_NEW` nicht zwischen Läufen abweichen kann — die Env-Datei ist die Wahrheit. Fehlt `BASE_URL_NEW` in `.env`, bricht jedes Migrations-Target vor der ersten Aktion ab.

### D7 — Explizite Bestätigungsdialoge in `sync-data` und `cutover`

`server-sync-data` und `server-cutover` prompten `printf "…überschreibt Testdaten…, ok? [y/N] "; read ans; case "$$ans" in y|Y) …;; *) exit 1;; esac`, analog zu `restore-local` im bestehenden Makefile.

**Grund:** Beide Targets sind zerstörerisch (überschreiben die Ziel-DB bzw. den Alt-Host-Nginx). Kein `--force`-Flag — wer diese Targets versehentlich tippt, sollte innehalten.

## Risks / Trade-offs

- **Risiko:** Env-Klonen kopiert versehentlich einen alten `BASE_URL`, weil das `sed`-Muster nicht greift → Ziel-Host läuft mit falscher `BASE_URL` weiter, E-Mail-Links und Push-URLs zeigen auf alten Host. **Mitigation:** Nach `server-bootstrap` explizit `curl -s <neu>/api/healthz` und `grep BASE_URL /etc/teamwerk/env` als letzter Skript-Schritt; Ausgabe zeigt, was gesetzt wurde.
- **Risiko:** DB-Snapshot dauert bei künftig größerer DB (>500 MB) länger, während dieser Zeit weiter geschrieben wird → Snapshot enthält sehr frische Änderungen aus Quell-DB, aber nicht die allerneuesten. **Mitigation:** Für den Cutover ist das akzeptabel (kurz vorher Vorstand bitten, nichts zu tippen). `.backup` selbst ist WAL-safe — kein korrupter Zustand, nur „ein paar Sekunden zurück".
- **Risiko:** Ziel-Host hat nicht genug Disk für `/storage/videos/` (Standard-Setup ist 10 GB). **Mitigation:** Vor `server-bootstrap` warnt das Target, wenn `ssh <ziel> "df -B1 /storage | tail -1"` weniger frei zeigt als `ssh <quell> "du -sb /storage/videos"`. Betreuer muss dann Disk erweitern (steht schon in `vps-setup-runbook.md` unter „Storage-Erweiterung").
- **Risiko:** Nutzer sind während der Testphase auf `teamwerk.team-stuttgart.org` gelandet (z. B. weil DNS schon zeigt und jemand die URL in WhatsApp geteilt hat), tippen dort, verlieren beim `server-sync-data` diese Änderungen. **Mitigation:** Runbook rät, DNS erst *nach* Cutover umzustellen, sondern für die Testphase das Ziel nur über IP + Host-Header (`curl -H "Host: teamwerk.team-stuttgart.org" https://31.70.110.19/...`) oder eine lokale `/etc/hosts`-Zeile beim Betreuer anzusteuern.
- **Trade-off:** Push-Subscriptions in der kopierten DB sind auf die alte Origin registriert. Nach dem Cutover werden Sende-Versuche mit HTTP 410 von den Push-Endpoints abgelehnt, das eingebaute Cleanup in `internal/notifications` räumt sie ab. Neue Subscriptions kommen erst, wenn User die PWA auf `teamwerk.team-stuttgart.org` neu installieren. → Nach dem Cutover ist Push bei allen Alt-Usern kurz stumm — akzeptabel, im Runbook dokumentiert.
- **Trade-off:** Die alte Nginx-Redirect-Konfiguration bleibt auf dem Alt-Host bestehen, solange dieser existiert. Aufräumen (Host abbauen) ist Provider-Handarbeit, nicht Teil dieser Change.

## Migration Plan

Nicht anwendbar im klassischen Sinn — diese Change *ist* das Migrations-Werkzeug. Konkreter Umzug von `intern.` auf `teamwerk.` wird im neuen Runbook (`deploy/server-migration-runbook.md`) beschrieben und läuft:

1. **Vorbereitung:** `REMOTE_NEW=vServerNeu`, `REMOTE_NEW_DIR=/usr/local/bin` in `.env` ergänzen. SSH-Alias `vServerNeu` in `~/.ssh/config` einrichten (Root auf 31.70.110.19). Ziel-VPS provisioniert, ansonsten leer.
2. **Bootstrap:** `make server-bootstrap NEW_REMOTE=vServerNeu` einmalig. Ziel-Host läuft dann mit Snapshot der Produktion.
3. **Testphase (Tage bis Wochen):** Betreuer testet über `/etc/hosts`-Zeile lokal oder direkt via IP. Bei Bedarf mehrfach `make server-sync-data NEW_REMOTE=vServerNeu`, um frischen Produktions-Stand zu holen.
4. **DNS + Certbot:** DNS-A-Record `teamwerk.team-stuttgart.org` → 31.70.110.19 setzen (Provider-Panel). Warten auf DNS-Propagation (~15 min bis 1h). `certbot --nginx -d teamwerk.team-stuttgart.org` auf Ziel.
5. **Cutover:** `make server-cutover NEW_REMOTE=vServerNeu`. Prompt bestätigen. Frischer DB-Sync, Alt-Service stoppt, Alt-Nginx wechselt auf 301.
6. **Nachpflege:** Better-Stack-Monitor-URLs auf neue Domain umhängen, User-Kommunikation (Push-Broadcast oder Vorstandsansage) mit Erinnerung an PWA-Neuinstallation.

**Rollback (falls Cutover schiefging):**
- SSH auf Quell-Host, alte Nginx-Config zurücksichern (das `cutover`-Target schreibt vor dem Ersetzen ein Backup nach `/etc/nginx/sites-available/teamwerk.<timestamp>.bak`).
- `sudo systemctl enable teamwerk && sudo systemctl start teamwerk` auf Quell-Host.
- `nginx -t && systemctl reload nginx` auf Quell-Host.
- DNS-A-Record zurück auf alte IP (falls schon umgehängt).
- Runbook enthält diese Schritte als „Wenn was schiefgeht"-Sektion.

## Open Questions

- **User-Kommunikation vor oder nach dem Cutover?** Vor dem Cutover fühlt sich sauberer an, aber viele werden es überlesen und trotzdem auf die alte URL klicken → der Redirect erledigt es dann. Runbook empfiehlt Push-Broadcast *nach* erfolgreichem Cutover, wenn Redirect steht.
- **Was tun mit den bestehenden `push_subscriptions` beim Cutover?** Optionen: (a) belassen, HTTP-410-Cleanup übernimmt es → default. (b) proaktiv per SQL löschen, damit die nächste Push-Runde nicht in Fehlerlogs steht. Entscheidung: (a), weil das Cleanup vorhanden und der Log-Spam kurz ist. Kann in einer Folge-Change kommen, wenn's stört.
