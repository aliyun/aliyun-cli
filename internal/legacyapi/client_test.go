package legacyapi

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fixture(t *testing.T) (*Client, *Request) {
	t.Helper()
	for _, key := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy"} {
		t.Setenv(key, "")
	}
	c := NewClient("cn-hangzhou", "fixture-id", "fixture-secret", "fixture-token")
	r := NewRequest()
	r.Domain, r.Scheme, r.Product, r.Version, r.ApiName = "fixture.invalid", "https", "Ecs", "2014-05-26", "DescribeInstances"
	return c, r
}
func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Fixture": {"one", "two"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func signature(secret, text string) string {
	h := hmac.New(sha1.New, []byte(secret))
	_, _ = h.Write([]byte(text))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func TestRPCWirePreservesFlatStringsAndLegacySignature(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			c, r := fixture(t)
			r.Method, r.AcceptFormat = method, "XML"
			r.Headers["Content-Type"] = "application/json"
			r.QueryParams = map[string]string{"Empty": "", "Bool": "True", "Number": "01", "Decimal": "1.0", "JSON": `{"x":1}`, "Foo.1.Bar": "中文 +/%&="}
			r.FormParams = map[string]string{"FormEmpty": "", "Servers.1.Id": "i-1", "Large": "9007199254740993"}
			original := copyMap(r.QueryParams)
			c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
				require.Equal(t, method, h.Method)
				require.Equal(t, "/", h.URL.Path)
				require.Equal(t, "application/x-www-form-urlencoded", h.Header.Get("Content-Type"))
				q := h.URL.Query()
				for key, value := range original {
					require.Contains(t, q, key)
					require.Equal(t, value, q.Get(key))
				}
				require.Equal(t, "XML", q.Get("Format"))
				require.Equal(t, "cn-hangzhou", q.Get("RegionId"))
				require.Equal(t, "fixture-token", q.Get("SecurityToken"))
				require.Equal(t, "HMAC-SHA1", q.Get("SignatureMethod"))
				require.Equal(t, "1.0", q.Get("SignatureVersion"))
				require.Contains(t, q, "SignatureType")
				body, err := io.ReadAll(h.Body)
				require.NoError(t, err)
				form, err := url.ParseQuery(string(body))
				require.NoError(t, err)
				for key, value := range r.FormParams {
					require.Contains(t, form, key)
					require.Equal(t, value, form.Get(key))
				}
				actualSignature := q.Get("Signature")
				q.Del("Signature")
				for key, value := range form {
					q[key] = value
				}
				canonical := strings.NewReplacer("+", "%20", "*", "%2A", "%7E", "~").Replace(q.Encode())
				require.Equal(t, signature("fixture-secret&", method+"&%2F&"+url.QueryEscape(canonical)), actualSignature)
				return reply(200, `{"Number":9007199254740993,"Empty":null}`), nil
			}))
			resp, err := c.Send(r)
			require.NoError(t, err)
			require.Equal(t, `{"Number":9007199254740993,"Empty":null}`, resp.GetHttpContentString())
			require.Equal(t, original, r.QueryParams, "SDK signing must not mutate preparation state")
		})
	}
}

func TestRPCRegionOverrideAndEmptyKey(t *testing.T) {
	for _, tc := range []struct {
		region string
		query  map[string]string
		want   string
	}{
		{"cn-shanghai", map[string]string{}, "cn-shanghai"},
		{"", map[string]string{"RegionId": ""}, ""},
		{"cn-shanghai", map[string]string{"RegionId": "explicit"}, "explicit"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			c, r := fixture(t)
			r.RegionId = tc.region
			r.QueryParams = tc.query
			c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
				require.Equal(t, tc.want, h.URL.Query().Get("RegionId"))
				return reply(200, ""), nil
			}))
			_, err := c.Send(r)
			require.NoError(t, err)
		})
	}
}

