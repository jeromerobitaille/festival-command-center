# hub — Festival Command Center

> Déployé sur le VPS OVH `51.161.82.197` (`ssh debian@…`, dossier `~/hub`) : https://hub.veam.ca (Headscale, `/admin` = Headplane) et https://panel.veam.ca (dashboard).

Tout le côté serveur, sur **un seul VPS** (OVH VPS-2 à Beauharnois suffit), en Docker Compose :

| Service | Rôle | Exposé |
|---|---|---|
| `caddy` | TLS Let's Encrypt et reverse proxy | 80, 443 (tcp+udp) |
| `headscale` | plan de contrôle du réseau WireGuard **et** relais DERP intégré avec STUN | via `hub.DOMAINE`, 3478/udp |
| `hbbs` / `hbbr` | serveur d'identifiants et relais RustDesk | 21115-21119/tcp, 21116/udp |
| `dashboard` | le **portail** : comptes et rôles, projets, approbation et configuration à distance des appareils, tableaux de bord, actions HTTP, automatisations | via `panel.DOMAINE` |
| `headplane` | interface d'administration Headscale (noeuds, utilisateurs, clés, ACL) | via `hub.DOMAINE/admin` |

Le DERP intégré de Headscale est déclaré région 900 « mtl ». Les relais publics de Tailscale restent en
secours : chaque client mesure la latence et choisit le plus proche, donc le nôtre. Seuls les noeuds
enregistrés dans Headscale peuvent utiliser le relais (`verify_clients`).

## Mise en place (une fois)

1. **DNS** : deux enregistrements A vers l'IP du VPS : `hub.DOMAINE` et `panel.DOMAINE`.
2. **VPS** : Ubuntu 24.04, Docker installé (`curl -fsSL https://get.docker.com | sh`). Pare-feu :

   ```
   ufw allow 22/tcp 80/tcp 443/tcp 443/udp 3478/udp 21115:21119/tcp 21116/udp && ufw enable
   ```

3. Copier ce dossier `hub/` sur le VPS, puis :

   ```
   cp .env.example .env      # éditer DOMAIN, ACME_EMAIL, PUBLIC_IPV4
   ./setup.sh                # génère le jeton, demande le mot de passe du dashboard, rend les configs
   docker compose up -d --build
   ./scripts/new-preauth-key.sh    # clé pour agent.toml (hub.auth_key), valide 90 jours
   ./scripts/rustdesk-key.sh       # clé publique RustDesk pour les clients
   ./scripts/new-api-key.sh        # clé API pour se connecter à hub.DOMAINE/admin (Headplane)
   ```

4. **Régie** : installer le client Tailscale officiel sur le poste de l'opérateur et le pointer sur
   Headscale :

   ```
   tailscale up --login-server https://hub.DOMAINE --auth-key <clé de pré-auth>
   ```

   Ensuite Tessera Remote se connecte en Direct Connect à `<ip tailnet de l'appareil>:37564`, adresse
   affichée dans le dashboard.

5. **RustDesk** sur les laptops et sur le poste de la régie : dans Paramètres → Réseau, `ID server` et
   `Relay server` = `DOMAINE`, `Key` = sortie de `scripts/rustdesk-key.sh`. Sur les laptops, activer un
   mot de passe permanent. Le bouton « Bureau à distance » du dashboard ouvre `rustdesk://connection/new/<id>`
   dans le client RustDesk de l'opérateur.

## Portail

- **Comptes** : le premier `admin` est créé au premier démarrage (mot de passe `ADMIN_PASSWORD` du `.env`,
  sinon généré et affiché dans `docker compose logs dashboard`). Deux rôles : `admin` (tout) et `user`
  (voit les projets dont il est membre, lance les actions, ne modifie rien). Gestion dans *Utilisateurs*.
- **Projets** : chaque projet a sa **clé** (onglet *Configuration*), à mettre dans le `agent.toml` des
  laptops. Les membres non-admin y sont cochés au même endroit.
- **Appareils** : un agent qui se connecte avec la clé apparaît *en attente*. À l'approbation, le portail
  crée une clé Headscale via l'API (d'où `./scripts/portal-headscale-key.sh`) et l'agent rejoint le réseau.
  La configuration de chaque appareil (nom, sous-appareils, forwards, mises à jour) se modifie dans le portail et
  l'agent l'applique au prochain heartbeat. Boutons : sonder, redémarrer, mettre à jour, révoquer.
- **Actions** : requêtes HTTP exécutées par le hub (API externe) ou par l'agent d'un appareil (réseau local,
  ex. l'API HTTP d'un processeur LED). **Automatisations** : horaire cron dans le fuseau du projet → action.
- **Tableau de bord** : grille de widgets par projet (statut des appareils, détail d'un appareil, bouton
  d'action, automatisations, note), déplaçables et redimensionnables en mode *Modifier*.
- Données : SQLite dans le volume `dashboard-data` (`portal.sqlite`). À sauvegarder avec les autres volumes.

## Agents

Dans `agent.toml` de chaque laptop : `[portal] url = "https://panel.DOMAINE"` et `project_key` copiée
depuis la Configuration du projet. Voir `agent/README.md`.

## Essai en local (sans VPS)

Avec Docker sur un poste de travail :

```
./scripts/local-up.sh          # pile complète en HTTP : http://hub.localhost, /admin, http://panel.localhost
# portail : http://panel.localhost, admin / admin-local ; donner au portail une clé API Headscale :
COMPOSE_ARGS="--env-file local.env -f docker-compose.yml -f docker-compose.local.yml" ./scripts/portal-headscale-key.sh
# puis un agent.toml avec [portal] url = "http://127.0.0.1:8090" et la clé du projet créé dans le portail
./scripts/local-down.sh -v     # arrêt et nettoyage
```

Les fichiers `docker-compose.local.yml`, `caddy/Caddyfile.local`, `headscale/config.local.yaml` et
`local.env` ne servent qu'à ça.

## Exploitation

- `./scripts/nodes.sh` : liste des noeuds et IP tailnet vues par Headscale.
- `docker compose logs -f headscale` / `dashboard` / `caddy`.
- L'état du dashboard est dans le volume `dashboard-data` (`state.json`) ; l'historique n'est pas conservé
  pour l'instant, seulement le dernier heartbeat par écran.
- Sauvegarde : volumes `headscale-data` (base et clé Noise) et `rustdesk-data` (clés RustDesk). Perdre
  la clé RustDesk oblige à reconfigurer tous les clients.
- Le dashboard répond sur `/api/screens` en JSON (même basic auth) pour des intégrations futures.
