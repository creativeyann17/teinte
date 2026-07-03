package color

import "math"

// RampSize is the number of LUT entries per channel expected by
// SetDeviceGammaRamp (WORD[3][256]).
const RampSize = 256

// Ramp is the hardware LUT layout: [R|G|B][256] 16-bit values,
// memory-compatible with the Win32 gamma ramp structure.
type Ramp [3][RampSize]uint16

// BuildRamp computes the LUT for the given settings.
//
// Per-channel pipeline, input x in [0,1]:
//  1. contrast: pivot around mid gray, x = (x-0.5)*c + 0.5
//  2. brightness: linear gain, x = x*b
//  3. gamma: x = x^(1/g)
//  4. white point: x = x*mult[channel]
func BuildRamp(s Settings) *Ramp {
	s = s.Clamp()

	mr, mg, mb := TemperatureMultipliers(float64(s.Temperature))
	mult := [3]float64{mr, mg, mb}
	contrast := float64(s.Contrast) / 100
	brightness := float64(s.Brightness) / 100
	invGamma := 1 / s.Gamma

	var ramp Ramp
	for i := range RampSize {
		x := float64(i) / (RampSize - 1)
		x = (x-0.5)*contrast + 0.5
		x = clamp01(x * brightness)
		x = math.Pow(x, invGamma)
		for c := range 3 {
			ramp[c][i] = uint16(clamp01(x*mult[c])*65535 + 0.5)
		}
	}
	return &ramp
}
