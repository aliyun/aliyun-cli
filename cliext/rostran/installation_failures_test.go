package rostran

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type failedArchiveBody struct{}

func (failedArchiveBody) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (failedArchiveBody) Close() error             { return nil }

func TestRostranDownloadFailures(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	for _, tc := range []struct {
		name          string
		status        int
		body          io.ReadCloser
		requestErr    error
		missingParent bool
		want          string
	}{
		{name: "network", requestErr: errors.New("offline"), want: "offline"},
		{name: "HTTP status", status: 404, body: io.NopCloser(strings.NewReader("missing")), want: "404"},
		{name: "body", status: 200, body: failedArchiveBody{}, want: "body failed"},
		{name: "destination", status: 200, body: io.NopCloser(strings.NewReader("archive")), missingParent: true, want: "create file"},
		{name: "invalid gzip", status: 200, body: io.NopCloser(strings.NewReader("invalid")), want: "extract"},
		{name: "empty package", status: 200, body: io.NopCloser(bytes.NewReader(buildTarGz(t, map[string]string{}))), want: "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.requestErr != nil {
					return nil, tc.requestErr
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			}
			dir := t.TempDir()
			dest := filepath.Join(dir, "archive.tar.gz")
			if tc.missingParent {
				dest = filepath.Join(dir, "missing", "archive.tar.gz")
			}
			if err := os.Mkdir(filepath.Join(dir, "rostran_extract"), 0700); err != nil {
				t.Fatal(err)
			}
			err := DownloadAndExtract("https://example.invalid/archive", dest, filepath.Join(dir, "installed"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("download error=%v,want=%q", err, tc.want)
			}
		})
	}
}

func TestRostranReplacementPreservesPreviousVersion(t *testing.T) {
	oldMove, oldNow := moveDirFunc, timeNowFunc
	t.Cleanup(func() { moveDirFunc = oldMove; timeNowFunc = oldNow })
	timeNowFunc = func() time.Time { return time.Unix(100, 0) }
	for _, scenario := range []string{"replace", "activation failure", "backup collision"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source")
			installed := filepath.Join(dir, "installed")
			if err := os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "new"), []byte("new"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(installed, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(installed, "old"), []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			moveDirFunc = func(src, dst string) error {
				if scenario == "activation failure" {
					return nil
				}
				if err := moveDir(src, dst); err != nil {
					return err
				}
				if scenario == "backup collision" {
					backup := installSiblingPath(installed, "old")
					if err := os.Mkdir(backup, 0700); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(backup, "concurrent"), []byte("other"), 0600)
				}
				return nil
			}
			err := replaceInstallDir(source, installed)
			if scenario == "replace" {
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(installed, "new"))
				if err != nil || string(data) != "new" {
					t.Fatalf("new package=%q,%v", data, err)
				}
				if fileExists(filepath.Join(installed, "old")) {
					t.Fatal("old package contents survived replacement")
				}
			} else {
				if err == nil {
					t.Fatal("installation failure lost")
				}
				data, readErr := os.ReadFile(filepath.Join(installed, "old"))
				if readErr != nil || string(data) != "old" {
					t.Fatalf("previous version lost: %q,%v; installation error=%v", data, readErr, err)
				}
			}
		})
	}
}

