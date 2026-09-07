#!/usr/bin/env bash
# Démarre la pile complète en local, sans TLS. Arrêt : ./scripts/local-down.sh
cd "$(dirname "$0")/.." && docker compose --env-file local.env -f docker-compose.yml -f docker-compose.local.yml up -d --build "$@"
