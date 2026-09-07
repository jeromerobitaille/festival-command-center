#!/usr/bin/env bash
# Arrête la pile locale. Ajouter -v pour effacer aussi les volumes (base Headscale, clés RustDesk).
cd "$(dirname "$0")/.." && docker compose --env-file local.env -f docker-compose.yml -f docker-compose.local.yml down "$@"
