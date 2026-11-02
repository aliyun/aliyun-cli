// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	credentials "github.com/aliyun/credentials-go/credentials"
)

const pricingListPath = "/api/v1/price/supported-apis"

// PricingClientOptions contains quote-service settings, independent of the
// target product's endpoint and protocol. Credentials remain refreshable.
type PricingClientOptions struct {
	Credential                  credentials.Credential
	Region                      string
	ReadTimeout, ConnectTimeout time.Duration
	RetryCount                  int // retries after the first attempt, as in the CLI profile
	SkipSecureVerify            bool
	UserAgent                   string
}

// PricingRequest identifies an API to quote without executing it.
type PricingRequest struct {
	Language   string         `json:"language"`
	PopCode    string         `json:"popCode"`
	PopVersion string         `json:"popVersion"`
	ApiName    string         `json:"apiName"`
	Parameters map[string]any `json:"parameters"`
}

// PricingClient is shared by the main CLI and metadata plugins.
type PricingClient struct {
	client  *openapi.Client
	runtime *dara.RuntimeOptions
	retries int
}

func NewPricingClient(options PricingClientOptions) (*PricingClient, error) {
	if options.Credential == nil {
		return nil, fmt.Errorf("price mode requires resolved credentials; run `aliyun configure` to configure credentials")
	}
	if options.ReadTimeout <= 0 {
		options.ReadTimeout = 10 * time.Second
	}
	if options.ConnectTimeout <= 0 {
		options.ConnectTimeout = 5 * time.Second
	}
	conf := &openapi.Config{
		Credential: options.Credential,
		Endpoint:   tea.String(PricingEndpoint()),
		RegionId:   tea.String(options.Region),
		UserAgent:  tea.String(options.UserAgent),
		// Migrate the SDK without changing the quote service's ROA signature.
		SignatureAlgorithm: tea.String("v2"),
	}
	if options.ReadTimeout > 0 {
		conf.ReadTimeout = tea.Int(int(options.ReadTimeout / time.Millisecond))
	}
	if options.ConnectTimeout > 0 {
		conf.ConnectTimeout = tea.Int(int(options.ConnectTimeout / time.Millisecond))
	}
	// Own the retry boundary: SDK v2's default fallback also retries 4xx.
	conf.RetryOptions = &dara.RetryOptions{Retryable: false}
	client, err := openapi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	client.DisableSDKError = tea.Bool(true)
	host := os.Getenv(pricingHostEnv)
	if host != "" {
		// Tea v1.5.2 uses the host header as its dial endpoint. Override only
		// net/http's Request.Host after SDK signing, keeping the pricing URL
		// and TLS server name intact. Legacy ROA signing does not sign Host.
		client.HttpClient = &pricingHostClient{host: host, client: http.Client{
			Timeout: options.ReadTimeout + options.ConnectTimeout,
		}}
	}
	return &PricingClient{
		client:  client,
		runtime: &dara.RuntimeOptions{IgnoreSSL: tea.Bool(options.SkipSecureVerify)},
		retries: options.RetryCount,
	}, nil
}

type pricingHostClient struct {
	once   sync.Once
	client http.Client
	host   string
}

// Close releases the private transport used for a Host override. The default
// SDK transport is cached and managed by the SDK itself.
func (c *PricingClient) Close() {
	if client, ok := c.client.HttpClient.(*pricingHostClient); ok {
		client.client.CloseIdleConnections()
	}
}

func (c *pricingHostClient) Call(request *http.Request, transport *http.Transport) (*http.Response, error) {
	c.once.Do(func() { c.client.Transport = transport })
	request.Host = c.host
	return c.client.Do(request)
}

// PricingEndpoint deliberately ignores the endpoint of the API being quoted.
func PricingEndpoint() string {
	if endpoint := os.Getenv(pricingEndpointEnv); endpoint != "" {
		return endpoint
	}
	return defaultPricingEndpoint
}

func (c *PricingClient) Quote(request PricingRequest) (string, error) {
	if strings.TrimSpace(request.Language) == "" {
		request.Language = "en"
	}
	// Validate before entering the SDK, whose JSON stringifier ignores marshal
	// errors. A struct also needs conversion to a JSON object for the SDK.
	raw, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode pricing request: %w", err)
	}
	var body map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	out, err := c.call("GetApiPrice", "POST", pricingQuotePath, nil, body)
	if err == nil {
		return out, nil
	}
	// v2 classifies HTTP 400 as ClientError. Data is the server document;
	// Message is SDK-decorated and must not replace the server's own message.
	var responseError interface{ GetData() map[string]interface{} }
	if errors.As(err, &responseError) {
		if raw, marshalErr := json.Marshal(responseError.GetData()); marshalErr == nil {
			code, _, _, _ := pricingErrorFields(raw)
			if code == pricingCodeNotRequired || code == pricingCodeNotSupported {
				return string(raw), nil
			}
		}
	}
	return "", err
}

// ListSupportedAPIs returns one page; the CLI owns aggregation and filtering.
func (c *PricingClient) ListSupportedAPIs(nextToken string) (string, error) {
	query := map[string]*string{}
	if nextToken != "" {
		query["nextToken"] = tea.String(nextToken)
	}
	return c.call("ListSupportedPricingApis", "GET", pricingListPath, query, nil)
}

func (c *PricingClient) call(action, method, path string, query map[string]*string, body any) (string, error) {
	headers := map[string]*string{}
	params := &openapi.Params{
		Action: tea.String(action), Version: tea.String(pricingApiVersion),
		Protocol: tea.String("HTTPS"), Method: tea.String(method),
		AuthType: tea.String("AK"), Style: tea.String("ROA"),
		Pathname: tea.String(path), ReqBodyType: tea.String("json"),
		BodyType: tea.String("string"),
	}
	request := &openapiutil.OpenApiRequest{Headers: headers, Query: query, Body: body}
	var result map[string]any
	var err error
	for attempt := 0; ; attempt++ {
		result, err = c.client.CallApi(params, request, c.runtime)
		if err == nil || attempt >= c.retries || !retryPricingRequest(err) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		var coded interface{ GetCode() *string }
		if errors.As(err, &coded) && tea.StringValue(coded.GetCode()) != "" {
			return "", fmt.Errorf("call pricing service: %s: %w", tea.StringValue(coded.GetCode()), err)
		}
		return "", fmt.Errorf("call pricing service: %w", err)
	}
	out, ok := result["body"].(string)
	if !ok {
		return "", fmt.Errorf("pricing service returned no response body")
	}
	return out, nil
}

func retryPricingRequest(err error) bool {
	var response interface {
		GetStatusCode() *int
		GetData() map[string]interface{}
	}
	if errors.As(err, &response) {
		if raw, e := json.Marshal(response.GetData()); e == nil {
			code, _, _, _ := pricingErrorFields(raw)
			if code == pricingCodeNotRequired || code == pricingCodeNotSupported {
				return false
			}
		}
		return tea.IntValue(response.GetStatusCode()) >= 500
	}
	var certificate x509.UnknownAuthorityError
	var invalidCertificate x509.CertificateInvalidError
	var hostname x509.HostnameError
	if errors.As(err, &certificate) || errors.As(err, &invalidCertificate) || errors.As(err, &hostname) {
		return false
	}
	var transport *url.Error
	return errors.As(err, &transport)
}
