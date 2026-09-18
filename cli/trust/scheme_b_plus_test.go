package trust_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTrustURL(t *testing.T) {
	require.NoError(t, trust.ValidateTrustURL("https://api.aliyun.com/cli-trust/v1/x", trust.BuiltinHostAllowlist))
	require.Error(t, trust.ValidateTrustURL("http://api.aliyun.com/x", trust.BuiltinHostAllowlist))
	require.Error(t, trust.ValidateTrustURL("https://evil.example/x", trust.BuiltinHostAllowlist))
	require.Error(t, trust.ValidateTrustURL("https://127.0.0.1/x", trust.BuiltinHostAllowlist))
}

func TestPlusClientRefresh(t *testing.T) {
	rootA := mustSeed(t, "Wxli+f+mtpQv1Xq5XN1YPeBlt73nVT5x0En0ZVcdaRA=")
	rootB := mustSeed(t, "rWV8nTq5EfYP0z5C8THMK3D2iPKb/eycV9Is9XZLQvo=")
	pluginsSeed := mustSeed(t, "afJSG9y3n22bSzYb0VkYV8IA9vMdZBzq1/zwzDwv4AY=")
	dir := t.TempDir()
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)

	// 2.root.json authorizing plugins via roles + we'll also publish artifact-keys.
	rootDoc := trust.RootMetaFile{
		Signed: trust.RootMeta{
			Type:          trust.RootMetaType,
			SchemaVersion: 1,
			Version:       2,
			ExpiresAt:     "2099-01-01T00:00:00Z",
			Keys: map[string]trust.KeySpec{
				"root-a": {Algorithm: "Ed25519", PublicKey: "w5o+mm2S3+BnIC6VRmDIkpF2MdCD/3/WVckw7W4aSGc="},
				"root-b": {Algorithm: "Ed25519", PublicKey: "8yoxXpLE6JrQRayPKu8ac8WAdCoLM3hzNE49xIuGQ10="},
				"root-c": {Algorithm: "Ed25519", PublicKey: "Ia+1RKbG4xPWL5HBzmzI6f0zkYK/510/XqHdx48U8MA="},
			},
			Roles: map[string]trust.RoleSpec{
				trust.RoleRoot: {Threshold: 2, KeyIDs: []string{"root-a", "root-b", "root-c"}},
			},
		},
	}
	rootPayload, err := rootDoc.SignedPayload()
	require.NoError(t, err)
	require.NoError(t, trust.SignMetaPayload(&rootDoc.Signatures, "root-a", rootA, rootPayload))
	require.NoError(t, trust.SignMetaPayload(&rootDoc.Signatures, "root-b", rootB, rootPayload))
	rootRaw, err := json.Marshal(rootDoc)
	require.NoError(t, err)

	ak := trust.ArtifactKeysFile{
		Signed: trust.ArtifactKeysMeta{
			Type:          trust.ArtifactKeysType,
			SchemaVersion: 1,
			Version:       12,
			ExpiresAt:     "2099-01-01T00:00:00Z",
			Authority:     "https://api.aliyun.com",
			Keys: []trust.ArtifactKeyEntry{{
				KeyID:     "plugins-dev",
				Algorithm: "Ed25519",
				PublicKey: "rsZqqgDW5ovboRIQvyNBdKAtEjfjsw28uJJ7w5+fLrQ=",
				Purposes:  []string{"plugin-index"},
			}},
		},
	}
	akRawSigned, err := json.Marshal(ak.Signed)
	require.NoError(t, err)
	akPayload, err := trust.CanonicalSignedBytes(akRawSigned)
	require.NoError(t, err)
	require.NoError(t, trust.SignMetaPayload(&ak.Signatures, "root-a", rootA, akPayload))
	require.NoError(t, trust.SignMetaPayload(&ak.Signatures, "root-b", rootB, akPayload))
	akRaw, err := json.Marshal(ak)
	require.NoError(t, err)

	discovery := []byte(`{
	  "schema_version": 1,
	  "authority": "https://api.aliyun.com",
	  "root_metadata_uri": "https://api.aliyun.com/cli-trust/v1/roots/",
	  "artifact_keys_uri": "https://api.aliyun.com/cli-trust/v1/artifact-keys.json",
	  "timestamp_uri": "https://api.aliyun.com/cli-trust/v1/timestamp.json",
	  "official_artifact_origins": ["https://aliyuncli.alicdn.com"],
	  "supported_signature_algorithms": ["Ed25519"],
	  "supported_artifact_types": ["plugin-index", "cli-upgrade-manifest"],
	  "expires_at": "2099-01-01T00:00:00Z"
	}`)

	files := map[string][]byte{
		trust.DefaultDiscoveryURL: discovery,
		"https://api.aliyun.com/cli-trust/v1/roots/2.root.json": rootRaw,
		"https://api.aliyun.com/cli-trust/v1/artifact-keys.json": akRaw,
	}
	fetch := func(u string) ([]byte, error) {
		if b, ok := files[u]; ok {
			return b, nil
		}
		return nil, fmt.Errorf("404 %s", u)
	}

	plus := trust.NewPlusClient(dir, fetch)
	plus.Now = now
	res, err := plus.Refresh()
	require.NoError(t, err)
	require.NotNil(t, res.Discovery)
	require.NotNil(t, res.ArtifactKeys)
	keys := res.RoleKeys[trust.RolePlugins]
	require.Len(t, keys, 1)
	assert.Equal(t, "plugins-dev", keys[0].KeyID)

	// Signed index verifies with discovered key.
	payload := []byte(`{"index_version":1,"expires_at":"2099-01-01T00:00:00Z","plugins":[]}`)
	sig, err := trust.Sign("plugins-dev", pluginsSeed, payload, now)
	require.NoError(t, err)
	sigBytes, _ := trust.MarshalSignature(sig)
	check, err := trust.VerifyArtifactBytes(payload, sigBytes, trust.RolePlugins, "plugin_pkg_index", keys, trust.Policy{TrustDir: dir, Now: now})
	require.NoError(t, err)
	assert.True(t, check.Verified)
}
