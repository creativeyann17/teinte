package color

import (
	"math"
	"testing"
)

func TestTemperatureMultipliersNeutralAt6500(t *testing.T) {
	r, g, b := TemperatureMultipliers(6500)
	for name, v := range map[string]float64{"r": r, "g": g, "b": b} {
		if math.Abs(v-1) > 1e-9 {
			t.Errorf("6500K %s = %f, want 1.0", name, v)
		}
	}
}

func TestTemperatureMultipliersWarm(t *testing.T) {
	r, g, b := TemperatureMultipliers(3000)
	if r < 0.99 {
		t.Errorf("3000K red = %f, want ~1.0 (red stays full when warm)", r)
	}
	if g >= r {
		t.Errorf("3000K green %f not below red %f", g, r)
	}
	if b >= g {
		t.Errorf("3000K blue %f not below green %f", b, g)
	}
	if b > 0.75 {
		t.Errorf("3000K blue = %f, want strongly reduced", b)
	}
}

func TestTemperatureMultipliersCool(t *testing.T) {
	r, _, b := TemperatureMultipliers(10000)
	if r >= 1 {
		t.Errorf("10000K red = %f, want < 1 (red reduced when cool)", r)
	}
	if b < 1 {
		t.Errorf("10000K blue = %f, want >= 1", b)
	}
}
