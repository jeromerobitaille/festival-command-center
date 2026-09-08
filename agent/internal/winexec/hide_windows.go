//go:build windows

package winexec

// Hide empêche l'apparition d'une fenêtre de console lors du lancement d'un programme
// console sous Windows. Sans cela, un appel répété dans une boucle de rafraîchissement
// fait clignoter une fenêtre noire à chaque tour.

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func Hide(c *exec.Cmd) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.HideWindow = true
	c.SysProcAttr.CreationFlags |= createNoWindow
}
