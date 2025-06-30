.PHONY: help
help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
	go install github.com/securego/gosec/v2/cmd/gosec@v2.22.5
	go install github.com/swaggo/swag/cmd/swag@v1.16.4
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/vektra/mockery/v2@latest

.PHONY: mod
mod: ## Download and tidy Go modules
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the application
	go build -ldflags="-s -w" -o bin/api cmd/api/main.go

.PHONY: run
run: ## Run the application
	go run cmd/api/main.go

.PHONY: test
test: ## Run unit tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	go test -v -race -tags=integration ./internal/test/integration/...

.PHONY: test-e2e
test-e2e: ## Run e2e tests with k6
	k6 run scripts/k6/auth_flow.js

.PHONY: test-all
test-all: test test-integration ## Run all tests

.PHONY: bench
bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix ./...

.PHONY: security-scan
security-scan: ## Run security scan with gosec
	gosec -fmt json -out gosec-report.json ./...
	@echo "Security scan complete. Check gosec-report.json for details."

.PHONY: vuln-check
vuln-check: ## Check for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: docs
docs: ## Generate API documentation
	swag init -g cmd/api/main.go -o docs/swagger

.PHONY: migrate-up
migrate-up: ## Run database migrations up
	migrate -path migrations -database "$${DB_DSN}" up

.PHONY: migrate-down
migrate-down: ## Run database migrations down
	migrate -path migrations -database "$${DB_DSN}" down

.PHONY: migrate-create
migrate-create: ## Create a new migration (usage: make migrate-create name=create_users_table)
	migrate create -ext sql -dir migrations -seq $(name)

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t go-auth-jwt:latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env go-auth-jwt:latest

.PHONY: compose-up
compose-up: ## Start services with docker-compose
	docker compose up -d

.PHONY: compose-down
compose-down: ## Stop services with docker-compose
	docker compose down

.PHONY: compose-logs
compose-logs: ## View docker-compose logs
	docker compose logs -f

.PHONY: dev-up
dev-up: ## Start development environment with docker-compose
	docker compose -f docker-compose.dev.yml up -d
	@echo "Development environment started!"
	@echo "API: http://localhost:8080"
	@echo "MailHog: http://localhost:8025"

.PHONY: dev-down
dev-down: ## Stop development environment
	docker compose -f docker-compose.dev.yml down

.PHONY: dev-logs
dev-logs: ## View development environment logs
	docker compose -f docker-compose.dev.yml logs -f

.PHONY: dev-clean
dev-clean: ## Clean development environment (including volumes)
	docker compose -f docker-compose.dev.yml down -v

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.* gosec-report.json
	go clean -cache -testcache

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: mock
mock: ## Generate mocks
	mockery --all --output=internal/mocks

.PHONY: keys
keys: ## Generate RSA key pair for JWT
	@mkdir -p certs
	openssl genrsa -out certs/private.pem 4096
	openssl rsa -in certs/private.pem -pubout -out certs/public.pem
	@echo "RSA key pair generated in certs/"

.PHONY: dev
dev: ## Run in development mode with hot reload (requires air)
	@if ! command -v air &> /dev/null; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air

.PHONY: ci
ci: lint test security-scan ## Run CI checks locally