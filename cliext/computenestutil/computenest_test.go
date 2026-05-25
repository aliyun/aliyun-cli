package computenestutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func prepareConfig(t *testing.T, home string, language string) {
	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := fmt.Sprintf(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou","language":"%s"}]}`, language)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func newOriginCtx() (*cli.Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := cli.NewCommandContext(out, errOut)
	return ctx, out, errOut
}

func addConfigFlag(ctx *cli.Context, name string, value string) {
	f := &cli.Flag{Name: name, AssignedMode: cli.AssignedOnce, Category: "config"}
	f.SetAssigned(true)
	f.SetValue(value)
	ctx.Flags().Add(f)
}

func TestNewContext(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	if c == nil {
		t.Fatalf("NewContext returned nil")
	}
	if c.originCtx != ctx {
		t.Errorf("originCtx mismatch")
	}
}

func TestInitBasicInfo(t *testing.T) {
	tmpDir := t.TempDir()
	old := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = old }()

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	err := c.InitBasicInfo()
	if err != nil {
		t.Fatalf("InitBasicInfo failed: %v", err)
	}

	if c.configPath != tmpDir {
		t.Errorf("configPath expected %s, got %s", tmpDir, c.configPath)
	}
	if c.installed {
		t.Errorf("should not be installed initially")
	}

	// Verify venv paths are set correctly for linux
	expectedVenv := filepath.Join(tmpDir, venvDirName)
	if c.venvPath != expectedVenv {
		t.Errorf("venvPath expected %s, got %s", expectedVenv, c.venvPath)
	}
	expectedExec := filepath.Join(expectedVenv, "bin", "computenest-cli")
	if c.execFilePath != expectedExec {
		t.Errorf("execFilePath expected %s, got %s", expectedExec, c.execFilePath)
	}
	// Verify embeddedPythonDir is set
	expectedEmbedded := filepath.Join(tmpDir, embeddedPythonDirName)
	if c.embeddedPythonDir != expectedEmbedded {
		t.Errorf("embeddedPythonDir expected %s, got %s", expectedEmbedded, c.embeddedPythonDir)
	}
}

func TestInitBasicInfoWindows(t *testing.T) {
	tmpDir := t.TempDir()
	old := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = old }()

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "windows" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	err := c.InitBasicInfo()
	if err != nil {
		t.Fatalf("InitBasicInfo failed: %v", err)
	}

	expectedExec := filepath.Join(tmpDir, venvDirName, "Scripts", "computenest-cli.exe")
	if c.execFilePath != expectedExec {
		t.Errorf("execFilePath expected %s, got %s", expectedExec, c.execFilePath)
	}
}

func TestInitBasicInfoInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	old := getConfigurePathFunc
	getConfigurePathFunc = func() string { return tmpDir }
	defer func() { getConfigurePathFunc = old }()

	oldGOOS := runtimeGOOSFunc
	runtimeGOOSFunc = func() string { return "linux" }
	defer func() { runtimeGOOSFunc = oldGOOS }()

	// Create fake entry point
	execPath := filepath.Join(tmpDir, venvDirName, "bin", "computenest-cli")
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(execPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("write exec: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	_ = c.InitBasicInfo()
	if !c.installed {
		t.Errorf("should be installed now")
	}
}

func TestIsPythonVersionSufficient(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)

	tests := []struct {
		version  string
		expected bool
	}{
		{"3.12.1", true},
		{"3.10.0", true},
		{"3.10.5", true},
		{"3.11.0", true},
		{"3.9.7", false},
		{"3.8.0", false},
		{"2.7.18", false},
		{"4.0.0", true},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		result := c.isPythonVersionSufficient(tt.version)
		if result != tt.expected {
			t.Errorf("isPythonVersionSufficient(%q) = %v, want %v", tt.version, result, tt.expected)
		}
	}
}

func TestEnsurePythonAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// This test requires python3 to be available on the system
	_, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}

	tmpDir := t.TempDir()
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.osType = runtime.GOOS
	c.osArch = runtime.GOARCH
	c.embeddedPythonDir = filepath.Join(tmpDir, "python-embedded-computenest")
	err = c.EnsurePythonAvailable()
	if err != nil {
		t.Errorf("EnsurePythonAvailable failed: %v", err)
	}
	if c.pythonPath == "" {
		t.Errorf("pythonPath should not be empty")
	}
}

