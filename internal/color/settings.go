// Package color holds the pure color math: settings model, white point
// conversion and gamma ramp generation. No OS calls here so everything
// is unit-testable on any platform.
package color

// Settings mirrors the AMD Adrenalin "custom color" controls.
//
// Temperature/Brightness/Contrast/Gamma are applied through the display
// gamma ramp (works on any GPU). Saturation/Hue need channel mixing which
// a gamma ramp cannot do; they are applied through NvAPI digital vibrance
// and hue angle when an NVIDIA GPU drives the display.
type Settings struct {
	Temperature int     `json:"temperature"` // white point in Kelvin, 1000..10000, neutral 6500
	Brightness  int     `json:"brightness"`  // percent, 0..200, neutral 100
	Contrast    int     `json:"contrast"`    // percent, 0..200, neutral 100
	Gamma       float64 `json:"gamma"`       // 0.30..2.80, neutral 1.00
	Saturation  int     `json:"saturation"`  // percent of the driver vibrance range, 0..100
	Hue         int     `json:"hue"`         // degrees, -180..180, neutral 0
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
)

// Defaults returns neutral settings. defaultSaturation comes from the
// driver (NvAPI reports its own default vibrance level) and is 0 when
// vibrance is unavailable.
func Defaults(defaultSaturation int) Settings {
	return Settings{
		Temperature: 6500,
		Brightness:  100,
		Contrast:    100,
		Gamma:       1.0,
		Saturation:  defaultSaturation,
		Hue:         0,
	}
}

// Clamp returns a copy with every field forced into its valid range.
func (s Settings) Clamp() Settings {
	s.Temperature = clampInt(s.Temperature, MinTemperature, MaxTemperature)
	s.Brightness = clampInt(s.Brightness, MinPercent, MaxPercent)
	s.Contrast = clampInt(s.Contrast, MinPercent, MaxPercent)
	s.Saturation = clampInt(s.Saturation, 0, 100)
	s.Hue = clampInt(s.Hue, MinHue, MaxHue)
	if s.Gamma < MinGamma {
		s.Gamma = MinGamma
	} else if s.Gamma > MaxGamma {
		s.Gamma = MaxGamma
	}
	return s
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
