// Package tray owns the system tray icon and its menu. It stays
// deliberately dumb: it knows how to draw the icon and invoke
// callbacks. The wiring to window show/hide and quit lives in main.go
// so the tray has zero knowledge of Wails internals (same pattern as
// mist-drive desktop).
package tray

import (
	"fyne.io/systray"
)

// Callbacks is the minimal surface the tray needs from the app.
type Callbacks struct {
	OnOpen func() // show the main window
	OnQuit func() // graceful full exit
}

// Start launches the systray. fyne.io/systray runs its onReady callback
// on a background goroutine on Linux/Windows, so this is non-blocking.
func Start(icon []byte, tooltip string, cb Callbacks) {
	onReady := func() {
		systray.SetIcon(icon)
		systray.SetTooltip(tooltip)

		open := systray.AddMenuItem("Open Teinte", "Show the main window")
		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Exit Teinte (colors stay until logout)")

		go func() {
			for {
				select {
				case <-open.ClickedCh:
					if cb.OnOpen != nil {
						cb.OnOpen()
					}
				case <-quit.ClickedCh:
					if cb.OnQuit != nil {
						cb.OnQuit()
					}
					systray.Quit()
					return
				}
			}
		}()
	}
	// onExit is a no-op — OnQuit does the app-level cleanup before
	// calling systray.Quit().
	go systray.Run(onReady, func() {})
}
