#!/usr/bin/env bash
# Liste les noeuds connus de Headscale (écrans + régie) avec leur IP tailnet.
cd "$(dirname "$0")/.." && docker compose ${COMPOSE_ARGS:-} exec headscale headscale nodes list
