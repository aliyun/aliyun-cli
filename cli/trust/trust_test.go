package trust

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSigner struct {
	key  Key
	priv ed25519.PrivateKey
}

func newTestSigner(t *testing.T, id string, purposes ...string) testSigner {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return testSigner{
		key: Key{
			KeyID:     id,
			Algorithm: AlgorithmEd25519,
			PublicKey: base64.StdEncoding.EncodeToString(pub),
			Purposes:  purposes,
		},
		priv: priv,
	}
}

func signedBytes(t *testing.T, payload []byte, signers ...testSigner) []byte {
	t.Helper()
	envelope := SignedEnvelope{
		SchemaVersion: SchemaVersion,
		Payload:       base64.StdEncoding.EncodeToString(payload),
	}
	for _, signer := range signers {
		envelope.Signatures = append(envelope.Signatures, Signature{
			KeyID:     signer.key.KeyID,
			Algorithm: AlgorithmEd25519,
			Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(signer.priv, payload)),
		})
	}
	data, err := json.Marshal(envelope)
	require.NoError(t, err)
	return data
}

func signedJSON(t *testing.T, value any, signers ...testSigner) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return signedBytes(t, payload, signers...)
}

func TestStrictJSONRequiresEOF(t *testing.T) {
	var value struct {
		Value string `json:"value"`
	}
	require.NoError(t, strictJSON([]byte(`{"value":"ok"}   `), &value))
	assert.Error(t, strictJSON([]byte(`{"value":"ok"} {}`), &value))
	assert.Error(t, strictJSON([]byte(`{"value":"ok","extra":true}`), &value))
	require.NoError(t, DecodeStrictJSON([]byte(`{"value":"ok"}`), &value))
}

