# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY worker/go.mod worker/go.sum ./worker/
COPY shared/go.mod shared/go.sum ./shared/
RUN cd /app/worker && go mod download

# Copy source code
COPY shared ./shared
COPY worker ./worker

# Ensure go.sum is complete for builds inside container
RUN cd /app/worker && go mod tidy

# Build the application
RUN cd /app/worker && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/worker/worker .

# Create temp directory
RUN mkdir -p /tmp

# Expose port (if needed for health checks)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD ps aux | grep -v grep | grep worker || exit 1

# Run the application
CMD ["./worker"]

