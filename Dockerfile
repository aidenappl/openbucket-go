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
CMD ["./openbucket"]
