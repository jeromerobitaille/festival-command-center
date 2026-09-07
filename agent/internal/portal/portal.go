// Package portal : client de l'API agent du portail (inscription, approbation, config, heartbeat, commandes).
package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

var ErrUnauthorized = errors.New("jeton refusé par le portail")

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api/agent"+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode/100 != 2 {
		var e struct{ Error string `json:"error"` }
		json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("portail %s %s : %s %s", method, path, resp.Status, e.Error)
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// ---- types partagés avec le portail ----

type DeviceConfig struct {
	Name             string          `json:"name"`
	HeartbeatSeconds int             `json:"heartbeat_seconds"`
	SubDevices       []SubDeviceConf `json:"sub_devices"`
	Forwards         []ForwardConf   `json:"forwards"`
	Update           UpdateConf      `json:"update"`
}
type SubDeviceConf struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}
type ForwardConf struct {
	Name   string `json:"name"`
	Proto  string `json:"proto"`
	Listen int    `json:"listen"`
	Target string `json:"target"`
}
type UpdateConf struct {
	Enabled    bool `json:"enabled"`
	CheckHours int  `json:"check_hours"`
}

type EnrollRequest struct {
	ProjectKey   string        `json:"project_key"`
	Slug         string        `json:"slug"` // identifiant de l'appareil dans le projet
	Name         string        `json:"name"`
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	Arch         string        `json:"arch"`
	AgentVersion string        `json:"agent_version"`
	LocalConfig  *DeviceConfig `json:"local_config,omitempty"`
}

type Status struct {
	Status        string `json:"status"`
	ConfigVersion int64  `json:"config_version"`
	ControlURL    string `json:"control_url"`
	AuthKey       string `json:"auth_key"`
}

type Command struct {
	ID      int64           `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type HeartbeatResponse struct {
	Status   Status    `json:"status"`
	Commands []Command `json:"commands"`
}

func (c *Client) Enroll(ctx context.Context, req EnrollRequest) (deviceID int64, token string, err error) {
	var res struct {
		DeviceID    int64  `json:"device_id"`
		DeviceToken string `json:"device_token"`
	}
	if err = c.do(ctx, "POST", "/enroll", req, &res); err != nil {
		return 0, "", err
	}
	return res.DeviceID, res.DeviceToken, nil
}

func (c *Client) Status(ctx context.Context) (*Status, error) {
	var s Status
	return &s, c.do(ctx, "GET", "/status", nil, &s)
}

func (c *Client) AckKey(ctx context.Context) error { return c.do(ctx, "POST", "/ack-key", struct{}{}, nil) }

func (c *Client) Config(ctx context.Context) (int64, *DeviceConfig, error) {
	var res struct {
		Version int64        `json:"version"`
		Config  DeviceConfig `json:"config"`
	}
	if err := c.do(ctx, "GET", "/config", nil, &res); err != nil {
		return 0, nil, err
	}
	return res.Version, &res.Config, nil
}

func (c *Client) Heartbeat(ctx context.Context, payload any) (*HeartbeatResponse, error) {
	var res HeartbeatResponse
	return &res, c.do(ctx, "POST", "/heartbeat", payload, &res)
}

func (c *Client) CommandResult(ctx context.Context, id int64, ok bool, result string) error {
	return c.do(ctx, "POST", fmt.Sprintf("/commands/%d/result", id), map[string]any{"ok": ok, "result": result}, nil)
}

func (c *Client) ProbeKey(ctx context.Context) (controlURL, authKey string, err error) {
	var res struct {
		ControlURL string `json:"control_url"`
		AuthKey    string `json:"auth_key"`
	}
	err = c.do(ctx, "POST", "/probe-key", struct{}{}, &res)
	return res.ControlURL, res.AuthKey, err
}

// ---- état local persistant (jeton, clé Headscale) ----

type DeviceState struct {
	DeviceID   int64  `json:"device_id"`
	Token      string `json:"token"`
	ControlURL string `json:"control_url"`
	Joined     bool   `json:"joined"` // le noeud tsnet a déjà été inscrit auprès de Headscale
}

func LoadState(dir string) (*DeviceState, error) {
	b, err := os.ReadFile(filepath.Join(dir, "device.json"))
	if err != nil {
		return nil, err
	}
	var s DeviceState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveState(dir string, s *DeviceState) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := filepath.Join(dir, "device.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "device.json"))
}
