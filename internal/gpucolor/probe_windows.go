//go:build windows

package gpucolor

// New probes the vendor backends in order and returns the first one
// with a driven display. The MUX position decides the winner: NVIDIA
// direct mode → NvAPI, hybrid/AMD mode → ADL. When neither answers, the
// combined reasons are surfaced in the UI.
func New() Controller {
	nv := newNvAPI()
	if nv.Available() {
		return nv
	}
	adl := newADL()
	if adl.Available() {
		return adl
	}
	return &unavailable{reason: "NvAPI: " + nv.Describe() + " | ADL: " + adl.Describe()}
}
