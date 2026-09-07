// Package localapi : API et panneau web locaux (127.0.0.1) pour l'icône de la barre des tâches et l'opérateur.
package localapi

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/festival/command-center/agent/internal/config"
	"github.com/festival/command-center/agent/internal/status"
)

const DefaultAddr = "127.0.0.1:47632"

//go:embed panel.html
var panelHTML []byte

// Controller : ce que l'API peut demander à la boucle principale.
type Controller interface {
	Reload()          // relire agent.toml et redémarrer le cycle
	ResetEnrollment() // oublier jeton et réseau, puis Reload
	Restart()         // redémarrer le service (ou quitter en mode interactif)
}

type Server struct {
	cfgPath string
	ctrl    Controller
}

func Serve(ctx context.Context, addr, cfgPath string, ctrl Controller) error {
	s := &Server{cfgPath: cfgPath, ctrl: ctrl}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(panelHTML)
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		snap := status.Global.Get()
		snap.ConfigPath = cfgPath
		writeJSON(w, snap)
	})
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			b = []byte(DefaultTOML())
		}
		writeJSON(w, map[string]any{"path": cfgPath, "toml": string(b)})
	})
	mux.HandleFunc("PUT /api/config", s.putConfig)
	mux.HandleFunc("POST /api/project", s.postProject)
	mux.HandleFunc("POST /api/reload", func(w http.ResponseWriter, r *http.Request) { ctrl.Reload(); w.WriteHeader(204) })
	mux.HandleFunc("POST /api/reset", func(w http.ResponseWriter, r *http.Request) { ctrl.ResetEnrollment(); w.WriteHeader(204) })
	mux.HandleFunc("POST /api/restart", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204); go func() { time.Sleep(300 * time.Millisecond); ctrl.Restart() }() })

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("API locale %s : %w (un autre agent tourne déjà ?)", addr, err)
	}
	srv := &http.Server{Handler: noStore(mux), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	log.Printf("[local] panneau sur http://%s", addr)
	err = srv.Serve(ln)
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func noStore(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

// putConfig remplace agent.toml après validation.
func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var in struct{ TOML string `json:"toml"` }
	if err := json.NewDecoder(io.LimitReader(r.Body, 256<<10)).Decode(&in); err != nil {
		jsonError(w, 400, "JSON invalide")
		return
	}
	if err := s.writeConfig(in.TOML); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	s.ctrl.Reload()
	w.WriteHeader(204)
}

// postProject met à jour les champs essentiels (portail, clé, écran) sans toucher au reste du fichier.
func (s *Server) postProject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PortalURL  string `json:"portal_url"`
		ProjectKey string `json:"project_key"`
		DeviceID   string `json:"device_id"`
		Name       string `json:"name"`
		SubName    string `json:"sub_name"`
		SubIP      string `json:"sub_ip"`
		SubPort    int    `json:"sub_port"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&in); err != nil {
		jsonError(w, 400, "JSON invalide")
		return
	}
	var doc map[string]any
	if b, err := os.ReadFile(s.cfgPath); err == nil {
		toml.Unmarshal(b, &doc)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	set := func(section, key, val string) {
		sec, _ := doc[section].(map[string]any)
		if sec == nil {
			sec = map[string]any{}
		}
		if val != "" {
			sec[key] = val
		}
		doc[section] = sec
	}
	set("portal", "url", strings.TrimSpace(in.PortalURL))
	set("portal", "project_key", strings.TrimSpace(in.ProjectKey))
	set("device", "id", config.Slugify(in.DeviceID))
	set("device", "name", strings.TrimSpace(in.Name))
	delete(doc, "screen") // anciennes clés remplacées
	delete(doc, "processor")
	if ip := strings.TrimSpace(in.SubIP); ip != "" {
		name := strings.TrimSpace(in.SubName)
		if name == "" {
			name = "sous-appareil"
		}
		port := in.SubPort
		if port == 0 {
			port = 80
		}
		doc["sub_device"] = []map[string]any{{"name": name, "ip": ip, "port": port}}
	}
	var sb strings.Builder
	sb.WriteString("# Configuration locale de l'agent Festival Command Center (modifiée depuis le panneau).\n")
	if err := toml.NewEncoder(&sb).Encode(doc); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	if err := s.writeConfig(sb.String()); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	// Une nouvelle clé de projet ou un nouvel identifiant impliquent une nouvelle inscription.
	s.ctrl.ResetEnrollment()
	w.WriteHeader(204)
}

func (s *Server) writeConfig(content string) error {
	tmp := s.cfgPath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.cfgPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return fmt.Errorf("écriture : %w", err)
	}
	if _, err := config.Load(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.cfgPath)
}

// DefaultTOML : contenu initial de agent.toml.
func DefaultTOML() string {
	return `# Configuration locale de l'agent Festival Command Center.
[portal]
url         = "` + config.DefaultPortalURL + `"
project_key = ""

[device]
id   = ""      # identifiant dans le projet ; vide = nom de l'ordinateur
name = ""

# Sous-appareils sur le réseau local de cet appareil (facultatif, répétable).
# [[sub_device]]
# name = "Processeur LED"
# ip   = "192.168.0.10"
# port = 37564
`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
