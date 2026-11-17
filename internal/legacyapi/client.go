package legacyapi

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
)

// Client holds a credential snapshot for a single CLI invocation, including
// pagination. Each Send uses a fresh V2 client, so request-local endpoints,
// response capture, and dry runs never leak into another invocation.
type Client struct {
	Region, AccessKeyID, AccessKeySecret, SecurityToken string
	UseSTSCredential                                    bool
	RetryCount                                          int
	ExtraUserAgent                                      string
	readTimeout, connectTimeout                         time.Duration
	insecure                                            bool
	userAgent                                           map[string]string
	transport                                           http.RoundTripper
}

// Legacy ECS fallback values are compatibility data from V1 v1.63.107.
// Explicit profile/request and current CLI metadata timeouts take precedence.
//
//go:embed api_timeouts.json
var fallbackTimeoutJSON []byte
var fallbackTimeouts = func() map[string]map[string]int {
	var values map[string]map[string]int
	if err := json.Unmarshal(fallbackTimeoutJSON, &values); err != nil {
		panic(err)
	}
	return values
}()

func NewClient(region, id, secret, token string) *Client {
	return &Client{Region: region, AccessKeyID: id, AccessKeySecret: secret, SecurityToken: token, UseSTSCredential: token != "", userAgent: map[string]string{}}
}
func (c *Client) SetReadTimeout(v time.Duration)    { c.readTimeout = v }
func (c *Client) GetReadTimeout() time.Duration     { return c.readTimeout }
func (c *Client) SetConnectTimeout(v time.Duration) { c.connectTimeout = v }
func (c *Client) GetConnectTimeout() time.Duration  { return c.connectTimeout }
func (c *Client) SetHTTPSInsecure(v bool)           { c.insecure = v }
func (c *Client) SetTransport(v http.RoundTripper)  { c.transport = v }
func (c *Client) AppendUserAgent(key, value string) {
	if !strings.EqualFold(key, "core") && !strings.EqualFold(key, "go") {
		c.userAgent[key] = value
	}
}

func (c *Client) Send(r *Request) (*Response, error) {
	return c.send(r, false)
}

// Preview signs through exactly the same V2 call path, replacing only the
// final network operation. It neither dials an endpoint nor consumes a retry.
func (c *Client) Preview(r *Request) error {
	_, err := c.send(r, true)
	return err
}

