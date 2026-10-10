BINARY     := teamwerk
BUILD_DIR  := bin
# Prefer the system Go at /usr/local/go if present (matches go.mod toolchain).
# Falls back to whatever 'go' is on PATH.
GO         := $(or $(wildcard /usr/local/go/bin/go),go)
REPO_ROOT  := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
REMOTE     := $(shell grep '^REMOTE=' .env 2>/dev/null | cut -d= -f2)
REMOTE_DIR := $(shell grep '^REMOTE_DIR=' .env 2>/dev/null | cut -d= -f2)
BASE_URL   := $(shell grep '^BASE_URL=' .env 2>/dev/null | cut -d= -f2-)
# Server-Umzug (aus .env; nur gesetzt während einer Migration)
REMOTE_NEW     := $(shell grep '^REMOTE_NEW=' .env 2>/dev/null | cut -d= -f2-)
REMOTE_NEW_DIR := $(shell grep '^REMOTE_NEW_DIR=' .env 2>/dev/null | cut -d= -f2-)
BASE_URL_NEW   := $(shell grep '^BASE_URL_NEW=' .env 2>/dev/null | cut -d= -f2-)
# CLI-Argument NEW_REMOTE hat Vorrang vor REMOTE_NEW aus .env
NEW_REMOTE_RESOLVED     := $(or $(NEW_REMOTE),$(REMOTE_NEW))
NEW_REMOTE_DIR_RESOLVED := $(or $(REMOTE_NEW_DIR),/usr/local/bin)
# Domains ohne Schema (für Host-Header, nginx server_name, Cert-Pfade)
SOURCE_DOMAIN := $(patsubst https://%,%,$(BASE_URL))
NEW_DOMAIN    := $(patsubst https://%,%,$(BASE_URL_NEW))
# ── Ziel-Umgebung: ENV=prod (Default) | ENV=beta ─────────────────────────────
# Alle Server-Targets (deploy, deploy-rollback, migrate-remote-up,
# create-admin-remote, push-test-remote, backup*, pull-*, setup-vps, dev-remote)
# wirken auf die gewählte Umgebung. Ohne ENV bleibt alles wie bisher Prod.
# Beta = zweite, vollständig getrennte Instanz auf demselben VPS (eigenes
# Binary, eigene Env, DB + Storage unter /var/lib/teamwerk-beta, Dienst
# teamwerk-beta auf Port 8081) — Details: docs/agent/10-deployment.md.
# Branch-Bindung: deploy ENV=prod nur von main oder einem vX.Y.Z-Tag,
# deploy ENV=beta nur vom Branch beta (jeweils ohne uncommittete Änderungen);
# Notausgang ALLOW_BRANCH=1.
ENV ?= prod
ifeq ($(ENV),prod)
REMOTE_DATA      := /var/lib/teamwerk
REMOTE_ENV_FILE  := /etc/teamwerk/env
REMOTE_SERVICE   := teamwerk
REMOTE_BIN       := $(REMOTE_DIR)/teamwerk
REMOTE_PORT      := 8080
VIDEO_STORAGE_DIR_REMOTE := /storage/videos
DEPLOY_BRANCH    := main
DEPLOYED_HASH    := .deployed-hash
BACKUP_ROOT      := $(REPO_ROOT)/backup
# Prod läuft historisch als SSH-Nutzer (root) und chownt danach; das trägt auch
# den allerersten Deploy auf einen frischen Server (server-bootstrap).
AS_SERVICE_USER  :=
else ifeq ($(ENV),beta)
REMOTE_DATA      := /var/lib/teamwerk-beta
REMOTE_ENV_FILE  := /etc/teamwerk-beta/env
REMOTE_SERVICE   := teamwerk-beta
REMOTE_BIN       := $(REMOTE_DIR)/teamwerk-beta
REMOTE_PORT      := 8081
VIDEO_STORAGE_DIR_REMOTE := /var/lib/teamwerk-beta/videos
DEPLOY_BRANCH    := beta
DEPLOYED_HASH    := .deployed-beta-hash
# Eigener Unterordner: restore-local sucht backup/20*/ und darf nie einen
# Beta-Stand für einen Prod-Stand halten.
BACKUP_ROOT      := $(REPO_ROOT)/backup/beta
# Auf der Beta entsteht nichts root-eigenes im Datenverzeichnis.
AS_SERVICE_USER  := sudo -u www-data
else
$(error ENV=$(ENV) unbekannt — erlaubt: prod, beta)
endif
DB_PATH    := $(REMOTE_DATA)/teamwerk.db
# Storage-Verzeichnisse auf dem VPS (unter REMOTE_DATA der gewählten Umgebung).
# UPLOAD_DIR enthält Profilfotos + SEPA-Mandat-PDFs (Tresor-Blobs) und gehört
# semantisch zum `backup`-Target neben der DB. Die anderen sind File-Bulk.
UPLOAD_DIR_REMOTE                 := $(REMOTE_DATA)/uploads
FILES_DIR_REMOTE                  := $(REMOTE_DATA)/files
MEDIA_DIR_REMOTE                  := $(REMOTE_DATA)/media
BEITRAGSLAUF_DIR_REMOTE           := $(REMOTE_DATA)/beitragslauf-protokolle
MATCH_REPORT_IMAGE_DIR_REMOTE     := $(REMOTE_DATA)/match-report-images
TRAINING_DIARY_DIR_REMOTE         := $(REMOTE_DATA)/training-diary
BWHV_REPORT_DIR_REMOTE            := $(REMOTE_DATA)/bwhv-reports
UPLOAD_DIR_LOCAL                  := $(REPO_ROOT)/storage/uploads
FILES_DIR_LOCAL                   := $(REPO_ROOT)/storage/files
MEDIA_DIR_LOCAL                   := $(REPO_ROOT)/storage/media
BEITRAGSLAUF_DIR_LOCAL            := $(REPO_ROOT)/storage/beitragslauf-protokolle
MATCH_REPORT_IMAGE_DIR_LOCAL      := $(REPO_ROOT)/storage/match-report-images
TRAINING_DIARY_DIR_LOCAL          := $(REPO_ROOT)/storage/training-diary
BWHV_REPORT_DIR_LOCAL             := $(REPO_ROOT)/storage/bwhv-reports
VIDEO_STORAGE_DIR_LOCAL           := $(REPO_ROOT)/storage/videos
EMAIL      ?= $(shell grep '^EMAIL=' .env 2>/dev/null | cut -d= -f2-)
PASSWORD   ?= $(shell grep '^PASSWORD=' .env 2>/dev/null | cut -d= -f2-)
NAME       ?= $(shell grep '^NAME=' .env 2>/dev/null | cut -d= -f2-)
TS         := $(shell date +%Y-%m-%dT%H-%M-%S)
BACKUP_DIR := $(BACKUP_ROOT)/$(TS)

.PHONY: help init hooks dev dev-remote build deploy deploy-rollback deploy-new setup-beta seed-beta mirror-beta deploy-beta _check-branch _check-env _check-prod-only _smoke setup-vps migrate-up migrate-down migrate-remote-up create-admin create-admin-remote push-test-remote env clean backup backup-files backup-videos restore-local restore-local-files restore-local-videos pull-db pull-files pull-videos test test-race test-e2e folien schulung lint coverage metrics metrics-gate measure server-bootstrap server-sync-data server-cutover _check-remote _check-new-remote _check-base-url-new

.DEFAULT_GOAL := help

help: ## Diesen Hilfetext anzeigen
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-22s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

env: ## .env aus .env.example erstellen
	@if [ -f .env ]; then echo ".env existiert bereits – nichts geändert."; else \
		SECRET=$$(openssl rand -hex 32); \
		sed "s/change-me-to-a-random-secret/$$SECRET/" .env.example > .env; \
		echo ".env erstellt (JWT_SECRET automatisch gesetzt)."; \
	fi

init: hooks ## Abhängigkeiten installieren (go mod tidy, pnpm install) + Git-Hooks aktivieren
	$(GO) mod tidy
	cd web && pnpm install
	# Playwright-Chromium für E2E (make test-e2e). Idempotent: skippt still, wenn schon da.
	cd web && pnpm exec playwright install chromium

hooks: ## Git-Hooks aktivieren (core.hooksPath → .githooks: pre-commit gofmt, pre-push Gate)
	git config core.hooksPath .githooks
	@echo "Git-Hooks aktiv (.githooks). pre-commit: gofmt · pre-push: vet+test+lint+build."

dev: ## Backend (mit air Auto-Reload) + Vite Dev-Server lokal starten
	@echo "Starting backend on :8080 (with auto-reload) and frontend dev server..."
	@mkdir -p web/dist
	@AIR="$$($(GO) env GOPATH)/bin/air"; \
	if [ -x "$$AIR" ]; then \
		"$$AIR" -build.cmd "$(GO) build -o ./tmp/main ./cmd/teamwerk" & \
	else \
		echo "air not found, using go run (no auto-reload)"; \
		$(GO) run ./cmd/teamwerk & \
	fi
	@sleep 1
	@cd web && pnpm dev

dev-remote: ## SSH-Tunnel zum VPS + Vite Dev-Server (kein lokales Backend; ENV=beta tunnelt zur Beta)
	@echo "Opening SSH tunnel to $(REMOTE) ($(REMOTE_SERVICE), Port $(REMOTE_PORT)) and starting frontend dev server..."
	@ssh -N -L 8080:localhost:$(REMOTE_PORT) $(REMOTE) &
	@cd web && pnpm dev

build: ## Frontend + Backend für Linux/amd64 bauen
	@git log --format="%ad|%s" --date=format:"%d.%m.%Y" --no-merges \
	  | grep -E "\|(feat|fix)(\([^)]*\))?:" \
	  | python3 scripts/gen-changelog.py > web/public/CHANGELOG.md
	cd web && pnpm build
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X 'main.buildHash=$(shell git rev-parse --short HEAD)'" -o $(BUILD_DIR)/$(BINARY) ./cmd/teamwerk

setup-vps: ## VPS einmalig einrichten (Nginx, Certbot, systemd; ENV=beta richtet die Beta-Instanz ein)
ifeq ($(ENV),beta)
	@$(MAKE) --no-print-directory setup-beta
else
	rsync -az deploy/ $(REMOTE):/tmp/teamwerk-deploy/
	ssh $(REMOTE) "cd /tmp/teamwerk-deploy && sudo bash setup-vps.sh"
endif

# Branch-Bindung (siehe ENV-Block oben). Prüft VOR dem Build: Prod nur von
# main oder einem vX.Y.Z-Tag (deploy.yml checkt das Tag detached aus), Beta nur
# vom Branch beta; uncommittete Änderungen an versionierten Dateien brechen ab,
# sonst wäre „von main deployt" keine Aussage über den Stand.
_check-branch:
	@if [ "$(ALLOW_BRANCH)" = "1" ]; then echo "WARNUNG: Branch-Bindung für ENV=$(ENV) übersprungen (ALLOW_BRANCH=1)"; exit 0; fi; \
	branch=$$(git rev-parse --abbrev-ref HEAD); \
	tag=$$(git describe --exact-match --tags --match 'v[0-9]*.[0-9]*.[0-9]*' HEAD 2>/dev/null || true); \
	if [ -n "$$(git status --porcelain --untracked-files=no)" ]; then \
		echo "Abbruch: uncommittete Änderungen — ENV=$(ENV) deployt nur committete Stände (Notausgang ALLOW_BRANCH=1)." >&2; exit 1; fi; \
	case "$(ENV)" in \
		prod) if [ "$$branch" != main ] && [ -z "$$tag" ]; then \
			echo "Abbruch: Prod wird nur von main oder einem vX.Y.Z-Tag deployt (aktuell: $$branch)." >&2; exit 1; fi ;; \
		beta) if [ "$$branch" != beta ]; then \
			echo "Abbruch: Beta wird nur vom Branch beta deployt (aktuell: $$branch)." >&2; exit 1; fi ;; \
	esac; \
	echo "Branch-Bindung OK: ENV=$(ENV), $${tag:-$$branch} @ $$(git rev-parse --short HEAD)"

