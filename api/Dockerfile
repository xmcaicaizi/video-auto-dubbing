# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY api/go.mod api/go.sum ./api/
COPY shared/go.mod shared/go.sum ./shared/
RUN cd /app/api && go mod download

# Copy source code
COPY shared ./shared
COPY api ./api

# Ensure go.sum is complete for builds inside container
RUN cd /app/api && go mod tidy

# Build the application
RUN cd /app/api && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api/api .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./api"]

