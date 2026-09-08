//go:build windows

// agent-tray : icône de la barre des tâches Windows pour l'agent Festival Command Center.
// Tourne dans la session de l'utilisateur et parle au service via l'API locale (127.0.0.1:47632).
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/festival/command-center/agent/internal/winexec"
	"syscall"
	"time"

	"fyne.io/systray"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var version = "dev"

//go:embed icons/online.ico
var icoOnline []byte

//go:embed icons/warn.ico
var icoWarn []byte

//go:embed icons/offline.ico
var icoOffline []byte

const localBase = "http://127.0.0.1:47632"

type snapshot struct {
	Version    string `json:"version"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	PortalURL  string `json:"portal_url"`
	Phase      string `json:"phase"`
	Message    string `json:"message"`
	TailnetIP  string `json:"tailnet_ip"`
	SubDevices []struct {
		Name      string `json:"name"`
		Reachable bool   `json:"reachable"`
	} `json:"sub_devices"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

func (s *snapshot) subsOK() (ok, total int) {
	for _, d := range s.SubDevices {
		total++
		if d.Reachable {
			ok++
		}
	}
	return
}

var (
	mStatus, mIP, mProc, mMode, mVersion      *systray.MenuItem
	mPanel, mKey, mFolder, mRestart, mInstall *systray.MenuItem
	mQuit                                     *systray.MenuItem
	client                                    = &http.Client{Timeout: 3 * time.Second}
	userAgent                                 *exec.Cmd // agent lancé en mode utilisateur (sans service)
)

// findAgentExe retrouve le binaire de l'agent à côté de l'icône, quel que soit son nom (agent.exe ou
// agent_0.4.0_windows_amd64.exe tel que téléchargé depuis la release).
func findAgentExe() (string, error) {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "agent.exe")); err == nil {
		return filepath.Join(dir, "agent.exe"), nil
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "agent_*_windows_*.exe"))
	if len(matches) > 0 {
		return matches[len(matches)-1], nil
	}
	return "", fmt.Errorf("agent.exe introuvable dans %s\n\nPlacez agent.exe (ou agent_<version>_windows_amd64.exe) dans le même dossier que agent-tray.exe.", dir)
}

func configPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "agent.toml")
}

func msgBox(title, text string, flags uint32) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	windows.MessageBox(0, b, t, flags|windows.MB_SETFOREGROUND)
}

// startUserModeAgent lance `agent.exe run` dans la session courante quand aucun service ne répond.
func startUserModeAgent() {
	if userAgent != nil && userAgent.ProcessState == nil {
		return
	}
	agent, err := findAgentExe()
	if err != nil {
		return
	}
	cmd := exec.Command(agent, "-config", configPath(), "run")
	winexec.Hide(cmd)
	if err := cmd.Start(); err == nil {
		userAgent = cmd
		go cmd.Wait()
	}
}

func stopUserModeAgent() {
	if userAgent != nil && userAgent.Process != nil && userAgent.ProcessState == nil {
		userAgent.Process.Kill()
	}
	userAgent = nil
}

func main() {
	// agent-tray.exe --panel [#fragment] : fenêtre du panneau (processus dédié lancé par l'icône).
	if len(os.Args) > 1 && os.Args[1] == "--panel" {
		frag := ""
		if len(os.Args) > 2 {
			frag = os.Args[2]
		}
		if !runPanelWindow(frag) {
			openURL(localBase + "/" + frag)
		}
		return
	}
	registerAutostart()
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetIcon(icoOffline)
	systray.SetTitle("Festival Command Center")
	systray.SetTooltip("Agent Festival Command Center")
	mStatus = systray.AddMenuItem("Statut : …", "")
	mIP = systray.AddMenuItem("IP privée : —", "")
	mProc = systray.AddMenuItem("Sous-appareils : —", "")
	mMode = systray.AddMenuItem("Mode : …", "")
	mStatus.Disable()
	mIP.Disable()
	mProc.Disable()
	mMode.Disable()
	systray.AddSeparator()
	mPanel = systray.AddMenuItem("Configuration…", "Statut détaillé, clé de projet, fichier de configuration")
	mKey = systray.AddMenuItem("Panneau web (avancé)", "Le même panneau dans le navigateur")
	mFolder = systray.AddMenuItem("Ouvrir le dossier de l'agent", "")
	systray.AddSeparator()
	mRestart = systray.AddMenuItem("Redémarrer l'agent", "")
	mInstall = systray.AddMenuItem("Installer le service (administrateur)", "Démarre l'agent avec Windows, avant l'ouverture de session")
	mVersion = systray.AddMenuItem("agent-tray "+version, "")
	mVersion.Disable()
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Fermer l'icône", "Le service, s'il est installé, continue de tourner")
	if _, err := findAgentExe(); err != nil {
		msgBox("Agent Festival Command Center", err.Error(), windows.MB_ICONWARNING)
	}

	go loop()
	go func() {
		for {
			select {
			case <-mPanel.ClickedCh:
				openPanelWindow("")
			case <-mKey.ClickedCh:
				openPanelWindow("#key")
			case <-mFolder.ClickedCh:
				exe, _ := os.Executable()
				exec.Command("explorer.exe", filepath.Dir(exe)).Start()
			case <-mRestart.ClickedCh:
				client.Post(localBase+"/api/restart", "application/json", nil)
			case <-mInstall.ClickedCh:
				go installService()
			case <-mQuit.ClickedCh:
				stopUserModeAgent()
				systray.Quit()
				return
			}
		}
	}()
}

