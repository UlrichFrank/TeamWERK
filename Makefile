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
DB_PATH        := /var/lib/teamwerk/teamwerk.db
UPLOAD_DIR_REMOTE        := /var/lib/teamwerk/uploads
FILES_DIR_REMOTE         := /var/lib/teamwerk/files
BEITRAGSLAUF_DIR_REMOTE  := /var/lib/teamwerk/beitragslauf-protokolle
UPLOAD_DIR_LOCAL         := $(REPO_ROOT)/storage/uploads
FILES_DIR_LOCAL          := $(REPO_ROOT)/storage/files
BEITRAGSLAUF_DIR_LOCAL   := $(REPO_ROOT)/storage/beitragslauf-protokolle
EMAIL      ?= $(shell grep '^EMAIL=' .env 2>/dev/null | cut -d= -f2-)
PASSWORD   ?= $(shell grep '^PASSWORD=' .env 2>/dev/null | cut -d= -f2-)
NAME       ?= $(shell grep '^NAME=' .env 2>/dev/null | cut -d= -f2-)
TS         := $(shell date +%Y-%m-%dT%H-%M-%S)
BACKUP_DIR := $(REPO_ROOT)/backup/$(TS)

.PHONY: help init hooks dev dev-remote build deploy deploy-new setup-vps migrate-up migrate-down migrate-remote-up create-admin create-admin-remote push-test-remote env clean backup backup-files restore-local restore-local-files pull-db pull-files test test-race lint coverage metrics metrics-gate server-bootstrap server-sync-data server-cutover _check-remote _check-new-remote _check-base-url-new

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

dev-remote: ## SSH-Tunnel zum VPS + Vite Dev-Server (kein lokales Backend)
	@echo "Opening SSH tunnel to $(REMOTE) and starting frontend dev server..."
	@ssh -N -L 8080:localhost:8080 $(REMOTE) &
	@cd web && pnpm dev

build: ## Frontend + Backend für Linux/amd64 bauen
	@git log --format="%ad|%s" --date=format:"%d.%m.%Y" --no-merges \
	  | grep -E "\|(feat|fix)(\([^)]*\))?:" \
	  | python3 scripts/gen-changelog.py > web/public/CHANGELOG.md
	cd web && pnpm build
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X 'main.buildHash=$(shell git rev-parse --short HEAD)'" -o $(BUILD_DIR)/$(BINARY) ./cmd/teamwerk

setup-vps: ## VPS einmalig einrichten (Nginx, Certbot, systemd)
	rsync -az deploy/ $(REMOTE):/tmp/teamwerk-deploy/
	ssh $(REMOTE) "cd /tmp/teamwerk-deploy && sudo bash setup-vps.sh"

deploy: build ## Build + Deploy auf VPS (Binary, Migrations, Service-Neustart)
	rsync -az $(BUILD_DIR)/$(BINARY) $(REMOTE):/tmp/$(BINARY).new
	rsync -az deploy/teamwerk.service $(REMOTE):/tmp/teamwerk.service
	ssh $(REMOTE) "[ -f /etc/teamwerk/env ]" 2>/dev/null || \
		grep -E '^(PORT|DB_PATH|JWT_SECRET|BASE_URL|SMTP_HOST|SMTP_PORT|SMTP_USER|SMTP_PASS|SMTP_FROM)=' .env | \
		sed 's|DB_PATH=.*|DB_PATH=/var/lib/teamwerk/teamwerk.db|; s|BASE_URL=.*|BASE_URL=https://intern.team-stuttgart.org|' | \
		ssh $(REMOTE) "sudo mkdir -p /etc/teamwerk && sudo tee /etc/teamwerk/env > /dev/null && sudo chmod 600 /etc/teamwerk/env"
	ssh $(REMOTE) "sudo mkdir -p $(dir $(DB_PATH)) && \
		if ! [ -f /etc/systemd/system/teamwerk.service ]; then \
			sudo mv /tmp/teamwerk.service /etc/systemd/system/teamwerk.service && \
			sudo systemctl daemon-reload && sudo systemctl enable teamwerk; \
		fi && \
		sudo mv /tmp/$(BINARY).new $(REMOTE_DIR)/$(BINARY) && \
		$(REMOTE_DIR)/$(BINARY) migrate up --db $(DB_PATH) && \
		sudo chown www-data:www-data $(DB_PATH) $(DB_PATH)-shm $(DB_PATH)-wal 2>/dev/null; \
		sudo systemctl restart teamwerk"
	@echo "Deployed successfully."
	@git rev-parse --short HEAD > .deployed-hash

deploy-new: _check-new-remote ## Build + Deploy auf Umzugs-Zielhost (NEW_REMOTE=<alias> oder REMOTE_NEW aus .env)
	$(MAKE) deploy REMOTE=$(NEW_REMOTE_RESOLVED) REMOTE_DIR=$(NEW_REMOTE_DIR_RESOLVED)

migrate-up: ## Migrationen lokal anwenden
	$(GO) run ./cmd/teamwerk migrate up

migrate-down: ## Letzte Migration lokal rückgängig machen
	$(GO) run ./cmd/teamwerk migrate down

migrate-remote-up: ## Ausstehende Migrationen auf VPS anwenden
	ssh $(REMOTE) "$(REMOTE_DIR)/$(BINARY) migrate up --db $(DB_PATH)"

create-admin: ## Admin lokal anlegen (EMAIL= PASSWORD= NAME=)
	$(GO) run ./cmd/teamwerk create-admin --db ./teamwerk.db --email=$(EMAIL) --password=$(PASSWORD) --name=$(NAME)

create-admin-remote: ## Admin auf VPS anlegen (EMAIL= PASSWORD= NAME=)
	ssh $(REMOTE) "/usr/local/bin/teamwerk create-admin --db $(DB_PATH) --email=$(EMAIL) --password=$(PASSWORD) --name='$(NAME)'"

