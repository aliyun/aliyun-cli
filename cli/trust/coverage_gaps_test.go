package trust

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryHelpersAndInvalidInputs(t *testing.T) {
	client := New(Config{
		SourceBase:    "https://aliyuncli.alicdn.com/plugins-v2",
		DiscoveryURL:  DiscoveryURLCN,
		AllowTestHTTP: true,
	})
	assert.Equal(t,
		"https://aliyuncli.alicdn.com/plugins-v2/trust/timestamp.json",
		client.repositoryTimestampURL(),
	)
	assert.False(t, directRepositoryHost("", DiscoveryURLCN))
	assert.False(t, directRepositoryHost("aliyuncli.alicdn.com", DiscoveryURLCN))
	assert.True(t, directRepositoryHost("cli.aliyun-inc.com", DiscoveryURLCN))
	assert.Equal(t, "", sourceHost("://bad"))
	_, err := sourceOrigin("not a url")
	assert.Error(t, err)
	_, err = sourceOrigin("https://example.com/plugins")
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com"}, appendHost([]string{"example.com"}, ""))
	assert.Equal(t, []string{"example.com", "other.com"}, appendHost([]string{"example.com"}, "other.com"))
	assert.True(t, isLoopback("localhost"))
	assert.True(t, isLoopback("127.0.0.1"))
	assert.False(t, isLoopback("example.com"))
}

func TestDefaultSourceBaseUsesConfiguredHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(pluginsettings.EnvSourceBase, "")
	assert.Equal(t, "https://aliyuncli.alicdn.com/plugins-v2", defaultSourceBase())

	require.NoError(t, pluginsettings.Save(
		filepath.Join(home, ".aliyun"),
		&pluginsettings.PluginSettings{SourceBase: "https://cli.aliyun-inc.com/plugins-v2"},
	))
	assert.Equal(t, "https://cli.aliyun-inc.com/plugins-v2", defaultSourceBase())
}

func TestBootstrapAndSelfSignedRootErrors(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	client := New(Config{BootstrapRoot: []byte("not-json"), Now: func() time.Time { return now }})
	_, err := client.bootstrapRoot(now)
	assert.ErrorContains(t, err, "invalid bootstrap root")

	signer := newTestSigner(t, "root")
	payload := Root{
		Type: "timestamp", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{signer.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root"}, Threshold: 1}},
	}
	client.cfg.BootstrapRoot = signedJSON(t, payload, signer)
	_, err = client.bootstrapRoot(now)
	assert.ErrorContains(t, err, "invalid bootstrap root")

	noRole := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys: []Key{signer.key},
	}
	client.cfg.BootstrapRoot = signedJSON(t, noRole, signer)
	_, err = client.bootstrapRoot(now)
	assert.ErrorContains(t, err, "bootstrap root role is missing")

	_, err = verifySelfSignedRoot([]byte("{"), now, nil)
	assert.Error(t, err)
	_, err = client.parseTrustedRoot([]byte("{"), now, nil)
	assert.Error(t, err)

	expired := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(-time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{signer.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root"}, Threshold: 1}},
	}
	_, err = client.parseTrustedRoot(signedJSON(t, expired, signer), now, nil)
	assert.ErrorContains(t, err, "expired")

	_, err = VerifyRootUpdate(signedJSON(t, expired, signer), expired, now, nil)
	assert.Error(t, err)
}

func TestVerifyHelpersRejectInvalidDocuments(t *testing.T) {
	now := time.Now().UTC()
	root := Root{Authority: "https://api.aliyun.com"}
	_, err := VerifyArtifactKeys([]byte("{}"), root, 1, now, nil)
	assert.ErrorContains(t, err, "artifact-keys role is missing")

	keys := ArtifactKeys{Roles: map[string]Role{}}
	_, _, err = VerifyTyped[Timestamp]([]byte("{}"), keys, "timestamp", "timestamp", now, nil)
	assert.ErrorContains(t, err, "timestamp role is missing")

	assert.ErrorContains(t, VerifyDescriptor([]byte("ab"), ArtifactDescriptor{
		Length: 1, ExpiresAt: now.Add(time.Hour), SHA256: strings.Repeat("0", 64),
	}, now), "length mismatch")
	assert.ErrorContains(t, VerifyDescriptor([]byte("ab"), ArtifactDescriptor{
		Length: 2, ExpiresAt: now.Add(-time.Hour), SHA256: strings.Repeat("0", 64),
	}, now), "expired")
	assert.ErrorContains(t, VerifyDescriptor([]byte("ab"), ArtifactDescriptor{
		Length: 2, ExpiresAt: now.Add(time.Hour), SHA256: strings.Repeat("0", 64),
	}, now), "SHA-256")

	client := New(Config{})
	_, _, err = client.verifyArtifactEnvelope(nil, ArtifactKeys{Roles: map[string]Role{}}, "plugins", PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "plugins role is missing")

	_, _, err = DecodeEnvelope([]byte(`{"schema_version":2,"payload":"e30=","signatures":[]}`))
	assert.ErrorContains(t, err, "unsupported envelope schema")
	_, _, err = DecodeEnvelope([]byte(`{"schema_version":1,"payload":"%%%","signatures":[]}`))
	assert.ErrorContains(t, err, "decode envelope payload")
}

func TestLoadPersistedRootAndClearCachesErrors(t *testing.T) {
	now := time.Now().UTC()
	client := New(Config{Now: func() time.Time { return now }})
	anchor := Root{Version: 1, Authority: "https://api.aliyun.com"}
	_, err := client.loadPersistedRoot(t.TempDir(), anchor, State{RecoveryEpoch: 1}, now, nil)
	assert.Error(t, err)
	missing := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(missing, "recovery-patch.json"), []byte("{}"), 0600))
	_, err = client.loadPersistedRoot(missing, anchor, State{RecoveryEpoch: 1}, now, nil)
	assert.ErrorContains(t, err, "recovery trust anchor is not configured")

	require.NoError(t, clearSourceCaches(t.TempDir(), nil))
	require.NoError(t, clearSourceCaches(filepath.Join(t.TempDir(), "missing", "leaf"), map[string]bool{
		"https://example.com": true,
	}))

	require.NoError(t, atomicWrite(filepath.Join(t.TempDir(), "nested", "ok.json"), []byte("{}"), 0600))
}

