package mock

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestResolvePathUsesEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.json")
	t.Setenv(EnvMockPath, path)

	got := ResolvePath(func() string {
		t.Fatal("default config dir should not be used when override is set")
		return ""
	})

	if got != path {
		t.Fatalf("ResolvePath() = %q, want %q", got, path)
	}
}

func TestResolvePathUsesDefaultConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvMockPath, "")

	got := ResolvePath(func() string { return dir })
	want := filepath.Join(dir, MockFileName)
	if got != want {
		t.Fatalf("ResolvePath() = %q, want %q", got, want)
	}
}

func TestRecordUnmarshalRejectsInvalidJSON(t *testing.T) {
	var record Record
	if err := json.Unmarshal([]byte("{bad"), &record); err == nil {
		t.Fatal("Unmarshal invalid JSON returned nil error")
	}
}
