package dgate

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/aliyun/aliyun-cli/v3/cli/assetmanifest"
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
)

type signedDgateFixture struct {
	manifest                assetmanifest.Manifest
	data, signature         []byte
	private                 ed25519.PrivateKey
	status, signatureStatus int
	requests                int
}

func (f *signedDgateFixture) sign(t *testing.T) {
	t.Helper()
	var err error
	f.data, err = assetmanifest.Encode(f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	f.signature, err = publishsig.Sign(f.private, assetmanifest.FileName, bytes.NewReader(f.data))
	if err != nil {
		t.Fatal(err)
	}
}

func signedDgateManifest(t *testing.T, c *Context, version string, payload []byte) *signedDgateFixture {
	t.Helper()
	pub, private, err := publishsig.Generate()
	if err != nil {
		t.Fatal(err)
	}
	now := timeNowFunc()
	sum := sha256.Sum256(payload)
	f := &signedDgateFixture{private: private, status: http.StatusOK, signatureStatus: http.StatusOK,
		manifest: assetmanifest.Manifest{Format: assetmanifest.Format, Sequence: 10,
			IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour),
			Assets: []assetmanifest.Entry{{Product: "dgate", Purpose: assetmanifest.Purpose,
				Version: version, Platform: c.osType + "-" + c.osArch, Asset: c.artifactName(),
				URL: c.baseURL() + "/" + version + "/" + c.artifactName(), SHA256: hex.EncodeToString(sum[:]),
				Size: int64(len(payload)), MinVersion: "0.0.0"}}}}
	f.sign(t)
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests++
		switch r.URL.Path {
		case "/" + assetmanifest.FileName:
			w.WriteHeader(f.status)
			_, _ = w.Write(f.data)
		case "/" + assetmanifest.FileName + ".minisig":
			w.WriteHeader(f.signatureStatus)
			_, _ = w.Write(f.signature)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	c.assets = &assetmanifest.Client{SourceURL: s.URL + "/" + assetmanifest.FileName,
		HTTPClient: s.Client(), Keys: []ed25519.PublicKey{pub}, Now: func() time.Time { return now }}
	return f
}

func signedDgateContext(t *testing.T) (*Context, *bytes.Buffer) {
	t.Helper()
	t.Setenv(execPathEnv, "")
	root := t.TempDir()
	prevPath, prevOS, prevArch := getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc
	getConfigurePathFunc = func() string { return root }
	runtimeGOOSFunc = func() string { return "linux" }
	runtimeGOARCHFunc = func() string { return "amd64" }
	t.Cleanup(func() { getConfigurePathFunc, runtimeGOOSFunc, runtimeGOARCHFunc = prevPath, prevOS, prevArch })
	var stderr bytes.Buffer
	c := NewContext(cli.NewCommandContext(io.Discard, &stderr))
	if err := c.initialize(); err != nil {
		t.Fatal(err)
	}
	return c, &stderr
}

// A separate publisher keeps product download and checksum assertions observable.
func signedDgateRelease(t *testing.T, c *Context, version string, payload []byte) (*signedDgateFixture, *[]string) {
	t.Helper()
	var paths []string
	sum := sha256.Sum256(payload)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/latest.txt":
			_, _ = io.WriteString(w, version+"\n")
		case "/" + version + "/" + c.artifactName():
			_, _ = w.Write(payload)
		case "/" + version + "/" + c.artifactName() + ".sha256":
			_, _ = io.WriteString(w, hex.EncodeToString(sum[:])+"\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	t.Setenv(downloadBaseEnv, s.URL)
	return signedDgateManifest(t, c, version, payload), &paths
}

type dgateCloseHook struct {
	io.Reader
	close func() error
}

func (r dgateCloseHook) Close() error { return r.close() }

func dgateChecksumFixture(t *testing.T, payload []byte) {
	t.Helper()
	prev := httpDoFunc
	t.Cleanup(func() { httpDoFunc = prev })
	sum := sha256.Sum256(payload)
	httpDoFunc = func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, ".sha256") {
			t.Fatalf("unexpected fixture request: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(hex.EncodeToString(sum[:]) + "\n"))}, nil
	}
}

func TestMissingChecksumIsRejected(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }))
	defer s.Close()
	file := filepath.Join(t.TempDir(), "dgate")
	if err := os.WriteFile(file, []byte("unverified"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(s.URL+"/dgate.sha256", file); err == nil {
		t.Fatal("missing checksum must reject newly downloaded content")
	}
}

func TestSignedDgateEndpointSuffixAssetIdentity(t *testing.T) {
	for _, rewritten := range []bool{true, false} {
		t.Run(fmt.Sprintf("rewritten=%t", rewritten), func(t *testing.T) {
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			t.Setenv(downloadBaseEnv, "https://assets.oss.aliyuncs.com")
			c, _ := signedDgateContext(t)
			c.versionRemote, c.versionLocal, c.installed = "1.2.3", "1.2.2", true
			if err := os.WriteFile(c.execFilePath, []byte("old executable"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(c.versionFilePath, []byte(c.versionLocal), 0600); err != nil {
				t.Fatal(err)
			}
			payload := []byte("new executable")
			f := signedDgateManifest(t, c, c.versionRemote, payload)
			if rewritten {
				f.manifest.Assets[0].URL = strings.Replace(f.manifest.Assets[0].URL, ".aliyuncs.com", ".aliyuncs.alibaba", 1)
				f.sign(t)
			}
			previousClient, previousDo := httpClient, httpDoFunc
			t.Cleanup(func() { httpClient, httpDoFunc = previousClient, previousDo })
			calls := 0
			httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "assets.oss.aliyuncs.alibaba" {
					t.Fatalf("unexpected asset request: %s", r.URL)
				}
				body := payload
				switch r.URL.Path {
				case "/1.2.3/" + c.artifactName():
				case "/1.2.3/" + c.artifactName() + ".sha256":
					body = []byte(fmt.Sprintf("%x", sha256.Sum256(payload)))
				default:
					t.Fatalf("unexpected asset path: %s", r.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}, nil
			})}
			httpDoFunc = httpClient.Do
			err := c.install()
			want := "old executable"
			if rewritten {
				if err != nil || calls != 2 {
					t.Fatalf("signed destination rejected: err=%v calls=%d", err, calls)
				}
				want = string(payload)
			} else if err == nil || !strings.Contains(err.Error(), "not authorized") || calls != 0 {
				t.Fatalf("unsigned destination reached download: err=%v calls=%d", err, calls)
			}
			data, err := os.ReadFile(c.execFilePath)
			if err != nil || string(data) != want {
				t.Fatalf("installed=%q want=%q err=%v", data, want, err)
			}
			wantVersion := "1.2.2"
			if rewritten {
				wantVersion = "1.2.3"
			}
			version, err := os.ReadFile(c.versionFilePath)
			if err != nil || string(version) != wantVersion {
				t.Fatalf("version=%q want=%q err=%v", version, wantVersion, err)
			}
		})
	}
}

func TestSignedDgateDefaultsUseEmbeddedKeys(t *testing.T) {
	t.Setenv(assetmanifest.SourceEnv, "https://publisher.invalid/"+assetmanifest.FileName)
	c := NewContext(nil)
	if c.assets == nil || c.assets.SourceURL != os.Getenv(assetmanifest.SourceEnv) ||
		!reflect.DeepEqual(c.assets.Keys, publishsig.TrustedPublicKeys()) || c.assets.Now == nil || c.assets.HTTPClient == nil || c.rename == nil {
		t.Fatal("default context must use embedded publisher keys and os.Rename")
	}
}

func TestSignedDgateSuccessfulInstallAndUpdate(t *testing.T) {
	c, _ := signedDgateContext(t)
	_, paths := signedDgateRelease(t, c, "1.2.3", []byte("first package"))
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"/latest.txt", "/1.2.3/" + c.artifactName(), "/1.2.3/" + c.artifactName() + ".sha256"}
	if !reflect.DeepEqual(*paths, wantPaths) {
		t.Fatalf("product paths=%v", *paths)
	}
	if got, err := os.ReadFile(c.execFilePath); err != nil || string(got) != "first package" {
		t.Fatalf("installed=%q %v", got, err)
	}
	if err := os.Remove(c.checkVersionCacheFilePath); err != nil {
		t.Fatal(err)
	}
	f, _ := signedDgateRelease(t, c, "1.2.4", []byte("updated package"))
	f.manifest.Sequence++
	f.sign(t)
	if err := c.ensureInstalled(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(c.execFilePath); err != nil || string(got) != "updated package" {
		t.Fatalf("updated=%q %v", got, err)
	}
	if got, err := os.ReadFile(c.versionFilePath); err != nil || string(got) != "1.2.4" {
		t.Fatalf("version=%q %v", got, err)
	}
}

func TestSignedDgateActualInstalledHelperRunsAndUpdates(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("POSIX executable fixture; Windows execution is covered by the Go helper")
	}
	c, stderr := signedDgateContext(t)
	var stdout bytes.Buffer
	c.originCtx = cli.NewCommandContext(&stdout, stderr)
	for i, version := range []string{"1.2.3", "1.2.4"} {
		payload := []byte(fmt.Sprintf("#!/bin/sh\nprintf 'signed dgate helper %s\\n'\n", version))
		f, _ := signedDgateRelease(t, c, version, payload)
		f.manifest.Sequence += uint64(i)
		f.sign(t)
		if i != 0 {
			if err := os.Remove(c.checkVersionCacheFilePath); err != nil {
				t.Fatal(err)
			}
		}
		stdout.Reset()
		// Run executes the signed local shell fixture without execCommandFunc mocks.
		if err := c.Run([]string{"help"}); err != nil {
			t.Fatal(err)
		}
		if stdout.String() != "signed dgate helper "+version+"\n" {
			t.Fatalf("actual helper output=%q", stdout.String())
		}
	}
}

