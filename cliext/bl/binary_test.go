package bl

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func binaryContext(t *testing.T) (*Context, *bytes.Buffer) {
	t.Helper()
	t.Setenv(execPathEnv, "")
	t.Setenv(versionEnv, "")
	t.Setenv(noUpdateEnv, "")
	ctx, _, stderr := newTestContext()
	c := NewContext(ctx)
	root := t.TempDir()
	c.configPath = func() string { return root }
	c.goos = func() string { return "linux" }
	c.goarch = func() string { return "amd64" }
	c.loadProfile = func(*cli.Context) (config.Profile, error) { return config.Profile{}, nil }
	if err := c.InitBasicInfo(); err != nil {
		t.Fatal(err)
	}
	return c, stderr
}

func zipFixture(t *testing.T, names []string, mode os.FileMode, body []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func assetFixture(t *testing.T, version, platform string) (releaseAsset, []byte) {
	t.Helper()
	inner := "bl-" + version + "-" + platform
	if strings.HasPrefix(platform, "windows") {
		inner += ".exe"
	}
	payload := zipFixture(t, []string{inner}, 0755, []byte("standalone-"+version))
	digest := sha256.Sum256(payload)
	file := "bl-" + version + "-" + platform + ".zip"
	return releaseAsset{Version: version, File: file, Inner: inner, SHA256: hex.EncodeToString(digest[:])}, payload
}

func releaseServer(t *testing.T, c *Context, a releaseAsset, payload []byte, paths *[]string) {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if paths != nil {
			*paths = append(*paths, r.URL.Path)
		}
		switch r.URL.Path {
		case "/manifest.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "bailian-cli", "channel": "latest", "version": a.Version, "assets": map[string]releaseAsset{"linux-x64": a, "darwin-arm64": a, "darwin-x64": a, "windows-x64": a}})
		case "/v" + a.Version + "/SHA256SUMS":
			fmt.Fprintf(w, "%s  unrelated.zip\n%s  %s\n", strings.Repeat("0", 64), a.SHA256, a.File)
		case "/v" + a.Version + "/" + a.File:
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	c.releaseURL, c.httpClient = s.URL, s.Client()
	c.httpClient.Timeout = time.Second
}

func installOld(t *testing.T, c *Context, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.execFilePath, []byte("old executable"), 0755); err != nil {
		t.Fatal(err)
	}
	c.installed = true
	c.environ = func() []string { return append(os.Environ(), "BL_TEST_VERSION="+version) }
	c.command = func(name string, args ...string) *exec.Cmd {
		if name != c.execFilePath || !reflect.DeepEqual(args, []string{"--version"}) {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestBinaryVersionHelper$")
		cmd.Env = append(os.Environ(), "BL_TEST_VERSION="+version)
		return cmd
	}
}

func TestBinaryVersionHelper(t *testing.T) {
	if v := os.Getenv("BL_TEST_VERSION"); v != "" {
		if v == "oversized" {
			fmt.Println("2.6.2")
			fmt.Print(strings.Repeat("x", metadataLimit+1))
		} else {
			fmt.Println(v)
		}
		os.Exit(0)
	}
}

func TestBinaryVersionOutputBound(t *testing.T) {
	c, _ := binaryContext(t)
	installOld(t, c, "oversized")
	if _, err := c.installedVersion(); err == nil {
		t.Fatal("oversized --version output must not bypass bound through io.Copy")
	}
}

func TestBinaryReleaseInstallation(t *testing.T) {
	for _, pin := range []string{"", "2.6.2"} {
		t.Run("pin="+pin, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.6.2", "linux-x64")
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			t.Setenv(versionEnv, pin)
			if err := c.EnsureBinary(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(c.execFilePath)
			if err != nil || string(data) != "standalone-2.6.2" {
				t.Fatalf("binary = %q, %v", data, err)
			}
			info, _ := os.Stat(c.execFilePath)
			if info.Mode().Perm() != 0755 || !c.installed {
				t.Fatalf("mode/installed = %v/%v", info.Mode(), c.installed)
			}
			meta := "/manifest.json"
			if pin != "" {
				meta = "/v2.6.2/SHA256SUMS"
			}
			if !reflect.DeepEqual(paths, []string{meta, "/v2.6.2/" + a.File}) {
				t.Fatalf("requests = %v", paths)
			}
		})
	}
}

func TestBinaryManagedInternalUpdateLinks(t *testing.T) {
	for _, directoryLink := range []bool{false, true} {
		t.Run(fmt.Sprint(directoryLink), func(t *testing.T) {
			c, _ := binaryContext(t)
			realDir := filepath.Join(c.prefixPath, "versions", "2.7.1")
			if err := os.MkdirAll(realDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(realDir, "bl"), []byte("updated"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join("versions", "2.7.1"), filepath.Join(c.prefixPath, "current")); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if directoryLink {
				if err := os.Symlink("current", filepath.Join(c.prefixPath, "bin")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(c.execFilePath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join("..", "current", "bl"), c.execFilePath); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.InitBasicInfo(); err != nil {
				t.Fatal(err)
			}
			if !c.installed {
				t.Fatal("valid internal binary update link is not installed")
			}
			t.Setenv(noUpdateEnv, "1")
			if err := c.EnsureBinary(); err != nil {
				t.Fatalf("internal update link rejected: %v", err)
			}
		})
	}
}

func TestBinaryConcurrentFirstInstall(t *testing.T) {
	c, _ := binaryContext(t)
	a, payload := assetFixture(t, "2.6.2", "linux-x64")
	started, finish := make(chan struct{}), make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".zip") {
			close(started)
			<-finish
			_, _ = w.Write(payload)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "bailian-cli", "channel": "latest", "version": a.Version, "assets": map[string]releaseAsset{"linux-x64": a}})
	}))
	defer s.Close()
	c.releaseURL, c.httpClient = s.URL, s.Client()
	result := make(chan error, 1)
	go func() { result <- c.EnsureBinary() }()
	<-started
	other := NewContext(c.originCtx)
	other.prefixPath, other.execFilePath = c.prefixPath, c.execFilePath
	other.goos, other.goarch = c.goos, c.goarch
	other.releaseURL = "http://127.0.0.1:1"
	err := other.EnsureBinary()
	close(finish)
	if firstErr := <-result; firstErr != nil {
		t.Fatal(firstErr)
	}
	if err == nil || !strings.Contains(err.Error(), "install") || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("concurrent installation was not fenced: %v", err)
	}
	other.installed = true
	other.now = func() time.Time { return c.now().Add(updateTTL + time.Hour) }
	lock := filepath.Join(c.prefixPath, ".install-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(lock)
	if err := other.EnsureBinary(); err != nil {
		t.Fatalf("busy installation blocked existing binary: %v", err)
	}
}

