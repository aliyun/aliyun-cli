package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type bucketConfigurationCase struct {
	name, query, body string
	newCommand        func(Command) Commander
	putArgs           []string
	getArgs           []string
	deleteArgs        []string
	fileInput         bool
}

func bucketConfigurationCases() []bucketConfigurationCase {
	return []bucketConfigurationCase{
		{"cors", "cors", `<CORSConfiguration><CORSRule><AllowedOrigin>https://example.test</AllowedOrigin><AllowedMethod>GET</AllowedMethod></CORSRule></CORSConfiguration>`, func(c Command) Commander { return &CorsCommand{command: c} }, nil, nil, []string{}, true},
		{"lifecycle", "lifecycle", `<LifecycleConfiguration><Rule><ID>logs</ID><Prefix>logs/</Prefix><Status>Enabled</Status><Expiration><Days>30</Days></Expiration></Rule></LifecycleConfiguration>`, func(c Command) Commander { return &BucketLifeCycleCommand{command: c} }, nil, nil, []string{}, true},
		{"logging", "logging", `<BucketLoggingStatus><LoggingEnabled><TargetBucket>logs-bucket</TargetBucket><TargetPrefix>logs/</TargetPrefix></LoggingEnabled></BucketLoggingStatus>`, func(c Command) Commander { return &BucketLogCommand{command: c} }, []string{"oss://logs-bucket/logs/"}, nil, []string{}, false},
		{"website", "website", `<WebsiteConfiguration><IndexDocument><Suffix>index.html</Suffix></IndexDocument></WebsiteConfiguration>`, func(c Command) Commander { return &BucketWebSiteCommand{command: c} }, nil, nil, []string{}, true},
		{"policy", "policy", `{"Version":"1","Statement":[{"Effect":"Allow","Action":["oss:GetObject"],"Resource":["acs:oss:*:*:bucket/*"]}]}`, func(c Command) Commander { return &BucketPolicyCommand{command: c} }, nil, nil, []string{}, true},
		{"qos", "qos", `<QoSConfiguration><TotalUploadBandwidth>1024</TotalUploadBandwidth></QoSConfiguration>`, func(c Command) Commander { return &BucketQosCommand{command: c} }, nil, nil, []string{}, true},
		{"access-monitor", "accessmonitor", `<AccessMonitorConfiguration><Status>Enabled</Status></AccessMonitorConfiguration>`, func(c Command) Commander { return &BucketAccessMonitorCommand{command: c} }, nil, nil, nil, true},
		{"resource-group", "resourceGroup", `<BucketResourceGroupConfiguration><ResourceGroupId>rg-test</ResourceGroupId></BucketResourceGroupConfiguration>`, func(c Command) Commander { return &BucketResourceGroupCommand{command: c} }, nil, nil, nil, true},
		{"inventory", "inventory", `<InventoryConfiguration><Id>daily</Id><IsEnabled>true</IsEnabled></InventoryConfiguration>`, func(c Command) Commander { return &BucketInventoryCommand{command: c} }, nil, []string{"daily"}, []string{"daily"}, true},
		{"style", "style", `<Style><Name>thumbnail</Name><Content>image/resize,w_100</Content></Style>`, func(c Command) Commander { return &BucketStyleCommand{command: c} }, []string{"thumbnail"}, []string{"thumbnail"}, []string{"thumbnail"}, true},
		{"tagging", "tagging", `<Tagging><TagSet><Tag><Key>team</Key><Value>cli</Value></Tag></TagSet></Tagging>`, func(c Command) Commander { return &BucketTagCommand{command: c} }, []string{"team#cli"}, nil, []string{}, false},
		{"encryption", "encryption", `<ServerSideEncryptionRule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>KMS</SSEAlgorithm><KMSMasterKeyID>key</KMSMasterKeyID></ApplyServerSideEncryptionByDefault></ServerSideEncryptionRule>`, func(c Command) Commander { return &BucketEncryptionCommand{command: c} }, nil, nil, []string{}, false},
		{"versioning", "versioning", `<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`, func(c Command) Commander { return &BucketVersioningCommand{command: c} }, []string{"Enabled"}, nil, nil, false},
		{"referer", "referer", `<RefererConfiguration><AllowEmptyReferer>true</AllowEmptyReferer><RefererList><Referer>https://example.test</Referer></RefererList></RefererConfiguration>`, func(c Command) Commander { return &BucketRefererCommand{command: c} }, []string{"https://example.test"}, nil, []string{}, false},
	}
}