func TestFetchStatusAndRecoveryErrorPaths(t *testing.T) {
	f := newTrustFixture(t)
	_, _, err := f.client.fetchStatus(context.Background(), "https://evil.example/x", f.client.cfg.MetadataHosts)
	assert.Error(t, err)
	_, err = f.client.fetch(context.Background(), f.server.URL+"/missing", f.client.cfg.MetadataHosts)
	assert.ErrorContains(t, err, "HTTP 404")

	_, cached, err := f.client.fetchWithLKG(context.Background(), f.server.URL+"/missing", filepath.Join(t.TempDir(), "none"), f.client.cfg.MetadataHosts)
	assert.Error(t, err)
	assert.False(t, cached)

	root := f.root
	state := State{}
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/missing", t.TempDir(), root, &state)
	require.NoError(t, err)

	f.statuses["/recovery-bad"] = http.StatusInternalServerError
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-bad", t.TempDir(), root, &state)
	assert.ErrorContains(t, err, "HTTP 500")

	f.client.cfg.RecoveryKeys = nil
	f.responses["/recovery-ok"] = []byte(`{}`)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), root, &state)
	assert.ErrorContains(t, err, "recovery trust anchor is not configured")
}

func TestFetchArtifactRejectsMissingBootstrapAndExpiredFinalRoot(t *testing.T) {
	f := newTrustFixture(t)
	empty := New(Config{CacheDir: t.TempDir(), SourceBase: f.client.cfg.SourceBase})
	_, err := empty.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "bootstrap root is not configured")

	f.root.ExpiresAt = f.now.Add(-time.Minute)
	f.client.cfg.BootstrapRoot = signedJSON(t, f.root, f.rootSigner)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "is expired")
}

func TestDiscoveryURLRejectsInvalidSourceAndQuery(t *testing.T) {
	client := New(Config{SourceBase: "ftp://x", DiscoveryURL: DiscoveryURLCN})
	_, err := client.discoveryURL()
	assert.ErrorContains(t, err, "invalid plugin source-base")

	client = New(Config{
		SourceBase:    "https://aliyuncli.alicdn.com/plugins-v2",
		DiscoveryURL:  DiscoveryURLCN,
		MetadataHosts: []string{"api.aliyun.com"},
	})
	assert.Error(t, client.validateURL(DiscoveryURLCN+"?other=1", client.cfg.MetadataHosts))
	assert.Error(t, client.validateURL("https://user:pass@api.aliyun.com/x", client.cfg.MetadataHosts))

	var discovery Discovery
	assert.Error(t, json.Unmarshal([]byte("{"), &discovery))
}

