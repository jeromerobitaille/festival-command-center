package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (u *User) IsAdmin() bool { return u != nil && u.Role == "admin" }

type Project struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Slug       string          `json:"slug"`
	ProjectKey string          `json:"project_key,omitempty"`
	Timezone   string          `json:"timezone"`
	Dashboard  json.RawMessage `json:"dashboard"`
	CreatedAt  string          `json:"created_at"`
}

// DeviceConfig est la configuration poussée à l'agent.
type DeviceConfig struct {
	Name             string          `json:"name"`
	HeartbeatSeconds int             `json:"heartbeat_seconds"`
	SubDevices       []SubDeviceConf `json:"sub_devices"`
	Forwards         []ForwardConf   `json:"forwards"`
	Update           UpdateConf      `json:"update"`
	// Compatibilité agents ≤ 0.4 : premier sous-appareil, en lecture (config envoyée) et en écriture (proposition reçue).
	Processor *SubDeviceConf `json:"processor,omitempty"`
}

// SubDeviceConf : équipement sur le réseau local de l'appareil (processeur LED, projecteur, automate…).
type SubDeviceConf struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// normalize fusionne l'ancien champ processor et complète les noms.
func (c *DeviceConfig) normalize() {
	if c.Processor != nil && c.Processor.IP != "" && len(c.SubDevices) == 0 {
		name := c.Processor.Name
		if name == "" {
			name = "Processeur"
		}
		c.SubDevices = []SubDeviceConf{{Name: name, IP: c.Processor.IP, Port: c.Processor.Port}}
	}
	c.Processor = nil
	if c.SubDevices == nil {
		c.SubDevices = []SubDeviceConf{}
	}
	if c.Forwards == nil {
		c.Forwards = []ForwardConf{}
	}
	for i := range c.SubDevices {
		if c.SubDevices[i].Name == "" {
			c.SubDevices[i].Name = fmt.Sprintf("sous-appareil %d", i+1)
		}
	}
}

// forAgent : copie envoyée aux agents, avec le champ processor pour les anciennes versions.
func (c DeviceConfig) forAgent() DeviceConfig {
	c.normalize()
	if len(c.SubDevices) > 0 {
		sd := c.SubDevices[0]
		c.Processor = &sd
	}
	return c
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

type Device struct {
	ID            int64           `json:"id"`
	ProjectID     int64           `json:"project_id"`
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	Hostname      string          `json:"hostname"`
	OS            string          `json:"os"`
	Arch          string          `json:"arch"`
	AgentVersion  string          `json:"agent_version"`
	TailnetIP     string          `json:"tailnet_ip"`
	Config        DeviceConfig    `json:"config"`
	ConfigVersion int64           `json:"config_version"`
	LastHeartbeat json.RawMessage `json:"last_heartbeat"`
	LastSeen      string          `json:"last_seen"`
	EnrolledAt    string          `json:"enrolled_at"`
	ApprovedAt    string          `json:"approved_at"`
	// dérivés
	Online     bool  `json:"online"`
	AgeSeconds int64 `json:"age_seconds"`
}

type Action struct {
	ID             int64             `json:"id"`
	ProjectID      int64             `json:"project_id"`
	Name           string            `json:"name"`
	Kind           string            `json:"kind"` // hub | agent
	DeviceID       *int64            `json:"device_id"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	LastRun        string            `json:"last_run"`
	LastResult     string            `json:"last_result"`
}

type Automation struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"project_id"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Cron       string `json:"cron"`
	ActionID   int64  `json:"action_id"`
	ActionName string `json:"action_name,omitempty"`
	LastRun    string `json:"last_run"`
	LastResult string `json:"last_result"`
	NextRun    string `json:"next_run,omitempty"`
}

type Command struct {
	ID        int64           `json:"id"`
	DeviceID  int64           `json:"device_id"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	Result    string          `json:"result"`
	CreatedAt string          `json:"created_at"`
}

// ---- requêtes ----

func (s *Server) getUserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(`SELECT id, username, role, created_at FROM users WHERE id=?`, id).Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Server) listUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, role, created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
		out = append(out, u)
	}
	return out, nil
}

func scanProject(row interface{ Scan(...any) error }) (*Project, error) {
	p := &Project{}
	var dash string
	if err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.ProjectKey, &p.Timezone, &dash, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Dashboard = json.RawMessage(dash)
	return p, nil
}

const projectCols = `id, name, slug, project_key, timezone, dashboard, created_at`

func (s *Server) getProjectBySlug(slug string) (*Project, error) {
	return scanProject(s.db.QueryRow(`SELECT `+projectCols+` FROM projects WHERE slug=?`, slug))
}
func (s *Server) getProjectByID(id int64) (*Project, error) {
	return scanProject(s.db.QueryRow(`SELECT `+projectCols+` FROM projects WHERE id=?`, id))
}
func (s *Server) getProjectByKey(key string) (*Project, error) {
	return scanProject(s.db.QueryRow(`SELECT `+projectCols+` FROM projects WHERE project_key=?`, key))
}

