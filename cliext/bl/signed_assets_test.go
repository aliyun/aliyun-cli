package bl

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

	"github.com/aliyun/aliyun-cli/v3/cli/assetmanifest"
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
)

func TestSignedBLExpiryDuringDownloadPreservesOldExecutable(t *testing.T) {
	c, _ := binaryContext(t)
	installOld(t, c, "1.2.2")
	a, payload := assetFixture(t, "1.2.3", binaryPlatform(t, c))
	f := signedManifestServer(t, c, a, payload)
	now := c.now()
	c.assets.Now = func() time.Time { return now }
	c.httpClient.Transport = binaryRoundTripper(func(*http.Request) (*http.Response, error) {
		now = f.manifest.ExpiresAt
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	})
	if err := c.installAsset(a); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired authorization installed: %v", err)
	}
	data, err := os.ReadFile(c.execFilePath)
	if err != nil || string(data) != "old executable" {
		t.Fatalf("expired new binary replaced old binary: %q, %v", data, err)
	}
}

type signedBLFixture struct {
	manifest                assetmanifest.Manifest
	data, signature         []byte
	private                 ed25519.PrivateKey
	status, signatureStatus int
	requests                int
}

func (f *signedBLFixture) sign(t *testing.T) {
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

func signedManifestServer(t *testing.T, c *Context, a releaseAsset, payload []byte) *signedBLFixture {
	t.Helper()
	// Direct resolveAsset tests run outside EnsureBinary's directory setup.
	if err := os.MkdirAll(c.prefixPath, 0755); err != nil {
		t.Fatal(err)
	}
	pub, private, err := publishsig.Generate()
	if err != nil {
		t.Fatal(err)
	}
	now := c.now()
	f := &signedBLFixture{private: private, status: http.StatusOK, signatureStatus: http.StatusOK,
		manifest: assetmanifest.Manifest{Format: assetmanifest.Format, Sequence: 10,
			IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour),
			Assets: []assetmanifest.Entry{{Product: "bl", Purpose: assetmanifest.Purpose,
				Version: a.Version, Platform: binaryPlatform(t, c), Asset: a.File,
				URL: c.releaseURL + "/v" + a.Version + "/" + a.File, SHA256: a.SHA256,
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

func TestSignedBLEndpointSuffixAssetIdentity(t *testing.T) {
	for _, rewritten := range []bool{true, false} {
		t.Run(fmt.Sprintf("rewritten=%t", rewritten), func(t *testing.T) {
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			c, _ := binaryContext(t)
			installOld(t, c, "1.2.2")
			c.versionLocal = "1.2.2"
			c.releaseURL = "https://assets.oss.aliyuncs.com"
			a, payload := assetFixture(t, "1.2.3", binaryPlatform(t, c))
			f := signedManifestServer(t, c, a, payload)
			if rewritten {
				f.manifest.Assets[0].URL = strings.Replace(f.manifest.Assets[0].URL, ".aliyuncs.com", ".aliyuncs.alibaba", 1)
				f.sign(t)
			}
			calls := 0
			c.httpClient.Transport = binaryRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "assets.oss.aliyuncs.alibaba" || r.URL.Path != "/v1.2.3/"+a.File {
					t.Fatalf("unexpected asset request: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(payload))}, nil
			})
			err := c.installAsset(a)
			want := "old executable"
			if rewritten {
				if err != nil || calls != 1 {
					t.Fatalf("signed destination rejected: err=%v calls=%d", err, calls)
				}
				want = "standalone-" + a.Version
			} else if err == nil || !strings.Contains(err.Error(), "not authorized") || calls != 0 {
				t.Fatalf("unsigned destination reached download: err=%v calls=%d", err, calls)
			}
			data, err := os.ReadFile(c.execFilePath)
			if err != nil || string(data) != want {
				t.Fatalf("installed=%q want=%q err=%v", data, want, err)
			}
		})
	}
}

func TestSignedBLDefaultClientUsesEmbeddedKeys(t *testing.T) {
	t.Setenv(assetmanifest.SourceEnv, "https://publisher.invalid/"+assetmanifest.FileName)
	c := NewContext(nil)
	if c.assets == nil || c.assets.SourceURL != os.Getenv(assetmanifest.SourceEnv) ||
		!reflect.DeepEqual(c.assets.Keys, publishsig.TrustedPublicKeys()) || c.assets.Now == nil || c.assets.HTTPClient == nil {
		t.Fatal("default client must use the explicit source and embedded publisher keys")
	}
}

func TestSignedBLRejectsUntrustedManifestBeforeProductFetch(t *testing.T) {
	for _, kind := range []string{"missing source", "HTTP source", "wrong basename", "manifest missing", "signature missing", "wrong key", "changed manifest", "changed signature", "expired", "future"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			f := signedManifestServer(t, c, a, payload)
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
				f.manifest.IssuedAt = c.now().Add(-2 * time.Hour)
				f.manifest.ExpiresAt = c.now().Add(-time.Hour)
				f.sign(t)
			case "future":
				f.manifest.IssuedAt = c.now().Add(time.Hour)
				f.sign(t)
			}
			if err := c.EnsureBinary(); err == nil {
				t.Fatal("untrusted publisher manifest accepted")
			}
			if len(paths) != 0 {
				t.Fatalf("product fetched before signature validation: %v", paths)
			}
			if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
				t.Fatalf("untrusted executable created: %v", err)
			}
		})
	}
}

func TestSignedBLAuthorizationConflicts(t *testing.T) {
	for _, kind := range []string{"product", "purpose", "platform", "version", "asset", "sourceURL", "digest", "size smaller", "size larger"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/direct=%t", kind, direct), func(t *testing.T) {
				c, _ := binaryContext(t)
				a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
				var paths []string
				releaseServer(t, c, a, payload, &paths)
				f := signedManifestServer(t, c, a, payload)
				e := &f.manifest.Assets[0]
				switch kind {
				case "product":
					e.Product = "other"
				case "purpose":
					e.Purpose = "other"
				case "platform":
					e.Platform = "other-amd64"
				case "version":
					e.Version = "2.6.3"
				case "asset":
					e.Asset = "other.zip"
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
				var err error
				if direct {
					err = c.installAsset(a)
				} else {
					err = c.EnsureBinary()
				}
				if err == nil {
					t.Fatal("conflicting signed authorization accepted")
				}
				if _, err := os.Stat(c.execFilePath); !os.IsNotExist(err) {
					t.Fatalf("conflicting package installed: %v", err)
				}
				identity := kind != "digest" && !strings.HasPrefix(kind, "size")
				for _, path := range paths {
					if identity && strings.HasSuffix(path, ".zip") {
						t.Fatalf("download before identity authorization: %v", paths)
					}
				}
			})
		}
	}
}

