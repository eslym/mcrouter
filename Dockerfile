FROM golang:1.20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcrouter .

# Create a minimal runtime image
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata openssh

# Copy the binary from the builder stage
COPY --from=builder /app/mcrouter /app/mcrouter

# Create directories for SSH key and auth
RUN mkdir -p /app/keys /app/users

# Set default environment variables
ENV SSH_LISTEN=0.0.0.0:2222
ENV MINECRAFT_LISTEN=0.0.0.0:25565
ENV SSH_KEY_PATH=/app/keys/id_rsa
ENV AUTH_DIR=/app/users
ENV BAN_IP=false
ENV BAN_DURATION=48
ENV LOG_REJECTED=false
ENV WHITELIST_DOMAINS=""
ENV BLACKLIST_DOMAINS=""

# Expose ports
EXPOSE 2222 25565

# Create volumes for persistent data
VOLUME ["/app/keys", "/app/users"]

# Copy entrypoint script and make it executable
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
