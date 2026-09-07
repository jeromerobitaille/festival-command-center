# Festival Command Center — Architecture proposée

Contexte : ~10 écrans LED répartis dans la ville. Chaque écran = 1 laptop + 1 processeur Brompton Tessera
branché en Ethernet direct sur le laptop. Le laptop est souvent sur LTE/WiFi de venue (donc derrière CGNAT).

## Objectifs
1. Un **agent** (« aggrégateur ») à installer en 2 minutes sur chaque laptop :
   - monte un tunnel vers le hub (remplace ZeroTier),
   - expose le processeur Tessera (port 37564 + HTTP IP Control) à travers le tunnel,
   - expose le bureau du laptop (remote desktop),
   - envoie un heartbeat (nom d'écran, IP, état du processeur, CPU/RAM/temp, uptime),
   - configurable via un seul fichier `agent.toml`.
2. Un **hub** (VPS) : plan de contrôle du tunnel + relais + plateforme web (liste des écrans, état, bouton « Prendre le contrôle »).

## Décision 1 — Tunnel : ne pas réécrire un VPN, embarquer WireGuard via tsnet + Headscale

| Option | Latence | NAT traversal (CGNAT LTE) | Effort | Verdict |
|---|---|---|---|---|
| ZeroTier (actuel) | bonne, userspace | oui (moons/roots) | nul | fonctionne, mais dépendance à leur infra et pas embarquable |
| WireGuard pur (hub-and-spoke) | excellente (kernel Linux côté hub) | non : tout passe par le VPS | faible | simple mais pas de P2P, config manuelle des clés |
| **Tailscale `tsnet` + Headscale self-hosted** | excellente (WireGuard) | oui : STUN + DERP self-hosted | moyen | **recommandé** |
| Tunnel maison (QUIC/UDP custom) | au mieux égale à WireGuard | à réimplémenter | très élevé | non : on réinventerait WireGuard + NAT traversal + crypto |

Pourquoi tsnet : c'est WireGuard (le plus rapide qui existe, chiffrement ChaCha20 très léger) avec
tout le plan de contrôle Tailscale, **embarqué dans notre binaire Go**. Aucun driver, aucun service tiers
à installer, pas de privilège admin pour le tunnel. Le plan de contrôle est **Headscale** (open source, sur
notre VPS), avec notre propre relais **DERP** à Montréal pour les cas CGNAT. On a donc « notre propre
système » sans en écrire la partie dangereuse.

Note : tsnet n'est pas un routeur de sous-réseau (pas de TUN). On ne « route » donc pas le LAN du
processeur : l'agent fait du **proxy TCP/UDP explicite** (listener sur l'IP tailnet → processeur).
C'est exactement le port-forwarding demandé, et c'est plus prévisible qu'un subnet route.

## Décision 2 — Remote desktop : RustDesk self-hosted (phase 1), Guacamole en option web (phase 2)

- **RustDesk** : open source, codec vidéo perf, client web hébergeable sur notre domaine, serveur ID/relais
  (`hbbs`/`hbbr`) sur le VPS. L'agent installe RustDesk en mode service avec notre serveur pré-configuré et un
  mot de passe permanent par écran. Un opérateur clique dans la plateforme web → ouverture du client web
  RustDesk directement sur l'écran voulu.
- **Apache Guacamole** (alternative/complement) : VNC/RDP rendu en HTML5 via `guacd`, entièrement dans notre
  page web, passe par le tunnel tailnet. Plus intégré, un peu plus de latence que RustDesk.
- Écrire notre propre remote desktop (WebRTC + capture) : non, pas pour 10 écrans.

## Décision 3 — Langage et packaging

- **Agent en Go** : 1 binaire statique Windows/macOS/Linux, tsnet natif, `kardianos/service` pour installer
  le service au boot, config TOML. Zéro dépendance runtime.
- **Installation** : `agent.exe install` (copie le binaire, enregistre le service, démarre). Fichier
  `agent.toml` à côté. Une clé de pré-auth Headscale dans la config suffit pour joindre le réseau.
- **Hub** : Docker Compose sur **un seul VPS OVH (Beauharnois)** : Caddy (TLS), Headscale avec son
  **DERP intégré** (relais + STUN, `verify_clients`), RustDesk hbbs/hbbr, dashboard. Pas de Postgres :
  SQLite pour Headscale, un fichier JSON pour le dashboard. Voir `hub/README.md`.
- **Portail** (Go + SQLite, `hub/dashboard`) : comptes `admin`/`user`, projets multi-tenant avec clé
  d'inscription, approbation des écrans, configuration des agents **à distance** (forwards, processeur,
  mises à jour) appliquée à chaud, tableaux de bord en grille de widgets, actions HTTP (hub ou agent),
  automatisations cron. Le portail crée lui-même les clés Headscale via l'API à l'approbation.

## Cycle de vie d'un écran

1. `agent.toml` local : URL du portail + clé du projet + identifiant de l'écran.
2. L'agent s'inscrit (`/api/agent/enroll`), reçoit un jeton d'appareil et attend (`/api/agent/status`).
3. Un admin approuve dans le portail → clé Headscale à usage unique livrée à l'agent → tsnet rejoint le réseau.
4. L'agent charge sa configuration (`/api/agent/config`), démarre les forwards, envoie ses heartbeats ;
   la réponse du heartbeat porte la version de config et les commandes à exécuter (http_request, probe,
   restart, update). Un changement dans le portail est appliqué au heartbeat suivant.

## Fichier de config local (minimal)

```toml
[portal]
url         = "https://panel.veam.ca"
project_key = "fcc_..."          # Configuration du projet dans le portail

[screen]
id   = "ecran-03"
name = "Place des Festivals — Nord"

[processor]                       # proposition initiale, ensuite gérée dans le portail
ip = "192.168.0.10"
```

## Topologie

```
Opérateur (navigateur / Tessera Remote)
        │  HTTPS / tailnet
        ▼
   VPS hub : Caddy + Headscale (DERP intégré) + RustDesk hbbs/hbbr + dashboard
        │  WireGuard (P2P quand possible, sinon relais DERP)
        ▼
 Laptop écran N : agent (tsnet) ──proxy 37564──▶ Tessera (Ethernet direct)
                  └─ RustDesk service ──▶ bureau du laptop
                  └─ heartbeat → dashboard
```

## Phases

1. **Agent MVP** : tsnet + proxy TCP configurable + heartbeat + service Windows. Test avec 1 laptop + 1 Tessera.
2. **Hub** : Docker Compose (Headscale + DERP intégré, RustDesk server, Caddy) + dashboard. **Livré, à déployer.**
3. **Remote desktop depuis le web** : aujourd'hui le bouton ouvre le client RustDesk installé ;
   ensuite héberger le client web RustDesk derrière Caddy pour rester dans le navigateur.
4. **Confort** : mise à jour auto de l'agent, alertes (écran hors ligne > 2 min), logs centralisés,
   sonde HTTP du processeur (IP Control) pour afficher l'état Tessera dans le dashboard.

## Points à valider
- OS des laptops (Windows présumé). Tessera Remote existe Windows/macOS.
- Firmware Tessera ≥ 3.4 (Direct Connect + IP Control HTTP) ; sinon seul le port 37564 est utile.
- Connectivité des sites : LTE (CGNAT → relais DERP) vs WiFi venue (souvent P2P direct possible).
- Qui opère : combien d'utilisateurs du dashboard, besoin d'auth SSO ou simple login suffit.
