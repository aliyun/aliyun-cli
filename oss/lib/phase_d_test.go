package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhaseDValidateOfflineAndUnsupported(t *testing.T) {
	clearEndpointTestEnv(t)
	src := filepath.Join(t.TempDir(), "a file.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0600))
	var output bytes.Buffer
	ctx := cli.NewCommandContext(&output, io.Discard)
	ctx.SetCommand(NewCommandBridge(copyCommand.command))
	require.NoError(t, parseAndRunCommandFromCli(ctx, []string{src, "oss://bucket/key", "--cli-validate", "--cli-output=json"}, &copyCommand.command))
	var p ossPreview
	require.NoError(t, json.Unmarshal(output.Bytes(), &p))
	assert.Equal(t, "validate", p.Mode)
	assert.Equal(t, "none", p.SideEffects)
	require.Len(t, p.Items, 1)
	assert.Nil(t, p.Items[0].Exists)
	for _, args := range [][]string{
		{src, "oss://bucket/", "--cli-plan"},
		{src, src, "oss://bucket/key", "--cli-validate"},
		{src, "oss://bucket/key", "-r", "--cli-plan"},
		{src, "oss://bucket/key", "--include=*", "--cli-plan"},
		{src, "oss://bucket/key", "--cli-plan", "--cli-validate"},
		{src, "oss://bucket/key", "--cli-validate", "--cli-failure-report=x"},
	} {
		require.Error(t, parseAndRunCommandFromCli(ctx, args, &copyCommand.command), "%v", args)
	}
}

func TestPhaseDPlanNeverWrites(t *testing.T) {
	clearEndpointTestEnv(t)
	src := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(src, []byte("unchanged"), 0600))
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, http.MethodHead, r.Method)
		assert.Equal(t, "/bucket/key", r.URL.Path)
		w.Header().Set("ETag", `"existing"`)
		w.Header().Set("Content-Length", "9")
	}))
	defer server.Close()
	for _, tc := range []struct {
		cmd  Command
		args []string
	}{
		{copyCommand.command, []string{src, "oss://bucket/key"}},
		{removeCommand.command, []string{"oss://bucket/key", "--version-id=version one"}},
	} {
		text, err := phaseCInvoke(t, tc.cmd, server.URL, append(tc.args, "--cli-plan", "--cli-output=json")...)
		require.NoError(t, err)
		var p ossPreview
		require.NoError(t, json.Unmarshal([]byte(text), &p))
		require.Len(t, p.Items, 1)
		assert.True(t, *p.Items[0].Exists)
		assert.Equal(t, `"existing"`, p.Items[0].ETag)
	}
	assert.Equal(t, 2, calls)
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", string(data))
	_, err = phaseCInvoke(t, syncCommand.command, server.URL, src, "oss://bucket/", "--delete", "--cli-plan")
	require.Error(t, err)
	assert.Equal(t, 2, calls)
}

type rotatingCredentials struct {
	count int
	fail  error
}

