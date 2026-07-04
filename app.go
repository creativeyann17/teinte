package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"teinte/internal/autostart"
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

	selected     string
	perDisplay   map[string]color.Settings
	userProfiles map[string]color.Settings
	quitting     bool // set by the tray Quit so OnBeforeClose lets the app die
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
	Presets             []string          `json:"presets"`
	UserPresets         []string          `json:"userPresets"`
	Autostart           bool              `json:"autostart"`
	AutostartAvailable  bool              `json:"autostartAvailable"`
	Errors              string            `json:"errors"`
	Version             string            `json:"version"`
}

func NewApp() *App {
	return &App{
		disp:         display.New(),
		gpu:          gpucolor.New(),
		perDisplay:   map[string]color.Settings{},
		userProfiles: map[string]color.Settings{},
	}
}

// showWindow restores the main window from tray. WindowShow brings
// back a hidden window, WindowUnminimise covers one minimised before
// hiding; both are cheap no-ops when already in the target state. The
// nil guard protects the second-instance callback, which can in theory
// fire before startup has stored the context.
func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

// startup wires the tray and reapplies the saved colors, so they come
// back automatically after reboot/login.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	tray.Start(trayIcon(), "Teinte", tray.Callbacks{
		OnOpen: a.showWindow,
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
		a.adoptLocked(saved)
	}
	return join(errs, a.applyAllLocked())
}

// adoptLocked replaces the in-memory state with a loaded/imported
// config. Caller holds a.mu.
func (a *App) adoptLocked(f config.File) {
	a.perDisplay = f.Displays
	a.selected = f.Selected
	a.userProfiles = f.UserProfiles
	if a.perDisplay == nil {
		a.perDisplay = map[string]color.Settings{}
	}
	if a.userProfiles == nil {
		a.userProfiles = map[string]color.Settings{}
	}
}

// applyAllLocked pushes the in-memory settings to every connected
// display plus the GPU-global controls. Caller holds a.mu.
func (a *App) applyAllLocked() string {
	var errs string
	displays, err := a.disp.List()
	if err != nil {
		return "displays: " + err.Error()
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

// ApplyPreset applies a built-in or user profile to the selected
// display. Unknown names are a no-op returning current state.
func (a *App) ApplyPreset(name string) State {
	for _, p := range color.Presets(a.defaultSaturationPercent()) {
		if p.Profile == name {
			return a.Apply(p)
		}
	}
	a.mu.Lock()
	s, ok := a.userProfiles[name]
	a.mu.Unlock()
	if ok {
		return a.Apply(s)
	}
	return a.GetState()
}

// maxUserProfiles caps the "Yours" section: the sidebar has fixed
// space, and export/import is the intended path for collections.
const maxUserProfiles = 3

// SaveProfile stores the selected display's current settings as a named
// user profile (overwriting an existing one with the same name). New
// names are rejected once the cap is reached; overwrites always pass.
func (a *App) SaveProfile(name string) State {
	name = strings.TrimSpace(name)

	a.mu.Lock()
	defer a.mu.Unlock()
	if name == "" {
		return a.state("profile name is empty")
	}
	if _, exists := a.userProfiles[name]; !exists && len(a.userProfiles) >= maxUserProfiles {
		return a.state("max 3 profiles — overwrite or delete one")
	}
	if name == color.CustomProfile {
		return a.state(`"Custom" is reserved`)
	}
	for _, p := range color.Presets(0) {
		if p.Profile == name {
			return a.state(`"` + name + `" is a built-in profile, pick another name`)
		}
	}

	s := a.settingsFor(a.selected)
	s.Profile = name
	a.userProfiles[name] = s
	a.perDisplay[a.selected] = s // the saved profile becomes the active one
	return a.state(a.save())
}

const (
	websiteURL  = "https://github.com/creativeyann17/teinte"
	releasesURL = websiteURL + "/releases"
	latestAPI   = "https://api.github.com/repos/creativeyann17/teinte/releases/latest"
)

// UpdateInfo tells the frontend whether a newer release exists.
type UpdateInfo struct {
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	URL             string `json:"url"`
}

// CheckUpdate asks GitHub for the latest release tag. Called async by
// the frontend after first paint so a slow network never blocks the UI.
// Offline/rate-limited = silently no update. Dev builds always report
// an update so the red path is easy to eyeball.
func (a *App) CheckUpdate() UpdateInfo {
	info := UpdateInfo{URL: releasesURL, UpdateAvailable: version == "dev"}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(latestAPI)
	if err != nil {
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info
	}
	info.Latest = strings.TrimPrefix(release.TagName, "v")
	info.UpdateAvailable = version == "dev" || semverLess(version, info.Latest)
	return info
}

// semverLess reports a < b for dotted numeric versions ("0.4.2").
// Non-numeric parts compare as 0, so garbage never claims an update.
func semverLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		na, nb := 0, 0
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na != nb {
			return na < nb
		}
	}
	return false
}

// SetAutostart registers/unregisters starting Teinte (hidden, in the
// tray) at login.
func (a *App) SetAutostart(enable bool) State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := autostart.Set(enable); err != nil {
		return a.state("autostart: " + err.Error())
	}
	return a.state("")
}

// DeleteProfile removes a user profile (built-ins are untouchable).
func (a *App) DeleteProfile(name string) State {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.userProfiles, name)
	return a.state(a.save())
}

// ExportConfig writes the full settings file (displays + user profiles)
// to a user-chosen path. Cancelled dialog = no-op.
func (a *App) ExportConfig() State {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Teinte settings",
		DefaultFilename: "teinte-settings.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		return a.state("export: " + err.Error())
	}
	if path == "" {
		return a.state("")
	}
	data, err := json.MarshalIndent(config.File{
		Selected:     a.selected,
		Displays:     a.perDisplay,
		UserProfiles: a.userProfiles,
	}, "", "  ")
	if err != nil {
		return a.state("export: " + err.Error())
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return a.state("export: " + err.Error())
	}
	return a.state("")
}

// ImportConfig loads a previously exported file, applies it to the
// hardware and persists it. Cancelled dialog = no-op.
func (a *App) ImportConfig() State {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import Teinte settings",
		Filters: []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		return a.state("import: " + err.Error())
	}
	if path == "" {
		return a.state("")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return a.state("import: " + err.Error())
	}
	var f config.File
	if err := json.Unmarshal(data, &f); err != nil {
		return a.state("import: not valid JSON: " + err.Error())
	}
	if f.Displays == nil && f.UserProfiles == nil {
		return a.state("import: not a Teinte settings file")
	}
	a.adoptLocked(f.Normalize())
	return a.state(join(a.applyAllLocked(), a.save()))
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
	err := config.Save(config.File{
		Selected:     a.selected,
		Displays:     a.perDisplay,
		UserProfiles: a.userProfiles,
	})
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
	presetNames := []string{}
	for _, p := range color.Presets(0) {
		presetNames = append(presetNames, p.Profile)
	}
	userNames := []string{}
	for name := range a.userProfiles {
		userNames = append(userNames, name)
	}
	sort.Strings(userNames)
	return State{
		Settings:            a.settingsFor(a.selected),
		Displays:            displays,
		Selected:            a.selected,
		Presets:             presetNames,
		UserPresets:         userNames,
		Autostart:           autostart.Enabled(),
		AutostartAvailable:  autostart.Available(),
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
