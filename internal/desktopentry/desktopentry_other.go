//go:build !linux

package desktopentry

// Install is a no-op on every platform except Linux.
//
//   - Windows: a Start Menu entry belongs to an installer (NSIS/MSIX),
//     not a runtime poke at the user's profile — and the autostart Run
//     key already covers login start.
//   - macOS: the right unit of installation is a .app bundle produced
//     by `wails build`, not a runtime-generated launcher.
func Install(_ []byte) error { return nil }
