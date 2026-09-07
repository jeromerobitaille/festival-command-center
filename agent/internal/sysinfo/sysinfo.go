// Package sysinfo collecte l'état de la machine pour le heartbeat.
package sysinfo

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

type Info struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemPercent    float64 `json:"mem_percent"`
}

func Collect(ctx context.Context) Info {
	i := Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
	i.Hostname, _ = os.Hostname()
	if up, err := host.UptimeWithContext(ctx); err == nil {
		i.UptimeSeconds = up
	}
	if pct, err := cpu.PercentWithContext(ctx, 500*time.Millisecond, false); err == nil && len(pct) > 0 {
		i.CPUPercent = pct[0]
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		i.MemPercent = vm.UsedPercent
	}
	return i
}

// RustDeskID tente de lire l'identifiant RustDesk local via `rustdesk --get-id`.
func RustDeskID(ctx context.Context) string {
	candidates := []string{"rustdesk"}
	switch runtime.GOOS {
	case "windows":
		candidates = append(candidates,
			`C:\Program Files\RustDesk\rustdesk.exe`,
			`C:\Program Files (x86)\RustDesk\rustdesk.exe`)
	case "darwin":
		candidates = append(candidates, "/Applications/RustDesk.app/Contents/MacOS/RustDesk")
	}
	for _, bin := range candidates {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		out, err := exec.CommandContext(cctx, bin, "--get-id").Output()
		cancel()
		if err == nil {
			if id := strings.TrimSpace(string(out)); id != "" {
				return id
			}
		}
	}
	return ""
}
