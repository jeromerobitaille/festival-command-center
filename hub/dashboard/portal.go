package main

// Pages HTML et API JSON du portail (utilisateurs connectés).

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) registerPortal(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", s.pageLogin)
	mux.HandleFunc("POST /login", s.pageLogin)
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		s.destroySession(w, r)
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	// pages
	mux.HandleFunc("GET /{$}", s.requireUser(s.pageProjects))
	mux.HandleFunc("GET /admin/users", s.requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		users, _ := s.listUsers()
		if users == nil {
			users = []User{}
		}
		s.render(w, "users.html", s.base(r, map[string]any{"Users": users, "Title": "Utilisateurs", "Subtitle": "Comptes et rôles du portail", "Page": "users"}))
	}))
	for _, page := range []string{"dashboard", "devices", "actions", "automations", "notifications", "settings"} {
		page := page
		path := "GET /p/{slug}"
		if page != "dashboard" {
			path += "/" + page
		}
		mux.HandleFunc(path, s.requireUser(func(w http.ResponseWriter, r *http.Request) { s.pageProject(w, r, page) }))
	}

	// API portail
	mux.HandleFunc("GET /api/me", s.requireUser(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, currentUser(r)) }))
	mux.HandleFunc("POST /api/me/password", s.requireUser(s.apiChangePassword))

	mux.HandleFunc("GET /api/users", s.requireAdmin(func(w http.ResponseWriter, r *http.Request) { u, _ := s.listUsers(); writeJSON(w, u) }))
	mux.HandleFunc("POST /api/users", s.requireAdmin(s.apiCreateUser))
	mux.HandleFunc("DELETE /api/users/{id}", s.requireAdmin(s.apiDeleteUser))
	mux.HandleFunc("POST /api/users/{id}/password", s.requireAdmin(s.apiResetPassword))

	mux.HandleFunc("GET /api/projects", s.requireUser(func(w http.ResponseWriter, r *http.Request) {
		ps, _ := s.listProjectsFor(currentUser(r))
		type row struct {
			*Project
			Devices int `json:"devices"`
			Online  int `json:"online"`
			Pending int `json:"pending"`
		}
		out := []row{}
		for _, p := range ps {
			d, o, pe := s.projectStats(p)
			out = append(out, row{p, d, o, pe})
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("POST /api/projects", s.requireAdmin(s.apiCreateProject))
	mux.HandleFunc("GET /api/projects/{id}", s.requireUser(s.withProject(s.apiGetProject)))
	mux.HandleFunc("PUT /api/projects/{id}", s.requireAdmin(s.withProject(s.apiUpdateProject)))
	mux.HandleFunc("DELETE /api/projects/{id}", s.requireAdmin(s.withProject(s.apiDeleteProject)))
	mux.HandleFunc("POST /api/projects/{id}/rotate-key", s.requireAdmin(s.withProject(s.apiRotateKey)))
	mux.HandleFunc("GET /api/projects/{id}/dashboards", s.requireUser(s.withProject(func(w http.ResponseWriter, r *http.Request, p *Project) {
		d, _ := s.listDashboards(p.ID)
		writeJSON(w, d)
	})))
	mux.HandleFunc("POST /api/projects/{id}/dashboards", s.requireAdmin(s.withProject(s.apiCreateDashboard)))
	mux.HandleFunc("PUT /api/dashboards/{id}", s.requireAdmin(s.apiUpdateDashboard))
	mux.HandleFunc("DELETE /api/dashboards/{id}", s.requireAdmin(s.apiDeleteDashboard))
	mux.HandleFunc("GET /api/projects/{id}/events", s.requireUser(s.withProject(func(w http.ResponseWriter, r *http.Request, p *Project) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		ev, _ := s.listEvents(p.ID, limit, after)
		writeJSON(w, ev)
	})))
	mux.HandleFunc("GET /api/projects/{id}/members", s.requireAdmin(s.withProject(s.apiListMembers)))
	mux.HandleFunc("PUT /api/projects/{id}/members", s.requireAdmin(s.withProject(s.apiSetMembers)))

	mux.HandleFunc("GET /api/projects/{id}/devices", s.requireUser(s.withProject(s.apiListDevices)))
	mux.HandleFunc("GET /api/projects/{id}/overview", s.requireUser(s.withProject(s.apiOverview)))
	mux.HandleFunc("GET /api/projects/{id}/variables", s.requireUser(s.withProject(s.apiVariables)))
	mux.HandleFunc("POST /api/devices/{id}/approve", s.requireAdmin(s.withDevice(s.apiApproveDevice)))
	mux.HandleFunc("POST /api/devices/{id}/reject", s.requireAdmin(s.withDevice(func(w http.ResponseWriter, r *http.Request, d *Device) {
		warn, _ := s.revokeDevice(r.Context(), d, "rejected")
		writeJSON(w, map[string]string{"status": "rejected", "warning": warn})
	})))
	mux.HandleFunc("POST /api/devices/{id}/revoke", s.requireAdmin(s.withDevice(func(w http.ResponseWriter, r *http.Request, d *Device) {
		warn, _ := s.revokeDevice(r.Context(), d, "revoked")
		writeJSON(w, map[string]string{"status": "revoked", "warning": warn})
	})))
	mux.HandleFunc("DELETE /api/devices/{id}", s.requireAdmin(s.withDevice(func(w http.ResponseWriter, r *http.Request, d *Device) {
		warn, _ := s.revokeDevice(r.Context(), d, "revoked")
		s.db.Exec(`DELETE FROM commands WHERE device_id=?`, d.ID)
		s.db.Exec(`DELETE FROM devices WHERE id=?`, d.ID)
		writeJSON(w, map[string]string{"status": "deleted", "warning": warn})
	})))
	mux.HandleFunc("POST /api/devices/{id}/rekey", s.requireAdmin(s.withDevice(s.apiRekeyDevice)))
	mux.HandleFunc("PUT /api/devices/{id}/config", s.requireAdmin(s.withDevice(s.apiSaveDeviceConfig)))
	mux.HandleFunc("POST /api/devices/{id}/command", s.requireAdmin(s.withDevice(s.apiDeviceCommand)))
	mux.HandleFunc("GET /api/devices/{id}/commands", s.requireUser(s.withDevice(s.apiDeviceCommands)))

	mux.HandleFunc("GET /api/projects/{id}/actions", s.requireUser(s.withProject(func(w http.ResponseWriter, r *http.Request, p *Project) {
		a, _ := s.listActions(p.ID)
		if a == nil {
			a = []*Action{}
		}
		writeJSON(w, a)
	})))
	mux.HandleFunc("POST /api/projects/{id}/actions", s.requireAdmin(s.withProject(s.apiSaveAction)))
	mux.HandleFunc("PUT /api/actions/{id}", s.requireAdmin(s.apiUpdateAction))
	mux.HandleFunc("DELETE /api/actions/{id}", s.requireAdmin(s.apiDeleteAction))
	mux.HandleFunc("POST /api/actions/{id}/run", s.requireUser(s.apiRunAction))

	mux.HandleFunc("GET /api/projects/{id}/automations", s.requireUser(s.withProject(s.apiListAutomations)))
	mux.HandleFunc("POST /api/projects/{id}/automations", s.requireAdmin(s.withProject(s.apiSaveAutomation)))
	mux.HandleFunc("PUT /api/automations/{id}", s.requireAdmin(s.apiUpdateAutomation))
	mux.HandleFunc("DELETE /api/automations/{id}", s.requireAdmin(s.apiDeleteAutomation))
}

func (s *Server) base(r *http.Request, data map[string]any) map[string]any {
	u := currentUser(r)
	data["User"] = u
	if _, ok := data["Title"]; !ok {
		data["Title"] = "Command Center"
	}
	if _, ok := data["Page"]; !ok {
		data["Page"] = ""
	}
	if _, ok := data["Project"]; !ok {
		data["Project"] = nil
	}
	if _, ok := data["Subtitle"]; !ok {
		data["Subtitle"] = ""
	}
	ps, _ := s.listProjectsFor(u)
	type pl struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	list := []pl{}
	for _, p := range ps {
		list = append(list, pl{p.ID, p.Name, p.Slug})
	}
	data["Projects"] = list
	ini := strings.ToUpper(u.Username)
	if len(ini) > 2 {
		ini = ini[:2]
	}
	data["Initials"] = ini
	return data
}

// ---- pages ----

func (s *Server) pageLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		u, ok := s.authenticate(strings.TrimSpace(r.FormValue("username")), r.FormValue("password"))
		if !ok {
			time.Sleep(500 * time.Millisecond)
			s.render(w, "login.html", map[string]any{"Error": "Identifiants invalides"})
			return
		}
		s.createSession(w, r, u)
		next := r.FormValue("next")
		if next == "" || !strings.HasPrefix(next, "/") {
			next = "/"
		}
		http.Redirect(w, r, next, http.StatusFound)
		return
	}
	if s.userFromRequest(r) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.render(w, "login.html", map[string]any{"Next": r.URL.Query().Get("next")})
}

