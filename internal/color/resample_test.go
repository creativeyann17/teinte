package color

import "testing"

func TestResampleChannelSameSizeIsIdentity(t *testing.T) {
	ramp := BuildRamp(Defaults(0))
	out := ramp.ResampleChannel(0, RampSize)
	for i := range RampSize {
		if out[i] != ramp[0][i] {
			t.Fatalf("index %d: %d != %d", i, out[i], ramp[0][i])
		}
	}
}

func TestResampleChannelUpscaleKeepsEndpointsAndMonotonicity(t *testing.T) {
	ramp := BuildRamp(Defaults(0))
	out := ramp.ResampleChannel(1, 1024)
	if out[0] != ramp[1][0] || out[1023] != ramp[1][RampSize-1] {
		t.Fatalf("endpoints wrong: %d..%d", out[0], out[1023])
	}
	for i := 1; i < len(out); i++ {
		if out[i] < out[i-1] {
			t.Fatalf("not monotonic at %d", i)
		}
	}
}

func TestResampleChannelDownscale(t *testing.T) {
	ramp := BuildRamp(Defaults(0))
	out := ramp.ResampleChannel(2, 64)
	if len(out) != 64 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0] != 0 || out[63] != 65535 {
		t.Fatalf("endpoints wrong for identity ramp: %d..%d", out[0], out[63])
	}
}
