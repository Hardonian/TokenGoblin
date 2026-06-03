# Build stage
FROM golang:1.25-alpine AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /token-goblin ./cmd/server

# Frontend build stage
FROM node:22-alpine AS frontend-build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Final stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy Go binary
COPY --from=go-build /token-goblin /app/token-goblin

# Copy Next.js standalone output
COPY --from=frontend-build /app/.next/standalone /app/frontend
COPY --from=frontend-build /app/.next/static /app/frontend/.next/static
COPY --from=frontend-build /app/public /app/frontend/public

# Default env
ENV PORT=8080
ENV FRONTEND_PORT=3000
ENV DATABASE_URL="file:/data/token_goblin.db"
ENV TZ=UTC

VOLUME ["/data"]
EXPOSE 8080 3000

# Start both services
CMD ["sh", "-c", "/app/token-goblin & cd /app/frontend && PORT=$FRONTEND_PORT node server.js"]