func TestSignedDgateRejectsUntrustedPublisher(t *testing.T) {
	for _, kind := range []string{"missing source", "HTTP source", "wrong basename", "manifest missing", "signature missing", "wrong key", "changed manifest", "changed signature", "expired", "future"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := signedDgateContext(t)
			f, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
			c.versionRemote = "1.2.3"
			switch kind {
			case "missing source":
				c.assets.SourceURL = ""
			case "HTTP source":
				c.assets.SourceURL = strings.Replace(c.assets.SourceURL, "https:", "http:", 1)
			case "wrong basename":
				c.assets.SourceURL += ".wrong"
			case "manifest missing":
				f.status = http.StatusNotFound
			case "signature missing":
				f.signatureStatus = http.StatusNotFound
			case "wrong key":
				pub, _, err := publishsig.Generate()
				if err != nil {
					t.Fatal(err)
				}
				c.assets.Keys = []ed25519.PublicKey{pub}
			case "changed manifest":
				f.data = append(f.data, ' ')
			case "changed signature":
				f.signature[len(f.signature)-3] ^= 1
			case "expired":
				f.manifest.IssuedAt = timeNowFunc().Add(-2 * time.Hour)
				f.manifest.ExpiresAt = timeNowFunc().Add(-time.Hour)
				f.sign(t)
			case "future":
				f.manifest.IssuedAt = timeNowFunc().Add(time.Hour)
				f.sign(t)
			}
			if err := c.install(); err == nil {
				t.Fatal("untrusted signed manifest accepted")
			}
			if err := c.ensureInstalled(); err == nil || !strings.Contains(err.Error(), "auto-download failed") {
				t.Fatalf("fresh install failed open on untrusted publisher: %v", err)
			}
			if len(*paths) != 0 {
				t.Fatalf("download before signature validation: %v", *paths)
			}
			if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
				t.Fatalf("untrusted package installed: %v", err)
			}
		})
	}
}

