FROM golang:1.24-alpine AS builder

# Install dependencies for the build
RUN apk add --no-cache git

WORKDIR /app

# Copy only go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -a -installsuffix cgo -o user-search-service .

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Add CA certificates and tzdata for HTTPS and proper timezone
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/user-search-service .

# Create a non-root user to run the application
RUN adduser -D appuser && \
    chown -R appuser:appuser /app
USER appuser

# Expose the application port
EXPOSE 8084

# Set environment variables
ENV GIN_MODE=release

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8084/health || exit 1

# Set the entrypoint
CMD ["./user-search-service"] 