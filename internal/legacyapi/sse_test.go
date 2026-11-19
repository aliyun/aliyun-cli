package legacyapi

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/dara"
	"github.com/stretchr/testify/require"
)

func sseFixture(t *testing.T) (*Client, *Request) {
	t.Helper()
	c, r := fixture(t)
	r.Product, r.Version, r.ApiName = "Das", "2020-01-16", "GetDasAgentSSE"
	r.Method = "POST"
	r.QueryParams = map[string]string{"Query": "which instances are slow"}
	return c, r
}

func TestStreamSSEUsesV3SignatureAndStreamsIncrementally(t *testing.T) {
	c, r := sseFixture(t)
	pr, pw := io.Pipe()
	type wireFacts struct {
		method, action, version, authorization, contentSHA256, nonce, query, regionID, signature, signatureMethod string
	}
	seen := make(chan wireFacts, 4)
	c.SetTransport(roundTrip(func(h *http.Request) (*http.Response, error) {
		// The SDK writes lowercase keys straight into the header map, so a
		// canonicalizing Get would miss them.
		header := func(key string) string {
			for k, values := range h.Header {
				if strings.EqualFold(k, key) && len(values) > 0 {
					return values[0]
				}
			}
			return ""
		}
		q := h.URL.Query()
		seen <- wireFacts{
			method: h.Method, action: header("x-acs-action"), version: header("x-acs-version"),
			authorization: header("Authorization"), contentSHA256: header("x-acs-content-sha256"),
			nonce: header("x-acs-signature-nonce"), query: q.Get("Query"), regionID: q.Get("RegionId"),
			signature: q.Get("Signature"), signatureMethod: q.Get("SignatureMethod"),
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr}, nil
	}))

	var mu sync.Mutex
	var events []string
	firstEvent := make(chan struct{})
	emit := func(event *dara.SSEEvent) error {
		mu.Lock()
		events = append(events, dara.StringValue(event.Data))
		n := len(events)
		mu.Unlock()
		if n == 1 {
			close(firstEvent)
		}
		return nil
	}
	writeErr := make(chan error, 1)
	go func() {
		if _, err := pw.Write([]byte("data: one\n\n")); err != nil {
			writeErr <- err
			return
		}
		// The first event must reach the caller while the response is still open.
		select {
		case <-firstEvent:
		case <-time.After(10 * time.Second):
			writeErr <- fmt.Errorf("first event was not delivered before the stream ended")
			return
		}
		if _, err := pw.Write([]byte("data: two\n\n")); err != nil {
			writeErr <- err
			return
		}
		writeErr <- pw.Close()
	}()

	require.NoError(t, c.StreamSSE(r, emit))
	require.NoError(t, <-writeErr)
	mu.Lock()
	require.Equal(t, []string{"one", "two"}, events)
	mu.Unlock()

	facts := <-seen
	require.Equal(t, "POST", facts.method)
	require.Equal(t, "GetDasAgentSSE", facts.action)
	require.Equal(t, "2020-01-16", facts.version)
	require.True(t, strings.HasPrefix(facts.authorization, "ACS3-HMAC-SHA256 Credential=fixture-id,"),
		"SSE must be signed with the v3 algorithm, got %q", facts.authorization)
	require.NotEmpty(t, facts.contentSHA256)
	require.NotEmpty(t, facts.nonce)
	require.Equal(t, "which instances are slow", facts.query)
	require.Equal(t, "cn-hangzhou", facts.regionID, "legacy RegionId default is preserved")
	require.Empty(t, facts.signature, "the legacy query signature must not be added")
	require.Empty(t, facts.signatureMethod)
}

func TestStreamSSEReturnsServerErrorWithoutDeadlock(t *testing.T) {
	c, r := sseFixture(t)
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) {
		return reply(404, `{"Code":"InvalidAccessKeyId.NotFound","Message":"Specified access key is not found.","RequestId":"fixture-request"}`), nil
	}))

	done := make(chan error, 1)
	go func() { done <- c.StreamSSE(r, nil) }()
	select {
	case err := <-done:
		require.Error(t, err)
		require.Contains(t, err.Error(), "InvalidAccessKeyId.NotFound")
		require.Contains(t, err.Error(), "404")
	case <-time.After(10 * time.Second):
		t.Fatal("StreamSSE blocked on a non-2xx SSE response")
	}
}

func TestStreamSSEEmitErrorDrainsProducer(t *testing.T) {
	c, r := sseFixture(t)
	pr, pw := io.Pipe()
	c.SetTransport(roundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr}, nil
	}))
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		for _, event := range []string{"one", "two", "three"} {
			if _, err := pw.Write([]byte("data: " + event + "\n\n")); err != nil {
				return
			}
		}
		_ = pw.Close()
	}()

	calls := 0
	err := c.StreamSSE(r, func(*dara.SSEEvent) error {
		calls++
		return fmt.Errorf("sink closed")
	})
	require.ErrorContains(t, err, "sink closed")
	require.Equal(t, 1, calls, "a failing sink must stop further emissions")

	select {
	case <-producerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("producer was not drained after an emit failure")
	}
}

