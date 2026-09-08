package main

import (
	"encoding/json"
	"testing"
)

func testDevice() *Device {
	hb := `{"sub_devices":[{"name":"Processeur LED","target":"192.168.0.10:37564","reachable":true,"rtt_ms":0.94}],
	        "peers":[{"hostname":"hub","online":true,"direct":false,"relay":"mtl"}],
	        "system":{"hostname":"regie.local","os":"windows","cpu_percent":23.6,"mem_percent":79.2,"uptime_seconds":864000},
	        "remote_desktop":{"id":"147430505"}}`
	return &Device{
		ID: 1, Slug: "ecran-01", Name: "Place des Festivals - Nord", Status: "approved",
		Hostname: "PC-01", OS: "windows", Arch: "amd64", AgentVersion: "0.3.0",
		TailnetIP: "100.64.0.3", Online: true, AgeSeconds: 12,
		Config: DeviceConfig{HeartbeatSeconds: 10, SubDevices: []SubDeviceConf{
			{Name: "Processeur LED", IP: "192.168.0.10", Port: 37564, Expose: true, Listen: 37564},
		}},
		LastHeartbeat: json.RawMessage(hb),
	}
}

func TestBuildVariables(t *testing.T) {
	p := &Project{Name: "Festival 2027", Slug: "festival-2027", Timezone: "America/Toronto"}
	off := &Device{ID: 2, Slug: "ecran-02", Name: "Sud", Status: "approved", Online: false}
	pend := &Device{ID: 3, Slug: "ecran-03", Name: "Est", Status: "pending"}
	m := varsMap(buildVariables(p, []*Device{testDevice(), off, pend}))

	want := map[string]string{
		"ecran_01.name":                          "Place des Festivals - Nord",
		"ecran_01.online":                        "oui",
		"ecran_01.ip":                            "100.64.0.3",
		"ecran_01.hostname":                      "regie.local", // le heartbeat prime sur la valeur d'inscription
		"ecran_01.cpu":                           "24",
		"ecran_01.ram":                           "79",
		"ecran_01.uptime":                        "10 j",
		"ecran_01.relayed":                       "oui",
		"ecran_01.rustdesk":                      "147430505",
		"ecran_01.subs_total":                    "1",
		"ecran_01.subs_ko":                       "0",
		"ecran_01.subs.processeur_led.rtt":       "0.9",
		"ecran_01.subs.processeur_led.ip":        "192.168.0.10",
		"ecran_01.subs.processeur_led.port":      "37564",
		"ecran_01.subs.processeur_led.access":    "100.64.0.3:37564",
		"ecran_01.subs.processeur_led.reachable": "oui",
		"ecran_02.online":                        "non",
		"ecran_02.cpu":                           "", // hors ligne : la clé existe mais sans valeur
		"project.devices_total":                  "2",
		"project.devices_online":                 "1",
		"project.devices_offline":                "1",
		"project.pending":                        "1",
	}
	for k, v := range want {
		if got, ok := m[k]; !ok {
			t.Errorf("variable absente : %s", k)
		} else if got != v {
			t.Errorf("%s = %q, attendu %q", k, got, v)
		}
	}
	// Un appareil en attente n'expose pas de variables.
	if _, ok := m["ecran_03.name"]; ok {
		t.Error("un appareil en attente ne doit pas apparaître dans le catalogue")
	}
}

func TestResolveVars(t *testing.T) {
	vars := map[string]string{"a.ip": "10.0.0.1", "a.cpu": ""}
	cases := []struct {
		in, out string
		missing int
	}{
		{"http://{{ a.ip }}/on", "http://10.0.0.1/on", 0},
		{"{{a.ip}} sans espaces", "10.0.0.1 sans espaces", 0},
		{"CPU {{ a.cpu }} %", "CPU  %", 0}, // connue mais vide : pas une erreur
		{"{{ a.oops }}", "{{ a.oops }}", 1},
		{"rien à substituer", "rien à substituer", 0},
		{"{{ a.ip }} et {{ a.ip }}", "10.0.0.1 et 10.0.0.1", 0},
	}
	for _, c := range cases {
		got, miss := resolveVars(c.in, vars)
		if got != c.out {
			t.Errorf("resolveVars(%q) = %q, attendu %q", c.in, got, c.out)
		}
		if len(miss) != c.missing {
			t.Errorf("resolveVars(%q) : %d manquantes, attendu %d", c.in, len(miss), c.missing)
		}
	}
}

func TestVarKey(t *testing.T) {
	for in, want := range map[string]string{
		"ecran-01": "ecran_01", "Processeur LED": "processeur_led",
		"Écran Nord": "ecran_nord", "Régie – Façade": "regie_facade", "Projecteur n°2": "projecteur_n_2", "": "",
	} {
		if got := varKey(in); got != want {
			t.Errorf("varKey(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestSplitTarget(t *testing.T) {
	h, p := splitTarget("192.168.0.10:37564")
	if h != "192.168.0.10" || p != "37564" {
		t.Errorf("splitTarget = %q, %q", h, p)
	}
	if h, p := splitTarget(""); h != "" || p != "" {
		t.Errorf("cible vide = %q, %q", h, p)
	}
}

// Une valeur remontée par un agent ne doit pas pouvoir détourner la requête du hub.
func TestResolvedURLSchemeGuard(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "javascript:alert(1)", "pas une url du tout"} {
		u, err := parseAndCheck(raw)
		if err == nil {
			t.Errorf("%q aurait dû être refusée (obtenu %q)", raw, u)
		}
	}
	if _, err := parseAndCheck("http://10.0.0.1:8080/api"); err != nil {
		t.Errorf("URL http refusée à tort : %v", err)
	}
}
