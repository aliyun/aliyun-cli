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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func phaseCInvoke(t *testing.T, cmd Command, endpoint string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	ctx.SetCommand(NewCommandBridge(cmd))
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fake-ak","access_key_secret":"fake-secret","region_id":"cn-hangzhou"}]}`), 0600))
	flags := []string{"--config-path", path, "--access-key-id=fake-ak", "--access-key-secret=fake-secret", "--region=cn-hangzhou", "--retry-count=1"}
	if endpoint != "" {
		flags = append(flags, "--endpoint", endpoint)
	}
	err := parseAndRunCommandFromCli(ctx, append(flags, args...), &cmd)
	return out.String(), err
}
func TestPhaseCObjectPagination(t *testing.T) {
	clearEndpointTestEnv(t)
	var markers []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markers = append(markers, r.URL.Query().Get("marker"))
		assert.Equal(t, "1", r.URL.Query().Get("max-keys"))
		io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>a</Key><Size>0</Size></Contents><Contents><Key>b</Key><Size>2</Size></Contents><CommonPrefixes><Prefix>dir/</Prefix></CommonPrefixes></ListBucketResult>`)
	}))
	defer server.Close()
	var keys []string
	cursor := ""
	for i := 0; i < 3; i++ {
		args := []string{"oss://bucket/", "-d", "--cli-output=json", "--limited-num=1"}
		if cursor != "" {
			args = append(args, "--cli-cursor", cursor)
		}
		text, err := phaseCInvoke(t, listCommand.command, server.URL, args...)
		require.NoError(t, err)
		var result machineListResult
		require.NoError(t, json.Unmarshal([]byte(text), &result))
		require.Len(t, result.Items, 1)
		keys = append(keys, result.Items[0].Key)
		cursor = result.NextCursor
		assert.Equal(t, i == 2, result.Complete)
		assert.Equal(t, i != 2, result.Truncated)
	}
	assert.Equal(t, []string{"a", "b", "dir/"}, keys)
	assert.Equal(t, []string{"", "", ""}, markers)
}
func TestPhaseCVersionAndUploadCursors(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, tc := range []struct{ flag, body, param string }{
		{"--all-versions", `<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>key</NextKeyMarker><NextVersionIdMarker>v2</NextVersionIdMarker><DeleteMarker><Key>key</Key><VersionId>v1</VersionId></DeleteMarker></ListVersionsResult>`, "version-id-marker"},
		{"-m", `<ListMultipartUploadsResult><IsTruncated>true</IsTruncated><NextKeyMarker>key</NextKeyMarker><NextUploadIdMarker>v2</NextUploadIdMarker><Upload><Key>key</Key><UploadId>v1</UploadId></Upload></ListMultipartUploadsResult>`, "upload-id-marker"},
	} {
		t.Run(tc.param, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 2 {
					assert.Equal(t, "key", r.URL.Query().Get("key-marker"))
					assert.Equal(t, "v2", r.URL.Query().Get(tc.param))
					root := "ListVersionsResult"
					if tc.param == "upload-id-marker" {
						root = "ListMultipartUploadsResult"
					}
					fmt.Fprintf(w, "<%s><IsTruncated>false</IsTruncated></%s>", root, root)
				} else {
					io.WriteString(w, tc.body)
				}
			}))
			defer server.Close()
			args := []string{"oss://bucket", tc.flag, "--cli-output=json", "--limited-num=1"}
			text, err := phaseCInvoke(t, listCommand.command, server.URL, args...)
			require.NoError(t, err)
			var result machineListResult
			require.NoError(t, json.Unmarshal([]byte(text), &result))
			require.NotEmpty(t, result.NextCursor)
			_, err = phaseCInvoke(t, listCommand.command, server.URL, append(args, "--cli-cursor", result.NextCursor)...)
			require.NoError(t, err)
			assert.Equal(t, 2, calls)
		})
	}
}
func TestPhaseCJSONLFilterAndEmptyPage(t *testing.T) {
	clearEndpointTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "a" {
			io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>b.txt</Key></Contents></ListBucketResult>`)
		} else {
			io.WriteString(w, `<ListBucketResult><IsTruncated>true</IsTruncated><NextMarker>a</NextMarker><Contents><Key>a.bin</Key></Contents></ListBucketResult>`)
		}
	}))
	defer server.Close()
	args := []string{"oss://bucket", "--cli-output=jsonl", "--exclude=*", "--include=*.txt"}
	text, err := phaseCInvoke(t, listCommand.command, server.URL, args...)
	require.NoError(t, err)
	var summary struct {
		Type       string
		Returned   int
		Complete   bool
		NextCursor string `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &summary))
	assert.Equal(t, "summary", summary.Type)
	assert.Zero(t, summary.Returned)
	assert.False(t, summary.Complete)
	require.NotEmpty(t, summary.NextCursor)
	text, err = phaseCInvoke(t, listCommand.command, server.URL, append(args, "--cli-cursor", summary.NextCursor)...)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	require.Len(t, lines, 2)
	var item map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &item))
	assert.Equal(t, "item", item["type"])
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &summary))
	assert.True(t, summary.Complete)
}
func TestPhaseCCursorSafety(t *testing.T) {
	clearEndpointTestEnv(t)
	changed := false
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		key := "a"
		if changed {
			key = "changed"
		}
		fmt.Fprintf(w, `<ListBucketResult><Contents><Key>%s</Key></Contents><Contents><Key>b</Key></Contents></ListBucketResult>`, key)
	}))
	defer server.Close()
	args := []string{"oss://bucket", "--cli-output=json", "--limited-num=1"}
	text, err := phaseCInvoke(t, listCommand.command, server.URL, args...)
	require.NoError(t, err)
	var result machineListResult
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	badArgs := []string{"oss://other", "--cli-output=json", "--cli-cursor", result.NextCursor}
	_, err = phaseCInvoke(t, listCommand.command, server.URL, badArgs...)
	require.ErrorContains(t, err, "different list query")
	assert.Equal(t, 1, calls)
	changed = true
	text, err = phaseCInvoke(t, listCommand.command, server.URL, append(args, "--cli-cursor", result.NextCursor)...)
	require.ErrorContains(t, err, "page changed")
	assert.Empty(t, text)
}
func TestPhaseCErrorsAndModes(t *testing.T) {
	clearEndpointTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Oss-Request-Id", "request")
		w.WriteHeader(403)
		io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>denied fake-secret</Message><RequestId>request</RequestId></Error>`)
	}))
	defer server.Close()
	out, err := phaseCInvoke(t, listCommand.command, server.URL, "oss://bucket", "--cli-ai-mode")
	require.Empty(t, out)
	var structured *ossAgentError
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, 1, structured.ExitCode())
	assert.Equal(t, "AccessDenied", structured.Envelope().ErrorCode)
	assert.Equal(t, "request", structured.Envelope().RequestId)
	assert.NotContains(t, err.Error(), "fake-secret")
	assert.Equal(t, "check_permissions", structured.Envelope().Recovery.Action)
	var service oss.ServiceError
	require.ErrorAs(t, err, &service)
	var encoded bytes.Buffer
	require.NoError(t, structured.RenderError(&encoded))
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(encoded.Bytes(), &decoded))
	assert.Contains(t, decoded, "oss")
	for _, args := range [][]string{{"--unknown", "--cli-output=json"}, {"--limited-num=0", "--cli-output=json"}, {"--cli-output=json", "--cli-dry-run"}} {
		_, err := phaseCInvoke(t, listCommand.command, "", args...)
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, "validation", structured.facts.Phase)
	}
	t.Setenv("ALIBABA_CLOUD_CLI_AI_MODE", "1")
	_, err = phaseCInvoke(t, copyCommand.command, "", "--cli-output=json")
	require.ErrorAs(t, err, &structured)
	_, err = phaseCInvoke(t, copyCommand.command, "", "--no-cli-ai-mode")
	assert.False(t, errors.As(err, &structured))
}
func TestPhaseCNonInteractiveAndRawCat(t *testing.T) {
	clearEndpointTestEnv(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "DELETE" {
			t.Error("unexpected deletion")
		}
		io.WriteString(w, "hello\x00")
	}))
	defer server.Close()
	_, err := phaseCInvoke(t, removeCommand.command, server.URL, "oss://bucket", "-r", "-a", "--cli-non-interactive")
	require.ErrorIs(t, err, errConfirmationRequired)
	assert.Zero(t, calls)
	assert.Nil(t, activeMachine)
	// A held-open stdin must never be read.
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	defer reader.Close()
	defer writer.Close()
	old := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = old }()
	activeMachine = &machineInvocation{nonInteractive: true}
	var value string
	_, err = scanOSSInput(&value)
	require.ErrorIs(t, err, errConfirmationRequired)
	activeMachine = nil
	// cat uses the raw process stdout, not the JSON renderer.
	outFile, err := os.CreateTemp(t.TempDir(), "cat")
	require.NoError(t, err)
	defer outFile.Close()
	stdout := os.Stdout
	os.Stdout = outFile
	_, err = phaseCInvoke(t, catCommand.command, server.URL, "oss://bucket/key", "--cli-output=json")
	os.Stdout = stdout
	require.NoError(t, err)
	data, err := os.ReadFile(outFile.Name())
	require.NoError(t, err)
	assert.Equal(t, []byte("hello\x00"), data)
}

