# ─────────────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install certificates + git (needed for private modules)
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy source (vendor folder provides all dependencies offline)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
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
