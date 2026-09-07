// Package probe vérifie que le processeur Tessera répond sur son port de contrôle.
package probe

import (
	"context"
	"fmt"
	"net"
	"time"
)

type Result struct {
	Target    string  `json:"target"`
	Reachable bool    `json:"reachable"`
	RTTMS     float64 `json:"rtt_ms"`
	Error     string  `json:"error,omitempty"`
}

// TCP tente une connexion TCP et mesure le temps de handshake.
func TCP(ctx context.Context, host string, port int) Result {
	target := net.JoinHostPort(host, fmt.Sprint(port))
	r := Result{Target: target}
	d := net.Dialer{Timeout: 2 * time.Second}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	c.Close()
	r.Reachable = true
	r.RTTMS = float64(time.Since(start).Microseconds()) / 1000
	return r
}