func TestPhaseCCommandEntryRendersOnceAndRestoresState(t *testing.T) {
	clearEndpointTestEnv(t)
	cli.DisableExitCode()
	defer cli.EnableExitCode()
	for _, args := range [][]string{
		{"cp", "--cli-output=json"},
		{"--cli-output=json", "cp"},
		{"cp", "--unknown", "--cli-ai-mode"},
	} {
		var out, errOut bytes.Buffer
		root := NewOssCommand()
		ctx := cli.NewCommandContext(&out, &errOut)
		ctx.SetCommand(root)
		root.Execute(ctx, args)
		require.Empty(t, out.String())
		var envelope cli.AgentErrorEnvelope
		require.NoError(t, json.Unmarshal(errOut.Bytes(), &envelope), "%s", errOut.String())
		assert.Equal(t, "InvalidArgument", envelope.ErrorCode)
		assert.Nil(t, activeMachine)
	}
	// The same context must not retain AI flags from a previous call.
	var out, errOut bytes.Buffer
	ctx := cli.NewCommandContext(&out, &errOut)
	ctx.SetCommand(NewCommandBridge(copyCommand.command))
	err := parseAndRunCommandFromCli(ctx, []string{"--cli-ai-mode"}, &copyCommand.command)
	var machine *ossAgentError
	require.ErrorAs(t, err, &machine)
	err = parseAndRunCommandFromCli(ctx, nil, &copyCommand.command)
	assert.False(t, errors.As(err, &machine))
}

