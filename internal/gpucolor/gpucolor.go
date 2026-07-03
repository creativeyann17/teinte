// Package gpucolor drives the GPU-vendor color controls that a gamma
// ramp cannot express (they mix color channels): saturation and hue.
//
// Backends, probed in order on Windows:
//   - NvAPI digital vibrance + hue (nvapi64.dll) — when the NVIDIA GPU
//     drives the display (MUX in direct mode).
//   - ADL, the official AMD Display Library behind the Adrenalin
//     "custom color" panel (atiadlxx.dll) — when the AMD GPU drives the
//     display (hybrid/iGPU mode on AMD-powered laptops).
//
// On Linux only NVIDIA vibrance exists (via nvidia-settings, X11).
// Whichever GPU actually owns the display answers; the other reports
// unavailable, and the UI disables the sliders with the reason.
package gpucolor

import "fmt"

// VibranceInfo is the raw driver saturation range. NvAPI is typically
// 0..63 default 0; ADL is typically 0..200 default 100.
type VibranceInfo struct {
	Current int
	Min     int
	Max     int
	Default int
}

// Controller exposes the vendor color controls.
type Controller interface {
	// Available reports whether the backend loaded AND at least one
	// display is driven by its GPU.
	Available() bool
	// SupportsHue reports whether the hue control exists on this
	// backend (NvAPI and ADL yes; Linux nvidia-settings no).
	SupportsHue() bool
	// HueRange is the driver's hue slider range in degrees
	// (NvAPI ±180, ADL typically ±30). Only meaningful when SupportsHue.
	HueRange() (min, max int)
	// Describe names the backend (or the reason it is unavailable).
	Describe() string
	// Vibrance returns the driver-reported saturation range and level.
	Vibrance() (VibranceInfo, error)
	// SetVibrance sets the raw saturation level on all driven displays.
	SetVibrance(level int) error
	// SetHue sets the hue in degrees on all driven displays; values are
	// clamped/wrapped to what the driver accepts.
	SetHue(degrees int) error
}

// unavailable is the shared degraded controller: everything off, with a
// human-readable reason surfaced in the UI.
type unavailable struct{ reason string }

func (u *unavailable) Available() bool                 { return false }
func (u *unavailable) SupportsHue() bool               { return false }
func (u *unavailable) HueRange() (int, int)            { return 0, 0 }
func (u *unavailable) Describe() string                { return u.reason }
func (u *unavailable) err() error                      { return fmt.Errorf("unavailable: %s", u.reason) }
func (u *unavailable) Vibrance() (VibranceInfo, error) { return VibranceInfo{}, u.err() }
func (u *unavailable) SetVibrance(int) error           { return u.err() }
func (u *unavailable) SetHue(int) error                { return u.err() }
