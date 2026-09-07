#!/usr/bin/env bash
# Crée (si besoin) l'utilisateur Headscale du festival et une clé de pré-auth réutilisable.
set -euo pipefail
cd "$(dirname "$0")/.."
USER_NAME=${1:-festival}
DAYS=${2:-90}
docker compose ${COMPOSE_ARGS:-} exec headscale headscale users list -o json 2>/dev/null | grep -q "\"name\": *\"$USER_NAME\"" \
  || docker compose ${COMPOSE_ARGS:-} exec headscale headscale users create "$USER_NAME"
USER_ID=$(docker compose ${COMPOSE_ARGS:-} exec headscale headscale users list -o json | python3 -c "import json,sys;print([u['id'] for u in json.load(sys.stdin) if u['name']=='$USER_NAME'][0])")
echo "Clé de pré-auth (utilisateur $USER_NAME, valide ${DAYS}j, réutilisable) — à mettre dans agent.toml hub.auth_key :"
docker compose ${COMPOSE_ARGS:-} exec headscale headscale preauthkeys create --user "$USER_ID" --reusable --expiration "${DAYS}d"
