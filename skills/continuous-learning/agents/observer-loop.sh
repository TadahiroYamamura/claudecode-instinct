#!/bin/bash
set -e

echo $$ > "${CLAUDE_PROJECT_DIR}/.observer.pid"

while true; do
  sleep 3600 & wait
done
