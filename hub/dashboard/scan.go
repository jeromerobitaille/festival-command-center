package main

// Résultat de découverte réseau remonté par l'agent (commande « scan »).
// L'agent ne renvoie que des faits bruts ; l'enrichissement — fabricant, adresse
// aléatoire, résumé lisible — se fait ici, à la lecture.

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ScanHost struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Ports    []int  `json:"ports,omitempty"`
	Source   string `json:"source"`
	// ajoutés par le hub
	Vendor string `json:"vendor,omitempty"`
	Random bool   `json:"random_mac,omitempty"`
	Known  string `json:"known,omitempty"` // nom du sous-appareil déjà déclaré à cette IP
}

type ScanResult struct {
	CIDR     string     `json:"cidr"`
	Hosts    []ScanHost `json:"hosts"`
	Scanned  int        `json:"scanned"`
	Duration float64    `json:"duration_seconds"`
}

// parseScan enrichit un résultat de scan et produit un résumé d'une ligne.
// Renvoie nil si le contenu n'est pas un résultat de scan exploitable.
func parseScan(raw string, d *Device) (*ScanResult, string) {
	if raw == "" || raw[0] != '{' {
		return nil, ""
	}
	var res ScanResult
	if json.Unmarshal([]byte(raw), &res) != nil || res.CIDR == "" {
		return nil, ""
	}
	declared := map[string]string{}
	for _, sd := range d.Config.SubDevices {
		if sd.IP != "" {
			declared[sd.IP] = sd.Name
		}
	}
	nouveaux := 0
	for i := range res.Hosts {
		h := &res.Hosts[i]
		h.Random = MACIsRandom(h.MAC)
		h.Vendor = VendorOf(h.MAC)
		h.Known = declared[h.IP]
		if h.Known == "" {
			nouveaux++
		}
		sort.Ints(h.Ports)
	}
	// Les hôtes déjà déclarés en premier, puis ceux qui ont des ports ouverts.
	summary := fmt.Sprintf("%d hôtes sur %s (%d non déclarés) · %d adresses en %.0f s",
		len(res.Hosts), res.CIDR, nouveaux, res.Scanned, res.Duration)
	return &res, summary
}
