package agentbay

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

type agentBodyFailure struct{}

func (agentBodyFailure) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (agentBodyFailure) Close() error             { return nil }

type agentTransport func(*http.Request) (*http.Response, error)

func (f agentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAgentBayRequestBoundaries(t *testing.T) {
	oldGet, oldTransport, oldURL := httpGetFunc, http.DefaultTransport, versionManifestURL
	t.Cleanup(func() { httpGetFunc = oldGet; http.DefaultTransport = oldTransport; versionManifestURL = oldURL })
	httpGetFunc = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: agentBodyFailure{}}, nil
	}
	if err := downloadFile("https://example.invalid/binary", filepath.Join(t.TempDir(), "file")); err == nil || !strings.Contains(err.Error(), "body failed") {
		t.Fatalf("read failure=%v", err)
	}
	httpGetFunc = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("binary"))}, nil
	}
	if err := downloadFile("https://example.invalid/binary", filepath.Join(t.TempDir(), "missing", "file")); err == nil {
		t.Fatal("missing parent accepted")
	}
	http.DefaultTransport = agentTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: agentBodyFailure{}}, nil
	})
	if _, err := GetLatestAgentBayVersion(); err == nil || !strings.Contains(err.Error(), "body failed") {
		t.Fatalf("version read failure=%v", err)
	}
	versionManifestURL = "://invalid"
	if _, err := GetLatestAgentBayVersion(); err == nil {
		t.Fatal("invalid version URL accepted")
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	if _, err := agentBayCredentialEnvFromProfile(ctx, config.Profile{Mode: config.StsToken, AccessKeyId: "id", AccessKeySecret: "secret"}); err == nil {
		t.Fatal("STS credentials without token accepted")
	}
	if _, err := agentBayCredentialEnvFromProfile(ctx, config.Profile{Mode: config.External}); err == nil {
		t.Fatal("invalid external credentials accepted")
	}
}