func TestBinaryPlatforms(t *testing.T) {
	for _, tc := range []struct{ os, arch, want string }{
		{"darwin", "arm64", "darwin-arm64"}, {"darwin", "amd64", "darwin-x64"}, {"linux", "amd64", "linux-x64"}, {"windows", "amd64", "windows-x64"},
		{"linux", "arm64", ""}, {"windows", "arm64", ""}, {"freebsd", "amd64", ""},
	} {
		t.Run(tc.os+tc.arch, func(t *testing.T) {
			c, _ := binaryContext(t)
			c.goos = func() string { return tc.os }
			c.goarch = func() string { return tc.arch }
			got, err := c.platform()
			if tc.want == "" {
				if err == nil || !strings.Contains(err.Error(), "darwin-arm64, darwin-x64, linux-x64, windows-x64") {
					t.Fatalf("unsupported error = %v", err)
				}
				c.releaseURL = "http://127.0.0.1:1"
				if err := c.EnsureBinary(); err == nil || !strings.Contains(err.Error(), "unsupported") {
					t.Fatalf("must fail before download: %v", err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("platform = %s, %v", got, err)
			}
		})
	}
}

func TestBinaryInvalidPins(t *testing.T) {
	for _, pin := range []string{"../2.6.2", "2.6", "v2.6.2", "2.6.2-beta", "2.6.2?x=1", "02.6.2", " 2.6.2 ", "2.6.2/evil"} {
		t.Run(pin, func(t *testing.T) {
			c, _ := binaryContext(t)
			t.Setenv(versionEnv, pin)
			c.releaseURL = "http://127.0.0.1:1"
			if err := c.EnsureBinary(); err == nil || !strings.Contains(err.Error(), versionEnv) {
				t.Fatalf("pin error = %v", err)
			}
		})
	}
}

func TestBinaryManifestValidation(t *testing.T) {
	for _, bad := range []string{"json", "name", "channel", "version", "missing", "file", "inner", "hash", "traversal"} {
		t.Run(bad, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.6.2", "linux-x64")
			m := map[string]any{"name": "bailian-cli", "channel": "latest", "version": "2.6.2", "assets": map[string]releaseAsset{"linux-x64": a}}
			switch bad {
			case "name":
				m["name"] = "other"
			case "channel":
				m["channel"] = "beta"
			case "version":
				m["version"] = "../../escape"
			case "missing":
				m["assets"] = map[string]releaseAsset{}
			case "file":
				a.File = "other.zip"
			case "inner":
				a.Inner = "other"
			case "hash":
				a.SHA256 = "invalid"
			case "traversal":
				a.File = "../" + a.File
			}
			if bad == "file" || bad == "inner" || bad == "hash" || bad == "traversal" {
				m["assets"] = map[string]releaseAsset{"linux-x64": a}
			}
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/manifest.json" {
					t.Error("download before validation")
					_, _ = w.Write(payload)
					return
				}
				if bad == "json" {
					_, _ = io.WriteString(w, "not json")
				} else {
					_ = json.NewEncoder(w).Encode(m)
				}
			}))
			defer s.Close()
			c.releaseURL = s.URL
			if err := c.EnsureBinary(); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func TestBinaryChecksumValidation(t *testing.T) {
	for _, body := range []string{"", strings.Repeat("z", 64) + "  bl-2.6.2-linux-x64.zip", strings.Repeat("0", 64) + "  bl-2.6.2-linux-x64.zip.other", strings.Repeat("0", 64) + "  bl-2.6.2-linux-x64.zip\n" + strings.Repeat("0", 64) + "  bl-2.6.2-linux-x64.zip"} {
		t.Run(body, func(t *testing.T) {
			c, _ := binaryContext(t)
			t.Setenv(versionEnv, "2.6.2")
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
			defer s.Close()
			c.releaseURL = s.URL
			if err := c.EnsureBinary(); err == nil {
				t.Fatal("bad checksums accepted")
			}
		})
	}
}

func TestBinaryUpdatePolicy(t *testing.T) {
	for _, tc := range []struct {
		name, current, latest, pin string
		disable, fresh, update     bool
	}{
		{name: "latest upgrade", current: "2.6.2", latest: "2.7.1", update: true},
		{name: "same", current: "2.7.1", latest: "2.7.1"},
		{name: "no downgrade", current: "2.8.0", latest: "2.7.1"},
		{name: "numeric comparison", current: "2.9.0", latest: "2.10.0", update: true},
		{name: "fixed upgrade", current: "2.5.0", latest: "2.6.2", pin: "2.6.2", update: true},
		{name: "fixed no downgrade", current: "2.7.1", latest: "2.6.2", pin: "2.6.2"},
		{name: "disabled", current: "2.6.2", latest: "2.7.1", disable: true},
		{name: "fresh", current: "2.6.2", latest: "2.7.1", fresh: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := binaryContext(t)
			installOld(t, c, tc.current)
			a, payload := assetFixture(t, tc.latest, "linux-x64")
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			t.Setenv(versionEnv, tc.pin)
			if tc.disable {
				t.Setenv(noUpdateEnv, "1")
			}
			if tc.fresh {
				c.saveCheck(tc.pin)
			}
			if err := c.EnsureBinary(); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(c.execFilePath)
			want := "old executable"
			if tc.update {
				want = "standalone-" + tc.latest
			}
			if string(data) != want {
				t.Fatalf("binary=%q, want=%q", data, want)
			}
			if (tc.fresh || tc.disable) && len(paths) != 0 {
				t.Fatalf("unneeded requests %v", paths)
			}
			if tc.pin != "" && len(paths) > 0 && paths[0] != "/v"+tc.pin+"/SHA256SUMS" {
				t.Fatalf("pin checked latest %v", paths)
			}
			paths = nil
			if err := c.EnsureBinary(); err != nil {
				t.Fatal(err)
			}
			if len(paths) != 0 {
				t.Fatalf("repeated update check %v", paths)
			}
		})
	}
}

