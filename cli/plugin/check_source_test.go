package plugin

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/stretchr/testify/require"
)

type checkIndexSource struct {
	offline bool
	calls   int
}

type checkIndexTransport func(*http.Request) (*http.Response, error)

func (f checkIndexTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Serve signed metadata through the real trust verifier without any network.
// The fixture root is confined to this test and restored before the next test.
func newCheckIndexSource(t *testing.T, payload []byte) *checkIndexSource {
	t.Helper()
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	key := trust.Key{KeyID: "fixture", Algorithm: trust.AlgorithmEd25519,
		PublicKey: base64.StdEncoding.EncodeToString(pub), Purposes: []string{trust.PurposePluginIndex, "timestamp"}}
	role := trust.Role{KeyIDs: []string{key.KeyID}, Threshold: 1}
	sign := func(value any) []byte {
		body, err := json.Marshal(value)
		require.NoError(t, err)
		if raw, ok := value.(json.RawMessage); ok {
			body = raw
		}
		out, err := json.Marshal(trust.SignedEnvelope{SchemaVersion: 1, Payload: base64.StdEncoding.EncodeToString(body),
			Signatures: []trust.Signature{{KeyID: key.KeyID, Algorithm: key.Algorithm,
				Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, body))}}})
		require.NoError(t, err)
		return out
	}
	const authority = "https://api.aliyun.com"
	const source = "https://aliyuncli.alicdn.com/plugins-v2"
	expires := time.Now().UTC().Add(time.Hour)
	root := trust.Root{Type: "root", SchemaVersion: 1, Version: 1, Authority: authority,
		ExpiresAt: expires, Keys: []trust.Key{key}, Roles: map[string]trust.Role{"root": role, "artifact-keys": role}}
	keys := trust.ArtifactKeys{Type: "artifact-keys", SchemaVersion: 1, Version: 1, Authority: authority,
		ExpiresAt: expires, Keys: []trust.Key{key}, Roles: map[string]trust.Role{"plugins": role, "timestamp": role}}
	sum := sha256.Sum256(payload)
	timestamp := trust.Timestamp{Type: "timestamp", SchemaVersion: 1, Version: 1, Authority: authority, ExpiresAt: expires,
		Artifacts: map[string]trust.ArtifactDescriptor{trust.PurposePluginIndex: {
			URL: source + "/plugin_pkg_index.json", SHA256: hex.EncodeToString(sum[:]), Length: int64(len(payload)), Version: 1, ExpiresAt: expires,
		}}}
	discovery, err := json.Marshal(trust.Discovery{SchemaVersion: 1, Authority: authority, SourceBase: source,
		RootMetadataURI: source + "/trust/roots/", ArtifactKeysURI: source + "/trust/artifact-keys.json",
		TimestampURI: source + "/trust/timestamp.json", OfficialArtifactOrigins: []string{"https://aliyuncli.alicdn.com"},
		SupportedSignatureAlgorithms: []string{trust.AlgorithmEd25519}, SupportedArtifactTypes: []string{trust.PurposePluginIndex},
		CacheMaxAgeSeconds: 3600, ExpiresAt: expires})
	require.NoError(t, err)
	responses := map[string][]byte{
		"/.well-known/aliyun-cli-trust.json":   discovery,
		"/plugins-v2/trust/artifact-keys.json": sign(keys),
		"/plugins-v2/trust/timestamp.json":     sign(timestamp),
		"/plugins-v2/plugin_pkg_index.json":    sign(json.RawMessage(payload)),
	}
	previousRoot, previousTransport := trust.BuiltinRootEnvelope, http.DefaultTransport
	trust.BuiltinRootEnvelope = string(sign(root))
	f := &checkIndexSource{}
	http.DefaultTransport = checkIndexTransport(func(r *http.Request) (*http.Response, error) {
		f.calls++
		if f.offline && responses[r.URL.Path] != nil {
			return nil, errors.New("offline fixture")
		}
		status, body := http.StatusOK, responses[r.URL.Path]
		if body == nil {
			status = http.StatusNotFound
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	t.Cleanup(func() { trust.BuiltinRootEnvelope, http.DefaultTransport = previousRoot, previousTransport })
	return f
}

func TestCheckUpdatesVerifiedSourceAndCachedRefreshFailure(t *testing.T) {
	index := Index{Plugins: []PluginInfo{{Name: "example", Versions: map[string]VersionInfo{
		"1.1.0": {Platforms: map[string]PlatformInfo{PluginPlatformAny: {}}},
	}}}}
	payload, err := json.Marshal(index)
	require.NoError(t, err)
	source := newCheckIndexSource(t, payload)
	dir, root := t.TempDir(), t.TempDir()
	scope := UpdateScope{Root: root, SourceBase: "https://aliyuncli.alicdn.com/plugins-v2", Site: "cn"}
	report, err := CheckUpdates(context.Background(), dir, scope)
	require.NoError(t, err)
	require.Empty(t, report.Plugins)
	require.True(t, report.Fresh)
	require.Zero(t, source.calls, "no index is needed when no plugins are installed")
	manifest := []byte(`{"plugins":{"example":{"name":"example","version":"1.0.0"}}}`)
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), manifest, 0600))
	report, err = CheckUpdates(context.Background(), dir, scope)
	require.NoError(t, err)
	require.Equal(t, "complete", report.Status)
	require.True(t, report.Fresh)
	require.True(t, report.Proof.Valid(time.Now()))
	require.Len(t, report.Plugins, 1)
	require.True(t, report.Plugins[0].UpdateAvailable)
	require.Equal(t, "1.1.0", report.Plugins[0].Latest)
	source.offline = true
	cached, err := CheckUpdates(context.Background(), dir, scope)
	require.ErrorContains(t, err, "using valid verified cache")
	require.Equal(t, "cached", cached.Status)
	require.False(t, cached.Fresh)
	require.Equal(t, report.Plugins, cached.Plugins)
	require.True(t, cached.Proof.Valid(time.Now()))
	after, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	require.NoError(t, err)
	require.Equal(t, manifest, after, "checking must not install or rewrite plugin records")
}