func TestEnsurePythonAvailableUsesEmbedded(t *testing.T) {
	// When embedded python already exists, it should be used directly
	tmpDir := t.TempDir()

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.osType = "linux"
	c.osArch = "amd64"
	c.embeddedPythonDir = filepath.Join(tmpDir, "python-embedded-computenest")

	// Create fake embedded python binary at expected path
	embeddedPython := filepath.Join(c.embeddedPythonDir, "python", "bin", "python3")
	if err := os.MkdirAll(filepath.Dir(embeddedPython), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(embeddedPython, []byte("fake"), 0755); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := c.EnsurePythonAvailable()
	if err != nil {
		t.Fatalf("EnsurePythonAvailable failed: %v", err)
	}
	if c.pythonPath != embeddedPython {
		t.Errorf("expected pythonPath=%s, got %s", embeddedPython, c.pythonPath)
	}
}

func TestGetEmbeddedPythonPath(t *testing.T) {
	tests := []struct {
		osType   string
		expected string
	}{
		{"linux", "python/bin/python3"},
		{"darwin", "python/bin/python3"},
		{"windows", "python/python.exe"},
	}

	for _, tt := range tests {
		c := &Context{osType: tt.osType, embeddedPythonDir: "/tmp/embedded"}
		result := c.getEmbeddedPythonPath()
		expected := filepath.Join("/tmp/embedded", tt.expected)
		if result != expected {
			t.Errorf("osType=%s: expected %s, got %s", tt.osType, expected, result)
		}
	}
}

func TestGetEmbeddedPythonDownloadURL(t *testing.T) {
	tests := []struct {
		osType   string
		osArch   string
		expected string
	}{
		{"darwin", "arm64", downloadBaseURL + "/python-" + embeddedPythonVersion + "-darwin-arm64.tar.gz"},
		{"darwin", "amd64", downloadBaseURL + "/python-" + embeddedPythonVersion + "-darwin-amd64.tar.gz"},
		{"linux", "amd64", downloadBaseURL + "/python-" + embeddedPythonVersion + "-linux-amd64.tar.gz"},
		{"linux", "arm64", downloadBaseURL + "/python-" + embeddedPythonVersion + "-linux-arm64.tar.gz"},
		{"windows", "amd64", downloadBaseURL + "/python-" + embeddedPythonVersion + "-windows-amd64.zip"},
	}

	for _, tt := range tests {
		c := &Context{osType: tt.osType, osArch: tt.osArch}
		result := c.getEmbeddedPythonDownloadURL()
		if result != tt.expected {
			t.Errorf("%s/%s: expected %s, got %s", tt.osType, tt.osArch, tt.expected, result)
		}
	}
}

func TestExtractTarGz(t *testing.T) {
	// Create a simple tar.gz in memory
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a directory
	tarWriter.WriteHeader(&tar.Header{
		Name:     "python/bin/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})

	// Add a file
	content := []byte("#!/bin/sh\necho hello")
	tarWriter.WriteHeader(&tar.Header{
		Name:     "python/bin/python3",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(content)),
	})
	tarWriter.Write(content)

	tarWriter.Close()
	gzWriter.Close()

	// Extract
	destDir := t.TempDir()
	err := extractTarGz(buf.Bytes(), destDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Verify
	extractedFile := filepath.Join(destDir, "python", "bin", "python3")
	if !fileExists(extractedFile) {
		t.Errorf("expected file not found: %s", extractedFile)
	}
	data, _ := os.ReadFile(extractedFile)
	if string(data) != string(content) {
		t.Errorf("file content mismatch")
	}
}

func TestExtractZip(t *testing.T) {
	// Create a simple zip in memory
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	w, _ := zipWriter.Create("python/python.exe")
	content := []byte("fake-python")
	w.Write(content)

	zipWriter.Close()

	// Extract
	destDir := t.TempDir()
	err := extractZip(buf.Bytes(), destDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	// Verify
	extractedFile := filepath.Join(destDir, "python", "python.exe")
	if !fileExists(extractedFile) {
		t.Errorf("expected file not found: %s", extractedFile)
	}
	data, _ := os.ReadFile(extractedFile)
	if string(data) != string(content) {
		t.Errorf("file content mismatch")
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	addConfigFlag(ctx, "region", "cn-hangzhou")
	addConfigFlag(ctx, "profile", "test")

	c := NewContext(ctx)
	args := []string{"list", "--region", "cn-hangzhou", "--profile", "test", "--key", "value"}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli failed: %v", err)
	}

	for _, arg := range newArgs {
		if arg == "--region" || arg == "--profile" {
			t.Errorf("config flag should be removed: %s", arg)
		}
	}

	hasKey := false
	for _, arg := range newArgs {
		if arg == "--key" {
			hasKey = true
		}
	}
	if !hasKey {
		t.Errorf("non-config flag --key should remain")
	}
}

func TestRemoveFlagsForMainCli_UserAgentPassthrough(t *testing.T) {
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	args := []string{"deploy", "--user-agent", "AlibabaCloud-Agent-Skills/alibabacloud-ecs-code-deploy"}
	newArgs, err := c.RemoveFlagsForMainCli(args)
	if err != nil {
		t.Fatalf("RemoveFlagsForMainCli failed: %v", err)
	}

	hasUA := false
	hasUAValue := false
	for i, arg := range newArgs {
		if arg == "--user-agent" {
			hasUA = true
			if i+1 < len(newArgs) && newArgs[i+1] == "AlibabaCloud-Agent-Skills/alibabacloud-ecs-code-deploy" {
				hasUAValue = true
			}
		}
	}
	if !hasUA {
		t.Errorf("--user-agent should be passed through to computenest cli, got: %v", newArgs)
	}
	if !hasUAValue {
		t.Errorf("--user-agent value should be passed through, got: %v", newArgs)
	}
}

func TestPrepareEnv_AIModeUserAgentInjected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.Unsetenv("ALIBABA_CLOUD_USER_AGENT")
	prepareConfig(t, home, "en")

	cfgDir := filepath.Join(home, ".aliyun")
	aiJSON := `{"enabled":false,"user_agent":"AlibabaCloud-Agent-Skills/alibabacloud-ecs-code-deploy"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "ai-mode.json"), []byte(aiJSON), 0600); err != nil {
		t.Fatalf("write ai-mode: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv failed: %v", err)
	}

	want := "ALIBABA_CLOUD_USER_AGENT=AlibabaCloud-Agent-Skills/alibabacloud-ecs-code-deploy"
	found := false
	for _, e := range envs {
		if e == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expect env %q to be injected, envs: %v", want, filterUA(envs))
	}
}

func TestPrepareEnv_AIModeUserAgentNotOverrideExportedEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("ALIBABA_CLOUD_USER_AGENT", "UserExported/1")
	prepareConfig(t, home, "en")

	cfgDir := filepath.Join(home, ".aliyun")
	aiJSON := `{"enabled":true,"user_agent":"FromAiMode/2"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "ai-mode.json"), []byte(aiJSON), 0600); err != nil {
		t.Fatalf("write ai-mode: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv failed: %v", err)
	}

	for _, e := range envs {
		if e == "ALIBABA_CLOUD_USER_AGENT=FromAiMode/2" {
			t.Errorf("ai-mode UA should not override user-exported ALIBABA_CLOUD_USER_AGENT, got envs: %v", filterUA(envs))
		}
	}
}

func TestPrepareEnv_AIModeUserAgentEmptyNotInjected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.Unsetenv("ALIBABA_CLOUD_USER_AGENT")
	prepareConfig(t, home, "en")

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv failed: %v", err)
	}

	for _, e := range envs {
		if strings.HasPrefix(e, "ALIBABA_CLOUD_USER_AGENT=") {
			t.Errorf("should not inject ALIBABA_CLOUD_USER_AGENT when ai-mode.json absent, got: %s", e)
		}
	}
}

