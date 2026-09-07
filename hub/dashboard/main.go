// Dashboard : reçoit les heartbeats des agents et affiche l'état des écrans.
package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var assets embed.FS

// Heartbeat reflète le JSON envoyé par l'agent (voir agent/internal/heartbeat).
type Heartbeat struct {
	ScreenID     string    `json:"screen_id"`
	ScreenName   string    `json:"screen_name"`
	AgentVersion string    `json:"agent_version"`
	Timestamp    time.Time `json:"timestamp"`
	TailnetIP    string    `json:"tailnet_ip"`
	Processor    *struct {
		Target    string  `json:"target"`
		Reachable bool    `json:"reachable"`
		RTTMS     float64 `json:"rtt_ms"`
		Error     string  `json:"error,omitempty"`
	} `json:"processor"`
	Forwards []struct {
		Name        string `json:"name"`
		Proto       string `json:"proto"`
		Listen      int    `json:"listen"`
		Target      string `json:"target"`
		ActiveConns int64  `json:"active_conns"`
		TotalConns  int64  `json:"total_conns"`
		LastError   string `json:"last_error,omitempty"`
	} `json:"forwards"`
	Peers []struct {
		Hostname  string  `json:"hostname"`
		IP        string  `json:"ip"`
		Online    bool    `json:"online"`
		Direct    bool    `json:"direct"`
		Endpoint  string  `json:"endpoint,omitempty"`
		Relay     string  `json:"relay,omitempty"`
		LatencyMS float64 `json:"latency_ms"`
		PingError string  `json:"ping_error,omitempty"`
	} `json:"peers"`
	System *struct {
		Hostname      string  `json:"hostname"`
		OS            string  `json:"os"`
		UptimeSeconds uint64  `json:"uptime_seconds"`
		CPUPercent    float64 `json:"cpu_percent"`
		MemPercent    float64 `json:"mem_percent"`
	} `json:"system"`
	RemoteDesktop struct {
		Provider string `json:"provider"`
		ID       string `json:"id,omitempty"`
	} `json:"remote_desktop"`
}

// Screen est l'état conservé côté hub.
type Screen struct {
	Heartbeat
	ReceivedAt time.Time `json:"received_at"`
	// Champs dérivés au moment de la lecture.
	Online     bool   `json:"online"`
	AgeSeconds int64  `json:"age_seconds"`
	TesseraURL string `json:"tessera_direct_connect"`
	RustDeskURL string `json:"rustdesk_url"`
}

type Store struct {
	mu      sync.RWMutex
	screens map[string]*Screen
	file    string
}

func NewStore(file string) *Store {
	s := &Store{screens: map[string]*Screen{}, file: file}
	if b, err := os.ReadFile(file); err == nil {
		if err := json.Unmarshal(b, &s.screens); err != nil {
			log.Printf("état précédent illisible (%v), on repart à vide", err)
			s.screens = map[string]*Screen{}
		} else {
			log.Printf("état restauré : %d écran(s)", len(s.screens))
		}
	}
	return s
}

func (s *Store) Put(h Heartbeat) {
	s.mu.Lock()
	s.screens[h.ScreenID] = &Screen{Heartbeat: h, ReceivedAt: time.Now().UTC()}
	snapshot, _ := json.MarshalIndent(s.screens, "", " ")
	s.mu.Unlock()
	if s.file == "" {
		return
	}
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, snapshot, 0o644); err == nil {
		_ = os.Rename(tmp, s.file)
	}
}

func (s *Store) List(stale time.Duration, rustdeskServer string) []*Screen {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	out := make([]*Screen, 0, len(s.screens))
	for _, sc := range s.screens {
		c := *sc
		c.AgeSeconds = int64(now.Sub(c.ReceivedAt).Seconds())
		c.Online = now.Sub(c.ReceivedAt) < stale
		if c.TailnetIP != "" {
			port := "37564"
			for _, f := range c.Forwards {
				if f.Name == "tessera-remote" {
					port = strconv.Itoa(f.Listen)
				}
			}
			c.TesseraURL = c.TailnetIP + ":" + port
		}
		if c.RemoteDesktop.ID != "" {
			c.RustDeskURL = "rustdesk://connection/new/" + c.RemoteDesktop.ID
		}
		out = append(out, &c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ScreenID < out[j].ScreenID })
	return out
}

