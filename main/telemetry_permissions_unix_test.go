//go:build linux || darwin

package main

import (
	"os"
	"testing"
)

func denyTelemetryWrites(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	originalMode := info.Mode()
	t.Cleanup(func() {
		if err := os.Chmod(path, originalMode); err != nil && !os.IsNotExist(err) {
			t.Errorf("restore telemetry fixture permissions: %v", err)
		}
	})
	if err := os.Chmod(path, originalMode&^0222); err != nil {
		t.Fatal(err)
	}
}
