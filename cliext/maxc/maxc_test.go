package maxc

import (
	"testing"
)

func TestPlatformWhitelist(t *testing.T) {
	cases := map[string]bool{
		"linux-amd64":   true,
		"linux-arm64":   true,
		"darwin-amd64":  true,
		"darwin-arm64":  true,
		"windows-amd64": true,
		"windows-arm64": false,
		"freebsd-amd64": false,
		"linux-386":     false,
	}
	for k, want := range cases {
		_, got := platformPaths[k]
		if got != want {
			t.Errorf("platformPaths[%q] = %v, want %v", k, got, want)
		}
	}
}

func TestInitBasicInfo_DerivesPathFromConfig(t *testing.T) {
	origGetConfigurePath := getConfigurePathFunc
	defer func() { getConfigurePathFunc = origGetConfigurePath }()
	getConfigurePathFunc = func() string { return "/tmp/test-aliyun" }

	c := &Context{}
	c.InitBasicInfo()

	if c.installDir != "/tmp/test-aliyun/maxc" {
		t.Errorf("installDir = %q, want /tmp/test-aliyun/maxc", c.installDir)
	}
	wantExec := "/tmp/test-aliyun/maxc/maxc"
	if runtimeGOOSFunc() == "windows" {
		wantExec += ".exe"
	}
	if c.execFilePath != wantExec {
		t.Errorf("execFilePath = %q, want %q", c.execFilePath, wantExec)
	}
}

func TestInitBasicInfo_ExecPathEnvOverride(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_MAXC_EXEC_PATH", "/opt/maxc/maxc")
	origGetConfigurePath := getConfigurePathFunc
	defer func() { getConfigurePathFunc = origGetConfigurePath }()
	getConfigurePathFunc = func() string { return "/tmp/test-aliyun" }

	c := &Context{}
	c.InitBasicInfo()

	if c.execFilePath != "/opt/maxc/maxc" {
		t.Errorf("env override ignored: execFilePath = %q", c.execFilePath)
	}
}

func TestCheckOsTypeAndArch_SetsSupportFlag(t *testing.T) {
	origGOOS, origGOARCH := runtimeGOOSFunc, runtimeGOARCHFunc
	defer func() {
		runtimeGOOSFunc = origGOOS
		runtimeGOARCHFunc = origGOARCH
	}()

	cases := []struct {
		os, arch string
		support  bool
	}{
		{"linux", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"windows", "arm64", false},
		{"freebsd", "amd64", false},
		{"linux", "386", false},
	}
	for _, tc := range cases {
		runtimeGOOSFunc = func() string { return tc.os }
		runtimeGOARCHFunc = func() string { return tc.arch }
		c := &Context{}
		c.CheckOsTypeAndArch()
		if c.osSupport != tc.support {
			t.Errorf("%s-%s: osSupport = %v, want %v", tc.os, tc.arch, c.osSupport, tc.support)
		}
	}
}
