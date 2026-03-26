# syntax=docker/dockerfile:1
# Build stage — run on the BUILD host platform so Go cross-compiles natively (no QEMU needed)
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Cross-compile for the target platform without QEMU emulation
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /app/bin/issue-kanban-mcp ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/bin/issue-kanban-mcp /app/

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 9292

# Set environment variables
ENV PORT=9292
ENV DB_PATH=/app/data/tasks.db

# Run the binary
ENTRYPOINT ["/app/issue-kanban-mcp"]
CMD ["-port=9292", "-db=/app/data/tasks.db", "-mcp=http"]
