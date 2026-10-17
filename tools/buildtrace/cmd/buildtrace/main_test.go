package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExitCodes(t *testing.T) {
	if code := run([]string{"buildtrace"}); code != 1 {
		t.Fatalf("missing flags exit = %d", code)
	}
	if code := run([]string{"buildtrace", "retarget"}); code != 1 {
		t.Fatalf("missing retarget flags exit = %d", code)
	}
	dir := t.TempDir()
	asset := filepath.Join(dir, "a.tgz")
	if err := os.WriteFile(asset, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	code := run([]string{
		"buildtrace",
		"--binary", bin,
		"--asset", asset,
		"--version", "1.2.3",
		"--tag", "v1.2.3",
		"--commit", strings.Repeat("a", 40),
		"--gitlink", strings.Repeat("b", 40),
		"--run-url", "https://github.com/aliyun/aliyun-cli/actions/runs/2",
	})
	if code != 0 {
		t.Fatal("expected write to succeed")
	}
	latest := filepath.Join(dir, "latest.tgz")
	if err := os.WriteFile(latest, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latest+".cdx.json", mustRead(t, asset+".cdx.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	code = run([]string{
		"buildtrace", "retarget",
		"--from", asset + ".provenance.json",
		"--asset", latest,
		"--sbom", latest + ".cdx.json",
	})
	if code != 0 {
		t.Fatal("expected retarget to succeed")
	}
}

func TestCLIRejectsBadFlags(t *testing.T) {
	if err := write([]string{"--missing-flag"}); err == nil {
		t.Fatal("expected unknown flag to fail")
	}
	if err := retarget(nil); err == nil {
		t.Fatal("expected missing retarget flags to fail")
	}
	if err := retarget([]string{"--nope"}); err == nil {
		t.Fatal("expected unknown retarget flag to fail")
	}
}

func TestCLIWritesAndRetargets(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "aliyun-cli-linux-1.2.3-amd64.tgz")
	if err := os.WriteFile(asset, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	err = write([]string{
		"--binary", bin,
		"--asset", asset,
		"--version", "1.2.3",
		"--tag", "v1.2.3",
		"--commit", strings.Repeat("c", 40),
		"--gitlink", strings.Repeat("d", 40),
		"--run-url", "https://github.com/aliyun/aliyun-cli/actions/runs/9",
	})
	if err != nil {
		t.Fatal(err)
	}
	latest := filepath.Join(dir, "aliyun-cli-linux-latest-amd64.tgz")
	if err := os.WriteFile(latest, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latest+".cdx.json", mustRead(t, asset+".cdx.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = retarget([]string{
		"--from", asset + ".provenance.json",
		"--asset", latest,
		"--sbom", latest + ".cdx.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(latest + ".provenance.json"); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
