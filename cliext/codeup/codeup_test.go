package codeup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOsTypeAndArch(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		goarch     string
		wantSupp   bool
		wantSuffix string
	}{
		{"darwin-arm64", "darwin", "arm64", true, "codeup-cli-macOS-arm64.zip"},
		{"darwin-amd64", "darwin", "amd64", true, "codeup-cli-macOS-64.zip"},
		{"linux-amd64", "linux", "amd64", true, "codeup-cli-Linux-64.zip"},
		{"linux-arm64", "linux", "arm64", true, "codeup-cli-Linux-64-arm64.zip"},
		{"linux-386", "linux", "386", true, "codeup-cli-Linux-32.zip"},
		{"windows-amd64", "windows", "amd64", true, "codeup-cli-Windows-64.zip"},
		{"windows-386", "windows", "386", true, "codeup-cli-Windows-32.zip"},
		{"unsupported-os", "freebsd", "amd64", false, ""},
		{"unsupported-arch", "linux", "mips", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origGOOS := runtimeGOOSFunc
			origGOARCH := runtimeGOARCHFunc
			defer func() {
				runtimeGOOSFunc = origGOOS
				runtimeGOARCHFunc = origGOARCH
			}()
			runtimeGOOSFunc = func() string { return tt.goos }
			runtimeGOARCHFunc = func() string { return tt.goarch }

			c := &Context{}
			c.CheckOsTypeAndArch()

			if c.osSupport != tt.wantSupp {
				t.Errorf("osSupport = %v, want %v", c.osSupport, tt.wantSupp)
			}
			if c.downloadPathSuffix != tt.wantSuffix {
				t.Errorf("downloadPathSuffix = %q, want %q", c.downloadPathSuffix, tt.wantSuffix)
			}
		})
	}
}

func TestInitBasicInfo(t *testing.T) {
	tmpDir := t.TempDir()
	origGetPath := getConfigurePathFunc
	origGOOS := runtimeGOOSFunc
	defer func() {
		getConfigurePathFunc = origGetPath
		runtimeGOOSFunc = origGOOS
	}()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "linux" }

	c := &Context{}
	c.InitBasicInfo()

	expectedPath := filepath.Join(tmpDir, "codeup-cli")
	if c.execFilePath != expectedPath {
		t.Errorf("execFilePath = %q, want %q", c.execFilePath, expectedPath)
	}
	if c.installed {
		t.Errorf("installed should be false when binary doesn't exist")
	}

	// create a fake binary
	_ = os.WriteFile(expectedPath, []byte("fake"), 0755)
	c.InitBasicInfo()
	if !c.installed {
		t.Errorf("installed should be true when binary exists")
	}
}

func TestInitBasicInfoWindows(t *testing.T) {
	tmpDir := t.TempDir()
	origGetPath := getConfigurePathFunc
	origGOOS := runtimeGOOSFunc
	defer func() {
		getConfigurePathFunc = origGetPath
		runtimeGOOSFunc = origGOOS
	}()
	getConfigurePathFunc = func() string { return tmpDir }
	runtimeGOOSFunc = func() string { return "windows" }

	c := &Context{}
	c.InitBasicInfo()

	expectedPath := filepath.Join(tmpDir, "codeup-cli.exe")
	if c.execFilePath != expectedPath {
		t.Errorf("execFilePath = %q, want %q", c.execFilePath, expectedPath)
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	c := &Context{}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			"no flags to strip",
			[]string{"import", "--run", "true"},
			[]string{"import", "--run", "true"},
		},
		{
			"strip profile flag",
			[]string{"import", "--profile", "default", "--run", "true"},
			[]string{"import", "--run", "true"},
		},
		{
			"strip multiple flags",
			[]string{"--profile", "test", "status", "--quiet"},
			[]string{"status"},
		},
		{
			"empty args",
			[]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.RemoveFlagsForMainCli(tt.args)
			if err != nil {
				t.Fatalf("RemoveFlagsForMainCli error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildSubprocessEnv(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=stale",
		"HOME=/home/u",
	}
	overrides := map[string]string{
		"ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE": "aliyun codeup-cli",
	}

	got := buildSubprocessEnv(base, overrides)

	seenIntegration := 0
	seenPath := false
	seenHome := false
	var integrationValue string
	for _, item := range got {
		switch {
		case item == "PATH=/usr/bin":
			seenPath = true
		case item == "HOME=/home/u":
			seenHome = true
		case len(item) > len("ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=") &&
			item[:len("ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=")] == "ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE=":
			seenIntegration++
			integrationValue = item[len("ALIBABA_CLOUD_CODEUP_INTEGRATION_MODE="):]
		}
	}
	if !seenPath || !seenHome {
		t.Errorf("expected PATH and HOME to be preserved, got %v", got)
	}
	if seenIntegration != 1 {
		t.Errorf("expected exactly one integration mode entry, got %d", seenIntegration)
	}
	if integrationValue != "aliyun codeup-cli" {
		t.Errorf("integration mode = %q, want %q", integrationValue, "aliyun codeup-cli")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	existingFile := filepath.Join(tmpDir, "exists")
	_ = os.WriteFile(existingFile, []byte("data"), 0644)

	if !fileExists(existingFile) {
		t.Errorf("fileExists should return true for existing file")
	}
	if fileExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Errorf("fileExists should return false for non-existing file")
	}
}
