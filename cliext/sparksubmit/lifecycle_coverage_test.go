package sparksubmit

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func sparkToolZip(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range []string{"emr-serverless-spark-tool-2.0/", "emr-serverless-spark-tool-2.0/bin/", "emr-serverless-spark-tool-2.0/bin/spark-submit", "emr-serverless-spark-tool-2.0/conf/connection.properties"} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, "spark-submit") {
			_, err = f.Write([]byte("new binary"))
		} else if strings.HasSuffix(name, ".properties") {
			_, err = f.Write([]byte("# comment\nworkspaceId = original\nendpoint=custom.endpoint\ninvalid\n"))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSparkRunInstallsAndPassesConfiguration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("spark-submit requires Unix")
	}
	dir := t.TempDir()
	oldPath, oldLatest, oldLook, oldHTTP, oldLoad, oldExec := getConfigurePathFunc, getLatestVersionFunc, lookPathFunc, httpGetFunc, loadProfileFunc, execCommandFunc
	t.Cleanup(func() {
		getConfigurePathFunc = oldPath
		getLatestVersionFunc = oldLatest
		lookPathFunc = oldLook
		httpGetFunc = oldHTTP
		loadProfileFunc = oldLoad
		execCommandFunc = oldExec
	})
	t.Setenv(EnvExecPath, "")
	t.Setenv(EnvNoUpdateCheck, "")
	t.Setenv(EnvWorkspaceID, "")
	t.Setenv(EnvEndpoint, "")
	t.Setenv("SPARK_CONF_DIR", "")
	getConfigurePathFunc = func() string { return dir }
	getLatestVersionFunc = func(string) (string, error) { return "2.0", nil }
	lookPathFunc = func(string) (string, error) { return "java", nil }
	data := sparkToolZip(t)
	downloads := 0
	httpGetFunc = func(url string) (*http.Response, error) {
		downloads++
		if !strings.HasSuffix(url, "2.0-bin.zip") {
			t.Errorf("download URL=%q", url)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data))}, nil
	}
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.StsToken, AccessKeyId: "ak", AccessKeySecret: "sk", StsToken: "token", RegionId: "cn-hangzhou"}, nil
	}
	var child *exec.Cmd
	var gotArgs []string
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		child = exec.Command(os.Args[0], "-test.run=^TestSparkChildSuccess$")
		return child
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	c := NewContext(ctx)
	args := []string{"--region=cn-hangzhou", "-p", "default", "--cli-ai-mode", "--version", "--name", "job", "oss://bucket/job.jar"}
	if err := c.Run(args); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 || !c.installed || c.versionLocal != "2.0" {
		t.Fatalf("install state: downloads=%d, installed=%v, version=%q", downloads, c.installed, c.versionLocal)
	}
	if !reflect.DeepEqual(gotArgs, []string{"--version", "--name", "job", "oss://bucket/job.jar"}) {
		t.Fatalf("child arguments=%v", gotArgs)
	}
	if !strings.Contains(strings.Join(child.Env, "\n"), "SPARK_CONF_DIR="+c.confDir) {
		t.Fatalf("missing child conf dir: %v", child.Env)
	}
	p, err := readPropertiesFile(filepath.Join(c.confDir, "connection.properties"))
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"accessKeyId": "ak", "accessKeySecret": "sk", "securityToken": "token", "regionId": "cn-hangzhou", "workspaceId": "original", "endpoint": "custom.endpoint"} {
		if p[k] != want {
			t.Errorf("property %s=%q,want %q", k, p[k], want)
		}
	}
	if got := c.readLocalVersion(); got != "2.0" {
		t.Fatalf("installed version=%q", got)
	}
	if err := c.touchCache(); err != nil {
		t.Fatal(err)
	}
	if c.cacheStale() {
		t.Fatal("fresh cache stale")
	}
	if err := c.Run(nil); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 {
		t.Fatal("fresh installation downloaded again")
	}
}
func TestSparkChildSuccess(t *testing.T) {}

func TestSparkVersionUpdatePaths(t *testing.T) {
	oldLatest, oldHTTP, oldInstall, oldTime := getLatestVersionFunc, httpGetFunc, installFromZipFunc, timeNowFunc
	t.Cleanup(func() {
		getLatestVersionFunc = oldLatest
		httpGetFunc = oldHTTP
		installFromZipFunc = oldInstall
		timeNowFunc = oldTime
	})
	t.Setenv(EnvExecPath, "")
	t.Setenv(EnvNoUpdateCheck, "")
	timeNowFunc = func() time.Time { return time.Unix(100000, 0) }
	for _, tc := range []struct {
		name         string
		installed    bool
		local        string
		latestErr    bool
		installErr   bool
		wantDownload bool
		wantErr      bool
	}{
		{name: "new lookup fails", latestErr: true, wantErr: true}, {name: "existing lookup fails", installed: true, latestErr: true}, {name: "same version", installed: true, local: "2.0"}, {name: "upgrade", installed: true, local: "1.0", wantDownload: true}, {name: "invalid archive", installed: true, local: "1.0", installErr: true, wantDownload: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var stderr bytes.Buffer
			c := &Context{originCtx: cli.NewCommandContext(io.Discard, &stderr), installDir: filepath.Join(dir, "tool"), versionCachePath: filepath.Join(dir, "cache"), versionFilePath: filepath.Join(dir, "version"), installed: tc.installed}
			if err := os.WriteFile(c.versionFilePath, []byte(tc.local), 0600); err != nil {
				t.Fatal(err)
			}
			getLatestVersionFunc = func(string) (string, error) {
				if tc.latestErr {
					return "", errors.New("offline")
				}
				return "2.0", nil
			}
			downloads := 0
			httpGetFunc = func(string) (*http.Response, error) {
				downloads++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("archive"))}, nil
			}
			installFromZipFunc = func(src, dest string) error {
				if tc.installErr {
					return errors.New("broken archive")
				}
				return nil
			}
			err := c.ensureInstalledAndUpdated()
			if (err != nil) != tc.wantErr {
				t.Fatalf("update error=%v", err)
			}
			if (downloads > 0) != tc.wantDownload {
				t.Fatalf("downloads=%d", downloads)
			}
			if tc.installed && tc.latestErr && !strings.Contains(stderr.String(), "offline") {
				t.Fatalf("missing update warning: %s", stderr.String())
			}
		})
	}
	c := &Context{installed: true}
	t.Setenv(EnvNoUpdateCheck, "1")
	if err := c.ensureInstalledAndUpdated(); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvExecPath, "custom")
	if err := c.ensureInstalledAndUpdated(); err != nil {
		t.Fatal(err)
	}
	c.installed = false
	if c.cacheStale() {
		t.Fatal("uninstalled cache stale")
	}
}