func (s *Server) pageProjects(w http.ResponseWriter, r *http.Request) {
	s.render(w, "projects.html", s.base(r, map[string]any{"Title": "Projets", "Subtitle": "Vos projets et leurs appareils", "Page": "projects"}))
}

// projectStats : compteurs d'appareils d'un projet.
func (s *Server) projectStats(p *Project) (devices, online, pending int) {
	devs, _ := s.listDevices(p.ID)
	for _, d := range devs {
		switch d.Status {
		case "approved":
			devices++
			if d.Online {
				online++
			}
		case "pending":
			pending++
		}
	}
	return
}

func (s *Server) pageProject(w http.ResponseWriter, r *http.Request, page string) {
	p, err := s.getProjectBySlug(r.PathValue("slug"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := currentUser(r)
	if !s.userCanAccess(u, p.ID) {
		http.Error(w, "accès refusé à ce projet", http.StatusForbidden)
		return
	}
	if !u.IsAdmin() {
		p.ProjectKey = ""
		if page == "settings" {
			http.Error(w, "réservé aux administrateurs", http.StatusForbidden)
			return
		}
	}
	titles := map[string][2]string{
		"dashboard": {"Tableau de bord", p.Name}, "devices": {"Appareils", "Appareils rattachés à " + p.Name},
		"actions": {"Actions", "Requêtes HTTP et automatisations de " + p.Name}, "automations": {"Actions", "Requêtes HTTP et automatisations de " + p.Name},
		"notifications": {"Notifications", "Journal des événements de " + p.Name}, "settings": {"Configuration", "Projet, clé, membres"},
	}
	t := titles[page]
	s.render(w, page+".html", s.base(r, map[string]any{"Project": p, "Page": page, "Title": t[0], "Subtitle": t[1]}))
}

// ---- helpers d'accès ----

func (s *Server) withProject(next func(http.ResponseWriter, *http.Request, *Project)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := s.getProjectByID(pathID(r, "id"))
		if err != nil {
			jsonError(w, 404, "projet introuvable")
			return
		}
		if !s.userCanAccess(currentUser(r), p.ID) {
			jsonError(w, 403, "accès refusé")
			return
		}
		next(w, r, p)
	}
}

func (s *Server) withDevice(next func(http.ResponseWriter, *http.Request, *Device)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := s.getDevice(pathID(r, "id"))
		if err != nil {
			jsonError(w, 404, "appareil introuvable")
			return
		}
		if !s.userCanAccess(currentUser(r), d.ProjectID) {
			jsonError(w, 403, "accès refusé")
			return
		}
		next(w, r, d)
	}
}

