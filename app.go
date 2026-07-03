package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"teinte/internal/color"
	"teinte/internal/config"
	"teinte/internal/display"
	"teinte/internal/gpucolor"
	"teinte/internal/tray"
)

// App is the Wails-bound backend.
type App struct {
	ctx  context.Context
	mu   sync.Mutex
	disp display.Manager
	gpu  gpucolor.Controller

	selected   string
	perDisplay map[string]color.Settings
	quitting   bool // set by the tray Quit so OnBeforeClose lets the app die
}

// State is everything the frontend needs to render: current settings
// plus full transparency about which backends are actually active.
//
// Gamma settings are per display (the dropdown selection); saturation
// and hue are GPU-global — the vendor drivers expose them per GPU, not
// per output on every backend, so the UI states that instead of
// pretending otherwise.
type State struct {
	Settings            color.Settings    `json:"settings"`
	Displays            []display.Display `json:"displays"`
	Selected            string            `json:"selected"`
	GammaBackend        string            `json:"gammaBackend"`
	VendorBackend       string            `json:"vendorBackend"`
	SaturationAvailable bool              `json:"saturationAvailable"`
	HueAvailable        bool              `json:"hueAvailable"`
	HueMin              int               `json:"hueMin"`
	HueMax              int               `json:"hueMax"`
	SaturationDefault   int               `json:"saturationDefault"`
	Errors              string            `json:"errors"`
	Version             string            `json:"version"`
}

func NewApp() *App {
	return &App{
		disp:       display.New(),
		gpu:        gpucolor.New(),
		perDisplay: map[string]color.Settings{},
	}
}

// startup wires the tray and reapplies the saved colors, so they come
// back automatically after reboot/login.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	tray.Start(trayIcon(), "Teinte", tray.Callbacks{
		OnOpen: func() { runtime.WindowShow(a.ctx) },
		OnQuit: func() {
			a.quitting = true
			runtime.Quit(a.ctx)
		},
	})

	// SIGTERM/SIGINT must be a real quit: without this the
	// close-to-tray hook would swallow them and the app would resist
	// pkill and session logout.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		a.quitting = true
		runtime.Quit(a.ctx)
	}()

	if errs := a.loadAndApplyAll(); errs != "" {
		slog.Warn("startup apply", "errors", errs)
	}
}

// loadAndApplyAll loads the saved config and pushes every display's
// settings plus the GPU-global controls. Shared by GUI startup and the
// headless --apply mode; ramps and vibrance persist for the whole
// session after the process exits, so --apply can quit immediately.
func (a *App) loadAndApplyAll() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var errs string
	saved, found, err := config.Load()
	if err != nil {
		errs = join(errs, "config: "+err.Error())
	}
	if found {
		a.perDisplay = saved.Displays
		a.selected = saved.Selected
	}

	displays, err := a.disp.List()
	if err != nil {
		return join(errs, "displays: "+err.Error())
	}
	if !a.selectionValid(displays) && len(displays) > 0 {
		a.selected = displays[0].ID
	}
	for _, d := range displays {
		if err := a.disp.Apply(d.ID, color.BuildRamp(a.settingsFor(d.ID))); err != nil {
			errs = join(errs, d.ID+": "+err.Error())
		}
	}
	return join(errs, a.applyVendorLocked(a.settingsFor(a.selected)))
}

// GetState returns the current state for the frontend.
func (a *App) GetState() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state("")
}

// SelectDisplay switches the dropdown target and returns its settings.
func (a *App) SelectDisplay(id string) State {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.selected = id
	return a.state(a.save())
}

// Apply clamps, applies and persists the settings for the selected display.
func (a *App) Apply(s color.Settings) State {
	a.mu.Lock()
	defer a.mu.Unlock()

	s = s.Clamp()
	a.perDisplay[a.selected] = s

	var errs string
	if err := a.disp.Apply(a.selected, color.BuildRamp(s)); err != nil {
		errs = join(errs, "gamma: "+err.Error())
	}
	errs = join(errs, a.applyVendorLocked(s), a.save())
	return a.state(errs)
}

// Reset restores neutral settings on the selected display.
func (a *App) Reset() State {
	return a.Apply(color.Defaults(a.defaultSaturationPercent()))
}

// applyVendorLocked pushes the GPU-global controls (saturation, hue).
// Gamma and vendor controls are independent; one failing must not block
// the other, so errors are collected as strings. Caller holds a.mu.
func (a *App) applyVendorLocked(s color.Settings) string {
	if !a.gpu.Available() {
		return ""
	}
	var errs string
	if err := a.gpu.SetVibrance(a.saturationToLevel(s.Saturation)); err != nil {
		errs = join(errs, "saturation: "+err.Error())
	}
	if a.gpu.SupportsHue() {
		if err := a.gpu.SetHue(s.Hue); err != nil {
			errs = join(errs, "hue: "+err.Error())
		}
	}
	return errs
}

// settingsFor returns the saved settings for a display, or neutral
// defaults for one never configured. Caller holds a.mu.
func (a *App) settingsFor(id string) color.Settings {
	if s, ok := a.perDisplay[id]; ok {
		return s
	}
	return color.Defaults(a.defaultSaturationPercent())
}

func (a *App) selectionValid(displays []display.Display) bool {
	for _, d := range displays {
		if d.ID == a.selected {
			return true
		}
	}
	return false
}

func (a *App) save() string {
	err := config.Save(config.File{Selected: a.selected, Displays: a.perDisplay})
	if err != nil {
		return "save: " + err.Error()
	}
	return ""
}

// saturationToLevel maps the UI percent (0..100) onto the raw range the
// driver reports (NvAPI typically 0..63, ADL 0..200).
func (a *App) saturationToLevel(percent int) int {
	info, err := a.gpu.Vibrance()
	if err != nil || info.Max <= info.Min {
		return percent
	}
	return info.Min + (info.Max-info.Min)*percent/100
}

// defaultSaturationPercent converts the driver default saturation level
// back to the UI percent scale.
func (a *App) defaultSaturationPercent() int {
	if !a.gpu.Available() {
		return 0
	}
	info, err := a.gpu.Vibrance()
	if err != nil || info.Max <= info.Min {
		return 0
	}
	return (info.Default - info.Min) * 100 / (info.Max - info.Min)
}

func (a *App) state(errs string) State {
	displays, err := a.disp.List()
	if err != nil {
		errs = join(errs, "displays: "+err.Error())
	}
	if !a.selectionValid(displays) && len(displays) > 0 {
		a.selected = displays[0].ID
	}
	hueMin, hueMax := -180, 180
	hueAvailable := a.gpu.Available() && a.gpu.SupportsHue()
	if hueAvailable {
		hueMin, hueMax = a.gpu.HueRange()
	}
	return State{
		Settings:            a.settingsFor(a.selected),
		Displays:            displays,
		Selected:            a.selected,
		GammaBackend:        a.disp.Describe(),
		VendorBackend:       a.gpu.Describe(),
		SaturationAvailable: a.gpu.Available(),
		HueAvailable:        hueAvailable,
		HueMin:              hueMin,
		HueMax:              hueMax,
		SaturationDefault:   a.defaultSaturationPercent(),
		Errors:              errs,
		Version:             version + " (" + commit + ")",
	}
}

func join(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "; ")
}
