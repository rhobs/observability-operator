package main

import "testing"

func TestValidLogFileName(t *testing.T) {
	tests := map[string]bool{
		"gather-debug.log": true,
		"gather.log":       true,
		"":                 false,
		".":                false,
		"..":               false,
		"../gather.log":    false,
		"logs/gather.log":  false,
		"/tmp/gather.log":  false,
	}

	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			if got := validLogFileName(name); got != expected {
				t.Fatalf("validLogFileName(%q) = %t, want %t", name, got, expected)
			}
		})
	}
}
