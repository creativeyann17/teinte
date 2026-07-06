//go:build windows

// Package autostart registers/unregisters Teinte to start at login,
// pointing at the current binary with --hidden so it lands in the tray
// without flashing a window.
package autostart

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "Teinte"
)

func Available() bool { return true }

// Enabled reports whether the HKCU Run entry exists.
func Enabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(valueName)
	return err == nil
}

// Set writes or removes the HKCU Run entry (no admin rights needed).
// The path is the running binary, so re-toggling after moving the
// binary refreshes it.
func Set(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !enable {
		if err := k.DeleteValue(valueName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	if err := devBinaryGuard(); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return k.SetStringValue(valueName, `"`+exe+`" --hidden`)
}
