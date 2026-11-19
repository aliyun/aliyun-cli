package trust

import (
	"context"
	"encoding/json"
	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNotificationProofFreshnessExpiryAndRecovery(t *testing.T) {
	f := newTrustFixture(t)
	a, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.True(t, a.Fresh)
	require.True(t, a.Proof.Valid(f.now))
	require.False(t, a.Proof.Valid(a.Proof.ValidUntil))
	f.statuses["/index"] = http.StatusServiceUnavailable
	b, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.False(t, b.Fresh)
	var state State
	data, err := os.ReadFile(a.Proof.StatePath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &state))
	state.ArtifactVersions[PurposeUpgrade] = 5
	require.NoError(t, atomicWrite(a.Proof.StatePath, mustJSON(state), 0600))
	require.True(t, a.Proof.Valid(f.now), "an unrelated artifact does not invalidate plugin notices")
	state.RecoveryEpoch++
	require.NoError(t, atomicWrite(a.Proof.StatePath, mustJSON(state), 0600))
	require.False(t, a.Proof.Valid(f.now))
}

func TestNotificationProofExpiresWithTimestampSigningKey(t *testing.T) {
	for _, background := range []bool{false, true} {
		name := "foreground"
		if background {
			name = "background"
		}
		t.Run(name, func(t *testing.T) {
			f := newTrustFixture(t)
			deadline := f.now.Add(time.Minute)
			f.timestamp.key.NotAfter = deadline
			f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
			fetch := f.client.FetchArtifact
			if background {
				fetch = f.client.FetchArtifactForUpdateCheck
			}
			artifact, err := fetch(context.Background(), PurposePluginIndex, "plugins")
			require.NoError(t, err)
			require.True(t, artifact.Proof.Valid(f.now))
			f.now = f.now.Add(2 * time.Minute)
			_, err = fetch(context.Background(), PurposePluginIndex, "plugins")
			require.ErrorContains(t, err, "verify timestamp")
			require.False(t, artifact.Proof.Valid(f.now), "a rejected timestamp must not leave a usable notification proof")
			require.Equal(t, deadline, artifact.Proof.ValidUntil)
			require.True(t, artifact.Proof.Valid(deadline.Add(-time.Nanosecond)))
			require.False(t, artifact.Proof.Valid(deadline))
		})
	}
}

func TestNotificationProofUsesCachedTimestampSignerAfterRollback(t *testing.T) {
	f := newTrustFixture(t)
	deadline := f.now.Add(time.Minute)
	f.timestamp.key.NotAfter = deadline
	f.publish([]byte(`{"plugins":[]}`), 1, 1, f.plugin)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	// A newer key set allows both signers. The remote timestamp is correctly
	// signed by the long-lived signer but rolled back, so only LKG is accepted.
	longLived := newTestSigner(t, "timestamp-long", "timestamp")
	_, payload, err := DecodeEnvelope(f.responses["/keys"])
	require.NoError(t, err)
	var keys ArtifactKeys
	require.NoError(t, json.Unmarshal(payload, &keys))
	keys.Version++
	keys.Keys = append(keys.Keys, longLived.key)
	role := keys.Roles["timestamp"]
	role.KeyIDs = append(role.KeyIDs, longLived.key.KeyID)
	keys.Roles["timestamp"] = role
	f.responses["/keys"] = signedJSON(t, keys, f.rootSigner)
	_, payload, err = DecodeEnvelope(f.responses["/timestamp"])
	require.NoError(t, err)
	var timestamp Timestamp
	require.NoError(t, json.Unmarshal(payload, &timestamp))
	timestamp.Version = 0
	f.responses["/timestamp"] = signedJSON(t, timestamp, longLived)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.False(t, artifact.Fresh)
	require.Equal(t, deadline, artifact.Proof.ValidUntil)
	require.Equal(t, []string{f.plugin.key.KeyID}, artifact.KeyIDs, "artifact audit IDs must remain artifact-specific")
	f.now = deadline
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.ErrorContains(t, err, "verify timestamp")
	require.False(t, artifact.Proof.Valid(f.now))
}

func TestNotificationProofIgnoresUnusedTimestampKeys(t *testing.T) {
	f := newTrustFixture(t)
	unused := newTestSigner(t, "unused-timestamp", "timestamp")
	unused.key.NotAfter = f.now.Add(time.Minute)
	_, payload, err := DecodeEnvelope(f.responses["/keys"])
	require.NoError(t, err)
	var keys ArtifactKeys
	require.NoError(t, json.Unmarshal(payload, &keys))
	keys.Keys = append(keys.Keys, unused.key)
	role := keys.Roles["timestamp"]
	role.KeyIDs = append(role.KeyIDs, unused.key.KeyID)
	keys.Roles["timestamp"] = role
	f.responses["/keys"] = signedJSON(t, keys, f.rootSigner)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.Equal(t, f.now.Add(time.Hour), artifact.Proof.ValidUntil)
	f.now = f.now.Add(2 * time.Minute)
	require.True(t, artifact.Proof.Valid(f.now), "an unused role key must not expire a valid proof")
	_, err = f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
}

func TestNotificationProofRejectsOlderAndUnknownSchemas(t *testing.T) {
	f := newTrustFixture(t)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	body, err := json.Marshal(artifact.Proof)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &fields))
	delete(fields, "schema_version")
	legacy, err := json.Marshal(fields)
	require.NoError(t, err)
	var proof CacheProof
	require.NoError(t, json.Unmarshal(legacy, &proof))
	require.False(t, proof.Valid(f.now), "pre-fix proofs must not retain their overlong lifetime")
	require.NoError(t, json.Unmarshal(body, &proof))
	require.True(t, proof.Valid(f.now), "current proofs must round-trip through the cache")
	for _, version := range []int{1, cacheProofSchemaVersion + 1} {
		proof.SchemaVersion = version
		require.False(t, proof.Valid(f.now), "old and unknown formats must not authorize reminders")
	}
}

func TestTrustLockHonorsDeadline(t *testing.T) {
	f := newTrustFixture(t)
	require.NoError(t, os.MkdirAll(f.cacheDir, 0700))
	unlock, err := localstate.Lock(nil, filepath.Join(f.cacheDir, ".metadata.lock"))
	require.NoError(t, err)
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = f.client.FetchArtifact(ctx, PurposePluginIndex, "plugins")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestMappedTrustHostRemainsAllowlisted(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ENDPOINT_SUFFIX", "aliyuncs.alibaba")
	c := New(Config{})
	hosts := transportHosts([]string{"packages.oss.aliyuncs.com"})
	require.NoError(t, c.validateURL("https://packages.oss.aliyuncs.alibaba/index", hosts))
	require.Error(t, c.validateURL("https://other.oss.aliyuncs.alibaba/index", hosts))
}
