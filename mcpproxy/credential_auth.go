// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
)

const credentialRefreshSkew = 60 * time.Second

type credentialAuth struct {
	source mcpCredentialSource
	region RegionType
	client *http.Client
	now    func() time.Time

	mu        sync.Mutex
	token     string
	expiresAt time.Time
	lastErr   error
}

func newCredentialAuth(source mcpCredentialSource, region RegionType) *credentialAuth {
	return &credentialAuth{
		source: source,
		region: region,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (a *credentialAuth) GetToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && a.now().Add(credentialRefreshSkew).Before(a.expiresAt) {
		return a.token, nil
	}
	return a.refreshLocked()
}

func (a *credentialAuth) RefreshToken(rejectedToken string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Concurrent 401 responses may refer to a token that has already been replaced.
	if a.token != "" && a.token != rejectedToken && a.now().Add(credentialRefreshSkew).Before(a.expiresAt) {
		return a.token, nil
	}
	if a.token == rejectedToken {
		a.token, a.expiresAt = "", time.Time{}
	}
	return a.refreshLocked()
}

func (a *credentialAuth) Snapshot() (time.Time, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.expiresAt, a.lastErr != nil
}

func (a *credentialAuth) refreshLocked() (string, error) {
	startedAt := a.now()
	token, expiresAt, err := a.exchange(startedAt)
	a.lastErr = err
	if err != nil {
		return "", err
	}
	if !a.now().Before(expiresAt) {
		a.lastErr = fmt.Errorf("MCP access token expired during acquisition")
		return "", a.lastErr
	}
	a.token, a.expiresAt = token, expiresAt
	log.Printf("MCP auth: GenerateAccessToken succeeded; MCP access token cached, expires_at=%s", expiresAt.Format(time.RFC3339))
	return token, nil
}

type credentialHTTPClient struct {
	*http.Client
}

func (c credentialHTTPClient) Call(request *http.Request, _ *http.Transport) (*http.Response, error) {
	return c.Do(request)
}

func (a *credentialAuth) exchange(startedAt time.Time) (string, time.Time, error) {
	endpoint, clientID := "ramoauth.aliyuncs.com", "4071151845732613353"
	switch a.region {
	case RegionCN:
	case RegionINTL:
		endpoint, clientID = "ramoauth.alibabacloudcs.com", "4195410055503316452"
	default:
		return "", time.Time{}, fmt.Errorf("unsupported MCP credential authentication region")
	}
	if a.source == nil {
		return "", time.Time{}, fmt.Errorf("MCP credential source is not configured")
	}
	credential, err := a.source()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to load CLI credentials for MCP authentication")
	}
	if credential == nil || strings.TrimSpace(dara.StringValue(credential.AccessKeyId)) == "" || strings.TrimSpace(dara.StringValue(credential.AccessKeySecret)) == "" || dara.StringValue(credential.BearerToken) != "" {
		return "", time.Time{}, fmt.Errorf("MCP authentication requires an AccessKey ID and secret, with an optional STS token")
	}
	client, err := openapiClient.NewClient(&openapiutil.Config{
		AccessKeyId:        credential.AccessKeyId,
		AccessKeySecret:    credential.AccessKeySecret,
		SecurityToken:      credential.SecurityToken,
		Endpoint:           dara.String(endpoint),
		SignatureAlgorithm: dara.String("v2"),
		ConnectTimeout:     dara.Int(30000),
		ReadTimeout:        dara.Int(30000),
		RetryOptions:       &dara.RetryOptions{Retryable: false, MaxAttempts: 1},
		HttpClient:         credentialHTTPClient{a.client},
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to initialize MCP credential authentication")
	}
	log.Printf("MCP auth: calling GenerateAccessToken, endpoint=%s, credential_type=%q, has_sts_token=%t", endpoint, dara.StringValue(credential.Type), dara.StringValue(credential.SecurityToken) != "")
	response, err := client.CallApi(&openapiutil.Params{
		Action:      dara.String("GenerateAccessToken"),
		Version:     dara.String("2026-04-21"),
		Protocol:    dara.String("HTTPS"),
		Method:      dara.String("POST"),
		AuthType:    dara.String("AK"),
		Style:       dara.String("RPC"),
		Pathname:    dara.String("/"),
		ReqBodyType: dara.String("json"),
		BodyType:    dara.String("json"),
	}, &openapiutil.OpenApiRequest{
		Query: map[string]*string{
			"ClientId": dara.String(clientID),
			"Scope":    dara.String("/internal/acs/openapi"),
		},
	}, &dara.RuntimeOptions{})
	if err != nil {
		// SDK errors can include signed request URLs and unredacted response bodies.
		return "", time.Time{}, fmt.Errorf("GenerateAccessToken failed; verify CLI credentials, AliyunOpenAPIMCPServerStaticCredentialAccess permission, and the account's MCP static-credential pre-check")
	}
	if fmt.Sprint(response["statusCode"]) != "200" {
		return "", time.Time{}, fmt.Errorf("GenerateAccessToken returned an unexpected HTTP status")
	}
	return parseCredentialToken(response, startedAt)
}

func parseCredentialToken(response map[string]interface{}, startedAt time.Time) (string, time.Time, error) {
	payload, ok := response["body"].(map[string]interface{})
	if !ok {
		return "", time.Time{}, fmt.Errorf("GenerateAccessToken returned an invalid response body")
	}
	for _, key := range []string{"Data", "data"} {
		if nested, ok := payload[key].(map[string]interface{}); ok {
			payload = nested
			break
		}
	}
	token, _ := payload["AccessToken"].(string)
	if token == "" {
		token, _ = payload["access_token"].(string)
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", time.Time{}, fmt.Errorf("GenerateAccessToken returned an invalid access token")
	}
	expiresIn := payload["ExpiresIn"]
	if expiresIn == nil {
		expiresIn = payload["expires_in"]
	}
	seconds, err := strconv.ParseInt(fmt.Sprint(expiresIn), 10, 64)
	if err != nil || seconds <= 0 || seconds > int64((1<<63-1)/time.Second) {
		return "", time.Time{}, fmt.Errorf("GenerateAccessToken returned an invalid token expiration")
	}
	return token, startedAt.Add(time.Duration(seconds) * time.Second), nil
}
