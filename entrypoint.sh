#!/bin/sh

# Check if SSH key exists, if not generate it
if [ ! -f "$SSH_KEY_PATH" ]; then
  echo "SSH key not found at $SSH_KEY_PATH, generating new key..."
  # Ensure directory exists
  mkdir -p "$(dirname "$SSH_KEY_PATH")"
  # Generate SSH key without passphrase
  ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
  echo "SSH key generated successfully."
fi

ARGS="-S $SSH_LISTEN -M $MINECRAFT_LISTEN -k $SSH_KEY_PATH -a $AUTH_DIR"

# Add optional flags based on environment variables
if [ "$BAN_IP" = "true" ]; then
  ARGS="$ARGS -I -D $BAN_DURATION"
fi

if [ "$LOG_REJECTED" = "true" ]; then
  ARGS="$ARGS -R"
fi

# Add any whitelist domains
if [ -n "$WHITELIST_DOMAINS" ]; then
  for domain in $WHITELIST_DOMAINS; do
    ARGS="$ARGS -w $domain"
  done
fi

# Add any blacklist domains
if [ -n "$BLACKLIST_DOMAINS" ]; then
  for domain in $BLACKLIST_DOMAINS; do
    ARGS="$ARGS -b $domain"
  done
fi

# Execute with any additional arguments passed to the container
exec /app/mcrouter $ARGS "$@"
