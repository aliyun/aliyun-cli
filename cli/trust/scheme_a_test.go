package trust_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustSeed(t *testing.T, b64 string) ed25519.PrivateKey {
	t.Helper()
	priv, err := trust.DecodePrivateSeed(b64)
	require.NoError(t, err)
	return priv
}

func TestEmbeddedBootstrapKeys(t *testing.T) {
	b := trust.EmbeddedBootstrap()
	assert.Equal(t, 2, b.RootRole.Threshold)
	assert.Equal(t, 2, b.RecoveryRole.Threshold)
	rootKeys, err := b.RootVerifyKeys()
	require.NoError(t, err)
	assert.Len(t, rootKeys, 3)
	recKeys, err := b.RecoveryVerifyKeys()
	require.NoError(t, err)
	assert.Len(t, recKeys, 3)
}

func TestRootChainDualThreshold(t *testing.T) {
	// Dev seeds matching EmbeddedBootstrap public keys (tests only).
	rootA := mustSeed(t, "Wxli+f+mtpQv1Xq5XN1YPeBlt73nVT5x0En0ZVcdaRA=")
	rootB := mustSeed(t, "rWV8nTq5EfYP0z5C8THMK3D2iPKb/eycV9Is9XZLQvo=")
	rootC := mustSeed(t, "/hHSK+TgTHQ//KHvdEsrlGiDIIVR7He8cD7h7uCxaVs=")

	pluginsPub := "rsZqqgDW5ovboRIQvyNBdKAtEjfjsw28uJJ7w5+fLrQ="
	dir := t.TempDir()
	now := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)

	makeRoot := func(version int64, rootIDs []string, privs []ed25519.PrivateKey) []byte {
		keys := map[string]trust.KeySpec{
			"root-a": {Algorithm: "Ed25519", PublicKey: "w5o+mm2S3+BnIC6VRmDIkpF2MdCD/3/WVckw7W4aSGc="},
			"root-b": {Algorithm: "Ed25519", PublicKey: "8yoxXpLE6JrQRayPKu8ac8WAdCoLM3hzNE49xIuGQ10="},
			"root-c": {Algorithm: "Ed25519", PublicKey: "Ia+1RKbG4xPWL5HBzmzI6f0zkYK/510/XqHdx48U8MA="},
			"plugins-dev": {Algorithm: "Ed25519", PublicKey: pluginsPub},
		}
		doc := trust.RootMetaFile{
			Signed: trust.RootMeta{
				Type:          trust.RootMetaType,
				SchemaVersion: 1,
				Version:       version,
				ExpiresAt:     "2099-01-01T00:00:00Z",
				Keys:          keys,
				Roles: map[string]trust.RoleSpec{
					trust.RoleRoot:    {Threshold: 2, KeyIDs: rootIDs},
					trust.RolePlugins: {Threshold: 1, KeyIDs: []string{"plugins-dev"}},
				},
			},
		}
		payload, err := doc.SignedPayload()
		require.NoError(t, err)
		for i, id := range rootIDs {
			if i >= 2 {
				break // 2-of-3
			}
			require.NoError(t, trust.SignMetaPayload(&doc.Signatures, id, privs[i], payload))
		}
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		return raw
	}

	v2 := makeRoot(2, []string{"root-a", "root-b", "root-c"}, []ed25519.PrivateKey{rootA, rootB, rootC})
	files := map[string][]byte{
		"https://example.test/trust/roots/2.root.json": v2,
	}
	fetch := func(u string) ([]byte, error) {
		if b, ok := files[u]; ok {
			return b, nil
		}
		return nil, fmt.Errorf("404")
	}

	root, err := trust.UpdateRootChain(dir, "https://example.test/trust/roots/", fetch, trust.EmbeddedBootstrap(), now)
	require.NoError(t, err)
	require.NotNil(t, root)
	assert.EqualValues(t, 2, root.Signed.Version)
	keys, spec, err := root.RoleKeys(trust.RolePlugins)
	require.NoError(t, err)
	assert.Equal(t, 1, spec.Threshold)
	assert.Len(t, keys, 1)
	assert.FileExists(t, filepath.Join(dir, "roots", "2.root.json"))
}

func TestRecoveryPatchApply(t *testing.T) {
	recA := mustSeed(t, "sV4zWSFDzFgCFiLdznBBUTzt5dPhUKcUz1FSVXG7+n0=")
	recC := mustSeed(t, "BvO5wqf/EkyZniEWpUgOmzgB23S/dBF7R8Q/GLjON9c=")
	dir := t.TempDir()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	doc := trust.RecoveryFile{
		Signed: trust.RecoveryPatch{
			Type:          trust.RecoveryMetaType,
			SchemaVersion: 1,
			RecoveryEpoch: 2,
			ExpiresAt:     "2099-01-01T00:00:00Z",
			IncidentID:    "TEST-1",
			ReplacementBootstrapRoot: &struct {
				Version   int64                    `json:"version"`
				Threshold int                      `json:"threshold"`
				Keys      map[string]trust.KeySpec `json:"keys"`
			}{
				Version:   19,
				Threshold: 2,
				Keys: map[string]trust.KeySpec{
					"root-x": {Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(recA.Public().(ed25519.PublicKey))},
					"root-y": {Algorithm: "Ed25519", PublicKey: "8yoxXpLE6JrQRayPKu8ac8WAdCoLM3hzNE49xIuGQ10="},
					"root-z": {Algorithm: "Ed25519", PublicKey: "Ia+1RKbG4xPWL5HBzmzI6f0zkYK/510/XqHdx48U8MA="},
				},
			},
			Invalidate: map[string]bool{"plugin_index_cache": true, "timestamp_cache": true},
		},
	}
	rawSigned, err := json.Marshal(doc.Signed)
	require.NoError(t, err)
	payload, err := trust.CanonicalSignedBytes(rawSigned)
	require.NoError(t, err)
	require.NoError(t, trust.SignMetaPayload(&doc.Signatures, "recovery-a", recA, payload))
	require.NoError(t, trust.SignMetaPayload(&doc.Signatures, "recovery-c", recC, payload))

	keys, err := trust.EmbeddedBootstrap().RecoveryVerifyKeys()
	require.NoError(t, err)
	require.NoError(t, trust.VerifyRecovery(&doc, keys, 2, now, 0))
	require.NoError(t, trust.ApplyRecoveryPatch(dir, &doc, now))

	st, err := trust.LoadRecoveryState(dir)
	require.NoError(t, err)
	assert.EqualValues(t, 2, st.HighestRecoveryEpoch)
	assert.EqualValues(t, 19, st.ActiveBootstrapRootVersion)
	assert.FileExists(t, filepath.Join(dir, "roots", "19.root.json"))
}

func TestDeriveRootsBaseURL(t *testing.T) {
	assert.Equal(t, "https://aliyuncli.alicdn.com/trust/roots/",
		trust.DeriveRootsBaseURL("https://aliyuncli.alicdn.com/plugins/plugin_pkg_index.json"))
	assert.Equal(t, "https://aliyuncli.alicdn.com/trust/roots/",
		trust.DeriveRootsBaseURL("https://aliyuncli.alicdn.com/plugins-v2/plugin_pkg_index.json"))
}
