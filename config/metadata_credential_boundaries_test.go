package config

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleCredentialsUseOnlyLocalMetadataProxy(t *testing.T) {
	old := http.DefaultTransport
	transport := old.(*http.Transport).Clone()
	t.Cleanup(func() { http.DefaultTransport = old; transport.CloseIdleConnections() })
	var failed atomic.Bool
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		require.Equal(t, "100.100.100.200", req.URL.Host)
		if failed.Load() {
			http.Error(w, "metadata unavailable", 503)
			return
		}
		switch req.URL.Path {
		case "/latest/api/token":
			io.WriteString(w, "metadata-token")
		case "/latest/meta-data/ram/security-credentials/role":
			require.Equal(t, "metadata-token", req.Header.Get("X-Aliyun-Ecs-Metadata-Token"))
			io.WriteString(w, `{"Code":"Success","AccessKeyId":"metadata-id","AccessKeySecret":"metadata-secret","SecurityToken":"metadata-sts","Expiration":"2099-01-01T00:00:00Z"}`)
		default:
			http.Error(w, "unexpected metadata path", 404)
		}
	}))
	t.Cleanup(server.Close)
	proxy, err := url.Parse(server.URL)
	require.NoError(t, err)
	transport.Proxy = http.ProxyURL(proxy)
	http.DefaultTransport = transport
	ctx := lifecycleContext(io.Discard)
	t.Setenv("ALIBABA_CLOUD_ECS_METADATA_DISABLED", "true")
	profile := &Profile{Mode: RamRoleArnWithEcs, RamRoleName: "role", RegionId: "cn-hangzhou", RamRoleArn: "acs:ram::123:role/final", RoleSessionName: "session"}
	_, err = profile.GetCredential(ctx, nil)
	require.ErrorContains(t, err, "IMDS credentials is disabled")
	require.Zero(t, requests.Load())
	t.Setenv("ALIBABA_CLOUD_ECS_METADATA_DISABLED", "false")
	credential, err := profile.GetCredential(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, credential)
	require.Equal(t, "ram_role_arn", *credential.GetType())
	require.Equal(t, int64(2), requests.Load())
	failed.Store(true)
	_, err = profile.GetCredential(ctx, nil)
	require.Error(t, err)
	chain := &Profile{Name: "chain", Mode: ChainableRamRoleArn, SourceProfile: "source", parent: &Configuration{Profiles: []Profile{{Name: "source", Mode: EcsRamRole, RegionId: "cn-hangzhou", RamRoleName: "role"}}}}
	_, err = chain.GetCredential(ctx, nil)
	require.Error(t, err)
}

func TestCredentialURITransportAndBodyErrors(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	ctx := lifecycleContext(io.Discard)
	for _, tc := range []struct {
		name         string
		body         io.ReadCloser
		transportErr bool
	}{
		{name: "transport", transportErr: true}, {name: "body", body: configureFailingBody{io.ErrUnexpectedEOF}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = configureEdgeTransport(func(*http.Request) (*http.Response, error) {
				if tc.transportErr {
					return nil, io.ErrClosedPipe
				}
				return &http.Response{StatusCode: 200, Body: tc.body, Header: make(http.Header)}, nil
			})
			profile := &Profile{Mode: CredentialsURI, RegionId: "cn-hangzhou", CredentialsURI: "https://credentials.example.invalid"}
			_, err := profile.GetCredential(ctx, nil)
			require.Error(t, err)
		})
	}
	profile := &Profile{Mode: AK, RegionId: "cn-hangzhou", AccessKeyId: "id", AccessKeySecret: "secret"}
	proxy := "http://proxy.example.invalid:8080"
	credential, err := profile.GetCredential(ctx, &proxy)
	require.NoError(t, err)
	model, err := credential.GetCredential()
	require.NoError(t, err)
	require.Equal(t, "id", *model.AccessKeyId)
	t.Setenv("HTTPS_PROXY", proxy)
	credential, err = profile.GetCredential(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, credential)
}