func TestPrepareEnv_RamRoleArnNoStaticAKLeak(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.Unsetenv("ALIBABA_CLOUD_USER_AGENT")

	cfgDir := filepath.Join(home, ".aliyun")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	configJSON := `{"current":"default","profiles":[{"name":"default","mode":"RamRoleArn","access_key_id":"STATIC_AK","access_key_secret":"STATIC_SK","ram_role_arn":"acs:ram::123:role/fake","ram_session_name":"test","region_id":"cn-hangzhou"}]}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	envs, err := c.PrepareEnv()
	if err != nil {
		t.Fatalf("PrepareEnv failed: %v", err)
	}

	for _, e := range envs {
		if strings.Contains(e, "STATIC_AK") {
			t.Errorf("should NOT leak static parent AK in RamRoleArn mode, got: %s", e)
		}
		if strings.Contains(e, "STATIC_SK") {
			t.Errorf("should NOT leak static parent SK in RamRoleArn mode, got: %s", e)
		}
	}
}

func filterUA(envs []string) []string {
	out := make([]string, 0)
	for _, e := range envs {
		if strings.HasPrefix(e, "ALIBABA_CLOUD_") {
			out = append(out, e)
		}
	}
	return out
}

func TestFileExists(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if fileExists(tmpFile) {
		t.Errorf("file should not exist")
	}

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if !fileExists(tmpFile) {
		t.Errorf("file should exist")
	}
}

func TestNeedCheckVersion(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		installed      bool
		cacheExists    bool
		cacheContent   string
		currentTime    int64
		expectedResult bool
	}{
		{
			name:           "not installed",
			installed:      false,
			expectedResult: false,
		},
		{
			name:           "cache file not exist",
			installed:      true,
			cacheExists:    false,
			expectedResult: true,
		},
		{
			name:           "cache file invalid",
			installed:      true,
			cacheExists:    true,
			cacheContent:   "invalid",
			expectedResult: true,
		},
		{
			name:           "need check - expired",
			installed:      true,
			cacheExists:    true,
			cacheContent:   "1000000",
			currentTime:    2000000,
			expectedResult: true,
		},
		{
			name:           "no need check - fresh",
			installed:      true,
			cacheExists:    true,
			cacheContent:   fmt.Sprintf("%d", time.Now().Unix()),
			currentTime:    time.Now().Unix(),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheFile := filepath.Join(tmpDir, ".computenest_cli_version_check_"+tt.name)

			c := &Context{
				installed:                 tt.installed,
				checkVersionCacheFilePath: cacheFile,
			}

			if tt.cacheExists {
				err := os.WriteFile(cacheFile, []byte(tt.cacheContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write cache file: %v", err)
				}
			}

			if tt.currentTime > 0 {
				oldTimeNowFunc := timeNowFunc
				timeNowFunc = func() time.Time {
					return time.Unix(tt.currentTime, 0)
				}
				defer func() { timeNowFunc = oldTimeNowFunc }()
			}

			result := c.NeedCheckVersion()
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestUpdateCheckCacheTime(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".computenest_cli_version_check")

	fixedTime := time.Unix(1234567890, 0)
	oldTimeNowFunc := timeNowFunc
	timeNowFunc = func() time.Time {
		return fixedTime
	}
	defer func() { timeNowFunc = oldTimeNowFunc }()

	c := &Context{
		checkVersionCacheFilePath: cacheFile,
	}

	err := c.UpdateCheckCacheTime()
	if err != nil {
		t.Fatalf("UpdateCheckCacheTime failed: %v", err)
	}

	content, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	expectedContent := fmt.Sprintf("%d", fixedTime.Unix())
	if string(content) != expectedContent {
		t.Errorf("Expected cache content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestPipInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	ctx, _, errOut := newOriginCtx()
	c := NewContext(ctx)
	c.venvPipPath = "/nonexistent/pip3"

	// Test install failure (non-existent pip)
	err := c.pipInstall(false)
	if err == nil {
		t.Errorf("Expected error for non-existent pip")
	}

	// Test upgrade failure is not fatal
	err = c.pipInstall(true)
	if err != nil {
		t.Errorf("Upgrade failure should not be fatal, got: %v", err)
	}
	_ = errOut
}
