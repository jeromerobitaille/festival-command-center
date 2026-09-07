// Package forward relaie des connexions reçues sur le tailnet vers une cible locale
// (typiquement le processeur Tessera branché en direct sur le laptop).
package forward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Listener interface {
	Listen(network, addr string) (net.Listener, error)
	ListenPacket(network, addr string) (net.PacketConn, error)
}

type Rule struct {
	Name   string
	Proto  string
	Listen int
	Target string
}

type Stats struct {
	Name        string `json:"name"`
	Proto       string `json:"proto"`
	Listen      int    `json:"listen"`
	Target      string `json:"target"`
	Active      int64  `json:"active_conns"`
	Total       int64  `json:"total_conns"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
	LastError   string `json:"last_error,omitempty"`
}

type Forwarder struct {
	rule      Rule
	active    atomic.Int64
	total     atomic.Int64
	bytesIn   atomic.Int64
	bytesOut  atomic.Int64
	lastErr   atomic.Value // string
}

func (f *Forwarder) Stats() Stats {
	s := Stats{
		Name: f.rule.Name, Proto: f.rule.Proto, Listen: f.rule.Listen, Target: f.rule.Target,
		Active: f.active.Load(), Total: f.total.Load(),
		BytesIn: f.bytesIn.Load(), BytesOut: f.bytesOut.Load(),
	}
	if v, ok := f.lastErr.Load().(string); ok {
		s.LastError = v
	}
	return s
}

// StartAll démarre chaque règle et retourne les forwarders (pour les stats).
func StartAll(ctx context.Context, l Listener, rules []Rule) ([]*Forwarder, error) {
	var out []*Forwarder
	for _, r := range rules {
		f := &Forwarder{rule: r}
		var err error
		switch r.Proto {
		case "tcp":
			err = f.startTCP(ctx, l)
		case "udp":
			err = f.startUDP(ctx, l)
		default:
			err = fmt.Errorf("proto inconnu %q", r.Proto)
		}
		if err != nil {
			return out, fmt.Errorf("forward %q : %w", r.Name, err)
		}
		out = append(out, f)
	}
	return out, nil
}

func (f *Forwarder) startTCP(ctx context.Context, l Listener) error {
	ln, err := l.Listen("tcp", fmt.Sprintf(":%d", f.rule.Listen))
	if err != nil {
		return err
	}
	log.Printf("[forward] %s : tcp :%d -> %s", f.rule.Name, f.rule.Listen, f.rule.Target)
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[forward] %s : accept : %v", f.rule.Name, err)
				time.Sleep(200 * time.Millisecond)
				continue
			}
			go f.handleTCP(ctx, c)
		}
	}()
	return nil
}

func (f *Forwarder) handleTCP(ctx context.Context, src net.Conn) {
	defer src.Close()
	f.total.Add(1)
	f.active.Add(1)
	defer f.active.Add(-1)

	d := net.Dialer{Timeout: 5 * time.Second}
	dst, err := d.DialContext(ctx, "tcp", f.rule.Target)
	if err != nil {
		f.lastErr.Store(fmt.Sprintf("%s : %v", time.Now().Format(time.RFC3339), err))
		log.Printf("[forward] %s : cible injoignable %s : %v", f.rule.Name, f.rule.Target, err)
		return
	}
	defer dst.Close()
	if tc, ok := dst.(*net.TCPConn); ok {
		tc.SetNoDelay(true) // trafic de contrôle : on privilégie la latence
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(dst, src)
		f.bytesIn.Add(n)
		closeWrite(dst)
	}()
	go func() {
		defer wg.Done()
		n, _ := io.Copy(src, dst)
		f.bytesOut.Add(n)
		closeWrite(src)
	}()
	wg.Wait()
}

func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
		return
	}
	c.Close()
}

// startUDP : relais UDP simple avec table de sessions par adresse source.
func (f *Forwarder) startUDP(ctx context.Context, l Listener) error {
	pc, err := l.ListenPacket("udp", fmt.Sprintf(":%d", f.rule.Listen))
	if err != nil {
		return err
	}
	target, err := net.ResolveUDPAddr("udp", f.rule.Target)
	if err != nil {
		pc.Close()
		return err
	}
	log.Printf("[forward] %s : udp :%d -> %s", f.rule.Name, f.rule.Listen, f.rule.Target)

	type session struct {
		conn *net.UDPConn
		last atomic.Int64
	}
	var mu sync.Mutex
	sessions := map[string]*session{}

	go func() {
		<-ctx.Done()
		pc.Close()
	}()
	// Purge des sessions inactives.
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				cutoff := time.Now().Add(-2 * time.Minute).UnixNano()
				mu.Lock()
				for k, s := range sessions {
					if s.last.Load() < cutoff {
						s.conn.Close()
						delete(sessions, k)
					}
				}
				mu.Unlock()
			}
		}
	}()
	go func() {
		buf := make([]byte, 65535)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}
			key := from.String()
			mu.Lock()
			s, ok := sessions[key]
			if !ok {
				uc, err := net.DialUDP("udp", nil, target)
				if err != nil {
					mu.Unlock()
					f.lastErr.Store(fmt.Sprintf("%s : %v", time.Now().Format(time.RFC3339), err))
					continue
				}
				s = &session{conn: uc}
				sessions[key] = s
				f.total.Add(1)
				f.active.Add(1)
				go func(s *session, from net.Addr) {
					defer f.active.Add(-1)
					rb := make([]byte, 65535)
					for {
						n, err := s.conn.Read(rb)
						if err != nil {
							return
						}
						s.last.Store(time.Now().UnixNano())
						if _, err := pc.WriteTo(rb[:n], from); err != nil {
							return
						}
						f.bytesOut.Add(int64(n))
					}
				}(s, from)
			}
			mu.Unlock()
			s.last.Store(time.Now().UnixNano())
			if _, err := s.conn.Write(buf[:n]); err == nil {
				f.bytesIn.Add(int64(n))
			}
		}
	}()
	return nil
}
