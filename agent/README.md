# agent — Festival Command Center

Binaire unique, installé sur chaque appareil. Il :

- s'inscrit auprès du **portail** avec la clé du projet et attend qu'un administrateur l'approuve ;
- reçoit alors sa clé Headscale et rejoint le réseau WireGuard via **tsnet** (aucun logiciel tiers) ;
- applique la configuration du portail (nom, sous-appareil, forwards TCP/UDP, mises à jour) et la recharge
  à chaud quand elle change ;
- sonde le sous-appareil et remonte un **heartbeat** : état, forwards, chemin (direct ou relais DERP) et
  latence vers chaque pair, CPU/RAM/uptime, identifiant RustDesk ;
- exécute les commandes envoyées depuis le portail (requête HTTP locale, sonde, redémarrage, mise à jour) ;
- s'installe comme service système (Windows, macOS, Linux) et redémarre seul en cas de perte réseau.

## Installation sur un appareil Windows

1. Télécharger `FestivalCommandCenter-Agent-Setup-<version>.exe` de la dernière release GitHub et le lancer
   (droits administrateur demandés). L'installateur copie l'agent dans *Program Files*, enregistre le
   service Windows, met l'icône dans la barre des tâches au démarrage de session et ouvre la fenêtre de
   configuration.
2. Dans la fenêtre : coller la clé du projet (portail, onglet Configuration du projet), donner un
   identifiant et un nom à l'appareil, éventuellement un premier sous-appareil, puis « Enregistrer et
   s'inscrire ».
3. Approuver l'appareil dans le portail (onglet Appareils). Icône rouge = arrêté ou sans clé, orange = en
   attente / connexion / un sous-appareil injoignable, verte = en ligne.

Le menu de l'icône donne le statut, l'IP privée, l'état des sous-appareils, et ouvre la fenêtre
« Configuration… » (contrôles Windows natifs, sans navigateur). Un panneau web équivalent reste
disponible sur http://127.0.0.1:47632 pour les cas avancés. Désinstallation par *Ajout/Suppression de
programmes*. Installation manuelle possible avec le zip `festival-agent_<version>_windows_amd64.zip`.

Sans icône (Linux, macOS, ou en ligne de commande) : `agent check`, `agent install`, `agent panel`.

Autres commandes : `run` (premier plan, `-v` pour les logs du tunnel), `status`, `stop`, `start`,
`restart`, `uninstall`.

**Diagnostic depuis n'importe quel poste** ayant un `agent.toml` valide :

```
agent probe 100.64.0.5:37564
```

rejoint le réseau avec un noeud `<id>-probe`, affiche le chemin (direct ou relais) et la latence vers
chaque pair, puis tente une connexion TCP à l'adresse donnée **à travers le tunnel**. C'est le test à
faire quand Tessera Remote n'arrive pas à joindre un appareil. Le fichier de config par défaut est `agent.toml` à côté du binaire ;
`-config chemin` pour en utiliser un autre. L'état du noeud (clés WireGuard) vit dans `state/` à
côté de la config : le supprimer force une ré-inscription auprès de Headscale.

## Côté opérateur

- **Tessera Remote** : Direct Connect vers `<ip tailnet de l'appareil>:37564`.
- **Contrôle HTTP Tessera** : `http://<ip tailnet>:8080/` si le forward `tessera-http` est actif.
- **Remote desktop** : RustDesk sur le laptop, pointé vers le serveur du hub (phase 2).

## Releases et mise à jour automatique

- Un tag `vX.Y.Z` poussé sur GitHub déclenche `.github/workflows/release.yml` : GoReleaser compile
  Windows/Linux/macOS, publie les binaires en release et signe `checksums.txt` (ed25519) avec le secret
  `RELEASE_SIGNING_KEY` (généré par `go run ./cmd/relsign keygen`). La clé publique correspondante est
  embarquée dans l'agent (`internal/update/public_key.txt`).
- Chaque agent vérifie la dernière release toutes les `update.check_hours` heures (1 à 3 minutes après le
  démarrage, puis à intervalle). S'il existe une version plus récente pour sa plateforme : téléchargement,
  vérification de la signature de `checksums.txt` puis du SHA-256 du binaire, remplacement du fichier
  (l'ancien reste en `agent.exe.old` jusqu'au prochain démarrage) et redémarrage du service.
- `agent update` force une vérification immédiate ; `-force` installe même depuis un build `dev`.
- Désactiver sur un appareil : `[update] enabled = false`.

Publier une version :

```
git tag v0.3.0 && git push origin v0.3.0
```

## Build

```
go build -ldflags "-X main.version=0.1.0" -o dist/agent ./cmd/agent
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=0.1.0" -o dist/agent.exe ./cmd/agent
```

## Format du heartbeat (POST `{api_url}/heartbeat`, JSON, `Authorization: Bearer {api_token}`)

```json
{
  "screen_id": "ecran-01", "screen_name": "…", "agent_version": "0.1.0",
  "timestamp": "…", "tailnet_ip": "100.64.0.5",
  "processor": {"target": "192.168.0.10:37564", "reachable": true, "rtt_ms": 0.4},
  "forwards": [{"name": "tessera-remote", "proto": "tcp", "listen": 37564, "target": "…",
                "active_conns": 1, "total_conns": 3, "bytes_in": 0, "bytes_out": 0}],
  "peers": [{"hostname": "regie", "ip": "100.64.0.2", "online": true,
             "direct": true, "endpoint": "203.0.113.4:41641", "latency_ms": 12.3}],
  "system": {"hostname": "…", "os": "windows", "uptime_seconds": 0, "cpu_percent": 0, "mem_percent": 0},
  "remote_desktop": {"provider": "rustdesk", "id": "123456789"}
}
```
