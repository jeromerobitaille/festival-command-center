# Festival Command Center

Supervision et contrôle à distance des écrans LED du festival (laptop + processeur Brompton Tessera par écran).

- [ARCHITECTURE.md](ARCHITECTURE.md) — décisions techniques (tsnet + Headscale, RustDesk, Go).
- [agent/](agent/) — l'agent installé sur chaque laptop (phase 1, fonctionnel).
- [hub/](hub/) — le serveur : Headscale avec DERP intégré, RustDesk, Caddy et dashboard, en Docker Compose sur un VPS OVH (phase 2, livré).
