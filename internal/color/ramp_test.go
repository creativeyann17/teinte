package color

import "testing"

func TestBuildRampIdentityAtDefaults(t *testing.T) {
	ramp := BuildRamp(Defaults(0))
	for c := range 3 {
		for i := range RampSize {
			want := uint16(i * 257) // i/255*65535 is exactly i*257
			if ramp[c][i] != want {
				t.Fatalf("channel %d index %d = %d, want %d", c, i, ramp[c][i], want)
			}
		}
	}
}

func TestBuildRampBrightnessScalesOutput(t *testing.T) {
	s := Defaults(0)
	s.Brightness = 50
	ramp := BuildRamp(s)
	if got, want := ramp[0][RampSize-1], uint16(32768); got != want {
		t.Errorf("50%% brightness top entry = %d, want %d", got, want)
	}
}

func TestBuildRampZeroContrastIsFlatMidGray(t *testing.T) {
	s := Defaults(0)
	s.Contrast = 0
	ramp := BuildRamp(s)
	first := ramp[0][0]
	if first < 32000 || first > 33500 {
		t.Fatalf("zero contrast entry = %d, want ~32768 (mid gray)", first)
	}
	for i := 1; i < RampSize; i++ {
		if ramp[0][i] != first {
			t.Fatalf("zero contrast ramp not flat at index %d: %d != %d", i, ramp[0][i], first)
		}
	}
}

func TestBuildRampMonotonicForAnySaneSettings(t *testing.T) {
	cases := []Settings{
		{Temperature: 3000, Brightness: 120, Contrast: 140, Gamma: 0.8},
		{Temperature: 9000, Brightness: 80, Contrast: 60, Gamma: 2.2},
		{Temperature: 6500, Brightness: 200, Contrast: 200, Gamma: 2.8},
	}
	for _, s := range cases {
		ramp := BuildRamp(s)
		for c := range 3 {
			for i := 1; i < RampSize; i++ {
				if ramp[c][i] < ramp[c][i-1] {
					t.Fatalf("%+v channel %d decreasing at %d", s, c, i)
				}
			}
		}
	}
}

func TestBuildRampWarmTemperatureLowersBlueOnly(t *testing.T) {
	s := Defaults(0)
	s.Temperature = 3500
	ramp := BuildRamp(s)
	top := RampSize - 1
	if ramp[0][top] != 65535 {
		t.Errorf("warm red top = %d, want 65535", ramp[0][top])
	}
	if ramp[2][top] >= ramp[1][top] || ramp[1][top] >= ramp[0][top] {
		t.Errorf("warm ramp tops not ordered b<g<r: r=%d g=%d b=%d",
			ramp[0][top], ramp[1][top], ramp[2][top])
	}
}

func TestClamp(t *testing.T) {
	s := Settings{Temperature: 99999, Brightness: -5, Contrast: 999, Gamma: 0, Saturation: 500, Hue: -999}
	c := s.Clamp()
	if c.Temperature != MaxTemperature || c.Brightness != 0 || c.Contrast != MaxPercent ||
		c.Gamma != MinGamma || c.Saturation != 100 || c.Hue != MinHue {
		t.Errorf("clamp wrong: %+v", c)
	}
}
