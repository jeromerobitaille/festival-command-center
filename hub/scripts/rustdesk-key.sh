#!/usr/bin/env bash
# Affiche la clé publique du serveur RustDesk, à saisir dans les clients (ID server = DOMAIN, Key = cette valeur).
# L'image rustdesk-server n'a pas de shell : on lit le volume avec une image alpine.
set -euo pipefail
cd "$(dirname "$0")/.."
VOL=$(docker volume ls -q | grep -m1 'rustdesk-data$')
docker run --rm -v "$VOL:/r:ro" alpine:3.21 cat /r/id_ed25519.pub; echo