func TestVerifyEnvelopeSkipsUnusableSignatures(t *testing.T) {
	now := time.Now().UTC()
	signer := newTestSigner(t, "plugins", PurposePluginIndex)
	payload := []byte(`{"plugins":[]}`)
	envelope := signedBytes(t, payload, signer)
	denied := map[string]bool{"plugins": true}
	_, _, err := VerifyEnvelope(envelope, []Key{signer.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, denied)
	assert.ErrorContains(t, err, "threshold not met")

	wrongPurpose := signer
	wrongPurpose.key.Purposes = []string{"timestamp"}
	_, _, err = VerifyEnvelope(envelope, []Key{wrongPurpose.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "threshold not met")

	expired := signer
	expired.key.NotAfter = now.Add(-time.Minute)
	_, _, err = VerifyEnvelope(envelope, []Key{expired.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "threshold not met")

	badKey := signer
	badKey.key.PublicKey = "not-base64"
	_, _, err = VerifyEnvelope(envelope, []Key{badKey.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "threshold not met")

	_, _, err = VerifyEnvelope(envelope, []Key{signer.key, signer.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "duplicate key id")
	_, _, err = VerifyEnvelope(envelope, []Key{signer.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 0}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "invalid signature threshold")
}

func TestUpdateRootsStopsOnAbsentOrInvalidNextRoot(t *testing.T) {
	f := newTrustFixture(t)
	f.statuses["/roots/2.root.json"] = http.StatusForbidden
	root, err := f.client.updateRoots(context.Background(), f.server.URL+"/roots", t.TempDir(), f.root, f.now, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), root.Version)

	delete(f.statuses, "/roots/2.root.json")
	f.responses["/roots/2.root.json"] = []byte("not-a-root")
	root, err = f.client.updateRoots(context.Background(), f.server.URL+"/roots", t.TempDir(), f.root, f.now, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), root.Version)

	_, err = f.client.updateRoots(context.Background(), "https://evil.example/roots", t.TempDir(), f.root, f.now, nil)
	require.NoError(t, err)
}

func TestFetchArtifactRequiresOfficialOriginOnPublicSource(t *testing.T) {
	f := newTrustFixture(t)
	var discovery Discovery
	require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
	discovery.OfficialArtifactOrigins = nil
	data, err := json.Marshal(discovery)
	require.NoError(t, err)
	f.responses["/discovery"] = data
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "no valid official artifact origin")
}

func TestLoadDiscoveryFallsBackToCachedCopy(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	f.statuses["/discovery"] = http.StatusServiceUnavailable
	delete(f.responses, "/discovery")
	discovery, err := f.client.loadDiscovery(context.Background(), PurposePluginIndex, f.now)
	require.NoError(t, err)
	assert.Equal(t, f.server.URL+"/plugins-v2", discovery.SourceBase)
}

func TestCheckAndApplyRecoveryRejectsInvalidPatch(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery-a")
	f.client.cfg.RecoveryKeys = []Key{recovery.key}
	f.client.cfg.RecoveryRole = Role{KeyIDs: []string{"recovery-a"}, Threshold: 1}
	f.responses["/recovery-invalid"] = signedJSON(t, map[string]any{
		"type": "root", "schema_version": 1, "recovery_epoch": 1,
	}, recovery)
	state := State{}
	_, err := f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-invalid", t.TempDir(), f.root, &state)
	assert.Error(t, err)
}

func TestFetchArtifactReportsMissingDescriptorAndKeyFailure(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposeUpgrade, "plugins")
	assert.ErrorContains(t, err, "timestamp has no")

	f.responses["/keys"] = []byte(`{"schema_version":1,"payload":"e30=","signatures":[]}`)
	f = newTrustFixture(t)
	f.responses["/keys"] = []byte(`{"schema_version":1,"payload":"e30=","signatures":[]}`)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "verify artifact keys")
}

func TestLoadPersistedRootWalksMissingLaterRoot(t *testing.T) {
	now := time.Now().UTC()
	f := newTrustFixture(t)
	_, err := f.client.loadPersistedRoot(t.TempDir(), f.root, State{RootVersion: 2}, now, nil)
	assert.Error(t, err)
}

