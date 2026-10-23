package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePluginArtifactFetcher struct {
	artifact *trust.VerifiedArtifact
	err      error
	calls    int
}

func (f *fakePluginArtifactFetcher) FetchArtifact(_ context.Context, purpose, role string) (*trust.VerifiedArtifact, error) {
	f.calls++
	if purpose != trust.PurposePluginIndex || role != "plugins" {
		return nil, fmt.Errorf("unexpected trust request %s/%s", purpose, role)
	}
	return f.artifact, f.err
}

func usePluginTrustIndex(t *testing.T, index *Index) *fakePluginArtifactFetcher {
	t.Helper()
	payload, err := json.Marshal(index)
	require.NoError(t, err)
	fetcher := &fakePluginArtifactFetcher{artifact: &trust.VerifiedArtifact{
		Bytes: payload,
		Descriptor: trust.ArtifactDescriptor{
			Version: 42,
		},
		KeyIDs:   []string{"plugins-2026"},
		SignedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}}
	original := newArtifactTrustClient
	newArtifactTrustClient = func(string, string) artifactFetcher { return fetcher }
	t.Cleanup(func() { newArtifactTrustClient = original })
	return fetcher
}

func TestOfficialIndexUsesTrustClientWithoutUnsignedFallback(t *testing.T) {
	index := &Index{Plugins: []PluginInfo{{Name: "aliyun-cli-fc"}}}
	fetcher := usePluginTrustIndex(t, index)
	manager := &Manager{rootDir: t.TempDir(), trustClient: fetcher}

	got, err := manager.GetIndex()
	require.NoError(t, err)
	assert.Equal(t, "aliyun-cli-fc", got.Plugins[0].Name)
	assert.Equal(t, 1, fetcher.calls)
	require.NotNil(t, manager.indexTrustVerification)
	assert.Equal(t, uint64(42), manager.indexTrustVerification.IndexVersion)

	fetcher.err = fmt.Errorf("bad signature")
	_, err = manager.GetIndex()
	assert.ErrorContains(t, err, "bad signature")
}

func TestSourceBaseReconfiguresRepositoryTrustClient(t *testing.T) {
	var capturedSource string
	fetcher := &fakePluginArtifactFetcher{err: fmt.Errorf("not configured")}
	original := newArtifactTrustClient
	newArtifactTrustClient = func(_ string, sourceBase string) artifactFetcher {
		capturedSource = sourceBase
		return fetcher
	}
	t.Cleanup(func() { newArtifactTrustClient = original })

	manager := &Manager{rootDir: t.TempDir(), trustCacheDir: t.TempDir()}
	require.NoError(t, manager.ApplySourceBaseOverride(
		"https://cli-inner.example.com/pre/plugins-v2/",
	))
	assert.Equal(
		t,
		"https://cli-inner.example.com/pre/plugins-v2",
		capturedSource,
	)

	_, err := manager.GetIndex()
	assert.ErrorContains(t, err, "failed to verify plugin index")
}

func TestPluginAuditFieldsRoundTripAndDisplay(t *testing.T) {
	verification := &TrustVerification{
		KeyIDs: []string{"key-a", "key-b"}, IndexVersion: 9,
		Checksum: "abc", VerifiedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}
	manifest := LocalManifest{Plugins: map[string]LocalPlugin{
		"aliyun-cli-fc": {
			Name: "aliyun-cli-fc", Version: "1.0.0", TrustVerification: verification,
		},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	var decoded LocalManifest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, verification, decoded.Plugins["aliyun-cli-fc"].TrustVerification)
}
