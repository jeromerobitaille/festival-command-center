// Commande agent : noeud tsnet + forwards vers le processeur + heartbeat, installable comme service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"errors"
	"math/rand/v2"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kardianos/service"

	"github.com/festival/command-center/agent/internal/config"
	"github.com/festival/command-center/agent/internal/forward"
	"github.com/festival/command-center/agent/internal/heartbeat"
	"github.com/festival/command-center/agent/internal/probe"
	"github.com/festival/command-center/agent/internal/sysinfo"
	"github.com/festival/command-center/agent/internal/tunnel"
	"github.com/festival/command-center/agent/internal/update"
)

var version = "dev" // remplacé par -ldflags "-X main.version=..."

const serviceName = "festival-agent"

func usage() {
	fmt.Fprintf(os.Stderr, `agent %s — Festival Command Center

Usage : agent [-config agent.toml] [-v] <commande>

Commandes :
  run         démarre l'agent au premier plan (Ctrl+C pour arrêter)
  install     enregistre l'agent comme service système et le démarre
  uninstall   arrête et retire le service
  start|stop|restart|status   contrôle du service
  check       valide la configuration et teste le processeur, sans rejoindre le réseau
  probe <ip:port>  rejoint le réseau avec un noeud de diagnostic et teste une adresse tailnet
                   (ex. : agent probe 100.64.0.5:37564 pour vérifier le forward d'un autre écran)
  update      vérifie et installe la dernière release GitHub (signée), puis redémarre le service
  version
`, version)
}

func main() {
	cfgPath := flag.String("config", config.DefaultPath(), "chemin de agent.toml")
	verbose := flag.Bool("v", false, "journal détaillé du tunnel")
	force := flag.Bool("force", false, "update : installer même si la version n'est pas plus récente (builds dev)")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}
	cmd := flag.Arg(0)
	if cmd == "version" {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	setupLog(cfg)

	if cmd == "check" {
		runCheck(cfg)
		return
	}
	if cmd == "update" {
		runUpdate(cfg, *force)
		return
	}
	if cmd == "probe" {
		if flag.NArg() < 2 {
			log.Fatal("usage : agent probe <ip:port>")
		}
		runProbe(cfg, flag.Arg(1), *verbose)
		return
	}

	prg := &program{cfg: cfg, verbose: *verbose}
	svcCfg := &service.Config{
		Name:        serviceName,
		DisplayName: "Festival Command Center Agent",
		Description: "Tunnel WireGuard, forwards vers le processeur LED et heartbeat.",
		Arguments:   []string{"-config", cfg.Path, "run"},
		Option: service.KeyValue{
			"OnFailure":              "restart", // Windows : relance si le processus meurt
			"OnFailureDelayDuration": "5s",
			"OnFailureResetPeriod":   60,
			"Restart":                "always", // systemd
			"KeepAlive":              true,     // launchd
		},
	}
	svc, err := service.New(prg, svcCfg)
	if err != nil {
		log.Fatal(err)
	}

	switch cmd {
	case "run":
		// Sous le gestionnaire de services, svc.Run appelle Start/Stop ; en console il attend Ctrl+C.
		if err := svc.Run(); err != nil {
			log.Fatal(err)
		}
	case "install":
		if err := service.Control(svc, "install"); err != nil {
			log.Fatal("install : ", err)
		}
		if err := service.Control(svc, "start"); err != nil {
			log.Fatal("start : ", err)
		}
		fmt.Println("service installé et démarré :", serviceName)
	case "uninstall":
		_ = service.Control(svc, "stop")
		if err := service.Control(svc, "uninstall"); err != nil {
			log.Fatal("uninstall : ", err)
		}
		fmt.Println("service retiré :", serviceName)
	case "start", "stop", "restart":
		if err := service.Control(svc, cmd); err != nil {
			log.Fatal(cmd, " : ", err)
		}
		fmt.Println(cmd, "ok")
	case "status":
		st, err := svc.Status()
		if err != nil {
			log.Fatal("status : ", err)
		}
		fmt.Println(map[service.Status]string{
			service.StatusRunning: "en cours d'exécution",
			service.StatusStopped: "arrêté",
			service.StatusUnknown: "inconnu",
		}[st])
	default:
		usage()
		os.Exit(2)
	}
}

func setupLog(cfg *config.Config) {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	if cfg.Agent.LogFile == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(cfg.Agent.LogFile), 0o755)
	f, err := os.OpenFile(cfg.Agent.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("log_file : %v (journal sur stdout)", err)
		return
	}
	log.SetOutput(f)
}