func TestVerifyTypedRejectsUndecodablePayload(t *testing.T) {
	now := time.Now().UTC()
	signer := newTestSigner(t, "timestamp", "timestamp")
	keys := ArtifactKeys{
		Keys:  []Key{signer.key},
		Roles: map[string]Role{"timestamp": {KeyIDs: []string{"timestamp"}, Threshold: 1}},
	}
	_, _, err := VerifyTyped[Timestamp](signedBytes(t, []byte("not-json"), signer), keys, "timestamp", "timestamp", now, nil)
	assert.Error(t, err)
}

func TestFetchStatusFollowsRedirectAndRejectsDisallowedHost(t *testing.T) {
	f := newTrustFixture(t)
	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redir" {
			http.Redirect(w, r, "https://evil.example/trust", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	_, _, err := f.client.fetchStatus(context.Background(), f.server.URL+"/redir", f.client.cfg.MetadataHosts)
	assert.Error(t, err)
}

func TestFetchArtifactCoversRemainingFailureAndInnerSourcePaths(t *testing.T) {
	t.Run("invalid source-base", func(t *testing.T) {
		f := newTrustFixture(t)
		f.client.cfg.SourceBase = "ftp://x"
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "invalid plugin source-base")
	})

	t.Run("bootstrap fails after discovery", func(t *testing.T) {
		f := newTrustFixture(t)
		f.client.cfg.BootstrapRoot = []byte("not-json")
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "invalid bootstrap root")
	})

	t.Run("disallowed metadata urls", func(t *testing.T) {
		for _, mutate := range []func(*Discovery){
			func(d *Discovery) { d.RootMetadataURI = "https://evil.example/roots" },
			func(d *Discovery) { d.ArtifactKeysURI = "https://evil.example/keys" },
			func(d *Discovery) { d.TimestampURI = "https://evil.example/timestamp" },
		} {
			f := newTrustFixture(t)
			var discovery Discovery
			require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
			mutate(&discovery)
			data, err := json.Marshal(discovery)
			require.NoError(t, err)
			f.responses["/discovery"] = data
			_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
			assert.Error(t, err)
		}
	})

	t.Run("disallowed recovery host", func(t *testing.T) {
		f := newTrustFixture(t)
		f.client.cfg.SourceBase = "https://aliyuncli.alicdn.com/plugins-v2"
		f.client.cfg.RecoveryHosts = []string{"127.0.0.1"}
		var discovery Discovery
		require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
		discovery.SourceBase = "https://aliyuncli.alicdn.com/plugins-v2"
		data, err := json.Marshal(discovery)
		require.NoError(t, err)
		f.responses["/discovery"] = data
		_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.Error(t, err)
	})

	t.Run("persisted root missing", func(t *testing.T) {
		f := newTrustFixture(t)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err)
		cacheDir := filepath.Join(f.cacheDir, digest(f.server.URL), digest(f.server.URL+"/plugins-v2"))
		state := f.client.readState(cacheDir)
		state.RootVersion = 2
		require.NoError(t, atomicWrite(filepath.Join(cacheDir, "trust-state.json"), mustJSON(state), 0600))
		_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "load persisted root")
	})

	t.Run("recovery endpoint 500", func(t *testing.T) {
		f := newTrustFixture(t)
		f.statuses[fixtureRecoveryPath] = http.StatusInternalServerError
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "apply recovery")
	})

	t.Run("keys and timestamp missing", func(t *testing.T) {
		f := newTrustFixture(t)
		delete(f.responses, "/keys")
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "HTTP 404")

		f = newTrustFixture(t)
		delete(f.responses, "/timestamp")
		_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("artifact version rollback", func(t *testing.T) {
		f := newTrustFixture(t)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err)
		index := []byte(`{"plugins":[]}`)
		sum := sha256.Sum256(index)
		f.responses["/index"] = signedBytes(t, index, f.plugin)
		timestamp := Timestamp{
			Type: "timestamp", SchemaVersion: SchemaVersion, Version: 2,
			ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
			Artifacts: map[string]ArtifactDescriptor{
				PurposePluginIndex: {
					URL: f.server.URL + "/index", SHA256: hex.EncodeToString(sum[:]),
					Length: int64(len(index)), Version: 0, ExpiresAt: f.now.Add(time.Hour),
				},
			},
		}
		f.responses["/timestamp"] = signedJSON(t, timestamp, f.timestamp)
		_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "rollback")
	})

	t.Run("invalid descriptor url", func(t *testing.T) {
		f := newTrustFixture(t)
		index := []byte(`{"plugins":[]}`)
		sum := sha256.Sum256(index)
		timestamp := Timestamp{
			Type: "timestamp", SchemaVersion: SchemaVersion, Version: 1,
			ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
			Artifacts: map[string]ArtifactDescriptor{
				PurposePluginIndex: {
					URL: "https://evil.example/index", SHA256: hex.EncodeToString(sum[:]),
					Length: int64(len(index)), Version: 1, ExpiresAt: f.now.Add(time.Hour),
				},
			},
		}
		f.responses["/timestamp"] = signedJSON(t, timestamp, f.timestamp)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.Error(t, err)
	})

	t.Run("missing envelope", func(t *testing.T) {
		f := newTrustFixture(t)
		index := []byte(`{"plugins":[]}`)
		sum := sha256.Sum256(index)
		timestamp := Timestamp{
			Type: "timestamp", SchemaVersion: SchemaVersion, Version: 1,
			ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
			Artifacts: map[string]ArtifactDescriptor{
				PurposePluginIndex: {
					URL: f.server.URL + "/missing-index", SHA256: hex.EncodeToString(sum[:]),
					Length: int64(len(index)), Version: 1, ExpiresAt: f.now.Add(time.Hour),
				},
			},
		}
		f.responses["/timestamp"] = signedJSON(t, timestamp, f.timestamp)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("unsigned envelope", func(t *testing.T) {
		f := newTrustFixture(t)
		f.responses["/index"] = []byte(`{"plugins":[]}`)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.Error(t, err)
	})

	t.Run("inner source rewrites metadata urls", func(t *testing.T) {
		f := newTrustFixture(t)
		parsed, err := url.Parse(f.server.URL)
		require.NoError(t, err)
		f.client.cfg.SourceBase = "http://localhost:" + parsed.Port() + "/plugins-v2"
		f.client.cfg.MetadataHosts = append(f.client.cfg.MetadataHosts, "localhost")
		f.client.cfg.RecoveryHosts = append(f.client.cfg.RecoveryHosts, "localhost")

		var discovery Discovery
		require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
		discovery.SourceBase = "https://aliyuncli.alicdn.com/plugins-v2"
		f.responses["/discovery"], err = json.Marshal(discovery)
		require.NoError(t, err)
		f.responses["/plugins-v2/trust/artifact-keys.json"] = f.responses["/keys"]
		f.responses["/plugins-v2/trust/timestamp.json"] = f.responses["/timestamp"]
		f.statuses["/plugins-v2/trust/roots/2.root.json"] = http.StatusNotFound
		f.statuses[fixtureRecoveryPath] = http.StatusNotFound

		artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err)
		assert.JSONEq(t, `{"plugins":[]}`, string(artifact.Bytes))
	})
}

