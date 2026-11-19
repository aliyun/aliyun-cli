package releasever

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionFileIsPublishable guards the tracked version file, because the
// Makefile now derives the default VERSION from it.
func TestVersionFileIsPublishable(t *testing.T) {
	raw := readVersionFile(t)
	if _, err := Classify(raw); err != nil {
		t.Fatalf("version file %q is not a publishable version: %v", raw, err)
	}
}

// TestMakefileDefaultsToVersionFile checks that a plain build injects the
// version file value. -n keeps it parse-only: no submodule, network or build.
func TestMakefileDefaultsToVersionFile(t *testing.T) {
	wantFlag := "cli.Version=" + readVersionFile(t)
	out, _ := runMake(t, "-n", "build")
	if !strings.Contains(out, wantFlag) {
		t.Fatalf("make -n build does not inject %s\noutput:\n%s", wantFlag, out)
	}
}

// TestMakefileRejectsEmptyVersion keeps the hard failure for an empty VERSION
// instead of silently building with an empty cli.Version.
func TestMakefileRejectsEmptyVersion(t *testing.T) {
	out, err := runMake(t, "-n", "build", "VERSION=")
	if err == nil {
		t.Fatalf("make -n build VERSION= succeeded, want a non-zero exit\noutput:\n%s", out)
	}
	if !strings.Contains(out, "VERSION is empty") {
		t.Fatalf("make -n build VERSION= exit %v without a clear VERSION error\noutput:\n%s", err, out)
	}
}

func readVersionFile(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "version"))
	if err != nil {
		t.Fatalf("read version file: %v", err)
	}
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		t.Fatal("version file is empty")
	}
	return raw
}

func runMake(t *testing.T, args ...string) (string, error) {
	t.Helper()
	bin, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is not installed")
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	// Keep VERSION/MAKEFLAGS from an outer make from leaking into the probe.
	cmd.Env = withoutEnv(os.Environ(), "VERSION", "MAKEFLAGS", "MFLAGS", "MAKELEVEL")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "Makefile")); err != nil {
		t.Fatalf("repo root %s does not hold the Makefile: %v", root, err)
	}
	return root
}

func withoutEnv(env []string, names ...string) []string {
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[name] = true
	}
	kept := make([]string, 0, len(env))
	for _, kv := range env {
		if key, _, ok := strings.Cut(kv, "="); ok && drop[key] {
			continue
		}
		kept = append(kept, kv)
	}
	return kept
}