func (p *rotatingCredentials) GetCredential() (*credentials.CredentialModel, error) {
	p.count++
	if p.fail != nil {
		return nil, p.fail
	}
	id, secret, token := fmt.Sprintf("id%d", p.count), fmt.Sprintf("secret%d", p.count), fmt.Sprintf("token%d", p.count)
	return &credentials.CredentialModel{AccessKeyId: &id, AccessKeySecret: &secret, SecurityToken: &token}, nil
}
func TestPhaseDProviderRefreshAndFailure(t *testing.T) {
	var tokens []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens = append(tokens, r.Header.Get("X-Oss-Security-Token"))
		io.WriteString(w, `<ListAllMyBucketsResult/>`)
	}))
	defer server.Close()
	source := &rotatingCredentials{}
	provider := &hostOSSProvider{source: source}
	client, err := oss.New(server.URL, "", "", oss.SetCredentialsProvider(provider))
	require.NoError(t, err)
	_, err = client.ListBuckets()
	require.NoError(t, err)
	_, err = client.ListBuckets()
	require.NoError(t, err)
	cause := errors.New("refresh unavailable")
	source.fail = cause
	_, err = client.ListBuckets()
	require.ErrorIs(t, err, cause)
	assert.False(t, retryableOSS(err, true))
	assert.Equal(t, []string{"token1", "token2"}, tokens)
	assert.NotContains(t, provider.redact("secret1 token2"), "secret1")
	assert.NotContains(t, provider.redact("secret1 token2"), "token2")
}
func TestPhaseDRetryClassificationAndBudget(t *testing.T) {
	old := retrySleep
	defer func() { retrySleep = old }()
	var delays []time.Duration
	retrySleep = func(d time.Duration) { delays = append(delays, d) }
	for _, code := range []int{400, 403, 404} {
		assert.False(t, retryOSS(ObjectError{oss.ServiceError{StatusCode: code}, "bucket", "key"}, 1, 10, true))
	}
	assert.False(t, retryOSS(oss.ServiceError{StatusCode: 503}, 1, 10, false))
	assert.False(t, retryOSS(io.ErrUnexpectedEOF, 1, 10, false))
	assert.False(t, retryOSS(os.ErrPermission, 1, 10, true))
	assert.True(t, retryOSS(oss.ServiceError{StatusCode: 429}, 1, 3, false))
	assert.True(t, retryOSS(oss.ServiceError{StatusCode: 503}, 2, 3, true))
	assert.False(t, retryOSS(oss.ServiceError{StatusCode: 503}, 3, 3, true))
	require.Len(t, delays, 2)
	assert.GreaterOrEqual(t, delays[0], 100*time.Millisecond)
	assert.LessOrEqual(t, delays[1], 400*time.Millisecond)
}
func TestPhaseDFailureManifestConcurrentAndExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failures.jsonl")
	manifest, err := newFailureManifest(path)
	require.NoError(t, err)
	_, err = newFailureManifest(path)
	require.Error(t, err)
	var workers sync.WaitGroup
	for i := 0; i < 30; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			manifest.write(failureItem{Type: "failed_item", Source: fmt.Sprintf("source %d\nline", i), Target: "oss://bucket/key", VersionID: "v1", Outcome: "unknown"})
		}(i)
	}
	workers.Wait()
	require.NoError(t, manifest.finish(nil))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 31)
	for _, line := range lines {
		var item failureItem
		require.NoError(t, json.Unmarshal([]byte(line), &item))
		assert.False(t, item.AutomaticRetry)
	}
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
func TestPhaseDFailedUploadManifest(t *testing.T) {
	clearEndpointTestEnv(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "source with spaces")
	report := filepath.Join(dir, "failures.jsonl")
	require.NoError(t, os.WriteFile(src, []byte("payload"), 0600))
	puts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(503)
		io.WriteString(w, `<Error><Code>ServiceUnavailable</Code><Message>fake-secret</Message></Error>`)
	}))
	defer server.Close()
	_, err := phaseCInvoke(t, copyCommand.command, server.URL, src, "oss://bucket/key", "-f", "--retry-times=3", "--cli-failure-report", report, "--output-dir", filepath.Join(dir, "legacy"))
	require.Error(t, err)
	assert.Equal(t, 1, puts)
	data, err := os.ReadFile(report)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "fake-secret")
	var found bool
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var item failureItem
		require.NoError(t, json.Unmarshal([]byte(line), &item))
		if item.Type == "failed_item" {
			found = true
			assert.Equal(t, src, item.Source)
			assert.Equal(t, "oss://bucket/key", item.Target)
		}
	}
	assert.True(t, found)
}
func TestPhaseDReporterNamesAndCleanup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reports")
	var a, b Reporter
	require.NoError(t, a.Init(dir, "a"))
	require.NoError(t, b.Init(dir, "b"))
	assert.NotEqual(t, a.path, b.path)
	b.ReportError("keep this report")
	a.Clear()
	b.Clear()
	_, err := os.Stat(b.path)
	require.NoError(t, err)
}

func TestPhaseDBatchFailureJoinsWorkersAndBlocksDelete(t *testing.T) {
	clearEndpointTestEnv(t)
	root := t.TempDir()
	src := filepath.Join(root, "src")
	require.NoError(t, os.Mkdir(src, 0700))
	for i := 0; i < 12; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(src, fmt.Sprintf("file%d", i)), []byte("data"), 0600))
	}
	var puts, deletes atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>old</Key><Size>1</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
		case http.MethodPut:
			puts.Add(1)
			w.WriteHeader(400)
			io.WriteString(w, `<Error><Code>InvalidDigest</Code><Message>failed</Message></Error>`)
		default:
			deletes.Add(1)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	report := filepath.Join(root, "failures.jsonl")
	_, err := phaseCInvoke(t, syncCommand.command, server.URL, src+string(os.PathSeparator), "oss://bucket/", "--delete", "-f", "--jobs=2", "--cli-failure-report", report, "--output-dir", filepath.Join(root, "legacy"))
	require.Error(t, err)
	assert.Zero(t, deletes.Load())
	data, err := os.ReadFile(report)
	require.NoError(t, err)
	var items int64
	var last failureItem
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		require.NoError(t, json.Unmarshal([]byte(line), &last))
		if last.Type == "failed_item" {
			items++
		}
	}
	assert.Positive(t, items)
	assert.Equal(t, puts.Load(), items)
	assert.Equal(t, "summary", last.Type)
	assert.Equal(t, "failed", last.Outcome)
	assert.Equal(t, "not_started", last.DeletePhase)
	assert.Nil(t, bridgeCredentialProvider)
}
