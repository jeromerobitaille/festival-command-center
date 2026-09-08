//go:build !windows

package winexec

import "os/exec"

// Hide n'a pas d'objet hors Windows.
func Hide(c *exec.Cmd) {}
