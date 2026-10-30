package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePluginArtifactFetcher struct {
	artifact *trust.VerifiedArtifact
	err      error
	calls    int
}

func (f *fakePluginArtifactFetcher) FetchArtifact(_ context.Context, purpose, role string) (*trust.VerifiedArtifact, error) {
	f.calls++
	if purpose != trust.PurposePluginIndex || role != "plugins" {
		return nil, fmt.Errorf("unexpected trust request %s/%s", purpose, role)
	}
	return f.artifact, f.err
}

func usePluginTrustIndex(t *testing.T, index *Index) *fakePluginArtifactFetcher {
	t.Helper()
	payload, err := json.Marshal(index)
	require.NoError(t, err)
	fetcher := &fakePluginArtifactFetcher{artifact: &trust.VerifiedArtifact{
		Bytes: payload,
		Descriptor: trust.ArtifactDescriptor{
			Version: 42,
		},
		KeyIDs:   []string{"plugins-2026"},
		SignedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}}
	original := newArtifactTrustClient
	newArtifactTrustClient = func(string, string) artifactFetcher { return fetcher }
	t.Cleanup(func() { newArtifactTrustClient = original })
	return fetcher
}

func TestOfficialIndexUsesTrustClientWithoutUnsignedFallback(t *testing.T) {
	index := &Index{Plugins: []PluginInfo{{Name: "aliyun-cli-fc"}}}
	fetcher := usePluginTrustIndex(t, index)
	manager := &Manager{rootDir: t.TempDir(), trustClient: fetcher}

	got, err := manager.GetIndex()
	require.NoError(t, err)
	assert.Equal(t, "aliyun-cli-fc", got.Plugins[0].Name)
	assert.Equal(t, 1, fetcher.calls)
	require.NotNil(t, manager.indexTrustVerification)
	assert.Equal(t, uint64(42), manager.indexTrustVerification.IndexVersion)

	fetcher.err = fmt.Errorf("bad signature")
	_, err = manager.GetIndex()
	assert.ErrorContains(t, err, "bad signature")
	assert.Nil(t, manager.indexTrustVerification)
}