# Schutz der Prod-Instanz vor Beta-Aufrufen: jeder Pfad und Dienstname einer
# ENV=beta-Operation muss „teamwerk-beta" enthalten. Durch den ENV-Block gilt
# das per Konstruktion; der Check fängt künftige Tippfehler dort ab.
_check-env:
ifeq ($(ENV),beta)
	@for v in "$(REMOTE_DATA)" "$(REMOTE_ENV_FILE)" "$(REMOTE_SERVICE)" "$(REMOTE_BIN)" "$(DB_PATH)" "$(VIDEO_STORAGE_DIR_REMOTE)"; do \
		case "$$v" in *teamwerk-beta*) ;; *) echo "Abbruch: ENV=beta zeigt auf '$$v' — kein Beta-Pfad" >&2; exit 1 ;; esac; \
	done
	@ssh $(REMOTE) "test -f $(REMOTE_ENV_FILE)" || { echo "Beta nicht eingerichtet — erst 'make setup-vps ENV=beta'" >&2; exit 1; }
else
	@true
endif

# Targets, die es nur für Prod gibt (Server-Umzug).
_check-prod-only:
	@[ "$(ENV)" = prod ] || { echo "Abbruch: dieses Target gibt es nur für ENV=prod" >&2; exit 1; }

deploy: _check-env _check-branch build ## Build + Deploy auf VPS (ENV=prod|beta; Binary, Migrations, Service-Neustart, Smoke-Test)
	rsync -az $(BUILD_DIR)/$(BINARY) $(REMOTE):/tmp/$(REMOTE_SERVICE).new
ifeq ($(ENV),prod)
	rsync -az deploy/teamwerk.service $(REMOTE):/tmp/teamwerk.service
	rsync -az deploy/backup-cron.sh $(REMOTE):/tmp/teamwerk-backup.sh
	ssh $(REMOTE) "[ -f /etc/teamwerk/env ]" 2>/dev/null || \
		grep -E '^(PORT|DB_PATH|JWT_SECRET|BASE_URL|SMTP_HOST|SMTP_PORT|SMTP_USER|SMTP_PASS|SMTP_FROM)=' .env | \
		sed 's|DB_PATH=.*|DB_PATH=/var/lib/teamwerk/teamwerk.db|; s|BASE_URL=.*|BASE_URL=https://teamwerk.team-stuttgart.org|' | \
		ssh $(REMOTE) "sudo mkdir -p /etc/teamwerk && sudo tee /etc/teamwerk/env > /dev/null && sudo chmod 600 /etc/teamwerk/env"
