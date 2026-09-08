package main

import (
	"encoding/json"
	"testing"
)

func TestParseScan(t *testing.T) {
	// Résultat réellement produit par l'agent sur un réseau local.
	raw := `{"cidr":"10.0.10.0/23","scanned":510,"duration_seconds":23.9,"hosts":[
	 {"ip":"10.0.10.1","mac":"34:60:f9:fd:84:4c","ports":[443,80],"source":"tcp+arp"},
	 {"ip":"10.0.10.24","mac":"c4:38:75:9a:fa:64","source":"arp"},
	 {"ip":"10.0.10.11","mac":"da:02:f4:a3:21:ae","source":"arp"},
	 {"ip":"192.168.0.10","mac":"","ports":[37564],"source":"tcp"}]}`
	d := &Device{Config: DeviceConfig{SubDevices: []SubDeviceConf{{Name: "Processeur LED", IP: "192.168.0.10", Port: 37564}}}}
	res, summary := parseScan(raw, d)
	if res == nil {
		t.Fatal("resultat non reconnu")
	}
	t.Logf("resume : %s", summary)
	for _, h := range res.Hosts {
		t.Logf("  %-13s vendor=%-22q aleatoire=%-5v deja=%q ports=%v", h.IP, h.Vendor, h.Random, h.Known, h.Ports)
	}
	if res.Hosts[0].Vendor == "" {
		t.Error("fabricant non resolu pour une MAC attribuee")
	}
	if !res.Hosts[2].Random || res.Hosts[2].Vendor != "" {
		t.Error("MAC aleatoire mal traitee")
	}
	if res.Hosts[3].Known != "Processeur LED" {
		t.Errorf("sous-appareil deja declare non reconnu : %q", res.Hosts[3].Known)
	}
	if res.Hosts[0].Ports[0] != 80 {
		t.Errorf("ports non tries : %v", res.Hosts[0].Ports)
	}
	// Ce qui n'est pas un scan doit être laissé tel quel.
	if r, _ := parseScan("appareil joignable en 0.9 ms", d); r != nil {
		t.Error("un resultat texte a ete pris pour un scan")
	}
	b, _ := json.Marshal(res)
	t.Logf("JSON servi au navigateur : %d octets", len(b))
}
