// Package config persists settings as plain JSON under the user config
// dir (%AppData%\teinte on Windows, ~/.config/teinte on Linux) so
// they survive restarts and are trivially inspectable/editable by hand.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"teinte/internal/color"
)

// File is the on-disk format: one settings block per display, keyed by
// the display ID from the display backend, the last selection, and the
// user-saved profiles. The same format is used by export/import.
type File struct {
	Selected     string                    `json:"selected"`
	Displays     map[string]color.Settings `json:"displays"`
	UserProfiles map[string]color.Settings `json:"userProfiles,omitempty"`
}

// Normalize clamps every settings block; call after any unmarshal
// (disk load or user import).
func (f File) Normalize() File {
	for id, s := range f.Displays {
		f.Displays[id] = s.Clamp()
	}
	for name, s := range f.UserProfiles {
		f.UserProfiles[name] = s.Clamp()
	}
	return f
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "teinte", "settings.json"), nil
}

// Load returns the saved file. found is false when no config exists yet
// (first run, or a pre-per-display format) — that is not an error.
func Load() (f File, found bool, err error) {
	p, err := path()
	if err != nil {
		return f, false, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return f, false, nil
	}
	if err != nil {
		return f, false, err
	}
	if err := json.Unmarshal(data, &f); err != nil || f.Displays == nil {
		return File{}, false, nil // unknown/old format: start fresh
	}
	return f.Normalize(), true, nil
}

// Save writes the file, creating the directory on first run.
func Save(f File) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
