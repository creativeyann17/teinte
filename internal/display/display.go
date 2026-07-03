// Package display applies gamma ramps to the OS display pipeline.
// Real implementations: Windows (gdi32 SetDeviceGammaRamp) and Linux
// (X11 RandR SetCrtcGamma); other platforms get a logging stub so the
// UI can be developed anywhere.
package display

import "teinte/internal/color"

// Display is one connected output the user can target.
type Display struct {
	ID   string `json:"id"`   // stable identifier used as the settings key
	Name string `json:"name"` // human-readable label for the dropdown
}

// Manager enumerates displays and pushes gamma ramps to them.
type Manager interface {
	// Describe names the backend so the UI can be transparent about
	// what is actually being driven.
	Describe() string
	// List returns the currently connected displays.
	List() ([]Display, error)
	// Apply sets the ramp on the display with the given ID.
	Apply(id string, ramp *color.Ramp) error
}
