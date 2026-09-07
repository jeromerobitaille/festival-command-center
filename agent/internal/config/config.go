// Package config charge et valide agent.toml (configuration locale minimale ; le reste vient du portail).
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
	Portal    Portal    `toml:"portal"`
	Screen    Screen    `toml:"screen"`
	Processor Processor `toml:"processor"`
	Forwards  []Forward `toml:"forward"`
	Agent     Agent     `toml:"agent"`
	Update    Update    `toml:"update"`

	Path string `toml:"-"`
}

type Portal struct {
	URL        string `toml:"url"`
	ProjectKey string `toml:"project_key"`
}

type Screen struct {
	ID   string `toml:"id"`
	Name string `toml:"name"`
}

// Processor et Forward sont la proposition initiale envoyée au portail à l'inscription.
// Après approbation, la configuration du portail fait foi.
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

type Agent struct {
	StateDir string `toml:"state_dir"`
	LogFile  string `toml:"log_file"`
}

type Update struct {
	BaseURL string `toml:"base_url"` // remplace l'URL des releases GitHub (tests)
}

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
		return nil, fmt.Errorf("clés inconnues dans %s : %s (l'ancienne section [hub] est remplacée par [portal])", abs, strings.Join(keys, ", "))
	}
	c.Path = abs
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("configuration invalide (%s) : %w", abs, err)
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	c.Portal.URL = strings.TrimRight(strings.TrimSpace(c.Portal.URL), "/")
	if c.Screen.ID == "" {
		h, _ := os.Hostname()
		c.Screen.ID = h
	}
	c.Screen.ID = Slugify(c.Screen.ID)
	if c.Screen.Name == "" {
		c.Screen.Name = c.Screen.ID
	}
	if c.Processor.Port == 0 {
		c.Processor.Port = 37564
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
	// Forward par défaut vers le processeur si rien n'est déclaré.
	if len(c.Forwards) == 0 && c.Processor.IP != "" {
		c.Forwards = []Forward{{Name: "tessera-remote", Proto: "tcp", Listen: c.Processor.Port, Target: net.JoinHostPort(c.Processor.IP, fmt.Sprint(c.Processor.Port))}}
	}
}

func (c *Config) validate() error {
	if c.Portal.URL == "" || !strings.HasPrefix(c.Portal.URL, "http") {
		return fmt.Errorf("portal.url est requis (ex. https://panel.veam.ca)")
	}
	if c.Portal.ProjectKey == "" {
		return fmt.Errorf("portal.project_key est requis (Configuration du projet dans le portail)")
	}
	if c.Screen.ID == "" {
		return fmt.Errorf("screen.id est requis")
	}
	if c.Processor.IP != "" && net.ParseIP(c.Processor.IP) == nil {
		return fmt.Errorf("processor.ip invalide : %q", c.Processor.IP)
	}
	return ValidateForwards(c.Forwards)
}

// ValidateForwards sert aussi pour la configuration reçue du portail.
func ValidateForwards(fw []Forward) error {
	seen := map[string]bool{}
	for i, f := range fw {
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

// Slugify : identifiant sûr pour un nom de noeud (a-z, 0-9, '-').
func Slugify(s string) string {
	var b strings.Builder
	last := '-'
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			last = r
		default:
			if last != '-' {
				b.WriteRune('-')
				last = '-'
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
