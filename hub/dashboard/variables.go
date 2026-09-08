package main

// Catalogue de variables dérivées des appareils, sous-appareils et du projet.
// Construit ici et nulle part ailleurs : le portail web ne fait que substituer sur
// cette table, ce qui évite d'avoir deux implémentations qui divergent.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Variable est une entrée du catalogue, telle qu'affichée dans le sélecteur.
type Variable struct {
	Key   string `json:"key"`   // ecran_01.subs.processeur.rtt
	Value string `json:"value"` // valeur courante, déjà formatée
	Label string `json:"label"` // libellé lisible
	Group string `json:"group"` // regroupement dans le sélecteur
}

// varKey transforme un nom en identifiant utilisable dans {{ … }} : « Écran 01 » → ecran_01.
// Les accents sont translittérés et non supprimés : slugify seul donnerait « cran_01 ».
func varKey(s string) string { return strings.ReplaceAll(slugify(unaccent(s)), "-", "_") }

var accents = strings.NewReplacer(
	"à", "a", "â", "a", "ä", "a", "á", "a", "ã", "a", "å", "a",
	"ç", "c", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"î", "i", "ï", "i", "í", "i", "ì", "i",
	"ô", "o", "ö", "o", "ó", "o", "ò", "o", "õ", "o",
	"ù", "u", "û", "u", "ü", "u", "ú", "u",
	"ÿ", "y", "ñ", "n", "æ", "ae", "œ", "oe",
)

func unaccent(s string) string { return accents.Replace(strings.ToLower(s)) }

// hbVars : sous-ensemble du heartbeat utilisé par les variables.
type hbVars struct {
	SubDevices []struct {
		Name      string  `json:"name"`
		Target    string  `json:"target"`
		Reachable bool    `json:"reachable"`
		RTTMs     float64 `json:"rtt_ms"`
		Error     string  `json:"error"`
	} `json:"sub_devices"`
	Peers []struct {
		Hostname  string  `json:"hostname"`
		Online    bool    `json:"online"`
		Direct    bool    `json:"direct"`
		Relay     string  `json:"relay"`
		LatencyMs float64 `json:"latency_ms"`
	} `json:"peers"`
	System struct {
		Hostname      string  `json:"hostname"`
		OS            string  `json:"os"`
		CPUPercent    float64 `json:"cpu_percent"`
		MemPercent    float64 `json:"mem_percent"`
		UptimeSeconds int64   `json:"uptime_seconds"`
		LANIP         string  `json:"lan_ip"`
		LANCIDR       string  `json:"lan_cidr"`
		LANIface      string  `json:"lan_iface"`
	} `json:"system"`
	RemoteDesktop struct {
		ID string `json:"id"`
	} `json:"remote_desktop"`
}

