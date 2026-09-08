// Package scan découvre les équipements présents sur le réseau local de l'agent.
//
// Deux sources complémentaires, sans dépendance externe ni socket brut :
//   - un balayage TCP sur une liste de ports, qui identifie les services ouverts ;
//   - la table ARP du système, relue après une sonde UDP, qui révèle les équipements
//     muets en TCP et fournit leur adresse MAC (donc le fabricant).
package scan

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultPorts : contrôle d'équipements scéniques, administration web, accès distant.
// Volontairement court — un balayage large est lent et bruyant sur un réseau de salle.
var DefaultPorts = []int{22, 23, 80, 443, 502, 4352, 5900, 6454, 8000, 8080, 8443, 9090, 21063, 37564}

// MaxHosts borne le balayage : au-delà, on refuse plutôt que de marteler le réseau.
const MaxHosts = 1024

type Host struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Ports    []int  `json:"ports,omitempty"`
	Source   string `json:"source"` // tcp, arp, ou tcp+arp
}

type Result struct {
	CIDR     string  `json:"cidr"`
	Hosts    []Host  `json:"hosts"`
	Scanned  int     `json:"scanned"`
	Duration float64 `json:"duration_seconds"`
}

// LocalNet renvoie le premier réseau IPv4 privé de la machine, hors loopback et hors
// interface du tunnel (100.64.0.0/10, plage CGNAT utilisée par le réseau privé).
func LocalNet() (*net.IPNet, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, "", err
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok || n.IP.To4() == nil || !n.IP.IsPrivate() || IsTailnet(n.IP) {
				continue
			}
			return n, ifc.Name, nil
		}
	}
	return nil, "", fmt.Errorf("aucune interface locale IPv4 privée trouvée")
}

// IsTailnet écarte la plage CGNAT du réseau privé : la scanner n'a pas de sens ici.
func IsTailnet(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}

// Run balaye le réseau indiqué (vide = réseau local détecté automatiquement).
func Run(ctx context.Context, cidr string, ports []int, timeout time.Duration) (*Result, error) {
	if len(ports) == 0 {
		ports = DefaultPorts
	}
	if timeout <= 0 {
		timeout = 600 * time.Millisecond
	}
	var ipnet *net.IPNet
	if cidr == "" {
		n, _, err := LocalNet()
		if err != nil {
			return nil, err
		}
		ipnet = n
	} else {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("réseau invalide : %w", err)
		}
		ipnet = n
	}

	targets := hosts(ipnet)
	if len(targets) == 0 {
		return nil, fmt.Errorf("aucune adresse à balayer dans %s", ipnet)
	}
	if len(targets) > MaxHosts {
		return nil, fmt.Errorf("%s contient %d adresses, au-delà de la limite de %d : indiquez un sous-réseau plus petit", ipnet, len(targets), MaxHosts)
	}

	start := time.Now()
	found := map[string]*Host{}
	var mu sync.Mutex
	sem := make(chan struct{}, 128)
	var wg sync.WaitGroup

	for _, ip := range targets {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Sonde UDP : aucune réponse n'est attendue, seul compte l'effet de bord
			// sur la table ARP, relue plus bas.
			nudge(ip)
			var open []int
			for _, p := range ports {
				if ctx.Err() != nil {
					return
				}
				if dialOK(ctx, ip, p, timeout) {
					open = append(open, p)
				}
			}
			if len(open) == 0 {
				return
			}
			mu.Lock()
			found[ip] = &Host{IP: ip, Ports: open, Source: "tcp"}
			mu.Unlock()
		}(ip)
	}
	wg.Wait()

	// Les équipements muets en TCP figurent tout de même dans la table ARP, puisqu'ils
	// ont répondu à la résolution d'adresse déclenchée par la sonde.
	for ip, mac := range arpTable(ctx) {
		p := net.ParseIP(ip)
		if p == nil || !ipnet.Contains(p) {
			continue
		}
		if h, ok := found[ip]; ok {
			h.MAC, h.Source = mac, "tcp+arp"
			continue
		}
		found[ip] = &Host{IP: ip, MAC: mac, Source: "arp"}
	}

	out := make([]Host, 0, len(found))
	for _, h := range found {
		h.Hostname = reverseDNS(ctx, h.IP)
		out = append(out, *h)
	}
	sort.Slice(out, func(i, j int) bool { return ipLess(out[i].IP, out[j].IP) })
	return &Result{CIDR: ipnet.String(), Hosts: out, Scanned: len(targets), Duration: time.Since(start).Seconds()}, nil
}

func dialOK(ctx context.Context, ip string, port int, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func nudge(ip string) {
	c, err := net.DialTimeout("udp", net.JoinHostPort(ip, "9"), 200*time.Millisecond)
	if err != nil {
		return
	}
	c.Write([]byte{0})
	c.Close()
}

func reverseDNS(ctx context.Context, ip string) string {
	cctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(cctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

var (
	reIP  = regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3})\b`)
	reMAC = regexp.MustCompile(`\b([0-9a-fA-F]{1,2}(?:[:-][0-9a-fA-F]{1,2}){5})\b`)
)

// arpTable lit le cache ARP du système. Les formats diffèrent selon l'OS ; plutôt que de
// les modéliser, on extrait la première IP et la première MAC de chaque ligne.
func arpTable(ctx context.Context) map[string]string {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Mode numérique impératif : sur BSD/macOS, « arp -a » résout le DNS inverse de chaque
	// entrée, ce qui prend plusieurs secondes une fois le cache rempli par le balayage.
	// Windows n'accepte pas -n et n'en a pas besoin (sortie déjà numérique).
	cmds := [][]string{{"arp", "-a", "-n"}}
	switch runtime.GOOS {
	case "windows":
		cmds = [][]string{{"arp", "-a"}}
	case "linux":
		cmds = [][]string{{"ip", "neigh"}, {"arp", "-a", "-n"}}
	}
	out := map[string]string{}
	for _, c := range cmds {
		b, err := exec.CommandContext(cctx, c[0], c[1:]...).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			ip, mac := reIP.FindString(line), reMAC.FindString(line)
			if ip == "" || mac == "" {
				continue
			}
			if m := NormalizeMAC(mac); m != "" && m != "00:00:00:00:00:00" && m != "ff:ff:ff:ff:ff:ff" {
				out[ip] = m
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}

// NormalizeMAC uniformise « 0:1B:c5:1:2:3 » et « 00-1b-c5-01-02-03 » en « 00:1b:c5:01:02:03 ».
func NormalizeMAC(s string) string {
	parts := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != 6 {
		return ""
	}
	for i, p := range parts {
		if len(p) == 1 {
			p = "0" + p
		}
		if len(p) != 2 {
			return ""
		}
		parts[i] = p
	}
	return strings.Join(parts, ":")
}

// hosts énumère les adresses utilisables du réseau, hors adresse réseau et broadcast.
func hosts(n *net.IPNet) []string {
	ones, bits := n.Mask.Size()
	if bits != 32 || ones > 30 {
		return nil
	}
	base := n.IP.Mask(n.Mask).To4()
	count := 1 << uint(32-ones)
	out := make([]string, 0, count-2)
	v0 := uint32(base[0])<<24 | uint32(base[1])<<16 | uint32(base[2])<<8 | uint32(base[3])
	for i := 1; i < count-1; i++ {
		v := v0 + uint32(i)
		out = append(out, net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).String())
	}
	return out
}

func ipLess(a, b string) bool {
	x, y := net.ParseIP(a).To4(), net.ParseIP(b).To4()
	if x == nil || y == nil {
		return a < b
	}
	for i := 0; i < 4; i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}
