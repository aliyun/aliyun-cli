package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactCommandLineArgs(t *testing.T) {
	args := []string{
		"oss", "cp", "local", "oss://bucket/key",
		"--access-key-id", "fake-ak",
		"--access-key-secret=fake-secret",
		"-t", "fake-token",
		"--proxy-pwd", "fake-password",
		"https://bucket.example.com/key?AccessKeyId=url-ak&Signature=url-signature&other=visible",
	}

	got := strings.Join(redactCommandLineArgs(args), " ")
	for _, secret := range []string{"fake-ak", "fake-secret", "fake-token", "fake-password", "url-ak", "url-signature"} {
		assert.NotContains(t, got, secret)
	}
	assert.Contains(t, got, "other=visible")
	assert.Contains(t, got, "[REDACTED]")
}

func TestCopyCatBodyPreservesBytesAndReturnsWriteError(t *testing.T) {
	payload := []byte{'h', 'e', 'l', 'l', 'o', 0, 0xff}
	var out bytes.Buffer
	require.NoError(t, copyCatBody(&out, bytes.NewReader(payload)))
	assert.Equal(t, payload, out.Bytes())

	want := errors.New("write failed")
	err := copyCatBody(errorWriter{err: want}, strings.NewReader("content"))
	assert.ErrorIs(t, err, want)
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestStripOptionWithValue(t *testing.T) {
	assert.Equal(t,
		[]string{"oss://bucket", "--include=*.txt"},
		stripOptionWithValue([]string{"oss://bucket", "--endpoint=https://example.com", "--include=*.txt"}, "--endpoint"),
	)
	assert.Equal(t,
		[]string{"oss://bucket", "--include", "*.txt"},
		stripOptionWithValue([]string{"oss://bucket", "--endpoint", "https://example.com", "--include", "*.txt"}, "--endpoint"),
	)
}

func TestOssPositionalArgsKeepsOrderAroundFlags(t *testing.T) {
	command := NewCommandBridge(copyCommand.command)
	ctx := bridgeTestContext(command)
	assert.Equal(t,
		[]string{"src/", "oss://bucket/"},
		ossPositionalArgs(ctx, []string{"--recursive", "--include=*.txt", "src/", "oss://bucket/", "--include", "*.log", "--force"}),
	)
}

func TestBridgeValidatesArgumentsBeforeCredentials(t *testing.T) {
	ctx := bridgeTestContext(NewCommandBridge(copyCommand.command))
	err := parseAndRunCommandFromCli(ctx, nil, &copyCommand.command)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs at least 2 arguments")
	assert.NotContains(t, err.Error(), "credential")
}

func TestBridgeRunsHashWithoutCredentials(t *testing.T) {
	originalRun := parseAndRunCommandImpl
	originalArgs := os.Args
	defer func() {
		parseAndRunCommandImpl = originalRun
		os.Args = originalArgs
	}()

	command := NewCommandBridge(hashCommand.command)
	ctx := bridgeTestContext(command)
	var captured []string
	parseAndRunCommandImpl = func() error {
		captured = append([]string(nil), os.Args...)
		return nil
	}

	require.NoError(t, parseAndRunCommandFromCli(ctx, []string{"file.txt", "--type=md5"}, &hashCommand.command))
	assert.Equal(t, []string{"oss", "hash", "file.txt", "--type=md5"}, captured)
}

func bridgeTestContext(command *cli.Command) *cli.Context {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	ctx.SetCommand(command)
	return ctx
}

func TestSyncStopsBeforeDeleteAfterTransferFailure(t *testing.T) {
	var deleteRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<ListBucketResult><Name>review-bucket</Name><Prefix></Prefix><IsTruncated>false</IsTruncated><Contents><Key>old.txt</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>abc</ETag><Size>3</Size><StorageClass>Standard</StorageClass></Contents></ListBucketResult>`)
		case http.MethodPut:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `<Error><Code>InvalidDigest</Code><Message>mock upload failure</Message><RequestId>review-request</RequestId></Error>`)
		case http.MethodPost:
			deleteRequests.Add(1)
			_, _ = io.WriteString(w, `<DeleteResult/>`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	workingDir := t.TempDir()
	sourceDir := filepath.Join(workingDir, "src")
	require.NoError(t, os.Mkdir(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "new.txt"), []byte("new content"), 0600))

	args := []string{
		"oss", "sync", sourceDir + string(os.PathSeparator), "oss://review-bucket/",
		"--delete", "--force", "--retry-times", "1", "--endpoint", server.URL,
		"--access-key-id", "phase-a-ak", "--access-key-secret", "phase-a-secret",
	}
	encodedArgs, err := json.Marshal(args)
	require.NoError(t, err)
	testBinary, err := os.Executable()
	require.NoError(t, err)
	command := exec.Command(testBinary, "-test.run=^TestPhaseAHelperProcess$")
	command.Dir = workingDir
	command.Env = append(os.Environ(),
		"OSS_PHASE_A_HELPER=1",
		"OSS_PHASE_A_ARGS="+string(encodedArgs),
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Zero(t, deleteRequests.Load(), "delete phase must not start after an upload failure")

	reports, globErr := filepath.Glob(filepath.Join(workingDir, DefaultOutputDir, ReportPrefix+"*"+ReportSuffix))
	require.NoError(t, globErr)
	require.NotEmpty(t, reports)
	report, readErr := os.ReadFile(reports[0])
	require.NoError(t, readErr)
	assert.NotContains(t, string(report), "phase-a-ak")
	assert.NotContains(t, string(report), "phase-a-secret")
}

func TestPhaseAHelperProcess(t *testing.T) {
	if os.Getenv("OSS_PHASE_A_HELPER") != "1" {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("OSS_PHASE_A_ARGS")), &args))
	os.Args = args
	require.Error(t, ParseAndRunCommand(), "the mocked upload failure must reach the command result")
}
