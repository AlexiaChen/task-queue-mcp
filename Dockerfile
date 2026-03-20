# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/task-queue-mcp ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/bin/task-queue-mcp /app/

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 9292

# Set environment variables
ENV PORT=9292
ENV DB_PATH=/app/data/tasks.db

# Run the binary
ENTRYPOINT ["/app/task-queue-mcp"]
CMD ["-port=9292", "-db=/app/data/tasks.db", "-mcp=http"]
