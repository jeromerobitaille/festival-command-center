package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Headscale : client minimal de l'API v1 pour créer les clés de pré-auth des écrans approuvés
// et retirer les noeuds révoqués.
type Headscale struct {
	URL    string // interne, ex. http://headscale:8080
	Public string // ce que reçoivent les agents, ex. https://hub.veam.ca
	APIKey string
	User   string // utilisateur Headscale propriétaire des noeuds (créé si absent)
	Client *http.Client
}

func (h *Headscale) enabled() bool { return h != nil && h.URL != "" && h.APIKey != "" }

func (h *Headscale) do(ctx context.Context, method, path string, in, out any) error {
	var body *bytes.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.URL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var e struct{ Message string `json:"message"` }
		json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("headscale %s %s : %s %s", method, path, resp.Status, e.Message)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (h *Headscale) userID(ctx context.Context) (string, error) {
	var res struct {
		Users []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"users"`
	}
	if err := h.do(ctx, "GET", "/api/v1/user?name="+url.QueryEscape(h.User), nil, &res); err != nil {
		return "", err
	}
	for _, u := range res.Users {
		if u.Name == h.User {
			return u.ID, nil
		}
	}
	var created struct {
		User struct{ ID string `json:"id"` } `json:"user"`
	}
	if err := h.do(ctx, "POST", "/api/v1/user", map[string]string{"name": h.User}, &created); err != nil {
		return "", err
	}
	return created.User.ID, nil
}

// PreauthKey crée une clé à usage unique (ephemeral pour les noeuds de diagnostic).
func (h *Headscale) PreauthKey(ctx context.Context, ephemeral bool, ttl time.Duration) (string, error) {
	uid, err := h.userID(ctx)
	if err != nil {
		return "", err
	}
	var res struct {
		PreAuthKey struct{ Key string `json:"key"` } `json:"preAuthKey"`
	}
	err = h.do(ctx, "POST", "/api/v1/preauthkey", map[string]any{
		"user": uid, "reusable": false, "ephemeral": ephemeral,
		"expiration": time.Now().Add(ttl).UTC().Format(time.RFC3339),
	}, &res)
	if err != nil {
		return "", err
	}
	return res.PreAuthKey.Key, nil
}

// DeleteNodes retire les noeuds dont le nom correspond (ex. ecran-03 et ecran-03-probe).
func (h *Headscale) DeleteNodes(ctx context.Context, names ...string) error {
	var res struct {
		Nodes []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			GivenName string `json:"givenName"`
		} `json:"nodes"`
	}
	if err := h.do(ctx, "GET", "/api/v1/node", nil, &res); err != nil {
		return err
	}
	for _, n := range res.Nodes {
		for _, want := range names {
			if n.Name == want || n.GivenName == want {
				if err := h.do(ctx, "DELETE", "/api/v1/node/"+n.ID, nil, nil); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