func TestUnsignedFallbackDoesNotInheritEarlierVerification(t *testing.T) {
	index := &Index{Plugins: []PluginInfo{{Name: "aliyun-cli-fc"}}}
	fetcher := usePluginTrustIndex(t, index)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"plugins":[{"name":"aliyun-cli-fc"}]}`))
	}))
	t.Cleanup(server.Close)
	manager := &Manager{
		rootDir:                    t.TempDir(),
		trustClient:                fetcher,
		sourceBase:                 server.URL,
		skipSignatureVerify:        true,
		skipPluginIndexCacheForCLI: true,
		httpTransport:              server.Client().Transport,
	}

	_, err := manager.GetIndex()
	require.NoError(t, err)
	require.NotNil(t, manager.indexTrustVerification)

	fetcher.err = fmt.Errorf("bad signature")
	_, err = manager.GetIndex()
	require.NoError(t, err)
	assert.Nil(t, manager.indexTrustVerification)
}

func TestSourceBaseReconfiguresRepositoryTrustClient(t *testing.T) {
	var capturedSource string
	fetcher := &fakePluginArtifactFetcher{err: fmt.Errorf("not configured")}
	original := newArtifactTrustClient
	newArtifactTrustClient = func(_ string, sourceBase string) artifactFetcher {
		capturedSource = sourceBase
		return fetcher
	}
	t.Cleanup(func() { newArtifactTrustClient = original })

	manager := &Manager{rootDir: t.TempDir(), trustCacheDir: t.TempDir()}
	require.NoError(t, manager.ApplySourceBaseOverride(
		"https://cli-inner.example.com/pre/plugins-v2/",
	))
	assert.Equal(
		t,
		"https://cli-inner.example.com/pre/plugins-v2",
		capturedSource,
	)

	_, err := manager.GetIndex()
	assert.ErrorContains(t, err, "failed to verify plugin index")
}

func TestPluginAuditFieldsRoundTripAndDisplay(t *testing.T) {
	verification := &TrustVerification{
		KeyIDs: []string{"key-a", "key-b"}, IndexVersion: 9,
		Checksum: "abc", VerifiedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-fc": {
			Name: "aliyun-cli-fc", Version: "1.0.0", TrustVerification: verification,
		},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	var decoded LocalManifest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, verification, decoded.Plugins["aliyun-cli-fc"].TrustVerification)
}

func TestDisplayInstalledPluginDetailsShowsUnverified(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	require.NoError(t, displayInstalledPluginDetails(ctx, "aliyun-cli-fc", &LocalPlugin{
		Name: "aliyun-cli-fc", Version: "1.0.0",
	}))
	assert.Contains(t, stdout.String(), "Trust:\tunverified")
}

func TestUnsignedFallbackHelpersAndVerifiedIndexErrors(t *testing.T) {
	t.Cleanup(func() { AllowUnsignedFallbackForTest(false) })
	AllowUnsignedFallbackForTest(true)
	home := t.TempDir()
	cleanup := setTestHomeDir(t, home)
	t.Cleanup(cleanup)
	mgr, err := NewManager()
	require.NoError(t, err)
	assert.True(t, mgr.skipSignatureVerify)

	other := &Manager{}
	other.AllowUnsignedFallback()
	assert.True(t, other.skipSignatureVerify)

	fetcher := &fakePluginArtifactFetcher{artifact: &trust.VerifiedArtifact{Bytes: []byte("not-json")}}
	_, err = (&Manager{rootDir: t.TempDir(), trustClient: fetcher}).GetIndex()
	assert.ErrorContains(t, err, "failed to decode verified plugin index")
}

func TestVerifiedInstallRecordsTrustChecksumAndLength(t *testing.T) {
	body := []byte("plugin-archive")
	sum := sha256.Sum256(body)
	checksum := hex.EncodeToString(sum[:])
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	manager := &Manager{
		rootDir:       t.TempDir(),
		httpTransport: server.Client().Transport,
		indexTrustVerification: &TrustVerification{
			KeyIDs: []string{"plugins-2026"}, IndexVersion: 7,
		},
	}
	plugin := &PluginInfo{
		Name: "aliyun-cli-fc",
		Versions: map[string]VersionInfo{
			"1.0.0": {Platforms: map[string]PlatformInfo{
				PluginPlatformAny: {URL: server.URL + "/pkg.tgz", Checksum: checksum, Length: int64(len(body) + 1)},
			}},
		},
	}
	err := manager.installPlugin(ctx, plugin, "1.0.0", false, false)
	assert.ErrorContains(t, err, "plugin length verification failed")
	require.NotNil(t, manager.indexTrustVerification)
	assert.Equal(t, checksum, manager.indexTrustVerification.Checksum)

	plugin.Versions["1.0.0"].Platforms[PluginPlatformAny] = PlatformInfo{
		URL: server.URL + "/pkg.tgz", Checksum: checksum, Length: int64(len(body)),
	}
	archive, err := manager.downloadAndVerifyPlugin(ctx, &PlatformInfo{
		URL: server.URL + "/pkg.tgz", Checksum: checksum, Length: int64(len(body)),
	}, "aliyun-cli-fc", "1.0.0")
	require.NoError(t, err)
	assert.NotEmpty(t, archive)
}

func TestSavePluginToManifestCopiesTrustVerification(t *testing.T) {
	manager := &Manager{
		rootDir: t.TempDir(),
		indexTrustVerification: &TrustVerification{
			KeyIDs: []string{"plugins-2026", "extra"}, IndexVersion: 11, Checksum: "abc",
		},
	}
	require.NoError(t, manager.savePluginToManifest(
		"aliyun-cli-fc",
		"1.0.0",
		manager.rootDir,
		&PluginManifest{Name: "aliyun-cli-fc", Command: "fc"},
	))
	local, err := manager.GetLocalManifest()
	require.NoError(t, err)
	got := local.Plugins["aliyun-cli-fc"].TrustVerification
	require.NotNil(t, got)
	assert.Equal(t, []string{"plugins-2026", "extra"}, got.KeyIDs)
	assert.Equal(t, uint64(11), got.IndexVersion)
	manager.indexTrustVerification.KeyIDs[0] = "mutated"
	assert.Equal(t, "plugins-2026", got.KeyIDs[0])
}
