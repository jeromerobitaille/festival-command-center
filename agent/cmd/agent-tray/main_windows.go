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
	Version       string    `json:"version"`
	ScreenID      string    `json:"screen_id"`
	Name          string    `json:"name"`
	Phase         string    `json:"phase"`
	Message       string    `json:"message"`
	TailnetIP     string    `json:"tailnet_ip"`
	ProcessorOK   *bool     `json:"processor_ok"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

var (
	mStatus, mIP, mProc, mVersion            *systray.MenuItem
	mPanel, mKey, mFolder, mRestart, mInstall *systray.MenuItem
	mQuit                                     *systray.MenuItem
	client                                    = &http.Client{Timeout: 3 * time.Second}
)

func main() {
	registerAutostart()
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetIcon(icoOffline)
	systray.SetTitle("Festival Command Center")
	systray.SetTooltip("Agent Festival Command Center")
	mStatus = systray.AddMenuItem("Statut : …", "")
	mIP = systray.AddMenuItem("IP privée : —", "")
	mProc = systray.AddMenuItem("Processeur : —", "")
	mStatus.Disable()
	mIP.Disable()
	mProc.Disable()
	systray.AddSeparator()
	mPanel = systray.AddMenuItem("Ouvrir le panneau", "Statut détaillé, clé de projet, configuration")
	mKey = systray.AddMenuItem("Entrer la clé de projet…", "Rattacher cet écran à un projet")
	mFolder = systray.AddMenuItem("Ouvrir le dossier de l'agent", "")
	systray.AddSeparator()
	mRestart = systray.AddMenuItem("Redémarrer l'agent", "")
	mInstall = systray.AddMenuItem("Installer le service (administrateur)", "Démarre l'agent avec Windows, avant l'ouverture de session")
	mVersion = systray.AddMenuItem("agent-tray "+version, "")
	mVersion.Disable()
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Fermer l'icône", "L'agent continue de tourner en service")

	go loop()
	go func() {
		for {
			select {
			case <-mPanel.ClickedCh:
				openURL(localBase + "/")
			case <-mKey.ClickedCh:
				openURL(localBase + "/#key")
			case <-mFolder.ClickedCh:
				exe, _ := os.Executable()
				exec.Command("explorer.exe", filepath.Dir(exe)).Start()
			case <-mRestart.ClickedCh:
				client.Post(localBase+"/api/restart", "application/json", nil)
			case <-mInstall.ClickedCh:
				runElevated("install")
			case <-mQuit.ClickedCh:
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
		if err != nil {
			systray.SetIcon(icoOffline)
			mStatus.SetTitle("Statut : agent injoignable (service arrêté ?)")
			mIP.SetTitle("IP privée : —")
			mProc.SetTitle("Processeur : —")
			systray.SetTooltip("Agent Festival Command Center — arrêté")
			mInstall.Show()
		} else {
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
			switch {
			case snap.ProcessorOK == nil:
				mProc.SetTitle("Processeur : —")
			case *snap.ProcessorOK:
				mProc.SetTitle("Processeur : joignable")
			default:
				mProc.SetTitle("Processeur : injoignable")
			}
			switch snap.Phase {
			case "online":
				if snap.ProcessorOK != nil && !*snap.ProcessorOK {
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
			if serviceInstalled() {
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
	out, err := exec.Command("sc", "query", "festival-agent").Output()
	return err == nil && strings.Contains(string(out), "STATE")
}

// runElevated lance `agent.exe -config <toml> <cmd>` avec élévation UAC.
func runElevated(cmd string) {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	agent := filepath.Join(dir, "agent.exe")
	args := fmt.Sprintf(`-config "%s" %s`, filepath.Join(dir, "agent.toml"), cmd)
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(agent)
	params, _ := syscall.UTF16PtrFromString(args)
	cwd, _ := syscall.UTF16PtrFromString(dir)
	windows.ShellExecute(0, verb, file, params, cwd, windows.SW_HIDE)
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
