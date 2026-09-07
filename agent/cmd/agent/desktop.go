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
		return false
	}
	cmd := exec.Command(tray)
	return cmd.Start() == nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
