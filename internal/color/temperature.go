package color

import "math"

// kelvinToRGB converts a color temperature to RGB in [0,1] using Tanner
// Helland's blackbody approximation (same family of curves f.lux and
// redshift use).
func kelvinToRGB(kelvin float64) (r, g, b float64) {
	t := kelvin / 100

	if t <= 66 {
		r = 255
	} else {
		r = 329.698727446 * math.Pow(t-60, -0.1332047592)
	}

	if t <= 66 {
		g = 99.4708025861*math.Log(t) - 161.1195681661
	} else {
		g = 288.1221695283 * math.Pow(t-60, -0.0755148492)
	}

	switch {
	case t >= 66:
		b = 255
	case t <= 19:
		b = 0
	default:
		b = 138.5177312231*math.Log(t-10) - 305.0447927307
	}

	return clamp01(r / 255), clamp01(g / 255), clamp01(b / 255)
}

// TemperatureMultipliers returns per-channel gains for the given white
// point, normalized so 6500K is exactly (1,1,1) — the panel's native
// white stays untouched at the neutral setting.
func TemperatureMultipliers(kelvin float64) (r, g, b float64) {
	nr, ng, nb := kelvinToRGB(6500)
	r, g, b = kelvinToRGB(kelvin)
	return r / nr, g / ng, b / nb
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
