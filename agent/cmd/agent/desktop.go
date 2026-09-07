package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// launchTray démarre agent-tray(.exe) s'il est à côté du binaire (double-clic sous Windows).
func launchTray() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	name := "agent-tray"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	tray := filepath.Join(filepath.Dir(exe), name)
	if _, err := os.Stat(tray); err != nil {
		matches, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), "agent-tray*"))
		if len(matches) == 0 {
			return false
		}
		tray = matches[0]
	}
	cmd := exec.Command(tray)
	return cmd.Start() == nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		if exe, err := os.Executable(); err == nil {
			tray := filepath.Join(filepath.Dir(exe), "agent-tray.exe")
			if _, err := os.Stat(tray); err == nil {
				return exec.Command(tray, "--panel").Start() // fenêtre native WebView2
			}
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