func TestSignedBLDirectInstallValidatesContract(t *testing.T) {
	for _, kind := range []string{"unsupported platform", "invalid version", "wrong file", "wrong inner", "signed size limit", "corrupt state"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			f := signedManifestServer(t, c, a, payload)
			switch kind {
			case "unsupported platform":
				c.goos = func() string { return "freebsd" }
			case "invalid version":
				a.Version = "../invalid"
			case "wrong file":
				a.File = "other.zip"
			case "wrong inner":
				a.Inner = "other"
			case "signed size limit":
				f.manifest.Assets[0].Size = binaryLimit + 1
				f.sign(t)
			case "corrupt state":
				if err := os.MkdirAll(c.prefixPath, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(c.prefixPath, ".asset-manifest-state.json"), []byte("invalid JSON"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.installAsset(a); err == nil {
				t.Fatal("unsafe direct installation accepted")
			}
			if len(paths) != 0 {
				t.Fatalf("product download before direct validation: %v", paths)
			}
		})
	}
}

func TestSignedBLInstallAlwaysRefetchesAuthorization(t *testing.T) {
	c, _ := binaryContext(t)
	a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
	var paths []string
	releaseServer(t, c, a, payload, &paths)
	f := signedManifestServer(t, c, a, payload)
	if _, err := c.resolveAsset(binaryPlatform(t, c)); err != nil {
		t.Fatal(err)
	}
	paths = nil
	f.signatureStatus = http.StatusNotFound
	if err := c.installAsset(a); err == nil {
		t.Fatal("cached verification bypassed mandatory install verification")
	}
	if len(paths) != 0 {
		t.Fatalf("download after publisher signature disappeared: %v", paths)
	}
}

func TestSignedBLSequenceReplayAndDowngrade(t *testing.T) {
	for _, kind := range []string{"sequence rollback", "same sequence changed bytes", "higher sequence lower version", "current version downgrade"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := binaryContext(t)
			a, payload := assetFixture(t, "2.7.1", binaryPlatform(t, c))
			var paths []string
			releaseServer(t, c, a, payload, &paths)
			f := signedManifestServer(t, c, a, payload)
			if err := c.installAsset(a); err != nil {
				t.Fatal(err)
			}
			// Identical signed retries are idempotent, unlike rollback or equivocation.
			if err := c.installAsset(a); err != nil {
				t.Fatal("idempotent manifest rejected:", err)
			}
			paths = nil
			switch kind {
			case "sequence rollback":
				f.manifest.Sequence--
			case "same sequence changed bytes":
				f.manifest.ExpiresAt = f.manifest.ExpiresAt.Add(time.Hour)
			case "higher sequence lower version":
				f.manifest.Sequence++
				a, payload = assetFixture(t, "2.6.2", binaryPlatform(t, c))
				e := &f.manifest.Assets[0]
				e.Version, e.Asset, e.SHA256, e.Size = a.Version, a.File, a.SHA256, int64(len(payload))
				e.URL = c.releaseURL + "/v" + a.Version + "/" + a.File
				c.versionLocal = "" // Persisted highest authorization must still prevent downgrade.
			case "current version downgrade":
				c.versionLocal = "2.8.0"
			}
			f.sign(t)
			if err := c.installAsset(a); err == nil || (!strings.Contains(err.Error(), "sequence") && !strings.Contains(err.Error(), "downgrade")) {
				t.Fatalf("signed rollback not rejected at authorization boundary: %v", err)
			}
			if len(paths) != 0 {
				t.Fatalf("download before rollback rejection: %v", paths)
			}
			got, err := os.ReadFile(c.execFilePath)
			if err != nil || string(got) != "standalone-2.7.1" {
				t.Fatalf("old package damaged: %q %v", got, err)
			}
		})
	}
}