func TestSharedPythonProtocolVector(t *testing.T) {
	publicKey, err := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	require.NoError(t, err)
	envelope := []byte(`{"schema_version":1,"payload":"eyJleHBpcmVzX2F0IjoiMjAzMC0wMS0wMVQwMDowMDowMFoiLCJzY2hlbWFfdmVyc2lvbiI6MSwidHlwZSI6ImFydGlmYWN0LWtleXMiLCJ2ZXJzaW9uIjoxfQ==","signatures":[{"key_id":"rfc8032-test-1","algorithm":"Ed25519","signature":"XqINGqKObgrVIbNaGQYybyTdLlj0VcjlqGt8QlcuCKzD4/Rin7y6D/tMTgxKpxU2HZyrhilvYpv1H6EjJRlUBQ=="}]}`)
	payload, ids, err := VerifyEnvelope(
		envelope,
		[]Key{{
			KeyID:     "rfc8032-test-1",
			Algorithm: AlgorithmEd25519,
			PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		}},
		Role{KeyIDs: []string{"rfc8032-test-1"}, Threshold: 1},
		"",
		time.Now().UTC(),
		nil,
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"expires_at":"2030-01-01T00:00:00Z","schema_version":1,"type":"artifact-keys","version":1}`, string(payload))
	assert.Equal(t, []string{"rfc8032-test-1"}, ids)
}

func TestEnvelopePreservesRawPayloadAndThreshold(t *testing.T) {
	now := time.Now().UTC()
	a := newTestSigner(t, "a", PurposePluginIndex)
	b := newTestSigner(t, "b", PurposePluginIndex)
	payload := []byte("{\n  \"plugins\": []\n}\n")
	envelope := signedBytes(t, payload, a, b)

	got, ids, err := VerifyEnvelope(envelope, []Key{a.key, b.key}, Role{
		KeyIDs: []string{"a", "b"}, Threshold: 2,
	}, PurposePluginIndex, now, nil)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
	assert.Equal(t, []string{"a", "b"}, ids)

	_, _, err = VerifyEnvelope(signedBytes(t, payload, a), []Key{a.key, b.key}, Role{
		KeyIDs: []string{"a", "b"}, Threshold: 2,
	}, PurposePluginIndex, now, nil)
	assert.ErrorContains(t, err, "threshold not met")
}

func TestVerifyRootUpdateRequiresOldAndNewThresholds(t *testing.T) {
	now := time.Now().UTC()
	oldA := newTestSigner(t, "old-a")
	oldB := newTestSigner(t, "old-b")
	newA := newTestSigner(t, "new-a")
	newB := newTestSigner(t, "new-b")
	old := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: now.Add(time.Hour), Authority: "https://api.aliyun.com",
		Keys:  []Key{oldA.key, oldB.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"old-a", "old-b"}, Threshold: 2}},
	}
	next := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 2,
		ExpiresAt: now.Add(time.Hour), Authority: old.Authority,
		Keys:  []Key{newA.key, newB.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"new-a", "new-b"}, Threshold: 2}},
	}
	allSigned := signedJSON(t, next, oldA, oldB, newA, newB)
	verified, err := VerifyRootUpdate(allSigned, old, now, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), verified.Version)

	_, err = VerifyRootUpdate(signedJSON(t, next, oldA, oldB, newA), old, now, nil)
	assert.ErrorContains(t, err, "new root approval")
}

type trustFixture struct {
	t          *testing.T
	now        time.Time
	server     *httptest.Server
	responses  map[string][]byte
	statuses   map[string]int
	rootSigner testSigner
	plugin     testSigner
	timestamp  testSigner
	root       Root
	client     *Client
	cacheDir   string
}

func newTrustFixture(t *testing.T) *trustFixture {
	t.Helper()
	f := &trustFixture{
		t:         t,
		now:       time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
		responses: make(map[string][]byte),
		statuses:  make(map[string]int),
		cacheDir:  t.TempDir(),
	}
	f.rootSigner = newTestSigner(t, "root")
	f.plugin = newTestSigner(t, "plugin-a", PurposePluginIndex)
	f.timestamp = newTestSigner(t, "timestamp", "timestamp")
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status := f.statuses[r.URL.Path]; status != 0 {
			w.WriteHeader(status)
			return
		}
		data, ok := f.responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(f.server.Close)
	f.root = Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 1,
		ExpiresAt: f.now.Add(24 * time.Hour), Authority: f.server.URL,
		Keys: []Key{f.rootSigner.key},
		Roles: map[string]Role{
			"root":          {KeyIDs: []string{"root"}, Threshold: 1},
			"artifact-keys": {KeyIDs: []string{"root"}, Threshold: 1},
		},
	}
	f.responses["/roots/2.root.json"] = nil
	f.statuses["/roots/2.root.json"] = http.StatusNotFound
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	f.client = New(Config{
		CacheDir:      f.cacheDir,
		DiscoveryURL:  f.server.URL + "/discovery",
		SourceBase:    f.server.URL + "/plugins-v2",
		HTTPClient:    f.server.Client(),
		Now:           func() time.Time { return f.now },
		MetadataHosts: []string{"127.0.0.1"},
		RecoveryHosts: []string{"127.0.0.1"},
		BootstrapRoot: signedJSON(t, f.root, f.rootSigner),
		AllowTestHTTP: true,
	})
	return f
}

func (f *trustFixture) publish(index []byte, keysVersion, artifactVersion uint64, pluginSigner testSigner) {
	f.t.Helper()
	f.responses["/index"] = signedBytes(f.t, index, pluginSigner)
	keys := ArtifactKeys{
		Type: "artifact-keys", SchemaVersion: SchemaVersion, Version: keysVersion,
		ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
		Keys: []Key{pluginSigner.key, f.timestamp.key},
		Roles: map[string]Role{
			"plugins":   {KeyIDs: []string{pluginSigner.key.KeyID}, Threshold: 1},
			"timestamp": {KeyIDs: []string{f.timestamp.key.KeyID}, Threshold: 1},
		},
	}
	f.responses["/keys"] = signedJSON(f.t, keys, f.rootSigner)
	sum := sha256.Sum256(index)
	timestamp := Timestamp{
		Type: "timestamp", SchemaVersion: SchemaVersion, Version: artifactVersion,
		ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
		Artifacts: map[string]ArtifactDescriptor{
			PurposePluginIndex: {
				URL: f.server.URL + "/index", SHA256: hex.EncodeToString(sum[:]),
				Length: int64(len(index)), Version: artifactVersion, ExpiresAt: f.now.Add(time.Hour),
			},
		},
	}
	f.responses["/timestamp"] = signedJSON(f.t, timestamp, f.timestamp)
	discovery := Discovery{
		SchemaVersion: SchemaVersion, Authority: f.server.URL,
		SourceBase:                   f.server.URL + "/plugins-v2",
		RootMetadataURI:              f.server.URL + "/roots",
		ArtifactKeysURI:              f.server.URL + "/keys",
		TimestampURI:                 f.server.URL + "/timestamp",
		OfficialArtifactOrigins:      []string{f.server.URL},
		SupportedSignatureAlgorithms: []string{AlgorithmEd25519},
		SupportedArtifactTypes:       []string{PurposePluginIndex, PurposeUpgrade},
		CacheMaxAgeSeconds:           3600,
		ExpiresAt:                    f.now.Add(time.Hour),
	}
	data, err := json.Marshal(discovery)
	require.NoError(f.t, err)
	f.responses["/discovery"] = data
}

func TestClientFetchArtifactAndKeyRotation(t *testing.T) {
	f := newTrustFixture(t)
	first, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	assert.JSONEq(t, `{"plugins":[]}`, string(first.Bytes))
	assert.Equal(t, []string{"plugin-a"}, first.KeyIDs)

	rotated := newTestSigner(t, "plugin-b", PurposePluginIndex)
	f.publish([]byte(`{"plugins":[{"name":"next"}]}`), 2, 2, rotated)
	second, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), second.Descriptor.Version)
	assert.Equal(t, []string{"plugin-b"}, second.KeyIDs)
}

func TestClientAllowsDiscoveryMirrorWithSharedSignedMetadata(t *testing.T) {
	f := newTrustFixture(t)
	var discovery Discovery
	require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
	discovery.Authority = "https://api.alibabacloud.com"
	data, err := json.Marshal(discovery)
	require.NoError(t, err)
	f.responses["/discovery"] = data

	artifact, err := f.client.FetchArtifact(
		context.Background(), PurposePluginIndex, "plugins",
	)

	require.NoError(t, err)
	assert.JSONEq(t, `{"plugins":[]}`, string(artifact.Bytes))
}

func TestClientBindsDiscoveryToConfiguredSourceBase(t *testing.T) {
	f := newTrustFixture(t)
	discoveryURL, err := f.client.discoveryURL()
	require.NoError(t, err)
	parsed, err := url.Parse(discoveryURL)
	require.NoError(t, err)
	assert.Equal(t, f.server.URL+"/plugins-v2", parsed.Query().Get("source_base"))

	var discovery Discovery
	require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
	discovery.SourceBase = f.server.URL + "/other"
	f.responses["/discovery"], err = json.Marshal(discovery)
	require.NoError(t, err)

	_, err = f.client.FetchArtifact(
		context.Background(), PurposePluginIndex, "plugins",
	)
	assert.ErrorContains(t, err, "invalid or expired discovery")
}

func TestClientUsesUnexpiredDiscoveryAndIndexLKGOnNetworkFailure(t *testing.T) {
	f := newTrustFixture(t)
	first, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)

	f.statuses["/discovery"] = http.StatusServiceUnavailable
	f.statuses["/keys"] = http.StatusServiceUnavailable
	f.statuses["/timestamp"] = http.StatusServiceUnavailable
	f.statuses["/index"] = http.StatusServiceUnavailable
	second, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	assert.Equal(t, first.Bytes, second.Bytes)
}

func TestPersistedRootMustHaveAContinuousTrustedChain(t *testing.T) {
	f := newTrustFixture(t)
	nextSigner := newTestSigner(t, "root-next")
	next := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 2,
		ExpiresAt: f.now.Add(time.Hour), Authority: f.root.Authority,
		Keys:  []Key{nextSigner.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root-next"}, Threshold: 1}},
	}
	cache := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(cache, "2.root.json"),
		signedJSON(t, next, f.rootSigner, nextSigner),
		0600,
	))
	loaded, err := f.client.loadPersistedRoot(
		cache, f.root, State{RootVersion: 2}, f.now, nil,
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), loaded.Version)

	require.NoError(t, os.Remove(filepath.Join(cache, "2.root.json")))
	jump := next
	jump.Version = 3
	require.NoError(t, os.WriteFile(
		filepath.Join(cache, "3.root.json"),
		signedJSON(t, jump, nextSigner),
		0600,
	))
	_, err = f.client.loadPersistedRoot(
		cache, f.root, State{RootVersion: 3}, f.now, nil,
	)
	assert.Error(t, err)
}

func TestUpdateRootsAcceptsOnlyTheNextDualSignedVersion(t *testing.T) {
	f := newTrustFixture(t)
	nextSigner := newTestSigner(t, "root-next")
	next := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 2,
		ExpiresAt: f.now.Add(time.Hour), Authority: f.root.Authority,
		Keys:  []Key{nextSigner.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"root-next"}, Threshold: 1}},
	}
	f.responses["/roots/2.root.json"] = signedJSON(t, next, f.rootSigner, nextSigner)
	delete(f.statuses, "/roots/2.root.json")
	f.statuses["/roots/3.root.json"] = http.StatusNotFound

	cache := t.TempDir()
	got, err := f.client.updateRoots(
		context.Background(), f.server.URL+"/roots", cache, f.root, f.now, nil,
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), got.Version)
	_, err = os.Stat(filepath.Join(cache, "2.root.json"))
	require.NoError(t, err)

	f.responses["/roots/2.root.json"] = []byte(`{"tampered":true}`)
	got, err = f.client.updateRoots(
		context.Background(), f.server.URL+"/roots", t.TempDir(), f.root, f.now, nil,
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), got.Version)
}

func TestTrustURLHelpersRejectUnsafeDestinations(t *testing.T) {
	client := New(Config{AllowTestHTTP: false})
	assert.True(t, hostAllowed("API.ALIYUN.COM.", []string{"api.aliyun.com"}))
	assert.True(t, hostAllowed("api.aliyun.com", []string{"https://api.aliyun.com"}))
	assert.False(t, hostAllowed("evil.example", []string{"api.aliyun.com"}))
	assert.Error(t, client.validateURL("http://api.aliyun.com/keys", []string{"api.aliyun.com"}))
	assert.Error(t, client.validateURL("https://127.0.0.1/keys", []string{"127.0.0.1"}))
	assert.Error(t, client.validateURL("https://user@api.aliyun.com/keys", []string{"api.aliyun.com"}))
	_, err := checkedOrigin("https://api.aliyun.com/path", false)
	assert.Error(t, err)
}

func TestBootstrapFailsClosedWithoutEmbeddedMaterial(t *testing.T) {
	client := New(Config{
		CacheDir:     t.TempDir(),
		DiscoveryURL: "https://api.aliyun.com/.well-known/aliyun-cli-trust.json",
	})
	_, err := client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "bootstrap root is not configured")
}

func TestClientRejectsTamperingAndFakeDiscoveryKeys(t *testing.T) {
	t.Run("tampered signed index", func(t *testing.T) {
		f := newTrustFixture(t)
		f.responses["/index"] = signedBytes(t, []byte(`{"plugins":["tampered"]}`), f.plugin)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mismatch")
	})

	t.Run("fake keys from discovery target", func(t *testing.T) {
		f := newTrustFixture(t)
		attacker := newTestSigner(t, "attacker")
		f.responses["/keys"] = signedJSON(t, ArtifactKeys{
			Type: "artifact-keys", SchemaVersion: SchemaVersion, Version: 9,
			ExpiresAt: f.now.Add(time.Hour), Authority: f.server.URL,
		}, attacker)
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		assert.ErrorContains(t, err, "signature threshold not met")
	})
}

func TestInvalidRemoteMetadataDoesNotReplaceLKGAndExpiredLKGFails(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	cacheDir := filepath.Join(
		f.cacheDir,
		digest(f.server.URL),
		digest(f.server.URL+"/plugins-v2"),
	)
	keysPath := filepath.Join(cacheDir, "artifact-keys.json")
	before, err := os.ReadFile(keysPath)
	require.NoError(t, err)

	f.responses["/keys"] = []byte(`{"not":"signed"}`)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	after, err := os.ReadFile(keysPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)

	f.statuses["/keys"] = http.StatusServiceUnavailable
	f.statuses["/timestamp"] = http.StatusServiceUnavailable
	f.now = f.now.Add(2 * time.Hour)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "expired")
}

func TestRecoveryReplacementAndDualAnchorMismatch(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery", "recovery")
	replacementSigner := newTestSigner(t, "replacement")
	replacement := Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: 10,
		ExpiresAt: f.now.Add(time.Hour), Authority: f.root.Authority,
		Keys:  []Key{replacementSigner.key},
		Roles: map[string]Role{"root": {KeyIDs: []string{"replacement"}, Threshold: 1}},
	}
	replacementEnvelope := signedJSON(t, replacement, replacementSigner)
	patch := RecoveryPatch{
		Type: "recovery", SchemaVersion: SchemaVersion, RecoveryEpoch: 1, PatchEpoch: 2,
		NotBefore: f.now.Add(-time.Minute), ExpiresAt: f.now.Add(time.Hour),
		AffectedRootMin: 1, AffectedRootMax: 2,
		NewBootstrapRoot: replacementEnvelope, DenyKeyIDs: []string{"root"},
		ClearOrigins: []string{f.server.URL},
	}
	f.responses["/recovery"] = signedJSON(t, patch, recovery)
	f.client.cfg.RecoveryKeys = []Key{recovery.key}
	f.client.cfg.RecoveryRole = Role{KeyIDs: []string{"recovery"}, Threshold: 1}
	state := State{RootVersion: 1, KeysVersion: 4, TimestampVersion: 5, IndexVersion: 6}
	cache := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(cache, "artifact-keys.json"), []byte("old"), 0600))
	clearedOrigin := filepath.Join(filepath.Dir(cache), digest(f.server.URL+"/plugins"))
	require.NoError(t, os.MkdirAll(clearedOrigin, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(clearedOrigin, sourceBaseFile), []byte(f.server.URL+"/plugins"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(clearedOrigin, "old"), []byte("old"), 0600))

	got, err := f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery", cache, f.root, &state)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), got.Version)
	assert.Equal(t, uint64(1), state.RecoveryEpoch)
	assert.Equal(t, []string{"root"}, state.DenyKeyIDs)
	_, err = os.Stat(filepath.Join(cache, "artifact-keys.json"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(clearedOrigin)
	assert.True(t, os.IsNotExist(err))
	persisted, err := f.client.loadPersistedRoot(
		cache,
		f.root,
		state,
		f.now,
		map[string]bool{"root": true},
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), persisted.Version)

	patch.RecoveryEpoch = 2
	patch.DenyKeyIDs = []string{"replacement"}
	f.responses["/recovery"] = signedJSON(t, patch, recovery)
	_, err = f.client.CheckAndApplyRecovery(context.Background(), f.server.URL+"/recovery", cache, f.root, &state)
	assert.ErrorContains(t, err, "contains denied key")
}

func (f *trustFixture) signerRoot(version uint64, signer testSigner, expires time.Time) Root {
	return Root{
		Type: "root", SchemaVersion: SchemaVersion, Version: version,
		ExpiresAt: expires, Authority: f.root.Authority,
		Keys: []Key{signer.key},
		Roles: map[string]Role{
			"root":          {KeyIDs: []string{signer.key.KeyID}, Threshold: 1},
			"artifact-keys": {KeyIDs: []string{signer.key.KeyID}, Threshold: 1},
		},
	}
}

const fixtureRecoveryPath = "/plugins-v2/trust/recovery/current.json"

func (f *trustFixture) publishRecovery(recovery testSigner, epoch uint64, replacement Root, replacementSigner testSigner, deny ...string) {
	f.t.Helper()
	patch := RecoveryPatch{
		Type: "recovery", SchemaVersion: SchemaVersion,
		RecoveryEpoch: epoch, PatchEpoch: epoch,
		NotBefore: f.now.Add(-time.Minute), ExpiresAt: f.now.Add(time.Hour),
		AffectedRootMin: 1, AffectedRootMax: 100,
		NewBootstrapRoot: signedJSON(f.t, replacement, replacementSigner),
		DenyKeyIDs:       deny,
	}
	f.responses[fixtureRecoveryPath] = signedJSON(f.t, patch, recovery)
	f.client.cfg.RecoveryKeys = []Key{recovery.key}
	f.client.cfg.RecoveryRole = Role{KeyIDs: []string{recovery.key.KeyID}, Threshold: 1}
}

func TestUnsignedDiscoveryFieldsDoNotResetRollbackState(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	oldKeys, oldTimestamp, oldIndex := f.responses["/keys"], f.responses["/timestamp"], f.responses["/index"]

	f.publish([]byte(`{"plugins":[{"name":"next"}]}`), 2, 2, f.plugin)
	second, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.Equal(t, uint64(2), second.Descriptor.Version)

	for name, mutate := range map[string]func(*Discovery){
		"authority": func(d *Discovery) { d.Authority = "https://api.alibabacloud.com" },
		"origins": func(d *Discovery) {
			d.OfficialArtifactOrigins = append([]string{"https://mirror.example"}, d.OfficialArtifactOrigins...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			f.responses["/keys"], f.responses["/timestamp"], f.responses["/index"] = oldKeys, oldTimestamp, oldIndex
			var discovery Discovery
			require.NoError(t, json.Unmarshal(f.responses["/discovery"], &discovery))
			mutate(&discovery)
			data, err := json.Marshal(discovery)
			require.NoError(t, err)
			f.responses["/discovery"] = data

			got, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
			if err == nil {
				assert.Equal(t, uint64(2), got.Descriptor.Version, "replayed v1 must not be accepted")
			}
		})
	}
}

func TestRecoveryIsCheckedEvenWhenDiscoveryOmitsIt(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery", "recovery")
	replacementSigner := newTestSigner(t, "replacement")
	replacement := f.signerRoot(1, replacementSigner, f.now.Add(24*time.Hour))
	f.publishRecovery(recovery, 1, replacement, replacementSigner, "root")

	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "signature threshold not met")

	f.rootSigner = replacementSigner
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
}

func TestRecoveryReplacesSameVersionRootAcrossRuns(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery", "recovery")
	replacementSigner := newTestSigner(t, "replacement")
	f.publishRecovery(recovery, 1, f.signerRoot(1, replacementSigner, f.now.Add(24*time.Hour)), replacementSigner, "root")
	f.rootSigner = replacementSigner
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)

	for run := 0; run < 2; run++ {
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err, "run %d", run)
	}
}

func TestRecoveryRootContinuesThroughLaterRotation(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery", "recovery")
	v10Signer := newTestSigner(t, "root-v10")
	f.publishRecovery(recovery, 1, f.signerRoot(10, v10Signer, f.now.Add(24*time.Hour)), v10Signer, "root")
	f.rootSigner = v10Signer
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)

	v11Signer := newTestSigner(t, "root-v11")
	f.responses["/roots/11.root.json"] = signedJSON(t, f.signerRoot(11, v11Signer, f.now.Add(24*time.Hour)), v10Signer, v11Signer)
	f.rootSigner = v11Signer
	f.publish([]byte(`{"plugins":[]}`), 2, 2, f.plugin)
	for run := 0; run < 2; run++ {
		_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err, "run %d", run)
	}
}

func TestExpiredPersistedRootDoesNotBlockRecovery(t *testing.T) {
	f := newTrustFixture(t)
	nextSigner := newTestSigner(t, "root-next")
	delete(f.statuses, "/roots/2.root.json")
	f.responses["/roots/2.root.json"] = signedJSON(t, f.signerRoot(2, nextSigner, f.now.Add(time.Minute)), f.rootSigner, nextSigner)
	f.rootSigner = nextSigner
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)

	f.now = f.now.Add(2 * time.Minute)
	f.publish([]byte(`{"plugins":[]}`), 2, 2, f.plugin)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	assert.ErrorContains(t, err, "expired")

	recovery := newTestSigner(t, "recovery", "recovery")
	replacementSigner := newTestSigner(t, "replacement")
	f.publishRecovery(recovery, 1, f.signerRoot(3, replacementSigner, f.now.Add(24*time.Hour)), replacementSigner, "root-next")
	f.rootSigner = replacementSigner
	f.publish([]byte(`{"plugins":[]}`), 3, 3, f.plugin)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
}

func TestDiscoveryURLForSite(t *testing.T) {
	assert.Equal(t, DiscoveryURLCN, DiscoveryURLForSite("cn"))
	assert.Equal(t, DiscoveryURLIntl, DiscoveryURLForSite("intl"))
	assert.Equal(t, DiscoveryURLCN, DiscoveryURLForSite(""))
	assert.Equal(t, DiscoveryURLCN, DiscoveryURLForSite("invalid"))
}

func TestInnerSourceStaysOffPublicDiscovery(t *testing.T) {
	client := New(Config{
		DiscoveryURL: DiscoveryURLCN,
		SourceBase:   "https://cli.aliyun-inc.com/registry_id/2/env/pre/plugins-v2",
	})

	got, err := client.discoveryURL()
	require.NoError(t, err)
	assert.NotContains(t, got, "cli.aliyun-inc.com")
	assert.Contains(t, got, "source_base=https%3A%2F%2Faliyuncli.alicdn.com%2Fplugins-v2")
	base := "https://cli.aliyun-inc.com/registry_id/2/env/pre/plugins-v2/trust/"
	discovery := Discovery{}
	client.useRepositoryMetadata(&discovery)
	assert.Equal(t, base+"roots/", discovery.RootMetadataURI)
	assert.Equal(t, base+"artifact-keys.json", discovery.ArtifactKeysURI)
	assert.Equal(t, base+"timestamp.json", discovery.TimestampURI)
	assert.Equal(t, base+"recovery/current.json", discovery.RecoveryPatchURI)
	assert.Contains(t, client.cfg.MetadataHosts, "cli.aliyun-inc.com")
	assert.Contains(t, client.cfg.RecoveryHosts, "cli.aliyun-inc.com")
	assert.NoError(t, client.validateURL(got, client.cfg.MetadataHosts))
	assert.NoError(t, client.validateURL(discovery.ArtifactKeysURI, client.cfg.MetadataHosts))
	assert.NoError(t, client.validateURL(discovery.RecoveryPatchURI, client.cfg.RecoveryHosts))
	assert.Error(t, client.validateURL(
		"https://evil.example/trust/timestamp.json",
		client.cfg.MetadataHosts,
	))
}

func TestNewUsesConfiguredPluginSite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(pluginsettings.EnvSite, "")
	require.NoError(t, pluginsettings.Save(
		filepath.Join(home, ".aliyun"),
		&pluginsettings.PluginSettings{Site: pluginsettings.SiteIntl},
	))

	client := New(Config{})

	assert.Equal(t, DiscoveryURLIntl, client.cfg.DiscoveryURL)
	assert.Contains(t, client.cfg.MetadataHosts, "api.alibabacloud.com")
}

func TestBuiltinMaterialPrefersLinkerInjectedBase64(t *testing.T) {
	assert.Equal(t, []byte("encoded"), builtinMaterial(
		"plain", base64.StdEncoding.EncodeToString([]byte("encoded")),
	))
	assert.Nil(t, builtinMaterial("plain", "not-base64"))
	assert.Equal(t, []byte("plain"), builtinMaterial("plain", ""))
}

// The release platform (portal-cluster-v2 _validate_recovery_payload) publishes
// exactly these fields; keep both sides in sync.
func TestRecoveryPatchContractWithPublisher(t *testing.T) {
	published := []byte(`{"type":"recovery","schema_version":1,"recovery_epoch":1,` +
		`"patch_epoch":1,"not_before":"2026-01-01T00:00:00Z","expires_at":"2027-01-01T00:00:00Z",` +
		`"affected_root_min":1,"affected_root_max":3,"new_bootstrap_root":{"schema_version":1,` +
		`"payload":"e30=","signatures":[]},"deny_key_ids":["leaked-root"],` +
		`"clear_origins":["https://mirror.example"]}`)
	var patch RecoveryPatch
	require.NoError(t, strictJSON(published, &patch))
	assert.Equal(t, uint64(3), patch.AffectedRootMax)

	withAuthority := []byte(`{"type":"recovery","authority":"https://api.aliyun.com"}`)
	assert.Error(t, strictJSON(withAuthority, &RecoveryPatch{}))
}

func TestPluginIndexStillRequiresMatchingPurpose(t *testing.T) {
	now := time.Now().UTC()
	signer := newTestSigner(t, "plugins")
	payload := []byte(`{"plugins":[]}`)
	_, _, err := VerifyEnvelope(
		signedBytes(t, payload, signer),
		[]Key{signer.key},
		Role{KeyIDs: []string{"plugins"}, Threshold: 1},
		PurposePluginIndex,
		now,
		nil,
	)
	assert.ErrorContains(t, err, "threshold not met")
}

// Build anchors come from portal-cluster-v2 generate_cli_trust_bundle.py, which
// writes recovery keys without purposes.
func TestGeneratedRecoveryAnchorWithoutPurposesAppliesAcrossRuns(t *testing.T) {
	f := newTrustFixture(t)
	recovery := newTestSigner(t, "recovery-a")
	replacementSigner := newTestSigner(t, "replacement")
	f.publishRecovery(recovery, 1, f.signerRoot(1, replacementSigner, f.now.Add(24*time.Hour)), replacementSigner, "root")

	generated, err := json.Marshal([]map[string]string{{
		"algorithm": AlgorithmEd25519, "key_id": recovery.key.KeyID, "public_key": recovery.key.PublicKey,
	}})
	require.NoError(t, err)
	var anchors []Key
	require.NoError(t, strictJSON(generated, &anchors))
	require.Empty(t, anchors[0].Purposes)
	f.client.cfg.RecoveryKeys = anchors

	f.rootSigner = replacementSigner
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	for run := 0; run < 2; run++ {
		_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
		require.NoError(t, err, "run %d", run)
	}
}
