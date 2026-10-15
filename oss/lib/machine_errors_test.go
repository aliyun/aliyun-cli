package lib

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/require"
)

func TestPreviewRejectsInvalidInputs(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0600))
	for _, tc := range []struct {
		name string
		cmd  *Command
		args []string
		want string
	}{
		{"missing command", nil, nil, "subcommand"},
		{"missing upload args", &copyCommand.command, nil, "exactly one"},
		{"cloud source", &copyCommand.command, []string{"oss://bucket/a", "oss://bucket/b"}, "single local"},
		{"implicit target", &copyCommand.command, []string{src, "local"}, "explicit oss://"},
		{"invalid bucket", &copyCommand.command, []string{src, "oss:///key"}, "bucket"},
		{"directory key", &copyCommand.command, []string{src, "oss://bucket/dir/"}, "trailing slash"},
		{"missing source", &copyCommand.command, []string{src + "missing", "oss://bucket/key"}, ""},
		{"local directory", &copyCommand.command, []string{filepath.Dir(src), "oss://bucket/key"}, "regular local file"},
		{"delete count", &removeCommand.command, nil, "exactly one"},
		{"delete target", &removeCommand.command, []string{"local"}, "explicit oss://"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := preparePreview(tc.cmd, tc.args, nil, nil, "validate")
			require.Error(t, err)
			require.Nil(t, p)
			if tc.want != "" {
				require.Contains(t, err.Error(), tc.want)
			}
		})
	}
	version := "v1"
	p, err := preparePreview(&removeCommand.command, []string{"oss://bucket/key"}, []string{"--version-id", version}, OptionMapType{OptionVersionId: &version}, "validate")
	require.NoError(t, err)
	require.Equal(t, version, p.Items[0].VersionID)
}

func TestPlanMetadataErrorsNeverWrite(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, tc := range []struct {
		code    string
		status  int
		success bool
	}{{"NoSuchKey", 404, true}, {"NoSuchVersion", 404, true}, {"AccessDenied", 403, false}, {"NoSuchBucket", 404, false}} {
		t.Run(tc.code, func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "HEAD" {
					t.Errorf("unexpected write method: %s", r.Method)
				}
				w.Header().Set(oss.HTTPHeaderOssErr, base64.StdEncoding.EncodeToString([]byte("<Error><Code>"+tc.code+"</Code></Error>")))
				w.WriteHeader(tc.status)
			}))
			defer s.Close()
			out, err := phaseCInvoke(t, removeCommand.command, s.URL, "oss://bucket/key", "--cli-plan", "--version-id=v1")
			if tc.success {
				require.NoError(t, err)
				var p ossPreview
				require.NoError(t, json.Unmarshal([]byte(out), &p))
				require.NotNil(t, p.Items[0].Exists)
				require.False(t, *p.Items[0].Exists)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, 1, calls)
		})
	}
}

func TestMachineValidationAndNetworkErrors(t *testing.T) {
	for _, m := range []*machineInvocation{{format: "xml"}, {cursor: "x", command: "ls"}, {command: "config", nonInteractive: true}} {
		require.Error(t, m.validate(nil))
	}
	for _, cmd := range []string{"ls", "cp"} {
		m := machineInvocation{format: "json", command: cmd, phase: "execution"}
		err := m.adaptError(&net.DNSError{Err: "timeout", Name: "example.invalid", IsTimeout: true})
		var ae *ossAgentError
		require.ErrorAs(t, err, &ae)
		require.Equal(t, cmd == "ls", ae.facts.Retryable)
		require.Equal(t, "check_connection_and_state", ae.Envelope().Recovery.Action)
		err = m.adaptError(&oss.ServiceError{StatusCode: 503, Code: "ServiceUnavailable"})
		require.ErrorAs(t, err, &ae)
		require.Equal(t, cmd == "ls", ae.facts.Retryable)
	}
	m := machineInvocation{format: "json", command: "rm", phase: "policy"}
	var ae *ossAgentError
	require.ErrorAs(t, m.adaptError(errConfirmationRequired), &ae)
	require.Equal(t, "ConfirmationRequired", ae.Envelope().ErrorCode)
	require.Contains(t, ae.Envelope().Recovery.Hint, "--yes")
}