// loop interroge l'agent toutes les 4 s et met à jour icône et menu.
func loop() {
	var lastAgentVersion string
	for {
		snap, err := fetchStatus()
		installed := serviceInstalled()
		if err != nil {
			systray.SetIcon(icoOffline)
			mStatus.SetTitle("Statut : agent injoignable")
			mIP.SetTitle("IP privée : —")
			mProc.SetTitle("Sous-appareils : —")
			systray.SetTooltip("Agent Festival Command Center — arrêté")
			mInstall.Show()
			if installed {
				mMode.SetTitle("Mode : service installé mais arrêté (Redémarrer l'agent)")
			} else {
				mMode.SetTitle("Mode : démarrage en session utilisateur…")
				startUserModeAgent() // pas de service : on fait tourner l'agent ici pour pouvoir le configurer
			}
		} else {
			if installed {
				mMode.SetTitle("Mode : service Windows (démarre avec l'ordinateur)")
			} else {
				mMode.SetTitle("Mode : session utilisateur — installer le service pour démarrer avec Windows")
			}
			label := snap.Message
			if snap.Name != "" {
				label = snap.Name + " — " + label
			}
			mStatus.SetTitle("Statut : " + label)
			ip := snap.TailnetIP
			if ip == "" {
				ip = "—"
			}
			mIP.SetTitle("IP privée : " + ip)
			okN, total := snap.subsOK()
			if total == 0 {
				mProc.SetTitle("Sous-appareils : aucun")
			} else {
				mProc.SetTitle(fmt.Sprintf("Sous-appareils : %d/%d joignables", okN, total))
			}
			switch snap.Phase {
			case "online":
				if okN < total {
					systray.SetIcon(icoWarn)
				} else {
					systray.SetIcon(icoOnline)
				}
			case "pending", "connecting", "enrolling":
				systray.SetIcon(icoWarn)
			default:
				systray.SetIcon(icoOffline)
			}
			systray.SetTooltip("Agent Festival Command Center — " + label)
			if installed {
				mInstall.Hide()
			} else {
				mInstall.Show()
			}
			// L'agent s'est mis à jour : relancer l'icône si un nouveau agent-tray.exe est en place.
			if lastAgentVersion != "" && snap.Version != lastAgentVersion && snap.Version != version {
				relaunchSelf()
			}
			lastAgentVersion = snap.Version
		}
		time.Sleep(4 * time.Second)
	}
}

func fetchStatus() (*snapshot, error) {
	resp, err := client.Get(localBase + "/api/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var s snapshot
	return &s, json.NewDecoder(resp.Body).Decode(&s)
}

func serviceInstalled() bool {
	// Appelé toutes les 3-4 s par les boucles d'état de l'icône et de la fenêtre :
	// sans masquage, chaque appel fait clignoter une console.
	c := exec.Command("sc", "query", "festival-agent")
	winexec.Hide(c)
	out, err := c.Output()
	return err == nil && strings.Contains(string(out), "STATE")
}

// runElevated lance `agent.exe -config <toml> <cmd>` avec élévation UAC.
func runElevated(cmd string) error {
	agent, err := findAgentExe()
	if err != nil {
		return err
	}
	dir := filepath.Dir(agent)
	args := fmt.Sprintf(`-config "%s" %s`, configPath(), cmd)
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(agent)
	params, _ := syscall.UTF16PtrFromString(args)
	cwd, _ := syscall.UTF16PtrFromString(dir)
	return windows.ShellExecute(0, verb, file, params, cwd, windows.SW_HIDE)
}

// installService : arrête l'agent en mode utilisateur (le service reprendra le port), installe avec UAC,
// puis vérifie le résultat et l'affiche.
func installService() {
	stopUserModeAgent()
	time.Sleep(500 * time.Millisecond)
	if err := runElevated("install"); err != nil {
		msgBox("Installation du service", "Impossible de lancer l'installation :\n"+err.Error(), windows.MB_ICONERROR)
		return
	}
	for i := 0; i < 20; i++ { // jusqu'à 20 s : invite UAC + démarrage du service
		time.Sleep(time.Second)
		if serviceInstalled() {
			if _, err := fetchStatus(); err == nil {
				msgBox("Installation du service", "Le service Festival Command Center est installé et démarré.\nIl se lancera désormais avec Windows.", windows.MB_ICONINFORMATION)
				return
			}
		}
	}
	logTxt := ""
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(configPath()), "install.log")); err == nil {
		logTxt = "\n\n" + string(b)
	}
	if serviceInstalled() {
		msgBox("Installation du service", "Le service est installé mais ne répond pas encore. Essayez « Redémarrer l'agent »."+logTxt, windows.MB_ICONWARNING)
	} else {
		msgBox("Installation du service", "Le service n'a pas été installé (invite refusée ou erreur)."+logTxt, windows.MB_ICONWARNING)
	}
}

func openURL(u string) { exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start() }

// registerAutostart inscrit l'icône au démarrage de la session (HKCU\...\Run).
func registerAutostart() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	k.SetStringValue("FestivalCommandCenterTray", `"`+exe+`"`)
}

func relaunchSelf() {
	exe, _ := os.Executable()
	if exec.Command(exe).Start() == nil {
		systray.Quit()
		os.Exit(0)
	}
}
