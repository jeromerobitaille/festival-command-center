// Package tunnel encapsule un noeud tsnet (WireGuard + plan de contrôle Headscale).
package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"time"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

type Options struct {
	Hostname   string
	ControlURL string
	AuthKey    string
	StateDir   string
	Verbose    bool
	Ephemeral  bool // noeud non persistant (état non conservé)
}

type Tunnel struct {
	srv *tsnet.Server
}

// PeerPath décrit comment on joint un pair : direct ou relayé, et sa latence.
type PeerPath struct {
	Hostname  string  `json:"hostname"`
	IP        string  `json:"ip"`
	Online    bool    `json:"online"`
	Direct    bool    `json:"direct"`
	Endpoint  string  `json:"endpoint,omitempty"` // adresse directe si Direct
	Relay     string  `json:"relay,omitempty"`    // code région DERP sinon
	LatencyMS float64 `json:"latency_ms"`
	PingError string  `json:"ping_error,omitempty"`
}

func Start(ctx context.Context, o Options) (*Tunnel, error) {
	if err := os.MkdirAll(o.StateDir, 0o700); err != nil {
		return nil, fmt.Errorf("state dir : %w", err)
	}
	srv := &tsnet.Server{
		Dir:        o.StateDir,
		Hostname:   o.Hostname,
		ControlURL: o.ControlURL,
		AuthKey:    o.AuthKey,
		Ephemeral:  o.Ephemeral,
	}
	if o.Verbose {
		srv.Logf = log.Printf
	} else {
		srv.Logf = func(string, ...any) {}
	}
	srv.UserLogf = log.Printf // messages utilisateur (état, login) toujours journalisés
	upCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if _, err := srv.Up(upCtx); err != nil {
		srv.Close()
		return nil, fmt.Errorf("connexion au réseau : %w", err)
	}
	return &Tunnel{srv: srv}, nil
}

func (t *Tunnel) Close() error { return t.srv.Close() }

// IPv4 retourne l'adresse tailnet v4 du noeud.
func (t *Tunnel) IPv4() netip.Addr {
	ip4, _ := t.srv.TailscaleIPs()
	return ip4
}

// Dial ouvre une connexion à travers le tunnel (vers un autre noeud du tailnet).
func (t *Tunnel) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	return t.srv.Dial(ctx, network, addr)
}

func (t *Tunnel) Listen(network, addr string) (net.Listener, error) {
	return t.srv.Listen(network, addr)
}

func (t *Tunnel) ListenPacket(network, addr string) (net.PacketConn, error) {
	return t.srv.ListenPacket(network, addr)
}

func (t *Tunnel) Status(ctx context.Context) (*ipnstate.Status, error) {
	lc, err := t.srv.LocalClient()
	if err != nil {
		return nil, err
	}
	return lc.Status(ctx)
}

// PeerPaths mesure le chemin vers chaque pair en ligne (ping disco, sans passer par la pile IP).
func (t *Tunnel) PeerPaths(ctx context.Context) ([]PeerPath, error) {
	lc, err := t.srv.LocalClient()
	if err != nil {
		return nil, err
	}
	st, err := lc.Status(ctx)
	if err != nil {
		return nil, err
	}
	var out []PeerPath
	for _, p := range st.Peer {
		if len(p.TailscaleIPs) == 0 {
			continue
		}
		pp := PeerPath{
			Hostname: p.HostName,
			IP:       p.TailscaleIPs[0].String(),
			Online:   p.Online,
			Direct:   p.CurAddr != "",
			Endpoint: p.CurAddr,
			Relay:    p.Relay,
		}
		if p.Online {
			pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			res, err := lc.Ping(pctx, p.TailscaleIPs[0], tailcfg.PingDisco)
			cancel()
			if err != nil {
				pp.PingError = err.Error()
			} else if res.Err != "" {
				pp.PingError = res.Err
			} else {
				pp.LatencyMS = res.LatencySeconds * 1000
				if res.Endpoint != "" {
					pp.Direct = true
					pp.Endpoint = res.Endpoint
				} else if res.DERPRegionCode != "" {
					pp.Direct = false
					pp.Relay = res.DERPRegionCode
				}
			}
		}
		out = append(out, pp)
	}
	return out, nil
}