endif
	@# Storage-Verzeichnisse + zugehörige Env-Schlüssel idempotent sicherstellen,
	@# dazu reine Konfig-Schlüssel ohne Verzeichnis (HSTS_ENABLED, nur Prod). Nötig,
	@# weil die Env-Datei nur beim ERSTEN Deploy (Prod) bzw. von setup-beta (Beta)
	@# geschrieben wird: ein neuer Schlüssel fehlt auf Bestandsservern sonst
	@# dauerhaft. Schlüssel mit Endung _DIR bekommen zusätzlich mkdir+chown; ohne
	@# Env-Eintrag greift bei diesen der relative Default (./storage/... relativ
	@# zum WorkingDirectory) — dort darf www-data nicht schreiben, der Prozess
	@# kommt nicht hoch und Nginx antwortet mit 502. HSTS_ENABLED=true setzt
	@# voraus, dass bereits ein gültiges TLS-Zertifikat aktiv ist (siehe
	@# docs/agent/10-deployment.md); die Beta-Env trägt bewusst false.
	@for kv in "TRAINING_DIARY_DIR=$(TRAINING_DIARY_DIR_REMOTE)" "BEITRAGSLAUF_DIR=$(BEITRAGSLAUF_DIR_REMOTE)" "BWHV_REPORT_DIR=$(BWHV_REPORT_DIR_REMOTE)" $(if $(filter prod,$(ENV)),"HSTS_ENABLED=true"); do \
		key=$${kv%%=*}; val=$${kv#*=}; \
		mkcmd=""; \
		case "$$key" in \
			*_DIR) mkcmd="sudo mkdir -p $$val && sudo chown www-data:www-data $$val && " ;; \
		esac; \
		ssh $(REMOTE) "$${mkcmd}if ! sudo grep -q '^$$key=' $(REMOTE_ENV_FILE); then \
				echo '$$kv' | sudo tee -a $(REMOTE_ENV_FILE) > /dev/null; \
				echo '  $$key in $(REMOTE_ENV_FILE) ergänzt'; \
			fi" || exit 1; \
	done
	@# JWT_SECRET-Mindestlänge VOR dem Restart prüfen (analog config.Load): ein zu
	@# kurzes oder fehlendes Secret darf nicht erst beim Prozessstart auffallen,
	@# sonst bleibt der alte Prozess unten (kein Halb-Deploy), aber die Migration
	@# lief schon — deshalb Abbruch hier, bevor migrate/restart überhaupt laufen.
	@echo "Prüfe JWT_SECRET-Länge in $(REMOTE_ENV_FILE) auf $(REMOTE)..."
	@ssh $(REMOTE) 'secret=$$(sudo grep "^JWT_SECRET=" $(REMOTE_ENV_FILE) | cut -d= -f2-); \
		if [ -z "$$secret" ] || [ $${#secret} -lt 32 ]; then \
			echo "JWT_SECRET in $(REMOTE_ENV_FILE) fehlt oder ist kuerzer als 32 Byte - Deploy abgebrochen (kein Restart)." >&2; \
			exit 1; \
		fi' || exit 1
ifeq ($(ENV),prod)
	@# Serverseitigen Backup-Cron installieren/aktualisieren (idempotent) — das
	@# Skript selbst darf sich mit jedem Deploy ändern, der Cron-Eintrag wird
	@# nur einmal ergänzt. Analog zum Scheduler-Cron in deploy/setup-vps.sh.
	@# Nur Prod: die Beta ist Wegwerf-Zustand und hat bewusst kein Backup.
	ssh $(REMOTE) "sudo mkdir -p /var/backups/teamwerk && \
		sudo mv /tmp/teamwerk-backup.sh /usr/local/bin/teamwerk-backup.sh && \
		sudo chmod +x /usr/local/bin/teamwerk-backup.sh && \
		if ! sudo crontab -l 2>/dev/null | grep -qF '/usr/local/bin/teamwerk-backup.sh'; then \
			( sudo crontab -l 2>/dev/null; echo '30 3 * * * /usr/local/bin/teamwerk-backup.sh >> /var/log/teamwerk-backup.log 2>&1' ) | sudo crontab -; \
			echo '  Backup-Cron ergaenzt'; \
		fi"
	@# Systemd-Unit beim allerersten Deploy installieren (Beta: setup-beta).
	ssh $(REMOTE) "sudo mkdir -p $(dir $(DB_PATH)) && \
		if ! [ -f /etc/systemd/system/teamwerk.service ]; then \
			sudo mv /tmp/teamwerk.service /etc/systemd/system/teamwerk.service && \
			sudo systemctl daemon-reload && sudo systemctl enable teamwerk; \
		fi"
endif
	@# Vor dem Austausch das laufende Binary als .prev sichern (Rückweg für
	@# `make deploy-rollback`, falls der Smoke-Test unten fehlschlägt). Auf dem
	@# allerersten Deploy existiert noch kein Binary am Zielpfad — die
	@# `|| true` macht das harmlos statt den Deploy abzubrechen.
	ssh $(REMOTE) "sudo cp $(REMOTE_BIN) $(REMOTE_BIN).prev 2>/dev/null || true; \
		sudo mv /tmp/$(REMOTE_SERVICE).new $(REMOTE_BIN) && \
		$(AS_SERVICE_USER) $(REMOTE_BIN) migrate up --db $(DB_PATH) && \
		sudo chown www-data:www-data $(DB_PATH) $(DB_PATH)-shm $(DB_PATH)-wal 2>/dev/null; \
		sudo systemctl restart $(REMOTE_SERVICE)"
	@# Smoke-Test: bis zu 30s auf /api/healthz warten (health.Handler.Healthz,
	@# internal/health/health.go — 200 bei gesunder DB, sonst 503; `curl -f`
	@# behandelt beides korrekt als Erfolg/Fehlschlag). Schlägt das fehl, bricht
	@# der Deploy ab BEVOR der Deploy-Hash geschrieben wird (siehe unten) — der
	@# Hinweis zeigt auf `make deploy-rollback`.
	@$(MAKE) --no-print-directory _smoke ENV=$(ENV) SMOKE_FAIL="Deploy fehlgeschlagen: Prozess antwortet nicht - make deploy-rollback ENV=$(ENV)"
	@echo "Deployed successfully (ENV=$(ENV))."
	@git rev-parse --short HEAD > $(DEPLOYED_HASH)

# Smoke-Test der gewählten Umgebung (Port aus ihrer Env-Datei).
_smoke:
	@echo "Smoke-Test $(REMOTE_SERVICE): warte auf /api/healthz (bis 30s)..."
	@ssh $(REMOTE) 'PORT=$$(sudo grep -E "^PORT=" $(REMOTE_ENV_FILE) | cut -d= -f2-); PORT=$${PORT:-$(REMOTE_PORT)}; \
		ok=0; \
		for i in $$(seq 1 15); do \
			if curl -fsS "http://127.0.0.1:$$PORT/api/healthz" > /dev/null 2>&1; then ok=1; break; fi; \
			sleep 2; \
		done; \
		if [ "$$ok" != "1" ]; then \
			echo "systemctl status $(REMOTE_SERVICE):" >&2; \
			sudo systemctl status $(REMOTE_SERVICE) --no-pager | tail -20 >&2; \
			echo "$(SMOKE_FAIL)" >&2; \
			exit 1; \
		fi; \
		echo "Smoke-Test OK ($(REMOTE_SERVICE), Port $$PORT)."' || exit 1

