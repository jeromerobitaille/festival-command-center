// Package config charge et valide agent.toml.
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Screen        Screen        `toml:"screen"`
	Hub           Hub           `toml:"hub"`
	Processor     Processor     `toml:"processor"`
	Forwards      []Forward     `toml:"forward"`
	RemoteDesktop RemoteDesktop `toml:"remote_desktop"`
	Agent         Agent         `toml:"agent"`
	Update        Update        `toml:"update"`

	// Path est le chemin absolu du fichier chargé (non sérialisé).
	Path string `toml:"-"`
}

type Screen struct {
	ID   string `toml:"id"`
	Name string `toml:"name"`
}

type Hub struct {
	ControlURL       string `toml:"control_url"`
	AuthKey          string `toml:"auth_key"`
	APIURL           string `toml:"api_url"`
	APIToken         string `toml:"api_token"`
	HeartbeatSeconds int    `toml:"heartbeat_seconds"`
}

type Processor struct {
	IP   string `toml:"ip"`
	Port int    `toml:"port"`
}

type Forward struct {
	Name   string `toml:"name"`
	Proto  string `toml:"proto"`
	Listen int    `toml:"listen"`
	Target string `toml:"target"`
}

type RemoteDesktop struct {
	Provider string `toml:"provider"`
	ID       string `toml:"id"`
}

type Agent struct {
	StateDir string `toml:"state_dir"`
	LogFile  string `toml:"log_file"`
}

type Update struct {
	Enabled    *bool  `toml:"enabled"`     // défaut : true
	BaseURL    string `toml:"base_url"`    // défaut : releases GitHub du projet
	CheckHours int    `toml:"check_hours"` // défaut : 1
}

func (u Update) IsEnabled() bool { return u.Enabled == nil || *u.Enabled }

// DefaultPath retourne agent.toml à côté de l'exécutable.
func DefaultPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "agent.toml"
	}
	return filepath.Join(filepath.Dir(exe), "agent.toml")
}

func Load(path string) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var c Config
	md, err := toml.DecodeFile(abs, &c)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", abs, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		return nil, fmt.Errorf("clés inconnues dans %s : %s", abs, strings.Join(keys, ", "))
	}
	c.Path = abs
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("configuration invalide (%s) : %w", abs, err)
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Hub.HeartbeatSeconds <= 0 {
		c.Hub.HeartbeatSeconds = 15
	}
	if c.Processor.Port == 0 {
		c.Processor.Port = 37564
	}
	if c.Update.CheckHours <= 0 {
		c.Update.CheckHours = 1
	}
	if c.RemoteDesktop.Provider == "" {
		c.RemoteDesktop.Provider = "rustdesk"
	}
	if c.Agent.StateDir == "" {
		c.Agent.StateDir = filepath.Join(filepath.Dir(c.Path), "state")
	}
	for i := range c.Forwards {
		if c.Forwards[i].Proto == "" {
			c.Forwards[i].Proto = "tcp"
		}
		c.Forwards[i].Proto = strings.ToLower(c.Forwards[i].Proto)
	}
}

func (c *Config) validate() error {
	if c.Screen.ID == "" {
		return fmt.Errorf("screen.id est requis")
	}
	for _, r := range c.Screen.ID {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return fmt.Errorf("screen.id doit contenir uniquement a-z, 0-9 et '-' (reçu %q)", c.Screen.ID)
		}
	}
	if c.Hub.ControlURL == "" {
		return fmt.Errorf("hub.control_url est requis")
	}
	if c.Hub.AuthKey == "" {
		return fmt.Errorf("hub.auth_key est requis")
	}
	if c.Processor.IP != "" && net.ParseIP(c.Processor.IP) == nil {
		return fmt.Errorf("processor.ip invalide : %q", c.Processor.IP)
	}
	seen := map[string]bool{}
	for i, f := range c.Forwards {
		if f.Name == "" {
			return fmt.Errorf("forward[%d] : name est requis", i)
		}
		if f.Proto != "tcp" && f.Proto != "udp" {
			return fmt.Errorf("forward %q : proto doit être tcp ou udp", f.Name)
		}
		if f.Listen <= 0 || f.Listen > 65535 {
			return fmt.Errorf("forward %q : listen doit être un port 1-65535", f.Name)
		}
		if _, _, err := net.SplitHostPort(f.Target); err != nil {
			return fmt.Errorf("forward %q : target doit être host:port (%v)", f.Name, err)
		}
		key := fmt.Sprintf("%s/%d", f.Proto, f.Listen)
		if seen[key] {
			return fmt.Errorf("forward %q : port %s déjà utilisé", f.Name, key)
		}
		seen[key] = true
	}
	return nil
}