func TestSignedDgateAuthorizationConflicts(t *testing.T) {
	for _, kind := range []string{"product", "purpose", "platform", "version", "asset", "sourceURL", "digest", "size smaller", "size larger"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := signedDgateContext(t)
			f, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
			c.versionRemote = "1.2.3"
			e := &f.manifest.Assets[0]
			switch kind {
			case "product":
				e.Product = "other"
			case "purpose":
				e.Purpose = "other"
			case "platform":
				e.Platform = "other-amd64"
			case "version":
				e.Version = "1.2.4"
			case "asset":
				e.Asset = "other"
			case "sourceURL":
				e.URL += "?untrusted=1"
			case "digest":
				e.SHA256 = strings.Repeat("0", 64)
			case "size smaller":
				e.Size--
			case "size larger":
				e.Size++
			}
			f.sign(t)
			if err := c.install(); err == nil {
				t.Fatal("conflicting signed authorization accepted")
			}
			identity := kind != "digest" && !strings.HasPrefix(kind, "size")
			if identity && len(*paths) != 0 {
				t.Fatalf("download before identity authorization: %v", *paths)
			}
			if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
				t.Fatalf("conflicting package installed: %v", err)
			}
		})
	}
}

func TestSignedDgateRejectsReplayAndDowngrade(t *testing.T) {
	for _, kind := range []string{"sequence rollback", "same sequence changed bytes", "higher sequence lower version", "current version downgrade"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := signedDgateContext(t)
			f, paths := signedDgateRelease(t, c, "1.2.3", []byte("old package"))
			c.versionRemote = "1.2.3"
			if err := c.install(); err != nil {
				t.Fatal(err)
			}
			if err := c.install(); err != nil {
				t.Fatal("idempotent signed manifest rejected:", err)
			}
			*paths = nil
			switch kind {
			case "sequence rollback":
				f.manifest.Sequence--
			case "same sequence changed bytes":
				f.manifest.ExpiresAt = f.manifest.ExpiresAt.Add(time.Hour)
			case "higher sequence lower version":
				f.manifest.Sequence++
				c.versionRemote = "1.2.2"
				f.manifest.Assets[0].Version = c.versionRemote
				f.manifest.Assets[0].URL = c.baseURL() + "/" + c.versionRemote + "/" + c.artifactName()
				c.versionLocal = "" // Exercise persisted highest version, not only the in-memory version.
				if err := os.Remove(c.versionFilePath); err != nil {
					t.Fatal(err)
				}
			case "current version downgrade":
				c.versionLocal = "1.2.4"
			}
			f.sign(t)
			if err := c.install(); err == nil || (!strings.Contains(err.Error(), "sequence") && !strings.Contains(err.Error(), "downgrade")) {
				t.Fatalf("signed rollback not rejected at authorization boundary: %v", err)
			}
			if len(*paths) != 0 {
				t.Fatalf("download before replay rejection: %v", *paths)
			}
			if got, err := os.ReadFile(c.execFilePath); err != nil || string(got) != "old package" {
				t.Fatalf("old package lost: %q %v", got, err)
			}
		})
	}
}