func (c *Client) send(r *Request, preview bool) (*Response, error) {
	if r.Domain == "" {
		return nil, NewClientError("SDK.CanNotResolveEndpoint", "Can not resolve endpoint, please check your accessKey with secret, and read the user guide", nil)
	}
	readTimeout, connectTimeout := c.timeouts(r)
	transport := c.transport
	if transport == nil && !preview {
		proxy, err := legacyProxy(r)
		if err != nil {
			return nil, err
		}
		transport = &http.Transport{Proxy: proxy, DialContext: (&net.Dialer{Timeout: connectTimeout, DualStack: true}).DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: c.insecure}}
		defer transport.(*http.Transport).CloseIdleConnections()
	}
	maxRetries := c.RetryCount
	if maxRetries < 0 || preview {
		maxRetries = 0
	}
	for attempt := 0; ; attempt++ {
		capture := &wireCall{client: c, request: r, http: &http.Client{Transport: transport, Timeout: readTimeout}, preview: preview}
		conf := &openapi.Config{AccessKeyId: dara.String(c.AccessKeyID), AccessKeySecret: dara.String(c.AccessKeySecret),
			SecurityToken: dara.String(c.SecurityToken), Endpoint: dara.String(endpoint(r)), RegionId: dara.String(c.Region),
			SignatureAlgorithm: dara.String("v2"), HttpClient: capture}
		sdk, err := openapi.NewClient(conf)
		if err != nil {
			return nil, err
		}
		sdk.DisableSDKError = dara.Bool(true)
		query := copyMap(r.QueryParams)
		if r.GetStyle() == "RPC" {
			for _, key := range []string{"Action", "Version", "Format", "Timestamp", "SignatureNonce", "Signature", "SignatureMethod", "SignatureVersion", "AccessKeyId"} {
				delete(query, key)
			}
			query["Format"] = r.AcceptFormat
			if _, ok := query["RegionId"]; !ok {
				query["RegionId"] = c.Region
				if r.RegionId != "" {
					query["RegionId"] = r.RegionId
				}
			}
			query["SignatureType"] = ""
		}
		params := &openapi.Params{Action: dara.String(r.ApiName), Version: dara.String(r.Version), Protocol: dara.String(protocol(r)),
			Method: dara.String(r.Method), AuthType: dara.String("AK"), Style: dara.String(r.GetStyle()), Pathname: dara.String(r.path()),
			ReqBodyType: dara.String("json"), BodyType: dara.String("string")}
		req := &openapi.OpenApiRequest{Query: pointers(query), Headers: pointers(r.Headers)}
		// Host overrides affect the HTTP Host header, never the dial endpoint.
		for key := range req.Headers {
			if strings.EqualFold(key, "host") {
				delete(req.Headers, key)
			}
		}
		// ROA raw bodies are supplied by wireCall after SDK construction; the
		// V2 legacy ROA serializer cannot represent arbitrary bytes unchanged.
		if r.GetStyle() == "RPC" {
			req.Body = r.FormParams
		}
		// The HTTP extension owns proxy selection. Prevent Tea from parsing a
		// second proxy policy, including during entirely offline previews.
		_, sdkErr := sdk.CallApi(params, req, &dara.RuntimeOptions{IgnoreSSL: dara.Bool(c.insecure), NoProxy: dara.String(endpoint(r))})
		if capture.err != nil {
			// V1 retries failures obtaining HTTP headers, not errors while
			// reading a response that has already been received.
			if capture.response != nil {
				return capture.response, capture.err
			}
			if certificateError(capture.err) {
				return nil, capture.err
			}
			if attempt < maxRetries {
				continue
			}
			if maxRetries > 0 {
				n := attempt + 1
				msg := fmt.Sprintf("The request timed out %d times(%d for retry), perhaps we should have the threshold raised a little?", n, n)
				if strings.Contains(capture.err.Error(), "Client.Timeout") {
					msg += " Read timeout. Please set a valid ReadTimeout."
				} else {
					msg += " Connect timeout. Please set a valid ConnectTimeout."
				}
				return nil, NewClientError("SDK.TimeoutError", msg, capture.err)
			}
			return nil, capture.err
		}
		if capture.response == nil {
			return nil, sdkErr
		}
		resp := capture.response
		if resp.StatusCode >= 500 && attempt < maxRetries {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			e := NewServerError(resp.StatusCode, string(resp.Body), "").(*ServerError)
			e.RespHeaders = resp.Headers
			e.explainSignature(capture.stringToSign)
			return resp, e
		}
		return resp, nil
	}
}

func (c *Client) timeouts(r *Request) (time.Duration, time.Duration) {
	read, connect := 10*time.Second, 5*time.Second
	if value := fallbackTimeouts[strings.ToLower(r.Product)][r.ApiName]; value > 0 {
		read = time.Duration(value) * time.Second
	}
	if c.readTimeout != 0 {
		read = c.readTimeout
	}
	if c.connectTimeout != 0 {
		connect = c.connectTimeout
	}
	if r.ReadTimeout != 0 {
		read = r.ReadTimeout
	}
	if r.ConnectTimeout != 0 {
		connect = r.ConnectTimeout
	}
	return read, connect
}

type wireCall struct {
	client       *Client
	request      *Request
	http         *http.Client
	preview      bool
	response     *Response
	err          error
	stringToSign string
}

