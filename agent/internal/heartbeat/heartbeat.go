// Package heartbeat envoie périodiquement l'état de l'agent au hub.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Payload struct {
	ScreenID     string    `json:"screen_id"`
	ScreenName   string    `json:"screen_name"`
	AgentVersion string    `json:"agent_version"`
	Timestamp    time.Time `json:"timestamp"`
	TailnetIP    string    `json:"tailnet_ip"`
	Processor    any       `json:"processor"`
	Forwards     any       `json:"forwards"`
	Peers        any       `json:"peers"`
	System       any       `json:"system"`
	RemoteDesktop struct {
		Provider string `json:"provider"`
		ID       string `json:"id,omitempty"`
	} `json:"remote_desktop"`
}

type Sender struct {
	URL    string
	Token  string
	Client *http.Client
}

func NewSender(apiURL, token string) *Sender {
	if apiURL == "" {
		return nil
	}
	return &Sender{
		URL:    strings.TrimRight(apiURL, "/") + "/heartbeat",
		Token:  token,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Sender) Send(ctx context.Context, p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("hub a répondu %s", resp.Status)
	}
	return nil
}

// Run appelle build() à chaque tick, envoie au hub si configuré, et journalise un résumé.
func Run(ctx context.Context, every time.Duration, s *Sender, build func(context.Context) Payload) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		p := build(ctx)
		logSummary(p)
		if s != nil {
			if err := s.Send(ctx, p); err != nil {
				log.Printf("[heartbeat] envoi au hub : %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func logSummary(p Payload) {
	b, _ := json.Marshal(p.Processor)
	log.Printf("[heartbeat] %s ip=%s processeur=%s", p.ScreenID, p.TailnetIP, string(b))
}
