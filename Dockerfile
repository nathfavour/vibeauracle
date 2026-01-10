# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git build-base

WORKDIR /app

# Copy the entire workspace
COPY . .

# Build the main CLI tool
# Using go build from the root, targeting the vibe command
RUN go build -o /app/vibe ./cmd/vibe

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/vibe /usr/local/bin/vibe

# Expose port for the daemon/gRPC if needed (defaulting to a common one)
EXPOSE 50051

# Default command
ENTRYPOINT ["vibe"]
CMD ["--help"]

