// Portail Festival Command Center : projets, appareils, tableaux de bord, actions, automatisations.
package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	db    dbHandle
	tmpl  *template.Template
	hs    *Headscale
	sched *Scheduler
	stale time.Duration
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	listen := env("LISTEN", ":8090")
	dataDir := env("DATA_DIR", "/data")
	staleSec, _ := strconv.Atoi(env("STALE_SECONDS", "45"))
	_ = os.MkdirAll(dataDir, 0o755)

	db, err := openDB(filepath.Join(dataDir, "portal.sqlite"))
	if err != nil {
		log.Fatal(err)
	}
	s := &Server{
		db:    db,
		stale: time.Duration(staleSec) * time.Second,
		hs: &Headscale{
			URL: os.Getenv("HEADSCALE_URL"), Public: os.Getenv("HEADSCALE_PUBLIC_URL"),
			APIKey: os.Getenv("HEADSCALE_API_KEY"), User: env("HEADSCALE_USER", "festival"),
			Client: &http.Client{Timeout: 15 * time.Second},
		},
	}
	if !s.hs.enabled() {
		log.Printf("HEADSCALE_URL / HEADSCALE_API_KEY absents : les appareils approuvés ne recevront pas de clé Headscale")
	}
	s.tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"json": func(v any) template.JS { b, _ := json.Marshal(v); return template.JS(b) },
	}).ParseFS(templateFS, "templates/*.html"))
	s.bootstrapAdmin()
	s.sched = newScheduler(s)
	s.sched.ReloadAll()
	go func() {
		for {
			s.purgeSessions()
			time.Sleep(time.Hour)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	// agents
	mux.HandleFunc("POST /api/agent/enroll", s.handleEnroll)
	mux.HandleFunc("GET /api/agent/status", s.handleAgentStatus)
	mux.HandleFunc("POST /api/agent/ack-key", s.handleAckKey)
	mux.HandleFunc("GET /api/agent/config", s.handleAgentConfig)
	mux.HandleFunc("POST /api/agent/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("POST /api/agent/commands/{id}/result", s.handleCommandResult)
	mux.HandleFunc("POST /api/agent/probe-key", s.handleProbeKey)

	s.registerPortal(mux)

	log.Printf("portail sur %s (données : %s, hors ligne après %s)", listen, dataDir, s.stale)
	log.Fatal(http.ListenAndServe(listen, mux))
}

// bootstrapAdmin crée le premier administrateur si la table est vide.
func (s *Server) bootstrapAdmin() {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n > 0 {
		return
	}
	user := env("ADMIN_USER", "admin")
	pw := os.Getenv("ADMIN_PASSWORD")
	generated := false
	if pw == "" {
		pw = randomToken("", 9)
		generated = true
	}
	h, _ := hashPassword(pw)
	s.db.Exec(`INSERT INTO users(username, password_hash, role, created_at) VALUES(?,?,?,?)`, user, h, "admin", now())
	if generated {
		log.Printf("=== premier administrateur créé : %s / %s  (changer le mot de passe dans le portail) ===", user, pw)
	} else {
		log.Printf("premier administrateur créé : %s (mot de passe depuis ADMIN_PASSWORD)", user)
	}
}

// ---- utilitaires HTTP ----

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func readAll(r *http.Request, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, limit))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		jsonError(w, 400, "JSON invalide : "+err.Error())
		return false
	}
	return true
}

func pathID(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id
}

func (s *Server) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s : %v", name, err)
	}
}

func trimLower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
