// Package status : état courant de l'agent, partagé entre la boucle principale et l'API locale.
package status

import (
	"sync"
	"time"
)

type Snapshot struct {
	Version       string    `json:"version"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	PortalURL     string    `json:"portal_url"`
	HasProjectKey bool      `json:"has_project_key"`
	Phase         string    `json:"phase"`   // no-key | enrolling | pending | connecting | online | error | stopped
	Message       string    `json:"message"` // texte lisible
	TailnetIP     string    `json:"tailnet_ip"`
	DeviceID      int64     `json:"device_id"`
	SubDevices    []SubDev  `json:"sub_devices"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	LastError     string    `json:"last_error"`
	UpdatedAt     time.Time `json:"updated_at"`
	ConfigPath    string    `json:"config_path"`
}

type SubDev struct {
	Name      string  `json:"name"`
	Target    string  `json:"target"`
	Reachable bool    `json:"reachable"`
	RTTMS     float64 `json:"rtt_ms"`
}

type Store struct {
	mu   sync.RWMutex
	snap Snapshot
}

var Global = &Store{}

func (s *Store) Get() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

// Update applique une modification atomique.
func (s *Store) Update(fn func(*Snapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.snap)
	s.snap.UpdatedAt = time.Now()
}

func (s *Store) SetPhase(phase, msg string) {
	s.Update(func(x *Snapshot) {
		x.Phase, x.Message = phase, msg
		if phase != "error" {
			x.LastError = ""
		} else {
			x.LastError = msg
		}
	})
}
