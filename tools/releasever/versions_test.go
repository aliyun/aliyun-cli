package releasever

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendStableVersion(t *testing.T) {
	got, err := AppendStableVersion(nil, "v3.5.2")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "3.5.2\n" {
		t.Fatalf("first record = %q", got)
	}

	got, err = AppendStableVersion(got, "3.5.3")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "3.5.2\n3.5.3\n" {
		t.Fatalf("append = %q", got)
	}

	again, err := AppendStableVersion(got, "v3.5.2")
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Fatalf("duplicate changed history: %q", again)
	}

	withNote := []byte("# stable history\n3.5.2\n")
	got, err = AppendStableVersion(withNote, "3.5.4")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# stable history\n3.5.2\n3.5.4\n" {
		t.Fatalf("preserved lines = %q", got)
	}

	if _, err := AppendStableVersion(nil, "3.5.2-beta.1"); err == nil || !strings.Contains(err.Error(), "prerelease") {
		t.Fatalf("prerelease err = %v", err)
	}
	if _, err := AppendStableVersion(nil, "latest"); err == nil {
		t.Fatal("invalid version was recorded")
	}

	withoutNewline, err := AppendStableVersion([]byte("3.5.1"), "3.5.1")
	if err != nil {
		t.Fatal(err)
	}
	if string(withoutNewline) != "3.5.1\n" {
		t.Fatalf("missing newline = %q", withoutNewline)
	}
	withBeta, err := AppendStableVersion([]byte("3.5.2-beta.1\n"), "3.5.2")
	if err != nil {
		t.Fatal(err)
	}
	if string(withBeta) != "3.5.2-beta.1\n3.5.2\n" {
		t.Fatalf("prerelease line = %q", withBeta)
	}
}

func TestAppendVersionCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions")
	if err := os.WriteFile(path, []byte("3.5.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLI(t, "append-version", "--", "3.5.2", path)
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if stdout != "3.5.1\n3.5.2\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if _, stderr, code := runCLI(t, "append-version", "3.5.2"); code == 0 || !strings.Contains(stderr, "usage:") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}