func TestSignedDgateFailedUpdateRunUsesOnlyOldPackage(t *testing.T) {
	for _, kind := range []string{"missing signature", "digest", "size"} {
		t.Run(kind, func(t *testing.T) {
			c, stderr := signedDgateContext(t)
			if err := os.WriteFile(c.execFilePath, []byte("old package"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(c.versionFilePath, []byte("1.2.2"), 0644); err != nil {
				t.Fatal(err)
			}
			f, _ := signedDgateRelease(t, c, "1.2.3", []byte("REJECTED NEW CONTENT"))
			switch kind {
			case "missing signature":
				f.signatureStatus = http.StatusNotFound
			case "digest":
				f.manifest.Assets[0].SHA256 = strings.Repeat("0", 64)
				f.sign(t)
			case "size":
				f.manifest.Assets[0].Size++
				f.sign(t)
			}
			prev := execCommandFunc
			t.Cleanup(func() { execCommandFunc = prev })
			runs := 0
			execCommandFunc = func(name string, args ...string) *exec.Cmd {
				got, err := os.ReadFile(name)
				if err != nil || string(got) != "old package" {
					t.Fatalf("executing rejected content: %q %v", got, err)
				}
				if !reflect.DeepEqual(args, []string{"help", "--region", "cn-beijing"}) {
					t.Fatalf("argv changed: %v", args)
				}
				runs++
				return helperExit(0)
			}
			if err := c.Run([]string{"--profile", "demo", "help", "--region", "cn-beijing"}); err != nil {
				t.Fatal(err)
			}
			if runs != 1 || !strings.Contains(stderr.String(), "failed to update") {
				t.Fatalf("old helper runs=%d stderr=%s", runs, stderr)
			}
			if got, err := os.ReadFile(c.versionFilePath); err != nil || string(got) != "1.2.2" {
				t.Fatalf("old version lost: %q %v", got, err)
			}
		})
	}
}

func TestSignedDgateOverrideRunsOfflineWithoutManifest(t *testing.T) {
	c, _ := signedDgateContext(t)
	bin := filepath.Join(t.TempDir(), "fake-dgate")
	if err := os.WriteFile(bin, []byte("local package"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	c.assets = &assetmanifest.Client{SourceURL: "https://publisher.invalid/" + assetmanifest.FileName,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("override queried signed manifest")
			return nil, os.ErrNotExist
		})}}
	prev, prevVersion := execCommandFunc, getRemoteVersionFunc
	t.Cleanup(func() { execCommandFunc, getRemoteVersionFunc = prev, prevVersion })
	getRemoteVersionFunc = func(string) (string, error) { t.Fatal("override queried product version"); return "", os.ErrNotExist }
	runs := 0
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		if name != bin {
			t.Fatalf("wrong override %s", name)
		}
		runs++
		return helperExit(0)
	}
	if err := c.Run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Fatalf("helper runs=%d", runs)
	}
}