deploy-rollback: _check-env ## Vorheriges Binary zurückspielen (ENV=prod|beta; Notfall, wenn der Smoke-Test in `make deploy` fehlschlägt)
	@echo "Prüfe $(REMOTE_BIN).prev auf $(REMOTE)..."
	@ssh $(REMOTE) "test -f $(REMOTE_BIN).prev" \
		|| { echo "Fehler: kein $(REMOTE_BIN).prev vorhanden - kein Rollback moeglich (setzt einen vorherigen 'make deploy ENV=$(ENV)' voraus)."; exit 1; }
	@# Kein Rückwärts-Migrieren: Migrationen sind additiv (siehe
	@# docs/agent/10-deployment.md „Smoke-Test und Rollback") — die aktuelle
	@# DB bleibt mit dem vorherigen Binary kompatibel, ein `migrate down` ist
	@# hier bewusst nicht Teil des Ablaufs.
	ssh $(REMOTE) "sudo mv $(REMOTE_BIN) $(REMOTE_BIN).failed && \
		sudo mv $(REMOTE_BIN).prev $(REMOTE_BIN) && \
		sudo systemctl restart $(REMOTE_SERVICE)"
	@$(MAKE) --no-print-directory _smoke ENV=$(ENV) SMOKE_FAIL="Rollback-Smoke-Test fehlgeschlagen - Prozess antwortet weiterhin nicht."
	@echo "Rollback abgeschlossen — $(REMOTE_BIN) ist wieder die vorherige Version."
	@echo "Das fehlgeschlagene Binary liegt als $(REMOTE_BIN).failed zur Analyse bereit."

deploy-new: _check-prod-only _check-new-remote ## Build + Deploy auf Umzugs-Zielhost (NEW_REMOTE=<alias> oder REMOTE_NEW aus .env)
	$(MAKE) deploy REMOTE=$(NEW_REMOTE_RESOLVED) REMOTE_DIR=$(NEW_REMOTE_DIR_RESOLVED)

# ── Nur Beta ─────────────────────────────────────────────────────────────────
# Datenpflege, die es für Prod nicht gibt. Pfade fest auf die Beta, unabhängig
# von ENV.
BETA_DIR := /var/lib/teamwerk-beta
BETA_DB  := $(BETA_DIR)/teamwerk.db
BETA_BIN := $(REMOTE_DIR)/teamwerk-beta

setup-beta: ## Beta-Instanz einrichten (= make setup-vps ENV=beta; Dienst, Env, Verzeichnisse, nginx, Zertifikat; idempotent)
	rsync -az deploy/setup-beta.sh deploy/teamwerk-beta.service deploy/nginx-teamwerk-beta.conf $(REMOTE):/tmp/teamwerk-beta-deploy/
	ssh $(REMOTE) "sudo bash /tmp/teamwerk-beta-deploy/setup-beta.sh"

seed-beta: ## Beta-DB aus anonymisierter Kopie von ./teamwerk.db neu aufsetzen (ersetzt die Beta-DB! FORCE=1 nötig, wenn schon eine existiert)
	@ssh $(REMOTE) "test -f $(BETA_BIN)" || { echo "seed-beta: erst 'make deploy ENV=beta'"; exit 1; }
	@if ssh $(REMOTE) "sudo test -f $(BETA_DB)" && [ "$(FORCE)" != "1" ]; then \
		echo "seed-beta: $(BETA_DB) existiert bereits — mit FORCE=1 ersetzen"; exit 1; fi
	@mkdir -p $(BUILD_DIR)
	@bash deploy/beta-seed.sh $(BUILD_DIR)/beta-seed.db > $(BUILD_DIR)/beta-password.txt
	rsync -az $(BUILD_DIR)/beta-seed.db $(REMOTE):/tmp/teamwerk-beta-seed.db
	ssh $(REMOTE) "sudo systemctl stop teamwerk-beta 2>/dev/null; \
		sudo rm -f $(BETA_DB) $(BETA_DB)-wal $(BETA_DB)-shm && \
		sudo mv /tmp/teamwerk-beta-seed.db $(BETA_DB) && \
		sudo chown www-data:www-data $(BETA_DB) && \
		sudo -u www-data $(BETA_BIN) migrate up --db $(BETA_DB) && \
		sudo systemctl start teamwerk-beta"
	@rm -f $(BUILD_DIR)/beta-seed.db
	@echo "Beta-DB eingespielt. Logins: vorstand@beispiel.de / trainer@beispiel.de (und alle anderen anonymisierten Konten)"
	@echo "Passwort steht in $(BUILD_DIR)/beta-password.txt"

mirror-beta: ## Prod-Daten (DB, Dokumente, Protokolle, BWHV-PDFs; ohne Videos und Bilder) auf die Beta spiegeln — ersetzt die Beta-Daten, FORCE=1 nötig. Prod wird nur gelesen.
	@[ "$(FORCE)" = "1" ] || { echo "mirror-beta: ersetzt alle Beta-Daten durch einen Prod-Stand — mit FORCE=1 bestätigen"; exit 1; }
	rsync -az deploy/beta-mirror.sh $(REMOTE):/tmp/teamwerk-beta-mirror.sh
	ssh $(REMOTE) "sudo bash /tmp/teamwerk-beta-mirror.sh; rc=\$$?; rm -f /tmp/teamwerk-beta-mirror.sh; exit \$$rc"

deploy-beta: ## Kurzform für make deploy ENV=beta
	@$(MAKE) --no-print-directory deploy ENV=beta

migrate-up: ## Migrationen lokal anwenden
	$(GO) run ./cmd/teamwerk migrate up

migrate-down: ## Letzte Migration lokal rückgängig machen
	$(GO) run ./cmd/teamwerk migrate down

migrate-remote-up: _check-env ## Ausstehende Migrationen auf VPS anwenden (ENV=prod|beta)
	ssh $(REMOTE) "$(AS_SERVICE_USER) $(REMOTE_BIN) migrate up --db $(DB_PATH)"

create-admin: ## Admin lokal anlegen (EMAIL= PASSWORD= NAME=)
	$(GO) run ./cmd/teamwerk create-admin --db ./teamwerk.db --email=$(EMAIL) --password=$(PASSWORD) --name=$(NAME)

create-admin-remote: _check-env ## Admin auf VPS anlegen (ENV=prod|beta; EMAIL= PASSWORD= NAME=)
	ssh $(REMOTE) "$(AS_SERVICE_USER) $(REMOTE_BIN) create-admin --db $(DB_PATH) --email=$(EMAIL) --password=$(PASSWORD) --name='$(NAME)'"

push-test-remote: _check-env ## Test-Push an User senden (ENV=prod|beta; USER=<id> TITLE=... BODY=... URL=...; Beta hat keine VAPID-Schlüssel und sendet nicht)
	ssh $(REMOTE) "$(AS_SERVICE_USER) $(REMOTE_BIN) push-test --env=$(REMOTE_ENV_FILE) --db=$(DB_PATH) --user=$(USER) --title='$(TITLE)' --body='$(BODY)' --url='$(or $(URL),/)'"