func runCheck(cfg *config.Config) {
	fmt.Printf("configuration OK : %s\n", cfg.Path)
	fmt.Printf("écran      : %s (%s)\n", cfg.Screen.ID, cfg.Screen.Name)
	fmt.Printf("hub        : %s\n", cfg.Hub.ControlURL)
	fmt.Printf("state dir  : %s\n", cfg.Agent.StateDir)
	for _, f := range cfg.Forwards {
		fmt.Printf("forward    : %-16s %s :%d -> %s\n", f.Name, f.Proto, f.Listen, f.Target)
	}
	if cfg.Processor.IP != "" {
		r := probe.TCP(context.Background(), cfg.Processor.IP, cfg.Processor.Port)
		if r.Reachable {
			fmt.Printf("processeur : %s joignable (%.1f ms)\n", r.Target, r.RTTMS)
		} else {
			fmt.Printf("processeur : %s INJOIGNABLE : %s\n", r.Target, r.Error)
		}
	}
	if id := sysinfo.RustDeskID(context.Background()); id != "" {
		fmt.Printf("rustdesk   : id %s\n", id)
	} else {
		fmt.Println("rustdesk   : non détecté")
	}
}

// runUpdate : mise à jour manuelle (même chaîne que la mise à jour automatique).
func runUpdate(cfg *config.Config, force bool) {
	ctx := context.Background()
	opts := update.Options{BaseURL: cfg.Update.BaseURL, Current: version, Force: force, Logf: log.Printf}
	rel, err := update.Check(ctx, opts)
	if err != nil {
		if rel != nil && errors.Is(err, update.ErrUpToDate) {
			fmt.Printf("version courante %s, dernière release %s : %v\n", version, rel.Version, err)
			return
		}
		log.Fatal(err)
	}
	fmt.Printf("mise à jour %s -> %s (%s)\n", version, rel.Version, rel.Asset)
	if err := update.Apply(ctx, opts, rel); err != nil {
		log.Fatal(err)
	}
	fmt.Println("binaire remplacé ; redémarrage du service si installé")
	exe, _ := os.Executable()
	if err := update.SpawnRestart(exe, cfg.Path); err != nil {
		log.Printf("redémarrage automatique impossible (%v) : relancer le service à la main", err)
	}
}

// updateLoop vérifie périodiquement les releases et, si une version plus récente existe, l'installe
// puis redémarre le service. Retourne quand une mise à jour a été appliquée.
func updateLoop(ctx context.Context, cfg *config.Config) {
	if !cfg.Update.IsEnabled() || version == "dev" {
		return
	}
	// premier contrôle après 1 à 3 minutes, pour ne pas saturer au démarrage simultané des écrans
	first := time.Minute + time.Duration(rand.IntN(120))*time.Second
	every := time.Duration(cfg.Update.CheckHours) * time.Hour
	t := time.NewTimer(first)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		t.Reset(every)
		opts := update.Options{BaseURL: cfg.Update.BaseURL, Current: version, Logf: log.Printf}
		rel, err := update.Check(ctx, opts)
		if err != nil {
			if !errors.Is(err, update.ErrUpToDate) {
				log.Printf("[update] vérification : %v", err)
			}
			continue
		}
		log.Printf("[update] nouvelle version %s (courante %s) : installation", rel.Version, version)
		if err := update.Apply(ctx, opts, rel); err != nil {
			log.Printf("[update] échec : %v", err)
			continue
		}
		exe, _ := os.Executable()
		if service.Interactive() {
			log.Printf("[update] version %s installée : relancer l'agent", rel.Version)
			return
		}
		if err := update.SpawnRestart(exe, cfg.Path); err != nil {
			log.Printf("[update] version %s installée mais redémarrage impossible : %v", rel.Version, err)
		}
		return
	}
}