func TestSparkConfigurationFailuresAndOverrides(t *testing.T) {
	oldLoad := loadProfileFunc
	t.Cleanup(func() { loadProfileFunc = oldLoad })
	t.Setenv(EnvEndpoint, "configured.endpoint")
	t.Setenv(EnvWorkspaceID, "workspace")
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	ctx.Flags().Get(config.RegionFlagName).SetAssigned(true)
	ctx.Flags().Get(config.RegionFlagName).SetValue("cn-shanghai")
	c := &Context{originCtx: ctx, confDir: t.TempDir()}
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, nil
	}
	if err := c.prepareConnectionProperties(); err != nil {
		t.Fatal(err)
	}
	p, err := readPropertiesFile(filepath.Join(c.confDir, "connection.properties"))
	if err != nil {
		t.Fatal(err)
	}
	if p["regionId"] != "cn-shanghai" || p["endpoint"] != "configured.endpoint" || p["workspaceId"] != "workspace" {
		t.Fatalf("overrides=%v", p)
	}
	for _, tc := range []struct {
		name    string
		profile config.Profile
		err     error
		want    string
	}{
		{name: "load", err: errors.New("bad config"), want: "load profile"}, {name: "empty key", profile: config.Profile{Mode: config.AK}, want: "access key is empty"}, {name: "unsupported credential", profile: config.Profile{Mode: config.BearerToken}, want: "not supported"}, {name: "empty region", profile: config.Profile{Mode: config.AK, AccessKeyId: "ak", AccessKeySecret: "sk"}, want: "region is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loadProfileFunc = func(*cli.Context) (config.Profile, error) { return tc.profile, tc.err }
			c.originCtx = cli.NewCommandContext(io.Discard, io.Discard)
			err := c.prepareConnectionProperties()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if flagStringValue(nil, "region") != "" {
		t.Fatal("nil context has region")
	}
	path := filepath.Join(t.TempDir(), "properties")
	if err := writePropertiesFile(path, map[string]string{"b": "two", "a": "one", "empty": ""}, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "a=one\nb=two\n" {
		t.Fatalf("sorted properties=%q,%v", b, err)
	}
}

func TestSparkArchiveReplacementAndErrors(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "tool.zip")
	install := filepath.Join(dir, "installed")
	if err := os.WriteFile(zipPath, sparkToolZip(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(install, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "old"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := installFromZip(zipPath, install); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(install, "old")) || !fileExists(filepath.Join(install, "bin", "spark-submit")) {
		t.Fatal("installation did not replace old version")
	}
	if err := installFromZip(filepath.Join(dir, "missing"), install); err == nil {
		t.Fatal("missing archive accepted")
	}
	if _, err := findToolRoot(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing staging directory accepted")
	}
	if _, err := findToolRoot(t.TempDir()); err == nil {
		t.Fatal("missing tool root accepted")
	}
	var bad bytes.Buffer
	zw := zip.NewWriter(&bad)
	_, _ = zw.Create("bad..name")
	_ = zw.Close()
	if err := os.WriteFile(zipPath, bad.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := unzip(zipPath, t.TempDir()); err == nil {
		t.Fatal("invalid archive path accepted")
	}
}

func TestSparkExecuteFailureAndCacheEdges(t *testing.T) {
	old := execCommandFunc
	t.Cleanup(func() { execCommandFunc = old })
	c := &Context{originCtx: cli.NewCommandContext(io.Discard, io.Discard), execFilePath: "spark-submit", versionCachePath: filepath.Join(t.TempDir(), "cache")}
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	if err := c.execute(nil); err == nil {
		t.Fatal("missing executable accepted")
	}
	e := &ExitError{Code: 7}
	if e.ExitCode() != 7 || !strings.Contains(e.Error(), "7") {
		t.Fatalf("exit error=%v", e)
	}
	c.installed = true
	for value, want := range map[string]bool{"invalid": true, fmt.Sprint(time.Now().Unix()): false, "0": true} {
		if err := os.WriteFile(c.versionCachePath, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if c.cacheStale() != want {
			t.Errorf("cache %q stale=%v", value, c.cacheStale())
		}
	}
	if c.readLocalVersion() != "" {
		t.Fatal("missing local version not empty")
	}
}