// ---- utilisateurs ----

func (s *Server) apiCreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password, Role string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Username = trimLower(in.Username)
	if in.Username == "" || len(in.Password) < 8 {
		jsonError(w, 400, "nom requis et mot de passe d'au moins 8 caractères")
		return
	}
	if in.Role != "admin" {
		in.Role = "user"
	}
	h, _ := hashPassword(in.Password)
	if _, err := s.db.Exec(`INSERT INTO users(username, password_hash, role, created_at) VALUES(?,?,?,?)`, in.Username, h, in.Role, now()); err != nil {
		jsonError(w, 409, "ce nom existe déjà")
		return
	}
	w.WriteHeader(201)
}

func (s *Server) apiDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == currentUser(r).ID {
		jsonError(w, 400, "impossible de supprimer son propre compte")
		return
	}
	s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	s.db.Exec(`DELETE FROM project_members WHERE user_id=?`, id)
	s.db.Exec(`DELETE FROM users WHERE id=?`, id)
	w.WriteHeader(204)
}

func (s *Server) apiResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct{ Password string }
	if !decodeJSON(w, r, &in) || len(in.Password) < 8 {
		jsonError(w, 400, "mot de passe d'au moins 8 caractères")
		return
	}
	h, _ := hashPassword(in.Password)
	s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, h, pathID(r, "id"))
	s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, pathID(r, "id"))
	w.WriteHeader(204)
}