func TestSignedBLRunKeepsOldAndDoesNotExecuteRejectedContent(t *testing.T) {
	c, stderr := binaryContext(t)
	installOld(t, c, "2.6.2")
	a, payload := assetFixture(t, "2.7.1", binaryPlatform(t, c))
	releaseServer(t, c, a, payload, nil)
	f := signedManifestServer(t, c, a, payload)
	f.manifest.Assets[0].SHA256 = strings.Repeat("0", 64)
	f.sign(t)
	versionCommand := c.command
	runs := 0
	c.command = func(name string, args ...string) *exec.Cmd {
		if reflect.DeepEqual(args, []string{"--version"}) {
			return versionCommand(name, args...)
		}
		got, err := os.ReadFile(name)
		if err != nil || string(got) != "old executable" {
			t.Fatalf("executing rejected content: %q %v", got, err)
		}
		if !reflect.DeepEqual(args, []string{"help", "--literal=unchanged"}) {
			t.Fatalf("argv changed: %v", args)
		}
		runs++
		return helperCommand(0)
	}
	if err := c.Run([]string{"help", "--literal=unchanged"}); err != nil {
		t.Fatal(err)
	}
	if runs != 1 || !strings.Contains(stderr.String(), "Warning") {
		t.Fatalf("old helper not actually run: %d %s", runs, stderr)
	}
}

