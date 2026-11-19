package trust

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	"github.com/stretchr/testify/require"
)

type updateCheckTransportFunc func(*http.Request) (*http.Response, error)

func (f updateCheckTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateCheckNetworkDoesNotHoldSharedTrustLock(t *testing.T) {
	f := newTrustFixture(t)
	httpClient := *f.client.cfg.HTTPClient
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = updateCheckTransportFunc(func(r *http.Request) (*http.Response, error) {
		unlock, err := localstate.Lock(nil, filepath.Join(f.cacheDir, ".metadata.lock"))
		if err != nil {
			t.Errorf("update-check networking held the foreground trust lock: %v", err)
		} else {
			unlock()
		}
		return base.RoundTrip(r)
	})
	cfg := f.client.cfg
	cfg.HTTPClient = &httpClient
	a, err := New(cfg).FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.True(t, a.Fresh)
	require.True(t, a.Proof.Valid(f.now))
	require.Contains(t, a.Proof.StatePath, f.cacheDir)
}

func TestSlowUpdateCheckDoesNotBlockOrRollBackForeground(t *testing.T) {
	f := newTrustFixture(t)
	_, err := f.client.FetchArtifact(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	httpClient := *f.client.cfg.HTTPClient
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = updateCheckTransportFunc(func(r *http.Request) (*http.Response, error) {
		response, err := base.RoundTrip(r)
		if r.URL.Path == "/index" {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
		return response, err
	})
	cfg := f.client.cfg
	cfg.HTTPClient = &httpClient
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		_, err := New(cfg).FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
		done <- err
	}()
	t.Cleanup(func() {
		unblock()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("background update check did not stop")
		}
	})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("update check did not reach the held network request")
	}
	f.publish([]byte(`{"plugins":[],"release":2}`), 2, 2, f.plugin)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	foreground, err := f.client.FetchArtifact(ctx, PurposePluginIndex, "plugins")
	require.NoError(t, err, "foreground must finish before background networking is released")
	require.EqualValues(t, 2, foreground.Descriptor.Version)
	unblock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("update check did not finish")
	}
	data, err := os.ReadFile(foreground.Proof.StatePath)
	require.NoError(t, err)
	var state State
	require.NoError(t, json.Unmarshal(data, &state))
	require.EqualValues(t, 2, state.KeysVersion)
	require.EqualValues(t, 2, state.TimestampVersion)
	require.EqualValues(t, 2, state.ArtifactVersions[PurposePluginIndex])
	require.True(t, foreground.Proof.Valid(f.now))
}

func TestUpdateCheckVerifiedCacheFallbackAndSignatureValidation(t *testing.T) {
	f := newTrustFixture(t)
	first, err := f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	indexPath := filepath.Join(filepath.Dir(first.Proof.StatePath), PurposePluginIndex+".envelope.json")
	verifiedBytes, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	// The transport succeeds, but the untrusted index must not replace LKG.
	f.responses["/index"] = []byte(`{"tampered":true}`)
	_, err = f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.Error(t, err)
	after, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	require.Equal(t, verifiedBytes, after)
	require.True(t, first.Proof.Valid(f.now))
	for _, path := range []string{"/discovery", "/keys", "/timestamp", "/index"} {
		f.statuses[path] = http.StatusServiceUnavailable
	}
	cached, err := f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.False(t, cached.Fresh)
	require.Equal(t, first.Bytes, cached.Bytes)
	require.True(t, cached.Proof.Valid(f.now))
}

func TestUpdateCheckPersistsRecoveryEvenWhenLaterVerificationFails(t *testing.T) {
	f := newTrustFixture(t)
	first, err := f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	recovery := newTestSigner(t, "recovery", "recovery")
	replacementSigner := newTestSigner(t, "replacement")
	replacement := f.signerRoot(1, replacementSigner, f.now.Add(time.Hour))
	f.publishRecovery(recovery, 1, replacement, replacementSigner, "root", "plugin-a")
	// Keys still use the revoked signer, so the index check fails after applying
	// recovery. The shared recovery state must nevertheless be persisted.
	_, err = f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.Error(t, err)
	data, err := os.ReadFile(first.Proof.StatePath)
	require.NoError(t, err)
	var state State
	require.NoError(t, json.Unmarshal(data, &state))
	require.EqualValues(t, 1, state.RecoveryEpoch)
	require.Contains(t, state.DenyKeyIDs, "plugin-a")
	require.False(t, first.Proof.Valid(f.now))
	require.NoFileExists(t, filepath.Join(filepath.Dir(first.Proof.StatePath), PurposePluginIndex+".envelope.json"))
}

func TestUpdateCheckAcceptsRootRotationAndInvalidatesOldProof(t *testing.T) {
	f := newTrustFixture(t)
	first, err := f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	nextSigner := newTestSigner(t, "root-next")
	next := f.signerRoot(2, nextSigner, f.now.Add(time.Hour))
	f.responses["/roots/2.root.json"] = signedJSON(t, next, f.rootSigner, nextSigner)
	delete(f.statuses, "/roots/2.root.json")
	f.rootSigner = nextSigner
	f.publish([]byte(`{"plugins":[],"release":2}`), 2, 2, f.plugin)
	second, err := f.client.FetchArtifactForUpdateCheck(context.Background(), PurposePluginIndex, "plugins")
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Descriptor.Version)
	require.True(t, second.Fresh)
	require.True(t, second.Proof.Valid(f.now))
	require.False(t, first.Proof.Valid(f.now))
}