func TestBucketConfigurationLocalRoundTrips(t *testing.T) {
	for _, tc := range bucketConfigurationCases() {
		t.Run(tc.name, func(t *testing.T) {
			type request struct{ method, path, query, body string }
			requests := make(chan request, 20)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				requests <- request{r.Method, r.URL.Path, r.URL.RawQuery, string(body)}
				if r.Method == http.MethodGet {
					io.WriteString(w, tc.body)
				} else if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			defer server.Close()
			base := localOSSCommand(t, server.URL)
			base.options[OptionSSEAlgorithm] = stringPointer("KMS")
			base.options[OptionKMSMasterKeyID] = stringPointer("key")
			run := func(method string, args []string) request {
				t.Helper()
				base.options[OptionMethod] = &method
				base.args = append([]string{"oss://bucket"}, args...)
				require.NoError(t, tc.newCommand(base).RunCommand())
				got := <-requests
				require.Equal(t, "/bucket/", got.path)
				require.Contains(t, strings.ToLower(got.query), strings.ToLower(tc.query))
				return got
			}
			dir := t.TempDir()
			stdout, err := os.Create(filepath.Join(dir, "stdout"))
			require.NoError(t, err)
			previousStdout := os.Stdout
			os.Stdout = stdout
			t.Cleanup(func() { os.Stdout = previousStdout; stdout.Close() })
			input := filepath.Join(dir, "config")
			require.NoError(t, os.WriteFile(input, []byte(tc.body), 0600))
			putArgs := append([]string{}, tc.putArgs...)
			if tc.fileInput {
				putArgs = append(putArgs, input)
			}
			got := run("put", putArgs)
			require.Equal(t, http.MethodPut, got.method)
			require.NotEmpty(t, got.body)
			output := filepath.Join(dir, "output")
			getArgs := append(append([]string{}, tc.getArgs...), output)
			got = run("get", getArgs)
			require.Equal(t, http.MethodGet, got.method)
			printsOnly := tc.name == "tagging" || tc.name == "encryption" || tc.name == "versioning"
			if printsOnly {
				output = stdout.Name()
			}
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			require.NotEmpty(t, data)
			if tc.name == "policy" {
				require.JSONEq(t, tc.body, string(data))
			} else if !printsOnly {
				require.Contains(t, string(data), strings.SplitN(strings.TrimPrefix(tc.body, "<"), ">", 2)[0])
			}
			if !printsOnly {
				require.NoError(t, os.WriteFile(output, []byte("keep existing"), 0600))
				for _, answer := range []string{"no\n", "yes\n"} {
					withBucketInput(t, answer, func() { run("get", getArgs) })
					data, err = os.ReadFile(output)
					require.NoError(t, err)
					if answer == "no\n" {
						require.Equal(t, "keep existing", string(data))
					} else {
						require.NotEqual(t, "keep existing", string(data))
					}
				}
				base.options[OptionMethod] = stringPointer("get")
				base.args = append(append([]string{"oss://bucket"}, tc.getArgs...), filepath.Join(dir, "missing", "output"))
				require.Error(t, tc.newCommand(base).RunCommand())
				<-requests
				run("get", tc.getArgs)
			}
			if tc.deleteArgs != nil {
				got = run("delete", tc.deleteArgs)
				if tc.name == "referer" {
					require.Equal(t, http.MethodPut, got.method)
					require.NotContains(t, got.body, "https://example.test")
				} else {
					require.Equal(t, http.MethodDelete, got.method)
				}
			}
		})
	}
}

func TestBucketConfigurationRejectsInvalidInputs(t *testing.T) {
	for _, tc := range bucketConfigurationCases() {
		t.Run(tc.name, func(t *testing.T) {
			base := localOSSCommand(t, "http://127.0.0.1:1")
			for _, method := range []string{"", "unsupported"} {
				base.args = []string{"oss://bucket"}
				base.options[OptionMethod] = &method
				require.ErrorContains(t, tc.newCommand(base).RunCommand(), "method")
			}
			base.options[OptionMethod] = stringPointer("get")
			base.args = []string{"local-path"}
			require.Error(t, tc.newCommand(base).RunCommand())
			if !tc.fileInput {
				return
			}
			base.options[OptionMethod] = stringPointer("put")
			base.args = []string{"oss://bucket"}
			require.Error(t, tc.newCommand(base).RunCommand())
			dir := t.TempDir()
			empty := filepath.Join(dir, "empty")
			require.NoError(t, os.WriteFile(empty, nil, 0600))
			for _, file := range []string{filepath.Join(dir, "missing"), dir, empty} {
				base.args = append(append([]string{"oss://bucket"}, tc.putArgs...), file)
				require.Error(t, tc.newCommand(base).RunCommand())
			}
		})
	}
}

func TestBucketConfigurationPropagatesServiceAndClientErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>local test denied</Message></Error>`)
	}))
	defer server.Close()
	for _, tc := range bucketConfigurationCases() {
		t.Run(tc.name, func(t *testing.T) {
			base := localOSSCommand(t, server.URL)
			base.options[OptionSSEAlgorithm] = stringPointer("KMS")
			dir := t.TempDir()
			input := filepath.Join(dir, "config")
			require.NoError(t, os.WriteFile(input, []byte(tc.body), 0600))
			putArgs := append([]string{}, tc.putArgs...)
			if tc.fileInput {
				putArgs = append(putArgs, input)
			}
			for _, method := range []string{"get", "put", "delete"} {
				if method == "delete" && tc.deleteArgs == nil {
					continue
				}
				args := tc.getArgs
				if method == "put" {
					args = putArgs
				} else if method == "delete" {
					args = tc.deleteArgs
				}
				base.args = append([]string{"oss://bucket"}, args...)
				base.options[OptionMethod] = &method
				require.ErrorContains(t, tc.newCommand(base).RunCommand(), "AccessDenied")
				endpoint := base.options[OptionEndpoint]
				base.options[OptionEndpoint] = stringPointer("")
				require.ErrorContains(t, tc.newCommand(base).RunCommand(), "endpoint")
				base.options[OptionEndpoint] = endpoint
			}
			base.minArgc, base.maxArgc = 1, 3
			require.Error(t, tc.newCommand(base).Init(nil, base.options))
		})
	}
}

func withBucketInput(t *testing.T, text string, fn func()) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	defer file.Close()
	_, err = file.WriteString(text)
	require.NoError(t, err)
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	previous := os.Stdin
	os.Stdin = file
	defer func() { os.Stdin = previous }()
	fn()
}

func stringPointer(value string) *string { return &value }

func captureBucketOutput(t *testing.T, fn func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	previous := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = previous; file.Close() }()
	fn()
	data, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(data)
}
