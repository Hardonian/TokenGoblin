# Hardonian standard justfile — TokenGoblin
# Install just: pipx install rust-just or cargo install just

# Detect and install deps
bootstrap:
    go mod download
    go mod verify
    cd frontend && npm ci

# Run backend server locally
dev:
    go run ./cmd/server

# Run frontend dev server
dev-frontend:
    cd frontend && npm run dev

# Build both backend binaries
build:
    go build -o token-goblin ./cmd/server
    go build -o token-goblin-audit ./cmd/audit

# Run LLM spend audit demo on sample export
audit-demo:
    go run ./cmd/audit --input ./examples/sample_llm_export.csv --out ./out/sample_audit

# Run test suites (Go race-detector + frontend tests)
test:
    go test -v -race ./...
    cd frontend && npm run test:ci

# Run smoke test on ingestion, storage, and anomaly pipeline
smoke:
    go run ./cmd/smoke

# Check health and readiness probes
status:
    @curl -fsS http://127.0.0.1:8080/healthz || echo "Backend /healthz probe unavailable on :8080"
    @curl -fsS http://127.0.0.1:8080/readyz || echo "Backend /readyz probe unavailable on :8080"