func TestPhaseCBucketsAndAllTypes(t *testing.T) {
	clearEndpointTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "uploads") {
			io.WriteString(w, `<ListMultipartUploadsResult><Upload><Key>b</Key><UploadId>upload</UploadId></Upload></ListMultipartUploadsResult>`)
		} else if strings.Contains(r.Host, "bucket.") || strings.HasPrefix(r.URL.Path, "/bucket") {
			io.WriteString(w, `<ListBucketResult><Contents><Key>a</Key></Contents></ListBucketResult>`)
		} else {
			assert.Equal(t, "100", r.URL.Query().Get("max-keys"))
			io.WriteString(w, `<ListAllMyBucketsResult><Buckets><Bucket><Name>bucket</Name><Location>oss-cn-hangzhou</Location></Bucket></Buckets></ListAllMyBucketsResult>`)
		}
	}))
	defer server.Close()
	text, err := phaseCInvoke(t, listCommand.command, server.URL, "--cli-output=json")
	require.NoError(t, err)
	var result machineListResult
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	require.Len(t, result.Items, 1)
	assert.Equal(t, "bucket", result.Items[0].Kind)
	assert.True(t, result.Complete)
	args := []string{"oss://bucket", "-a", "--cli-output=json"}
	text, err = phaseCInvoke(t, listCommand.command, server.URL, args...)
	require.NoError(t, err)
	result = machineListResult{}
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	require.Len(t, result.Items, 1)
	assert.Equal(t, "object", result.Items[0].Kind)
	require.NotEmpty(t, result.NextCursor)
	text, err = phaseCInvoke(t, listCommand.command, server.URL, append(args, "--cli-cursor", result.NextCursor)...)
	require.NoError(t, err)
	result = machineListResult{}
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	require.Len(t, result.Items, 1)
	assert.Equal(t, "upload", result.Items[0].Kind)
	assert.True(t, result.Complete)
}

