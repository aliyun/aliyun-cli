package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAlibabaCloudEnvironmentNamesExcludeValues(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_REGION_ID", "cn-secret")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "must-not-be-collected")
	t.Setenv("OTHER_SECRET", "must-not-be-collected")

	names := alibabaCloudEnvironmentNames()
	for _, want := range []string{
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET",
		"ALIBABA_CLOUD_REGION_ID",
	} {
		if !stringSliceContains(names, want) {
			t.Fatalf("missing environment variable name %q in %v", want, names)
		}
	}
	for _, forbidden := range []string{"cn-secret", "must-not-be-collected", "OTHER_SECRET"} {
		if stringSliceContains(names, forbidden) {
			t.Fatalf("environment value or unrelated name leaked: %q in %v", forbidden, names)
		}
	}
}

func TestDetectShellTypeUsesKnownEnum(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("COMSPEC", "")
	if got := detectShellType(); got != "zsh" {
		t.Fatalf("detectShellType() = %q, want zsh", got)
	}

	t.Setenv("SHELL", "/private/customer-shell")
	if got := detectShellType(); got != "other" {
		t.Fatalf("unknown shell must not be uploaded verbatim: %q", got)
	}
}

func TestDetectLocaleUsesPriorityAndSafeFormat(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "zh_CN.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")
	if got := detectLocale(); got != "zh_CN.UTF-8" {
		t.Fatalf("detectLocale() = %q, want zh_CN.UTF-8", got)
	}

	t.Setenv("LC_ALL", "unsafe/value")
	t.Setenv("LC_MESSAGES", "")
	if got := detectLocale(); got != "en_US.UTF-8" {
		t.Fatalf("unsafe locale must be skipped, got %q", got)
	}
}

func TestDetectStdinTTY(t *testing.T) {
	original := stdinIsTerminal
	stdinIsTerminal = func(fd int) bool {
		return fd == int(os.Stdin.Fd())
	}
	t.Cleanup(func() {
		stdinIsTerminal = original
	})
	if !detectStdinTTY() {
		t.Fatal("expected stdin to be detected as TTY")
	}
}

func TestReadLinuxDistroUsesSafeOSReleaseFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	content := "NAME=\"Alibaba Cloud Linux\"\n" +
		"ID=alinux\n" +
		"VERSION_ID=\"3.2104\"\n" +
		"PRETTY_NAME=\"must-not-be-collected\"\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	name, id, version := readLinuxDistro(path)
	if name != "Alibaba Cloud Linux" || id != "alinux" || version != "3.2104" {
		t.Fatalf("unexpected distro fields: name=%q id=%q version=%q", name, id, version)
	}
}

func TestReadLinuxDistroDropsUnsafeValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(path, []byte("NAME=\"unsafe:value\"\nID=linux\nVERSION_ID=1\n"), 0600); err != nil {
		t.Fatal(err)
	}

	name, id, version := readLinuxDistro(path)
	if name != "" || id != "linux" || version != "1" {
		t.Fatalf("unexpected sanitized distro fields: name=%q id=%q version=%q", name, id, version)
	}
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
