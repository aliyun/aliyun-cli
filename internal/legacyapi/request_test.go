package legacyapi

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPreparedRequestTextAndResource(t *testing.T) {
	r := NewRequest()
	r.Domain = "fixture.invalid"
	r.Scheme = "https"
	r.QueryParams["Empty"] = ""
	r.QueryParams["Text"] = "a b"
	r.FormParams["Form"] = "value"
	require.Equal(t, "RPC", r.GetStyle())
	require.Equal(t, "/?Empty=&Text=a+b", r.Resource())
	require.Contains(t, r.String(), "GET /?Empty=&Text=a+b HTTPS/1.1\nHost: fixture.invalid\n")
	require.Contains(t, r.String(), "Form: value\n")
	r.SetContent([]byte("raw\x00"))
	r.SetContentType("text/plain")
	require.Contains(t, r.String(), "\n\nraw\x00\n")
	require.NotContains(t, r.String(), "Form: value")
	r.PathPattern = "/names/[name]"
	r.PathParams["name"] = "a/b"
	require.Equal(t, "ROA", r.GetStyle())
	require.Equal(t, "/names/a/b?Empty&Text=a b", r.Resource())
	r.SetReadTimeout(time.Second)
	r.SetConnectTimeout(2 * time.Second)
	require.Equal(t, time.Second, r.GetReadTimeout())
	require.Equal(t, 2*time.Second, r.GetConnectTimeout())
}

func TestCompatibilityErrorGolden(t *testing.T) {
	e := NewServerError(400, `{"Code":"Bad","Message":"bad","RequestId":"req","HostId":"host","Recommend":"fix","AccessDeniedDetail":{"PolicyType":"test"}}`, "prefix: ").(*ServerError)
	e.RespHeaders = http.Header{"X-Test": {"a", "b"}}
	require.Equal(t, "SDK.ServerError\nErrorCode: Bad\nRecommend: prefix: fix\nRequestId: req\nMessage: bad\nRespHeaders: map[X-Test:[a b]]\nAccessDeniedDetail: map[PolicyType:test]", e.Error())
	require.Equal(t, "Bad", e.ErrorCode())
	require.Equal(t, "bad", e.Message())
	require.Equal(t, "req", e.RequestId())
	require.Equal(t, "host", e.HostId())
	require.Equal(t, "fix", e.Recommend())
	require.Equal(t, "test", e.AccessDeniedDetail()["PolicyType"])
	for _, body := range []string{"<html>bad</html>", `{"Message":null}`, `{"Message":42}`} {
		e := NewServerError(502, body, "").(*ServerError)
		require.Equal(t, body, e.Message())
	}
	for _, tc := range []struct{ code, text, client, want string }{
		{"Other", "server string to sign is:same", "same", ""},
		{"SignatureDoesNotMatch", "message", "same", ""},
		{"SignatureDoesNotMatch", "server string to sign is:same", "same", "InvalidAccessKeySecret"},
		{"IncompleteSignature", "server string to sign is:other", "same", "This may be a bug"},
		{"IncompleteSignature", "server string to sign is:same", "", ""},
	} {
		e := NewServerError(400, `{}`, "").(*ServerError)
		e.code, e.message = tc.code, tc.text
		e.explainSignature(tc.client)
		require.Contains(t, e.Recommend(), tc.want)
	}
	cause := errors.New("cause")
	client := NewClientError("SDK.TimeoutError", "timeout", cause).(*ClientError)
	require.Equal(t, "[SDK.TimeoutError] timeout\ncaused by:\ncause", client.Error())
	require.ErrorIs(t, client, cause)
	require.Equal(t, "[Code] message", NewClientError("Code", "message", nil).Error())
}

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (brokenBody) Close() error             { return nil }

func TestResponseReadFailure(t *testing.T) {
	r := NewResponse()
	err := ReadResponse(r, &http.Response{StatusCode: 200, Header: http.Header{}, Body: brokenBody{}})
	require.ErrorContains(t, err, "read failed")
	require.Empty(t, r.GetHttpContentBytes())
	c, req := fixture(t)
	c.RetryCount = 2
	calls := 0
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: brokenBody{}}, nil
	}))
	_, err = c.Send(req)
	require.ErrorContains(t, err, "read failed")
	require.Equal(t, 1, calls)
}

func TestLocalErrorsAndNoCredentials(t *testing.T) {
	c, r := fixture(t)
	r.Domain = ""
	_, err := c.Send(r)
	require.ErrorContains(t, err, "SDK.CanNotResolveEndpoint")
	c, r = fixture(t)
	c.AccessKeyID = ""
	c.AccessKeySecret = ""
	_, err = c.Send(r)
	require.Error(t, err)
	c, r = fixture(t)
	c.RetryCount = -1
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { return reply(500, "failed"), nil }))
	_, err = c.Send(r)
	require.ErrorContains(t, err, "failed")
	r.Scheme = ""
	require.Equal(t, "HTTP", protocol(r))
	_, r = fixture(t)
	t.Setenv("HTTPS_PROXY", "%zz")
	_, err = c.Send(r)
	require.Error(t, err)
}
