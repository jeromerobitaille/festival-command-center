#!/usr/bin/env bash
# Liste les noeuds connus de Headscale (écrans + régie) avec leur IP tailnet.
cd "$(dirname "$0")/.." && docker compose exec headscale headscale nodes list