backup: _check-env ## DB + Tresor-Blobs (uploads: Profilfotos + SEPA-Mandat-PDFs) vom VPS sichern (ENV=prod → ./backup/<timestamp>/, ENV=beta → ./backup/beta/<timestamp>/)
	@echo "Erstelle DB-Backup ($(ENV)) auf VPS → $(BACKUP_DIR)/"
	@mkdir -p $(BACKUP_DIR)/uploads
	ssh $(REMOTE) "$(AS_SERVICE_USER) sqlite3 -readonly $(DB_PATH) '.backup /tmp/$(REMOTE_SERVICE)-backup.db'"
	scp $(REMOTE):/tmp/$(REMOTE_SERVICE)-backup.db $(BACKUP_DIR)/teamwerk.db
	ssh $(REMOTE) "rm -f /tmp/$(REMOTE_SERVICE)-backup.db"
	rsync -az $(REMOTE):$(UPLOAD_DIR_REMOTE)/ $(BACKUP_DIR)/uploads/
	@echo "Backup gespeichert: $(BACKUP_DIR)/"

backup-files: _check-env ## Alle kleinen Datei-Blobs vom VPS sichern (Dokumente, Beitragslauf, Chat-Media, Match-Report-Bilder, Trainingsnachweise) → ./backup/<timestamp>/
	@echo "Synchronisiere Datei-Blobs → $(BACKUP_DIR)/"
	@mkdir -p $(BACKUP_DIR)/files $(BACKUP_DIR)/beitragslauf-protokolle $(BACKUP_DIR)/media $(BACKUP_DIR)/match-report-images $(BACKUP_DIR)/training-diary
	@# files-Ordner (Dokumente-UI) — auf Prod immer vorhanden.
	rsync -az $(REMOTE):$(FILES_DIR_REMOTE)/ $(BACKUP_DIR)/files/
	@# Optionale Ordner mit test -d gaten, damit noch nicht angelegte
	@# Verzeichnisse den Backup-Lauf nicht abbrechen (Muster wie
	@# BEITRAGSLAUF_DIR).
	@if ssh $(REMOTE) "test -d $(BEITRAGSLAUF_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(BEITRAGSLAUF_DIR_REMOTE)/ $(BACKUP_DIR)/beitragslauf-protokolle/; \
	else \
		echo "  ($(BEITRAGSLAUF_DIR_REMOTE) existiert noch nicht — übersprungen.)"; \
	fi
	@if ssh $(REMOTE) "test -d $(MEDIA_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(MEDIA_DIR_REMOTE)/ $(BACKUP_DIR)/media/; \
	else \
		echo "  ($(MEDIA_DIR_REMOTE) existiert noch nicht — übersprungen.)"; \
	fi
	@if ssh $(REMOTE) "test -d $(MATCH_REPORT_IMAGE_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(MATCH_REPORT_IMAGE_DIR_REMOTE)/ $(BACKUP_DIR)/match-report-images/; \
	else \
		echo "  ($(MATCH_REPORT_IMAGE_DIR_REMOTE) existiert noch nicht — übersprungen.)"; \
	fi
	@# Trainingsnachweise: die Retention löscht sie 90 Tage nach Saisonende
	@# unwiderruflich vom Server — ein Backup ist der einzige Rückweg.
	@if ssh $(REMOTE) "test -d $(BWHV_REPORT_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(BWHV_REPORT_DIR_REMOTE)/ $(BACKUP_DIR)/bwhv-reports/; \
	fi
	@if ssh $(REMOTE) "test -d $(TRAINING_DIARY_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(TRAINING_DIARY_DIR_REMOTE)/ $(BACKUP_DIR)/training-diary/; \
	else \
		echo "  ($(TRAINING_DIARY_DIR_REMOTE) existiert noch nicht — übersprungen.)"; \
	fi
	@echo "Backup gespeichert: $(BACKUP_DIR)/"

backup-videos: _check-env ## Video-HLS-Transkodes vom VPS sichern (GB-Bereich; bewusst separat) → ./backup/<timestamp>/
	@echo "Prüfe Größe von $(VIDEO_STORAGE_DIR_REMOTE) auf $(REMOTE) …"
	@ssh $(REMOTE) "test -d $(VIDEO_STORAGE_DIR_REMOTE) && sudo du -sh $(VIDEO_STORAGE_DIR_REMOTE) 2>/dev/null || echo '  ($(VIDEO_STORAGE_DIR_REMOTE) existiert nicht)'"
	@mkdir -p $(BACKUP_DIR)/videos
	@if ssh $(REMOTE) "sudo test -d $(VIDEO_STORAGE_DIR_REMOTE)"; then \
		rsync -az --rsync-path='sudo rsync' $(REMOTE):$(VIDEO_STORAGE_DIR_REMOTE)/ $(BACKUP_DIR)/videos/; \
		echo "Videos gesichert: $(BACKUP_DIR)/videos/"; \
	else \
		echo "  Kein Video-Verzeichnis auf VPS — übersprungen."; \
	fi

restore-local: ## Letztes Backup (DB + Bilder) lokal einspielen (optional: BACKUP=/pfad/<timestamp>)
	@RESTORE="$${BACKUP:-$$(ls -dt $(BACKUP_ROOT)/20*/ 2>/dev/null | head -1)}"; \
	if [ -z "$$RESTORE" ] || [ ! -f "$$RESTORE/teamwerk.db" ]; then \
		echo "Fehler: kein Backup gefunden. Zuerst 'make backup' ausführen."; exit 1; \
	fi; \
	echo "WARNUNG: $(REPO_ROOT)/teamwerk.db und $(UPLOAD_DIR_LOCAL) werden mit Backup aus $$RESTORE überschrieben."; \
	printf "Fortfahren? [y/N] "; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		cp "$$RESTORE/teamwerk.db" $(REPO_ROOT)/teamwerk.db; \
		rm -f $(REPO_ROOT)/teamwerk.db-wal $(REPO_ROOT)/teamwerk.db-shm; \
		if [ -d "$$RESTORE/uploads" ]; then \
			mkdir -p $(UPLOAD_DIR_LOCAL) && rsync -a --delete "$$RESTORE/uploads/" $(UPLOAD_DIR_LOCAL)/; \
		fi; \
		echo "Restore abgeschlossen aus $$RESTORE."; \
	else \
		echo "Abgebrochen."; \
		exit 1; \
	fi