type fixedCredentialSource struct{ model *credentials.CredentialModel }

func (s fixedCredentialSource) GetCredential() (*credentials.CredentialModel, error) {
	return s.model, nil
}
func TestCredentialSnapshotAndIncompleteRefresh(t *testing.T) {
	id, secret := "id", "secret"
	for _, model := range []*credentials.CredentialModel{nil, {}, {AccessKeyId: &id}, {AccessKeyId: new(string), AccessKeySecret: &secret}} {
		p := &hostOSSProvider{source: fixedCredentialSource{model}, current: credentialSnapshot{id, secret, "token"}}
		c, err := p.GetCredentialsE()
		require.ErrorContains(t, err, "incomplete credentials")
		require.Nil(t, c)
		require.Equal(t, "token", p.GetCredentials().GetSecurityToken())
	}
	p := &hostOSSProvider{source: fixedCredentialSource{&credentials.CredentialModel{AccessKeyId: &id, AccessKeySecret: &secret}}}
	for i := 0; i < 2; i++ {
		c, err := p.GetCredentialsE()
		require.NoError(t, err)
		require.Equal(t, id, c.GetAccessKeyID())
		require.Empty(t, c.GetSecurityToken())
	}
	require.Len(t, p.secrets, 2)
	cause := errors.New("failed")
	err := &credentialRefreshError{cause}
	require.Contains(t, err.Error(), "failed")
	require.ErrorIs(t, err, cause)
	for _, attempt := range []int{-1, 99} {
		d := retryDelay(attempt)
		require.GreaterOrEqual(t, d, 100*time.Millisecond)
		require.LessOrEqual(t, d, 6400*time.Millisecond)
	}
	require.True(t, retryableOSS(&oss.ServiceError{StatusCode: 429}, false))
	require.False(t, retryableOSS(&oss.ServiceError{StatusCode: 503}, false))
}