func (s *Server) apiChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct{ Current, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	u := currentUser(r)
	if _, ok := s.authenticate(u.Username, in.Current); !ok {
		jsonError(w, 403, "mot de passe actuel incorrect")
		return
	}
	if len(in.Password) < 8 {
		jsonError(w, 400, "mot de passe d'au moins 8 caractères")
		return
	}
	h, _ := hashPassword(in.Password)
	s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, h, u.ID)
	w.WriteHeader(204)
}

// ---- projets ----

func (s *Server) apiCreateProject(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Timezone string }
	if !decodeJSON(w, r, &in) {
		return
	}
	slug := slugify(in.Name)
	if slug == "" {
		jsonError(w, 400, "nom requis")
		return
	}
	if in.Timezone == "" {
		in.Timezone = "America/Toronto"
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		jsonError(w, 400, "fuseau horaire inconnu")
		return
	}
	key := randomToken("fcc_", 24)
	res, err := s.db.Exec(`INSERT INTO projects(name, slug, project_key, timezone, created_at) VALUES(?,?,?,?,?)`, strings.TrimSpace(in.Name), slug, key, in.Timezone, now())
	if err != nil {
		jsonError(w, 409, "un projet avec ce nom existe déjà")
		return
	}
	id, _ := res.LastInsertId()
	p, _ := s.getProjectByID(id)
	writeJSON(w, p)
}

func (s *Server) apiGetProject(w http.ResponseWriter, r *http.Request, p *Project) {
	if !currentUser(r).IsAdmin() {
		p.ProjectKey = ""
	}
	writeJSON(w, p)
}

func (s *Server) apiUpdateProject(w http.ResponseWriter, r *http.Request, p *Project) {
	var in struct{ Name, Timezone string }
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name != "" {
		p.Name = strings.TrimSpace(in.Name)
	}
	if in.Timezone != "" {
		if _, err := time.LoadLocation(in.Timezone); err != nil {
			jsonError(w, 400, "fuseau horaire inconnu")
			return
		}
		p.Timezone = in.Timezone
	}
	s.db.Exec(`UPDATE projects SET name=?, timezone=? WHERE id=?`, p.Name, p.Timezone, p.ID)
	s.sched.Reload(p.ID)
	writeJSON(w, p)
}

func (s *Server) apiDeleteProject(w http.ResponseWriter, r *http.Request, p *Project) {
	devs, _ := s.listDevices(p.ID)
	for _, d := range devs {
		s.revokeDevice(r.Context(), d, "revoked")
	}
	for _, t := range []string{"commands", "automations", "actions", "devices", "project_members"} {
		if t == "commands" {
			s.db.Exec(`DELETE FROM commands WHERE device_id IN (SELECT id FROM devices WHERE project_id=?)`, p.ID)
			continue
		}
		s.db.Exec(`DELETE FROM `+t+` WHERE project_id=?`, p.ID)
	}
	s.db.Exec(`DELETE FROM projects WHERE id=?`, p.ID)
	s.sched.Reload(p.ID)
	w.WriteHeader(204)
}

func (s *Server) apiRotateKey(w http.ResponseWriter, r *http.Request, p *Project) {
	key := randomToken("fcc_", 24)
	s.db.Exec(`UPDATE projects SET project_key=? WHERE id=?`, key, p.ID)
	writeJSON(w, map[string]string{"project_key": key})
}

