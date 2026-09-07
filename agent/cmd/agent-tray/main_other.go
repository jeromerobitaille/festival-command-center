//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "agent-tray n'existe que pour Windows ; utiliser `agent panel` pour ouvrir le panneau local")
	os.Exit(1)
}