func TestBinaryUpdateFailureContinuesInstalled(t *testing.T) {
	for _, bad := range []string{"http", "checksum", "version", "download"} {
		t.Run(bad, func(t *testing.T) {
			c, stderr := binaryContext(t)
			installOld(t, c, "2.6.2")
			a, payload := assetFixture(t, "2.7.1", "linux-x64")
			if bad == "checksum" {
				payload = []byte("corrupt")
			}
			if bad == "version" {
				c.command = func(string, ...string) *exec.Cmd { return helperCommand(4) }
			}
			if bad == "http" || bad == "download" {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if bad == "download" && r.URL.Path == "/manifest.json" {
						_ = json.NewEncoder(w).Encode(map[string]any{"name": "bailian-cli", "channel": "latest", "version": a.Version, "assets": map[string]releaseAsset{"linux-x64": a}})
						return
					}
					http.Error(w, "offline", 503)
				}))
				defer s.Close()
				c.releaseURL = s.URL
			} else {
				releaseServer(t, c, a, payload, nil)
			}
			if err := c.EnsureBinary(); err != nil {
				t.Fatalf("update blocked command: %v", err)
			}
			if !strings.Contains(stderr.String(), "Warning") {
				t.Fatalf("missing warning: %s", stderr.String())
			}
			data, _ := os.ReadFile(c.execFilePath)
			if string(data) != "old executable" {
				t.Fatal("old binary damaged")
			}
			c.command = func(string, ...string) *exec.Cmd { t.Fatal("failed checks must be throttled"); return nil }
			if err := c.EnsureBinary(); err != nil {
				t.Fatal(err)
			}
			if bad != "version" {
				c.installed = false
				if err := os.Remove(c.execFilePath); err != nil {
					t.Fatal(err)
				}
				if err := c.EnsureBinary(); err == nil {
					t.Fatal("first install must return failure")
				}
			}
		})
	}
}

