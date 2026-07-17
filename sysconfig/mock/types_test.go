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

func TestExpandToSizePadsAndTruncates(t *testing.T) {
	if got := ExpandToSize("ab", 5); got != "abxxx" {
		t.Fatalf("ExpandToSize pad = %q, want abxxx", got)
	}
	if got := ExpandToSize("abcdef", 3); got != "abc" {
		t.Fatalf("ExpandToSize truncate = %q, want abc", got)
	}
	if got := ExpandToSize("keep", 0); got != "keep" {
		t.Fatalf("ExpandToSize zero = %q, want keep", got)
	}
	if got := ExpandToSize("", 4); got != "xxxx" {
		t.Fatalf("ExpandToSize empty = %q, want xxxx", got)
	}
}

func TestResolveStdoutUsesResponseBodySize(t *testing.T) {
	if got := ResolveStdout(Record{Stdout: "prefix", ResponseBodySize: 8}); got != "prefixxx" {
		t.Fatalf("ResolveStdout sized = %q, want prefixxx", got)
	}
	if got := ResolveStdout(Record{Stdout: "as-is"}); got != "as-is" {
		t.Fatalf("ResolveStdout default = %q, want as-is", got)
	}
}