func (s *Server) apiCreateDashboard(w http.ResponseWriter, r *http.Request, p *Project) {
	var in struct{ Name string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = "Nouveau tableau de bord"
	}
	var pos int
	s.db.QueryRow(`SELECT IFNULL(MAX(position),0)+1 FROM dashboards WHERE project_id=?`, p.ID).Scan(&pos)
	res, err := s.db.Exec(`INSERT INTO dashboards(project_id, name, layout, position, created_at) VALUES(?,?,?,?,?)`, p.ID, in.Name, "[]", pos, now())
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	d, _ := s.getDashboard(id)
	writeJSON(w, d)
}

func (s *Server) apiUpdateDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.getDashboard(pathID(r, "id"))
	if err != nil {
		jsonError(w, 404, "tableau de bord introuvable")
		return
	}
	var in struct {
		Name   *string         `json:"name"`
		Layout json.RawMessage `json:"layout"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		s.db.Exec(`UPDATE dashboards SET name=? WHERE id=?`, strings.TrimSpace(*in.Name), d.ID)
	}
	if len(in.Layout) > 0 {
		if !json.Valid(in.Layout) {
			jsonError(w, 400, "layout invalide")
			return
		}
		s.db.Exec(`UPDATE dashboards SET layout=? WHERE id=?`, string(in.Layout), d.ID)
	}
	d, _ = s.getDashboard(d.ID)
	writeJSON(w, d)
}

func (s *Server) apiDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	s.db.Exec(`DELETE FROM dashboards WHERE id=?`, pathID(r, "id"))
	w.WriteHeader(204)
}

func (s *Server) apiListMembers(w http.ResponseWriter, r *http.Request, p *Project) {
	rows, _ := s.db.Query(`SELECT user_id FROM project_members WHERE project_id=?`, p.ID)
	ids := []int64{}
	if rows != nil {
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
	}
	writeJSON(w, ids)
}

func (s *Server) apiSetMembers(w http.ResponseWriter, r *http.Request, p *Project) {
	var ids []int64
	if !decodeJSON(w, r, &ids) {
		return
	}
	s.db.Exec(`DELETE FROM project_members WHERE project_id=?`, p.ID)
	for _, id := range ids {
		s.db.Exec(`INSERT OR IGNORE INTO project_members(project_id, user_id) VALUES(?,?)`, p.ID, id)
	}
	w.WriteHeader(204)
}

// ---- appareils ----

func (s *Server) apiListDevices(w http.ResponseWriter, r *http.Request, p *Project) {
	devs, _ := s.listDevices(p.ID)
	if devs == nil {
		devs = []*Device{}
	}
	writeJSON(w, devs)
}

// apiOverview : données du tableau de bord (appareils approuvés + actions + automatisations).
func (s *Server) apiVariables(w http.ResponseWriter, r *http.Request, p *Project) {
	devs, _ := s.listDevices(p.ID)
	writeJSON(w, buildVariables(p, devs))
}

func (s *Server) apiOverview(w http.ResponseWriter, r *http.Request, p *Project) {
	devs, _ := s.listDevices(p.ID)
	approved := []*Device{}
	pending := 0
	for _, d := range devs {
		if d.Status == "approved" {
			approved = append(approved, d)
		} else if d.Status == "pending" {
			pending++
		}
	}
	actions, _ := s.listActions(p.ID)
	autos, _ := s.listAutomations(p.ID)
	for _, a := range autos {
		if a.Enabled {
			a.NextRun = s.sched.NextRun(p.ID, a.Cron)
		}
	}
	if actions == nil {
		actions = []*Action{}
	}
	if autos == nil {
		autos = []*Automation{}
	}
	writeJSON(w, map[string]any{"devices": approved, "pending": pending, "actions": actions, "automations": autos, "now": time.Now().UTC()})
}

func (s *Server) apiApproveDevice(w http.ResponseWriter, r *http.Request, d *Device) {
	if err := s.approveDevice(r.Context(), d); err != nil {
		jsonError(w, 502, "Headscale : "+err.Error())
		return
	}
	log.Printf("[portail] %s approuve l'appareil %s", currentUser(r).Username, d.Slug)
	writeJSON(w, map[string]string{"status": "approved"})
}

// apiRekeyDevice émet une nouvelle clé de pré-auth pour un appareil déjà approuvé.
//
// La clé n'est livrée qu'une fois puis effacée : un agent qui perd son état local
// (réinstallation, dossier d'état vidé) se retrouve approuvé sans clé, et son seul recours
// était de révoquer puis ré-approuver — ce qui le sort du réseau privé au passage.
func (s *Server) apiRekeyDevice(w http.ResponseWriter, r *http.Request, d *Device) {
	if d.Status != "approved" {
		jsonError(w, 400, "l'appareil doit être approuvé")
		return
	}
	if !s.hs.enabled() {
		jsonError(w, 503, "Headscale n'est pas configuré sur ce hub : aucune clé ne peut être émise")
		return
	}
	hctx, cancel := context.WithTimeout(r.Context(), hsTimeout)
	defer cancel()
	key, err := s.hs.PreauthKey(hctx, false, 24*time.Hour)
	if err != nil {
		jsonError(w, 502, "Headscale : "+err.Error())
		return
	}
	if _, err := s.db.Exec(`UPDATE devices SET headscale_key=?, headscale_key_delivered=0 WHERE id=?`, key, d.ID); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	s.logEvent(d.ProjectID, &d.ID, "device.rekey", "info", fmt.Sprintf("Nouvelle clé réseau émise pour « %s »", d.Name))
	log.Printf("[portail] %s émet une nouvelle clé réseau pour %s", currentUser(r).Username, d.Slug)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) apiSaveDeviceConfig(w http.ResponseWriter, r *http.Request, d *Device) {
	var cfg DeviceConfig
	if !decodeJSON(w, r, &cfg) {
		return
	}
	cfg.normalize()
	if cfg.Name == "" {
		cfg.Name = d.Slug
	}
	if cfg.HeartbeatSeconds < 5 {
		cfg.HeartbeatSeconds = 15
	}
	names := map[string]bool{}
	for i, sd := range cfg.SubDevices {
		if net.ParseIP(strings.TrimSpace(sd.IP)) == nil {
			jsonError(w, 400, fmt.Sprintf("sous-appareil %q : adresse IP invalide", sd.Name))
			return
		}
		if sd.Port <= 0 || sd.Port > 65535 {
			jsonError(w, 400, fmt.Sprintf("sous-appareil %q : port invalide", sd.Name))
			return
		}
		if sd.Expose && sd.Listen != 0 && (sd.Listen < 1 || sd.Listen > 65535) {
			jsonError(w, 400, fmt.Sprintf("sous-appareil %q : port d'écoute invalide", sd.Name))
			return
		}
		if names[sd.Name] {
			jsonError(w, 400, fmt.Sprintf("deux sous-appareils portent le nom %q", sd.Name))
			return
		}
		names[sd.Name] = true
		cfg.SubDevices[i].IP = strings.TrimSpace(sd.IP)
	}
	cfg.buildForwards() // les accès des sous-appareils exposés + les forwards libres
	if cfg.Update.CheckHours <= 0 {
		cfg.Update.CheckHours = 1
	}
	seen := map[string]string{}
	for i, f := range cfg.Forwards {
		f.Proto = trimLower(f.Proto)
		if f.Proto == "" {
			f.Proto = "tcp"
		}
		if f.Proto != "tcp" && f.Proto != "udp" {
			jsonError(w, 400, fmt.Sprintf("forward %q : proto tcp ou udp", f.Name))
			return
		}
		if f.Listen <= 0 || f.Listen > 65535 || !strings.Contains(f.Target, ":") || f.Name == "" {
			jsonError(w, 400, fmt.Sprintf("forward %d : nom, port d'écoute et cible host:port requis", i+1))
			return
		}
		who := "forward « " + f.Name + " »"
		if f.Sub != "" {
			who = "sous-appareil « " + f.Sub + " »"
		}
		k := fmt.Sprintf("%s/%d", f.Proto, f.Listen)
		if other, dup := seen[k]; dup {
			jsonError(w, 400, fmt.Sprintf("port %d déjà utilisé par %s : changez le port d'écoute de %s", f.Listen, other, who))
			return
		}
		seen[k] = who
		cfg.Forwards[i] = f
	}
	b, _ := json.Marshal(cfg)
	s.db.Exec(`UPDATE devices SET config=?, name=?, config_version=config_version+1 WHERE id=?`, string(b), cfg.Name, d.ID)
	d, _ = s.getDevice(d.ID)
	writeJSON(w, d)
}

// apiDeviceCommand : commandes manuelles (probe, restart, update, http_request).
func (s *Server) apiDeviceCommand(w http.ResponseWriter, r *http.Request, d *Device) {
	var in struct {
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	switch in.Kind {
	case "restart", "update", "probe", "http_request", "scan":
	default:
		jsonError(w, 400, "commande inconnue")
		return
	}
	var payload any = map[string]any{}
	if len(in.Payload) > 0 {
		payload = in.Payload
	}
	id, err := s.queueCommand(d.ID, in.Kind, payload, nil, nil)
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"command_id": id})
}

func (s *Server) apiDeviceCommands(w http.ResponseWriter, r *http.Request, d *Device) {
	rows, _ := s.db.Query(`SELECT id, device_id, kind, payload, status, result, created_at FROM commands WHERE device_id=? ORDER BY id DESC LIMIT 30`, d.ID)
	out := []Command{}
	if rows != nil {
		for rows.Next() {
			var c Command
			var payload string
			rows.Scan(&c.ID, &c.DeviceID, &c.Kind, &payload, &c.Status, &c.Result, &c.CreatedAt)
			c.Payload = json.RawMessage(payload)
			if c.Kind == "scan" {
				if res, summary := parseScan(c.Result, d); res != nil {
					c.Scan, c.Result = res, summary
				}
			}
			out = append(out, c)
		}
		rows.Close()
	}
	writeJSON(w, out)
}

// ---- actions ----

func validateAction(a *Action) error {
	a.Name = strings.TrimSpace(a.Name)
	a.Method = strings.ToUpper(strings.TrimSpace(a.Method))
	if a.Name == "" || a.URL == "" {
		return fmt.Errorf("nom et URL requis")
	}
	if a.Method == "" {
		a.Method = "GET"
	}
	if a.Kind != "agent" {
		a.Kind = "hub"
		a.DeviceID = nil
	} else if a.DeviceID == nil {
		return fmt.Errorf("une action exécutée par un agent doit cibler un écran")
	}
	if a.TimeoutSeconds <= 0 || a.TimeoutSeconds > 120 {
		a.TimeoutSeconds = 10
	}
	if a.Headers == nil {
		a.Headers = map[string]string{}
	}
	return nil
}

func (s *Server) apiSaveAction(w http.ResponseWriter, r *http.Request, p *Project) {
	var a Action
	if !decodeJSON(w, r, &a) {
		return
	}
	if err := validateAction(&a); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	hdr, _ := json.Marshal(a.Headers)
	res, err := s.db.Exec(`INSERT INTO actions(project_id, name, kind, device_id, method, url, headers, body, timeout_seconds) VALUES(?,?,?,?,?,?,?,?,?)`,
		p.ID, a.Name, a.Kind, a.DeviceID, a.Method, a.URL, string(hdr), a.Body, a.TimeoutSeconds)
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	out, _ := s.getAction(id)
	writeJSON(w, out)
}

func (s *Server) apiUpdateAction(w http.ResponseWriter, r *http.Request) {
	cur, err := s.getAction(pathID(r, "id"))
	if err != nil {
		jsonError(w, 404, "action introuvable")
		return
	}
	var a Action
	if !decodeJSON(w, r, &a) {
		return
	}
	if err := validateAction(&a); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	hdr, _ := json.Marshal(a.Headers)
	s.db.Exec(`UPDATE actions SET name=?, kind=?, device_id=?, method=?, url=?, headers=?, body=?, timeout_seconds=? WHERE id=?`,
		a.Name, a.Kind, a.DeviceID, a.Method, a.URL, string(hdr), a.Body, a.TimeoutSeconds, cur.ID)
	out, _ := s.getAction(cur.ID)
	writeJSON(w, out)
}

func (s *Server) apiDeleteAction(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM automations WHERE action_id=?`, id).Scan(&n)
	if n > 0 {
		jsonError(w, 409, "des automatisations utilisent encore cette action")
		return
	}
	s.db.Exec(`DELETE FROM actions WHERE id=?`, id)
	w.WriteHeader(204)
}

