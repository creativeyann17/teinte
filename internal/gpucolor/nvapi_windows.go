//go:build windows

package gpucolor

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Community-documented NvAPI function IDs, resolved through the exported
// nvapi_QueryInterface. Not in the public SDK but stable for 10+ years.
const (
	idInitialize              = 0x0150E828
	idEnumNvidiaDisplayHandle = 0x9ABDD40D
	idGetDVCInfoEx            = 0x0E45002D
	idSetDVCLevelEx           = 0x4A82C2B1
	idSetHUEAngle             = 0xF5A0F22C
)

// dvcInfoEx is NV_DISPLAY_DVC_INFO_EX. version encodes struct size and
// revision, NvAPI rejects the call if it does not match.
type dvcInfoEx struct {
	version      uint32
	currentLevel int32
	minLevel     int32
	maxLevel     int32
	defaultLevel int32
}

func dvcVersion() uint32 {
	return uint32(unsafe.Sizeof(dvcInfoEx{})) | (1 << 16)
}

type nvController struct {
	getDVC   uintptr
	setDVC   uintptr
	setHue   uintptr
	displays []uintptr
}

// newNvAPI loads nvapi64.dll and enumerates NVIDIA-driven displays.
// Every failure path degrades to an unavailable controller with a human
// readable reason so the probe chain can move on to the next backend.
func newNvAPI() Controller {
	dll := syscall.NewLazyDLL("nvapi64.dll")
	if err := dll.Load(); err != nil {
		return &unavailable{reason: "nvapi64.dll not found (no NVIDIA driver)"}
	}
	qi := dll.NewProc("nvapi_QueryInterface")
	if err := qi.Find(); err != nil {
		return &unavailable{reason: "nvapi_QueryInterface export missing"}
	}
	resolve := func(id uint32) uintptr {
		ptr, _, _ := qi.Call(uintptr(id))
		return ptr
	}

	fnInit := resolve(idInitialize)
	fnEnum := resolve(idEnumNvidiaDisplayHandle)
	c := &nvController{
		getDVC: resolve(idGetDVCInfoEx),
		setDVC: resolve(idSetDVCLevelEx),
		setHue: resolve(idSetHUEAngle),
	}
	if fnInit == 0 || fnEnum == 0 || c.getDVC == 0 || c.setDVC == 0 || c.setHue == 0 {
		return &unavailable{reason: "driver does not expose the DVC/HUE interfaces"}
	}
	if status, _, _ := syscall.SyscallN(fnInit); status != 0 {
		return &unavailable{reason: fmt.Sprintf("NvAPI_Initialize failed (status %d)", int32(status))}
	}

	for i := uintptr(0); ; i++ {
		var handle uintptr
		status, _, _ := syscall.SyscallN(fnEnum, i, uintptr(unsafe.Pointer(&handle)))
		if status != 0 {
			break
		}
		c.displays = append(c.displays, handle)
	}
	if len(c.displays) == 0 {
		return &unavailable{reason: "no display driven by the NVIDIA GPU"}
	}
	return c
}

func (c *nvController) Available() bool      { return true }
func (c *nvController) SupportsHue() bool    { return true }
func (c *nvController) HueRange() (int, int) { return -180, 180 }

func (c *nvController) Describe() string {
	return fmt.Sprintf("NvAPI digital vibrance + hue (%d display(s))", len(c.displays))
}

func (c *nvController) Vibrance() (VibranceInfo, error) {
	info := dvcInfoEx{version: dvcVersion()}
	status, _, _ := syscall.SyscallN(c.getDVC, c.displays[0], 0, uintptr(unsafe.Pointer(&info)))
	if status != 0 {
		return VibranceInfo{}, fmt.Errorf("GetDVCInfoEx failed (status %d)", int32(status))
	}
	return VibranceInfo{
		Current: int(info.currentLevel),
		Min:     int(info.minLevel),
		Max:     int(info.maxLevel),
		Default: int(info.defaultLevel),
	}, nil
}

func (c *nvController) SetVibrance(level int) error {
	for _, h := range c.displays {
		info := dvcInfoEx{version: dvcVersion()}
		if status, _, _ := syscall.SyscallN(c.getDVC, h, 0, uintptr(unsafe.Pointer(&info))); status != 0 {
			return fmt.Errorf("GetDVCInfoEx failed (status %d)", int32(status))
		}
		info.currentLevel = int32(level)
		if status, _, _ := syscall.SyscallN(c.setDVC, h, 0, uintptr(unsafe.Pointer(&info))); status != 0 {
			return fmt.Errorf("SetDVCLevelEx failed (status %d)", int32(status))
		}
	}
	return nil
}

func (c *nvController) SetHue(degrees int) error {
	degrees = ((degrees % 360) + 360) % 360
	for _, h := range c.displays {
		if status, _, _ := syscall.SyscallN(c.setHue, h, 0, uintptr(degrees)); status != 0 {
			return fmt.Errorf("SetHUEAngle failed (status %d)", int32(status))
		}
	}
	return nil
}