// Call is V2's supported HTTP extension point. SDK signing is retained; only
// the historical ROA body/headers and signature canonicalization are adapted
// using the V2 signing utilities. Full response facts are captured here before
// the SDK's JSON-only error parser or single-value header conversion.
func (w *wireCall) Call(h *http.Request, _ *http.Transport) (*http.Response, error) {
	r, c := w.request, w.client
	// Rebuild from CLI headers to avoid case-variant duplicates introduced by
	// SDK defaults (Go's Header map itself permits duplicates with different case).
	provider := "static_ak"
	if c.UseSTSCredential {
		provider = "static_sts"
	}
	h.Header = http.Header{}
	for key, value := range r.Headers {
		putHeader(h.Header, key, value)
	}
	putHeader(h.Header, "x-acs-version", r.Version)
	if r.GetStyle() == "RPC" || r.ApiName != "" {
		putHeader(h.Header, "x-acs-action", r.ApiName)
	}
	putHeader(h.Header, "x-acs-credentials-provider", provider)
	// Frozen legacy wire marker: signed --dryrun output historically exposes
	// this value. The actual caller is V2 (also identified in User-Agent).
	putHeader(h.Header, "x-sdk-core-version", "1.63.22")
	if host, assigned := r.Headers["Host"]; assigned {
		h.Host = host
		h.Header.Del("Host")
	}
	h.Header.Set("User-Agent", c.agent())
	if r.GetStyle() == "ROA" {
		putHeader(h.Header, "x-acs-signature-method", "HMAC-SHA1")
		putHeader(h.Header, "x-acs-signature-version", "1.0")
		if c.SecurityToken != "" {
			putHeader(h.Header, "x-acs-security-token", c.SecurityToken)
		}
		body := r.Content
		if len(r.FormParams) > 0 {
			body = []byte(encode(r.FormParams))
		}
		if h.Body != nil {
			_ = h.Body.Close()
		}
		h.Body = io.NopCloser(bytes.NewReader(body))
		h.ContentLength = int64(len(body))
		h.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
		if len(body) == 0 {
			h.Body = http.NoBody
		}
		putHeader(h.Header, "Date", time.Now().UTC().Format(http.TimeFormat))
		accept := "application/octet-stream"
		if r.AcceptFormat == "JSON" {
			accept = "application/json"
		} else if r.AcceptFormat == "XML" {
			accept = "application/xml"
		}
		putHeader(h.Header, "Accept", accept)
		contentType, exists := r.Headers["Content-Type"]
		if !exists {
			contentType = "application/octet-stream"
		}
		if len(r.FormParams) > 0 && (!exists || contentType == "") {
			contentType = "application/x-www-form-urlencoded"
		}
		putHeader(h.Header, "Content-Type", contentType)
		digest := md5.Sum(body)
		putHeader(h.Header, "Content-MD5", base64.StdEncoding.EncodeToString(digest[:]))
		// V1 canonicalizes x-acs keys using their original lower-case spelling.
		headers := map[string]*string{}
		for k, values := range h.Header {
			if strings.HasPrefix(k, "x-acs-") || k == "Accept" || k == "Content-MD5" || k == "Content-Type" || k == "Date" {
				headers[strings.ToLower(k)] = dara.String(strings.Join(values, ","))
			}
		}
		query := pointers(r.QueryParams)
		signing := &dara.Request{Method: dara.String(r.Method), Pathname: dara.String(r.path()), Headers: headers, Query: query}
		w.stringToSign = dara.StringValue(openapiutil.GetStringToSign(signing))
		putHeader(h.Header, "Authorization", "acs "+c.AccessKeyID+":"+dara.StringValue(openapiutil.GetROASignature(dara.String(w.stringToSign), dara.String(c.AccessKeySecret))))
	} else {
		putHeader(h.Header, "Content-Type", "application/x-www-form-urlencoded")
		// Keep the complete signed query for previews without polluting the
		// prepared request (pagination and cli-dry-run use preparation state).
		query := h.URL.Query()
		params := map[string]*string{}
		for key := range query {
			if key != "Signature" {
				params[key] = dara.String(query.Get(key))
			}
		}
		for key, value := range r.FormParams {
			params[key] = dara.String(value)
		}
		canonical := strings.NewReplacer("+", "%20", "*", "%2A", "%7E", "~").Replace(encodeStrings(params))
		w.stringToSign = r.Method + "&%2F&" + url.QueryEscape(canonical)
	}
	if w.preview {
		w.savePreview(h)
		w.response = &Response{StatusCode: 200, Headers: http.Header{}}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	resp, err := w.http.Do(h)
	if err != nil {
		w.err = err
		return nil, err
	}
	w.response = &Response{}
	if err = ReadResponse(w.response, resp); err != nil {
		w.err = err
		return nil, err
	}
	// Return a private replay to the SDK, which may consume or close it.
	resp.Body = io.NopCloser(bytes.NewReader(w.response.Body))
	return resp, nil
}

func (w *wireCall) savePreview(h *http.Request) {
	r := *w.request
	r.Scheme = protocol(w.request)
	r.Headers = copyMap(w.request.Headers)
	for key, values := range h.Header {
		if strings.EqualFold(key, "User-Agent") {
			continue
		}
		r.Headers[key] = strings.Join(values, ",")
	}
	r.QueryParams = map[string]string{}
	for key, values := range h.URL.Query() {
		r.QueryParams[key] = values[0]
	}
	if h.GetBody != nil {
		body, _ := h.GetBody()
		r.Content, _ = io.ReadAll(body)
		_ = body.Close()
	}
	r.signedText = ""
	w.request.signedText = r.String()
}

func (c *Client) agent() string {
	// Preserve historical CLI attribution; SDK implementation identity is V2.
	ua := dara.StringValue(openapiutil.GetUserAgent(nil))
	for _, key := range sortedKeys(c.userAgent) {
		ua += " " + key + "/" + c.userAgent[key]
	}
	if c.ExtraUserAgent != "" {
		ua += " Extra/" + c.ExtraUserAgent
	}
	return ua
}
func protocol(r *Request) string {
	if r.Scheme != "" {
		return r.Scheme
	}
	return "HTTP"
}
func endpoint(r *Request) string {
	if r.Port != "" {
		return net.JoinHostPort(r.Domain, r.Port)
	}
	return r.Domain
}
func copyMap(m map[string]string) map[string]string {
	n := make(map[string]string, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}
func pointers(m map[string]string) map[string]*string {
	n := make(map[string]*string, len(m))
	for k, v := range m {
		n[k] = dara.String(v)
	}
	return n
}
func encodeStrings(m map[string]*string) string {
	n := map[string]string{}
	for k, v := range m {
		n[k] = dara.StringValue(v)
	}
	return encode(n)
}

func certificateError(err error) bool {
	var unknown x509.UnknownAuthorityError
	return errors.As(err, &unknown) || strings.Contains(err.Error(), "x509: certificate signed by unknown authority")
}

// legacyProxy intentionally preserves V1's environment precedence and regex
// NO_PROXY matching, rather than silently changing existing proxy routing.
func legacyProxy(r *Request) (func(*http.Request) (*url.URL, error), error) {
	key := "HTTP_PROXY"
	if strings.EqualFold(protocol(r), "https") {
		key = "HTTPS_PROXY"
	}
	value := os.Getenv(key)
	if value == "" {
		value = os.Getenv(strings.ToLower(key))
	}
	var proxy *url.URL
	var err error
	if value != "" {
		proxy, err = url.Parse(value)
		if err != nil {
			return nil, err
		}
	}
	noProxy := os.Getenv("NO_PROXY")
	if noProxy == "" {
		noProxy = os.Getenv("no_proxy")
	}
	host := endpoint(r)
	if override, assigned := r.Headers["Host"]; assigned {
		host = override
	}
	for _, pattern := range strings.Split(noProxy, ",") {
		if noProxy == "" {
			break
		}
		if strings.HasPrefix(pattern, "*") {
			pattern = "." + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		if re.MatchString(host) {
			return nil, nil
		}
	}
	if proxy == nil {
		return nil, nil
	}
	return http.ProxyURL(proxy), nil
}

// Preserve the original header spelling used by legacy signing and previews.
func putHeader(headers http.Header, key, value string) { headers[key] = []string{value} }
