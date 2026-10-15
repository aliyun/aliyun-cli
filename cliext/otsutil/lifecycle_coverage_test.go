package otsutil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
)

type versionTransport func(*http.Request) (*http.Response, error)

func (f versionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTablestoreInstallAndUpdate(t *testing.T) {
	oldLatest, oldDownload, oldExec, oldPath, oldOS, oldArch := getLatestOtsUtilVersionFunc, downloadAndUnzipFunc, execCommandFunc, getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc
	t.Cleanup(func() {
		getLatestOtsUtilVersionFunc = oldLatest
		downloadAndUnzipFunc = oldDownload
		execCommandFunc = oldExec
		getConfigurePathFunc = oldPath
		runtimeGOOSFunc = oldOS
		runtimeGOARCHFunc = oldArch
	})
	t.Setenv("TMPDIR", t.TempDir())
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	for _, tc := range []struct {
		name                  string
		installed             bool
		version               string
		lookupErr, installErr bool
		wantInstalls          int
		wantErr               string
	}{
		{name: "install", wantInstalls: 1}, {name: "current", installed: true, version: "2.0"}, {name: "upgrade", installed: true, version: "1.0", wantInstalls: 1}, {name: "version failure", lookupErr: true, wantErr: "offline"}, {name: "download failure", installErr: true, wantInstalls: 1, wantErr: "download failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			prepareConfig(t, home, "en")
			dir := filepath.Join(home, ".aliyun")
			getConfigurePathFunc = func() string { return dir }
			ctx, _, _ := newOriginCtx()
			c := NewContext(ctx)
			if err := c.InitializeAndValidatePlatform(); err != nil {
				t.Fatal(err)
			}
			if tc.installed {
				if err := os.WriteFile(c.execFilePath, []byte("binary"), 0700); err != nil {
					t.Fatal(err)
				}
				c.installed = true
				if err := os.WriteFile(c.versionFilePath, []byte(tc.version), 0600); err != nil {
					t.Fatal(err)
				}
			}
			getLatestOtsUtilVersionFunc = func() (string, error) {
				if tc.lookupErr {
					return "", errors.New("offline")
				}
				return "2.0", nil
			}
			installs := 0
			downloadAndUnzipFunc = func(url, dest, exe, center string) error {
				installs++
				if !strings.Contains(url, "linux-amd64-2.0.zip") {
					t.Errorf("download URL=%s", url)
				}
				if tc.installErr {
					return errors.New("download failed")
				}
				return os.WriteFile(exe, []byte("binary"), 0700)
			}
			var child *exec.Cmd
			execCommandFunc = func(string, ...string) *exec.Cmd {
				child = exec.Command(os.Args[0], "-test.run=^TestTablestoreChildSuccess$")
				return child
			}
			err := c.EnsureInstalledAndUpdated()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("install error=%v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if err := c.Run([]string{"list"}); err != nil {
					t.Fatal(err)
				}
				if child.Dir != dir {
					t.Fatalf("child directory=%q", child.Dir)
				}
				if c.NeedCheckVersion() {
					t.Fatal("fresh cache stale")
				}
				if err := c.GetLocalVersion(); err != nil || c.versionLocal != "2.0" {
					t.Fatalf("installed version=%q,%v", c.versionLocal, err)
				}
			}
			if installs != tc.wantInstalls {
				t.Fatalf("install count=%d,want=%d", installs, tc.wantInstalls)
			}
		})
	}
	runtimeGOOSFunc = func() string { return "unsupported" }
	ctx, _, _ := newOriginCtx()
	if err := NewContext(ctx).InitializeAndValidatePlatform(); err == nil {
		t.Fatal("unsupported platform accepted")
	}
}
func TestTablestoreChildSuccess(t *testing.T) {}

func TestTablestoreConfigPreservesEndpointAndInstance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	prepareConfig(t, home, "en")
	dir := filepath.Join(home, ".aliyun")
	ctx, _, _ := newOriginCtx()
	config.AddFlags(ctx.Flags())
	for name, value := range map[string]string{"mode": "StsToken", "access-key-id": "new-id", "access-key-secret": "new-secret", "sts-token": "new-token"} {
		f := ctx.Flags().Get(name)
		f.SetAssigned(true)
		f.SetValue(value)
	}
	c := NewContext(ctx)
	c.configPath = dir
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"current":"default","profiles":[{"name":"default","mode":"StsToken","access_key_id":"new-id","access_key_secret":"new-secret","sts_token":"new-token","region_id":"cn-hangzhou","language":"en"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dir, ".tablestore_config")
	if err := os.WriteFile(filename, []byte(`{"Endpoint":"custom.endpoint","Instance":"instance","AccessKeyId":"old-id"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.PrepareEnv(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var got tablestoreConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Endpoint != "custom.endpoint" || got.Instance != "instance" || got.AccessKeyId != "new-id" || got.AccessKeySecret != "new-secret" || got.AccessKeySecretToken != "new-token" {
		t.Fatalf("merged config=%+v", got)
	}
	if err := os.WriteFile(filename, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.PrepareEnv(); err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("bad existing config=%v", err)
	}
	c.installed = true
	c.versionFilePath = filepath.Join(dir, "empty-version")
	if err := os.WriteFile(c.versionFilePath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.GetLocalVersion(); err == nil {
		t.Fatal("empty version accepted")
	}
}

func TestTablestoreVersionFetchAndExecutionFailures(t *testing.T) {
	oldTransport, oldExec := http.DefaultTransport, execCommandFunc
	t.Cleanup(func() { http.DefaultTransport = oldTransport; execCommandFunc = oldExec })
	for _, tc := range []struct {
		status  int
		body    string
		wantErr bool
	}{{200, " 2.0 \n", false}, {200, " ", true}, {503, "unavailable", true}} {
		http.DefaultTransport = versionTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})
		version, err := GetLatestOtsUtilVersion()
		if (err != nil) != tc.wantErr || (!tc.wantErr && version != "2.0") {
			t.Fatalf("version=%q,%v", version, err)
		}
	}
	http.DefaultTransport = versionTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	if _, err := GetLatestOtsUtilVersion(); err == nil {
		t.Fatal("network error lost")
	}
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.configPath = t.TempDir()
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	if err := c.ExecuteTablestoreCli(nil); err == nil {
		t.Fatal("subprocess error lost")
	}
	c.versionFilePath = filepath.Join(c.configPath, "missing", "version")
	if err := c.SaveLocalVersion(); err == nil {
		t.Fatal("unwritable version file accepted")
	}
	c.checkVersionCacheFilePath = c.versionFilePath
	if err := c.UpdateCheckCacheTime(); err == nil {
		t.Fatal("unwritable cache accepted")
	}
}
