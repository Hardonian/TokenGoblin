# TokenGoblin Makefile

.PHONY: build test lint fmt vet coverage run docker-build docker-run clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=gofmt
GOVET=$(GOCMD) vet
BINARY_NAME=token-goblin
BINARY_PATH=./$(BINARY_NAME)

# Build the binary
build:
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_NAME) ./cmd/server

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
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOCMD) mod download
	$(GOCMD) mod verify

# Run all checks
check: fmt vet test

# Frontend commands
frontend-install:
	cd frontend && pnpm install --frozen-lockfile

frontend-lint:
	cd frontend && pnpm run lint

frontend-typecheck:
	cd frontend && pnpm run typecheck

frontend-test:
	cd frontend && pnpm test --if-present

frontend-build:
	cd frontend && pnpm build

# Help
help:
	@echo "TokenGoblin Make targets:"
	@echo "  build         - Build Go binary"
	@echo "  test          - Run Go tests with race detector"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  run           - Build and run binary"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-run    - Run with docker-compose"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and verify Go modules"
	@echo "  check         - Run fmt, vet, test"
	@echo "  frontend-*    - Frontend targets (install, lint, typecheck, test, build)"