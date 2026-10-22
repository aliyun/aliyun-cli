package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePluginTransportURL(t *testing.T) {
	m := &Manager{}
	assert.NoError(t, m.validatePluginTransportURL("https://mirror.example/plugins/plugin.tgz"))
	assert.NoError(t, m.validatePluginTransportURL("http://127.0.0.1:8080/plugin.tgz"))
	assert.NoError(t, m.validatePluginTransportURL("http://localhost/plugin.tgz"))
	assert.NoError(t, m.validatePluginTransportURL("http://[::1]/plugin.tgz"))
	for _, raw := range []string{
		"http://mirror.example/plugin.tgz",
		"http://127.0.0.1:8080/plugin.tgz",
		"http://localhost/plugin.tgz",
	} {
		err := validatePluginTransport(raw, false)
		assert.ErrorContains(t, err, "refusing insecure HTTP")
	}
	assert.ErrorContains(t, m.validatePluginTransportURL("http://mirror.example/plugin.tgz"), "refusing insecure HTTP")
	assert.ErrorContains(t, m.validatePluginTransportURL("ftp://mirror.example/plugin.tgz"), "https://")
	assert.ErrorContains(t, m.validatePluginTransportURL("http:///plugin.tgz"), "https://")
	assert.ErrorContains(t, m.validatePluginTransportURL("://bad"), "https://")
	assert.False(t, httpsDowngradedToHTTP(nil, nil))
}

func TestCheckPluginRedirect(t *testing.T) {
	req := func(raw string) *http.Request {
		u, err := url.Parse(raw)
		require.NoError(t, err)
		return &http.Request{URL: u}
	}

	m := &Manager{}
	err := m.checkPluginRedirect(req("http://127.0.0.1/plugin.tgz"), []*http.Request{req("https://aliyuncli.alicdn.com/plugin.tgz")})
	assert.ErrorContains(t, err, "downgrades HTTPS to HTTP")

	err = m.checkPluginRedirect(req("http://mirror.example/plugin.tgz"), []*http.Request{req("http://127.0.0.1/plugin.tgz")})
	assert.ErrorContains(t, err, "refusing insecure HTTP")

	assert.NoError(t, m.checkPluginRedirect(req("https://mirror.example/plugin.tgz"), []*http.Request{req("https://aliyuncli.alicdn.com/start.tgz")}))
	assert.ErrorContains(t, m.checkPluginRedirect(nil, nil), "invalid plugin redirect")

	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = req("https://mirror.example/a.tgz")
	}
	assert.ErrorContains(t, m.checkPluginRedirect(req("https://mirror.example/b.tgz"), via), "10 redirects")
	err = m.checkPluginRedirect(req("http://mirror.example/plugin.tgz"), []*http.Request{req("https://aliyuncli.alicdn.com/plugin.tgz")})
	assert.ErrorContains(t, err, "refusing insecure HTTP")
}

func TestNormalizeAndVerifySHA256(t *testing.T) {
	body := []byte("plugin-bytes")
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])

	got, err := normalizeSHA256Checksum("SHA256:" + strings.ToUpper(digest))
	require.NoError(t, err)
	assert.Equal(t, digest, got)
	_, err = normalizeSHA256Checksum("abcd")
	assert.ErrorContains(t, err, "64-character")
	_, err = normalizeSHA256Checksum(strings.Repeat("zz", 32))
	assert.ErrorContains(t, err, "64-character")

	path := filepath.Join(t.TempDir(), "plugin.tgz")
	require.NoError(t, os.WriteFile(path, body, 0644))
	require.NoError(t, verifySHA256File(path, digest))
	err = verifySHA256File(path, strings.Repeat("ab", 32))
	assert.ErrorContains(t, err, "checksum verification failed")
	assert.ErrorContains(t, err, "Expected: "+strings.Repeat("ab", 32))
	assert.ErrorContains(t, verifySHA256File(filepath.Join(t.TempDir(), "missing.tgz"), digest), "failed to calculate checksum")
}

func TestReportPackageInstall(t *testing.T) {
	stdout, stderr := new(strings.Builder), new(strings.Builder)
	ctx := cli.NewCommandContext(stdout, stderr)
	reportPackageInstall(ctx, "demo", "1.0.0", true)
	reportPackageInstall(ctx, "demo", "1.0.0", false)
	out := stdout.String()
	assert.Contains(t, out, "Plugin demo 1.0.0 installed successfully!")
	assert.Contains(t, out, "SHA-256 checksum verified.")
	assert.Contains(t, out, "package integrity was not checked")
	assert.NotContains(t, strings.Split(out, "package integrity")[1], "SHA-256 checksum verified")
}

func TestInstallFromRemotePackageURL_RefusesPublicHTTPAndRedirects(t *testing.T) {
	mgr := &Manager{rootDir: t.TempDir()}
	ctx := newTestContext()

	err := mgr.installFromRemotePackageURL(ctx, "http://example.com/plugin.tgz")
	require.Error(t, err)
	assert.ErrorContains(t, err, "refusing insecure HTTP")

	redirectHits := 0
	redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectHits++
		http.Redirect(w, r, "http://example.com/evil.tgz", http.StatusFound)
	}))
	defer redirectSrv.Close()
	err = mgr.installFromRemotePackageURL(ctx, redirectSrv.URL+"/plugin.tgz")
	require.Error(t, err)
	assert.ErrorContains(t, err, "refusing insecure HTTP")
	assert.Equal(t, 1, redirectHits)

	plainHits := 0
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plainHits++
		http.NotFound(w, r)
	}))
	defer plain.Close()
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/plugin.tgz", http.StatusFound)
	}))
	defer tlsSrv.Close()
	tlsMgr := &Manager{rootDir: t.TempDir(), httpTransport: tlsSrv.Client().Transport}
	err = tlsMgr.installFromRemotePackageURL(ctx, tlsSrv.URL+"/plugin.tgz")
	require.Error(t, err)
	assert.ErrorContains(t, err, "downgrades HTTPS to HTTP")
	assert.Equal(t, 0, plainHits)
}