func TestStreamSSEDialsWithDefaultTransportAndKeepsRPCParams(t *testing.T) {
	c, r := sseFixture(t)
	r.Scheme = "HTTP"
	r.RegionId = "cn-shanghai"
	r.FormParams = map[string]string{"InstanceId": "i-1"}
	r.Headers["Host"] = "ignored.example"

	type requestFacts struct {
		form, query map[string][]string
		action      string
		host        string
	}
	facts := make(chan requestFacts, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = req.ParseForm()
		facts <- requestFacts{form: req.PostForm, query: req.URL.Query(), action: req.Header.Get("X-Acs-Action"), host: req.Host}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: live\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()
	r.Domain = strings.TrimPrefix(server.URL, "http://")

	var events []string
	require.NoError(t, c.StreamSSE(r, func(event *dara.SSEEvent) error {
		events = append(events, dara.StringValue(event.Data))
		return nil
	}))
	require.Equal(t, []string{"live"}, events)

	got := <-facts
	require.Equal(t, "GetDasAgentSSE", got.action)
	require.Equal(t, "i-1", got.form["InstanceId"][0], "RPC form params keep flowing to the SSE request")
	require.Equal(t, "cn-shanghai", got.query["RegionId"][0], "the prepared RegionId wins over the client region")
	// Reaching this server proves the dial target stayed on r.Domain; the Host
	// override must still be the one signed and sent on the wire.
	require.Equal(t, "ignored.example", got.host, "an explicit Host override must reach the wire")
}

func TestStreamSSEReportsInvalidProxy(t *testing.T) {
	c, r := sseFixture(t)
	t.Setenv("HTTPS_PROXY", "://bad")
	require.ErrorContains(t, c.StreamSSE(r, nil), "missing protocol scheme")
}

func TestStreamSSERequiresEndpoint(t *testing.T) {
	c, r := sseFixture(t)
	r.Domain = ""
	require.ErrorContains(t, c.StreamSSE(r, nil), "Can not resolve endpoint")
}

func TestStreamSSEHostOverrideKeepsHTTPProxy(t *testing.T) {
	c, r := sseFixture(t)
	r.Scheme = "HTTP"
	r.Headers["Host"] = "override.example"

	var directHits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		directHits.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: direct\n\n")
		w.(http.Flusher).Flush()
	}))
	defer target.Close()
	r.Domain = strings.TrimPrefix(target.URL, "http://")

	type proxied struct{ host, path string }
	seen := make(chan proxied, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		seen <- proxied{host: req.Host, path: req.URL.Path}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: via-proxy\n\n")
		w.(http.Flusher).Flush()
	}))
	defer proxy.Close()
	t.Setenv("HTTP_PROXY", proxy.URL)

	var events []string
	require.NoError(t, c.StreamSSE(r, func(event *dara.SSEEvent) error {
		events = append(events, dara.StringValue(event.Data))
		return nil
	}))
	require.Equal(t, []string{"via-proxy"}, events, "a configured HTTP proxy must carry the streaming request")
	require.Zero(t, int(directHits.Load()), "the endpoint must never be dialed directly while a proxy is configured")
	got := <-seen
	require.Equal(t, "override.example", got.host, "the signed Host override is still what the proxy receives")
	require.Equal(t, "/", got.path)
}

func TestStreamSSEHostOverrideKeepsHTTPSProxyConnect(t *testing.T) {
	c, r := sseFixture(t)
	c.SetHTTPSInsecure(true)
	r.Headers["Host"] = "override.example"

	hosts := make(chan string, 1)
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		select {
		case hosts <- req.Host:
		default:
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: one\n\ndata: two\n\n")
		w.(http.Flusher).Flush()
	}))
	target.StartTLS()
	defer target.Close()
	r.Domain = strings.TrimPrefix(target.URL, "https://")

	connects := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodConnect {
			http.Error(w, "CONNECT required", http.StatusMethodNotAllowed)
			return
		}
		select {
		case connects <- req.Host:
		default:
		}
		upstream, err := net.Dial("tcp", req.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			_ = upstream.Close()
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			_ = upstream.Close()
			return
		}
		_, _ = io.WriteString(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
		go func() {
			_, _ = io.Copy(upstream, clientConn)
			_ = upstream.Close()
		}()
		_, _ = io.Copy(clientConn, upstream)
		_ = clientConn.Close()
	}))
	defer proxy.Close()
	t.Setenv("HTTPS_PROXY", proxy.URL)

	var events []string
	require.NoError(t, c.StreamSSE(r, func(event *dara.SSEEvent) error {
		events = append(events, dara.StringValue(event.Data))
		return nil
	}))
	require.Equal(t, []string{"one", "two"}, events)
	require.Equal(t, r.Domain, <-connects, "CONNECT must address the real endpoint, not the Host override")
	require.Equal(t, "override.example", <-hosts, "the signed Host override must reach the tunneled request")
}

func TestStreamSSEWaitsForResponseHeadersWithReadTimeout(t *testing.T) {
	c, r := sseFixture(t)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)
	r.Scheme = "HTTP"
	r.Domain = strings.TrimPrefix(server.URL, "http://")
	c.SetReadTimeout(150 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- c.StreamSSE(r, nil) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "timeout awaiting response headers",
			"an explicit read timeout must bound the response-header wait")
	case <-time.After(10 * time.Second):
		t.Fatal("StreamSSE waited forever for response headers")
	}
}

func TestStreamSSEKeepsStreamOpenPastReadTimeout(t *testing.T) {
	c, r := sseFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "data: one\n\n")
		w.(http.Flusher).Flush()
		// Longer than the read timeout: only the header wait is bounded.
		time.Sleep(300 * time.Millisecond)
		_, _ = io.WriteString(w, "data: two\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()
	r.Scheme = "HTTP"
	r.Domain = strings.TrimPrefix(server.URL, "http://")
	c.SetReadTimeout(100 * time.Millisecond)

	var events []string
	require.NoError(t, c.StreamSSE(r, func(event *dara.SSEEvent) error {
		events = append(events, dara.StringValue(event.Data))
		return nil
	}))
	require.Equal(t, []string{"one", "two"}, events)
}