restore-local-files: ## Letztes Backup (Dokumente + Protokolle + Chat-Media + Match-Report-Bilder + Trainingsnachweise) lokal einspielen (optional: BACKUP=/pfad/<timestamp>)
	@RESTORE="$${BACKUP:-$$(ls -dt $(BACKUP_ROOT)/20*/ 2>/dev/null | head -1)}"; \
	if [ -z "$$RESTORE" ] || { [ ! -d "$$RESTORE/files" ] && [ ! -d "$$RESTORE/beitragslauf-protokolle" ] && [ ! -d "$$RESTORE/media" ] && [ ! -d "$$RESTORE/match-report-images" ] && [ ! -d "$$RESTORE/training-diary" ]; }; then \
		echo "Fehler: kein Backup gefunden. Zuerst 'make backup-files' ausführen."; exit 1; \
	fi; \
	echo "WARNUNG: $(FILES_DIR_LOCAL), $(BEITRAGSLAUF_DIR_LOCAL), $(MEDIA_DIR_LOCAL), $(MATCH_REPORT_IMAGE_DIR_LOCAL) und $(TRAINING_DIARY_DIR_LOCAL) werden mit Backup aus $$RESTORE überschrieben."; \
	printf "Fortfahren? [y/N] "; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		if [ -d "$$RESTORE/files" ]; then \
			mkdir -p $(FILES_DIR_LOCAL) && rsync -a --delete "$$RESTORE/files/" $(FILES_DIR_LOCAL)/; \
		fi; \
		if [ -d "$$RESTORE/beitragslauf-protokolle" ]; then \
			mkdir -p $(BEITRAGSLAUF_DIR_LOCAL) && rsync -a --delete "$$RESTORE/beitragslauf-protokolle/" $(BEITRAGSLAUF_DIR_LOCAL)/; \
		fi; \
		if [ -d "$$RESTORE/media" ]; then \
			mkdir -p $(MEDIA_DIR_LOCAL) && rsync -a --delete "$$RESTORE/media/" $(MEDIA_DIR_LOCAL)/; \
		fi; \
		if [ -d "$$RESTORE/match-report-images" ]; then \
			mkdir -p $(MATCH_REPORT_IMAGE_DIR_LOCAL) && rsync -a --delete "$$RESTORE/match-report-images/" $(MATCH_REPORT_IMAGE_DIR_LOCAL)/; \
		fi; \
		if [ -d "$$RESTORE/training-diary" ]; then \
			mkdir -p $(TRAINING_DIARY_DIR_LOCAL) && rsync -a --delete "$$RESTORE/training-diary/" $(TRAINING_DIARY_DIR_LOCAL)/; \
		fi; \
		echo "Restore abgeschlossen aus $$RESTORE."; \
	else \
		echo "Abgebrochen."; \
		exit 1; \
	fi

restore-local-videos: ## Letztes Backup (Videos) lokal einspielen (optional: BACKUP=/pfad/<timestamp>)
	@RESTORE="$${BACKUP:-$$(ls -dt $(BACKUP_ROOT)/20*/ 2>/dev/null | head -1)}"; \
	if [ -z "$$RESTORE" ] || [ ! -d "$$RESTORE/videos" ]; then \
		echo "Fehler: kein Video-Backup gefunden. Zuerst 'make backup-videos' ausführen."; exit 1; \
	fi; \
	echo "WARNUNG: $(VIDEO_STORAGE_DIR_LOCAL) wird mit Backup aus $$RESTORE überschrieben (potentiell mehrere GB)."; \
	printf "Fortfahren? [y/N] "; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		mkdir -p $(VIDEO_STORAGE_DIR_LOCAL) && rsync -a --delete "$$RESTORE/videos/" $(VIDEO_STORAGE_DIR_LOCAL)/; \
		echo "Restore abgeschlossen aus $$RESTORE."; \
	else \
		echo "Abgebrochen."; \
		exit 1; \
	fi

pull-db: backup restore-local ## Prod-DB + Tresor-Blobs (uploads) in einem Schritt sichern und lokal einspielen

pull-files: backup-files restore-local-files ## Alle kleinen Datei-Blobs in einem Schritt sichern und lokal einspielen

pull-videos: backup-videos restore-local-videos ## Videos in einem Schritt sichern und lokal einspielen (GB-Bereich, bewusst separates Target)

test: ## Backend + Frontend (vitest) Tests ausführen — schnell, ohne Race-Detector
	$(GO) test ./...
	cd web && pnpm test

test-race: ## Backend-Tests mit Race-Detector (~10× langsamer; vor Merge in heikle nebenläufige Bereiche)
	$(GO) test -race ./...

test-e2e: ## Playwright-E2E (echter Chromium gegen Prod-Binary + Seed-DB) — ~2–4 min, NICHT Teil von `make test`
	cd web && CI=true pnpm build
	cd web && CI=true ./node_modules/.bin/playwright test --config e2e/playwright.config.ts
	@rm -f e2e.db e2e.db-wal e2e.db-shm

folien: ## Schulungsfolien aus docs/schulung/folien.txt erzeugen → docs/schulung/folien/index.html (Sekunden, ohne DB)
	@python3 docs/schulung/tools/folien.py

schulung: folien ## Schulungsfolien + Screenshots aus anonymisierter Kopie der lokalen DB neu erzeugen → docs/schulung/folien/ (siehe docs/schulung/HOWTO.md)
	@GO="$(GO)" bash docs/schulung/tools/build.sh

lint: ## Statische Codeanalyse mit golangci-lint
	@if ! command -v golangci-lint > /dev/null 2>&1; then \
		echo "golangci-lint nicht gefunden. Installieren: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	golangci-lint run ./...

metrics: ## Code-Metriken erheben (Größe, Komplexität, Coverage, Lint-Dichte, Duplikation) — stdout + metrics/REPORT.md, Exit 0
	$(GO) run ./cmd/teamwerk metrics

metrics-gate: ## Wie metrics + Schwellwert-Prüfung gegen metrics/thresholds.yml (Exit 1 bei Regression)
	$(GO) run ./cmd/teamwerk metrics --gate

measure: ## Payload-/Fan-out-Messung erheben → metrics/PAYLOAD.md (build-tagged, nicht Teil von `make test`)
	$(GO) test -tags measure -run TestMeasure_WritesReport -count=1 ./internal/measure
	@echo "Payload-Report: metrics/PAYLOAD.md (Baseline: metrics/payload-baseline.md)"

coverage: ## Testabdeckung messen: Coverage-Bericht auf stdout + HTML nach /tmp/teamwerk-coverage.html
	$(GO) test -coverprofile=/tmp/teamwerk-coverage.out ./internal/...
	@$(GO) tool cover -func=/tmp/teamwerk-coverage.out | grep -E "^github|total:"
	$(GO) tool cover -html=/tmp/teamwerk-coverage.out -o /tmp/teamwerk-coverage.html
	@echo "HTML-Report: /tmp/teamwerk-coverage.html"

clean: ## Build-Artefakte löschen
	rm -rf $(BUILD_DIR) cmd/teamwerk/web/dist

# ── Server-Umzug ──────────────────────────────────────────────────────
# Ablauf: server-bootstrap (einmalig) → server-sync-data (beliebig oft) →
# server-cutover (finaler Umschalter). Siehe deploy/server-migration-runbook.md.

_check-remote:
	@if [ -z "$(REMOTE)" ]; then \
		echo "Fehler: REMOTE nicht in .env gesetzt. Ohne Quelle kein Umzug."; \
		exit 1; \
	fi

_check-new-remote:
	@if [ -z "$(NEW_REMOTE_RESOLVED)" ]; then \
		echo "Fehler: NEW_REMOTE=<alias> oder REMOTE_NEW= in .env setzen."; \
		echo "Beispiel: make server-bootstrap NEW_REMOTE=teamwerkNeu"; \
		exit 1; \
	fi