func TestLoadDiscoveryFallsBackWhenRemoteCopyIsInvalid(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	f.responses["/discovery"] = []byte(`{"schema_version":1}`)
	discovery, err := f.client.loadDiscovery(context.Background(), PurposePluginIndex, f.now)
	require.NoError(t, err)
	assert.Equal(t, f.server.URL+"/plugins-v2", discovery.SourceBase)

	f = newTrustFixture(t)
	f.responses["/discovery"] = []byte(`{"schema_version":1}`)
	_, err = f.client.loadDiscovery(context.Background(), PurposePluginIndex, f.now)
	assert.Error(t, err)
}

func TestDiscoveryURLRejectsBrokenDiscoveryURL(t *testing.T) {
	client := New(Config{
		SourceBase:   "https://aliyuncli.alicdn.com/plugins-v2",
		DiscoveryURL: "://bad",
	})
	_, err := client.discoveryURL()
	assert.ErrorContains(t, err, "invalid discovery URL")
}

func TestCheckAndApplyRecoveryRejectsEpochAndReplacementErrors(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery-a")
	f.client.cfg.RecoveryKeys = []Key{recovery.key}
	f.client.cfg.RecoveryRole = Role{KeyIDs: []string{"recovery-a"}, Threshold: 1}
	replacementSigner := newTestSigner(t, "replacement")
	replacement := f.signerRoot(2, replacementSigner, f.now.Add(time.Hour))
	patch := RecoveryPatch{
		Type: "recovery", SchemaVersion: SchemaVersion, RecoveryEpoch: 1, PatchEpoch: 1,
		NotBefore: f.now.Add(-time.Minute), ExpiresAt: f.now.Add(time.Hour),
		AffectedRootMin: 1, AffectedRootMax: 10,
		NewBootstrapRoot: signedJSON(t, replacement, replacementSigner),
	}
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)

	state := State{RecoveryEpoch: 2}
	_, err := f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "recovery epoch rollback")

	state = State{RecoveryEpoch: 1}
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	require.NoError(t, err)

	state = State{PatchEpoch: 5}
	patch.PatchEpoch = 4
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "invalid, expired, or rolled-back")

	patch.PatchEpoch = 1
	patch.ExpiresAt = f.now.Add(-time.Minute)
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	state = State{}
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "invalid, expired, or rolled-back")

	patch.ExpiresAt = f.now.Add(time.Hour)
	patch.AffectedRootMin, patch.AffectedRootMax = 50, 60
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "invalid, expired, or rolled-back")

	patch.AffectedRootMin, patch.AffectedRootMax = 1, 10
	patch.NewBootstrapRoot = []byte(`{}`)
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "invalid replacement root")

	other := replacement
	other.Authority = "https://other.example"
	patch.NewBootstrapRoot = signedJSON(t, other, replacementSigner)
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "authority mismatch")

	patch.NewBootstrapRoot = signedJSON(t, replacement, replacementSigner)
	patch.ClearOrigins = []string{"http://not-https.example"}
	f.responses["/recovery-ok"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery-ok", t.TempDir(), f.root, &state)
	assert.ErrorContains(t, err, "invalid recovery clear origin")
}

