# Stage 1: Build the Go backend binary
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git build-base

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the TokenGoblin server binary statically
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /go/bin/tokengoblin-server ./cmd/server

# Stage 2: Create the minimal production image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite

# Create a non-root user
RUN adduser -D -g '' appuser

WORKDIR /home/appuser/app

# Copy the binary from builder
COPY --from=builder /go/bin/tokengoblin-server ./tokengoblin-server

# Copy necessary static assets (like sqlite migrations if bundled locally)
# COPY --from=builder /app/data/migrations ./data/migrations

RUN chown -R appuser:appuser /home/appuser/app

USER appuser

EXPOSE 8080

CMD ["./tokengoblin-server"]