func TestROARawBodyPathAndSignature(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		form                    map[string]string
	}{
		{"json", `{ "Id":9007199254740993, "Empty":null }`, "application/json", nil},
		{"xml", "<Result>中文</Result>", "application/xml", nil},
		{"binary", "\x00\xffdata", "application/octet-stream", nil},
		{"empty", "", "application/octet-stream", nil},
		{"form", "ignored", "", map[string]string{"Foo.1": "", "Key": "a+b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, r := fixture(t)
			r.Method = "PUT"
			r.PathPattern = "/clusters/[name]/tasks"
			r.PathParams["name"] = "a/中文 x%2F"
			r.QueryParams = map[string]string{"empty": "", "query": "a +/&"}
			r.Content = []byte(tc.body)
			if tc.contentType != "application/octet-stream" {
				r.Headers["Content-Type"] = tc.contentType
			}
			r.Headers["x-acs-region-id"] = "cn-hangzhou"
			r.FormParams = tc.form
			c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
				require.Equal(t, "/clusters/a/中文 x/", strings.TrimSuffix(h.URL.Path, "/tasks"))
				body, err := io.ReadAll(h.Body)
				require.NoError(t, err)
				wantBody := tc.body
				wantType := tc.contentType
				if tc.form != nil {
					wantBody = "Foo.1=&Key=a%2Bb"
					wantType = "application/x-www-form-urlencoded"
				}
				require.Equal(t, wantBody, string(body))
				require.Equal(t, wantType, h.Header.Get("Content-Type"))
				md := md5.Sum(body)
				require.Equal(t, base64.StdEncoding.EncodeToString(md[:]), h.Header["Content-MD5"][0])
				require.Equal(t, "fixture-token", h.Header["x-acs-security-token"][0])
				require.NotContains(t, h.Header, "x-acs-signature-nonce")
				// Build the old unescaped canonical resource independently of URL.Path.
				resource := "/clusters/a/中文 x%2F/tasks?empty&query=a +/&"
				xacs := "x-acs-action:DescribeInstances\nx-acs-credentials-provider:" + h.Header["x-acs-credentials-provider"][0] + "\nx-acs-region-id:cn-hangzhou\nx-acs-security-token:fixture-token\nx-acs-signature-method:HMAC-SHA1\nx-acs-signature-version:1.0\nx-acs-version:2014-05-26\n"
				canonical := "PUT\napplication/json\n" + h.Header["Content-MD5"][0] + "\n" + wantType + "\n" + h.Header.Get("Date") + "\n" + xacs + resource
				require.Equal(t, "acs fixture-id:"+signature("fixture-secret", canonical), h.Header.Get("Authorization"))
				return reply(200, "<ok/>"), nil
			}))
			resp, err := c.Send(r)
			require.NoError(t, err)
			require.Equal(t, "<ok/>", resp.GetHttpContentString())
		})
	}
}

func TestResponseAndErrorsRetainRawBytesAndAllHeaders(t *testing.T) {
	for _, status := range []int{200, 204, 400, 429, 500} {
		for _, body := range []string{"", `{"Code":"Bad","Message":"bad","RequestId":"r","Recommend":"help"}`, "<html>proxy error</html>"} {
			t.Run(fmt.Sprintf("%d/%s", status, body), func(t *testing.T) {
				c, r := fixture(t)
				c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { return reply(status, body), nil }))
				resp, err := c.Send(r)
				require.Equal(t, body, resp.GetHttpContentString())
				require.Equal(t, []string{"one", "two"}, resp.Headers.Values("X-Fixture"))
				if status < 300 {
					require.NoError(t, err)
				} else {
					var server *ServerError
					require.ErrorAs(t, err, &server)
					require.Equal(t, status, server.HttpStatus())
					require.Equal(t, []string{"one", "two"}, server.RespHeaders.Values("X-Fixture"))
					require.Contains(t, err.Error(), "SDK.ServerError\nErrorCode:")
				}
			})
		}
	}
}

