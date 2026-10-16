# ---- Build Stage ----
FROM golang:1.26 AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the entire project
COPY . .

# Build the Go binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o openbucket .

# ---- Runtime Stage ----
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/openbucket /app/

EXPOSE 8080

# Healthy only when the server can use a pooled DB connection within 2s
# (GET /_health via `openbucket healthcheck`; the image has no curl/wget).
# A plain process check stayed green through the 2026-10-15 outage, when the
# DB pool deadlocked and cdn.trailblaze.to hung for ~15 h (#33). Lattice
# health-gates deploys on this and reports unhealthy containers; it does not
# restart them.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
  CMD ["./openbucket", "healthcheck"]

CMD ["./openbucket"]
