// Package color holds the pure color math: settings model, white point
// conversion, gamma ramp generation and profile presets. No OS calls
// here so everything is unit-testable on any platform.
package color

// ChannelSettings tunes one RGB channel on top of the global sliders
// (multiplied together in the ramp). This is how panel-level looks like
// ASUS "Vivid" are reproduced: e.g. lowering the blue channel gamma.
type ChannelSettings struct {
	Brightness int     `json:"brightness"` // percent, 0..200, neutral 100
	Contrast   int     `json:"contrast"`   // percent, 0..200, neutral 100
	Gamma      float64 `json:"gamma"`      // 0.30..2.80, neutral 1.00
}

// NeutralChannel is the identity per-channel adjustment.
func NeutralChannel() ChannelSettings {
	return ChannelSettings{Brightness: 100, Contrast: 100, Gamma: 1.0}
}

// Settings mirrors the AMD Adrenalin "custom color" controls plus
// per-channel RGB adjustments (NVIDIA-control-panel style).
//
// Temperature/Brightness/Contrast/Gamma and the RGB channels are applied
// through the display gamma ramp (works on any GPU). Saturation/Hue need
// channel mixing which a gamma ramp cannot do; they go through the GPU
// vendor driver when it drives the display.
type Settings struct {
	Temperature int     `json:"temperature"` // white point in Kelvin, 1000..10000, neutral 6500
	Brightness  int     `json:"brightness"`  // percent, 0..200, neutral 100
	Contrast    int     `json:"contrast"`    // percent, 0..200, neutral 100
	Gamma       float64 `json:"gamma"`       // 0.30..2.80, neutral 1.00
	Saturation  int     `json:"saturation"`  // percent of the driver vibrance range, 0..100
	Hue         int     `json:"hue"`         // degrees, -180..180, neutral 0

	Red   ChannelSettings `json:"red"`
	Green ChannelSettings `json:"green"`
	Blue  ChannelSettings `json:"blue"`

	// Profile is the preset these settings came from ("Standard",
	// "Vivid", ...) or "Custom" once any slider is touched manually.
	Profile string `json:"profile"`
}

const (
	MinTemperature = 1000
	MaxTemperature = 10000
	MinPercent     = 0
	MaxPercent     = 200
	MinGamma       = 0.30
	MaxGamma       = 2.80
	MinHue         = -180
	MaxHue         = 180

	CustomProfile = "Custom"
)

// Defaults returns neutral settings. defaultSaturation comes from the
// driver (it reports its own default vibrance level) and is 0 when
// vibrance is unavailable.
func Defaults(defaultSaturation int) Settings {
	return Settings{
		Temperature: 6500,
		Brightness:  100,
		Contrast:    100,
		Gamma:       1.0,
		Saturation:  defaultSaturation,
		Hue:         0,
		Red:         NeutralChannel(),
		Green:       NeutralChannel(),
		Blue:        NeutralChannel(),
		Profile:     "Standard",
	}
}

// Presets returns the built-in profiles; each carries its name in
// Profile. Saturation values are relative to the driver default so they
// behave the same on NvAPI (default 0%) and ADL (default 50%). Every
// preset defines absolute slider values — applying one is deterministic.
func Presets(defaultSaturation int) []Settings {
	sat := func(offset int) int { return clampInt(defaultSaturation+offset, 0, 100) }
	base := func(name string) Settings {
		s := Defaults(defaultSaturation)
		s.Profile = name
		return s
	}

	standard := base("Standard")

	// Saturation offsets are small on purpose: drivers put their
	// default mid-range (NvAPI/ADL 50%), so +20 already means 70%.
	vivid := base("Vivid")
	vivid.Temperature = 7200 // cold white, ASUS "Vivid coldest" territory
	vivid.Contrast = 105
	vivid.Saturation = sat(+20)

	// Tuned on real ASUS TUF panels that ship yellowish and washed
	// out: cold white point + blue channel push to kill the yellow, a
	// touch of gamma depth and a hair of saturation. Saturation stays
	// near the driver default on purpose — vendors put the default
	// mid-range (NvAPI 50%), so big offsets clamp to 100%.
	laptop := base("Laptop")
	laptop.Temperature = 9300
	laptop.Gamma = 1.05
	laptop.Saturation = sat(+2)
	laptop.Red = ChannelSettings{Brightness: 105, Contrast: 105, Gamma: 1.0}
	laptop.Green = ChannelSettings{Brightness: 105, Contrast: 100, Gamma: 1.0}
	laptop.Blue = ChannelSettings{Brightness: 115, Contrast: 100, Gamma: 1.0}

	cinema := base("Cinema")
	cinema.Temperature = 5900 // warm, film-like
	cinema.Gamma = 1.10
	cinema.Saturation = sat(-10)

	gaming := base("Gaming")
	gaming.Contrast = 108
	gaming.Saturation = sat(+12)

	fps := base("FPS Boost")
	fps.Saturation = sat(+45) // near-max vibrance: targets pop, vibranceGUI-style
	fps.Contrast = 105

	reading := base("Reading")
	reading.Temperature = 5000 // paper-like low blue light
	reading.Brightness = 95
	reading.Saturation = sat(-20)

	night := base("Night")
	night.Temperature = 4600 // heavy warm, low blue for evenings
	night.Brightness = 92

	return []Settings{standard, vivid, laptop, cinema, gaming, fps, reading, night}
}

// Clamp returns a copy with every field forced into its valid range.
// Zero-valued channels (configs saved before per-channel support, or a
// fresh struct from the frontend) normalize to neutral — gamma 0 is the
// sentinel since its real minimum is 0.30.
func (s Settings) Clamp() Settings {
	s.Temperature = clampInt(s.Temperature, MinTemperature, MaxTemperature)
	s.Brightness = clampInt(s.Brightness, MinPercent, MaxPercent)
	s.Contrast = clampInt(s.Contrast, MinPercent, MaxPercent)
	s.Saturation = clampInt(s.Saturation, 0, 100)
	s.Hue = clampInt(s.Hue, MinHue, MaxHue)
	s.Gamma = clampGamma(s.Gamma)
	s.Red = s.Red.clamp()
	s.Green = s.Green.clamp()
	s.Blue = s.Blue.clamp()
	if s.Profile == "" {
		s.Profile = CustomProfile
	}
	return s
}

func (c ChannelSettings) clamp() ChannelSettings {
	if c.Gamma == 0 {
		return NeutralChannel()
	}
	c.Brightness = clampInt(c.Brightness, MinPercent, MaxPercent)
	c.Contrast = clampInt(c.Contrast, MinPercent, MaxPercent)
	c.Gamma = clampGamma(c.Gamma)
	return c
}

func clampGamma(g float64) float64 {
	if g < MinGamma {
		return MinGamma
	}
	if g > MaxGamma {
		return MaxGamma
	}
	return g
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
