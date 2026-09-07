#!/usr/bin/env bash
# Crée une clé API Headscale pour le portail (création des clés de pré-auth à l'approbation d'un écran),
# l'écrit dans .env (HEADSCALE_API_KEY) et redémarre le portail.
set -euo pipefail
cd "$(dirname "$0")/.."
KEY=$(docker compose ${COMPOSE_ARGS:-} exec headscale headscale apikeys create --expiration "${1:-365d}" | tail -1 | tr -d '\r')
grep -q '^HEADSCALE_API_KEY=' .env && sed -i.bak "s|^HEADSCALE_API_KEY=.*|HEADSCALE_API_KEY=$KEY|" .env || echo "HEADSCALE_API_KEY=$KEY" >> .env
rm -f .env.bak
docker compose ${COMPOSE_ARGS:-} up -d dashboard >/dev/null
echo "clé API Headscale du portail renouvelée (valide ${1:-365d}), portail redémarré"
