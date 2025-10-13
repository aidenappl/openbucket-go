# ---- Build Stage ----
FROM golang:1.23 AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o openbucket .

# ---- Runtime Stage ----
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/openbucket /app/

EXPOSE 8080
CMD ["./openbucket"]
