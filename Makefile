BINARY     := teamwerk
BUILD_DIR  := bin
REMOTE     := $(shell grep '^REMOTE=' .env 2>/dev/null | cut -d= -f2)
REMOTE_DIR := $(shell grep '^REMOTE_DIR=' .env 2>/dev/null | cut -d= -f2)
DB_PATH    := /var/lib/teamwerk/teamwerk.db
EMAIL      ?= $(shell grep '^EMAIL=' .env 2>/dev/null | cut -d= -f2-)
PASSWORD   ?= $(shell grep '^PASSWORD=' .env 2>/dev/null | cut -d= -f2-)
NAME       ?= $(shell grep '^NAME=' .env 2>/dev/null | cut -d= -f2-)

.PHONY: help init dev dev-remote build deploy setup-vps migrate-up migrate-down migrate-remote-up migrate-remote-down create-admin create-admin-remote env clean

.DEFAULT_GOAL := help

help: ## Diesen Hilfetext anzeigen
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-22s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

env: ## .env aus .env.example erstellen
	@if [ -f .env ]; then echo ".env existiert bereits – nichts geändert."; else \
		SECRET=$$(openssl rand -hex 32); \
		sed "s/change-me-to-a-random-secret/$$SECRET/" .env.example > .env; \
		echo ".env erstellt (JWT_SECRET automatisch gesetzt)."; \
	fi

init: ## Abhängigkeiten installieren (go mod tidy, pnpm install)
	go mod tidy
	cd web && pnpm install

dev: ## Backend (mit air Auto-Reload) + Vite Dev-Server lokal starten
	@echo "Starting backend on :8080 (with auto-reload) and frontend dev server..."
	@if command -v air > /dev/null 2>&1; then air & \
	elif [ -x "$$(go env GOPATH)/bin/air" ]; then $$(go env GOPATH)/bin/air & \
	else echo "air not found, using go run (no auto-reload)"; go run ./cmd/teamwerk & fi
	@sleep 1
	@cd web && pnpm dev

dev-remote: ## SSH-Tunnel zum VPS + Vite Dev-Server (kein lokales Backend)
	@echo "Opening SSH tunnel to $(REMOTE) and starting frontend dev server..."
	@ssh -N -L 8080:localhost:8080 $(REMOTE) &
	@cd web && pnpm dev

build: ## Frontend + Backend für Linux/amd64 bauen
	cd web && pnpm build
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/teamwerk

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

migrate-up: ## Migrationen lokal anwenden
	go run ./cmd/teamwerk migrate up

migrate-down: ## Letzte Migration lokal rückgängig machen
	go run ./cmd/teamwerk migrate down

migrate-remote-up: ## Ausstehende Migrationen auf VPS anwenden
	ssh $(REMOTE) "$(REMOTE_DIR)/$(BINARY) migrate up --db $(DB_PATH)"

migrate-remote-down: ## Letzte Migration auf VPS rückgängig machen
	ssh $(REMOTE) "$(REMOTE_DIR)/$(BINARY) migrate down --db $(DB_PATH)"

create-admin: ## Admin lokal anlegen (EMAIL= PASSWORD= NAME=)
	go run ./cmd/teamwerk create-admin --db ./teamwerk.db --email=$(EMAIL) --password=$(PASSWORD) --name=$(NAME)

create-admin-remote: ## Admin auf VPS anlegen (EMAIL= PASSWORD= NAME=)
	ssh $(REMOTE) "/usr/local/bin/teamwerk create-admin --db $(DB_PATH) --email=$(EMAIL) --password=$(PASSWORD) --name='$(NAME)'"

clean: ## Build-Artefakte löschen
	rm -rf $(BUILD_DIR) cmd/teamwerk/web/dist