_check-base-url-new:
	@if [ -z "$(BASE_URL_NEW)" ]; then \
		echo "Fehler: BASE_URL_NEW in .env fehlt."; \
		echo "Beispiel: BASE_URL_NEW=https://teamwerk.team-stuttgart.org"; \
		exit 1; \
	fi
	@case "$(BASE_URL_NEW)" in \
		https://*) ;; \
		*) echo "Fehler: BASE_URL_NEW muss mit 'https://' beginnen (aktuell: $(BASE_URL_NEW))"; exit 1;; \
	esac

server-bootstrap: _check-prod-only _check-remote _check-new-remote _check-base-url-new build ## Server-Umzug: initialen Zielhost aufsetzen (setup + Env + DB + Storage + Deploy)
	@echo ">>> Bootstrap: Quelle=$(REMOTE)  Ziel=$(NEW_REMOTE_RESOLVED)  Domain=$(NEW_DOMAIN)"
	@echo ">>> A) setup-vps auf Ziel"
	rsync -az deploy/ $(NEW_REMOTE_RESOLVED):/tmp/teamwerk-deploy/
	ssh $(NEW_REMOTE_RESOLVED) "cd /tmp/teamwerk-deploy && sudo bash setup-vps.sh"
	@echo ">>> B) Env klonen (mit BASE_URL-Rewrite)"
	ssh $(REMOTE) "sudo cat /etc/teamwerk/env" \
		| sed -E "s|^BASE_URL=.*|BASE_URL=$(BASE_URL_NEW)|" \
		| ssh $(NEW_REMOTE_RESOLVED) "sudo tee /etc/teamwerk/env > /dev/null && sudo chmod 600 /etc/teamwerk/env"
	@echo ">>> C) Better-Stack-Konfigurationsdateien klonen"
	@for f in heartbeat-url betterstack-logs-token betterstack-metrics-token betterstack-metrics-endpoint; do \
		if ssh $(REMOTE) "sudo test -f /etc/teamwerk/$$f"; then \
			ssh $(REMOTE) "sudo cat /etc/teamwerk/$$f" \
				| ssh $(NEW_REMOTE_RESOLVED) "sudo tee /etc/teamwerk/$$f > /dev/null && sudo chmod 600 /etc/teamwerk/$$f"; \
			echo "    kopiert: $$f"; \
		else \
			echo "    übersprungen (Quelle hat kein /etc/teamwerk/$$f): $$f"; \
		fi \
	done
	@echo ">>> D) Zielhost-Service stoppen (falls existent)"
	ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl stop teamwerk 2>/dev/null || true"
	@echo ">>> E) DB-Snapshot Quelle → Ziel (sqlite3 .backup, WAL-safe)"
	ssh $(REMOTE) "sudo sqlite3 $(DB_PATH) '.backup /tmp/teamwerk-migration.db' && sudo chmod 644 /tmp/teamwerk-migration.db"
	ssh $(REMOTE) "sudo cat /tmp/teamwerk-migration.db" \
		| ssh $(NEW_REMOTE_RESOLVED) "sudo mkdir -p $(dir $(DB_PATH)) && sudo tee $(DB_PATH) > /dev/null && sudo rm -f $(DB_PATH)-wal $(DB_PATH)-shm"
	ssh $(REMOTE) "sudo rm -f /tmp/teamwerk-migration.db"
	@echo ">>> F) Storage-Ordner synchronisieren (Direkt-Rsync zwischen Remotes)"
	@for d in $(UPLOAD_DIR_REMOTE) $(FILES_DIR_REMOTE) $(MEDIA_DIR_REMOTE) $(BEITRAGSLAUF_DIR_REMOTE) $(MATCH_REPORT_IMAGE_DIR_REMOTE) $(TRAINING_DIARY_DIR_REMOTE) $(VIDEO_STORAGE_DIR_REMOTE); do \
		if ssh $(REMOTE) "sudo test -d $$d"; then \
			echo "    rsync $$d"; \
			ssh $(REMOTE) "sudo rsync -az -e 'ssh -o StrictHostKeyChecking=accept-new' $$d/ $(NEW_REMOTE_RESOLVED):$$d/" \
				|| { echo "    Direkt-Rsync fehlgeschlagen, fallback über Laptop-Disk"; \
				     TMP=$$(mktemp -d); \
				     rsync -az --rsync-path='sudo rsync' $(REMOTE):$$d/ $$TMP/ && rsync -az --rsync-path='sudo rsync' $$TMP/ $(NEW_REMOTE_RESOLVED):$$d/; \
				     rm -rf $$TMP; }; \
		else \
			echo "    übersprungen (Quelle hat kein $$d)"; \
		fi \
	done
	@echo ">>> G) Owner-Fix auf Ziel"
	ssh $(NEW_REMOTE_RESOLVED) "sudo chown -R www-data:www-data $(dir $(DB_PATH)) 2>/dev/null || true; sudo chown -R www-data:www-data /storage 2>/dev/null || true"
	@echo ">>> H) Binary deployen (mit umgebogenem REMOTE)"
	$(MAKE) deploy REMOTE=$(NEW_REMOTE_RESOLVED) REMOTE_DIR=$(NEW_REMOTE_DIR_RESOLVED)
	@echo ">>> I) Smoke-Test /api/healthz (IP + Host-Header)"
	@RESP=$$(ssh $(NEW_REMOTE_RESOLVED) "curl -k -s -H 'Host: $(NEW_DOMAIN)' https://localhost/api/healthz"); \
	echo "    Response: $$RESP"; \
	echo "$$RESP" | grep -q '"status":"ok"' && echo "$$RESP" | grep -q '"db":"ok"' \
		|| { echo "Fehler: /api/healthz auf Ziel nicht ok"; exit 1; }
	@echo ">>> J) BASE_URL auf Ziel verifizieren"
	@ssh $(NEW_REMOTE_RESOLVED) "sudo grep '^BASE_URL=' /etc/teamwerk/env"
	@echo ""
	@echo "Bootstrap fertig. Nächste Schritte:"
	@echo "  1. Testphase: /etc/hosts-Zeile lokal setzen: $$'\t'$(patsubst https://%,%,$(BASE_URL_NEW)) → Ziel-IP"
	@echo "  2. Bei Bedarf: make server-sync-data NEW_REMOTE=$(NEW_REMOTE_RESOLVED)"
	@echo "  3. DNS + Certbot: siehe deploy/server-migration-runbook.md Abschnitt 3"
	@echo "  4. Cutover: make server-cutover NEW_REMOTE=$(NEW_REMOTE_RESOLVED)"

