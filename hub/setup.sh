#!/usr/bin/env bash
# Prépare le hub : génère le jeton, hache le mot de passe du dashboard, rend les fichiers de config.
set -euo pipefail
cd "$(dirname "$0")"

[ -f .env ] || { cp .env.example .env; echo "→ .env créé depuis .env.example : édite DOMAIN, ACME_EMAIL, PUBLIC_IPV4 puis relance ./setup.sh"; exit 1; }
set -a; source .env; set +a

: "${DOMAIN:?DOMAIN manquant dans .env}"
ACME_EMAIL_LINE="email ${ACME_EMAIL:-}"
[ -n "${ACME_EMAIL:-}" ] || ACME_EMAIL_LINE="# pas d'adresse ACME_EMAIL : Let's Encrypt fonctionne sans, mais n'enverra pas d'avis d'expiration"
[ "$DOMAIN" != "festival.example.com" ] || { echo "DOMAIN est encore la valeur d'exemple"; exit 1; }

if [ -z "${HEARTBEAT_TOKEN:-}" ]; then
  HEARTBEAT_TOKEN=$(openssl rand -hex 24)
  sed -i.bak "s|^HEARTBEAT_TOKEN=.*|HEARTBEAT_TOKEN=$HEARTBEAT_TOKEN|" .env && rm -f .env.bak
  echo "→ HEARTBEAT_TOKEN généré et écrit dans .env"
fi

if [ -z "${COOKIE_SECRET:-}" ]; then
  COOKIE_SECRET=$(openssl rand -hex 16)
  grep -q '^COOKIE_SECRET=' .env && sed -i.bak "s|^COOKIE_SECRET=.*|COOKIE_SECRET=$COOKIE_SECRET|" .env || echo "COOKIE_SECRET=$COOKIE_SECRET" >> .env
  rm -f .env.bak; echo "→ COOKIE_SECRET (Headplane) généré et écrit dans .env"
fi

if [ -n "${PANEL_PASSWORD:-}" ]; then
  PW="$PANEL_PASSWORD"
else
  read -r -s -p "Mot de passe du dashboard pour ${PANEL_USER:-admin} : " PW; echo
fi
[ -n "$PW" ] || { echo "mot de passe vide"; exit 1; }
PANEL_HASH=$(docker run --rm caddy:2.11-alpine caddy hash-password --plaintext "$PW")

sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|__PUBLIC_IPV4__|${PUBLIC_IPV4:-}|g" \
  headscale/config.yaml.tmpl > headscale/config.yaml
[ -n "${PUBLIC_IPV4:-}" ] || sed -i.bak '/ipv4: ""/d' headscale/config.yaml && rm -f headscale/config.yaml.bak
sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|# __ACME_EMAIL_LINE__|$ACME_EMAIL_LINE|g" \
    -e "s|__PANEL_USER__|${PANEL_USER:-admin}|g" -e "s|__PANEL_HASH__|$PANEL_HASH|g" \
  caddy/Caddyfile.tmpl > caddy/Caddyfile
sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|__COOKIE_SECRET__|$COOKIE_SECRET|g" \
  headplane/config.yaml.tmpl > headplane/config.yaml

cat <<MSG

Configuration rendue. Prochaines étapes :
  docker compose up -d --build
  ./scripts/new-preauth-key.sh        # clé pour agent.toml (hub.auth_key)
  ./scripts/rustdesk-key.sh           # clé publique RustDesk pour les clients
  ./scripts/new-api-key.sh            # clé API pour se connecter à https://hub.$DOMAIN/admin

Valeurs pour agent.toml :
  [hub]
  control_url = "https://hub.$DOMAIN"
  api_url     = "https://panel.$DOMAIN/api"
  api_token   = "$HEARTBEAT_TOKEN"
MSG