type Summary struct {
	Total, Online, ProcessorKO, Relayed int
}

// summarize compte les écrans ; « relayé » ne regarde que les pairs de la régie (OPERATOR_HOSTS).
func summarize(list []*Screen, operators []string) Summary {
	isOperator := func(h string) bool {
		if len(operators) == 0 {
			return true
		}
		for _, o := range operators {
			if strings.EqualFold(o, h) {
				return true
			}
		}
		return false
	}
	var s Summary
	for _, sc := range list {
		s.Total++
		if sc.Online {
			s.Online++
		}
		if sc.Online && sc.Processor != nil && !sc.Processor.Reachable {
			s.ProcessorKO++
		}
		if sc.Online {
			for _, p := range sc.Peers {
				if p.Online && !p.Direct && isOperator(p.Hostname) {
					s.Relayed++
					break
				}
			}
		}
	}
	return s
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	listen := env("LISTEN", ":8090")
	token := os.Getenv("HEARTBEAT_TOKEN")
	dataFile := env("DATA_FILE", "state.json")
	rustdeskServer := os.Getenv("RUSTDESK_SERVER")
	staleSec, _ := strconv.Atoi(env("STALE_SECONDS", "45"))
	stale := time.Duration(staleSec) * time.Second
	var operators []string
	for _, h := range strings.Split(os.Getenv("OPERATOR_HOSTS"), ",") {
		if h = strings.TrimSpace(h); h != "" {
			operators = append(operators, h)
		}
	}
	if token == "" {
		log.Fatal("HEARTBEAT_TOKEN est requis")
	}
	_ = os.MkdirAll(filepath.Dir(dataFile), 0o755)
	store := NewStore(dataFile)

	tmpl := template.Must(template.New("index.html").Funcs(template.FuncMap{
		"pct": func(f float64) string { return strconv.FormatFloat(f, 'f', 0, 64) },
		"ms":  func(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) },
		"uptime": func(sec uint64) string {
			d := time.Duration(sec) * time.Second
			if d >= 24*time.Hour {
				return strconv.Itoa(int(d.Hours()/24)) + " j " + strconv.Itoa(int(d.Hours())%24) + " h"
			}
			return strconv.Itoa(int(d.Hours())) + " h " + strconv.Itoa(int(d.Minutes())%60) + " min"
		},
		"age": func(sec int64) string {
			switch {
			case sec < 60:
				return strconv.FormatInt(sec, 10) + " s"
			case sec < 3600:
				return strconv.FormatInt(sec/60, 10) + " min"
			default:
				return strconv.FormatInt(sec/3600, 10) + " h"
			}
		},
	}).ParseFS(assets, "index.html"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	mux.HandleFunc("POST /api/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(auth), []byte(token)) != 1 {
			http.Error(w, "jeton invalide", http.StatusUnauthorized)
			return
		}
		var h Heartbeat
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&h); err != nil {
			http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
			return
		}
		if h.ScreenID == "" {
			http.Error(w, "screen_id manquant", http.StatusBadRequest)
			return
		}
		store.Put(h)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/screens", func(w http.ResponseWriter, r *http.Request) {
		list := store.List(stale, rustdeskServer)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"summary": summarize(list, operators), "screens": list, "generated_at": time.Now().UTC(),
		})
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		list := store.List(stale, rustdeskServer)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := tmpl.Execute(w, map[string]any{
			"Screens": list, "Summary": summarize(list, operators),
			"RustDeskServer": rustdeskServer, "StaleSeconds": staleSec,
			"Now": time.Now().Format("15:04:05"),
		})
		if err != nil {
			log.Printf("template : %v", err)
		}
	})

	log.Printf("dashboard sur %s (état : %s, écrans hors ligne après %s)", listen, dataFile, stale)
	log.Fatal(http.ListenAndServe(listen, mux))
}