func TestMachineListErrorResponses(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, args := range [][]string{{}, {"oss://bucket"}, {"oss://bucket", "--all-versions"}, {"oss://bucket", "-m"}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
		}))
		_, err := phaseCInvoke(t, listCommand.command, s.URL, append(args, "--cli-output=json")...)
		s.Close()
		require.ErrorContains(t, err, "AccessDenied")
	}
	for _, args := range [][]string{{"oss://bucket", "--request-payer=bad"}, {"oss://bucket", "--include=a/b"}, {"oss://bucket", "--encoding-type=url", "--marker=%xx"}, {"oss://bucket", "--encoding-type=url", "--version-id-marker=%xx"}, {"--all-versions"}} {
		_, err := phaseCInvoke(t, listCommand.command, "http://127.0.0.1:1", append(args, "--cli-output=json")...)
		require.Error(t, err)
	}
	_, err := decodeListCursor(strings.Repeat("x", 32769), "q", 1)
	require.Error(t, err)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<ListBucketResult><IsTruncated>true</IsTruncated></ListBucketResult>`)
	}))
	defer s.Close()
	_, err = phaseCInvoke(t, listCommand.command, s.URL, "oss://bucket", "--cli-output=json")
	require.ErrorContains(t, err, "non-advancing")
}

func TestBridgeRejectsInvalidMachineInvocations(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, tc := range []struct {
		cmd  Command
		args []string
		want string
	}{
		{copyCommand.command, []string{"a", "oss://bucket/b", "--cli-failure-report="}, "nonempty path"},
		{listCommand.command, []string{"--retry-count=-1"}, "non-negative"},
		{listCommand.command, []string{"--read-timeout=bad"}, "not int64"},
		{listCommand.command, []string{"--connect-timeout=-1"}, "non-negative"},
		{listCommand.command, []string{"--cli-output=json", "--cli-cursor=x"}, "cursor"},
		{listCommand.command, []string{"a", "b"}, "at most"},
		{copyCommand.command, []string{}, "at least"},
		{copyCommand.command, []string{"a", "oss://bucket/b", "--dryrun"}, "unknown OSS option"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			_, err := phaseCInvoke(t, tc.cmd, "http://127.0.0.1:1", tc.args...)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestMachineListRejectsInvalidStateAndOutputFailures(t *testing.T) {
	clearEndpointTestEnv(t)
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	for _, tc := range []struct {
		args             []string
		key, value, want string
	}{
		{[]string{"oss://bucket"}, OptionRequestPayer, "invalid", "request payer"},
		{[]string{"oss://bucket"}, OptionLimitedNum, "0", "limited-num"},
		{[]string{"oss://"}, "", "", ""},
	} {
		_, opts, err := parseOSSOptions(nil)
		require.NoError(t, err)
		if tc.key != "" {
			v := tc.value
			opts[tc.key] = &v
		}
		lc := ListCommand{command: listCommand.command}
		lc.command.args = tc.args
		lc.command.options = opts
		os.Args = []string{"oss", "ls"}
		err = lc.listMachine(&machineInvocation{format: "json", writer: io.Discard})
		require.Error(t, err)
		if tc.want != "" {
			require.Contains(t, err.Error(), tc.want)
		}
	}
	for _, stage := range []string{"buckets", "objects"} {
		_, opts, err := parseOSSOptions(nil)
		require.NoError(t, err)
		endpoint := "://invalid"
		opts[OptionEndpoint] = &endpoint
		lc := ListCommand{command: listCommand.command}
		lc.command.options = opts
		_, _, _, _, err = lc.machinePage(CloudURL{bucket: "bucket"}, stage, false, listCursor{PageSize: 1})
		require.Error(t, err)
	}
}

func TestPreviewConfigurationAndURIErrorBoundaries(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, target := range []string{"oss://bucket", "oss://bucket/"} {
		_, err := preparePreview(&removeCommand.command, []string{target}, nil, nil, "validate")
		require.Error(t, err)
	}
	_, err := preparePreview(&removeCommand.command, []string{"oss://bucket/key"}, []string{"--version-id=v1"}, nil, "validate")
	require.NoError(t, err)
	for _, endpoint := range []string{"://invalid", "http://127.0.0.1:1"} {
		_, opts, err := parseOSSOptions(nil)
		require.NoError(t, err)
		opts[OptionEndpoint] = &endpoint
		cmd := removeCommand.command
		p := &ossPreview{Mode: "plan", Items: []previewItem{{Target: "oss://INVALID/key"}}}
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		require.Error(t, p.run(ctx, cmd, opts, nil))
	}
	p := &ossPreview{Mode: "plan"}
	cmd := removeCommand.command
	_, opts, err := parseOSSOptions(nil)
	require.NoError(t, err)
	bad := "invalid"
	opts[OptionRetryTimes] = &bad
	require.Error(t, p.run(cli.NewCommandContext(io.Discard, io.Discard), cmd, opts, nil))
}

func TestMachineErrorIncludesFailedItemReport(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "report"))
	require.NoError(t, err)
	defer file.Close()
	m := &machineInvocation{format: "json", command: "sync", phase: "execution", failures: &failureManifest{file: file, deletePhase: "not_started"}}
	var ae *ossAgentError
	require.ErrorAs(t, m.adaptError(errors.New("transfer failed")), &ae)
	require.Equal(t, file.Name(), ae.facts.FailureReport)
	require.Equal(t, "not_started", ae.facts.DeletePhase)
	require.Equal(t, "inspect_failed_items", ae.Envelope().Recovery.Action)
}
