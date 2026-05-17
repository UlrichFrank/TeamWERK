BINARY     := vereinswerk
BUILD_DIR  := bin
REMOTE     := $(shell grep REMOTE_USER .env 2>/dev/null | cut -d= -f2)@$(shell grep REMOTE_HOST .env 2>/dev/null | cut -d= -f2)
REMOTE_DIR := $(shell grep REMOTE_DIR .env 2>/dev/null | cut -d= -f2)
DB_PATH    := /var/lib/vereinswerk/vereinswerk.db

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
	@go run ./cmd/vereinswerk &
	@cd web && pnpm dev

build:
	cd web && pnpm build
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/vereinswerk

deploy: build
	rsync -az $(BUILD_DIR)/$(BINARY) $(REMOTE):$(REMOTE_DIR)/$(BINARY).new
	ssh $(REMOTE) "mv $(REMOTE_DIR)/$(BINARY).new $(REMOTE_DIR)/$(BINARY) && \
		$(REMOTE_DIR)/$(BINARY) migrate up --db $(DB_PATH) && \
		sudo systemctl restart vereinswerk"
	@echo "Deployed successfully."

migrate-up:
	go run ./cmd/vereinswerk migrate up --db ./vereinswerk.db

migrate-down:
	go run ./cmd/vereinswerk migrate down --db ./vereinswerk.db

create-admin:
	go run ./cmd/vereinswerk create-admin --db ./vereinswerk.db --email=$(EMAIL) --password=$(PASSWORD) --name=$(NAME)

clean:
	rm -rf $(BUILD_DIR) cmd/vereinswerk/web/dist