func TestSignedBLOverrideRunsOfflineWithoutManifest(t *testing.T) {
	c, _ := binaryContext(t)
	bin := filepath.Join(t.TempDir(), "fake-bl")
	if err := os.WriteFile(bin, []byte("trusted local"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(execPathEnv, bin)
	c.assets = &assetmanifest.Client{SourceURL: "https://publisher.invalid/" + assetmanifest.FileName,
		HTTPClient: &http.Client{Transport: binaryRoundTripper(func(*http.Request) (*http.Response, error) {
			t.Fatal("override queried signed manifest")
			return nil, os.ErrNotExist
		})}}
	runs := 0
	c.command = func(name string, args ...string) *exec.Cmd {
		if name != bin {
			t.Fatalf("wrong override %s", name)
		}
		runs++
		return helperCommand(0)
	}
	if err := c.Run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Fatalf("helper runs=%d", runs)
	}
}

func TestSignedBLReplayStateSurvivesContextRecreation(t *testing.T) {
	c, _ := binaryContext(t)
	a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
	var paths []string
	releaseServer(t, c, a, payload, &paths)
	f := signedManifestServer(t, c, a, payload)
	if err := c.installAsset(a); err != nil {
		t.Fatal(err)
	}
	f.manifest.Sequence--
	f.sign(t)
	other := NewContext(c.originCtx)
	other.configPath, other.goos, other.goarch = c.configPath, c.goos, c.goarch
	other.releaseURL, other.httpClient, other.assets = c.releaseURL, c.httpClient, c.assets
	if err := other.InitBasicInfo(); err != nil {
		t.Fatal(err)
	}
	paths = nil
	if err := other.installAsset(a); err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("persisted replay state bypassed: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("replayed package downloaded after restart: %v", paths)
	}
}

func TestSignedBLActualInstalledHelperRunsAndUpdates(t *testing.T) {
	c, _ := binaryContext(t)
	if c.goos() == "windows" {
		t.Skip("POSIX executable fixture; Windows execution is covered by the Go helper")
	}
	stdout := c.originCtx.Stdout().(*bytes.Buffer)
	for i, version := range []string{"2.6.2", "2.7.1"} {
		a := expectedAsset(version, binaryPlatform(t, c))
		script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then printf '%s\\n'; else printf 'signed helper %s\\n'; fi\n", version, version)
		payload := zipFixture(t, []string{a.Inner}, 0755, []byte(script))
		sum := sha256.Sum256(payload)
		a.SHA256 = hex.EncodeToString(sum[:])
		releaseServer(t, c, a, payload, nil)
		f := signedManifestServer(t, c, a, payload)
		f.manifest.Sequence += uint64(i)
		f.sign(t)
		if i != 0 {
			if err := os.Remove(filepath.Join(c.prefixPath, "update-check.json")); err != nil {
				t.Fatal(err)
			}
		}
		stdout.Reset()
		// No command override: Run actually executes the signed ZIP payload.
		if err := c.Run([]string{"help"}); err != nil {
			t.Fatal(err)
		}
		if stdout.String() != "signed helper "+version+"\n" {
			t.Fatalf("actual helper output=%q", stdout.String())
		}
	}
}

func TestSignedBLPayloadDigestAndSizeMatchFixture(t *testing.T) {
	c, _ := binaryContext(t)
	a, payload := assetFixture(t, "2.6.2", binaryPlatform(t, c))
	releaseServer(t, c, a, payload, nil)
	sum := sha256.Sum256(payload)
	if a.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("fixture digest does not authenticate exact ZIP bytes")
	}
	if err := c.installAsset(a); err != nil {
		t.Fatal(err)
	}
}