func TestCheckUpdatesRejectsUnreadableOrInvalidMetadata(t *testing.T) {
	for _, scenario := range []string{"manifest", "offline", "invalid_index"} {
		t.Run(scenario, func(t *testing.T) {
			source := newCheckIndexSource(t, []byte(`{"unexpected":true}`))
			dir, root := t.TempDir(), t.TempDir()
			manifest := `{"plugins":{"example":{"version":"1.0.0"}}}`
			if scenario == "manifest" {
				manifest = "broken"
			}
			source.offline = scenario == "offline"
			require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte(manifest), 0600))
			report, err := CheckUpdates(context.Background(), dir, UpdateScope{Root: root, SourceBase: "https://aliyuncli.alicdn.com/plugins-v2"})
			require.Error(t, err)
			require.Equal(t, "failed", report.Status)
			require.Empty(t, report.Plugins)
			switch scenario {
			case "manifest":
				require.ErrorContains(t, err, "manifest is unreadable")
				require.Zero(t, source.calls)
			case "offline":
				require.ErrorContains(t, err, "index check failed")
			case "invalid_index":
				require.ErrorContains(t, err, "verified plugin index is invalid")
			}
		})
	}
}

func TestUpdateScopeUsesConfiguredSourceAndEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(setTestHomeDir(t, home))
	t.Setenv(pluginsettings.EnvSourceBase, "")
	t.Setenv(pluginsettings.EnvSite, "")
	dir := filepath.Join(home, ".aliyun")
	scope, err := ResolveUpdateScope(dir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "plugins"), scope.Root)
	require.Equal(t, "https://aliyuncli.alicdn.com/plugins-v2", scope.SourceBase)
	require.Equal(t, "cn", scope.Site)
	require.NoError(t, pluginsettings.Save(dir, &pluginsettings.PluginSettings{SourceBase: "https://example.test/plugins/", Site: "intl"}))
	scope, err = ResolveUpdateScope(dir)
	require.NoError(t, err)
	require.Equal(t, "https://example.test/plugins", scope.SourceBase)
	require.Equal(t, "intl", scope.Site)
	t.Setenv(EnvPluginsDir, t.TempDir())
	t.Setenv(pluginsettings.EnvSourceBase, "https://override.test/plugins/")
	scope, err = ResolveUpdateScope(dir)
	require.NoError(t, err)
	require.Equal(t, os.Getenv(EnvPluginsDir), scope.Root)
	require.Equal(t, "https://override.test/plugins", scope.SourceBase)
	require.NoError(t, os.Remove(pluginsettings.GetConfigFilePath(dir)))
	require.NoError(t, os.Mkdir(pluginsettings.GetConfigFilePath(dir), 0700))
	_, err = ResolveUpdateScope(dir)
	require.Error(t, err)
}

func TestCompareUpdatesHandlesUncomparableAndPrereleaseVersions(t *testing.T) {
	for _, tc := range []struct{ installed, latest, minimum, status string }{
		{"1.0.0", "2.0.0-beta.1", "", "no_stable_release"},
		{"local-build", "2.0.0", "", "unknown_version"},
		{"1.0.0", "1.0.0", "3.6.0", "up_to_date"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			local := &LocalManifest{Plugins: map[string]LocalPlugin{"example": {Version: tc.installed, MinCliVersion: tc.minimum}}}
			index := &Index{Plugins: []PluginInfo{{Name: "example", Versions: map[string]VersionInfo{tc.latest: {
				Platforms: map[string]PlatformInfo{PluginPlatformAny: {}},
			}}}}}
			report := CompareUpdates(local, index, "3.5.3")
			require.Equal(t, tc.status, report.Plugins[0].Status)
			require.False(t, report.Plugins[0].UpdateAvailable)
			require.Equal(t, tc.minimum != "", report.Plugins[0].RequiresCLIUpgrade)
		})
	}
	name := "plugin with 'quotes'"
	local := &LocalManifest{Plugins: map[string]LocalPlugin{name: {Version: "1.0.0"}}}
	index := &Index{Plugins: []PluginInfo{{Name: name, Versions: map[string]VersionInfo{"2.0.0": {Platforms: map[string]PlatformInfo{PluginPlatformAny: {}}}}}}}
	report := CompareUpdates(local, index, "3.5.3")
	require.Equal(t, `aliyun plugin update --name 'plugin with '"'"'quotes'"'"''`, report.Plugins[0].Command)
}