func TestBinaryTTLUsesTargetAndExpires(t *testing.T) {
	c, _ := binaryContext(t)
	installOld(t, c, "2.6.2")
	c.saveCheck("")
	if !c.checkFresh("") || c.checkFresh("2.6.2") {
		t.Fatal("TTL target mismatch")
	}
	now := c.now()
	c.now = func() time.Time { return now.Add(updateTTL + time.Second) }
	if c.checkFresh("") {
		t.Fatal("expired TTL")
	}
	c.now = func() time.Time { return now.Add(-time.Hour) }
	if c.checkFresh("") {
		t.Fatal("future timestamp accepted")
	}
	if err := os.WriteFile(filepath.Join(c.prefixPath, "update-check.json"), []byte("bad json"), 0600); err != nil {
		t.Fatal(err)
	}
	if c.checkFresh("") {
		t.Fatal("malformed metadata accepted")
	}
}

func TestBinaryUnsafeZIPPreservesOld(t *testing.T) {
	for _, bad := range []string{"checksum", "invalid zip", "traversal", "absolute", "nested", "backslash", "symlink", "dir", "duplicate", "unexpected", "oversize", "corrupt data"} {
		t.Run(bad, func(t *testing.T) {
			c, _ := binaryContext(t)
			installOld(t, c, "2.5.0")
			a, payload := assetFixture(t, "2.6.2", "linux-x64")
			names := []string{a.Inner}
			mode := os.FileMode(0755)
			body := []byte("new")
			switch bad {
			case "traversal":
				names = []string{"../" + a.Inner}
			case "absolute":
				names = []string{"/" + a.Inner}
			case "nested":
				names = []string{"dir/" + a.Inner}
			case "backslash":
				names = []string{"dir\\" + a.Inner}
			case "symlink":
				mode = os.ModeSymlink | 0755
			case "dir":
				mode = os.ModeDir | 0755
			case "duplicate":
				names = append(names, a.Inner)
			case "unexpected":
				names = []string{"other"}
			case "oversize":
				body = bytes.Repeat([]byte{0}, binaryLimit+1)
			}
			if bad != "checksum" {
				payload = zipFixture(t, names, mode, body)
			} else {
				payload = []byte("checksum mismatch")
			}
			if bad == "invalid zip" {
				payload = []byte("invalid")
			}
			if bad == "corrupt data" {
				central := bytes.Index(payload, []byte{'P', 'K', 1, 2})
				payload[central+16] ^= 255 // Corrupt the central directory CRC, not an unused local header.
			}
			if bad != "checksum" {
				sum := sha256.Sum256(payload)
				a.SHA256 = hex.EncodeToString(sum[:])
			}
			releaseServer(t, c, a, payload, nil)
			if err := c.installAsset(a); err == nil {
				t.Fatal("unsafe ZIP accepted")
			}
			data, _ := os.ReadFile(c.execFilePath)
			if string(data) != "old executable" {
				t.Fatal("old binary damaged")
			}
			entries, _ := os.ReadDir(filepath.Dir(c.execFilePath))
			if len(entries) != 1 {
				t.Fatalf("temporary files leaked: %v", entries)
			}
		})
	}
}

func TestBinaryFetchBoundsAndStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		limit  int64
		fail   bool
	}{
		{"ok", 200, "abcd", 4, false}, {"overflow", 200, "abcde", 4, true}, {"status", 404, "missing", 100, true}, {"redirect failure", 302, "", 100, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := binaryContext(t)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer s.Close()
			c.releaseURL = s.URL
			_, err := c.fetch("/test", tc.limit)
			if (err != nil) != tc.fail {
				t.Fatalf("fetch error=%v", err)
			}
		})
	}
	c, _ := binaryContext(t)
	c.releaseURL = "http://127.0.0.1:1"
	c.httpClient.Timeout = 100 * time.Millisecond
	if _, err := c.fetch("/missing", metadataLimit); err == nil {
		t.Fatal("connection failure hidden")
	}
}

func TestBinaryAtomicRenameFailures(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		windows, failBackup, failInstall bool
	}{
		{name: "POSIX rename failure"}, {name: "Windows replace", windows: true}, {name: "Windows backup failure", windows: true, failBackup: true}, {name: "Windows rollback", windows: true, failInstall: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := binaryContext(t)
			installOld(t, c, "2.5.0")
			if tc.windows {
				c.goos = func() string { return "windows" }
			}
			a, payload := assetFixture(t, "2.6.2", "linux-x64")
			releaseServer(t, c, a, payload, nil)
			calls := 0
			c.rename = func(old, new string) error {
				calls++
				if calls == 1 || tc.failBackup && calls == 2 || tc.failInstall && calls == 3 {
					return fmt.Errorf("rename failure")
				}
				return os.Rename(old, new)
			}
			err := c.installAsset(a)
			success := tc.windows && !tc.failBackup && !tc.failInstall
			if (err == nil) != success {
				t.Fatalf("rename result=%v", err)
			}
			data, _ := os.ReadFile(c.execFilePath)
			want := "old executable"
			if success {
				want = "standalone-2.6.2"
			}
			if string(data) != want {
				t.Fatalf("binary=%q", data)
			}
			files, _ := os.ReadDir(filepath.Dir(c.execFilePath))
			if len(files) != 1 {
				t.Fatalf("temporary files leaked: %v", files)
			}
		})
	}
}