server-sync-data: _check-prod-only _check-remote _check-new-remote _check-base-url-new build ## Server-Umzug: DB + Storage von Quelle auf Ziel neu synchronisieren (überschreibt Testdaten auf Ziel)
	@if [ "$$MAKE_CONFIRMED" = "1" ]; then \
		echo ">>> Auto-Confirm (aus server-cutover)"; \
	else \
		printf "server-sync-data überschreibt DB und Storage auf $(NEW_REMOTE_RESOLVED) mit einem frischen Snapshot von $(REMOTE). Testdaten auf Ziel gehen verloren. Fortfahren? [y/N] "; \
		read ans; \
		case "$$ans" in y|Y) ;; *) echo "Abgebrochen." ; exit 1;; esac; \
	fi
	@echo ">>> Sync: Quelle=$(REMOTE)  Ziel=$(NEW_REMOTE_RESOLVED)"
	@echo ">>> A) Ziel-Service stoppen"
	ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl stop teamwerk"
	@echo ">>> A2) Aktuelles Binary auf Ziel installieren (sonst kennt migrate in Schritt E neuere Schema-Versionen aus dem Snapshot nicht)"
	rsync -az $(BUILD_DIR)/$(BINARY) $(NEW_REMOTE_RESOLVED):/tmp/$(BINARY).new
	ssh $(NEW_REMOTE_RESOLVED) "sudo mv /tmp/$(BINARY).new $(NEW_REMOTE_DIR_RESOLVED)/$(BINARY)"
	@echo ">>> B) DB-Snapshot Quelle → Ziel"
	ssh $(REMOTE) "sudo sqlite3 $(DB_PATH) '.backup /tmp/teamwerk-migration.db' && sudo chmod 644 /tmp/teamwerk-migration.db"
	ssh $(REMOTE) "sudo cat /tmp/teamwerk-migration.db" \
		| ssh $(NEW_REMOTE_RESOLVED) "sudo tee $(DB_PATH) > /dev/null && sudo rm -f $(DB_PATH)-wal $(DB_PATH)-shm"
	ssh $(REMOTE) "sudo rm -f /tmp/teamwerk-migration.db"
	@echo ">>> C) Storage-Ordner synchronisieren"
	@for d in $(UPLOAD_DIR_REMOTE) $(FILES_DIR_REMOTE) $(MEDIA_DIR_REMOTE) $(BEITRAGSLAUF_DIR_REMOTE) $(MATCH_REPORT_IMAGE_DIR_REMOTE) $(TRAINING_DIARY_DIR_REMOTE) $(VIDEO_STORAGE_DIR_REMOTE); do \
		if ssh $(REMOTE) "sudo test -d $$d"; then \
			echo "    rsync $$d"; \
			ssh $(REMOTE) "sudo rsync -az --delete -e 'ssh -o StrictHostKeyChecking=accept-new' $$d/ $(NEW_REMOTE_RESOLVED):$$d/" \
				|| { echo "    Direkt-Rsync fehlgeschlagen, fallback über Laptop-Disk"; \
				     TMP=$$(mktemp -d); \
				     rsync -az --delete --rsync-path='sudo rsync' $(REMOTE):$$d/ $$TMP/ && rsync -az --delete --rsync-path='sudo rsync' $$TMP/ $(NEW_REMOTE_RESOLVED):$$d/; \
				     rm -rf $$TMP; }; \
		fi \
	done
	@echo ">>> D) Owner-Fix auf Ziel"
	ssh $(NEW_REMOTE_RESOLVED) "sudo chown -R www-data:www-data $(dir $(DB_PATH)) 2>/dev/null || true; sudo chown -R www-data:www-data /storage 2>/dev/null || true"
	@echo ">>> E) migrate up auf Ziel (nach Snapshot; das in A2 installierte Binary bringt das Schema auf seinen Stand)"
	ssh $(NEW_REMOTE_RESOLVED) "$(NEW_REMOTE_DIR_RESOLVED)/$(BINARY) migrate up --db $(DB_PATH)"
	@echo ">>> F) Ziel-Service starten"
	ssh $(NEW_REMOTE_RESOLVED) "sudo systemctl start teamwerk"
	@echo ">>> G) Smoke-Test /api/healthz"
	@RESP=$$(ssh $(NEW_REMOTE_RESOLVED) "curl -k -s -H 'Host: $(NEW_DOMAIN)' https://localhost/api/healthz"); \
	echo "    Response: $$RESP"; \
	echo "$$RESP" | grep -q '"status":"ok"' && echo "$$RESP" | grep -q '"db":"ok"' \
		|| { echo "Fehler: /api/healthz auf Ziel nicht ok"; exit 1; }
	@echo "Sync fertig."

server-cutover: _check-prod-only _check-remote _check-new-remote _check-base-url-new ## Server-Umzug: Alt-Host auf 301-Redirect umschalten (final)
	@printf "server-cutover stoppt teamwerk auf $(REMOTE) und schaltet den Alt-Host auf 301 → $(BASE_URL_NEW). Ein letzter server-sync-data läuft davor. Fortfahren? [y/N] "; \
	read ans; \
	case "$$ans" in y|Y) ;; *) echo "Abgebrochen." ; exit 1;; esac
	@echo ">>> Cutover: Quelle=$(REMOTE) ($(SOURCE_DOMAIN))  Ziel=$(NEW_REMOTE_RESOLVED) ($(NEW_DOMAIN))"
	@echo ">>> A) Letzter Daten-Sync"
	MAKE_CONFIRMED=1 $(MAKE) server-sync-data NEW_REMOTE=$(NEW_REMOTE_RESOLVED)
	@echo ">>> B) Alt-Host: teamwerk-Service stoppen und disablen"
	ssh $(REMOTE) "sudo systemctl stop teamwerk && sudo systemctl disable teamwerk"
	@echo ">>> C) Alt-Host: Nginx-Config-Backup"
	ssh $(REMOTE) "sudo cp /etc/nginx/sites-available/$(SOURCE_DOMAIN) /etc/nginx/sites-available/$(SOURCE_DOMAIN).$(TS).bak && echo '    Backup: /etc/nginx/sites-available/$(SOURCE_DOMAIN).$(TS).bak'"
	@echo ">>> D) Alt-Host: Redirect-Config deployen"
	sed "s|{{SOURCE_DOMAIN}}|$(SOURCE_DOMAIN)|g; s|{{NEW_BASE_URL}}|$(BASE_URL_NEW)|g" deploy/nginx-redirect.conf \
		| ssh $(REMOTE) "sudo tee /etc/nginx/sites-available/$(SOURCE_DOMAIN) > /dev/null"
	@echo ">>> E) nginx -t und reload"
	ssh $(REMOTE) "sudo nginx -t && sudo systemctl reload nginx" \
		|| { echo "Fehler: nginx-Reload fehlgeschlagen — Backup zurücksichern und teamwerk-Service manuell starten"; exit 1; }
	@echo ">>> F) Verifikation: Redirect aktiv"
	@STATUS=$$(ssh $(REMOTE) "curl -k -s -o /dev/null -w '%{http_code}' -H 'Host: $(SOURCE_DOMAIN)' https://localhost/api/healthz"); \
	if [ "$$STATUS" != "301" ]; then \
		echo "Fehler: Erwartet HTTP 301 auf Alt-Host, bekommen: $$STATUS"; \
		exit 1; \
	fi; \
	echo "    /api/healthz auf Alt-Host liefert 301 (Redirect aktiv)"
	@echo ""
	@echo "Cutover fertig. Nachpflege (manuell):"
	@echo "  1. Better-Stack HTTP-Monitor umhängen: URL → $(BASE_URL_NEW)/api/healthz"
	@echo "  2. User informieren (Push/Broadcast/Vorstandsansage) — Kernpunkte:"
	@echo "     • Neue URL: $(BASE_URL_NEW)"
	@echo "     • Bookmarks werden per 301 weitergeleitet"
	@echo "     • PWA-Nutzer: alte PWA vom Homescreen löschen, neue URL aufrufen,"
	@echo "       „Zum Homescreen hinzufügen\" erneut, Push neu erlauben"
	@echo "  3. Push-Endpoints der alten Origin sterben mit HTTP 410 → automatisches Cleanup"
