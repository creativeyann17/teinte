package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	goruntime "runtime"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIconPNG []byte

//go:embed build/windows/icon.ico
var trayIconICO []byte

// trayIcon picks the icon format the OS tray expects.
func trayIcon() []byte {
	if goruntime.GOOS == "windows" {
		return trayIconICO
	}
	return trayIconPNG
}

// Injected at release time via -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = "none"
)

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Headless mode for autostart: apply the saved colors, exit. No
	// window, no tray — gamma ramps and vibrance outlive the process.
	if slices.Contains(os.Args[1:], "--apply") {
		if errs := app.loadAndApplyAll(); errs != "" {
			fmt.Fprintln(os.Stderr, "teinte --apply:", errs)
			os.Exit(1)
		}
		return
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Teinte — display color control",
		Width:     620,
		Height:    720,
		MinWidth:  560,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 22, G: 24, B: 29, A: 1},
		OnStartup:        app.startup,
		// Close-to-tray by default: the X button hides the window so
		// the saved colors keep being guarded by a live process; real
		// exit is the tray menu's Quit.
		OnBeforeClose: func(ctx context.Context) bool {
			if app.quitting {
				return false
			}
			runtime.WindowHide(ctx)
			return true
		},
		Bind: []any{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