func TestSignedDgateReplayStateSurvivesContextRecreation(t *testing.T) {
	c, _ := signedDgateContext(t)
	f, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
	c.versionRemote = "1.2.3"
	if err := c.install(); err != nil {
		t.Fatal(err)
	}
	f.manifest.Sequence--
	f.sign(t)
	other := NewContext(c.originCtx)
	other.assets = c.assets
	if err := other.initialize(); err != nil {
		t.Fatal(err)
	}
	other.versionRemote = c.versionRemote
	*paths = nil
	if err := other.install(); err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("persisted replay state bypassed: %v", err)
	}
	if len(*paths) != 0 {
		t.Fatalf("replay downloaded after restart: %v", *paths)
	}
}

func TestSignedDgateSignedSizeLimitAndMissingDownloadedFile(t *testing.T) {
	for _, kind := range []string{"signed size limit", "download removed file", "download failure", "corrupt state"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := signedDgateContext(t)
			f, _ := signedDgateRelease(t, c, "1.2.3", []byte("package"))
			c.versionRemote = "1.2.3"
			prev := downloadBinaryFunc
			t.Cleanup(func() { downloadBinaryFunc = prev })
			calls := 0
			downloadBinaryFunc = func(_ string, dest string) error {
				calls++
				if kind == "download failure" {
					return os.ErrPermission
				}
				return os.Remove(dest)
			}
			if kind == "signed size limit" {
				f.manifest.Assets[0].Size = maxDownloadSize + 1
				f.sign(t)
			}
			if kind == "corrupt state" {
				if err := os.WriteFile(filepath.Join(c.configPath, ".dgate-asset-manifest-state.json"), []byte("not JSON"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.install(); err == nil {
				t.Fatal("unsafe install accepted")
			}
			want := 1
			if kind == "signed size limit" || kind == "corrupt state" {
				want = 0
			}
			if calls != want {
				t.Fatalf("download branch calls=%d want=%d", calls, want)
			}
			if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
				t.Fatalf("invalid package installed: %v", err)
			}
			entries, err := os.ReadDir(c.configPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".dgate-install-") || strings.HasPrefix(entry.Name(), ".dgate-backup-") {
					t.Fatalf("temporary file/lock leaked: %s", entry.Name())
				}
			}
		})
	}
}

