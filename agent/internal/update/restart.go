package update

import (
	"os"
	"os/exec"
)

// SpawnRestart lance, détaché du processus courant, `<exe> -config <cfg> restart` : le nouveau binaire
// arrête puis redémarre le service. Le processus courant peut ensuite se terminer.
func SpawnRestart(exePath, cfgPath string) error {
	if exePath == "" {
		exePath, _ = os.Executable()
	}
	cmd := exec.Command(exePath, "-config", cfgPath, "restart")
	cmd.Stdout, cmd.Stderr = nil, nil
	detach(cmd)
	return cmd.Start()
}
