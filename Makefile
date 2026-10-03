# TokenGoblin Makefile

.PHONY: build audit audit-demo test lint fmt vet coverage run docker-build docker-run clean deps check frontend-install frontend-lint frontend-typecheck frontend-test frontend-build help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=gofmt
GOVET=$(GOCMD) vet
BINARY_NAME=token-goblin
AUDIT_BINARY_NAME=token-goblin-audit
BINARY_PATH=./$(BINARY_NAME)

# Build binaries
build:
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_NAME) ./cmd/server
	CGO_ENABLED=0 $(GOBUILD) -o $(AUDIT_BINARY_NAME) ./cmd/audit

# Build spend audit CLI
audit:
	CGO_ENABLED=0 $(GOBUILD) -o $(AUDIT_BINARY_NAME) ./cmd/audit

# Run audit demonstration on sample LLM usage export
audit-demo: audit
	./$(AUDIT_BINARY_NAME) --input ./examples/sample_llm_export.csv --out ./out/sample_audit

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
coverage:
	$(GOTEST) -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	$(GOFMT) -l -e -w .

# Vet code
vet:
	$(GOVET) ./...

# Run the application
run: build
	./$(BINARY_NAME)

# Build Docker images
docker-build:
	docker build -f Dockerfile.backend -t tokengoblin/backend .
	docker build -f Dockerfile.frontend -t tokengoblin/frontend .

# Run with docker-compose
docker-run:
	docker-compose up --build

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(AUDIT_BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf out/

# Download dependencies
deps:
	$(GOCMD) mod download
	$(GOCMD) mod verify

# Run all checks
check: fmt vet test

# Frontend commands
frontend-install:
	cd frontend && npm ci

frontend-lint:
	cd frontend && npm run lint

frontend-typecheck:
	cd frontend && npm run typecheck

frontend-test:
	cd frontend && npm run test:ci

frontend-build:
	cd frontend && npm run build

# Help
help:
	@echo "TokenGoblin Make targets:"
	@echo "  build         - Build Go binaries (server and audit CLI)"
	@echo "  audit         - Build spend audit CLI binary"
	@echo "  audit-demo    - Run spend audit generator on sample export"
	@echo "  test          - Run Go tests with race detector"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  run           - Build and run server binary"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-run    - Run with docker-compose"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and verify Go modules"
	@echo "  check         - Run fmt, vet, test"
	@echo "  frontend-*    - Frontend targets (install, lint, typecheck, test, build)"