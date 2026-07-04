package main

import "testing"

func TestSemverLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.4.2", "0.5.0", true},
		{"0.5.0", "0.4.2", false},
		{"0.4.2", "0.4.2", false},
		{"0.4.2", "0.4.10", true},
		{"1.0.0", "0.9.9", false},
		{"0.4", "0.4.1", true},
		{"dev", "0.4.2", true}, // "dev" parses as 0 — lower than any release
		{"0.4.2", "", false},   // garbage latest never claims an update
	}
	for _, c := range cases {
		if got := semverLess(c.a, c.b); got != c.want {
			t.Errorf("semverLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
