// Package heartbeat définit le contenu envoyé périodiquement au portail.
package heartbeat

import "time"

type Payload struct {
	ScreenID      string    `json:"screen_id"`
	ScreenName    string    `json:"screen_name"`
	AgentVersion  string    `json:"agent_version"`
	Timestamp     time.Time `json:"timestamp"`
	TailnetIP     string    `json:"tailnet_ip"`
	Processor     any       `json:"processor"`
	Forwards      any       `json:"forwards"`
	Peers         any       `json:"peers"`
	System        any       `json:"system"`
	RemoteDesktop struct {
		Provider string `json:"provider"`
		ID       string `json:"id,omitempty"`
	} `json:"remote_desktop"`
}