func yesNo(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

// buildVariables produit le catalogue complet du projet, dans l'ordre d'affichage.
// Les appareils non approuvés sont ignorés : ils n'ont pas d'état exploitable.
func buildVariables(p *Project, devices []*Device) []Variable {
	out := []Variable{}
	add := func(group, key, label, value string) {
		out = append(out, Variable{Key: key, Value: value, Label: label, Group: group})
	}

	online, offline, subsKO, pending := 0, 0, 0, 0
	approved := []*Device{}
	for _, d := range devices {
		switch d.Status {
		case "approved":
			approved = append(approved, d)
			if d.Online {
				online++
			} else {
				offline++
			}
		case "pending":
			pending++
		}
	}

	for _, d := range approved {
		k := varKey(d.Slug)
		if k == "" {
			continue
		}
		g := d.Name
		var hb hbVars
		if len(d.LastHeartbeat) > 0 {
			json.Unmarshal(d.LastHeartbeat, &hb)
		}

		add(g, k+".name", "Nom affiché", d.Name)
		add(g, k+".slug", "Identifiant", d.Slug)
		add(g, k+".online", "En ligne", yesNo(d.Online))
		add(g, k+".status", "Statut", d.Status)
		add(g, k+".ip", "IP réseau privé", d.TailnetIP)
		add(g, k+".lan_ip", "IP réseau local", hb.System.LANIP)
		add(g, k+".lan_cidr", "Sous-réseau local", hb.System.LANCIDR)
		add(g, k+".hostname", "Nom d'hôte", firstNonEmpty(hb.System.Hostname, d.Hostname))
		add(g, k+".os", "Système", firstNonEmpty(hb.System.OS, d.OS))
		add(g, k+".arch", "Architecture", d.Arch)
		add(g, k+".agent", "Version d'agent", d.AgentVersion)
		add(g, k+".heartbeat", "Intervalle heartbeat (s)", fmt.Sprint(d.Config.HeartbeatSeconds))
		add(g, k+".age", "Vu il y a (s)", fmt.Sprint(d.AgeSeconds))
		add(g, k+".last_seen", "Dernier contact", humanAge(d.AgeSeconds))
		add(g, k+".rustdesk", "Identifiant RustDesk", hb.RemoteDesktop.ID)

		// Télémétrie machine : absente tant que l'appareil n'a pas émis de heartbeat.
		if d.Online {
			add(g, k+".cpu", "CPU (%)", fmt.Sprintf("%.0f", hb.System.CPUPercent))
			add(g, k+".ram", "RAM (%)", fmt.Sprintf("%.0f", hb.System.MemPercent))
			add(g, k+".uptime", "Allumée depuis", humanAge(hb.System.UptimeSeconds))
		} else {
			add(g, k+".cpu", "CPU (%)", "")
			add(g, k+".ram", "RAM (%)", "")
			add(g, k+".uptime", "Allumée depuis", "")
		}

		relayed := false
		for _, pr := range hb.Peers {
			if pr.Online && !pr.Direct {
				relayed = true
			}
		}
		add(g, k+".relayed", "Passe par un relais", yesNo(relayed))

		ok, ko := 0, 0
		for _, sd := range hb.SubDevices {
			if sd.Reachable {
				ok++
			} else {
				ko++
			}
		}
		add(g, k+".subs_total", "Sous-appareils", fmt.Sprint(len(hb.SubDevices)))
		add(g, k+".subs_ok", "Sous-appareils joignables", fmt.Sprint(ok))
		add(g, k+".subs_ko", "Sous-appareils injoignables", fmt.Sprint(ko))
		subsKO += ko

		// Sous-appareils : l'état vient du heartbeat, l'adresse d'accès de la config.
		for _, sd := range hb.SubDevices {
			sk := varKey(sd.Name)
			if sk == "" {
				continue
			}
			base := k + ".subs." + sk
			sg := d.Name + " › " + sd.Name
			add(sg, base+".name", "Nom", sd.Name)
			add(sg, base+".target", "Cible locale", sd.Target)
			add(sg, base+".reachable", "Joignable", yesNo(sd.Reachable))
			add(sg, base+".error", "Erreur", sd.Error)
			if sd.Reachable {
				add(sg, base+".rtt", "Latence (ms)", fmt.Sprintf("%.1f", sd.RTTMs))
			} else {
				add(sg, base+".rtt", "Latence (ms)", "")
			}
			host, port := splitTarget(sd.Target)
			add(sg, base+".ip", "IP locale", host)
			add(sg, base+".port", "Port local", port)
			access := ""
			for _, c := range d.Config.SubDevices {
				if c.Name == sd.Name && c.Expose && d.TailnetIP != "" {
					l := c.Listen
					if l == 0 {
						l = c.Port
					}
					access = fmt.Sprintf("%s:%d", d.TailnetIP, l)
				}
			}
			add(sg, base+".access", "Accès réseau privé", access)
		}
	}

	g := "Projet"
	add(g, "project.name", "Nom du projet", p.Name)
	add(g, "project.slug", "Identifiant du projet", p.Slug)
	add(g, "project.timezone", "Fuseau horaire", p.Timezone)
	add(g, "project.devices_total", "Appareils approuvés", fmt.Sprint(len(approved)))
	add(g, "project.devices_online", "Appareils en ligne", fmt.Sprint(online))
	add(g, "project.devices_offline", "Appareils hors ligne", fmt.Sprint(offline))
	add(g, "project.pending", "En attente d'approbation", fmt.Sprint(pending))
	add(g, "project.subs_ko", "Sous-appareils injoignables", fmt.Sprint(subsKO))
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// splitTarget découpe « 192.168.0.10:37564 ». Renvoie deux chaînes vides si la cible est vide.
func splitTarget(t string) (string, string) {
	i := strings.LastIndex(t, ":")
	if i < 0 {
		return t, ""
	}
	return t[:i], t[i+1:]
}

func humanAge(sec int64) string {
	switch {
	case sec <= 0:
		return "—"
	case sec < 60:
		return fmt.Sprintf("%d s", sec)
	case sec < 3600:
		return fmt.Sprintf("%d min", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%d h", sec/3600)
	}
	return fmt.Sprintf("%d j", sec/86400)
}

func varsMap(list []Variable) map[string]string {
	m := make(map[string]string, len(list))
	for _, v := range list {
		m[v.Key] = v.Value
	}
	return m
}

var varRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// resolveVars remplace {{ clé }} par sa valeur. Une clé absente du catalogue est laissée
// telle quelle et signalée : côté action, on préfère refuser plutôt que d'émettre une
// requête vers une URL trouée. Une clé connue mais vide (appareil hors ligne) donne "".
func resolveVars(s string, vars map[string]string) (string, []string) {
	var missing []string
	out := varRe.ReplaceAllStringFunc(s, func(m string) string {
		key := varRe.FindStringSubmatch(m)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		missing = append(missing, key)
		return m
	})
	return out, missing
}

// projectVars construit le catalogue d'un projet.
func (s *Server) projectVars(projectID int64) ([]Variable, error) {
	p, err := s.getProjectByID(projectID)
	if err != nil {
		return nil, err
	}
	devs, err := s.listDevices(projectID)
	if err != nil {
		return nil, err
	}
	return buildVariables(p, devs), nil
}

// resolveAction substitue les variables dans l'URL, les en-têtes et le corps.
// Renvoie une erreur si une variable est inconnue, ou si l'URL résolue n'est plus exploitable :
// une valeur remontée par un agent ne doit pas pouvoir détourner la requête du hub.
func (s *Server) resolveAction(a *Action) (string, map[string]string, string, error) {
	list, err := s.projectVars(a.ProjectID)
	if err != nil {
		return "", nil, "", fmt.Errorf("variables indisponibles : %w", err)
	}
	vars := varsMap(list)
	var missing []string
	collect := func(s string) string {
		out, m := resolveVars(s, vars)
		missing = append(missing, m...)
		return out
	}
	rawURL := collect(a.URL)
	body := collect(a.Body)
	headers := map[string]string{}
	for k, v := range a.Headers {
		headers[k] = collect(v)
	}
	if len(missing) > 0 {
		return "", nil, "", fmt.Errorf("variable inconnue : %s", strings.Join(dedupe(missing), ", "))
	}
	if _, err := parseAndCheck(rawURL); err != nil {
		return "", nil, "", err
	}
	return rawURL, headers, body, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// parseAndCheck valide l'URL obtenue après substitution. Les valeurs proviennent en partie
// des heartbeats : on refuse tout ce qui n'est pas une requête http(s) vers un hôte.
func parseAndCheck(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("URL invalide après substitution : %s", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("URL invalide après substitution (schéma %q attendu http ou https) : %s", u.Scheme, raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL sans hôte après substitution : %s", raw)
	}
	return raw, nil
}