func rostranBoundaryTar(t *testing.T, headers []tar.Header, truncated bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, h := range headers {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if truncated {
				break
			}
			if _, err := tw.Write([]byte("data")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !truncated {
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRostranTarEntryAndFilesystemFailures(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "archive.tar.gz")
	headers := []tar.Header{{Name: ".", Typeflag: tar.TypeDir}, {Name: "package/", Typeflag: tar.TypeDir}, {Name: "package/link", Typeflag: tar.TypeLink, Linkname: "file"}, {Name: "package/device", Typeflag: tar.TypeChar}, {Name: "package/file", Typeflag: tar.TypeReg, Size: 4}}
	data := rostranBoundaryTar(t, headers, false)
	if err := os.WriteFile(src, data, 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := extractTarGz(src, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "package", "file"))
	if err != nil || string(got) != "data" {
		t.Fatalf("extracted data=%q,%v", got, err)
	}
	for _, name := range []string{"link", "device"} {
		if fileExists(filepath.Join(dest, "package", name)) {
			t.Errorf("unsupported entry %q extracted", name)
		}
	}
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, blocked); err == nil {
		t.Fatal("destination file accepted")
	}
	regOnly := rostranBoundaryTar(t, []tar.Header{{Name: "file", Typeflag: tar.TypeReg, Size: 4}}, false)
	if err := os.WriteFile(src, regOnly, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dest, "file"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, dest); err == nil {
		t.Fatal("output directory overwritten with file")
	}
	broken := rostranBoundaryTar(t, []tar.Header{{Name: "file", Typeflag: tar.TypeReg, Size: 4}}, true)
	if err := os.WriteFile(src, broken, 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(src, t.TempDir()); err == nil {
		t.Fatal("truncated tar file accepted")
	}
	oversized := rostranBoundaryTar(t, []tar.Header{{Name: "oversized", Typeflag: tar.TypeReg, Size: maxExtractSize + 1}}, true)
	if err := os.WriteFile(src, oversized, 0600); err != nil {
		t.Fatal(err)
	}
	oversizedDest := t.TempDir()
	_ = extractTarGz(src, oversizedDest)
	if fileExists(filepath.Join(oversizedDest, "oversized")) {
		t.Fatal("oversized entry extracted")
	}
}

func TestRostranUpgradeHintCacheBoundaries(t *testing.T) {
	oldExec, oldLatest, oldNow := execCommandFunc, getLatestRostranVersionFunc, timeNowFunc
	t.Cleanup(func() { execCommandFunc = oldExec; getLatestRostranVersionFunc = oldLatest; timeNowFunc = oldNow })
	ctx, _, stderr := newOriginCtx()
	c := NewContext(ctx)
	c.configPath = t.TempDir()
	timeNowFunc = func() time.Time { return time.Unix(200000, 0) }
	c.maybeShowUpgradeHint()
	if stderr.Len() != 0 {
		t.Fatal("uninstalled binary printed upgrade hint")
	}
	c.installed = true
	execCommandFunc = func(string, ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRostranVersionChild$")
		cmd.Env = append(os.Environ(), "CLI_TEST_ROSTRAN_VERSION=2.0", "GOCOVERDIR="+t.TempDir())
		return cmd
	}
	requests := 0
	getLatestRostranVersionFunc = func() (string, error) { requests++; return "3.0", nil }
	if err := os.WriteFile(c.versionCachePath(), []byte("invalid JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	c.maybeShowUpgradeHint()
	cache := readCache(t, c.configPath)
	if requests != 1 || cache.InstalledVersion != "2.0" || cache.LastKnownRemote != "3.0" || !strings.Contains(stderr.String(), "3.0") {
		t.Fatalf("bootstrap cache=%+v,requests=%d,hint=%s", cache, requests, stderr.String())
	}
	cache.LastRemoteCheck = 1
	writeCache(t, c.configPath, cache)
	getLatestRostranVersionFunc = func() (string, error) { requests++; return "4.0", nil }
	c.maybeShowUpgradeHint()
	cache = readCache(t, c.configPath)
	if cache.LastKnownRemote != "4.0" || cache.LastRemoteCheck != 200000 || requests != 2 {
		t.Fatalf("refresh cache=%+v,requests=%d", cache, requests)
	}
	getLatestRostranVersionFunc = func() (string, error) { return "", errors.New("offline") }
	c.bootstrapVersionCache()
	if got := readCache(t, c.configPath); got != cache {
		t.Fatalf("network failure replaced cache=%+v", got)
	}
	execCommandFunc = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing")) }
	c.bootstrapVersionCache()
}