func TestSignedDgateMissingExecutableDirectoryFailsBeforeDownload(t *testing.T) {
	c, _ := signedDgateContext(t)
	f, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
	c.versionRemote = "1.2.3"
	c.execFilePath = filepath.Join(c.configPath, "missing-directory", "dgate")
	if err := c.install(); !os.IsNotExist(err) {
		t.Fatalf("temporary-file creation error not returned: %v", err)
	}
	if f.requests == 0 {
		t.Fatal("test did not reach signed authorization")
	}
	if len(*paths) != 0 {
		t.Fatalf("download after temporary-file creation failed: %v", *paths)
	}
	if _, err := os.Stat(filepath.Join(c.configPath, ".dgate-install-lock")); !os.IsNotExist(err) {
		t.Fatalf("lock leaked: %v", err)
	}
}

func TestSignedDgateManagedPathsRejectSymlinksAndDirectories(t *testing.T) {
	for _, component := range []string{"config", "executable", "version", "authorization state"} {
		for _, symlink := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/symlink=%t", component, symlink), func(t *testing.T) {
				c, _ := signedDgateContext(t)
				_, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
				c.versionRemote = "1.2.3"
				target := c.execFilePath
				switch component {
				case "config":
					target = c.configPath
					if err := os.Remove(target); err != nil {
						t.Fatal(err)
					}
				case "version":
					target = c.versionFilePath
				case "authorization state":
					target = filepath.Join(c.configPath, ".dgate-asset-manifest-state.json")
				}
				outside := filepath.Join(t.TempDir(), "untouched")
				if err := os.WriteFile(outside, []byte("outside content"), 0600); err != nil {
					t.Fatal(err)
				}
				if symlink {
					if err := os.Symlink(outside, target); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				} else if component == "config" {
					if err := os.WriteFile(target, []byte("not a directory"), 0600); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				if err := c.install(); err == nil || !strings.Contains(err.Error(), "managed dgate path") {
					t.Fatalf("unsafe path accepted: %v", err)
				}
				if len(*paths) != 0 {
					t.Fatalf("download despite unsafe path: %v", *paths)
				}
				if got, err := os.ReadFile(outside); err != nil || string(got) != "outside content" {
					t.Fatalf("outside file changed: %q %v", got, err)
				}
			})
		}
	}
}

func TestSignedDgateWindowsRenameWithoutRegularDestination(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			c, _ := signedDgateContext(t)
			runtimeGOOSFunc = func() string { return "windows" }
			if err := c.initialize(); err != nil {
				t.Fatal(err)
			}
			if directory {
				if err := os.Mkdir(c.execFilePath, 0700); err != nil {
					t.Fatal(err)
				}
			}
			failure := fmt.Errorf("rename blocked")
			calls := 0
			c.rename = func(string, string) error { calls++; return failure }
			if err := c.replaceExecutable(filepath.Join(c.configPath, "missing temporary")); err != failure || calls != 1 {
				t.Fatalf("replacement=%v calls=%d", err, calls)
			}
		})
	}
}

func TestSignedDgateInstallAlwaysRefetchesAuthorization(t *testing.T) {
	c, _ := signedDgateContext(t)
	f, paths := signedDgateRelease(t, c, "1.2.3", []byte("package"))
	c.versionRemote = "1.2.3"
	if err := c.install(); err != nil {
		t.Fatal(err)
	}
	*paths = nil
	f.signatureStatus = http.StatusNotFound
	if err := c.install(); err == nil {
		t.Fatal("cached verification bypassed mandatory install signature fetch")
	}
	if len(*paths) != 0 {
		t.Fatalf("download after signature disappeared: %v", *paths)
	}
}

