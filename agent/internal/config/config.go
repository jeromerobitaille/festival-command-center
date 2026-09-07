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

// DefaultPortalURL est utilisé quand portal.url est absent.
const DefaultPortalURL = "https://panel.veam.ca"

// DefaultLocalAddr : API et panneau locaux.
const DefaultLocalAddr = "127.0.0.1:47632"

type Config struct {
	Portal     Portal      `toml:"portal"`
	Device     Device      `toml:"device"`
	SubDevices []SubDevice `toml:"sub_device"`
	Forwards   []Forward   `toml:"forward"`
	Agent      Agent       `toml:"agent"`
	Update     Update      `toml:"update"`

	// Anciennes clés (agent ≤ 0.4), fusionnées dans Device / SubDevices au chargement.
	Screen    Device    `toml:"screen"`
	Processor SubDevice `toml:"processor"`

	Path string `toml:"-"`
}

type Portal struct {
	URL        string `toml:"url"`
	ProjectKey string `toml:"project_key"`
}

// Device : identité de cet appareil dans le projet.
type Device struct {
	ID   string `toml:"id"`
	Name string `toml:"name"`
}

// SubDevice : équipement branché sur le réseau local de l'appareil (processeur LED, projecteur, automate…).
// Avec Forward, c'est la proposition initiale envoyée au portail à l'inscription ; ensuite le portail fait foi.
type SubDevice struct {
	Name string `toml:"name"`
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
	StateDir  string `toml:"state_dir"`
	LogFile   string `toml:"log_file"`
	LocalAddr string `toml:"local_addr"` // API/panneau local, défaut 127.0.0.1:47632
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
		if os.IsNotExist(err) {
			// Pas encore de fichier : configuration par défaut, à compléter depuis le panneau local.
			c.Path = abs
			c.applyDefaults()
			return &c, nil
		}
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
	if c.Portal.URL == "" {
		c.Portal.URL = DefaultPortalURL
	}
	c.Portal.ProjectKey = strings.TrimSpace(c.Portal.ProjectKey)
	// compatibilité : [screen] → [device], [processor] → [[sub_device]]
	if c.Device.ID == "" {
		c.Device.ID = c.Screen.ID
	}
	if c.Device.Name == "" {
		c.Device.Name = c.Screen.Name
	}
	if c.Processor.IP != "" {
		name := c.Processor.Name
		if name == "" {
			name = "Processeur"
		}
		port := c.Processor.Port
		if port == 0 {
			port = 37564 // ancien défaut de [processor]
		}
		c.SubDevices = append([]SubDevice{{Name: name, IP: c.Processor.IP, Port: port}}, c.SubDevices...)
	}
	if c.Device.ID == "" {
		h, _ := os.Hostname()
		c.Device.ID = h
	}
	c.Device.ID = Slugify(c.Device.ID)
	if c.Device.Name == "" {
		c.Device.Name = c.Device.ID
	}
	for i := range c.SubDevices {
		if c.SubDevices[i].Name == "" {
			c.SubDevices[i].Name = fmt.Sprintf("sous-appareil %d", i+1)
		}
	}
	if c.Agent.LocalAddr == "" {
		c.Agent.LocalAddr = DefaultLocalAddr
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
	// Sans forward déclaré : un forward TCP par sous-appareil qui a un port.
	if len(c.Forwards) == 0 {
		for _, sd := range c.SubDevices {
			if sd.IP != "" && sd.Port > 0 {
				c.Forwards = append(c.Forwards, Forward{Name: Slugify(sd.Name), Proto: "tcp", Listen: sd.Port, Target: net.JoinHostPort(sd.IP, fmt.Sprint(sd.Port))})
			}
		}
	}
}

func (c *Config) validate() error {
	if !strings.HasPrefix(c.Portal.URL, "http") {
		return fmt.Errorf("portal.url invalide : %q", c.Portal.URL)
	}
	// project_key peut être vide : l'agent démarre et attend qu'on la saisisse (panneau local).
	if c.Device.ID == "" {
		return fmt.Errorf("device.id est requis")
	}
	for _, sd := range c.SubDevices {
		if sd.IP == "" || net.ParseIP(sd.IP) == nil {
			return fmt.Errorf("sub_device %q : ip invalide %q", sd.Name, sd.IP)
		}
		if sd.Port <= 0 || sd.Port > 65535 {
			return fmt.Errorf("sub_device %q : port requis (1-65535)", sd.Name)
		}
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