func TestRetryCountsReplaysBodiesAndResigns(t *testing.T) {
	for _, status := range []int{400, 500} {
		for _, retry := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%d/%d", status, retry), func(t *testing.T) {
				c, r := fixture(t)
				c.RetryCount = retry
				r.Method = "POST"
				r.FormParams["Payload"] = "abc"
				calls := 0
				nonces := map[string]bool{}
				c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
					calls++
					nonce := h.URL.Query().Get("SignatureNonce")
					require.False(t, nonces[nonce])
					nonces[nonce] = true
					body, err := io.ReadAll(h.Body)
					require.NoError(t, err)
					require.Equal(t, "Payload=abc", string(body))
					if status == 500 && calls == retry+1 {
						return reply(200, `{"ok":true}`), nil
					}
					return reply(status, `{"Code":"Bad","Message":"bad"}`), nil
				}))
				_, err := c.Send(r)
				if status == 500 {
					require.NoError(t, err)
					require.Equal(t, retry+1, calls)
				} else {
					require.Error(t, err)
					require.Equal(t, 1, calls)
				}
			})
		}
	}
}

func TestTransportFailuresAndExhaustion(t *testing.T) {
	for _, retry := range []int{0, 2} {
		t.Run(fmt.Sprint(retry), func(t *testing.T) {
			c, r := fixture(t)
			c.RetryCount = retry
			calls := 0
			c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("network down") }))
			_, err := c.Send(r)
			require.ErrorContains(t, err, "network down")
			require.Equal(t, retry+1, calls)
			if retry > 0 {
				var clientErr *ClientError
				require.ErrorAs(t, err, &clientErr)
				require.NotNil(t, clientErr.OriginError())
			}
		})
	}
	c, r := fixture(t)
	c.RetryCount = 2
	calls := 0
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { calls++; return nil, x509.UnknownAuthorityError{} }))
	_, err := c.Send(r)
	require.Error(t, err)
	require.Equal(t, 1, calls)
	c, r = fixture(t)
	c.RetryCount = 1
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { return reply(503, "unavailable"), nil }))
	_, err = c.Send(r)
	require.ErrorContains(t, err, "unavailable")
}

func TestPreviewNeverDialsAndMatchesPreparedRequest(t *testing.T) {
	for _, path := range []string{"", "/clusters/[name]"} {
		t.Run(path, func(t *testing.T) {
			c, r := fixture(t)
			r.PathPattern = path
			r.PathParams["name"] = "a/b"
			r.Method = "POST"
			r.FormParams["Empty"] = ""
			c.RetryCount = 3
			c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("preview dialed network"); return nil, nil }))
			require.NoError(t, c.Preview(r))
			require.Contains(t, r.String(), "POST ")
			require.Contains(t, r.String(), "Host: fixture.invalid")
			if path == "" {
				require.Contains(t, r.String(), "Signature=")
			} else {
				require.Contains(t, r.String(), "Authorization: acs fixture-id:")
				require.Contains(t, r.String(), "/clusters/a/b")
			}
			require.Empty(t, r.QueryParams)
			require.Empty(t, r.Content)
		})
	}
}

func TestTimeoutAndTLSOnRealLocalHTTP(t *testing.T) {
	c, r := fixture(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("raw")) }))
	defer server.Close()
	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	r.Domain = u.Host
	_, err = c.Send(r)
	require.Error(t, err)
	c.SetHTTPSInsecure(true)
	resp, err := c.Send(r)
	require.NoError(t, err)
	require.Equal(t, "raw", resp.GetHttpContentString())
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.Write([]byte("late"))
	}))
	defer slow.Close()
	u, err = url.Parse(slow.URL)
	require.NoError(t, err)
	r.Domain = u.Host
	r.Scheme = "http"
	r.SetReadTimeout(5 * time.Millisecond)
	c.RetryCount = 1
	_, err = c.Send(r)
	require.ErrorContains(t, err, "Read timeout")
}

