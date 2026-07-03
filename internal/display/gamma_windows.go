//go:build windows

package display

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"teinte/internal/color"
)

var (
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	user32 = syscall.NewLazyDLL("user32.dll")

	procCreateDC           = gdi32.NewProc("CreateDCW")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procSetDeviceGammaRamp = gdi32.NewProc("SetDeviceGammaRamp")
	procEnumDisplayDevices = user32.NewProc("EnumDisplayDevicesW")
)

const displayDeviceActive = 0x00000001

// displayDevice is the Win32 DISPLAY_DEVICEW struct.
type displayDevice struct {
	cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type gdiManager struct{}

// New returns the Windows GDI gamma ramp backend. Displays are
// addressed by adapter device name (\\.\DISPLAY1, ...) — the same key
// CreateDC needs — with the attached monitor string as the label.
func New() Manager { return &gdiManager{} }

func (m *gdiManager) Describe() string { return "Windows GDI gamma ramp (gdi32)" }

func (m *gdiManager) List() ([]Display, error) {
	var displays []Display
	for i := uint32(0); ; i++ {
		var dd displayDevice
		dd.cb = uint32(unsafe.Sizeof(dd))
		r, _, _ := procEnumDisplayDevices.Call(0, uintptr(i), uintptr(unsafe.Pointer(&dd)), 0)
		if r == 0 {
			break
		}
		if dd.StateFlags&displayDeviceActive == 0 {
			continue
		}
		id := syscall.UTF16ToString(dd.DeviceName[:])

		// Second-level enumeration yields the monitor attached to the
		// adapter — a friendlier label than the adapter name.
		label := strings.TrimPrefix(id, `\\.\`)
		var mon displayDevice
		mon.cb = uint32(unsafe.Sizeof(mon))
		namePtr, _ := syscall.UTF16PtrFromString(id)
		if r, _, _ := procEnumDisplayDevices.Call(
			uintptr(unsafe.Pointer(namePtr)), 0, uintptr(unsafe.Pointer(&mon)), 0); r != 0 {
			if s := syscall.UTF16ToString(mon.DeviceString[:]); s != "" {
				label += " · " + s
			}
		}
		displays = append(displays, Display{ID: id, Name: label})
	}
	if len(displays) == 0 {
		return nil, fmt.Errorf("no active display found")
	}
	return displays, nil
}

func (m *gdiManager) Apply(id string, ramp *color.Ramp) error {
	namePtr, err := syscall.UTF16PtrFromString(id)
	if err != nil {
		return err
	}
	hdc, _, _ := procCreateDC.Call(0, uintptr(unsafe.Pointer(namePtr)), 0, 0)
	if hdc == 0 {
		return fmt.Errorf("CreateDC failed for %s (disconnected?)", id)
	}
	defer procDeleteDC.Call(hdc)

	if ok, _, _ := procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(ramp))); ok == 0 {
		return fmt.Errorf("gamma ramp rejected on %s — Windows clamps extreme ramps unless the "+
			"GdiIcmGammaRange registry key is set, see build/windows/enable-full-gamma-range.reg", id)
	}
	return nil
}
