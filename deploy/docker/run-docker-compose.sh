#!/usr/bin/env bash
# Launch workshop.ino via Docker Compose.
# Runs in the foreground; press Ctrl+C to stop the stack.
#
#   ./run-docker-compose.sh
#
set -euo pipefail

# Resolve the compose file from current working directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# HOST_UID/HOST_GID are captured so files created are owned by you
HOST_UID="$(id -u)" HOST_GID="$(id -g)" docker compose up
