package cloudsso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type closeFailureSSOBody struct {
	io.Reader
	closed bool
}

func (b *closeFailureSSOBody) Close() error { b.closed = true; return errors.New("close interrupted") }

func TestSSOPaginationLimitAndResponseBodyFailures(t *testing.T) {
	client := &http.Client{Transport: &MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "25", req.URL.Query().Get("MaxResults"))
		return mockHTTPResponse(200, `{"AccessConfigurationsForAccount":[{"AccessConfigurationId":"config"}]}`), nil
	}}}
	parameter := &AccessConfigurationsParameter{UrlPrefix: "https://sso.example.invalid", HttpClient: client}
	result, err := parameter.ListAccessConfigurationsForAccount(AccessConfigurationsRequest{AccountId: "account", MaxResults: 25})
	require.NoError(t, err)
	require.Equal(t, "config", result.AccessConfigurationsForAccount[0].AccessConfigurationId)
	client.Transport = &MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(&ssoReadFailure{}), Header: make(http.Header)}, nil
	}}
	_, err = (&ListUserParameter{BaseUrl: "https://sso.example.invalid", HttpClient: client}).ListUsers("", 10)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	body := &closeFailureSSOBody{Reader: strings.NewReader(`{"CloudCredential":{"AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"sts"}}`)}
	client.Transport = &MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
	}}
	credential, err := CreateCloudCredential("https://sso.example.invalid", "access", CloudCredentialOptions{}, client)
	require.NoError(t, err)
	require.Equal(t, "id", credential.AccessKeyId)
	require.True(t, body.closed)
	client.Transport = &MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) { return mockHTTPResponse(200, `{"AccessKeyId":123}`), nil }}
	_, err = CreateCloudCredential("https://sso.example.invalid", "access", CloudCredentialOptions{}, client)
	require.ErrorContains(t, err, "failed to parse cloud credential")
}

type ssoReadFailure struct{}

func (*ssoReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestSSOLoginFailureChild(t *testing.T) {
	mode := os.Getenv("ALIYUN_CLI_TEST_SSO_FAILURE")
	if mode == "" {
		return
	}
	t.Setenv("PATH", t.TempDir())
	installLoginTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/device-authorization" {
			return mockHTTPResponse(200, `{"DeviceCode":"device","Interval":0}`), nil
		}
		if mode == "expired" {
			return mockHTTPResponse(400, `{"ErrorCode":"InvalidDeviceCodeError"}`), nil
		}
		return mockHTTPResponse(400, `{"ErrorCode":"AuthorizationPending"}`), nil
	})
	_, _ = (SsoLogin{SignInUrl: "https://sso.example.invalid/start"}).GetAccessToken()
	os.Exit(90)
}

func TestSSOLoginExpirationAndPollingLimitTerminateProcess(t *testing.T) {
	for _, mode := range []string{"expired", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSSOLoginFailureChild$")
			command.Env = append(os.Environ(), "ALIYUN_CLI_TEST_SSO_FAILURE="+mode)
			output, err := command.CombinedOutput()
			require.NoError(t, ctx.Err())
			var status *exec.ExitError
			require.ErrorAs(t, err, &status, string(output))
			require.NotZero(t, status.ExitCode())
			if mode == "expired" {
				require.True(t, strings.Contains(string(output), "expired") || strings.Contains(string(output), "过期"), string(output))
			} else {
				require.True(t, strings.Contains(string(output), "timeout") || strings.Contains(string(output), "超时"), string(output))
			}
		})
	}
}
