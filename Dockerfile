# Multi-stage build for backend server
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files first (for Docker layer caching)
COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

# Copy source code
COPY . .

# Build server binary with optimizations and reduced memory usage
# Use -p=1 to limit parallelism and reduce memory consumption
# GOMEMLIMIT limits memory usage during compilation
ENV GOMEMLIMIT=512MiB
RUN CGO_ENABLED=0 GOOS=linux go build -p=1 -ldflags="-w -s" -o /bin/server ./cmd/server

# Final stage - minimal Alpine image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary from builder
COPY --from=builder /bin/server /usr/local/bin/server

# Set working directory
WORKDIR /app

# Health check for server
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider --method=GET http://localhost:8080/health || exit 1

# Default command: run server
CMD ["server"]