func TestBinaryActualVersionAndErrors(t *testing.T) {
	for _, tc := range []struct{ output, want string }{
		{"2.6.2", "2.6.2"}, {"bailian-cli v2.7.1", "2.7.1"}, {"invalid", ""}, {"2.6", ""}, {"2.6.2-beta", ""},
	} {
		t.Run(tc.output, func(t *testing.T) {
			c, _ := binaryContext(t)
			installOld(t, c, tc.output)
			v, err := c.installedVersion()
			if tc.want == "" {
				if err == nil {
					t.Fatal("malformed installed version accepted")
				}
			} else if err != nil || v != tc.want {
				t.Fatalf("version = %q, %v", v, err)
			}
		})
	}
	c, _ := binaryContext(t)
	if _, err := c.installedVersion(); err == nil {
		t.Fatal("missing executable accepted")
	}
}

func TestBinaryChildUpdateNotDowngradedAfterTTL(t *testing.T) {
	c, _ := binaryContext(t)
	installOld(t, c, "2.6.2")
	c.saveCheck("")
	installOld(t, c, "2.9.0") // The child replaced itself; host metadata has no authoritative version.
	now := c.now()
	c.now = func() time.Time { return now.Add(updateTTL + time.Second) }
	a, payload := assetFixture(t, "2.7.1", "linux-x64")
	var paths []string
	releaseServer(t, c, a, payload, &paths)
	if err := c.EnsureBinary(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"/manifest.json"}) {
		t.Fatalf("downgrade download: %v", paths)
	}
}

func TestBinaryFixedPlatformAssets(t *testing.T) {
	for _, platform := range []string{"darwin-arm64", "darwin-x64", "linux-x64", "windows-x64"} {
		t.Run(platform, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.6.2", platform)
			releaseServer(t, c, a, payload, nil)
			for _, pin := range []string{"2.6.2", ""} {
				got, err := c.resolveAsset(platform, pin)
				if err != nil || got != a {
					t.Fatalf("asset = %#v, %v", got, err)
				}
			}
		})
	}
	c, _ := binaryContext(t)
	c.releaseURL = "http://127.0.0.1:1"
	if _, err := c.resolveAsset("linux-x64", "2.6.2"); err == nil {
		t.Fatal("missing fixed checksum ignored")
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read failure") }

func TestBinaryLocalAndReadFailures(t *testing.T) {
	if _, err := readBounded(failedReader{}, metadataLimit); err == nil {
		t.Fatal("reader error ignored")
	}
	c, _ := binaryContext(t)
	c.saveCheck("") // Missing prefix: best-effort timestamp must not create anything.
	if c.checkFresh("") {
		t.Fatal("missing cache accepted")
	}
	if err := os.WriteFile(c.prefixPath, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.checkManagedPaths(); err == nil {
		t.Fatal("file prefix accepted")
	}
	ctx, _, _ := newTestContext()
	defaultContext := NewContext(ctx)
	if defaultContext.goarch() == "" {
		t.Fatal("missing runtime architecture")
	}
}

func TestBinaryTimeoutAndProductionMetadataBounds(t *testing.T) {
	c, _ := binaryContext(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	c.releaseURL = s.URL
	c.httpClient.Timeout = 20 * time.Millisecond
	if _, err := c.fetch("/timeout", metadataLimit); err == nil {
		t.Fatal("HTTP timeout ignored")
	}
	s.Close()
	for _, pin := range []string{"", "2.6.2"} {
		t.Run("pin="+pin, func(t *testing.T) {
			c, _ := binaryContext(t)
			t.Setenv(versionEnv, pin)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, strings.Repeat("x", metadataLimit+1))
			}))
			defer s.Close()
			c.releaseURL = s.URL
			if err := c.EnsureBinary(); err == nil || !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("oversized metadata = %v", err)
			}
		})
	}
}

