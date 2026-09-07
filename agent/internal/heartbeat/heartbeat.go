// Package heartbeat définit le contenu envoyé périodiquement au portail.
package heartbeat

import "time"

type Payload struct {
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	AgentVersion  string    `json:"agent_version"`
	Timestamp     time.Time `json:"timestamp"`
	TailnetIP     string    `json:"tailnet_ip"`
	SubDevices    any       `json:"sub_devices"`
	Forwards      any       `json:"forwards"`
	Peers         any       `json:"peers"`
	System        any       `json:"system"`
	RemoteDesktop struct {
		Provider string `json:"provider"`
		ID       string `json:"id,omitempty"`
	} `json:"remote_desktop"`
}