func TestInstallFromPackage_ChecksumOutcomes(t *testing.T) {
	archiveBody := createTestPluginArchive(t, "checksum-plugin", "1.2.3", "x")
	sum := sha256.Sum256(archiveBody)
	digest := hex.EncodeToString(sum[:])

	start := func() (*Manager, *cli.Context, string) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(archiveBody)
		}))
		t.Cleanup(srv.Close)
		return &Manager{rootDir: t.TempDir()}, newTestContext(), srv.URL + "/plugin.tgz"
	}

	t.Run("wrong checksum is not installed or marked verified", func(t *testing.T) {
		mgr, ctx, rawURL := start()
		mgr.expectedPackageChecksum = strings.Repeat("ab", 32)
		err := mgr.InstallFromPackage(ctx, rawURL)
		require.Error(t, err)
		assert.ErrorContains(t, err, "checksum verification failed")
		text := ctx.Stdout().(interface{ String() string }).String()
		assert.NotContains(t, text, "SHA-256 checksum verified")
		assert.NotContains(t, text, "installed successfully")
		manifest, err := mgr.GetLocalManifest()
		require.NoError(t, err)
		_, ok := manifest.Plugins["checksum-plugin"]
		assert.False(t, ok)
	})

	t.Run("malformed checksum", func(t *testing.T) {
		mgr, ctx, rawURL := start()
		mgr.expectedPackageChecksum = "not-a-digest"
		err := mgr.InstallFromPackage(ctx, rawURL)
		require.Error(t, err)
		assert.ErrorContains(t, err, "64-character")
	})

	t.Run("correct checksum is a verified install", func(t *testing.T) {
		mgr, ctx, rawURL := start()
		mgr.expectedPackageChecksum = "sha256:" + digest
		require.NoError(t, mgr.InstallFromPackage(ctx, rawURL))
		text := ctx.Stdout().(interface{ String() string }).String()
		assert.Contains(t, text, "SHA-256 checksum verified")
		assert.NotContains(t, text, "package integrity was not checked")
		manifest, err := mgr.GetLocalManifest()
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", manifest.Plugins["checksum-plugin"].Version)
	})
}

func TestExecuteInstall_ChecksumFlag(t *testing.T) {
	testHome := t.TempDir()
	cleanup := setTestHomeDir(t, testHome)
	defer cleanup()

	archiveBody := createTestPluginArchive(t, "cli-checksum", "9.9.9", "x")
	sum := sha256.Sum256(archiveBody)
	archivePath := filepath.Join(t.TempDir(), "cli-checksum.tgz")
	require.NoError(t, os.WriteFile(archivePath, archiveBody, 0644))

	run := func(assign map[string]string, none ...string) (string, error) {
		cmd := newInstallCommand()
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)
		for name, value := range assign {
			f := ctx.Flags().Get(name)
			require.NotNil(t, f, name)
			f.SetAssigned(true)
			f.SetValue(value)
		}
		for _, name := range none {
			f := ctx.Flags().Get(name)
			require.NotNil(t, f, name)
			f.SetAssigned(true)
		}
		err := cmd.Run(ctx, []string{})
		return stdout.String() + stderr.String(), err
	}

	_, err := run(map[string]string{"name": "demo", "checksum": strings.Repeat("ab", 32)})
	assert.ErrorContains(t, err, "--checksum is only valid with --package")

	out, err := run(map[string]string{
		"package":  archivePath,
		"checksum": hex.EncodeToString(sum[:]),
	})
	require.NoError(t, err)
	assert.Contains(t, out, "SHA-256 checksum verified")

	_, err = run(map[string]string{
		"package": "https://example.com/plugin.tgz",
	})
	assert.ErrorContains(t, err, "remote --package URL requires --allow-unverified-plugin")

	stdout := new(strings.Builder)
	stderr := new(strings.Builder)
	ctx := cli.NewCommandContext(stdout, stderr)
	cmd := newUpdateCommand()
	ctx.EnterCommand(cmd)
	base := ctx.Flags().Get("source-base")
	base.SetAssigned(true)
	base.SetValue("http://mirror.example/plugins")
	_, err = newManagerWithOptionalSourceBase(ctx)
	assert.ErrorContains(t, err, "source-base: refusing insecure HTTP URL")

	_, err = httpGet("http://example.com/plugin.tgz", time.Second)
	assert.ErrorContains(t, err, "refusing insecure HTTP")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	resp, err := (*Manager)(nil).httpGet(srv.URL, time.Second)
	require.NoError(t, err)
	resp.Body.Close()
	_, err = (&Manager{}).httpGet("https://example.com/a\nb.tgz", time.Second)
	assert.Error(t, err)
}