// runProbe rejoint le réseau comme noeud éphémère et tente une connexion TCP via le tunnel.
func runProbe(cfg *config.Config, target string, verbose bool) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("connexion à %s (noeud de diagnostic %s-probe)...\n", cfg.Hub.ControlURL, cfg.Screen.ID)
	tn, err := tunnel.Start(ctx, tunnel.Options{
		Hostname:   cfg.Screen.ID + "-probe",
		ControlURL: cfg.Hub.ControlURL,
		AuthKey:    cfg.Hub.AuthKey,
		StateDir:   filepath.Join(cfg.Agent.StateDir, "probe"), // état conservé : un seul noeud "-probe" par écran
		Verbose:    verbose,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tn.Close()
	fmt.Printf("ip tailnet %s\n", tn.IPv4())
	if peers, err := tn.PeerPaths(ctx); err == nil {
		for _, p := range peers {
			if !p.Online {
				continue
			}
			path := "relais " + p.Relay
			if p.Direct {
				path = "direct " + p.Endpoint
			}
			fmt.Printf("pair %-20s %-16s %s  %.1f ms %s\n", p.Hostname, p.IP, path, p.LatencyMS, p.PingError)
		}
	}
	start := time.Now()
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	c, err := tn.Dial(dctx, "tcp", target)
	if err != nil {
		fmt.Printf("ÉCHEC %s : %v\n", target, err)
		os.Exit(1)
	}
	defer c.Close()
	fmt.Printf("OK %s joignable via le tunnel en %.1f ms\n", target, float64(time.Since(start).Microseconds())/1000)
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	if n, _ := c.Read(buf); n > 0 {
		fmt.Printf("bannière : %q\n", strings.TrimSpace(string(buf[:n])))
	}
}

// program implémente service.Interface.
type program struct {
	cfg     *config.Config
	verbose bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func (p *program) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		p.loop(ctx)
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	select {
	case <-p.done:
	case <-time.After(10 * time.Second):
	}
	return nil
}

// loop relance l'agent tant que le contexte est vivant (perte réseau, hub indisponible, etc.).
func (p *program) loop(ctx context.Context) {
	if service.Interactive() {
		sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		ctx = sigCtx
	}
	backoff := 2 * time.Second
	for {
		start := time.Now()
		err := runOnce(ctx, p.cfg, p.verbose)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[agent] arrêt inattendu : %v ; redémarrage dans %s", err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if time.Since(start) > time.Minute {
			backoff = 2 * time.Second
		} else if backoff < time.Minute {
			backoff *= 2
		}
	}
}

func runOnce(ctx context.Context, cfg *config.Config, verbose bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	update.CleanupOld("")
	go updateLoop(ctx, cfg)

	log.Printf("[agent] %s v%s : connexion à %s", cfg.Screen.ID, version, cfg.Hub.ControlURL)
	tn, err := tunnel.Start(ctx, tunnel.Options{
		Hostname:   cfg.Screen.ID,
		ControlURL: cfg.Hub.ControlURL,
		AuthKey:    cfg.Hub.AuthKey,
		StateDir:   cfg.Agent.StateDir,
		Verbose:    verbose,
	})
	if err != nil {
		return err
	}
	defer tn.Close()
	log.Printf("[agent] connecté, ip tailnet %s", tn.IPv4())

	rules := make([]forward.Rule, 0, len(cfg.Forwards))
	for _, f := range cfg.Forwards {
		rules = append(rules, forward.Rule{Name: f.Name, Proto: f.Proto, Listen: f.Listen, Target: f.Target})
	}
	fwds, err := forward.StartAll(ctx, tn, rules)
	if err != nil {
		return err
	}

	rdID := cfg.RemoteDesktop.ID
	if rdID == "" {
		rdID = sysinfo.RustDeskID(ctx)
	}

	sender := heartbeat.NewSender(cfg.Hub.APIURL, cfg.Hub.APIToken)
	build := func(ctx context.Context) heartbeat.Payload {
		p := heartbeat.Payload{
			ScreenID: cfg.Screen.ID, ScreenName: cfg.Screen.Name,
			AgentVersion: version, Timestamp: time.Now().UTC(),
			TailnetIP: tn.IPv4().String(),
		}
		p.RemoteDesktop.Provider = cfg.RemoteDesktop.Provider
		p.RemoteDesktop.ID = rdID
		if cfg.Processor.IP != "" {
			p.Processor = probe.TCP(ctx, cfg.Processor.IP, cfg.Processor.Port)
		}
		stats := make([]forward.Stats, 0, len(fwds))
		for _, f := range fwds {
			stats = append(stats, f.Stats())
		}
		p.Forwards = stats
		if peers, err := tn.PeerPaths(ctx); err == nil {
			p.Peers = peers
		}
		p.System = sysinfo.Collect(ctx)
		return p
	}
	heartbeat.Run(ctx, time.Duration(cfg.Hub.HeartbeatSeconds)*time.Second, sender, build)
	return ctx.Err()
}
