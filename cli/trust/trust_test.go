package trust_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerifyDetached(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	payload := []byte(`{"index_version":3,"plugins":[]}`)
	sig, err := trust.Sign("plugins-test", priv, payload, time.Now())
	require.NoError(t, err)

	keys := []trust.VerifyKey{{
		KeyID:     "plugins-test",
		PublicKey: pub,
		Roles:     []string{trust.RolePlugins},
	}}
	require.NoError(t, trust.VerifyDetached(payload, sig, keys, trust.RolePlugins))

	bad := append([]byte(nil), payload...)
	bad[0] ^= 0xff
	assert.Error(t, trust.VerifyDetached(bad, sig, keys, trust.RolePlugins))
}

func TestVerifyArtifactBytes_UnsignedTransition(t *testing.T) {
	dir := t.TempDir()
	payload := []byte(`{"plugins":[]}`)
	policy := trust.Policy{TrustDir: dir, Now: time.Now()}
	check, err := trust.VerifyArtifactBytes(payload, nil, trust.RolePlugins, "plugin_pkg_index", nil, policy)
	require.NoError(t, err)
	assert.False(t, check.Verified)
	assert.Contains(t, check.Warning, "unsigned")
}

func TestVerifyArtifactBytes_EnforceRequiresSig(t *testing.T) {
	payload := []byte(`{"plugins":[]}`)
	policy := trust.Policy{Enforce: true, TrustDir: t.TempDir(), Now: time.Now()}
	_, err := trust.VerifyArtifactBytes(payload, nil, trust.RolePlugins, "plugin_pkg_index", nil, policy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing signature")
}

func TestVerifyArtifactBytes_Freshness(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	dir := t.TempDir()

	payload := []byte(`{"index_version":5,"expires_at":"2099-01-01T00:00:00Z","plugins":[]}`)
	sig, err := trust.Sign("k1", priv, payload, time.Now())
	require.NoError(t, err)
	sigBytes, err := trust.MarshalSignature(sig)
	require.NoError(t, err)

	keys := []trust.VerifyKey{{KeyID: "k1", PublicKey: pub, Roles: []string{trust.RolePlugins}}}
	policy := trust.Policy{TrustDir: dir, Now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	check, err := trust.VerifyArtifactBytes(payload, sigBytes, trust.RolePlugins, "plugin_pkg_index", keys, policy)
	require.NoError(t, err)
	assert.True(t, check.Verified)
	assert.EqualValues(t, 5, check.Freshness.IndexVersion)

	// Rollback rejected
	older := []byte(`{"index_version":4,"expires_at":"2099-01-01T00:00:00Z","plugins":[]}`)
	sig2, err := trust.Sign("k1", priv, older, time.Now())
	require.NoError(t, err)
	sig2Bytes, _ := trust.MarshalSignature(sig2)
	_, err = trust.VerifyArtifactBytes(older, sig2Bytes, trust.RolePlugins, "plugin_pkg_index", keys, policy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback")
}

func TestVerifyArtifactBytes_Expired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	payload := []byte(`{"index_version":1,"expires_at":"2020-01-01T00:00:00Z","plugins":[]}`)
	sig, err := trust.Sign("k1", priv, payload, time.Now())
	require.NoError(t, err)
	sigBytes, _ := trust.MarshalSignature(sig)
	keys := []trust.VerifyKey{{KeyID: "k1", PublicKey: pub}}
	policy := trust.Policy{TrustDir: t.TempDir(), Now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	_, err = trust.VerifyArtifactBytes(payload, sigBytes, trust.RolePlugins, "plugin_pkg_index", keys, policy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestRootSignVerifyAndDelegate(t *testing.T) {
	rootPub, rootPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	pluginPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	root := &trust.RootDocument{
		Schema:    trust.RootSchema,
		Version:   1,
		ExpiresAt: "2099-01-01T00:00:00Z",
		Keys: []trust.RootKey{{
			KeyID:     "plugins-v1",
			PublicKey: base64.StdEncoding.EncodeToString(pluginPub),
			Roles:     []string{trust.RolePlugins},
		}},
	}
	require.NoError(t, trust.SignRoot(root, "root-1", rootPriv))

	raw, err := json.Marshal(root)
	require.NoError(t, err)
	parsed, err := trust.ParseRoot(raw)
	require.NoError(t, err)

	rootKeys := []trust.VerifyKey{{KeyID: "root-1", PublicKey: rootPub, Roles: []string{trust.RoleRoot}}}
	require.NoError(t, trust.VerifyRoot(parsed, rootKeys, time.Now(), 0))

	delegated, err := trust.KeysFromRoot(parsed)
	require.NoError(t, err)
	require.Len(t, delegated, 1)
	assert.Equal(t, "plugins-v1", delegated[0].KeyID)
}

func TestDeriveTrustRootURL(t *testing.T) {
	assert.Equal(t,
		"https://example.com/reg/trust/root.json",
		trust.DeriveTrustRootURL("https://example.com/reg/plugins-pb/plugin_pkg_index.json"),
	)
	assert.Equal(t,
		"https://example.com/reg/trust/root.json",
		trust.DeriveTrustRootURL("https://example.com/reg/plugins/plugin_pkg_index.json"),
	)
}

func TestResolveVerifyKeys_FromRoot(t *testing.T) {
	rootPub, rootPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	pluginPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	root := &trust.RootDocument{
		Schema:  trust.RootSchema,
		Version: 2,
		Keys: []trust.RootKey{{
			KeyID:     "plugins-v1",
			PublicKey: base64.StdEncoding.EncodeToString(pluginPub),
			Roles:     []string{trust.RolePlugins},
		}},
	}
	require.NoError(t, trust.SignRoot(root, "root-1", rootPriv))
	raw, err := json.Marshal(root)
	require.NoError(t, err)

	dir := t.TempDir()
	policy := trust.Policy{TrustDir: dir, Now: time.Now()}
	bootstrap := []trust.VerifyKey{{KeyID: "root-1", PublicKey: rootPub, Roles: []string{trust.RoleRoot}}}

	keys, err := trust.ResolveVerifyKeys(trust.RolePlugins, func() ([]byte, error) {
		return raw, nil
	}, bootstrap, policy, true)
	require.NoError(t, err)
	require.True(t, len(keys) >= 1)
	_, ok := trust.FindKey(keys, "plugins-v1", trust.RolePlugins)
	assert.True(t, ok)

	_, data, err := trust.LoadCachedRoot(dir)
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(data))
}

func TestPubKeysEnv(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	b64 := base64.StdEncoding.EncodeToString(pub)
	t.Setenv("ALIBABA_CLOUD_CLI_TRUST_PUBKEYS", "env-k:"+b64+":plugins")
	trust.ResetVerifyKeys()
	keys := trust.ActiveVerifyKeys()
	_, ok := trust.FindKey(keys, "env-k", trust.RolePlugins)
	assert.True(t, ok)
}

func TestVersionStore(t *testing.T) {
	dir := t.TempDir()
	s := trust.VersionStore{Dir: dir}
	n, err := s.Load("plugin_pkg_index")
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)
	require.NoError(t, s.Save("plugin_pkg_index", 9))
	n, err = s.Load("plugin_pkg_index")
	require.NoError(t, err)
	assert.EqualValues(t, 9, n)
	_, err = os.Stat(filepath.Join(dir, "plugin_pkg_index_version"))
	require.NoError(t, err)
}