func (s *Server) apiRunAction(w http.ResponseWriter, r *http.Request) {
	a, err := s.getAction(pathID(r, "id"))
	if err != nil {
		jsonError(w, 404, "action introuvable")
		return
	}
	if !s.userCanAccess(currentUser(r), a.ProjectID) {
		jsonError(w, 403, "accès refusé")
		return
	}
	res := s.RunAction(r.Context(), a.ID, nil)
	log.Printf("[portail] %s lance l'action %q : %s", currentUser(r).Username, a.Name, res)
	s.logEvent(a.ProjectID, a.DeviceID, "action.run", "info", fmt.Sprintf("%s a lancé « %s » : %s", currentUser(r).Username, a.Name, res))
	writeJSON(w, map[string]string{"result": res})
}

// ---- automatisations ----

func (s *Server) apiListAutomations(w http.ResponseWriter, r *http.Request, p *Project) {
	autos, _ := s.listAutomations(p.ID)
	if autos == nil {
		autos = []*Automation{}
	}
	for _, a := range autos {
		if a.Enabled {
			a.NextRun = s.sched.NextRun(p.ID, a.Cron)
		}
	}
	writeJSON(w, autos)
}

func (s *Server) validateAutomation(a *Automation, projectID int64) error {
	a.Name = strings.TrimSpace(a.Name)
	a.Cron = strings.TrimSpace(a.Cron)
	if a.Name == "" {
		return fmt.Errorf("nom requis")
	}
	if err := ValidateCron(a.Cron); err != nil {
		return fmt.Errorf("horaire cron invalide : %v", err)
	}
	act, err := s.getAction(a.ActionID)
	if err != nil || act.ProjectID != projectID {
		return fmt.Errorf("action introuvable dans ce projet")
	}
	return nil
}

