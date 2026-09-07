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

if [ -z "${COOKIE_SECRET:-}" ]; then
  COOKIE_SECRET=$(openssl rand -hex 16)
  grep -q '^COOKIE_SECRET=' .env && sed -i.bak "s|^COOKIE_SECRET=.*|COOKIE_SECRET=$COOKIE_SECRET|" .env || echo "COOKIE_SECRET=$COOKIE_SECRET" >> .env
  rm -f .env.bak; echo "→ COOKIE_SECRET (Headplane) généré et écrit dans .env"
fi

if [ -z "${ADMIN_PASSWORD:-}" ]; then
  ADMIN_PASSWORD=$(openssl rand -base64 18 | tr -d '/+=' | cut -c1-20)
  grep -q '^ADMIN_PASSWORD=' .env && sed -i.bak "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$ADMIN_PASSWORD|" .env || echo "ADMIN_PASSWORD=$ADMIN_PASSWORD" >> .env
  rm -f .env.bak; echo "→ ADMIN_PASSWORD (premier compte du portail) généré et écrit dans .env"
fi

sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|__PUBLIC_IPV4__|${PUBLIC_IPV4:-}|g" \
  headscale/config.yaml.tmpl > headscale/config.yaml
[ -n "${PUBLIC_IPV4:-}" ] || sed -i.bak '/ipv4: ""/d' headscale/config.yaml && rm -f headscale/config.yaml.bak
sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|# __ACME_EMAIL_LINE__|$ACME_EMAIL_LINE|g" \
  caddy/Caddyfile.tmpl > caddy/Caddyfile
sed -e "s|__DOMAIN__|$DOMAIN|g" -e "s|__COOKIE_SECRET__|$COOKIE_SECRET|g" \
  headplane/config.yaml.tmpl > headplane/config.yaml

cat <<MSG

Configuration rendue. Prochaines étapes :
  docker compose up -d --build
  ./scripts/portal-headscale-key.sh   # donne au portail une clé API Headscale (approbation des écrans)
  ./scripts/rustdesk-key.sh           # clé publique RustDesk pour les clients
  ./scripts/new-api-key.sh            # clé API pour se connecter à https://hub.$DOMAIN/admin (Headplane)

Portail : https://panel.$DOMAIN — compte ${ADMIN_USER:-admin} / $ADMIN_PASSWORD
Créer un projet dans le portail, puis copier sa clé dans le agent.toml des laptops.
MSG
