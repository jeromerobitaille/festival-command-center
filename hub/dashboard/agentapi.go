package main

// API consommée par les agents (Bearer = jeton d'appareil, sauf enroll qui utilise la clé de projet).

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type enrollRequest struct {
	ProjectKey   string        `json:"project_key"`
	Slug         string        `json:"slug"`
	ScreenID     string        `json:"screen_id"` // agents ≤ 0.4
	Name         string        `json:"name"`
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	Arch         string        `json:"arch"`
	AgentVersion string        `json:"agent_version"`
	LocalConfig  *DeviceConfig `json:"local_config"` // proposition initiale (processeur, forwards)
}

// POST /api/agent/enroll : crée (ou retrouve) l'appareil en attente d'approbation.
func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req enrollRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		jsonError(w, 400, "JSON invalide")
		return
	}
	p, err := s.getProjectByKey(strings.TrimSpace(req.ProjectKey))
	if err != nil {
		jsonError(w, 401, "clé de projet inconnue")
		return
	}
	if req.Slug == "" {
		req.Slug = req.ScreenID
	}
	req.Slug = slugify(req.Slug)
	if req.Slug == "" {
		jsonError(w, 400, "slug requis")
		return
	}
	if req.Name == "" {
		req.Name = req.Slug
	}
	// Un appareil déjà connu (même projet, même slug) se ré-inscrit : nouveau jeton, statut conservé
	// sauf s'il avait été révoqué, auquel cas il repasse en attente.
	token := randomToken("dev_", 32)
	var existing Device
	err = s.db.QueryRow(`SELECT id, status FROM devices WHERE project_id=? AND slug=?`, p.ID, req.Slug).Scan(&existing.ID, &existing.Status)
	cfg := DeviceConfig{Name: req.Name, HeartbeatSeconds: 15, Update: UpdateConf{Enabled: true, CheckHours: 1}}
	if req.LocalConfig != nil {
		req.LocalConfig.normalize()
		cfg.SubDevices = req.LocalConfig.SubDevices
		cfg.Forwards = req.LocalConfig.Forwards
	}
	cfg.normalize()
	cfgJSON, _ := json.Marshal(cfg)
	if err == nil {
		status := existing.Status
		if status == "revoked" || status == "rejected" {
			status = "pending"
		}
		_, err = s.db.Exec(`UPDATE devices SET token_hash=?, status=?, hostname=?, os=?, arch=?, agent_version=?, headscale_key_delivered=0 WHERE id=?`,
			hashToken(token), status, req.Hostname, req.OS, req.Arch, req.AgentVersion, existing.ID)
	} else {
		res, e := s.db.Exec(`INSERT INTO devices(project_id, slug, name, status, token_hash, hostname, os, arch, agent_version, config, enrolled_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, p.ID, req.Slug, req.Name, "pending", hashToken(token), req.Hostname, req.OS, req.Arch, req.AgentVersion, string(cfgJSON), now())
		err = e
		if e == nil {
			existing.ID, _ = res.LastInsertId()
			log.Printf("[agent] nouvel appareil %s en attente dans le projet %s", req.Slug, p.Slug)
		}
	}
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"device_id": existing.ID, "device_token": token, "status": "pending"})
}

// deviceFromRequest authentifie un agent par son jeton.
func (s *Server) deviceFromRequest(w http.ResponseWriter, r *http.Request) *Device {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tok == "" {
		jsonError(w, 401, "jeton manquant")
		return nil
	}
	d, err := s.getDeviceByToken(tok)
	if err != nil {
		jsonError(w, 401, "jeton inconnu : l'appareil doit se ré-inscrire")
		return nil
	}
	return d
}

type statusResponse struct {
	Status        string `json:"status"`
	ConfigVersion int64  `json:"config_version"`
	ControlURL    string `json:"control_url,omitempty"`
	AuthKey       string `json:"auth_key,omitempty"` // livrée une seule fois
}

// GET /api/agent/status : l'agent l'interroge en attendant l'approbation, puis à chaque heartbeat via la réponse.
func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	s.db.Exec(`UPDATE devices SET last_seen=? WHERE id=?`, now(), d.ID)
	writeJSON(w, s.statusFor(d))
}

func (s *Server) statusFor(d *Device) statusResponse {
	resp := statusResponse{Status: d.Status, ConfigVersion: d.ConfigVersion}
	if d.Status != "approved" {
		return resp
	}
	resp.ControlURL = s.hs.Public
	var key string
	var delivered int
	s.db.QueryRow(`SELECT headscale_key, headscale_key_delivered FROM devices WHERE id=?`, d.ID).Scan(&key, &delivered)
	if key != "" && delivered == 0 {
		resp.AuthKey = key
	}
	return resp
}

// POST /api/agent/ack-key : l'agent confirme avoir consommé la clé Headscale.
func (s *Server) handleAckKey(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	s.db.Exec(`UPDATE devices SET headscale_key_delivered=1, headscale_key='' WHERE id=?`, d.ID)
	w.WriteHeader(204)
}

// GET /api/agent/config
func (s *Server) handleAgentConfig(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	if d.Status != "approved" {
		jsonError(w, 403, "appareil non approuvé")
		return
	}
	writeJSON(w, map[string]any{"version": d.ConfigVersion, "config": d.Config.forAgent()})
}

// POST /api/agent/heartbeat : état de l'agent ; réponse = statut, version de config, commandes en attente.
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	body, err := readAll(r, 256<<10)
	if err != nil || !json.Valid(body) {
		jsonError(w, 400, "JSON invalide")
		return
	}
	var hb struct {
		TailnetIP    string `json:"tailnet_ip"`
		AgentVersion string `json:"agent_version"`
		System       struct {
			Hostname string `json:"hostname"`
			OS       string `json:"os"`
		} `json:"system"`
	}
	json.Unmarshal(body, &hb)
	s.db.Exec(`UPDATE devices SET last_heartbeat=?, last_seen=?, tailnet_ip=?, agent_version=?, hostname=? WHERE id=?`,
		string(body), now(), hb.TailnetIP, hb.AgentVersion, hb.System.Hostname, d.ID)

	var cmds []Command
	if d.Status == "approved" {
		rows, _ := s.db.Query(`SELECT id, device_id, kind, payload, status, result, created_at FROM commands WHERE device_id=? AND status='queued' ORDER BY id LIMIT 20`, d.ID)
		if rows != nil {
			for rows.Next() {
				var c Command
				var payload string
				rows.Scan(&c.ID, &c.DeviceID, &c.Kind, &payload, &c.Status, &c.Result, &c.CreatedAt)
				c.Payload = json.RawMessage(payload)
				cmds = append(cmds, c)
			}
			rows.Close()
			for _, c := range cmds {
				s.db.Exec(`UPDATE commands SET status='sent' WHERE id=?`, c.ID)
			}
		}
	}
	if cmds == nil {
		cmds = []Command{}
	}
	writeJSON(w, map[string]any{"status": s.statusFor(d), "commands": cmds})
}

// POST /api/agent/commands/{id}/result
func (s *Server) handleCommandResult(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var res struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&res); err != nil {
		jsonError(w, 400, "JSON invalide")
		return
	}
	status := "done"
	if !res.OK {
		status = "failed"
	}
	s.db.Exec(`UPDATE commands SET status=?, result=?, finished_at=? WHERE id=? AND device_id=?`, status, res.Result, now(), id, d.ID)
	// remonter le résultat sur l'action / l'automatisation d'origine
	var actionID, automationID *int64
	s.db.QueryRow(`SELECT action_id, automation_id FROM commands WHERE id=?`, id).Scan(&actionID, &automationID)
	summary := d.Name + " : " + res.Result
	if actionID != nil {
		s.db.Exec(`UPDATE actions SET last_result=? WHERE id=?`, summary, *actionID)
	}
	if automationID != nil {
		s.db.Exec(`UPDATE automations SET last_result=? WHERE id=?`, summary, *automationID)
	}
	w.WriteHeader(204)
}

// POST /api/agent/probe-key : clé Headscale éphémère pour `agent probe`.
func (s *Server) handleProbeKey(w http.ResponseWriter, r *http.Request) {
	d := s.deviceFromRequest(w, r)
	if d == nil {
		return
	}
	if d.Status != "approved" {
		jsonError(w, 403, "appareil non approuvé")
		return
	}
	if !s.hs.enabled() {
		jsonError(w, 503, "Headscale non configuré côté portail")
		return
	}
	key, err := s.hs.PreauthKey(r.Context(), true, time.Hour)
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	writeJSON(w, map[string]string{"control_url": s.hs.Public, "auth_key": key})
}

// approveDevice : passe l'appareil en approuvé et prépare sa clé Headscale.
func (s *Server) approveDevice(ctx context.Context, d *Device) error {
	key := ""
	if s.hs.enabled() {
		k, err := s.hs.PreauthKey(ctx, false, 24*time.Hour)
		if err != nil {
			return err
		}
		key = k
	}
	_, err := s.db.Exec(`UPDATE devices SET status='approved', approved_at=?, headscale_key=?, headscale_key_delivered=0 WHERE id=?`, now(), key, d.ID)
	return err
}

func (s *Server) revokeDevice(ctx context.Context, d *Device, status string) error {
	if s.hs.enabled() {
		if err := s.hs.DeleteNodes(ctx, d.Slug, d.Slug+"-probe"); err != nil {
			log.Printf("[headscale] retrait des noeuds de %s : %v", d.Slug, err)
		}
	}
	_, err := s.db.Exec(`UPDATE devices SET status=?, headscale_key='', headscale_key_delivered=0, tailnet_ip='' WHERE id=?`, status, d.ID)
	return err
}