func TestLoadPersistedRootRejectsMismatchedRecoveryAndBrokenChain(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery-a")
	f.client.cfg.RecoveryKeys = []Key{recovery.key}
	f.client.cfg.RecoveryRole = Role{KeyIDs: []string{"recovery-a"}, Threshold: 1}
	replacementSigner := newTestSigner(t, "replacement")
	replacement := f.signerRoot(1, replacementSigner, f.now.Add(time.Hour))
	patch := RecoveryPatch{
		Type: "recovery", SchemaVersion: SchemaVersion, RecoveryEpoch: 1, PatchEpoch: 1,
		NotBefore: f.now.Add(-time.Minute), ExpiresAt: f.now.Add(time.Hour),
		AffectedRootMin: 1, AffectedRootMax: 10,
		NewBootstrapRoot: signedJSON(t, replacement, replacementSigner),
	}
	cache := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(cache, "recovery-patch.json"), []byte("{}"), 0600))
	_, err := f.client.loadPersistedRoot(cache, f.root, State{RecoveryEpoch: 1}, f.now, nil)
	assert.Error(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(cache, "recovery-patch.json"), signedJSON(t, patch, recovery), 0600))
	_, err = f.client.loadPersistedRoot(cache, f.root, State{RecoveryEpoch: 2}, f.now, nil)
	assert.ErrorContains(t, err, "does not match trust state")

	other := replacement
	other.Authority = "https://other.example"
	patch.NewBootstrapRoot = signedJSON(t, other, replacementSigner)
	require.NoError(t, os.WriteFile(filepath.Join(cache, "recovery-patch.json"), signedJSON(t, patch, recovery), 0600))
	_, err = f.client.loadPersistedRoot(cache, f.root, State{RecoveryEpoch: 1}, f.now, nil)
	assert.ErrorContains(t, err, "does not match trust state")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, "2.root.json"), []byte("not-a-root"), 0600))
	_, err = f.client.loadPersistedRoot(broken, f.root, State{RootVersion: 2}, f.now, nil)
	assert.Error(t, err)
}

func TestVerifyRootUpdateAndTransitionErrors(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	oldSigner := newTestSigner(t, "root")
	nextSigner := newTestSigner(t, "root-next")
	old := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{oldSigner.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root"}, Threshold: 1}},
	}
	_, err := verifyRootTransition([]byte("{}"), Root{}, now, nil)
	assert.ErrorContains(t, err, "old root role is missing")

	_, err = verifyRootTransition(signedBytes(t, []byte("not-json"), oldSigner), old, now, nil)
	assert.Error(t, err)

	missingRole := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 2,
		ExpiresAt: now.Add(time.Hour), Authority: old.Authority, Keys: []Key{nextSigner.key},
	}
	_, err = verifyRootTransition(signedJSON(t, missingRole, oldSigner), old, now, nil)
	assert.ErrorContains(t, err, "new root role is missing")

	expiredNext := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 2,
		ExpiresAt: now.Add(-time.Hour), Authority: old.Authority,
		Keys:  []Key{nextSigner.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root-next"}, Threshold: 1}},
	}
	_, err = VerifyRootUpdate(signedJSON(t, expiredNext, oldSigner, nextSigner), old, now, nil)
	assert.ErrorContains(t, err, "invalid or non-contiguous root update")
}