func TestTimeoutPrecedenceAndAPIData(t *testing.T) {
	c, r := fixture(t)
	read, connect := c.timeouts(r)
	require.Equal(t, 10*time.Second, read)
	require.Equal(t, 5*time.Second, connect)
	r.ApiName = "CreateInstance"
	require.NotZero(t, fallbackTimeouts["ecs"][r.ApiName])
	read, _ = c.timeouts(r)
	require.Equal(t, time.Duration(fallbackTimeouts["ecs"][r.ApiName])*time.Second, read)
	c.SetReadTimeout(3 * time.Second)
	c.SetConnectTimeout(4 * time.Second)
	read, connect = c.timeouts(r)
	require.Equal(t, 3*time.Second, read)
	require.Equal(t, 4*time.Second, connect)
	r.SetReadTimeout(time.Second)
	r.SetConnectTimeout(2 * time.Second)
	read, connect = c.timeouts(r)
	require.Equal(t, time.Second, read)
	require.Equal(t, 2*time.Second, connect)
}

func TestProxyEnvironmentPrecedenceAndBypass(t *testing.T) {
	_, r := fixture(t)
	t.Setenv("HTTPS_PROXY", "http://upper.invalid:1234")
	t.Setenv("https_proxy", "http://lower.invalid:1234")
	p, err := legacyProxy(r)
	require.NoError(t, err)
	proxy, err := p(&http.Request{})
	require.NoError(t, err)
	require.Equal(t, "upper.invalid:1234", proxy.Host)
	t.Setenv("NO_PROXY", "*.invalid")
	p, err = legacyProxy(r)
	require.NoError(t, err)
	require.Nil(t, p)
	t.Setenv("NO_PROXY", "[")
	_, err = legacyProxy(r)
	require.Error(t, err)
	t.Setenv("NO_PROXY", "")
	t.Setenv("HTTPS_PROXY", "%zz")
	_, err = legacyProxy(r)
	require.Error(t, err)
	t.Setenv("HTTPS_PROXY", "")
	p, err = legacyProxy(r)
	require.NoError(t, err)
	proxy, err = p(&http.Request{})
	require.NoError(t, err)
	require.Equal(t, "lower.invalid:1234", proxy.Host)
	r.Scheme = "http"
	t.Setenv("HTTP_PROXY", "http://http.invalid")
	p, err = legacyProxy(r)
	require.NoError(t, err)
	proxy, err = p(&http.Request{})
	require.NoError(t, err)
	require.Equal(t, "http.invalid", proxy.Host)
}

func TestUserAgentAndCustomHost(t *testing.T) {
	c, r := fixture(t)
	r.PathPattern = "/price"
	r.Headers["Host"] = "gateway.invalid"
	r.Port = "8080"
	c.ExtraUserAgent = "extra"
	c.AppendUserAgent("Agent", "one")
	c.AppendUserAgent("Agent", "two")
	c.AppendUserAgent("Core", "ignored")
	c.AppendUserAgent("Go", "ignored")
	c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
		require.Equal(t, "fixture.invalid:8080", h.URL.Host)
		require.Equal(t, "gateway.invalid", h.Host)
		require.Contains(t, h.UserAgent(), "Agent/two")
		require.Contains(t, h.UserAgent(), "Extra/extra")
		require.NotContains(t, h.UserAgent(), "ignored")
		return reply(200, ""), nil
	}))
	_, err := c.Send(r)
	require.NoError(t, err)
}

func TestPreviewIgnoresNetworkProxyConfiguration(t *testing.T) {
	c, r := fixture(t)
	t.Setenv("HTTP_PROXY", "%zz")
	t.Setenv("HTTPS_PROXY", "%zz")
	t.Setenv("NO_PROXY", "[")
	require.NoError(t, c.Preview(r))
	require.Contains(t, r.String(), "Signature=")
	_, err := c.Send(r)
	require.Error(t, err)
}

func TestLowercaseHostHeaderDoesNotChangeEndpoint(t *testing.T) {
	c, r := fixture(t)
	r.Headers["host"] = "virtual-host.invalid"
	c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
		require.Equal(t, "fixture.invalid", h.URL.Host)
		require.Equal(t, "fixture.invalid", h.Host)
		return reply(200, ""), nil
	}))
	_, err := c.Send(r)
	require.NoError(t, err)
}
