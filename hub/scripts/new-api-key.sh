#!/usr/bin/env bash
# Crée une clé API Headscale (pour l'interface hub.DOMAINE/admin), valide 90 jours par défaut.
cd "$(dirname "$0")/.." && docker compose exec headscale headscale apikeys create --expiration "${1:-90d}"
