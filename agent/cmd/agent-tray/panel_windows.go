//go:build windows

package main

import (
	"os"
	"os/exec"

	webview2 "github.com/jchv/go-webview2"
)

// openPanelWindow lance le panneau dans un processus séparé (WebView2 exige sa propre boucle de messages).
func openPanelWindow(fragment string) {
	exe, _ := os.Executable()
	if err := exec.Command(exe, "--panel", fragment).Start(); err != nil {
		openURL(localBase + "/" + fragment)
	}
}

// runPanelWindow affiche le panneau local dans une fenêtre native (WebView2, moteur d'Edge intégré à
// Windows 10/11). Retourne false si WebView2 est indisponible : on ouvrira le navigateur à la place.
func runPanelWindow(fragment string) bool {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Agent Festival Command Center",
			Width:  760,
			Height: 900,
			Center: true,
		},
	})
	if w == nil {
		return false
	}
	defer w.Destroy()
	w.SetSize(760, 900, webview2.HintMin)
	w.Navigate(localBase + "/" + fragment)
	w.Run()
	return true
}