// listProjectsFor : tous pour un admin, sinon ceux dont l'utilisateur est membre.
func (s *Server) listProjectsFor(u *User) ([]*Project, error) {
	var rows *sql.Rows
	var err error
	if u.IsAdmin() {
		rows, err = s.db.Query(`SELECT ` + projectCols + ` FROM projects ORDER BY name`)
	} else {
		rows, err = s.db.Query(`SELECT `+projectCols+` FROM projects p JOIN project_members m ON m.project_id=p.id WHERE m.user_id=? ORDER BY name`, u.ID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		if !u.IsAdmin() {
			p.ProjectKey = ""
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Server) userCanAccess(u *User, projectID int64) bool {
	if u.IsAdmin() {
		return true
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM project_members WHERE project_id=? AND user_id=?`, projectID, u.ID).Scan(&n)
	return n > 0
}

const deviceCols = `id, project_id, slug, name, status, hostname, os, arch, agent_version, tailnet_ip, config, config_version, last_heartbeat, last_seen, enrolled_at, approved_at`

func (s *Server) scanDevice(row interface{ Scan(...any) error }) (*Device, error) {
	d := &Device{}
	var cfg, hb string
	if err := row.Scan(&d.ID, &d.ProjectID, &d.Slug, &d.Name, &d.Status, &d.Hostname, &d.OS, &d.Arch, &d.AgentVersion, &d.TailnetIP, &cfg, &d.ConfigVersion, &hb, &d.LastSeen, &d.EnrolledAt, &d.ApprovedAt); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(cfg), &d.Config)
	d.Config.normalize()
	if hb == "" {
		hb = "null"
	}
	d.LastHeartbeat = json.RawMessage(hb)
	if d.LastSeen != "" {
		if t, err := time.Parse(time.RFC3339, d.LastSeen); err == nil {
			d.AgeSeconds = int64(time.Since(t).Seconds())
			d.Online = time.Since(t) < s.stale
		}
	}
	return d, nil
}

func (s *Server) getDevice(id int64) (*Device, error) {
	return s.scanDevice(s.db.QueryRow(`SELECT `+deviceCols+` FROM devices WHERE id=?`, id))
}
func (s *Server) getDeviceByToken(token string) (*Device, error) {
	return s.scanDevice(s.db.QueryRow(`SELECT `+deviceCols+` FROM devices WHERE token_hash=?`, hashToken(token)))
}
func (s *Server) listDevices(projectID int64) ([]*Device, error) {
	rows, err := s.db.Query(`SELECT `+deviceCols+` FROM devices WHERE project_id=? ORDER BY status='pending' DESC, slug`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Device
	for rows.Next() {
		d, err := s.scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Server) listActions(projectID int64) ([]*Action, error) {
	rows, err := s.db.Query(`SELECT id, project_id, name, kind, device_id, method, url, headers, body, timeout_seconds, last_run, last_result FROM actions WHERE project_id=? ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Action
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
func scanAction(row interface{ Scan(...any) error }) (*Action, error) {
	a := &Action{}
	var hdr string
	var dev sql.NullInt64
	if err := row.Scan(&a.ID, &a.ProjectID, &a.Name, &a.Kind, &dev, &a.Method, &a.URL, &hdr, &a.Body, &a.TimeoutSeconds, &a.LastRun, &a.LastResult); err != nil {
		return nil, err
	}
	if dev.Valid {
		a.DeviceID = &dev.Int64
	}
	a.Headers = map[string]string{}
	json.Unmarshal([]byte(hdr), &a.Headers)
	return a, nil
}
func (s *Server) getAction(id int64) (*Action, error) {
	return scanAction(s.db.QueryRow(`SELECT id, project_id, name, kind, device_id, method, url, headers, body, timeout_seconds, last_run, last_result FROM actions WHERE id=?`, id))
}

func (s *Server) listAutomations(projectID int64) ([]*Automation, error) {
	rows, err := s.db.Query(`SELECT a.id, a.project_id, a.name, a.enabled, a.cron, a.action_id, IFNULL(x.name,''), a.last_run, a.last_result FROM automations a LEFT JOIN actions x ON x.id=a.action_id WHERE a.project_id=? ORDER BY a.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Automation
	for rows.Next() {
		a := &Automation{}
		var en int
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &en, &a.Cron, &a.ActionID, &a.ActionName, &a.LastRun, &a.LastResult); err != nil {
			return nil, err
		}
		a.Enabled = en == 1
		out = append(out, a)
	}
	return out, nil
}

func (s *Server) queueCommand(deviceID int64, kind string, payload any, actionID, automationID *int64) (int64, error) {
	b, _ := json.Marshal(payload)
	res, err := s.db.Exec(`INSERT INTO commands(device_id, kind, payload, created_at, action_id, automation_id) VALUES(?,?,?,?,?,?)`,
		deviceID, kind, string(b), now(), actionID, automationID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
