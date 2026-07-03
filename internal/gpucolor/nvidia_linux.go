//go:build linux

package gpucolor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Linux NVIDIA drivers expose digital vibrance as the X11 attribute
// "DigitalVibrance" (range -1024..1023, default 0), driven here through
// the nvidia-settings CLI. There is no hue control on Linux, and no ADL
// (AMD offers no equivalent Linux control either).
//
// Like on Windows, this only works when the NVIDIA GPU actually drives
// a display; in PRIME offload mode the query fails and we degrade to
// gamma-only.
const (
	vibranceMin     = -1024
	vibranceMax     = 1023
	vibranceDefault = 0
)

type linuxVibrance struct{}

func New() Controller {
	if _, err := queryVibrance(); err != nil {
		return &unavailable{reason: fmt.Sprintf(
			"nvidia-settings DigitalVibrance query failed (NVIDIA not driving a display?): %v", err)}
	}
	return &linuxVibrance{}
}

func queryVibrance() (int, error) {
	out, err := exec.Command("nvidia-settings", "-q", "DigitalVibrance", "-t").Output()
	if err != nil {
		return 0, err
	}
	// One line per NVIDIA-driven display; they are kept in sync by SetVibrance.
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if line == "" {
		return 0, fmt.Errorf("empty reply")
	}
	return strconv.Atoi(line)
}

func (l *linuxVibrance) Available() bool      { return true }
func (l *linuxVibrance) SupportsHue() bool    { return false }
func (l *linuxVibrance) HueRange() (int, int) { return 0, 0 }

func (l *linuxVibrance) Describe() string {
	return "NVIDIA digital vibrance via nvidia-settings (no hue on Linux)"
}

func (l *linuxVibrance) Vibrance() (VibranceInfo, error) {
	current, err := queryVibrance()
	if err != nil {
		return VibranceInfo{}, err
	}
	return VibranceInfo{Current: current, Min: vibranceMin, Max: vibranceMax, Default: vibranceDefault}, nil
}

func (l *linuxVibrance) SetVibrance(level int) error {
	out, err := exec.Command("nvidia-settings", "-a", fmt.Sprintf("DigitalVibrance=%d", level)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nvidia-settings failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (l *linuxVibrance) SetHue(int) error {
	return fmt.Errorf("hue is not supported on Linux")
}