func TestBinaryDisabledCheckMustBeExactOne(t *testing.T) {
	for _, value := range []string{"TRUE", "true", "0", "yes"} {
		t.Run(value, func(t *testing.T) {
			c, _ := binaryContext(t)
			installOld(t, c, "2.6.2")
			t.Setenv(noUpdateEnv, value)
			a, payload := assetFixture(t, "2.7.1", "linux-x64")
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			if err := c.EnsureBinary(); err != nil {
				t.Fatal(err)
			}
			if len(paths) != 2 {
				t.Fatalf("non-1 value disabled check: %v", paths)
			}
		})
	}
}

func TestBinaryReplacementFailureKeepsRollbackBackup(t *testing.T) {
	c, _ := binaryContext(t)
	installOld(t, c, "2.5.0")
	c.goos = func() string { return "windows" }
	a, payload := assetFixture(t, "2.6.2", "windows-x64")
	releaseServer(t, c, a, payload, nil)
	calls := 0
	c.rename = func(old, new string) error {
		calls++
		if calls == 2 {
			return os.Rename(old, new)
		}
		return fmt.Errorf("replacement/rollback unavailable")
	}
	if err := c.installAsset(a); err == nil || !strings.Contains(err.Error(), "previous binary retained at") {
		t.Fatalf("rollback failure = %v", err)
	}
	files, _ := os.ReadDir(filepath.Dir(c.execFilePath))
	if len(files) != 1 || !strings.HasPrefix(files[0].Name(), ".bl-backup-") {
		t.Fatalf("previous executable lost: %v", files)
	}
	old, err := os.ReadFile(filepath.Join(filepath.Dir(c.execFilePath), files[0].Name()))
	if err != nil || string(old) != "old executable" {
		t.Fatalf("backup = %q, %v", old, err)
	}
}

func TestBinaryRunReturnsPreparationAndInstallationErrors(t *testing.T) {
	c, _ := binaryContext(t)
	if err := c.Run([]string{"--profile"}); err == nil {
		t.Fatal("missing flag accepted")
	}
	c.releaseURL = "http://127.0.0.1:1"
	if err := c.Run([]string{"help"}); err == nil {
		t.Fatal("first install failure not returned")
	}
	t.Setenv(execPathEnv, filepath.Join(t.TempDir(), "missing"))
	if err := c.Run([]string{"help"}); err == nil {
		t.Fatal("bad override accepted")
	}
}

func TestBinaryManagedSymlinksRejected(t *testing.T) {
	for _, component := range []string{"prefix", "bin", "executable"} {
		t.Run(component, func(t *testing.T) {
			c, _ := binaryContext(t)
			outside := t.TempDir()
			target := c.prefixPath
			if component != "prefix" {
				if err := os.MkdirAll(c.prefixPath, 0755); err != nil {
					t.Fatal(err)
				}
				target = filepath.Dir(c.execFilePath)
			}
			if component == "executable" {
				if err := os.MkdirAll(target, 0755); err != nil {
					t.Fatal(err)
				}
				target = c.execFilePath
				outside = filepath.Join(outside, "real")
				if err := os.WriteFile(outside, []byte("external"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(outside, target); err != nil {
				t.Skip(err)
			}
			if err := c.EnsureBinary(); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("unsafe managed path=%v", err)
			}
		})
	}
}
