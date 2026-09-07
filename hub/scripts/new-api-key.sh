#!/usr/bin/env bash
# Crée une clé API Headscale (pour l'interface hub.DOMAINE/admin), valide 90 jours par défaut.
cd "$(dirname "$0")/.." && docker compose ${COMPOSE_ARGS:-} exec headscale headscale apikeys create --expiration "${1:-90d}"
