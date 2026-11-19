package trust

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func configureArtifactKeysProofSigners(f *trustFixture, threshold int, signers ...testSigner) {
	f.t.Helper()
	role := Role{Threshold: threshold}
	for _, signer := range signers {
		f.root.Keys = append(f.root.Keys, signer.key)
		role.KeyIDs = append(role.KeyIDs, signer.key.KeyID)
	}
	f.root.Roles["artifact-keys"] = role
	f.client.cfg.BootstrapRoot = signedJSON(f.t, f.root, f.rootSigner)
}

func TestNotificationProofExpiresWithArtifactKeysSigner(t *testing.T) {
	for _, background := range []bool{false, true} {
		name := "foreground"
		if background {
			name = "background"
		}
		t.Run(name, func(t *testing.T) {
			f := newTrustFixture(t)
			deadline := f.now.Add(time.Minute)
			signer := newTestSigner(t, "artifact-keys-signer")
			signer.key.NotAfter = deadline
			configureArtifactKeysProofSigners(f, 1, signer)
			_, payload, err := DecodeEnvelope(f.responses["/keys"])
			require.NoError(t, err)
			f.responses["/keys"] = signedBytes(t, payload, signer)
			fetch := f.client.FetchArtifact
			if background {
				fetch = f.client.FetchArtifactForUpdateCheck
			}
			artifact, err := fetch(context.Background(), PurposePluginIndex, "plugins")
			require.NoError(t, err)
			require.True(t, artifact.Proof.Valid(f.now))
			f.now = f.now.Add(2 * time.Minute)
			_, err = fetch(context.Background(), PurposePluginIndex, "plugins")
			require.ErrorContains(t, err, "verify artifact keys")
			require.False(t, artifact.Proof.Valid(f.now), "an expired artifact-keys signer must also expire cached reminders")
			require.Equal(t, deadline, artifact.Proof.ValidUntil)
			require.True(t, artifact.Proof.Valid(deadline.Add(-time.Nanosecond)))
			require.False(t, artifact.Proof.Valid(deadline))
		})
	}
}

func TestNotificationProofUsesCachedArtifactKeysSignerAfterRollback(t *testing.T) {
	f := newTrustFixture(t)
	shortLived := newTestSigner(t, "artifact-keys-short")
	shortLived.key.NotAfter = f.now.Add(time.Minute)
	longLived := newTestSigner(t, "artifact-keys-long")
	configureArtifactKeysProofSigners(f, 1, shortLived, longLived)
	_, payload, err := DecodeEnvelope(f.responses["/keys"])
	require.NoError(t, err)
	f.responses["/keys"] = signedBytes(t, payload, shortLived)
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	// The remote document is validly signed but rolled back. Its long-lived
	// signer must not replace the identity of the accepted cached document.
	var keys ArtifactKeys
	require.NoError(t, json.Unmarshal(payload, &keys))
	keys.Version = 0
	f.responses["/keys"] = signedJSON(t, keys, longLived)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.False(t, artifact.Fresh)
	require.Equal(t, shortLived.key.NotAfter, artifact.Proof.ValidUntil)
	require.Equal(t, []string{f.plugin.key.KeyID}, artifact.KeyIDs)
	f.now = shortLived.key.NotAfter
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.ErrorContains(t, err, "verify artifact keys")
	require.False(t, artifact.Proof.Valid(f.now))
}

func TestNotificationProofUsesOnlyVerifiedArtifactKeysSigners(t *testing.T) {
	for _, threshold := range []int{1, 2} {
		t.Run(map[int]string{1: "unused_key", 2: "threshold_signers"}[threshold], func(t *testing.T) {
			f := newTrustFixture(t)
			shortLived := newTestSigner(t, "artifact-keys-short")
			shortLived.key.NotAfter = f.now.Add(time.Minute)
			longLived := newTestSigner(t, "artifact-keys-long")
			configureArtifactKeysProofSigners(f, threshold, shortLived, longLived)
			_, payload, err := DecodeEnvelope(f.responses["/keys"])
			require.NoError(t, err)
			if threshold == 1 {
				f.responses["/keys"] = signedBytes(t, payload, longLived)
			} else {
				f.responses["/keys"] = signedBytes(t, payload, shortLived, longLived)
			}
			artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
			require.NoError(t, err)
			if threshold == 1 {
				require.Equal(t, f.now.Add(time.Hour), artifact.Proof.ValidUntil)
			} else {
				require.Equal(t, shortLived.key.NotAfter, artifact.Proof.ValidUntil)
			}
			f.now = f.now.Add(2 * time.Minute)
			require.Equal(t, threshold == 1, artifact.Proof.Valid(f.now))
			_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
			if threshold == 1 {
				require.NoError(t, err, "an unused root key must not expire a valid proof")
			} else {
				require.ErrorContains(t, err, "verify artifact keys")
			}
		})
	}
}
