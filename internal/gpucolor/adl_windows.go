//go:build windows

package gpucolor

import (
	"fmt"
	"syscall"
	"unsafe"
)

// ADL (AMD Display Library) backend — the official, documented AMD API
// behind the Adrenalin "custom color" panel (GPUOpen ADL SDK). All
// functions are exported by name from atiadlxx.dll and return ADL_OK=0.
//
// Color controls are addressed per (adapter, display) pair with a color
// type bit; the driver reports current/default/min/max per control, so
// ranges are never guessed (saturation is typically 0..200 default 100,
// hue typically -30..30 default 0 — exactly the Adrenalin sliders).
const (
	adlOK = 0

	adlColorSaturation = 1 << 2
	adlColorHue        = 1 << 3

	adlMaxPath = 256

	adlDisplayConnected = 0x1
	adlDisplayMapped    = 0x2
)

// adlDisplayInfo is ADLDisplayInfo from adl_structures.h (552 bytes).
type adlDisplayInfo struct {
	logicalIndex         int32
	physicalIndex        int32
	logicalAdapterIndex  int32
	physicalAdapterIndex int32
	controllerIndex      int32
	name                 [adlMaxPath]byte
	manufacturer         [adlMaxPath]byte
	displayType          int32
	outputType           int32
	connector            int32
	infoMask             int32
	infoValue            int32
}

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procGlobalAlloc = kernel32.NewProc("GlobalAlloc")
	procGlobalFree  = kernel32.NewProc("GlobalFree")

	// ADL_Main_Control_Create requires the caller to provide the
	// allocator it uses for out-buffers (e.g. the display list).
	// GPTR = zero-initialized fixed memory.
	adlMalloc = syscall.NewCallback(func(size uintptr) uintptr {
		p, _, _ := procGlobalAlloc.Call(0x0040 /* GPTR */, size&0xFFFFFFFF)
		return p
	})
)

type adlTarget struct{ adapter, display int32 }

type adlController struct {
	colorGet uintptr
	colorSet uintptr
	targets  []adlTarget
	hasHue   bool
}

func newADL() Controller {
	dll := syscall.NewLazyDLL("atiadlxx.dll")
	if err := dll.Load(); err != nil {
		return &unavailable{reason: "atiadlxx.dll not found (no AMD driver)"}
	}
	procs := map[string]*syscall.LazyProc{}
	for _, name := range []string{
		"ADL_Main_Control_Create",
		"ADL_Adapter_NumberOfAdapters_Get",
		"ADL_Adapter_Active_Get",
		"ADL_Display_DisplayInfo_Get",
		"ADL_Display_Color_Get",
		"ADL_Display_Color_Set",
	} {
		p := dll.NewProc(name)
		if err := p.Find(); err != nil {
			return &unavailable{reason: name + " export missing"}
		}
		procs[name] = p
	}

	if status, _, _ := procs["ADL_Main_Control_Create"].Call(adlMalloc, 1 /* connected adapters only */); int32(status) != adlOK {
		return &unavailable{reason: fmt.Sprintf("ADL_Main_Control_Create failed (status %d)", int32(status))}
	}

	c := &adlController{
		colorGet: procs["ADL_Display_Color_Get"].Addr(),
		colorSet: procs["ADL_Display_Color_Set"].Addr(),
	}

	var numAdapters int32
	procs["ADL_Adapter_NumberOfAdapters_Get"].Call(uintptr(unsafe.Pointer(&numAdapters)))
	for adapter := int32(0); adapter < numAdapters; adapter++ {
		var active int32
		procs["ADL_Adapter_Active_Get"].Call(uintptr(adapter), uintptr(unsafe.Pointer(&active)))
		if active == 0 {
			continue
		}

		var count int32
		var buf *adlDisplayInfo // array allocated by the driver through adlMalloc
		status, _, _ := procs["ADL_Display_DisplayInfo_Get"].Call(
			uintptr(adapter), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf)), 0)
		if int32(status) != adlOK || buf == nil {
			continue
		}
		size := unsafe.Sizeof(adlDisplayInfo{})
		for i := int32(0); i < count; i++ {
			info := (*adlDisplayInfo)(unsafe.Add(unsafe.Pointer(buf), uintptr(i)*size))
			wanted := int32(adlDisplayConnected | adlDisplayMapped)
			// The list repeats displays across logical adapters; keep
			// each physical display once, on its own adapter.
			if info.infoValue&wanted == wanted && info.physicalAdapterIndex == adapter {
				c.targets = append(c.targets, adlTarget{adapter: adapter, display: info.logicalIndex})
			}
		}
		procGlobalFree.Call(uintptr(unsafe.Pointer(buf)))
	}
	if len(c.targets) == 0 {
		return &unavailable{reason: "no display driven by the AMD GPU"}
	}

	// The saturation control must answer on at least the first display,
	// otherwise this driver/display combo has no color support at all.
	if _, err := c.color(c.targets[0], adlColorSaturation); err != nil {
		return &unavailable{reason: "driver rejected the saturation control: " + err.Error()}
	}
	_, hueErr := c.color(c.targets[0], adlColorHue)
	c.hasHue = hueErr == nil
	return c
}

// color wraps ADL_Display_Color_Get for one control on one display.
func (c *adlController) color(t adlTarget, colorType int32) (VibranceInfo, error) {
	var current, def, min, max, step int32
	status, _, _ := syscall.SyscallN(c.colorGet,
		uintptr(t.adapter), uintptr(t.display), uintptr(colorType),
		uintptr(unsafe.Pointer(&current)), uintptr(unsafe.Pointer(&def)),
		uintptr(unsafe.Pointer(&min)), uintptr(unsafe.Pointer(&max)),
		uintptr(unsafe.Pointer(&step)))
	if int32(status) != adlOK {
		return VibranceInfo{}, fmt.Errorf("ADL_Display_Color_Get(%d) failed (status %d)", colorType, int32(status))
	}
	return VibranceInfo{Current: int(current), Min: int(min), Max: int(max), Default: int(def)}, nil
}

func (c *adlController) setColor(colorType int32, value int) error {
	for _, t := range c.targets {
		status, _, _ := syscall.SyscallN(c.colorSet,
			uintptr(t.adapter), uintptr(t.display), uintptr(colorType), uintptr(int32(value)))
		if int32(status) != adlOK {
			return fmt.Errorf("ADL_Display_Color_Set(%d) failed (status %d)", colorType, int32(status))
		}
	}
	return nil
}

func (c *adlController) Available() bool   { return true }
func (c *adlController) SupportsHue() bool { return c.hasHue }

func (c *adlController) HueRange() (int, int) {
	if info, err := c.color(c.targets[0], adlColorHue); err == nil {
		return info.Min, info.Max
	}
	return -30, 30
}

func (c *adlController) Describe() string {
	return fmt.Sprintf("AMD ADL saturation + hue (%d display(s))", len(c.targets))
}

func (c *adlController) Vibrance() (VibranceInfo, error) {
	return c.color(c.targets[0], adlColorSaturation)
}

func (c *adlController) SetVibrance(level int) error {
	return c.setColor(adlColorSaturation, level)
}

func (c *adlController) SetHue(degrees int) error {
	min, max := c.HueRange()
	if degrees < min {
		degrees = min
	} else if degrees > max {
		degrees = max
	}
	return c.setColor(adlColorHue, degrees)
}
