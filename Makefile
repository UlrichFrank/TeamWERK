BINARY     := teamwerk
BUILD_DIR  := bin
# Prefer the system Go at /usr/local/go if present (matches go.mod toolchain).
# Falls back to whatever 'go' is on PATH.
GO         := $(or $(wildcard /usr/local/go/bin/go),go)
REPO_ROOT  := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
REMOTE     := $(shell grep '^REMOTE=' .env 2>/dev/null | cut -d= -f2)
REMOTE_DIR := $(shell grep '^REMOTE_DIR=' .env 2>/dev/null | cut -d= -f2)
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

.PHONY: help init hooks dev dev-remote build deploy setup-vps migrate-up migrate-down migrate-remote-up create-admin create-admin-remote push-test-remote env clean backup backup-files restore-local restore-local-files pull-db pull-files test test-race lint coverage metrics metrics-gate

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
