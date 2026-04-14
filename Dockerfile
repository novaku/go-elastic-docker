# ─────────────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install certificates + git (needed for private modules)
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go.mod + go.sum first for better caching
COPY go.mod go.sum ./

# Copy vendor folder (pre-resolved dependencies)
COPY vendor ./vendor

# Copy source code last (changes here won't invalidate dependency cache)
COPY . .

# Compute version once
RUN VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /app/server \
    ./cmd/api

# ─────────────────────────────────────────────────
# Stage 2: Runtime (distroless — minimal attack surface)
# ─────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot-arm64

# Copy CA certs so HTTPS to external ES clusters works
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Viper reads config files from ./config relative to the working directory.
WORKDIR /home/nonroot
COPY --from=builder /app/config ./config

COPY --from=builder /app/server /server

# Run as non-root user (uid 65532 = nonroot in distroless)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
