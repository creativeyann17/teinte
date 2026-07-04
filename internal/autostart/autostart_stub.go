//go:build !windows && !linux

package autostart

import "fmt"

func Available() bool { return false }
func Enabled() bool   { return false }

func Set(bool) error {
	return fmt.Errorf("autostart not supported on this platform")
}
