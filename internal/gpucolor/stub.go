//go:build !windows && !linux

package gpucolor

import "log/slog"

// stub emulates a typical driver (range 0..63, default 0) so the whole
// UI is exercisable on unsupported dev platforms.
type stub struct {
	vibrance int
}

func New() Controller { return &stub{} }

func (s *stub) Available() bool      { return true }
func (s *stub) SupportsHue() bool    { return true }
func (s *stub) HueRange() (int, int) { return -180, 180 }
func (s *stub) Describe() string     { return "no-op stub (dev build)" }

func (s *stub) Vibrance() (VibranceInfo, error) {
	return VibranceInfo{Current: s.vibrance, Min: 0, Max: 63, Default: 0}, nil
}

func (s *stub) SetVibrance(level int) error {
	s.vibrance = level
	slog.Info("stub vibrance", "level", level)
	return nil
}

func (s *stub) SetHue(degrees int) error {
	slog.Info("stub hue", "degrees", degrees)
	return nil
}
