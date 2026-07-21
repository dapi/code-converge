package main

import "testing"

func TestValidVersion(t *testing.T) {
	for _, value := range []string{"dev", "test", "1.0.0-next", "1.0.0-rc1"} {
		if !validVersion(value) {
			t.Errorf("valid version rejected: %q", value)
		}
	}
	for _, value := range []string{"", " ", "../1", `one\\two`, "has space", "line\nbreak", "tab\tvalue"} {
		if validVersion(value) {
			t.Errorf("invalid version accepted: %q", value)
		}
	}
}
