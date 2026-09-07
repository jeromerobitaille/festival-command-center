// Commande agent : inscription au portail, tunnel tsnet, forwards pilotés à distance, heartbeat, commandes.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kardianos/service"

	"github.com/festival/command-center/agent/internal/config"
	"github.com/festival/command-center/agent/internal/forward"
	"github.com/festival/command-center/agent/internal/heartbeat"
	"github.com/festival/command-center/agent/internal/localapi"
	"github.com/festival/command-center/agent/internal/portal"
	"github.com/festival/command-center/agent/internal/probe"
	"github.com/festival/command-center/agent/internal/status"
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
  check       valide la configuration locale et teste le processeur, sans contacter le portail
  probe <ip:port>  rejoint le réseau avec un noeud de diagnostic et teste une adresse du réseau privé
  update      vérifie et installe la dernière release GitHub (signée), puis redémarre le service
  reset       oublie l'inscription (jeton, réseau) : l'écran devra être ré-approuvé
  panel       ouvre le panneau local (statut, clé de projet, configuration) dans le navigateur
  version

Sans commande : lance l'icône de la barre des tâches (agent-tray) si elle est présente.
`, version)
}

func main() {
	cfgPath := flag.String("config", config.DefaultPath(), "chemin de agent.toml")
	verbose := flag.Bool("v", false, "journal détaillé du tunnel")
	force := flag.Bool("force", false, "update : installer même si la version n'est pas plus récente (builds dev)")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		if launchTray() {
			return
		}
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

	switch cmd {
	case "check":
		runCheck(cfg)
		return
	case "update":
		runUpdate(cfg, *force)
		return
	case "probe":
		if flag.NArg() < 2 {
			log.Fatal("usage : agent probe <ip:port>")
		}
		runProbe(cfg, flag.Arg(1), *verbose)
		return
	case "reset":
		resetEnrollment(cfg)
		fmt.Println("inscription oubliée : au prochain démarrage l'agent se réinscrira et attendra une approbation")
		return
	case "panel":
		if err := openBrowser("http://" + cfg.Agent.LocalAddr + "/"); err != nil {
			log.Fatal(err)
		}
		return
	}

	prg := &program{cfg: cfg, verbose: *verbose}
	svcCfg := &service.Config{
		Name:        serviceName,
		DisplayName: "Festival Command Center Agent",
		Description: "Tunnel WireGuard, forwards vers le processeur LED, heartbeat et commandes du portail.",
		Arguments:   []string{"-config", cfg.Path, "run"},
		Option: service.KeyValue{
			"OnFailure": "restart", "OnFailureDelayDuration": "5s", "OnFailureResetPeriod": 60, // Windows
			"Restart":   "always", // systemd
			"KeepAlive": true,     // launchd
		},
	}
	svc, err := service.New(prg, svcCfg)
	if err != nil {
		log.Fatal(err)
	}

	switch cmd {
	case "run":
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
		fmt.Println("→ approuver maintenant l'écran dans le portail :", cfg.Portal.URL)
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
		fmt.Println(map[service.Status]string{service.StatusRunning: "en cours d'exécution", service.StatusStopped: "arrêté", service.StatusUnknown: "inconnu"}[st])
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
	fmt.Printf("portail    : %s\n", cfg.Portal.URL)
	fmt.Printf("state dir  : %s\n", cfg.Agent.StateDir)
	if st, err := portal.LoadState(cfg.Agent.StateDir); err == nil {
		fmt.Printf("inscription: appareil #%d, réseau %s\n", st.DeviceID, st.ControlURL)
	} else {
		fmt.Println("inscription: aucune (se fera au premier démarrage)")
	}
	for _, f := range cfg.Forwards {
		fmt.Printf("forward    : %-16s %s :%d -> %s (proposition)\n", f.Name, f.Proto, f.Listen, f.Target)
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

// ---- mise à jour ----

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

// applyUpdate : utilisé par la boucle périodique et la commande "update" du portail.
func applyUpdate(ctx context.Context, cfg *config.Config, force bool) (string, error) {
	opts := update.Options{BaseURL: cfg.Update.BaseURL, Current: version, Force: force, Logf: log.Printf}
	rel, err := update.Check(ctx, opts)
	if err != nil {
		return "", err
	}
	if err := update.Apply(ctx, opts, rel); err != nil {
		return "", err
	}
	exe, _ := os.Executable()
	if !service.Interactive() {
		if err := update.SpawnRestart(exe, cfg.Path); err != nil {
			return "", fmt.Errorf("version %s installée mais redémarrage impossible : %v", rel.Version, err)
		}
	}
	return rel.Version, nil
}

// ---- service ----

type program struct {
	cfg     *config.Config
	verbose bool
	cancel  context.CancelFunc
	done    chan struct{}
	reload  chan struct{} // demande de relecture de agent.toml
	svc     service.Service
}

func (p *program) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.svc = s
	p.done = make(chan struct{})
	p.reload = make(chan struct{}, 1)
	go func() {
		if err := localapi.Serve(ctx, p.cfg.Agent.LocalAddr, p.cfg.Path, p); err != nil {
			log.Printf("[local] %v", err)
		}
	}()
	go func() {
		defer close(p.done)
		p.loop(ctx)
	}()
	return nil
}

// Reload / ResetEnrollment / Restart : implémentation de localapi.Controller.
func (p *program) Reload() {
	select {
	case p.reload <- struct{}{}:
	default:
	}
}

func (p *program) ResetEnrollment() {
	resetEnrollment(p.cfg)
	p.Reload()
}

func (p *program) Restart() {
	exe, _ := os.Executable()
	if service.Interactive() {
		log.Printf("[local] redémarrage demandé en mode interactif : arrêt")
		os.Exit(0)
	}
	if err := update.SpawnRestart(exe, p.cfg.Path); err != nil {
		log.Printf("[local] redémarrage impossible : %v", err)
	}
}

func resetEnrollment(cfg *config.Config) {
	os.Remove(filepath.Join(cfg.Agent.StateDir, "device.json"))
	os.RemoveAll(filepath.Join(cfg.Agent.StateDir, "tsnet"))
	status.Global.Update(func(x *status.Snapshot) { x.TailnetIP, x.DeviceID = "", 0 })
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

func (p *program) loop(ctx context.Context) {
	if service.Interactive() {
		sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		ctx = sigCtx
	}
	update.CleanupOld("")
	backoff := 2 * time.Second
	for {
		// Relire agent.toml à chaque cycle : le panneau local peut l'avoir modifié.
		if cfg, err := config.Load(p.cfg.Path); err == nil {
			p.cfg = cfg
		} else {
			log.Printf("[agent] %v", err)
		}
		start := time.Now()
		runCtx, cancelRun := context.WithCancel(ctx)
		reloaded := make(chan struct{})
		go func() {
			select {
			case <-p.reload:
				log.Printf("[agent] configuration modifiée : redémarrage du cycle")
				cancelRun()
			case <-reloaded:
			}
		}()
		err := runOnce(runCtx, p.cfg, p.verbose)
		close(reloaded)
		cancelRun()
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, context.Canceled) {
			backoff = time.Second
		} else {
			log.Printf("[agent] arrêt inattendu : %v ; redémarrage dans %s", err, backoff)
			status.Global.SetPhase("error", err.Error())
		}
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

// ---- cycle de vie principal ----

// agentState regroupe ce que les goroutines partagent.
type agentState struct {
	cfg     *config.Config
	pc      *portal.Client
	tn      *tunnel.Tunnel
	mu      sync.Mutex
	remote  portal.DeviceConfig
	version int64
	fwds    []*forward.Forwarder
	fwdStop context.CancelFunc
}

func runOnce(ctx context.Context, cfg *config.Config, verbose bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	status.Global.Update(func(x *status.Snapshot) {
		x.Version, x.ScreenID, x.Name, x.PortalURL = version, cfg.Screen.ID, cfg.Screen.Name, cfg.Portal.URL
		x.HasProjectKey, x.ConfigPath = cfg.Portal.ProjectKey != "", cfg.Path
		x.TailnetIP, x.ProcessorOK, x.LastHeartbeat = "", nil, time.Time{}
	})

	// 0. Sans clé de projet, on attend qu'elle soit saisie (panneau local ou agent.toml).
	if cfg.Portal.ProjectKey == "" {
		if _, err := portal.LoadState(cfg.Agent.StateDir); err != nil {
			status.Global.SetPhase("no-key", "Aucune clé de projet : ouvrir le panneau local pour rattacher l'écran à un projet")
			log.Printf("[agent] aucune clé de projet ; panneau : http://%s", cfg.Agent.LocalAddr)
			<-ctx.Done()
			return ctx.Err()
		}
	}

	// 1. Inscription (ou reprise) auprès du portail.
	st, err := portal.LoadState(cfg.Agent.StateDir)
	if err != nil {
		status.Global.SetPhase("enrolling", "Inscription auprès du portail…")
		st, err = enroll(ctx, cfg)
		if err != nil {
			return err
		}
	}
	status.Global.Update(func(x *status.Snapshot) { x.DeviceID = st.DeviceID })
	pc := portal.New(cfg.Portal.URL, st.Token)

	// 2. Attente de l'approbation. Le portail livre la clé Headscale une seule fois.
	var authKey string
	for {
		s, err := pc.Status(ctx)
		if errors.Is(err, portal.ErrUnauthorized) {
			log.Printf("[agent] jeton refusé (appareil supprimé ?) : nouvelle inscription")
			os.Remove(filepath.Join(cfg.Agent.StateDir, "device.json"))
			return err
		}
		if err != nil {
			return fmt.Errorf("portail : %w", err)
		}
		if s.Status == "approved" {
			if s.ControlURL != "" && s.ControlURL != st.ControlURL {
				st.ControlURL = s.ControlURL
				portal.SaveState(cfg.Agent.StateDir, st)
			}
			authKey = s.AuthKey
			break
		}
		log.Printf("[agent] appareil %s : %s — en attente d'approbation dans %s", cfg.Screen.ID, s.Status, cfg.Portal.URL)
		status.Global.SetPhase("pending", map[string]string{"pending": "En attente d'approbation dans le portail", "rejected": "Refusé dans le portail", "revoked": "Révoqué dans le portail"}[s.Status])
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	if !st.Joined && authKey == "" {
		return fmt.Errorf("approuvé mais aucune clé réseau reçue : révoquer puis ré-approuver l'écran dans le portail")
	}

	// 3. Tunnel.
	status.Global.SetPhase("connecting", "Connexion au réseau privé…")
	log.Printf("[agent] %s v%s : connexion au réseau %s", cfg.Screen.ID, version, st.ControlURL)
	tn, err := tunnel.Start(ctx, tunnel.Options{
		Hostname: cfg.Screen.ID, ControlURL: st.ControlURL, AuthKey: authKey,
		StateDir: filepath.Join(cfg.Agent.StateDir, "tsnet"), Verbose: verbose,
	})
	if err != nil {
		if authKey != "" {
			// la clé a peut-être expiré (24 h) : demander au portail d'en refaire une
			log.Printf("[agent] connexion refusée avec la clé reçue : %v", err)
		}
		return err
	}
	defer tn.Close()
	if authKey != "" {
		st.Joined = true
		portal.SaveState(cfg.Agent.StateDir, st)
		pc.AckKey(ctx)
	}
	log.Printf("[agent] connecté, ip réseau privé %s", tn.IPv4())
	status.Global.Update(func(x *status.Snapshot) { x.TailnetIP = tn.IPv4().String() })
	status.Global.SetPhase("online", "En ligne")

	a := &agentState{cfg: cfg, pc: pc, tn: tn}

	// 4. Configuration distante et forwards.
	if err := a.reloadConfig(ctx); err != nil {
		return err
	}
	defer a.stopForwards()

	// 5. Mises à jour automatiques.
	go a.updateLoop(ctx)

	// 6. Heartbeat + commandes.
	rdID := sysinfo.RustDeskID(ctx)
	for {
		hbCtx, hbCancel := context.WithTimeout(ctx, 20*time.Second)
		resp, err := a.pc.Heartbeat(hbCtx, a.buildHeartbeat(hbCtx, rdID))
		hbCancel()
		if errors.Is(err, portal.ErrUnauthorized) {
			os.Remove(filepath.Join(cfg.Agent.StateDir, "device.json"))
			return err
		}
		if err != nil {
			log.Printf("[heartbeat] %v", err)
			status.Global.Update(func(x *status.Snapshot) { x.LastError = "heartbeat : " + err.Error() })
		} else {
			status.Global.Update(func(x *status.Snapshot) { x.LastHeartbeat = time.Now(); x.LastError = "" })
			if resp.Status.Status != "approved" {
				return fmt.Errorf("appareil %s dans le portail : arrêt du tunnel", resp.Status.Status)
			}
			if resp.Status.ConfigVersion != a.version {
				if err := a.reloadConfig(ctx); err != nil {
					log.Printf("[config] rechargement : %v", err)
				}
			}
			for _, c := range resp.Commands {
				go a.runCommand(ctx, c)
			}
		}
		a.mu.Lock()
		every := time.Duration(a.remote.HeartbeatSeconds) * time.Second
		a.mu.Unlock()
		if every < 5*time.Second {
			every = 15 * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(every):
		}
	}
}

func enroll(ctx context.Context, cfg *config.Config) (*portal.DeviceState, error) {
	host, _ := os.Hostname()
	local := &portal.DeviceConfig{Processor: portal.ProcessorConf{IP: cfg.Processor.IP, Port: cfg.Processor.Port}}
	for _, f := range cfg.Forwards {
		local.Forwards = append(local.Forwards, portal.ForwardConf{Name: f.Name, Proto: f.Proto, Listen: f.Listen, Target: f.Target})
	}
	pc := portal.New(cfg.Portal.URL, "")
	id, token, err := pc.Enroll(ctx, portal.EnrollRequest{
		ProjectKey: cfg.Portal.ProjectKey, ScreenID: cfg.Screen.ID, Name: cfg.Screen.Name,
		Hostname: host, OS: runtime.GOOS, Arch: runtime.GOARCH, AgentVersion: version, LocalConfig: local,
	})
	if err != nil {
		return nil, fmt.Errorf("inscription au portail : %w", err)
	}
	st := &portal.DeviceState{DeviceID: id, Token: token}
	if err := portal.SaveState(cfg.Agent.StateDir, st); err != nil {
		return nil, err
	}
	log.Printf("[agent] inscrit comme appareil #%d : approuver l'écran dans %s", id, cfg.Portal.URL)
	return st, nil
}

// reloadConfig récupère la configuration du portail et (re)démarre les forwards.
func (a *agentState) reloadConfig(ctx context.Context) error {
	ver, remote, err := a.pc.Config(ctx)
	if err != nil {
		return err
	}
	rules := make([]forward.Rule, 0, len(remote.Forwards))
	var fw []config.Forward
	for _, f := range remote.Forwards {
		rules = append(rules, forward.Rule{Name: f.Name, Proto: strings.ToLower(f.Proto), Listen: f.Listen, Target: f.Target})
		fw = append(fw, config.Forward{Name: f.Name, Proto: strings.ToLower(f.Proto), Listen: f.Listen, Target: f.Target})
	}
	if err := config.ValidateForwards(fw); err != nil {
		return fmt.Errorf("configuration du portail refusée : %w", err)
	}
	a.stopForwards()
	fctx, cancel := context.WithCancel(ctx)
	fwds, err := forward.StartAll(fctx, a.tn, rules)
	if err != nil {
		cancel()
		return err
	}
	a.mu.Lock()
	a.remote, a.version, a.fwds, a.fwdStop = *remote, ver, fwds, cancel
	a.mu.Unlock()
	log.Printf("[config] version %d appliquée : %d forward(s), processeur %s:%d", ver, len(rules), remote.Processor.IP, remote.Processor.Port)
	status.Global.Update(func(x *status.Snapshot) { x.Name = remote.Name })
	return nil
}

func (a *agentState) stopForwards() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.fwdStop != nil {
		a.fwdStop()
		a.fwdStop = nil
		a.fwds = nil
		time.Sleep(100 * time.Millisecond) // laisser les listeners se fermer avant de réécouter
	}
}

func (a *agentState) buildHeartbeat(ctx context.Context, rdID string) heartbeat.Payload {
	a.mu.Lock()
	remote, fwds := a.remote, a.fwds
	a.mu.Unlock()
	p := heartbeat.Payload{
		ScreenID: a.cfg.Screen.ID, ScreenName: remote.Name, AgentVersion: version,
		Timestamp: time.Now().UTC(), TailnetIP: a.tn.IPv4().String(),
	}
	p.RemoteDesktop.Provider = "rustdesk"
	p.RemoteDesktop.ID = rdID
	if remote.Processor.IP != "" {
		r := probe.TCP(ctx, remote.Processor.IP, remote.Processor.Port)
		p.Processor = r
		status.Global.Update(func(x *status.Snapshot) { ok := r.Reachable; x.ProcessorOK, x.ProcessorAddr = &ok, r.Target })
	}
	stats := make([]forward.Stats, 0, len(fwds))
	for _, f := range fwds {
		stats = append(stats, f.Stats())
	}
	p.Forwards = stats
	if peers, err := a.tn.PeerPaths(ctx); err == nil {
		p.Peers = peers
	}
	p.System = sysinfo.Collect(ctx)
	return p
}

// runCommand exécute une commande du portail et remonte le résultat.
func (a *agentState) runCommand(ctx context.Context, c portal.Command) {
	ok, result := true, ""
	switch c.Kind {
	case "http_request":
		var req struct {
			Method         string            `json:"method"`
			URL            string            `json:"url"`
			Headers        map[string]string `json:"headers"`
			Body           string            `json:"body"`
			TimeoutSeconds int               `json:"timeout_seconds"`
		}
		if err := jsonUnmarshal(c.Payload, &req); err != nil {
			ok, result = false, "payload invalide : "+err.Error()
			break
		}
		ok, result = localHTTP(ctx, req.Method, req.URL, req.Headers, req.Body, time.Duration(req.TimeoutSeconds)*time.Second)
	case "probe":
		a.mu.Lock()
		pr := a.remote.Processor
		a.mu.Unlock()
		if pr.IP == "" {
			ok, result = false, "aucun processeur configuré"
			break
		}
		r := probe.TCP(ctx, pr.IP, pr.Port)
		if r.Reachable {
			result = fmt.Sprintf("processeur %s joignable en %.1f ms", r.Target, r.RTTMS)
		} else {
			ok, result = false, fmt.Sprintf("processeur %s injoignable : %s", r.Target, r.Error)
		}
	case "update":
		v, err := applyUpdate(ctx, a.cfg, false)
		if err != nil {
			if errors.Is(err, update.ErrUpToDate) {
				result = "déjà à jour (" + version + ")"
			} else {
				ok, result = false, err.Error()
			}
		} else {
			result = "version " + v + " installée, redémarrage"
		}
	case "restart":
		result = "redémarrage de l'agent"
		a.pc.CommandResult(ctx, c.ID, true, result)
		exe, _ := os.Executable()
		if service.Interactive() {
			log.Printf("[commande] redémarrage demandé : relancer l'agent")
			os.Exit(0)
		}
		if err := update.SpawnRestart(exe, a.cfg.Path); err != nil {
			log.Printf("[commande] redémarrage impossible : %v", err)
		}
		return
	default:
		ok, result = false, "commande inconnue : "+c.Kind
	}
	log.Printf("[commande] %s #%d : %s", c.Kind, c.ID, result)
	if err := a.pc.CommandResult(ctx, c.ID, ok, result); err != nil {
		log.Printf("[commande] envoi du résultat : %v", err)
	}
}

func localHTTP(ctx context.Context, method, url string, headers map[string]string, body string, timeout time.Duration) (bool, string) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return false, "erreur : " + err.Error()
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "erreur : " + err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return resp.StatusCode < 400, fmt.Sprintf("%s en %d ms : %s", resp.Status, time.Since(start).Milliseconds(), strings.TrimSpace(string(b)))
}

// updateLoop : vérification périodique selon la configuration du portail.
func (a *agentState) updateLoop(ctx context.Context) {
	if version == "dev" {
		return
	}
	first := time.Minute + time.Duration(rand.IntN(120))*time.Second
	t := time.NewTimer(first)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.mu.Lock()
		upd := a.remote.Update
		a.mu.Unlock()
		hours := upd.CheckHours
		if hours <= 0 {
			hours = 1
		}
		t.Reset(time.Duration(hours) * time.Hour)
		if !upd.Enabled {
			continue
		}
		v, err := applyUpdate(ctx, a.cfg, false)
		if err != nil {
			if !errors.Is(err, update.ErrUpToDate) {
				log.Printf("[update] %v", err)
			}
			continue
		}
		log.Printf("[update] version %s installée", v)
		if service.Interactive() {
			log.Printf("[update] relancer l'agent")
		}
		return
	}
}

// ---- probe ----

func runProbe(cfg *config.Config, target string, verbose bool) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	st, err := portal.LoadState(cfg.Agent.StateDir)
	if err != nil {
		log.Fatal("cet agent n'est pas encore inscrit/approuvé : lancer d'abord `agent run` et approuver l'écran")
	}
	pc := portal.New(cfg.Portal.URL, st.Token)
	controlURL, key, err := pc.ProbeKey(ctx)
	if err != nil {
		log.Fatal("clé de diagnostic : ", err)
	}
	fmt.Printf("connexion à %s (noeud de diagnostic %s-probe)...\n", controlURL, cfg.Screen.ID)
	tn, err := tunnel.Start(ctx, tunnel.Options{
		Hostname: cfg.Screen.ID + "-probe", ControlURL: controlURL, AuthKey: key,
		StateDir: filepath.Join(cfg.Agent.StateDir, "probe"), Verbose: verbose, Ephemeral: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tn.Close()
	fmt.Printf("ip réseau privé %s\n", tn.IPv4())
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

var _ = net.JoinHostPort
