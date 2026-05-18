BINARY     := teamwerk
BUILD_DIR  := bin
REMOTE     := $(shell grep REMOTE_USER .env 2>/dev/null | cut -d= -f2)@$(shell grep REMOTE_HOST .env 2>/dev/null | cut -d= -f2)
REMOTE_DIR := $(shell grep REMOTE_DIR .env 2>/dev/null | cut -d= -f2)
DB_PATH    := /var/lib/teamwerk/teamwerk.db

.PHONY: init dev build deploy migrate-up migrate-down create-admin env clean

env:
	@if [ -f .env ]; then echo ".env existiert bereits – nichts geändert."; else \
		SECRET=$$(openssl rand -hex 32); \
		sed "s/change-me-to-a-random-secret/$$SECRET/" .env.example > .env; \
		echo ".env erstellt (JWT_SECRET automatisch gesetzt)."; \
	fi

init:
	go mod tidy
	cd web && pnpm install

dev:
	@echo "Starting backend on :8080 and frontend dev server..."
	@go run ./cmd/teamwerk &
	@cd web && pnpm dev

build:
	cd web && pnpm build
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/teamwerk

deploy: build
	rsync -az $(BUILD_DIR)/$(BINARY) $(REMOTE):$(REMOTE_DIR)/$(BINARY).new
	ssh $(REMOTE) "mv $(REMOTE_DIR)/$(BINARY).new $(REMOTE_DIR)/$(BINARY) && \
		$(REMOTE_DIR)/$(BINARY) migrate up --db $(DB_PATH) && \
		sudo systemctl restart teamwerk"
	@echo "Deployed successfully."

migrate-up:
	go run ./cmd/teamwerk migrate up --db ./teamwerk.db

migrate-down:
	go run ./cmd/teamwerk migrate down --db ./teamwerk.db

create-admin:
	go run ./cmd/teamwerk create-admin --db ./teamwerk.db --email=$(EMAIL) --password=$(PASSWORD) --name=$(NAME)

clean:
	rm -rf $(BUILD_DIR) cmd/teamwerk/web/dist