func TestVerifyEnvelopeSkipsUnknownAlgorithmAndTrailingJSON(t *testing.T) {
	now := time.Now().UTC()
	signer := newTestSigner(t, "plugins", PurposePluginIndex)
	envelope := signedBytes(t, []byte(`{"plugins":[]}`), signer)
	var parsed SignedEnvelope
	require.NoError(t, json.Unmarshal(envelope, &parsed))
	parsed.Signatures[0].Algorithm = "rsa"
	data, err := json.Marshal(parsed)
	require.NoError(t, err)
	_, _, err = VerifyEnvelope(data, []Key{signer.key}, Role{KeyIDs: []string{"plugins"}, Threshold: 1}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "threshold not met")

	var value struct {
		Value string `json:"value"`
	}
	assert.Error(t, strictJSON([]byte(`{"value":"ok"} ,`), &value))

	root := Root{
		Authority: "https://api.aliyun.com",
		Roles:     map[string]Role{"artifact-keys": {KeyIDs: []string{"root"}, Threshold: 1}},
		Keys:      []Key{signer.key},
	}
	_, err = VerifyArtifactKeys(signedBytes(t, []byte("not-json"), signer), root, 1, now, nil)
	assert.Error(t, err)

	keys := ArtifactKeys{Roles: map[string]Role{"timestamp": {KeyIDs: []string{"timestamp"}, Threshold: 1}}}
	_, _, err = VerifyTyped[Timestamp]([]byte("{}"), keys, "timestamp", "timestamp", now, nil)
	assert.Error(t, err)
}

func TestAtomicWriteAndFetchStatusLimits(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0600))
	assert.Error(t, atomicWrite(filepath.Join(parent, "child.json"), []byte("{}"), 0600))

	f := newTrustFixture(t)
	f.responses["/huge"] = bytes.Repeat([]byte("a"), (16<<20)+1)
	_, _, err := f.client.fetchStatus(context.Background(), f.server.URL+"/huge", f.client.cfg.MetadataHosts)
	assert.ErrorContains(t, err, "size limit")

	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, f.server.URL+r.URL.Path, http.StatusFound)
	})
	_, _, err = f.client.fetchStatus(context.Background(), f.server.URL+"/loop", f.client.cfg.MetadataHosts)
	assert.Error(t, err)
}

func TestClearSourceCachesSkipsUnrelatedSiblings(t *testing.T) {
	authority := t.TempDir()
	cacheDir := filepath.Join(authority, "current")
	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(authority, "notdir"), []byte("x"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(authority, "empty"), 0700))
	bad := filepath.Join(authority, "bad")
	require.NoError(t, os.MkdirAll(bad, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(bad, sourceBaseFile), []byte("not-a-url"), 0600))
	other := filepath.Join(authority, "other")
	require.NoError(t, os.MkdirAll(other, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(other, sourceBaseFile), []byte("https://other.example/plugins"), 0600))
	require.NoError(t, clearSourceCaches(cacheDir, map[string]bool{"https://keep.example": true}))
	_, err := os.Stat(other)
	require.NoError(t, err)
}

func TestBootstrapRejectsWrongSelfSignature(t *testing.T) {
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	signer := newTestSigner(t, "root")
	attacker := newTestSigner(t, "attacker")
	root := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{signer.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root"}, Threshold: 1}},
	}
	client := New(Config{BootstrapRoot: signedJSON(t, root, attacker), Now: func() time.Time { return now }})
	_, err := client.bootstrapRoot(now)
	assert.ErrorContains(t, err, "self-signature")

	_, err = verifySelfSignedRoot(signedJSON(t, Root{
		Type: "timestamp", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{signer.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root"}, Threshold: 1}},
	}, signer), now, nil)
	assert.ErrorContains(t, err, "invalid trusted root")
}
