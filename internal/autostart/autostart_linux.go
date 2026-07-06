//go:build linux

// Package autostart registers/unregisters Teinte to start at login,
// pointing at the current binary with --hidden so it lands in the tray
// without flashing a window.
package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

func entryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "teinte.desktop"), nil
}

func Available() bool { return true }

// Enabled reports whether the XDG autostart entry exists.
func Enabled() bool {
	p, err := entryPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Set writes or removes the XDG autostart entry. The Exec path is the
// running binary, so re-toggling after moving the binary refreshes it.
func Set(enable bool) error {
	p, err := entryPath()
	if err != nil {
		return err
	}
	if !enable {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
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
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Teinte
Comment=Display color control
Exec="%s" --hidden
Terminal=false
X-GNOME-Autostart-enabled=true
`, exe)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}
