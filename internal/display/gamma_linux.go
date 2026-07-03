//go:build linux

package display

import (
	"fmt"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	"teinte/internal/color"
)

// x11Manager pushes gamma ramps through the RandR extension — the same
// LUT SetDeviceGammaRamp drives on Windows, so the color math is shared
// unchanged. Displays are addressed by RandR output name (eDP-1,
// HDMI-1, ...), which is stable across sessions and human-readable at
// the same time. A fresh X connection per call keeps the manager
// stateless; calls are debounced by the UI so the cost is irrelevant.
type x11Manager struct{}

func New() Manager { return &x11Manager{} }

func (m *x11Manager) Describe() string {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return "X11 RandR gamma — WARNING: Wayland session detected, ramps only affect XWayland, not the real display"
	}
	return "X11 RandR gamma (xgb)"
}

// withOutputs runs fn for every connected output that has an active CRTC.
func withOutputs(fn func(conn *xgb.Conn, name string, crtc randr.Crtc) error) error {
	conn, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("X connection failed: %w", err)
	}
	defer conn.Close()
	if err := randr.Init(conn); err != nil {
		return fmt.Errorf("RandR init failed: %w", err)
	}

	root := xproto.Setup(conn).DefaultScreen(conn).Root
	res, err := randr.GetScreenResourcesCurrent(conn, root).Reply()
	if err != nil {
		return fmt.Errorf("GetScreenResources failed: %w", err)
	}
	for _, output := range res.Outputs {
		info, err := randr.GetOutputInfo(conn, output, 0).Reply()
		if err != nil || info.Connection != randr.ConnectionConnected || info.Crtc == 0 {
			continue
		}
		if err := fn(conn, string(info.Name), info.Crtc); err != nil {
			return err
		}
	}
	return nil
}

func (m *x11Manager) List() ([]Display, error) {
	var displays []Display
	err := withOutputs(func(_ *xgb.Conn, name string, _ randr.Crtc) error {
		displays = append(displays, Display{ID: name, Name: name})
		return nil
	})
	return displays, err
}

func (m *x11Manager) Apply(id string, ramp *color.Ramp) error {
	applied := false
	err := withOutputs(func(conn *xgb.Conn, name string, crtc randr.Crtc) error {
		if name != id {
			return nil
		}
		gs, err := randr.GetCrtcGammaSize(conn, crtc).Reply()
		if err != nil || gs.Size == 0 {
			return fmt.Errorf("GetCrtcGammaSize failed on %s: %v", name, err)
		}
		n := int(gs.Size)
		if err := randr.SetCrtcGammaChecked(conn, crtc, gs.Size,
			ramp.ResampleChannel(0, n),
			ramp.ResampleChannel(1, n),
			ramp.ResampleChannel(2, n)).Check(); err != nil {
			return fmt.Errorf("SetCrtcGamma failed on %s: %w", name, err)
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if !applied {
		return fmt.Errorf("display %q not found (disconnected?)", id)
	}
	return nil
}