func TestPhaseCWriterFailureAndRecoveryClassification(t *testing.T) {
	for _, tc := range []struct {
		code   string
		status int
		action string
	}{
		{"NoSuchKey", 404, "check_resource"}, {"SecurityTokenExpired", 403, "refresh_credentials"},
		{"PermanentRedirect", 301, "check_endpoint"}, {"SlowDown", 429, "check_retry_safety"}, {"InternalError", 500, "check_retry_safety"},
	} {
		for _, command := range []string{"ls", "cp"} {
			m := machineInvocation{ai: true, phase: "execution", command: command}
			err := m.adaptError(ObjectError{err: oss.ServiceError{Code: tc.code, StatusCode: tc.status}, bucket: "bucket", object: "key"})
			var e *ossAgentError
			require.ErrorAs(t, err, &e)
			assert.Equal(t, tc.action, e.Envelope().Recovery.Action)
			if tc.status >= 429 {
				assert.Equal(t, command == "ls", e.facts.Retryable)
			}
			if command == "cp" {
				assert.Equal(t, "unknown", e.facts.SideEffects)
			}
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `<ListAllMyBucketsResult/>`) }))
	defer server.Close()
	_, opts, err := parseOSSOptions([]string{"--endpoint", server.URL, "-i", "fake-ak", "-k", "fake-secret"})
	require.NoError(t, err)
	lc := ListCommand{command: listCommand.command}
	require.NoError(t, lc.Init(nil, opts))
	err = lc.listMachine(&machineInvocation{format: "json", writer: phaseCFailWriter{}})
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

type phaseCFailWriter struct{}

func (phaseCFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestPhaseCProcessExit(t *testing.T) {
	if os.Getenv("OSS_PHASE_C_HELPER") == "1" {
		cli.EnableExitCode()
		root := NewOssCommand()
		ctx := cli.NewCommandContext(os.Stdout, os.Stderr)
		ctx.SetCommand(root)
		root.Execute(ctx, []string{"cp", "--cli-output=json"})
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPhaseCProcessExit$")
	cmd.Env = append(os.Environ(), "OSS_PHASE_C_HELPER=1")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	assert.Equal(t, 1, exit.ExitCode())
	assert.Empty(t, out.String())
	var envelope cli.AgentErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	assert.Equal(t, "InvalidArgument", envelope.ErrorCode)
}

func TestPhaseCForceAndInheritedDryRun(t *testing.T) {
	clearEndpointTestEnv(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "DELETE" {
			w.WriteHeader(204)
		} else {
			io.WriteString(w, `<ListBucketResult/>`)
		}
	}))
	defer server.Close()
	_, err := phaseCInvoke(t, removeCommand.command, server.URL, "oss://bucket/key", "-f", "--cli-non-interactive")
	require.NoError(t, err)
	assert.Positive(t, calls)
	ctx := bridgeTestContext(NewCommandBridge(copyCommand.command))
	flag := &cli.Flag{Name: "cli-dry-run", AssignedMode: cli.AssignedNone}
	flag.SetAssigned(true)
	ctx.Flags().Add(flag)
	err = parseAndRunCommandFromCli(ctx, []string{"source", "oss://bucket/key", "--cli-ai-mode"}, &copyCommand.command)
	require.ErrorContains(t, err, "no operation was executed")
}

func TestPhaseCVersionPageOffset(t *testing.T) {
	clearEndpointTestEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<ListVersionsResult><DeleteMarker><Key>a</Key><VersionId>deleted</VersionId></DeleteMarker><Version><Key>a</Key><VersionId>older</VersionId><Size>1</Size></Version></ListVersionsResult>`)
	}))
	defer server.Close()
	cursor := ""
	var versions []string
	for i := 0; i < 2; i++ {
		args := []string{"oss://bucket", "--all-versions", "--limited-num=1", "--cli-output=json"}
		if cursor != "" {
			args = append(args, "--cli-cursor", cursor)
		}
		text, err := phaseCInvoke(t, listCommand.command, server.URL, args...)
		require.NoError(t, err)
		var result machineListResult
		require.NoError(t, json.Unmarshal([]byte(text), &result))
		require.Len(t, result.Items, 1)
		versions = append(versions, result.Items[0].VersionID)
		cursor = result.NextCursor
	}
	assert.Empty(t, cursor)
	assert.Equal(t, []string{"deleted", "older"}, versions)
}
