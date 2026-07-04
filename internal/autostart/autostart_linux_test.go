package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestSetEnableDisableRoundTrip(t *testing.T) {
	if Enabled() {
		t.Skip("autostart already enabled by the user, not touching it")
	}
	if err := Set(true); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !Enabled() {
		t.Fatal("not enabled after Set(true)")
	}
	p, _ := entryPath()
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "--hidden") {
		t.Errorf("entry missing --hidden: %s", data)
	}
	if err := Set(false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if Enabled() {
		t.Fatal("still enabled after Set(false)")
	}
}
