//go:build windows

package main

// Fenêtre de configuration native (contrôles Windows via lxn/walk). Lancée dans un processus dédié :
// agent-tray.exe --window

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type configResp struct {
	Path string `json:"path"`
	TOML string `json:"toml"`
}

func apiJSON(method, path string, body any) error {
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, localBase+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("agent injoignable (%v)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var e struct{ Error string `json:"error"` }
		json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return fmt.Errorf("%s", e.Error)
	}
	return nil
}

func fetchConfig() (*configResp, error) {
	resp, err := client.Get(localBase + "/api/config")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var c configResp
	return &c, json.NewDecoder(resp.Body).Decode(&c)
}

func runWindow() {
	var mw *walk.MainWindow
	var lblStatus, lblIP, lblSubs, lblHB, lblMode *walk.Label
	var edPortal, edKey, edID, edName, edSubName, edSubIP, edSubPort *walk.LineEdit
	var edToml *walk.TextEdit
	var btnInstall *walk.PushButton
	prefilled := false

	info := func(msg string) { walk.MsgBox(mw, "Agent Festival Command Center", msg, walk.MsgBoxIconInformation) }
	fail := func(msg string) { walk.MsgBox(mw, "Agent Festival Command Center", msg, walk.MsgBoxIconError) }

	loadToml := func() {
		if c, err := fetchConfig(); err == nil {
			edToml.SetText(strings.ReplaceAll(c.TOML, "\n", "\r\n"))
		}
	}

	err := MainWindow{
		AssignTo: &mw,
		Title:    "Agent Festival Command Center",
		MinSize:  Size{Width: 560, Height: 640},
		Size:     Size{Width: 620, Height: 760},
		Layout:   VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}},
		Children: []Widget{
			GroupBox{
				Title:  "État",
				Layout: Grid{Columns: 2, Spacing: 6},
				Children: []Widget{
					Label{Text: "Statut :"}, Label{AssignTo: &lblStatus, Text: "…"},
					Label{Text: "IP réseau privé :"}, Label{AssignTo: &lblIP, Text: "—"},
					Label{Text: "Sous-appareils :"}, Label{AssignTo: &lblSubs, Text: "—"},
					Label{Text: "Dernier heartbeat :"}, Label{AssignTo: &lblHB, Text: "—"},
					Label{Text: "Mode :"}, Label{AssignTo: &lblMode, Text: "—"},
				},
			},
			GroupBox{
				Title:  "Rattacher cet appareil à un projet",
				Layout: Grid{Columns: 2, Spacing: 6},
				Children: []Widget{
					Label{Text: "Portail :"}, LineEdit{AssignTo: &edPortal, CueBanner: "https://panel.veam.ca"},
					Label{Text: "Clé de projet :"}, LineEdit{AssignTo: &edKey, CueBanner: "fcc_… (Configuration du projet dans le portail)"},
					Label{Text: "Identifiant :"}, LineEdit{AssignTo: &edID, CueBanner: "ecran-01"},
					Label{Text: "Nom affiché :"}, LineEdit{AssignTo: &edName, CueBanner: "Place des Festivals - Nord"},
					Label{Text: "Sous-appareil (facultatif) :"},
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 6},
						Children: []Widget{
							LineEdit{AssignTo: &edSubName, CueBanner: "Nom (ex. Processeur LED)"},
							LineEdit{AssignTo: &edSubIP, CueBanner: "IP", MaxSize: Size{Width: 130}},
							LineEdit{AssignTo: &edSubPort, CueBanner: "Port", MaxSize: Size{Width: 70}},
						},
					},
					Label{Text: ""},
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 6},
						Children: []Widget{
							PushButton{Text: "Enregistrer et s'inscrire", OnClicked: func() {
								port := 0
								fmt.Sscanf(strings.TrimSpace(edSubPort.Text()), "%d", &port)
								err := apiJSON("POST", "/api/project", map[string]any{
									"portal_url": edPortal.Text(), "project_key": edKey.Text(), "device_id": edID.Text(),
									"name": edName.Text(), "sub_name": edSubName.Text(), "sub_ip": edSubIP.Text(), "sub_port": port,
								})
								if err != nil {
									fail("Enregistrement impossible : " + err.Error())
									return
								}
								edKey.SetText("")
								loadToml()
								info("Enregistré. L'appareil va s'inscrire : approuvez-le dans le portail (onglet Appareils).")
							}},
							HSpacer{},
						},
					},
				},
			},
			GroupBox{
				Title:  "Actions",
				Layout: HBox{Spacing: 6},
				Children: []Widget{
					PushButton{Text: "Relire la configuration", OnClicked: func() { apiJSON("POST", "/api/reload", nil) }},
					PushButton{Text: "Redémarrer l'agent", OnClicked: func() { apiJSON("POST", "/api/restart", nil) }},
					PushButton{Text: "Réinscrire", OnClicked: func() {
						if walk.MsgBox(mw, "Réinscrire", "Oublier l'inscription ? L'appareil devra être ré-approuvé dans le portail.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
							apiJSON("POST", "/api/reset", nil)
						}
					}},
					PushButton{AssignTo: &btnInstall, Text: "Installer le service (admin)", OnClicked: func() {
						go installService()
					}},
					HSpacer{},
				},
			},
			GroupBox{
				Title:  "Fichier de configuration (agent.toml)",
				Layout: VBox{Spacing: 6},
				Children: []Widget{
					TextEdit{AssignTo: &edToml, VScroll: true, Font: Font{Family: "Consolas", PointSize: 9}, MinSize: Size{Height: 160}},
					Composite{
						Layout: HBox{MarginsZero: true, Spacing: 6},
						Children: []Widget{
							PushButton{Text: "Enregistrer le fichier", OnClicked: func() {
								toml := strings.ReplaceAll(edToml.Text(), "\r\n", "\n")
								if err := apiJSON("PUT", "/api/config", map[string]string{"toml": toml}); err != nil {
									fail("Fichier refusé : " + err.Error())
									return
								}
								info("Fichier enregistré, configuration relue.")
							}},
							PushButton{Text: "Recharger", OnClicked: loadToml},
							PushButton{Text: "Ouvrir le dossier", OnClicked: func() {
								exec.Command("explorer.exe", filepath.Dir(configPath())).Start()
							}},
							HSpacer{},
							Label{Text: "agent-tray " + version, TextColor: walk.RGB(120, 120, 120)},
						},
					},
				},
			},
		},
	}.Create()
	if err != nil {
		msgBox("Agent Festival Command Center", "Impossible d'ouvrir la fenêtre : "+err.Error(), 0x10)
		return
	}
	if ico, err := walk.NewIconFromResourceId(1); err == nil {
		mw.SetIcon(ico)
	}
	loadToml()

	// rafraîchissement de l'état
	go func() {
		for {
			snap, err := fetchStatus()
			installed := serviceInstalled()
			mw.Synchronize(func() {
				if err != nil {
					lblStatus.SetText("agent injoignable (service arrêté ?)")
					lblIP.SetText("—")
					lblSubs.SetText("—")
					lblHB.SetText("—")
				} else {
					lblStatus.SetText(snap.Message)
					ip := snap.TailnetIP
					if ip == "" {
						ip = "—"
					}
					lblIP.SetText(ip)
					ok, total := snap.subsOK()
					if total == 0 {
						lblSubs.SetText("aucun")
					} else {
						var parts []string
						for _, d := range snap.SubDevices {
							st := "joignable"
							if !d.Reachable {
								st = "injoignable"
							}
							parts = append(parts, d.Name+" : "+st)
						}
						lblSubs.SetText(fmt.Sprintf("%d/%d — %s", ok, total, strings.Join(parts, ", ")))
					}
					if snap.LastHeartbeat.IsZero() {
						lblHB.SetText("—")
					} else {
						lblHB.SetText(snap.LastHeartbeat.Local().Format("15:04:05"))
					}
					if !prefilled {
						edPortal.SetText(snap.PortalURL)
						edID.SetText(snap.Slug)
						edName.SetText(snap.Name)
						prefilled = true
					}
				}
				if installed {
					lblMode.SetText("service Windows (démarre avec l'ordinateur)")
					btnInstall.SetVisible(false)
				} else {
					lblMode.SetText("session utilisateur — installer le service pour démarrer avec Windows")
					btnInstall.SetVisible(true)
				}
			})
			time.Sleep(3 * time.Second)
		}
	}()
	mw.Run()
}

// openWindow lance la fenêtre dans un processus dédié (walk et systray ont chacun leur boucle de messages).
func openWindow() {
	exe, _ := os.Executable()
	exec.Command(exe, "--window").Start()
}
