//go:build windows || linux

package autostart

import (
	"fmt"

	"teinte/internal/desktopentry"
)

// devBinaryGuard refuses to register a `wails dev` binary: its file is
// deleted when the dev session ends, so an autostart entry pointing at
// it is guaranteed to break at next login.
func devBinaryGuard() error {
	if desktopentry.IsDevBinary() {
		return fmt.Errorf("running a wails dev binary — build the real one first (make build), " +
			"a dev binary disappears when the dev session ends")
	}
	return nil
}