push-test-remote: ## Test-Push an User senden (USER=<id> TITLE=... BODY=... URL=...)
	ssh $(REMOTE) "/usr/local/bin/teamwerk push-test --env=/etc/teamwerk/env --db=$(DB_PATH) --user=$(USER) --title='$(TITLE)' --body='$(BODY)' --url='$(or $(URL),/)'"

backup: ## Prod-DB + Bilder (uploads) auf VPS sichern (./backup/<timestamp>/)
	@echo "Erstelle DB-Backup auf VPS → $(BACKUP_DIR)/"
	@mkdir -p $(BACKUP_DIR)/uploads
	ssh $(REMOTE) "sqlite3 $(DB_PATH) '.backup /tmp/teamwerk-backup.db'"
	scp $(REMOTE):/tmp/teamwerk-backup.db $(BACKUP_DIR)/teamwerk.db
	ssh $(REMOTE) "rm -f /tmp/teamwerk-backup.db"
	rsync -az $(REMOTE):$(UPLOAD_DIR_REMOTE)/ $(BACKUP_DIR)/uploads/
	@echo "Backup gespeichert: $(BACKUP_DIR)/"

backup-files: ## Dokumente + Beitragslauf-Protokolle vom VPS sichern (./backup/<timestamp>/)
	@echo "Synchronisiere Dokumente + Protokolle → $(BACKUP_DIR)/"
	@mkdir -p $(BACKUP_DIR)/files $(BACKUP_DIR)/beitragslauf-protokolle
	rsync -az $(REMOTE):$(FILES_DIR_REMOTE)/ $(BACKUP_DIR)/files/
	@if ssh $(REMOTE) "test -d $(BEITRAGSLAUF_DIR_REMOTE)"; then \
		rsync -az $(REMOTE):$(BEITRAGSLAUF_DIR_REMOTE)/ $(BACKUP_DIR)/beitragslauf-protokolle/; \
	else \
		echo "  ($(BEITRAGSLAUF_DIR_REMOTE) existiert noch nicht — übersprungen.)"; \
	fi
	@echo "Backup gespeichert: $(BACKUP_DIR)/"

restore-local: ## Letztes Backup (DB + Bilder) lokal einspielen (optional: BACKUP=/pfad/<timestamp>)
	@RESTORE="$${BACKUP:-$$(ls -dt $(REPO_ROOT)/backup/20*/ 2>/dev/null | head -1)}"; \
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

restore-local-files: ## Letztes Backup (Dokumente + Protokolle) lokal einspielen (optional: BACKUP=/pfad/<timestamp>)
	@RESTORE="$${BACKUP:-$$(ls -dt $(REPO_ROOT)/backup/20*/ 2>/dev/null | head -1)}"; \
	if [ -z "$$RESTORE" ] || { [ ! -d "$$RESTORE/files" ] && [ ! -d "$$RESTORE/beitragslauf-protokolle" ]; }; then \
		echo "Fehler: kein Backup gefunden. Zuerst 'make backup-files' ausführen."; exit 1; \
	fi; \
	echo "WARNUNG: $(FILES_DIR_LOCAL) und $(BEITRAGSLAUF_DIR_LOCAL) werden mit Backup aus $$RESTORE überschrieben."; \
	printf "Fortfahren? [y/N] "; \
	read ans; \
	if [ "$$ans" = "y" ]; then \
		if [ -d "$$RESTORE/files" ]; then \
			mkdir -p $(FILES_DIR_LOCAL) && rsync -a --delete "$$RESTORE/files/" $(FILES_DIR_LOCAL)/; \
		fi; \
		if [ -d "$$RESTORE/beitragslauf-protokolle" ]; then \
			mkdir -p $(BEITRAGSLAUF_DIR_LOCAL) && rsync -a --delete "$$RESTORE/beitragslauf-protokolle/" $(BEITRAGSLAUF_DIR_LOCAL)/; \
		fi; \
		echo "Restore abgeschlossen aus $$RESTORE."; \
	else \
		echo "Abgebrochen."; \
		exit 1; \
	fi

pull-db: backup restore-local ## Prod-DB in einem Schritt sichern und lokal einspielen

pull-files: backup-files restore-local-files ## Dokumente + Protokolle in einem Schritt sichern und lokal einspielen

test: ## Backend + Frontend (vitest) Tests ausführen — schnell, ohne Race-Detector
	$(GO) test ./...
	cd web && pnpm test

test-race: ## Backend-Tests mit Race-Detector (~10× langsamer; vor Merge in heikle nebenläufige Bereiche)
	$(GO) test -race ./...

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
		echo "Beispiel: make server-bootstrap NEW_REMOTE=vServerNeu"; \
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

server-bootstrap: _check-remote _check-new-remote _check-base-url-new build ## Server-Umzug: initialen Zielhost aufsetzen (setup + Env + DB + Storage + Deploy)
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
	@for d in $(UPLOAD_DIR_REMOTE) $(FILES_DIR_REMOTE) $(BEITRAGSLAUF_DIR_REMOTE) /storage/videos; do \
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

server-sync-data: _check-remote _check-new-remote _check-base-url-new build ## Server-Umzug: DB + Storage von Quelle auf Ziel neu synchronisieren (überschreibt Testdaten auf Ziel)
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
	@for d in $(UPLOAD_DIR_REMOTE) $(FILES_DIR_REMOTE) $(BEITRAGSLAUF_DIR_REMOTE) /storage/videos; do \
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

server-cutover: _check-remote _check-new-remote _check-base-url-new ## Server-Umzug: Alt-Host auf 301-Redirect umschalten (final)
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
