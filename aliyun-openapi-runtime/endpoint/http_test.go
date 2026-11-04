package endpoint

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type suffixTransport func(*http.Request) (*http.Response, error)

func (f suffixTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUnsignedRequestsUseSuffixAndCallerClient(t *testing.T) {
	t.Setenv(EnvSuffix, "aliyuncs.alibaba")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := NewRequestWithContext(ctx, "POST", "https://bucket.oss.aliyuncs.com:443/a%2Fb?q=old.aliyuncs.com", strings.NewReader("payload.aliyuncs.com"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Context() != ctx || r.URL.Host != "bucket.oss.aliyuncs.alibaba:443" || r.URL.EscapedPath() != "/a%2Fb" || r.URL.RawQuery != "q=old.aliyuncs.com" {
		t.Fatalf("unexpected request: %s", r.URL)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || string(body) != "payload.aliyuncs.com" {
		t.Fatalf("body = %q, err = %v", body, err)
	}
	calls, redirects := 0, 0
	client := &http.Client{
		Transport: suffixTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host != "assets.oss.aliyuncs.alibaba" || r.Host != r.URL.Host {
				t.Fatalf("unexpected host: %s / %s", r.URL.Host, r.Host)
			}
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://example.com/next"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return http.ErrUseLastResponse },
	}
	response, err := Get(client, "https://assets.oss.aliyuncs.com/archive.zip")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if calls != 1 || redirects != 1 || response.StatusCode != 302 {
		t.Fatalf("calls=%d redirects=%d status=%d", calls, redirects, response.StatusCode)
	}
	t.Setenv(EnvSuffix, "invalid/suffix")
	if _, err := Get(client, "https://assets.oss.aliyuncs.com/archive.zip"); err == nil || calls != 1 {
		t.Fatal("invalid suffix reached transport")
	}
}

func TestHTTPGetUsesDefaultClient(t *testing.T) {
	t.Setenv(EnvSuffix, "aliyuncs.alibaba")
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	http.DefaultClient = &http.Client{Transport: suffixTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "download.aliyuncs.alibaba" {
			t.Fatalf("host = %s", r.URL.Host)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("archive"))}, nil
	})}
	r, err := HTTPGet("https://download.aliyuncs.com/tool.zip")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
}

func TestForwardedEndpointArguments(t *testing.T) {
	t.Setenv(EnvSuffix, "aliyuncs.alibaba")
	args := []string{"ls", "--endpoint", "oss.aliyuncs.com", "--sts-endpoint=https://sts.aliyuncs.com:443", "-e=custom.example.com", "--body", `{"url":"a.aliyuncs.com"}`, "--", "--endpoint", "opaque.aliyuncs.com"}
	want := []string{"ls", "--endpoint", "oss.aliyuncs.alibaba", "--sts-endpoint=https://sts.aliyuncs.alibaba:443", "-e=custom.example.com", "--body", `{"url":"a.aliyuncs.com"}`, "--", "--endpoint", "opaque.aliyuncs.com"}
	got := WithSuffixArgs(args)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %q", got)
	}
	if args[2] != "oss.aliyuncs.com" {
		t.Fatal("caller arguments mutated")
	}
	t.Setenv(EnvSuffix, "")
	if !reflect.DeepEqual(WithSuffixArgs(args), args) {
		t.Fatal("default suffix changed arguments")
	}
}
