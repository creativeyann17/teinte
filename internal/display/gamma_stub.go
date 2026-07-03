//go:build !windows && !linux

package display

import (
	"log/slog"

	"teinte/internal/color"
)

type stubManager struct{}

// New returns a no-op backend with two fake displays so the UI can be
// developed and tested on unsupported platforms.
func New() Manager { return &stubManager{} }

func (m *stubManager) Describe() string { return "no-op stub (dev build)" }

func (m *stubManager) List() ([]Display, error) {
	return []Display{
		{ID: "stub-0", Name: "Stub display 0"},
		{ID: "stub-1", Name: "Stub display 1"},
	}, nil
}

func (m *stubManager) Apply(id string, ramp *color.Ramp) error {
	slog.Info("stub gamma ramp apply", "display", id,
		"rTop", ramp[0][255], "gTop", ramp[1][255], "bTop", ramp[2][255])
	return nil
}
