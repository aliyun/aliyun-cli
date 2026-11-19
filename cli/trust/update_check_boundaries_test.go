package trust

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	"github.com/stretchr/testify/require"
)

func TestUpdateCheckSnapshotCopiesOnlyRelevantCacheEntries(t *testing.T) {
	f := newTrustFixture(t)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	repository := filepath.Dir(artifact.Proof.StatePath)
	for _, name := range []string{".pending.tmp", PurposeUpgrade + ".envelope.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(repository, name), []byte("unrelated"), 0600))
	}
	snapshot, err := f.client.snapshotForUpdateCheck(context.Background(), PurposePluginIndex)
	require.NoError(t, err)
	defer os.RemoveAll(snapshot)
	relative, err := filepath.Rel(f.cacheDir, repository)
	require.NoError(t, err)
	for _, name := range []string{".pending.tmp", PurposeUpgrade + ".envelope.json"} {
		require.NoFileExists(t, filepath.Join(snapshot, relative, name))
	}
	original, err := os.ReadFile(filepath.Join(repository, PurposePluginIndex+".envelope.json"))
	require.NoError(t, err)
	copy, err := os.ReadFile(filepath.Join(snapshot, relative, PurposePluginIndex+".envelope.json"))
	require.NoError(t, err)
	require.Equal(t, original, copy)
	require.True(t, artifact.Proof.Valid(f.now))
}

func TestUpdateCheckSnapshotFailuresLeaveVerifiedStateUntouched(t *testing.T) {
	for _, scenario := range []string{"bootstrap", "source", "cache_path", "lock_path", "cancelled", "oversized"} {
		t.Run(scenario, func(t *testing.T) {
			f := newTrustFixture(t)
			artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
			require.NoError(t, err)
			ctx := context.Background()
			switch scenario {
			case "bootstrap":
				f.client.cfg.BootstrapRoot = []byte("broken")
			case "source":
				f.client.cfg.SourceBase = "invalid"
			case "cache_path":
				blocked := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(blocked, []byte("file"), 0600))
				f.client.cfg.CacheDir = blocked
			case "lock_path":
				lock := filepath.Join(f.cacheDir, ".metadata.lock")
				require.NoError(t, os.Remove(lock))
				require.NoError(t, os.Mkdir(lock, 0700))
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "oversized":
				file, err := os.Create(filepath.Join(filepath.Dir(artifact.Proof.StatePath), "oversized.json"))
				require.NoError(t, err)
				require.NoError(t, file.Truncate(maxResponseBytes+1))
				require.NoError(t, file.Close())
			}
			_, err = f.client.FetchArtifactForUpdateCheck(ctx, PurposePluginIndex, "plugins")
			require.Error(t, err)
			if scenario == "cancelled" {
				require.ErrorIs(t, err, context.Canceled)
			}
			require.True(t, artifact.Proof.Valid(f.now))
		})
	}
}

func TestUpdateCheckSnapshotLockHonorsCancellation(t *testing.T) {
	f := newTrustFixture(t)
	unlock, err := localstate.Lock(nil, filepath.Join(f.cacheDir, ".metadata.lock"))
	require.NoError(t, err)
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = f.client.snapshotForUpdateCheck(ctx, PurposePluginIndex)
	require.ErrorIs(t, err, context.Canceled)
}

func TestUpdateCheckReplayDoesNotShareBodiesOrHeaders(t *testing.T) {
	var calls int
	fetched := &updateCheckResponses{responses: make(map[string]updateCheckResponse), base: updateCheckTransportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Fixture": {"original"}}, Body: io.NopCloser(bytes.NewBufferString("verified bytes"))}, nil
	})}
	request, err := http.NewRequest(http.MethodGet, "https://fixture.test/index", nil)
	require.NoError(t, err)
	first, err := fetched.RoundTrip(request)
	require.NoError(t, err)
	first.Header.Set("X-Fixture", "changed")
	_, err = io.ReadAll(first.Body)
	require.NoError(t, err)
	require.NoError(t, first.Body.Close())
	replay := updateCheckReplayTransport{fetched: fetched}
	second, err := replay.RoundTrip(request)
	require.NoError(t, err)
	defer second.Body.Close()
	body, err := io.ReadAll(second.Body)
	require.NoError(t, err)
	require.Equal(t, "verified bytes", string(body))
	require.Equal(t, "original", second.Header.Get("X-Fixture"))
	other, err := http.NewRequest(http.MethodGet, "https://fixture.test/new-root", nil)
	require.NoError(t, err)
	_, err = replay.RoundTrip(other)
	require.ErrorContains(t, err, "metadata changed")
	require.Equal(t, 1, calls, "commit-time replay must never make another network request")
	nilResponse := &updateCheckResponses{responses: make(map[string]updateCheckResponse), base: updateCheckTransportFunc(func(*http.Request) (*http.Response, error) { return nil, nil })}
	_, err = nilResponse.RoundTrip(request)
	require.ErrorContains(t, err, "empty trust response")
}

func TestNotificationProofAcceptsLegacyIndexStateButRejectsCorruption(t *testing.T) {
	f := newTrustFixture(t)
	artifact, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	body, err := os.ReadFile(artifact.Proof.StatePath)
	require.NoError(t, err)
	var state State
	require.NoError(t, json.Unmarshal(body, &state))
	state.IndexVersion = state.ArtifactVersions[PurposePluginIndex]
	delete(state.ArtifactVersions, PurposePluginIndex)
	require.NoError(t, os.WriteFile(artifact.Proof.StatePath, mustJSON(state), 0600))
	require.True(t, artifact.Proof.Valid(f.now))
	state.IndexVersion++
	require.NoError(t, os.WriteFile(artifact.Proof.StatePath, mustJSON(state), 0600))
	require.False(t, artifact.Proof.Valid(f.now))
	require.NoError(t, os.WriteFile(artifact.Proof.StatePath, []byte("broken"), 0600))
	require.False(t, artifact.Proof.Valid(f.now))
}