func TestSignedDgateConcurrentInstallUsesIndependentTemporaryFile(t *testing.T) {
	c, _ := signedDgateContext(t)
	_, _ = signedDgateRelease(t, c, "1.2.3", []byte("package"))
	c.versionRemote = "1.2.3"
	fixedTemp := c.execFilePath + ".tmp"
	if err := os.WriteFile(fixedTemp, []byte("other install's bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	started, finish := make(chan string, 1), make(chan struct{})
	prev := downloadBinaryFunc
	t.Cleanup(func() { downloadBinaryFunc = prev })
	downloadBinaryFunc = func(_ string, dest string) error {
		started <- dest
		<-finish
		return os.WriteFile(dest, []byte("package"), 0600)
	}
	result := make(chan error, 1)
	go func() { result <- c.install() }()
	var temporary string
	select {
	case temporary = <-started:
	case err := <-result:
		t.Fatalf("original download branch not reached: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("installer did not reach download")
	}
	other := *c
	err := other.install()
	close(finish)
	if firstErr := <-result; firstErr != nil {
		t.Fatal(firstErr)
	}
	if err == nil || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("concurrent installer not fenced: %v", err)
	}
	if temporary == fixedTemp || filepath.Dir(temporary) != filepath.Dir(c.execFilePath) {
		t.Fatalf("shared temporary path used: %s", temporary)
	}
	if got, err := os.ReadFile(fixedTemp); err != nil || string(got) != "other install's bytes" {
		t.Fatalf("foreign temporary file overwritten: %q %v", got, err)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatalf("temporary file leaked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(c.configPath, ".dgate-install-lock")); !os.IsNotExist(err) {
		t.Fatalf("lock leaked: %v", err)
	}
}

func TestSignedDgateWindowsRenameFailurePreservesOld(t *testing.T) {
	for _, kind := range []string{"POSIX rename", "Windows success", "Windows backup", "Windows rollback", "Windows rollback failure"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := signedDgateContext(t)
			if kind != "POSIX rename" {
				runtimeGOOSFunc = func() string { return "windows" }
				if err := c.initialize(); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(c.execFilePath, []byte("old package"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(c.versionFilePath, []byte("1.2.2"), 0644); err != nil {
				t.Fatal(err)
			}
			c.installed = true
			c.versionLocal = "1.2.2"
			c.versionRemote = "1.2.3"
			signedDgateRelease(t, c, "1.2.3", []byte("new package"))
			calls := 0
			c.rename = func(from, to string) error {
				calls++
				if calls == 1 || kind == "Windows backup" && calls == 2 ||
					(kind == "Windows rollback" || kind == "Windows rollback failure") && calls == 3 || kind == "Windows rollback failure" && calls == 4 {
					return fmt.Errorf("rename blocked")
				}
				return os.Rename(from, to)
			}
			err := c.install()
			if (err == nil) != (kind == "Windows success") {
				t.Fatalf("result=%v calls=%d", err, calls)
			}
			if kind == "Windows rollback failure" {
				if err == nil || !strings.Contains(err.Error(), "retained at") {
					t.Fatalf("missing recovery path: %v", err)
				}
				entries, err := os.ReadDir(c.configPath)
				if err != nil {
					t.Fatal(err)
				}
				found := 0
				for _, entry := range entries {
					data, err := os.ReadFile(filepath.Join(c.configPath, entry.Name()))
					if err == nil && string(data) == "old package" {
						found++
					}
				}
				if found != 1 {
					t.Fatalf("old package not retained exactly once: %v", entries)
				}
			} else {
				want := "old package"
				if kind == "Windows success" {
					want = "new package"
				}
				if got, err := os.ReadFile(c.execFilePath); err != nil || string(got) != want {
					t.Fatalf("package=%q %v", got, err)
				}
			}
			if kind != "Windows success" {
				if got, err := os.ReadFile(c.versionFilePath); err != nil || string(got) != "1.2.2" {
					t.Fatalf("version overwritten on failure: %q %v", got, err)
				}
			}
		})
	}
}
