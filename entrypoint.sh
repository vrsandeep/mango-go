#!/bin/sh

# Set default PUID and PGID
PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "Starting with UID : $PUID, GID: $PGID"

# Change the group ID of the existing 'mango' group
groupmod -o -g "$PGID" mango
# Change the user ID of the existing 'mango' user
usermod -o -u "$PUID" mango

# Set ownership of the app directory
chown -R mango:mango /app

# Execute the original CMD from the Dockerfile
exec /mango-go