func (s *Server) apiSaveAutomation(w http.ResponseWriter, r *http.Request, p *Project) {
	var a Automation
	if !decodeJSON(w, r, &a) {
		return
	}
	if err := s.validateAutomation(&a, p.ID); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	s.db.Exec(`INSERT INTO automations(project_id, name, enabled, cron, action_id) VALUES(?,?,?,?,?)`, p.ID, a.Name, boolInt(a.Enabled), a.Cron, a.ActionID)
	s.sched.Reload(p.ID)
	w.WriteHeader(201)
}

func (s *Server) apiUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var projectID int64
	if err := s.db.QueryRow(`SELECT project_id FROM automations WHERE id=?`, id).Scan(&projectID); err != nil {
		jsonError(w, 404, "automatisation introuvable")
		return
	}
	var a Automation
	if !decodeJSON(w, r, &a) {
		return
	}
	if err := s.validateAutomation(&a, projectID); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	s.db.Exec(`UPDATE automations SET name=?, enabled=?, cron=?, action_id=? WHERE id=?`, a.Name, boolInt(a.Enabled), a.Cron, a.ActionID, id)
	s.sched.Reload(projectID)
	w.WriteHeader(204)
}

func (s *Server) apiDeleteAutomation(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	var projectID int64
	s.db.QueryRow(`SELECT project_id FROM automations WHERE id=?`, id).Scan(&projectID)
	s.db.Exec(`DELETE FROM automations WHERE id=?`, id)
	s.sched.Reload(projectID)
	w.WriteHeader(204)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// heartbeatHasSubDeviceKO : au moins un sous-appareil injoignable dans le dernier heartbeat
// (champ sub_devices ; processor pour les agents ≤ 0.4).
func heartbeatHasSubDeviceKO(raw json.RawMessage) bool {
	var hb struct {
		SubDevices []struct {
			Reachable bool `json:"reachable"`
		} `json:"sub_devices"`
		Processor *struct {
			Reachable bool `json:"reachable"`
		} `json:"processor"`
	}
	if json.Unmarshal(raw, &hb) != nil {
		return false
	}
	for _, sd := range hb.SubDevices {
		if !sd.Reachable {
			return true
		}
	}
	return len(hb.SubDevices) == 0 && hb.Processor != nil && !hb.Processor.Reachable
}
