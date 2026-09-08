package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// Headscale injoignable ne doit plus retenir la réponse : la base est écrite d'abord,
// et le retrait des nœuds est borné. C'est ce qui figeait le portail.
func TestRevokeBorneParLeTimeout(t *testing.T) {
	// Serveur qui accepte la connexion et ne répond jamais.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
			select {} // jamais de réponse
		}
	}()

	h := &Headscale{URL: "http://" + ln.Addr().String(), APIKey: "x", User: "festival",
		Client: &http.Client{Timeout: 15 * time.Second}}
	if !h.enabled() {
		t.Fatal("client headscale desactive")
	}

	ctx, cancel := context.WithTimeout(context.Background(), hsTimeout)
	defer cancel()
	start := time.Now()
	err = h.DeleteNodes(ctx, "ecran-01", "ecran-01-probe")
	d := time.Since(start)

	if err == nil {
		t.Error("un Headscale muet aurait du produire une erreur")
	}
	if d > hsTimeout+2*time.Second {
		t.Errorf("appel non borne : %.1f s (limite %.0f s)", d.Seconds(), hsTimeout.Seconds())
	}
	t.Logf("Headscale muet : erreur en %.1f s (borne %.0f s, timeout client 15 s)", d.Seconds(), hsTimeout.Seconds())
}

func TestHsTimeoutInferieurAuClient(t *testing.T) {
	if hsTimeout >= 15*time.Second {
		t.Errorf("hsTimeout (%s) doit rester sous le timeout du client HTTP", hsTimeout)
	}
}
